package sessionv3runtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

func (b *Binder) executeGoTool(
	ctx context.Context,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.WorkExecutionResult, error) {
	binding := payload.Call.Binding.Go
	if binding == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "invalid_go_binding",
				Message: "go binding is missing",
			},
		}, nil
	}

	resolved, err := b.resolvePinnedGoTool(ctx, binding)
	if err != nil {
		//nolint:nilerr // Explicit.
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "go_binding_stale",
				Message: err.Error(),
			},
		}, nil
	}

	args := payload.Call.Arguments
	if strings.TrimSpace(string(args)) == "" {
		args = sessionv3Runtime.CanonicalValue(`{}`)
	}
	if !json.Valid([]byte(args)) {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{Code: "invalid_arguments", Message: "tool arguments are not JSON"},
		}, nil
	}

	timeout := time.Duration(binding.TimeoutMS) * time.Millisecond
	outputs, err := b.deps.ToolCaller.CallGoTool(
		ctx,
		resolved.Tool.Implementation.Function,
		json.RawMessage(args),
		timeout,
	)
	if err != nil {
		text, structured := toolOutputsToV3(outputs)
		if strings.TrimSpace(text) == "" {
			text = err.Error()
		}
		return sessionv3Runtime.WorkExecutionResult{
			Output: &sessionv3Runtime.WorkItemOutput{
				Tool: &sessionv3Runtime.ToolOutput{
					Kind:       sessionv3Runtime.ToolOutputKindError,
					Text:       text,
					Structured: structured,
				},
				Receipt: &sessionv3Runtime.ExternalReceipt{
					System:      "go-tool",
					Reference:   resolved.Tool.Implementation.Function,
					ConfirmedAt: time.Now().UTC(),
				},
			},
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "go_tool_failed",
				Message: err.Error(),
			},
			Confirmed: true,
		}, nil
	}
	text, structured := toolOutputsToV3(outputs)
	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
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

func (b *Binder) resolvePinnedGoTool(
	ctx context.Context,
	binding *sessionv3Runtime.GoToolBinding,
) (toolAPI.ResolvedToolView, error) {
	if binding == nil {
		return toolAPI.ResolvedToolView{}, errors.New("go binding is missing")
	}
	if binding.TimeoutMS < 0 {
		return toolAPI.ResolvedToolView{}, errors.New(
			"go Tool timeout cannot be negative",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(binding.Target.RootID),
		ArtifactID: artifactModel.ArtifactID(binding.Target.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return toolAPI.ResolvedToolView{}, err
	}

	target, err := b.deps.Tools.MapToolTarget(ctx, ref)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	resolved, err := b.deps.Tools.ResolveToolTarget(ctx, target)
	if err != nil {
		return toolAPI.ResolvedToolView{}, err
	}
	if resolved.Tool.Artifact.Ref() != ref ||
		resolved.Tool.Artifact.Revision != binding.Target.Revision ||
		resolved.Tool.Artifact.ResolvedDefinition == nil ||
		string(*resolved.Tool.Artifact.ResolvedDefinition) != binding.Target.Digest {
		return toolAPI.ResolvedToolView{}, errors.New(
			"go tool artifact changed since WorkItem binding",
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

func toolOutputsToV3(outputs []llmtoolsSpec.ToolOutputUnion) (string, sessionv3Runtime.CanonicalValue) {
	parts := []string{}
	for _, o := range outputs {
		switch o.Kind {
		case llmtoolsSpec.ToolOutputKindText:
			if o.TextItem != nil {
				parts = append(parts, o.TextItem.Text)
			}
		case llmtoolsSpec.ToolOutputKindImage:
			parts = append(parts, "[tool image omitted]")
		case llmtoolsSpec.ToolOutputKindFile:
			parts = append(parts, "[tool file omitted]")
		default:
			parts = append(parts, "[tool output omitted]")
		}
	}
	return strings.Join(parts, "\n\n"), sessionv3Runtime.MustCanonicalValue(outputs)
}

func mustMarshal(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return raw
}
