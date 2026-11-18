package workflowruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

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
	if defaults == nil || defaults.Model == nil {
		return nil, nil
	}
	ref := artifactModel.ArtifactRef{
		RootID:     model.RootID(defaults.Model.RootID),
		ArtifactID: artifactModel.ArtifactID(defaults.Model.ArtifactID),
	}
	pin, err := a.pinArtifact(ctx, ref)
	if err != nil {
		return nil, err
	}
	if defaults.Model.ExpectedRevision != 0 &&
		defaults.Model.ExpectedRevision != pin.Revision {
		return nil, errors.New("selected workflow owner Model revision changed")
	}
	if defaults.Model.ExpectedDigest != "" &&
		defaults.Model.ExpectedDigest != pin.Digest {
		return nil, errors.New("selected workflow owner Model definition changed")
	}
	if a.invocations == nil {
		return nil, fmt.Errorf(
			"%w: provider-managed Tool lowering requires a Model adapter",
			ErrBindingUnavailable,
		)
	}
	if err := a.invocations.AdmitWorkflowModelTarget(ctx, pin); err != nil {
		return nil, err
	}
	return &pin, nil
}

func workflowDirectCapabilityTarget(
	value workflowAPI.TargetPlan,
) (sessionv3Runtime.DirectCapabilityTarget, bool, error) {
	if value.Resolution == nil ||
		value.Resolution.Target == nil ||
		value.Resolution.Target.Form != composition.TargetFormDirect {
		return sessionv3Runtime.DirectCapabilityTarget{}, false, nil
	}
	target := value.Resolution.Target
	kind, err := workflowRuntimeTargetKind(value.Target.Kind)
	if err != nil {
		return sessionv3Runtime.DirectCapabilityTarget{}, false, err
	}
	output := sessionv3Runtime.DirectCapabilityTarget{
		Kind:             kind,
		Name:             string(target.Name),
		ProviderIdentity: target.ProviderIdentity,
		ProviderLocalID:  target.ProviderLocalID,
		Evidence:         string(target.Evidence),
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.DirectCapabilityTarget{}, false, err
	}
	return output, true, nil
}

func workflowRuntimeTargetKind(
	value workflowv1.TargetKind,
) (sessionv3Runtime.WorkflowTargetKind, error) {
	switch value {
	case workflowv1.TargetModel:
		return sessionv3Runtime.WorkflowTargetModel, nil
	case workflowv1.TargetAgent:
		return sessionv3Runtime.WorkflowTargetAgent, nil
	case workflowv1.TargetTool:
		return sessionv3Runtime.WorkflowTargetTool, nil
	case workflowv1.TargetService:
		return sessionv3Runtime.WorkflowTargetService, nil
	case workflowv1.TargetActivity:
		return sessionv3Runtime.WorkflowTargetActivity, nil
	case workflowv1.TargetWorkflow:
		return sessionv3Runtime.WorkflowTargetWorkflow, nil
	default:
		return "", fmt.Errorf("unsupported workflow target kind %q", value)
	}
}

func (a *Admission) bindDirectTarget(
	ctx context.Context,
	document workflowv1.WorkflowDocument,
	source workflowv1.Target,
	direct sessionv3Runtime.DirectCapabilityTarget,
	requirements []sessionv3Runtime.WorkflowCapabilityRequirement,
) (sessionv3Runtime.WorkflowTarget, error) {
	if a.direct == nil {
		return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
			"%w: direct target provider %q",
			ErrBindingUnavailable,
			direct.ProviderIdentity,
		)
	}
	plan, err := a.direct.PrepareWorkflowDirectTarget(
		ctx,
		DirectTargetPreparationRequest{
			Target:       direct,
			Requirements: requirements,
		},
	)
	if err != nil {
		return sessionv3Runtime.WorkflowTarget{}, err
	}

	target := sessionv3Runtime.WorkflowTarget{
		Kind:            direct.Kind,
		Direct:          &plan,
		Transport:       plan.Transport,
		CapabilityFacts: cloneDirectCapabilityFacts(plan.Capabilities),
		Effect: sessionv3Runtime.EffectContract{
			Kind: sessionv3Runtime.EffectKindNonIdempotent,
		},
	}
	switch direct.Kind {
	case sessionv3Runtime.WorkflowTargetTool:
		target.ToolName = direct.Name
	case sessionv3Runtime.WorkflowTargetService:
		target.ServiceID = direct.Name
		target.BindingVersion = plan.AdapterID
	case sessionv3Runtime.WorkflowTargetActivity:
		target.ActivityID = direct.Name
		target.BindingVersion = plan.AdapterID
	case sessionv3Runtime.WorkflowTargetModel,
		sessionv3Runtime.WorkflowTargetAgent:
	default:
		return sessionv3Runtime.WorkflowTarget{}, errors.New(
			"direct Workflow target kind is unavailable",
		)
	}
	return finalizeWorkflowTarget(document, source, target)
}

func workflowTargetRequirements(
	target workflowv1.Target,
) ([]sessionv3Runtime.WorkflowCapabilityRequirement, error) {
	if target.Interface == nil {
		return nil, nil
	}
	output := make(
		[]sessionv3Runtime.WorkflowCapabilityRequirement,
		0,
		len(target.Interface.Capabilities),
	)
	for _, name := range target.Interface.Capabilities {
		output = append(output, sessionv3Runtime.WorkflowCapabilityRequirement{
			Name: name,
		})
	}
	return output, nil
}

func workflowCallableRequirements(
	document workflowv1.WorkflowDocument,
	inventory workflowAPI.Inventory,
	target workflowAPI.TargetPlan,
) ([]sessionv3Runtime.WorkflowCapabilityRequirement, error) {
	names := make(map[string]struct{})
	appendContract := func(value *workflowv1.CallableInterface) {
		if value == nil {
			return
		}
		for _, name := range value.Capabilities {
			names[name] = struct{}{}
		}
	}
	appendContract(target.Target.Interface)

	for _, call := range inventory.Calls {
		selected := false
		for _, use := range call.TargetUses {
			if use.Name == target.Name {
				selected = true
				break
			}
		}
		if !selected {
			continue
		}
		appendContract(call.Call.Interface)
		if call.Call.Allowed == "" {
			continue
		}
		set, found := document.TargetSets[call.Call.Allowed]
		if !found {
			return nil, fmt.Errorf(
				"workflow target set %q is unavailable",
				call.Call.Allowed,
			)
		}
		if set.Resolved != nil {
			appendContract(&set.Resolved.Interface)
		} else {
			appendContract(set.Interface)
		}
	}

	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)

	output := make(
		[]sessionv3Runtime.WorkflowCapabilityRequirement,
		0,
		len(ordered),
	)
	for _, name := range ordered {
		output = append(output, sessionv3Runtime.WorkflowCapabilityRequirement{
			Name: name,
		})
	}
	return output, nil
}

func workflowRequirementFact(
	name string,
	target sessionv3Runtime.WorkflowTarget,
) sessionv3Runtime.WorkflowRequirementFact {
	output := sessionv3Runtime.WorkflowRequirementFact{
		Name:         name,
		Kind:         target.Kind,
		Capabilities: cloneDirectCapabilityFacts(target.CapabilityFacts),
	}
	switch target.Kind {
	case sessionv3Runtime.WorkflowTargetModel,
		sessionv3Runtime.WorkflowTargetAgent,
		sessionv3Runtime.WorkflowTargetTool:
		if target.Artifact != nil {
			value := *target.Artifact
			output.Artifact = &value
		}
	case sessionv3Runtime.WorkflowTargetMCPTool:
		if target.MCP != nil {
			value := target.MCP.Server
			output.Artifact = &value
		}
	case sessionv3Runtime.WorkflowTargetProviderTool:
		if target.ProviderTool != nil {
			value := target.ProviderTool.Choice.Artifact
			output.Artifact = &value
		}
	case sessionv3Runtime.WorkflowTargetService:
		output.BindingID = target.ServiceID
		output.BindingVersion = target.BindingVersion
	case sessionv3Runtime.WorkflowTargetActivity:
		output.BindingID = target.ActivityID
		output.BindingVersion = target.BindingVersion
	default:
	}
	if target.Direct != nil {
		output.Direct = cloneDirectCapabilityPlan(target.Direct)
	}
	return output
}

func cloneDirectCapabilityPlan(
	value *sessionv3Runtime.DirectCapabilityPlan,
) *sessionv3Runtime.DirectCapabilityPlan {
	if value == nil {
		return nil
	}
	output := *value
	output.Facts = value.Facts.Clone()
	output.Capabilities = cloneWorkflowCapabilityFacts(value.Capabilities)
	output.Transport.Modes = append(
		[]sessionv3Runtime.WorkflowCallMode(nil),
		value.Transport.Modes...,
	)
	output.Transport.ContextForms = append(
		[]string(nil),
		value.Transport.ContextForms...,
	)
	return &output
}

func cloneWorkflowCapabilityFacts(
	values []sessionv3Runtime.WorkflowCapabilityFact,
) []sessionv3Runtime.WorkflowCapabilityFact {
	if len(values) == 0 {
		return nil
	}
	output := make([]sessionv3Runtime.WorkflowCapabilityFact, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Details = values[index].Details.Clone()
	}
	return output
}
