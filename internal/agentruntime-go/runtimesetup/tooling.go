package runtimesetup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/toolruntime"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
)

type goToolExecutor struct {
	tools   *toolAPI.Service
	runtime *toolruntime.Adapter
}

type goToolBinding struct {
	Target    composition.CapabilityTarget    `json:"target"`
	Used      sessionRuntime.ResolvedArtifact `json:"used"`
	TimeoutMS int                             `json:"timeoutMS,omitempty"`
}

func newGoToolExecutor(
	tools *toolAPI.Service,
	runtime *toolruntime.Adapter,
) (*goToolExecutor, error) {
	if tools == nil || runtime == nil {
		return nil, errors.New(
			"go Tool executor dependencies are incomplete",
		)
	}

	return &goToolExecutor{
		tools:   tools,
		runtime: runtime,
	}, nil
}

func (e *goToolExecutor) EvaluateTool(
	_ context.Context,
	_ sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	return sessionRuntime.ToolEvaluation{
		Decision: sessionRuntime.ToolEvaluationAllowed,
	}, nil
}

func (e *goToolExecutor) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	var binding goToolBinding
	if err := json.Unmarshal(request.ToolCall.Binding.Data, &binding); err != nil {
		return sessionRuntime.ToolExecutionResult{}, fmt.Errorf(
			"decode Go Tool binding: %w",
			err,
		)
	}
	if binding.TimeoutMS < 0 {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"go Tool timeout cannot be negative",
		)
	}

	resolved, err := e.tools.ResolveToolTarget(ctx, binding.Target)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}
	if binding.Used != *toolResolvedArtifact(resolved) {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"prepared Go Tool binding is stale",
		)
	}

	if resolved.Tool.Implementation.Kind != toolv1.ImplementationKindGo {
		return sessionRuntime.ToolExecutionResult{}, fmt.Errorf(
			"tool %q is not a Go Tool",
			resolved.Tool.Name,
		)
	}

	args := request.ToolCall.Arguments
	if len(bytes.TrimSpace(args)) == 0 {
		args = json.RawMessage(`{}`)
	}
	if !json.Valid(args) {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"go Tool arguments are invalid JSON",
		)
	}

	outputs, invokeErr := e.runtime.CallGoTool(
		ctx,
		resolved.Tool.Implementation.Function,
		args,
		durationMilliseconds(binding.TimeoutMS),
	)

	rawOutputs, err := json.Marshal(outputs)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	result := sessionRuntime.ToolExecutionResult{
		HasResult:  true,
		Kind:       sessionRuntime.ToolResultKindSuccess,
		Text:       string(rawOutputs),
		Contents:   toolOutputItemsFromLLMToolOutputs(outputs),
		Structured: rawOutputs,
	}
	if invokeErr != nil {
		result.Kind = sessionRuntime.ToolResultKindError
		result.Error = invokeErr.Error()
	}
	return result, nil
}

func (e *goToolExecutor) ResolveToolWait(
	_ context.Context,
	_ sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	return sessionRuntime.ToolWaitResolution{}, errors.New(
		"go tool execution does not expose an approval wait",
	)
}

func (c *coordinator) prepareTools(
	ctx context.Context,
	contributions []toolContribution,
	summary *sessionRuntime.OperationResolutionSummary,
) ([]inferenceSpec.ToolChoice, preparedBindings, error) {
	choices := make(
		[]inferenceSpec.ToolChoice,
		0,
		len(contributions),
	)
	bindings := preparedBindings{
		byChoice: make(
			map[string]sessionRuntime.ToolCallProposal,
			len(contributions),
		),
		byName: make(
			map[string]sessionRuntime.ToolCallProposal,
			len(contributions),
		),
	}

	for _, contribution := range contributions {
		if contribution.target == nil {
			continue
		}

		resolved, err := c.tools.ResolveToolTarget(
			ctx,
			*contribution.target,
		)
		if err != nil {
			item := contribution.item
			item.State = sessionRuntime.ResolutionStateUnavailable
			item.Reason = err.Error()
			summary.Tools = append(
				summary.Tools,
				item,
			)
			severity := sessionRuntime.DiagnosticSeverityWarning
			if contribution.required {
				severity = sessionRuntime.DiagnosticSeverityError
			}
			summary.Diagnostics = append(
				summary.Diagnostics,
				sessionRuntime.ResolutionDiagnostic{
					Severity: severity,
					Code:     "tool_unavailable",
					Message:  err.Error(),
					Key:      contribution.key,
				},
			)
			if contribution.required {
				return nil, preparedBindings{}, operationPreparationError(
					"agent_required_tool_unavailable",
					err.Error(),
					*summary,
					err,
				)
			}
			continue
		}

		used := toolResolvedArtifact(resolved)
		if contribution.selection != nil &&
			contribution.selection.Capability.Artifact != nil {
			selected := *contribution.selection.Capability.Artifact
			if err := requireSelectionPin(selected, used); err != nil {
				item := contribution.item
				item.State = sessionRuntime.ResolutionStateStale
				item.Selected = &selected
				item.Used = used
				item.Reason = err.Error()
				summary.Tools = append(summary.Tools, item)

				return nil, preparedBindings{}, operationPreparationError(
					"tool_selection_stale",
					err.Error(),
					*summary,
					err,
				)
			}
		}

		choice, proposal, err := toolChoiceAndBinding(
			resolved,
			*contribution.target,
			contribution,
		)
		if err != nil {
			return nil, preparedBindings{}, operationPreparationError(
				"tool_preparation_failed",
				err.Error(),
				*summary,
				err,
			)
		}

		if _, duplicate := bindings.byChoice[choice.ID]; duplicate {
			return nil, preparedBindings{}, operationPreparationError(
				"tool_choice_collision",
				"effective Tool choices have a duplicate choice ID",
				*summary,
				nil,
			)
		}
		if _, duplicate := bindings.byName[choice.Name]; duplicate {
			return nil, preparedBindings{}, operationPreparationError(
				"tool_name_collision",
				"effective Tool choices have a duplicate provider name",
				*summary,
				nil,
			)
		}

		choices = append(choices, choice)
		bindings.byChoice[choice.ID] = proposal
		bindings.byName[choice.Name] = proposal

		item := contribution.item
		item.State = sessionRuntime.ResolutionStateIncluded
		item.Used = used
		summary.Tools = append(summary.Tools, item)
	}

	return choices, bindings, nil
}

func toolChoiceAndBinding(
	resolved toolAPI.ResolvedToolView,
	target composition.CapabilityTarget,
	contribution toolContribution,
) (
	inferenceSpec.ToolChoice,
	sessionRuntime.ToolCallProposal,
	error,
) {
	arguments, err := toolArguments(resolved.Tool.InputSchema)
	if err != nil {
		return inferenceSpec.ToolChoice{},
			sessionRuntime.ToolCallProposal{},
			err
	}

	choiceID := stableChoiceID(toolChoicePrefix, contribution.key)
	choice := inferenceSpec.ToolChoice{
		ID:          choiceID,
		Name:        string(resolved.Tool.Name),
		Description: resolved.Tool.Description,
	}
	proposal := sessionRuntime.ToolCallProposal{
		Name:     choice.Name,
		ChoiceID: choice.ID,
		Required: true,
	}

	if contribution.autoExecute != nil {
		proposal.Automatic = *contribution.autoExecute
	}

	switch resolved.Tool.Implementation.Kind {
	case toolv1.ImplementationKindGo:
		rawBinding, err := json.Marshal(goToolBinding{
			Target: target,
			Used:   *toolResolvedArtifact(resolved),
		})
		if err != nil {
			return inferenceSpec.ToolChoice{},
				sessionRuntime.ToolCallProposal{},
				err
		}

		choice.Type = inferenceSpec.ToolTypeFunction
		choice.Arguments = arguments
		proposal.Type = choice.Type
		proposal.Binding = sessionRuntime.ToolBinding{
			Kind: sessionRuntime.ToolBindingKindGo,
			Data: rawBinding,
		}
		return choice, proposal, nil

	case toolv1.ImplementationKindSDK:
		choice.Type = inferenceSpec.ToolType(
			resolved.Tool.Implementation.SDKToolType,
		)
		proposal.Type = choice.Type
		proposal.Required = false
		proposal.ProviderManaged = true
		proposal.Binding = sessionRuntime.ToolBinding{
			Kind: sessionRuntime.ToolBindingKindProvider,
		}

		switch resolved.Tool.Implementation.SDKToolType {
		case toolv1.SDKToolTypeWebSearch:
			var config inferenceSpec.WebSearchToolChoiceItem
			if contribution.selection != nil &&
				len(contribution.selection.UserArguments) != 0 {
				if err := json.Unmarshal(
					contribution.selection.UserArguments,
					&config,
				); err != nil {
					return inferenceSpec.ToolChoice{},
						sessionRuntime.ToolCallProposal{},
						fmt.Errorf(
							"decode SDK webSearch configuration: %w",
							err,
						)
				}
			}
			choice.Type = inferenceSpec.ToolTypeWebSearch
			choice.WebSearchArguments = &config

		case toolv1.SDKToolTypeFunction,
			toolv1.SDKToolTypeCustom:
			choice.Arguments = arguments

		default:
			return inferenceSpec.ToolChoice{},
				sessionRuntime.ToolCallProposal{},
				fmt.Errorf(
					"unsupported SDK Tool type %q",
					resolved.Tool.Implementation.SDKToolType,
				)
		}
		return choice, proposal, nil

	default:
		return inferenceSpec.ToolChoice{},
			sessionRuntime.ToolCallProposal{},
			fmt.Errorf(
				"unsupported Tool implementation %q",
				resolved.Tool.Implementation.Kind,
			)
	}
}

func toolArguments(
	raw json.RawMessage,
) (map[string]any, error) {
	raw = bytes.TrimSpace(raw)
	switch {
	case len(raw) == 0:
		return map[string]any{typeStr: jsonSchemaTypeObject}, nil
	case bytes.Equal(raw, []byte("true")):
		return map[string]any{typeStr: jsonSchemaTypeObject}, nil
	case bytes.Equal(raw, []byte("false")):
		return nil, errors.New(
			"false Tool input schema cannot be advertised",
		)
	}

	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf(
			"tool input schema must be an object: %w",
			err,
		)
	}
	if output == nil {
		output = map[string]any{typeStr: jsonSchemaTypeObject}
	}
	return output, nil
}

func mergeToolBindings(
	leftChoices []inferenceSpec.ToolChoice,
	leftBindings preparedBindings,
	rightChoices []inferenceSpec.ToolChoice,
	rightBindings preparedBindings,
) ([]inferenceSpec.ToolChoice, preparedBindings, error) {
	choices := append(
		cloneToolChoices(leftChoices),
		cloneToolChoices(rightChoices)...,
	)
	bindings := clonePreparedBindings(leftBindings)

	for choiceID, proposal := range rightBindings.byChoice {
		if _, duplicate := bindings.byChoice[choiceID]; duplicate {
			return nil, preparedBindings{}, errors.New(
				"provider Tool choice ID collision",
			)
		}
		bindings.byChoice[choiceID] = proposal
	}

	for name, proposal := range rightBindings.byName {
		if _, duplicate := bindings.byName[name]; duplicate {
			return nil, preparedBindings{}, errors.New(
				"provider Tool name collision",
			)
		}
		bindings.byName[name] = proposal
	}

	return choices, bindings, nil
}

func clonePreparedBindings(value preparedBindings) preparedBindings {
	output := preparedBindings{
		byChoice: make(
			map[string]sessionRuntime.ToolCallProposal,
			len(value.byChoice),
		),
		byName: make(
			map[string]sessionRuntime.ToolCallProposal,
			len(value.byName),
		),
	}

	for key, proposal := range value.byChoice {
		output.byChoice[key] = cloneToolProposal(proposal)
	}
	for key, proposal := range value.byName {
		output.byName[key] = cloneToolProposal(proposal)
	}
	return output
}

func cloneToolProposal(
	value sessionRuntime.ToolCallProposal,
) sessionRuntime.ToolCallProposal {
	output := value
	output.Arguments = append(
		json.RawMessage(nil),
		value.Arguments...,
	)
	output.Binding.Data = append(
		json.RawMessage(nil),
		value.Binding.Data...,
	)
	return output
}

func cloneToolChoices(
	values []inferenceSpec.ToolChoice,
) []inferenceSpec.ToolChoice {
	output := make([]inferenceSpec.ToolChoice, len(values))
	copy(output, values)
	return output
}

func toolResolvedArtifact(
	value toolAPI.ResolvedToolView,
) *sessionRuntime.ResolvedArtifact {
	digest := ""
	if value.Tool.Artifact.ResolvedDefinition != nil {
		digest = string(*value.Tool.Artifact.ResolvedDefinition)
	}
	return &sessionRuntime.ResolvedArtifact{
		RootID:     string(value.Tool.Artifact.RootID),
		ArtifactID: string(value.Tool.Artifact.ID),
		Revision:   sessionRuntime.Revision(value.Tool.Artifact.Revision),
		Digest:     digest,
	}
}

func stableChoiceID(prefix, key string) string {
	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes([]byte(prefix+"\x00"+key))),
		cryptoutil.DigestSHA256Prefix,
	)
	return prefix + "-" + digest[:16]
}

func durationMilliseconds(value int) time.Duration {
	if value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Millisecond
}
