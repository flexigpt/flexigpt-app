package toolruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
)

// GoBinding is durable semantic evidence for one Go Tool call. It contains an
// immutable Artifact Definition witness, not an allocation request.
type GoBinding struct {
	Target    composition.CapabilityTarget    `json:"target"`
	Used      sessionRuntime.ResolvedArtifact `json:"used"`
	TimeoutMS int                             `json:"timeoutMS,omitempty"`
}

// GoExecutor owns Go Tool invocation. It resolves current Tool availability
// and verifies that the durable binding still identifies the same executable
// Tool definition before invoking local code.
type GoExecutor struct {
	tools  *toolAPI.Service
	caller GoToolCaller
}

func NewGoToolExecutor(
	tools *toolAPI.Service,
	caller GoToolCaller,
) (*GoExecutor, error) {
	if tools == nil || caller == nil {
		return nil, errors.New(
			"go tool executor dependencies are incomplete",
		)
	}

	return &GoExecutor{
		tools:  tools,
		caller: caller,
	}, nil
}

func (e *GoExecutor) EvaluateTool(
	_ context.Context,
	_ sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolEvaluation, error) {
	return sessionRuntime.ToolEvaluation{
		Decision: sessionRuntime.ToolEvaluationAllowed,
	}, nil
}

func (e *GoExecutor) ExecuteTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	binding, err := jsonutil.DecodeJSONRaw[GoBinding](
		request.ToolCall.Binding.Data,
	)
	if err != nil {
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

	current := resolvedArtifact(resolved)
	if err := ValidateGoBindingCurrent(binding, current); err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
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
	canonicalArgs, err := jsonutil.Canonicalize(args)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, fmt.Errorf(
			"go Tool arguments are invalid JSON: %w",
			err,
		)
	}

	outputs, invokeErr := e.caller.CallGoTool(
		ctx,
		resolved.Tool.Implementation.Function,
		json.RawMessage(canonicalArgs),
		durationMilliseconds(binding.TimeoutMS),
	)

	rawOutputs, err := CanonicalToolOutputs(outputs)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	result := sessionRuntime.ToolExecutionResult{
		HasResult:  true,
		Kind:       sessionRuntime.ToolResultKindSuccess,
		Text:       string(rawOutputs),
		Contents:   OutputItemsFromLLMToolOutputs(outputs),
		Structured: rawOutputs,
	}
	if invokeErr != nil {
		result.Kind = sessionRuntime.ToolResultKindError
		result.Error = invokeErr.Error()
	}

	return result, nil
}

func (e *GoExecutor) ResolveToolWait(
	_ context.Context,
	_ sessionRuntime.ToolWaitResolutionRequest,
) (sessionRuntime.ToolWaitResolution, error) {
	return sessionRuntime.ToolWaitResolution{}, errors.New(
		"go Tool execution does not expose an approval wait",
	)
}

// ValidateGoBindingCurrent prevents a pending Tool call from silently
// executing a changed Tool definition. Revision changes alone are allowed
// because local metadata may change without changing executable semantics.
func ValidateGoBindingCurrent(
	binding GoBinding,
	current *sessionRuntime.ResolvedArtifact,
) error {
	if err := binding.Target.Validate(); err != nil {
		return fmt.Errorf("invalid persisted Go Tool target: %w", err)
	}

	if binding.Target.Form != composition.TargetFormArtifact ||
		binding.Target.Artifact == nil {
		return errors.New(
			"go Tool binding must retain an Artifact-backed Tool target",
		)
	}

	if strings.TrimSpace(binding.Used.RootID) == "" ||
		strings.TrimSpace(binding.Used.ArtifactID) == "" ||
		strings.TrimSpace(binding.Used.Digest) == "" {
		return errors.New(
			"go Tool binding has no immutable Artifact Definition witness",
		)
	}

	if string(binding.Target.Artifact.RootID) != binding.Used.RootID ||
		string(binding.Target.Artifact.ArtifactID) != binding.Used.ArtifactID {
		return errors.New(
			"go Tool binding target and Artifact Definition witness differ",
		)
	}

	if current == nil {
		return errors.New("current Go Tool Artifact witness is unavailable")
	}
	if current.RootID != binding.Used.RootID ||
		current.ArtifactID != binding.Used.ArtifactID {
		return errors.New(
			"go Tool Artifact identity changed since Tool call preparation",
		)
	}
	if strings.TrimSpace(current.Digest) == "" ||
		current.Digest != binding.Used.Digest {
		return errors.New(
			"go Tool Definition changed since Tool call preparation",
		)
	}

	return nil
}

func resolvedArtifact(
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

func durationMilliseconds(
	value int,
) time.Duration {
	if value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Millisecond
}
