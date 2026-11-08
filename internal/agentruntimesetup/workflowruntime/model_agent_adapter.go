package workflowruntime

import (
	"context"
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/modelruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

// ModelAgentInvocationAdapter is Workflow setup composition over existing
// Model and Agent APIs. It does not add a Workflow-specific Model or Agent
// domain request contract.
type ModelAgentInvocationAdapter struct {
	models *modelruntime.SourceResolver
	agents *agentAPI.Service
}

func NewModelAgentInvocationAdapter(
	models *modelruntime.SourceResolver,
	agents *agentAPI.Service,
) (*ModelAgentInvocationAdapter, error) {
	if models == nil || agents == nil {
		return nil, errors.New(
			"workflow model and agent invocation adapter dependencies are incomplete",
		)
	}
	return &ModelAgentInvocationAdapter{
		models: models,
		agents: agents,
	}, nil
}

func (*ModelAgentInvocationAdapter) ModelInvocationCapabilities() WorkflowInvocationTargetCapabilities {
	return WorkflowInvocationTargetCapabilities{
		Modes: []sessionv3Runtime.WorkflowCallMode{
			sessionv3Runtime.WorkflowCallModeInvoke,
			sessionv3Runtime.WorkflowCallModeStart,
		},
		ContextForms: []string{
			workflowInvocationContextInstructions,
			"messages",
			"defaults",
		},
		OutputPorts:       true,
		StreamOutputPorts: true,
	}
}

func (*ModelAgentInvocationAdapter) AgentInvocationCapabilities() WorkflowInvocationTargetCapabilities {
	return WorkflowInvocationTargetCapabilities{
		Modes: []sessionv3Runtime.WorkflowCallMode{
			sessionv3Runtime.WorkflowCallModeInvoke,
			sessionv3Runtime.WorkflowCallModeStart,
			sessionv3Runtime.WorkflowCallModeHandoff,
		},
		ContextForms: []string{
			workflowInvocationContextInstructions,
			"messages",
			"defaults",
		},
		OutputPorts:       true,
		StreamOutputPorts: true,
	}
}

// AdmitWorkflowModelTarget resolves the current source-backed Model through
// the existing Model runtime resolver. Credential and provider availability
// remain Model-domain concerns.
func (a *ModelAgentInvocationAdapter) AdmitWorkflowModelTarget(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) error {
	ref := workflowInvocationArtifactRef(pin)
	_, use, err := a.models.ResolveRuntimeModelWithRequestPatch(
		ctx,
		ref,
		nil,
	)
	if err != nil {
		return err
	}
	if use == nil ||
		use.RootID != pin.RootID ||
		use.ArtifactID != pin.ArtifactID ||
		use.Revision != pin.Revision ||
		use.Digest != pin.Digest {
		return errors.New(
			"workflow model target changed during invocation admission",
		)
	}
	return nil
}

// AdmitWorkflowAgentTarget resolves the Agent through the existing Agent API,
// then resolves the first currently usable Agent Model capability through the
// existing Model API. The Workflow adapter does not inspect Agent source or
// build a second Agent request contract.
func (a *ModelAgentInvocationAdapter) AdmitWorkflowAgentTarget(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) error {
	ref := workflowInvocationArtifactRef(pin)
	resolution, err := a.agents.ResolveAgent(ctx, ref)
	if err != nil {
		return err
	}
	if resolution.Agent.Ref != ref ||
		resolution.Agent.Revision != pin.Revision ||
		string(resolution.Agent.DefinitionDigest) != pin.Digest {
		return errors.New(
			"workflow agent target changed during invocation admission",
		)
	}

	var unavailable error
	for _, occurrence := range resolution.Capabilities.Occurrences {
		if occurrence.Type != declaration.TypeModel {
			continue
		}
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil {
			if occurrence.Required {
				if occurrence.Message != "" {
					return errors.New(occurrence.Message)
				}
				return errors.New(
					"required agent model capability is unavailable",
				)
			}
			continue
		}

		if _, err := a.models.ResolveRuntimeModelTarget(
			ctx,
			*occurrence.Target,
		); err != nil {
			unavailable = err
			if occurrence.Required {
				return err
			}
			continue
		}
		return nil
	}

	if unavailable != nil {
		return unavailable
	}
	return errors.New(
		"workflow agent target has no available model capability",
	)
}

func workflowInvocationArtifactRef(
	pin sessionv3Runtime.ArtifactPin,
) artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
}
