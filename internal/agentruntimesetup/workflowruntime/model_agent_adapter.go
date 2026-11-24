package workflowruntime

import (
	"context"
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/contributionruntime"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/modelruntime"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	textAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

const (
	invocationContextInstructions = "instructions"
	invocationContextMessages     = "messages"
	invocationContextDefaults     = "defaults"
)

// ModelAgentInvocationAdapter is Workflow setup composition over existing
// Model and Agent APIs. It does not add a Workflow-specific Model or Agent
// domain request contract.
type ModelAgentInvocationAdapter struct {
	models        *modelruntime.SourceResolver
	agents        *agentAPI.Service
	artifacts     artifact.API
	contributions *contributionruntime.Preparer
}

type ModelAgentAdapterOption func(*[]contributionruntime.Option)

func WithInstructionSkillRenderer(
	renderer contributionruntime.InstructionSkillRenderer,
) ModelAgentAdapterOption {
	return func(options *[]contributionruntime.Option) {
		*options = append(*options, contributionruntime.WithInstructionSkillRenderer(renderer))
	}
}

func WithMCPContributionPreparer(
	preparer contributionruntime.MCPContributionPreparer,
) ModelAgentAdapterOption {
	return func(options *[]contributionruntime.Option) {
		*options = append(*options, contributionruntime.WithMCPContributionPreparer(preparer))
	}
}

func NewModelAgentInvocationAdapter(
	models *modelruntime.SourceResolver,
	agents *agentAPI.Service,
	artifacts artifact.API,
	workspaces *workspaceAPI.Service,
	texts *textAPI.Service,
	options ...ModelAgentAdapterOption,
) (*ModelAgentInvocationAdapter, error) {
	if models == nil {
		return nil, errors.New(
			"workflow model invocation adapter dependencies are incomplete",
		)
	}
	contributionOptions := make([]contributionruntime.Option, 0, len(options))
	for _, option := range options {
		if option != nil {
			option(&contributionOptions)
		}
	}
	contributions, err := contributionruntime.New(
		artifacts,
		agents,
		workspaces,
		texts,
		contributionOptions...,
	)
	if err != nil {
		return nil, err
	}
	return &ModelAgentInvocationAdapter{
		models:        models,
		agents:        agents,
		artifacts:     artifacts,
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
			invocationContextInstructions,
			invocationContextMessages,
			invocationContextDefaults,
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
			invocationContextInstructions,
			invocationContextMessages,
			invocationContextDefaults,
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
	ref := contributionruntime.ArtifactRefForPin(pin)
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
	if a == nil || a.agents == nil || a.artifacts == nil {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow agent owner model resolver is unavailable",
		)
	}
	ref := contributionruntime.ArtifactRefForPin(pin)
	resolved, err := a.agents.ResolveAgent(ctx, ref)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if resolved.Agent.Ref != ref ||
		resolved.Agent.Revision != pin.Revision ||
		string(resolved.Agent.DefinitionDigest) != pin.Digest {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow agent changed during owner model resolution",
		)
	}
	for _, occurrence := range resolved.Capabilities.Occurrences {
		if occurrence.Type != declaration.TypeModel {
			continue
		}
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil ||
			occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			if occurrence.Required {
				return sessionv3Runtime.ArtifactPin{}, errors.New(
					"workflow agent required model contribution is unavailable",
				)
			}
			continue
		}
		model, pinErr := contributionruntime.PinArtifactRef(
			ctx,
			a.artifacts,
			*occurrence.Target.Artifact,
		)
		if pinErr != nil {
			return sessionv3Runtime.ArtifactPin{}, pinErr
		}
		return model, nil
	}
	return sessionv3Runtime.ArtifactPin{}, errors.New(
		"workflow agent target has no selected model capability",
	)
}

func (a *ModelAgentInvocationAdapter) PrepareWorkflowModelInvocation(
	ctx context.Context,
	model sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.WorkflowPreparedInvocation, error) {
	return a.PrepareWorkflowModelCallInvocation(ctx, model, nil)
}

func (a *ModelAgentInvocationAdapter) PrepareWorkflowModelCallInvocation(
	ctx context.Context,
	model sessionv3Runtime.ArtifactPin,
	defaults *sessionv3Runtime.SessionDefaultsOverride,
) (sessionv3Runtime.WorkflowPreparedInvocation, error) {
	if err := a.AdmitWorkflowModelTarget(ctx, model); err != nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, err
	}
	selection := model.Selection()
	value, err := a.contributions.Prepare(
		ctx,
		contributionruntime.Request{
			Model:            &selection,
			DefaultsOverride: defaults,
			SpecificOwner:    sessionv3Runtime.ConfigurationOwnerInvocation,
		},
	)
	if err != nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, err
	}
	if value.Model == nil || *value.Model != model {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, errors.New(
			"workflow model contribution facts differ from the selected model",
		)
	}
	return value, nil
}

func (a *ModelAgentInvocationAdapter) PrepareWorkflowAgentInvocation(
	ctx context.Context,
	agent sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.WorkflowPreparedInvocation, error) {
	return a.PrepareWorkflowAgentCallInvocation(ctx, agent, nil)
}

func (a *ModelAgentInvocationAdapter) PrepareWorkflowAgentCallInvocation(
	ctx context.Context,
	agent sessionv3Runtime.ArtifactPin,
	defaults *sessionv3Runtime.SessionDefaultsOverride,
) (sessionv3Runtime.WorkflowPreparedInvocation, error) {
	if a == nil || a.contributions == nil {
		return sessionv3Runtime.WorkflowPreparedInvocation{}, errors.New(
			"workflow contribution runtime is unavailable",
		)
	}
	selection := agent.Selection()
	value, err := a.contributions.Prepare(
		ctx,
		contributionruntime.Request{
			Agent:            &selection,
			DefaultsOverride: defaults,
			SpecificOwner:    sessionv3Runtime.ConfigurationOwnerInvocation,
		},
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

func (a *ModelAgentInvocationAdapter) setDefaultMCPPreparer(
	preparer contributionruntime.MCPContributionPreparer,
) {
	if a != nil && a.contributions != nil {
		a.contributions.SetDefaultMCPPreparer(preparer)
	}
}
