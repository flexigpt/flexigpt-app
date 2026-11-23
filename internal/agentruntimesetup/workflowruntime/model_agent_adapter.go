package workflowruntime

import (
	"context"
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/contributionruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

// ModelAgentInvocationAdapter is Workflow setup composition over existing
// Model and Agent APIs. It does not add a Workflow-specific Model or Agent
// domain request contract.
type ModelAgentInvocationAdapter struct {
	models        *modelruntime.SourceResolver
	contributions *contributionruntime.Preparer
}

func NewModelAgentInvocationAdapter(
	models *modelruntime.SourceResolver,
	agents *agentAPI.Service,
	artifacts artifact.API,
	workspaces *workspaceAPI.Service,
	texts *textAPI.Service,
) (*ModelAgentInvocationAdapter, error) {
	if models == nil {
		return nil, errors.New(
			"workflow model invocation adapter dependencies are incomplete",
		)
	}
	contributions, err := contributionruntime.New(
		artifacts,
		agents,
		workspaces,
		texts,
	)
	if err != nil {
		return nil, err
	}
	return &ModelAgentInvocationAdapter{
		models:        models,
		contributions: contributions,
	}, nil
}

func (*ModelAgentInvocationAdapter) ModelInvocationCapabilities() WorkflowInvocationTargetCapabilities {
	return WorkflowInvocationTargetCapabilities{
		Modes: []sessionv3Runtime.WorkflowCallMode{
			sessionv3Runtime.WorkflowCallModeInvoke,
			sessionv3Runtime.WorkflowCallModeStart,
		},
		ContextForms: []string{
			"instructions",
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
			"instructions",
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
	_, err := a.PrepareWorkflowAgentInvocation(ctx, pin)
	return err
}

// ResolveWorkflowAgentOwnerModel resolves and pins the Model selected by an
// Artifact-backed Agent. Provider SDK Tools use this owner when a Workflow
// invocation supplies an Agent but no explicit Model.
func (a *ModelAgentInvocationAdapter) ResolveWorkflowAgentOwnerModel(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.ArtifactPin, error) {
	invocation, err := a.PrepareWorkflowAgentInvocation(ctx, pin)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if invocation.Model == nil {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow agent target has no prepared model contribution",
		)
	}
	if err := a.AdmitWorkflowModelTarget(ctx, *invocation.Model); err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	return *invocation.Model, nil
}

func (a *ModelAgentInvocationAdapter) PrepareWorkflowModelInvocation(
	ctx context.Context,
	model sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.WorkflowPreparedInvocation, error) {
	if err := a.AdmitWorkflowModelTarget(ctx, model); err != nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, err
	}
	value := sessionv3Runtime.WorkflowPreparedInvocation{
		Model:           &model,
		OwnerProvenance: sessionv3Runtime.ConfigurationOwnerModel,
		ContributionOrder: []sessionv3Runtime.PreparedContributionOrder{{
			Kind:  sessionv3Runtime.PreparedContributionKindModel,
			Key:   sessionv3Runtime.PreparedContributionKey(sessionv3Runtime.PreparedContributionKindModel, model),
			Owner: sessionv3Runtime.ConfigurationOwnerModel,
		}},
	}
	return value, value.Validate()
}

func (a *ModelAgentInvocationAdapter) PrepareWorkflowAgentInvocation(
	ctx context.Context,
	agent sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.WorkflowPreparedInvocation, error) {
	if a == nil || a.contributions == nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, errors.New(
			"workflow contribution runtime is unavailable",
		)
	}
	selection := selectionForPreparedPin(agent)
	value, err := a.contributions.Prepare(
		ctx,
		contributionruntime.Request{Agent: &selection},
	)
	if err != nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, err
	}
	if value.Agent == nil || *value.Agent != agent {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, errors.New(
			"workflow agent contribution facts differ from the selected agent",
		)
	}
	if value.Model == nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, errors.New(
			"workflow agent target has no prepared model contribution",
		)
	}
	if err := a.AdmitWorkflowModelTarget(ctx, *value.Model); err != nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, err
	}
	return value, nil
}

func workflowInvocationArtifactRef(
	pin sessionv3Runtime.ArtifactPin,
) artifactModel.ArtifactRef {
	return artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
}

func selectionForPreparedPin(
	value sessionv3Runtime.ArtifactPin,
) sessionv3Runtime.ArtifactSelection {
	return sessionv3Runtime.ArtifactSelection{
		RootID:           value.RootID,
		ArtifactID:       value.ArtifactID,
		ExpectedRevision: value.Revision,
		ExpectedDigest:   value.Digest,
	}
}
