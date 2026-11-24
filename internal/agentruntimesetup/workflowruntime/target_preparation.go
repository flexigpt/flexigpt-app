package workflowruntime

import (
	"context"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/workbinding"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/contributionruntime"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

const wildcardContextForm = "*"

// ProviderManagedWorkflowToolPlanner is implemented by the Tool domain. It
// returns only a provider Tool choice; Model invocation remains the typed
// Model child plan owned by Workflow Runtime.
type ProviderManagedWorkflowToolPlanner interface {
	PrepareProviderManagedWorkflowTool(
		ctx context.Context,
		pin sessionv3Runtime.ArtifactPin,
	) (sessionv3Runtime.ProviderManagedToolChoice, error)
}

func (a *Admission) workflowOwnerModel(
	ctx context.Context,
	defaults *sessionv3Runtime.SessionDefaults,
) (*sessionv3Runtime.ArtifactPin, error) {
	if defaults == nil {
		return nil, nil
	}

	if defaults.Model != nil {
		pin, err := contributionruntime.PinArtifactSelection(
			ctx,
			a.artifacts,
			*defaults.Model,
		)
		if err != nil {
			return nil, err
		}
		if a.invocations == nil {
			return nil, fmt.Errorf(
				"%w: provider-managed tool lowering requires a model adapter",
				ErrBindingUnavailable,
			)
		}
		if err := a.invocations.AdmitWorkflowModelTarget(ctx, pin); err != nil {
			return nil, err
		}
		return &pin, nil
	}

	if defaults.Agent == nil || a.invocations == nil {
		return nil, nil
	}
	agent, err := contributionruntime.PinArtifactSelection(
		ctx,
		a.artifacts,
		*defaults.Agent,
	)
	if err != nil {
		return nil, err
	}
	model, err := a.invocations.ResolveWorkflowAgentOwnerModel(ctx, agent)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

// prepareRequirement preserves root requirements as named prerequisites.
// It does not create a callable target, WorkItem, direct target plan, or
// generic callable capability fact.
func (a *Admission) prepareRequirement(
	ctx context.Context,
	requirement workflowAPI.TargetPlan,
	closure *workflowAdmissionClosure,
) (sessionv3Runtime.WorkflowRequirementFact, error) {
	target := requirement.Target
	result := sessionv3Runtime.WorkflowRequirementFact{
		Name: requirement.Name,
	}

	switch target.Kind {
	case workflowv1.TargetService:
		binding, found := a.bindings.Service(target.Ref)
		if !found {
			return sessionv3Runtime.WorkflowRequirementFact{}, fmt.Errorf(
				"%w: service requirement %q",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		result.Kind = sessionv3Runtime.WorkflowTargetService
		result.BindingID = binding.ID()
		result.BindingVersion = binding.Version()

	case workflowv1.TargetActivity:
		binding, found := a.bindings.Activity(target.Ref)
		if !found {
			return sessionv3Runtime.WorkflowRequirementFact{}, fmt.Errorf(
				"%w: activity requirement %q",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		result.Kind = sessionv3Runtime.WorkflowTargetActivity
		result.BindingID = binding.ID()
		result.BindingVersion = binding.Version()

	case workflowv1.TargetModel:
		if a.invocations == nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, ErrBindingUnavailable
		}
		pin, err := a.pinInvocationTarget(ctx, requirement, closure)
		if err != nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, err
		}
		if err := a.invocations.AdmitWorkflowModelTarget(ctx, pin); err != nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, err
		}
		result.Kind = sessionv3Runtime.WorkflowTargetModel
		result.Artifact = &pin

	case workflowv1.TargetAgent:
		if a.invocations == nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, ErrBindingUnavailable
		}
		pin, err := a.pinInvocationTarget(ctx, requirement, closure)
		if err != nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, err
		}
		if err := a.invocations.AdmitWorkflowAgentTarget(ctx, pin); err != nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, err
		}
		result.Kind = sessionv3Runtime.WorkflowTargetAgent
		result.Artifact = &pin

	case workflowv1.TargetTool:
		ref, err := requireWorkflowTargetArtifactRef(
			requirement,
			"tool requirement",
		)
		if err != nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, err
		}
		pin, err := a.pinClosureArtifact(ctx, closure, ref)
		if err != nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, err
		}
		if target.Server != nil {
			if a.mcpTools == nil {
				return sessionv3Runtime.WorkflowRequirementFact{}, ErrBindingUnavailable
			}
			if _, err := a.mcpTools.PrepareWorkflowMCPTool(
				ctx,
				pin,
				target.Ref,
			); err != nil {
				return sessionv3Runtime.WorkflowRequirementFact{}, err
			}
			result.Kind = sessionv3Runtime.WorkflowTargetMCPTool
		} else {
			if _, err := a.tools.IsProviderManagedWorkflowTool(ctx, pin); err != nil {
				return sessionv3Runtime.WorkflowRequirementFact{}, err
			}
			result.Kind = sessionv3Runtime.WorkflowTargetTool
		}
		result.Artifact = &pin

	case workflowv1.TargetWorkflow:
		ref, err := requireWorkflowTargetArtifactRef(
			requirement,
			"nested workflow requirement",
		)
		if err != nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, err
		}
		if _, found := closure.workflow(ref); !found {
			return sessionv3Runtime.WorkflowRequirementFact{}, fmt.Errorf(
				"%w: nested workflow requirement %q is absent from the static artifact closure",
				ErrBindingUnavailable,
				requirement.Name,
			)
		}
		pin, err := a.pinClosureArtifact(ctx, closure, ref)
		if err != nil {
			return sessionv3Runtime.WorkflowRequirementFact{}, err
		}
		result.Kind = sessionv3Runtime.WorkflowTargetWorkflow
		result.Artifact = &pin

	default:
		return sessionv3Runtime.WorkflowRequirementFact{}, fmt.Errorf(
			"%w: unsupported workflow requirement kind %q",
			ErrBindingUnavailable,
			target.Kind,
		)
	}

	if err := result.Validate(); err != nil {
		return sessionv3Runtime.WorkflowRequirementFact{}, err
	}
	return result, nil
}

func (a *Admission) prepareBoundTarget(
	ctx context.Context,
	targetPlan workflowAPI.TargetPlan,
	target sessionv3Runtime.WorkflowTarget,
) (sessionv3Runtime.WorkflowTarget, error) {
	switch target.Kind {
	case sessionv3Runtime.WorkflowTargetModel:
		if a.invocations == nil || target.Artifact == nil {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: model target %q has no invocation adapter",
				ErrBindingUnavailable,
				targetPlan.Name,
			)
		}
		target.Transport = invocationTransport(
			a.invocations.ModelInvocationCapabilities(),
		)

	case sessionv3Runtime.WorkflowTargetAgent:
		if a.invocations == nil || target.Artifact == nil {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: agent target %q has no invocation adapter",
				ErrBindingUnavailable,
				targetPlan.Name,
			)
		}

		target.Transport = invocationTransport(
			a.invocations.AgentInvocationCapabilities(),
		)

	case sessionv3Runtime.WorkflowTargetTool,
		sessionv3Runtime.WorkflowTargetMCPTool:
		target.Transport = workflowToolTransport()
		target.KeyedCapable = false

	case sessionv3Runtime.WorkflowTargetProviderTool:
		if target.ProviderTool == nil || a.invocations == nil {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: provider-managed tool target %q has no model owner",
				ErrBindingUnavailable,
				targetPlan.Name,
			)
		}
		target.Transport = invocationTransport(
			a.invocations.ModelInvocationCapabilities(),
		)

	case sessionv3Runtime.WorkflowTargetService:
		binding, found := a.bindings.Service(target.ServiceID)
		if !found || binding.Version() != target.BindingVersion {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: service %q version %q",
				ErrBindingUnavailable,
				target.ServiceID,
				target.BindingVersion,
			)
		}
		target.KeyedCapable = target.KeyedCapable && workbinding.SupportsNativeIdempotencyKeying(binding)
		if target.KeyedCapable {
			target.Effect.Kind = sessionv3Runtime.EffectKindNonIdempotent
			target.Effect.IdempotencyKey = ""
			target.Effect.KeyedNative = false
		}
		target.Transport = externalBindingTransport(binding)

	case sessionv3Runtime.WorkflowTargetActivity:
		binding, found := a.bindings.Activity(target.ActivityID)
		if !found || binding.Version() != target.BindingVersion {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: activity %q version %q",
				ErrBindingUnavailable,
				target.ActivityID,
				target.BindingVersion,
			)
		}
		target.KeyedCapable = target.KeyedCapable && workbinding.SupportsNativeIdempotencyKeying(binding)
		if target.KeyedCapable {
			target.Effect.Kind = sessionv3Runtime.EffectKindNonIdempotent
			target.Effect.IdempotencyKey = ""
			target.Effect.KeyedNative = false
		}
		target.Transport = externalBindingTransport(binding)

	case sessionv3Runtime.WorkflowTargetWorkflow:
		target.Transport = sessionv3Runtime.WorkflowTargetTransport{
			Modes: []sessionv3Runtime.WorkflowCallMode{
				sessionv3Runtime.WorkflowCallModeInvoke,
				sessionv3Runtime.WorkflowCallModeStart,
			},
			ContextForms:      []string{wildcardContextForm},
			InputPorts:        true,
			FinalOutputPorts:  true,
			StreamOutputPorts: true,
		}

	default:
		return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
			"%w: unsupported target kind %q",
			ErrBindingUnavailable,
			target.Kind,
		)
	}

	if err := target.Transport.Validate(); err != nil {
		return sessionv3Runtime.WorkflowTarget{}, err
	}
	return target, nil
}

func workflowToolTransport() sessionv3Runtime.WorkflowTargetTransport {
	return sessionv3Runtime.WorkflowTargetTransport{
		Modes: []sessionv3Runtime.WorkflowCallMode{
			sessionv3Runtime.WorkflowCallModeInvoke,
			sessionv3Runtime.WorkflowCallModeStart,
		},
		ContextForms:      nil,
		InputPorts:        false,
		FinalOutputPorts:  true,
		StreamOutputPorts: false,
	}
}

// invocationTransport converts current Model or Agent adapter transport facts
// into the runtime-neutral target descriptor persisted in WorkflowProgram.
//
// The adapter remains the owner of Model and Agent native request behavior.
func invocationTransport(
	value WorkflowInvocationTargetCapabilities,
) sessionv3Runtime.WorkflowTargetTransport {
	return sessionv3Runtime.WorkflowTargetTransport{
		Modes: append(
			[]sessionv3Runtime.WorkflowCallMode(nil),
			value.Modes...,
		),
		ContextForms: append(
			[]string(nil),
			value.ContextForms...,
		),
		InputPorts:        value.InputPorts,
		FinalOutputPorts:  value.OutputPorts,
		StreamOutputPorts: value.StreamOutputPorts,
	}
}

// externalBindingTransport exposes only transport explicitly declared by a
// registered Service or Activity binding. Final-output delivery remains
// Runtime-owned, but the binding must explicitly opt in to that callable
// interface capability.
func externalBindingTransport(
	value WorkBinding,
) sessionv3Runtime.WorkflowTargetTransport {
	output := sessionv3Runtime.WorkflowTargetTransport{
		Modes: []sessionv3Runtime.WorkflowCallMode{
			sessionv3Runtime.WorkflowCallModeInvoke,
			sessionv3Runtime.WorkflowCallModeStart,
		},
	}

	capability, found := value.(WorkBindingPortCapability)
	if !found {
		return output
	}

	facts := capability.WorkflowPortCapabilities()
	output.InputPorts = facts.InputPorts
	output.FinalOutputPorts = facts.FinalOutputPorts
	output.StreamOutputPorts = facts.OutputStream
	if facts.Context {
		output.ContextForms = []string{wildcardContextForm}
	}

	return output
}
