package toolruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
)

// V3PreparedChoice is a provider Tool choice paired with its immutable v3
// semantic call binding.
type V3PreparedChoice struct {
	Choice inferenceSpec.ToolChoice
	Call   sessionv3Runtime.ToolCall
}

// V3Executor owns v3 source-backed Go and provider Tool preparation and Go
// Tool execution. It is intentionally independent of Session state mutation.
type V3Executor struct {
	tools  *toolAPI.Service
	caller GoToolCaller
}

// ErrProviderManagedWorkflowTool prevents a provider SDK Tool Artifact from
// being treated as a locally executable Tool. Such a Tool must be executed by
// its typed Model plan, not by a synthetic local success result.
var ErrProviderManagedWorkflowTool = errors.New(
	"provider-managed SDK Tool requires a typed Model execution plan",
)

func NewV3Executor(
	tools *toolAPI.Service,
	caller GoToolCaller,
) (*V3Executor, error) {
	if tools == nil || caller == nil {
		return nil, errors.New("session v3 Tool runtime dependencies are incomplete")
	}

	return &V3Executor{
		tools:  tools,
		caller: caller,
	}, nil
}

// IsProviderManagedWorkflowTool verifies a pinned Workflow Tool target before
// admission. Provider SDK Tools must remain inside a typed Model plan and
// cannot be admitted as local Tool WorkItems.
func (e *V3Executor) IsProviderManagedWorkflowTool(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) (bool, error) {
	resolved, err := e.resolvePinnedTool(ctx, pin)
	if err != nil {
		return false, err
	}

	switch resolved.Tool.Implementation.Kind {
	case toolv1.ImplementationKindGo:
		return false, nil
	case toolv1.ImplementationKindSDK:
		return true, nil
	default:
		return false, fmt.Errorf(
			"unsupported tool implementation %q",
			resolved.Tool.Implementation.Kind,
		)
	}
}

// Choices resolves selected Tool Artifacts, verifies requested pins, and
// creates provider choices plus durable v3 bindings.
func (e *V3Executor) Choices(
	ctx context.Context,
	selections []sessionv3Runtime.ArtifactSelection,
) ([]V3PreparedChoice, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	if e == nil || e.tools == nil {
		return nil, errors.New("session v3 Tool runtime is unavailable")
	}

	output := make([]V3PreparedChoice, 0, len(selections))
	for index := range selections {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		selection := selections[index]
		ref := artifactModel.ArtifactRef{
			RootID:     rootModel.RootID(selection.RootID),
			ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
		}
		if err := ref.Validate(); err != nil {
			return nil, fmt.Errorf("selected Tool %d: %w", index, err)
		}

		target, err := e.tools.MapToolTarget(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf(
				"map selected Tool %s/%s: %w",
				ref.RootID,
				ref.ArtifactID,
				err,
			)
		}
		resolved, err := e.tools.ResolveToolTarget(ctx, target)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve selected Tool %s/%s: %w",
				ref.RootID,
				ref.ArtifactID,
				err,
			)
		}
		if err := validateV3Selection(selection, resolved); err != nil {
			return nil, err
		}

		choice, call, err := v3ChoiceForResolvedTool(resolved)
		if err != nil {
			return nil, err
		}
		output = append(output, V3PreparedChoice{
			Choice: choice,
			Call:   call,
		})
	}
	return output, nil
}

// BindWorkflowTool resolves a Workflow Tool Artifact target to its current
// typed Tool binding. It verifies the durable Artifact pin before Session
// creates a dispatch attempt.
func (e *V3Executor) BindWorkflowTool(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.ToolCall, error) {
	if e == nil || e.tools == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"session v3 Tool runtime is unavailable",
		)
	}
	if payload == nil || payload.Target == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow tool Artifact target is missing",
		)
	}

	resolved, err := e.resolvePinnedTool(ctx, *payload.Target)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}

	_, call, err := v3ChoiceForResolvedTool(resolved)
	if err != nil {
		return sessionv3Runtime.ToolCall{}, err
	}
	if call.Binding.Kind == sessionv3Runtime.ToolBindingKindProvider {
		return sessionv3Runtime.ToolCall{}, fmt.Errorf(
			"%w: Tool %q",
			ErrProviderManagedWorkflowTool,
			resolved.Tool.Name,
		)
	}
	if payload.Call.Name != "" && payload.Call.Name != call.Name {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"workflow tool name changed since Workflow admission",
		)
	}

	call.ID = payload.Call.ID
	call.ProviderCallID = payload.Call.ProviderCallID
	call.Arguments = payload.Call.Arguments.Clone()
	call.Automatic = payload.Call.Automatic
	return call, nil
}

// BindGo verifies a current Go Tool binding and returns durable semantic
// binding facts before Session Runtime v3 persists a dispatched Attempt.
func (e *V3Executor) BindGo(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.SemanticBinding, error) {
	if e == nil || e.tools == nil {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"session v3 Tool runtime is unavailable",
		)
	}
	if payload == nil ||
		payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindGo ||
		payload.Call.Binding.Go == nil {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"go tool binding is missing",
		)
	}
	if err := payload.Call.Validate(); err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}

	binding := payload.Call.Binding.Go
	if _, err := e.resolvePinnedGoTool(ctx, binding); err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}

	return sessionv3Runtime.SemanticBinding{
		Version: "session-v3-go-tool/v1",
		Artifacts: []sessionv3Runtime.ArtifactPin{
			binding.Target,
		},
		TargetFingerprint: strings.Join([]string{
			"go",
			binding.Target.RootID,
			binding.Target.ArtifactID,
			binding.Target.Digest,
			binding.Function,
		}, ":"),
		BoundAt: time.Now().UTC(),
	}, nil
}

// ExecuteGo executes only a v3 Go Tool binding. It returns a durable result
// instead of exposing source-backed Tool or caller details to Session core.
func (e *V3Executor) ExecuteGo(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.WorkExecutionResult, error) {
	if e == nil || e.tools == nil || e.caller == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "go_runtime_unavailable",
				Message: "Go Tool runtime is unavailable",
			},
		}, nil
	}
	if payload == nil || payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindGo {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "invalid_go_binding",
				Message: "Go Tool binding is missing",
			},
		}, nil
	}

	binding := payload.Call.Binding.Go
	resolved, err := e.resolvePinnedGoTool(ctx, binding)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "go_binding_stale",
				Message: err.Error(),
			},
		}, nil
	}

	arguments := payload.Call.Arguments.Clone()
	if len(arguments) == 0 {
		arguments = sessionv3Runtime.MustCanonicalValue(map[string]any{})
	}
	if !arguments.Valid() {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "invalid_arguments",
				Message: "Go Tool arguments are not canonical JSON",
			},
		}, nil
	}

	outputs, invokeErr := e.caller.CallGoTool(
		ctx,
		resolved.Tool.Implementation.Function,
		json.RawMessage(arguments.Clone()),
		durationMilliseconds(binding.TimeoutMS),
	)
	text, structured, conversionErr := V3ToolOutputs(outputs)
	if conversionErr != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "go_tool_output_invalid",
				Message: conversionErr.Error(),
			},
		}, nil
	}
	if invokeErr != nil {
		if strings.TrimSpace(text) == "" {
			text = invokeErr.Error()
		}
		return sessionv3Runtime.WorkExecutionResult{
			Output: &sessionv3Runtime.WorkItemOutput{
				Value: structured,
				Tool: &sessionv3Runtime.ToolOutput{
					Kind:       sessionv3Runtime.ToolOutputKindError,
					Text:       text,
					Structured: structured,
					Error:      invokeErr.Error(),
				},
				Receipt: &sessionv3Runtime.ExternalReceipt{
					System:      "go-tool",
					Reference:   resolved.Tool.Implementation.Function,
					ConfirmedAt: time.Now().UTC(),
				},
			},
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "go_tool_failed",
				Message: invokeErr.Error(),
			},
			Confirmed: true,
		}, nil
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: structured,
			Tool: &sessionv3Runtime.ToolOutput{
				Kind:       sessionv3Runtime.ToolOutputKindSuccess,
				Text:       text,
				Structured: structured,
			},
			Receipt: &sessionv3Runtime.ExternalReceipt{
				System:      "go-tool",
				Reference:   resolved.Tool.Implementation.Function,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, nil
}

// V3ToolOutputs produces the v3 durable text projection and canonical
// structured representation from one Tool Runtime response.
func V3ToolOutputs(
	outputs []llmtoolsSpec.ToolOutputUnion,
) (string, sessionv3Runtime.CanonicalValue, error) {
	raw, err := CanonicalToolOutputs(outputs)
	if err != nil {
		return "", nil, err
	}

	value, err := sessionv3Runtime.CanonicalValueFrom(raw)
	if err != nil {
		return "", nil, err
	}
	return ToolOutputsText(outputs), value, nil
}

func (e *V3Executor) resolvePinnedTool(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) (toolAPI.ResolvedToolView, error) {
	if e == nil || e.tools == nil {
		return toolAPI.ResolvedToolView{}, errors.New(
			"session v3 Tool runtime is unavailable",
		)
	}
	if pin.RootID == "" ||
		pin.ArtifactID == "" ||
		pin.Revision == 0 ||
		strings.TrimSpace(pin.Digest) == "" {
		return toolAPI.ResolvedToolView{}, errors.New(
			"tool artifact pin is incomplete",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return toolAPI.ResolvedToolView{}, err
	}

	target, err := e.tools.MapToolTarget(ctx, ref)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	resolved, err := e.tools.ResolveToolTarget(ctx, target)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	if resolved.Tool.Artifact.Ref() != ref ||
		resolved.Tool.Artifact.Revision != pin.Revision ||
		resolved.Tool.Artifact.ResolvedDefinition == nil ||
		string(*resolved.Tool.Artifact.ResolvedDefinition) != pin.Digest {
		return toolAPI.ResolvedToolView{}, errors.New(
			"tool artifact changed since WorkItem binding",
		)
	}
	return resolved, nil
}

func (e *V3Executor) resolvePinnedGoTool(
	ctx context.Context,
	binding *sessionv3Runtime.GoToolBinding,
) (toolAPI.ResolvedToolView, error) {
	if binding == nil {
		return toolAPI.ResolvedToolView{}, errors.New("go tool binding is missing")
	}
	if binding.TimeoutMS < 0 {
		return toolAPI.ResolvedToolView{}, errors.New(
			"go tool timeout cannot be negative",
		)
	}

	resolved, err := e.resolvePinnedTool(ctx, binding.Target)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	if strings.TrimSpace(binding.LogicalName) != "" &&
		binding.LogicalName != string(resolved.Tool.Name) {
		return toolAPI.ResolvedToolView{}, errors.New(
			"go tool logical name changed since WorkItem binding",
		)
	}
	if resolved.Tool.Implementation.Kind != toolv1.ImplementationKindGo ||
		resolved.Tool.Implementation.Function != binding.Function {
		return toolAPI.ResolvedToolView{}, errors.New(
			"go tool implementation changed since WorkItem binding",
		)
	}
	return resolved, nil
}

func validateV3Selection(
	selection sessionv3Runtime.ArtifactSelection,
	resolved toolAPI.ResolvedToolView,
) error {
	artifact := resolved.Tool.Artifact
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != artifact.Revision {
		return errors.New("selected Tool revision changed")
	}
	if selection.ExpectedDigest != "" &&
		(artifact.ResolvedDefinition == nil ||
			selection.ExpectedDigest != string(*artifact.ResolvedDefinition)) {
		return errors.New("selected Tool Definition changed")
	}
	return nil
}

func v3ChoiceForResolvedTool(
	resolved toolAPI.ResolvedToolView,
) (inferenceSpec.ToolChoice, sessionv3Runtime.ToolCall, error) {
	name := strings.TrimSpace(string(resolved.Tool.Name))
	if name == "" {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			errors.New("resolved Tool has no logical name")
	}

	arguments, err := InputSchema(resolved.Tool.InputSchema)
	if err != nil {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			err
	}

	digest := ""
	if resolved.Tool.Artifact.ResolvedDefinition != nil {
		digest = string(*resolved.Tool.Artifact.ResolvedDefinition)
	}
	if strings.TrimSpace(digest) == "" {
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			errors.New("resolved Tool has no Definition digest")
	}

	pin := sessionv3Runtime.ArtifactPin{
		RootID:     string(resolved.Tool.Artifact.RootID),
		ArtifactID: string(resolved.Tool.Artifact.ID),
		Revision:   resolved.Tool.Artifact.Revision,
		Digest:     digest,
	}

	switch resolved.Tool.Implementation.Kind {
	case toolv1.ImplementationKindGo:
		function := strings.TrimSpace(resolved.Tool.Implementation.Function)
		if function == "" {
			return inferenceSpec.ToolChoice{},
				sessionv3Runtime.ToolCall{},
				errors.New("resolved Go Tool has no function")
		}

		choiceID := StableChoiceID(
			"v3-go",
			pin.RootID,
			pin.ArtifactID,
			name,
			pin.Digest,
		)
		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          choiceID,
			Name:        name,
			Description: resolved.Tool.Description,
			Arguments:   arguments,
		}
		call := sessionv3Runtime.ToolCall{
			Name:        name,
			ChoiceID:    choiceID,
			DisplayName: resolved.Tool.DisplayName,
			Description: resolved.Tool.Description,
			Binding: sessionv3Runtime.ToolBinding{
				Kind: sessionv3Runtime.ToolBindingKindGo,
				Go: &sessionv3Runtime.GoToolBinding{
					Target:      pin,
					LogicalName: name,
					Function:    function,
				},
			},
		}
		return choice, call, nil

	case toolv1.ImplementationKindSDK:
		sdkType := strings.TrimSpace(
			string(resolved.Tool.Implementation.SDKToolType),
		)
		if sdkType == "" {
			return inferenceSpec.ToolChoice{},
				sessionv3Runtime.ToolCall{},
				errors.New("resolved SDK Tool has no type")
		}

		choiceID := StableChoiceID(
			"v3-sdk",
			pin.RootID,
			pin.ArtifactID,
			name,
			pin.Digest,
		)
		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolType(sdkType),
			ID:          choiceID,
			Name:        name,
			Description: resolved.Tool.Description,
			Arguments:   arguments,
		}
		call := sessionv3Runtime.ToolCall{
			Name:        name,
			ChoiceID:    choiceID,
			DisplayName: resolved.Tool.DisplayName,
			Description: resolved.Tool.Description,
			Binding: sessionv3Runtime.ToolBinding{
				Kind: sessionv3Runtime.ToolBindingKindProvider,
				Provider: &sessionv3Runtime.ProviderToolBinding{
					Artifact: &pin,
					Type:     sdkType,
					Name:     name,
				},
			},
		}
		return choice, call, nil

	default:
		return inferenceSpec.ToolChoice{},
			sessionv3Runtime.ToolCall{},
			fmt.Errorf(
				"unsupported Tool implementation %q",
				resolved.Tool.Implementation.Kind,
			)
	}
}
