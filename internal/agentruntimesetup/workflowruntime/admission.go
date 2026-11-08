// Package workflowruntime compiles Artifact-backed Workflow declarations into
// runtime-neutral Session v3 WorkflowProgram snapshots. It does not execute
// Workflow behavior.
package workflowruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

const (
	InterpreterID      = sessionv3Runtime.WorkflowInterpreterID
	InterpreterVersion = sessionv3Runtime.WorkflowInterpreterVersion
)

var ErrBindingUnavailable = errors.New("workflow target binding is unavailable")

// ToolTargetInspector is the narrow setup-domain boundary used during
// admission to prevent provider-managed SDK Tools from becoming local Tool
// WorkItems. It resolves only a pinned Tool target and exposes no Artifact
// implementation details to Session Runtime.
type ToolTargetInspector interface {
	IsProviderManagedWorkflowTool(
		ctx context.Context,
		pin sessionv3Runtime.ArtifactPin,
	) (bool, error)
}

// WorkflowInvocationTargetCapabilities describe the current typed Workflow
// transport that a Model or Agent adapter can bind. These are setup-owned
// adapter facts, not a Model or Agent domain contract.
type WorkflowInvocationTargetCapabilities struct {
	Modes             []sessionv3Runtime.WorkflowCallMode
	ContextForms      []string
	InputPorts        bool
	OutputPorts       bool
	StreamOutputPorts bool
}

// WorkflowInvocationTargetAdapter verifies a current Model or Agent target
// through existing domain APIs. Domain APIs remain independent of Workflow.
type WorkflowInvocationTargetAdapter interface {
	AdmitWorkflowModelTarget(
		ctx context.Context,
		pin sessionv3Runtime.ArtifactPin,
	) error
	AdmitWorkflowAgentTarget(
		ctx context.Context,
		pin sessionv3Runtime.ArtifactPin,
	) error
	ModelInvocationCapabilities() WorkflowInvocationTargetCapabilities
	AgentInvocationCapabilities() WorkflowInvocationTargetCapabilities
}

// WorkBindingRequest is the typed application-domain boundary for a Workflow
// Service or Activity invocation. Ports are already durable Runtime facts.
// Stream is process-private and can only emit to declared streamed ports.
type WorkBindingRequest struct {
	Input   sessionv3Runtime.CanonicalValue
	Context sessionv3Runtime.CanonicalValue
	Ports   []sessionv3Runtime.PortInput
	Stream  sessionv3Runtime.WorkPortStream
}

// WorkBindingWaitRequest is the typed domain boundary for a Service or
// Activity wait resolution. It contains no Runtime private handles.
type WorkBindingWaitRequest struct {
	Wait       sessionv3Runtime.Wait
	Resolution sessionv3Runtime.WaitResolution
	Response   sessionv3Runtime.CanonicalValue
	Input      sessionv3Runtime.CanonicalValue
	Context    sessionv3Runtime.CanonicalValue
	Ports      []sessionv3Runtime.PortInput
}

// WorkBinding is an application-owned Service or Activity runtime binding.
// Artifact Workflow declarations only persist its stable ID and version.
type WorkBinding interface {
	ID() string
	Version() string
	Effect() sessionv3Runtime.EffectContract
	Bind(
		ctx context.Context,
		request WorkBindingRequest,
	) (sessionv3Runtime.SemanticBinding, any, error)
	Execute(
		ctx context.Context,
		request WorkBindingRequest,
		state any,
	) (sessionv3Runtime.WorkExecutionResult, error)
	ResolveWait(
		ctx context.Context,
		request WorkBindingWaitRequest,
		state any,
	) (sessionv3Runtime.WaitResolutionResult, error)
}

type WorkBindingPortCapabilities struct {
	Context      bool
	InputPorts   bool
	OutputStream bool
}

type WorkBindingPortCapability interface {
	WorkflowPortCapabilities() WorkBindingPortCapabilities
}

type BindingRegistry struct {
	mu       sync.RWMutex
	bindings map[string]WorkBinding
}

func NewBindingRegistry() *BindingRegistry {
	return &BindingRegistry{
		bindings: map[string]WorkBinding{},
	}
}

func (r *BindingRegistry) RegisterService(
	value WorkBinding,
) error {
	return r.register("service", value)
}

func (r *BindingRegistry) RegisterActivity(
	value WorkBinding,
) error {
	return r.register("activity", value)
}

func (r *BindingRegistry) Service(
	id string,
) (WorkBinding, bool) {
	return r.get("service", id)
}

func (r *BindingRegistry) Activity(
	id string,
) (WorkBinding, bool) {
	return r.get("activity", id)
}

func (r *BindingRegistry) register(
	kind string,
	value WorkBinding,
) error {
	if r == nil ||
		value == nil ||
		value.ID() == "" ||
		value.Version() == "" {
		return errors.New("invalid Workflow runtime binding")
	}
	effect := value.Effect()
	if err := effect.Validate(); err != nil {
		return fmt.Errorf(
			"workflow %s binding %q effect: %w",
			kind,
			value.ID(),
			err,
		)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := kind + ":" + value.ID()
	if _, found := r.bindings[key]; found {
		return fmt.Errorf(
			"duplicate Workflow %s binding %q",
			kind,
			value.ID(),
		)
	}
	r.bindings[key] = value
	return nil
}

func (r *BindingRegistry) get(
	kind string,
	id string,
) (WorkBinding, bool) {
	if r == nil {
		return nil, false
	}

	r.mu.RLock()
	value, found := r.bindings[kind+":"+id]
	r.mu.RUnlock()
	return value, found
}

type Admission struct {
	workflows   *workflowAPI.Service
	artifacts   artifact.API
	bindings    *BindingRegistry
	tools       ToolTargetInspector
	invocations WorkflowInvocationTargetAdapter
}

func NewAdmission(
	workflows *workflowAPI.Service,
	artifacts artifact.API,
	bindings *BindingRegistry,
	tools ToolTargetInspector,
	invocationAdapters ...WorkflowInvocationTargetAdapter,
) (*Admission, error) {
	if workflows == nil ||
		artifacts == nil ||
		bindings == nil ||
		tools == nil {
		return nil, errors.New(
			"workflow admission dependencies are incomplete",
		)
	}
	if len(invocationAdapters) > 1 {
		return nil, errors.New(
			"workflow admission has more than one model and agent invocation adapter",
		)
	}
	var invocations WorkflowInvocationTargetAdapter
	if len(invocationAdapters) == 1 {
		invocations = invocationAdapters[0]
	}
	return &Admission{
		workflows:   workflows,
		artifacts:   artifacts,
		bindings:    bindings,
		tools:       tools,
		invocations: invocations,
	}, nil
}

func (a *Admission) AdmitWorkflow(
	ctx context.Context,
	request sessionv3Runtime.WorkflowAdmissionRequest,
) (sessionv3Runtime.WorkflowAdmissionResult, error) {
	if request.Workflow.RootID == "" ||
		request.Workflow.ArtifactID == "" ||
		!request.Input.Valid() {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"invalid Workflow admission request",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(request.Workflow.RootID),
		ArtifactID: artifactModel.ArtifactID(request.Workflow.ArtifactID),
	}
	return a.admit(
		ctx,
		ref,
		request.Workflow,
		request.Input,
		make(map[artifactModel.ArtifactRef]struct{}),
		true,
	)
}

// AdmitWorkflowTriggers compiles an input-unbound Workflow program for
// Session-owned trigger registration. Trigger occurrence input is bound only
// when a durable occurrence is accepted.
func (a *Admission) AdmitWorkflowTriggers(
	ctx context.Context,
	request sessionv3Runtime.WorkflowTriggerRegistrationRequest,
) (sessionv3Runtime.WorkflowAdmissionResult, error) {
	if request.Workflow.RootID == "" || request.Workflow.ArtifactID == "" {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"invalid workflow trigger registration request",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(request.Workflow.RootID),
		ArtifactID: artifactModel.ArtifactID(request.Workflow.ArtifactID),
	}
	result, err := a.admit(
		ctx,
		ref,
		request.Workflow,
		nil,
		make(map[artifactModel.ArtifactRef]struct{}),
		false,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}

	program, err := sessionv3Runtime.DecodeCanonicalValue[sessionv3Runtime.WorkflowProgram](result.Program.Payload)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if len(program.Triggers) == 0 {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"workflow declares no triggers",
		)
	}
	return result, nil
}

// VerifyArtifactPin is the setup-owned current Artifact verification boundary
// used by Workflow binder dispatch. Session Runtime receives this only through
// the narrow setup interface and never imports Artifact Store packages.
func (a *Admission) VerifyArtifactPin(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) error {
	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
	current, err := a.pinArtifact(ctx, ref)
	if err != nil {
		return err
	}
	if current != pin {
		return errors.New("workflow Artifact pin changed since admission")
	}
	return nil
}

func (a *Admission) admit(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	selection sessionv3Runtime.ArtifactSelection,
	input sessionv3Runtime.CanonicalValue,
	active map[artifactModel.ArtifactRef]struct{},
	inputBound bool,
) (sessionv3Runtime.WorkflowAdmissionResult, error) {
	if _, found := active[ref]; found {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"nested Workflow cycle",
		)
	}
	active[ref] = struct{}{}
	defer delete(active, ref)

	plan, err := a.workflows.ResolvePlan(ctx, ref)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if !plan.CompositionComplete {
		return sessionv3Runtime.WorkflowAdmissionResult{}, fmt.Errorf(
			"%w: unresolved required Workflow capability",
			ErrBindingUnavailable,
		)
	}
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != plan.Revision {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"selected Workflow revision changed",
		)
	}
	if selection.ExpectedDigest != "" &&
		selection.ExpectedDigest != string(plan.Digest) {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"selected Workflow definition changed",
		)
	}

	pin, err := a.pinArtifact(ctx, plan.Workflow)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if pin.Revision != plan.Revision ||
		pin.Digest != string(plan.Digest) {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"workflow Artifact changed during admission",
		)
	}

	var program sessionv3Runtime.WorkflowProgram
	if inputBound {
		program, err = compileWorkflowProgram(plan.Document, input)
	} else {
		program, err = compileWorkflowProgramTemplate(plan.Document)
	}
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := a.bindTargets(ctx, plan, &program, active); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := a.verifyWorkflowInvocationTransport(plan.Inventory); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := rejectUnsupportedWorkflowToolTransport(plan.Inventory); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}

	if err := program.Validate(); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if inputBound {
		if err := program.ValidateInput(); err != nil {
			return sessionv3Runtime.WorkflowAdmissionResult{}, err
		}
	}

	return sessionv3Runtime.WorkflowAdmissionResult{
		Program: sessionv3Runtime.ProgramSnapshot{
			InterpreterID:      InterpreterID,
			InterpreterVersion: InterpreterVersion,
			Artifact:           &pin,
			DefinitionDigest:   pin.Digest,
			Payload:            sessionv3Runtime.MustCanonicalValue(program),
		},
	}, nil
}

func (a *Admission) bindTargets(
	ctx context.Context,
	plan workflowAPI.Plan,
	program *sessionv3Runtime.WorkflowProgram,
	active map[artifactModel.ArtifactRef]struct{},
) error {
	if program == nil {
		return errors.New("workflow program is unavailable")
	}

	for _, targetPlan := range plan.Inventory.Targets {
		if _, duplicate := program.Targets[targetPlan.Name]; duplicate {
			return fmt.Errorf(
				"workflow target %q is declared more than once",
				targetPlan.Name,
			)
		}

		bound, err := a.bindTarget(
			ctx,
			plan.Document,
			targetPlan,
			active,
		)
		if err != nil {
			return err
		}
		program.Targets[targetPlan.Name] = bound
	}
	return nil
}

func (a *Admission) bindTarget(
	ctx context.Context,
	document workflowv1.WorkflowDocument,
	targetPlan workflowAPI.TargetPlan,
	active map[artifactModel.ArtifactRef]struct{},
) (sessionv3Runtime.WorkflowTarget, error) {
	target := targetPlan.Target

	switch target.Kind {
	case workflowv1.TargetService:
		value, found := a.bindings.Service(target.Ref)
		if !found {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: service %q",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		bound := sessionv3Runtime.WorkflowTarget{
			Kind:           sessionv3Runtime.WorkflowTargetService,
			ServiceID:      value.ID(),
			BindingVersion: value.Version(),
			Effect:         value.Effect(),
		}
		return finalizeWorkflowTarget(document, target, bound)

	case workflowv1.TargetActivity:
		value, found := a.bindings.Activity(target.Ref)
		if !found {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: activity %q",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		bound := sessionv3Runtime.WorkflowTarget{
			Kind:           sessionv3Runtime.WorkflowTargetActivity,
			ActivityID:     value.ID(),
			BindingVersion: value.Version(),
			Effect:         value.Effect(),
		}
		return finalizeWorkflowTarget(document, target, bound)

	case workflowv1.TargetTool:
		ref, found := workflowTargetArtifactRef(targetPlan)
		if !found {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: Tool target %q",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		pin, err := a.pinArtifact(ctx, ref)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}
		if target.Server != nil {
			return finalizeWorkflowTarget(
				document,
				target,
				sessionv3Runtime.WorkflowTarget{
					Kind: sessionv3Runtime.WorkflowTargetMCPTool,
					MCP: &sessionv3Runtime.WorkflowMCP{
						Server:   pin,
						ToolName: target.Ref,
					},
					Effect: sessionv3Runtime.EffectContract{
						Kind: sessionv3Runtime.EffectKindNonIdempotent,
					},
				},
			)
		}

		providerManaged, inspectErr := a.tools.IsProviderManagedWorkflowTool(
			ctx,
			pin,
		)
		if inspectErr != nil {
			return sessionv3Runtime.WorkflowTarget{}, inspectErr
		}
		if providerManaged {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: provider-managed tool target %q requires a typed model execution plan",
				ErrBindingUnavailable,
				target.Ref,
			)
		}

		return finalizeWorkflowTarget(
			document,
			target,
			sessionv3Runtime.WorkflowTarget{
				Kind:     sessionv3Runtime.WorkflowTargetTool,
				Artifact: &pin,
				ToolName: target.Ref,
				Effect: sessionv3Runtime.EffectContract{
					Kind: sessionv3Runtime.EffectKindNonIdempotent,
				},
			},
		)

	case workflowv1.TargetWorkflow:
		ref, found := workflowTargetArtifactRef(targetPlan)
		if !found {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: nested Workflow target %q",
				ErrBindingUnavailable,
				target.Ref,
			)
		}

		pin, err := a.pinArtifact(ctx, ref)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}
		nested, err := a.admit(
			ctx,
			ref,
			sessionv3Runtime.ArtifactSelection{
				RootID:           pin.RootID,
				ArtifactID:       pin.ArtifactID,
				ExpectedRevision: pin.Revision,
				ExpectedDigest:   pin.Digest,
			},
			nil,
			active,
			false,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}

		return finalizeWorkflowTarget(document, target, sessionv3Runtime.WorkflowTarget{
			Kind:          sessionv3Runtime.WorkflowTargetWorkflow,
			NestedProgram: &nested.Program,
			Effect: sessionv3Runtime.EffectContract{
				Kind: sessionv3Runtime.EffectKindRead,
			},
		})

	case workflowv1.TargetAgent:
		if a.invocations == nil {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: workflow agent target %q requires a typed workflow invocation adapter",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		pin, err := a.pinInvocationTarget(ctx, targetPlan)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}
		if err := a.invocations.AdmitWorkflowAgentTarget(ctx, pin); err != nil {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: workflow agent target %q: %w",
				ErrBindingUnavailable,
				target.Ref,
				err,
			)
		}
		return finalizeWorkflowTarget(document, target, sessionv3Runtime.WorkflowTarget{
			Kind:     sessionv3Runtime.WorkflowTargetAgent,
			Artifact: &pin,
			Effect: sessionv3Runtime.EffectContract{
				Kind: sessionv3Runtime.EffectKindRead,
			},
		})

	case workflowv1.TargetModel:
		if a.invocations == nil {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: workflow model target %q requires a typed workflow invocation adapter",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		pin, err := a.pinInvocationTarget(ctx, targetPlan)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}
		if err := a.invocations.AdmitWorkflowModelTarget(ctx, pin); err != nil {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: workflow model target %q: %w",
				ErrBindingUnavailable,
				target.Ref,
				err,
			)
		}
		return finalizeWorkflowTarget(document, target, sessionv3Runtime.WorkflowTarget{
			Kind:     sessionv3Runtime.WorkflowTargetModel,
			Artifact: &pin,
			Effect: sessionv3Runtime.EffectContract{
				Kind: sessionv3Runtime.EffectKindRead,
			},
		},
		)

	default:
		return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
			"unsupported Workflow target kind %q",
			target.Kind,
		)
	}
}

func (a *Admission) pinInvocationTarget(
	ctx context.Context,
	target workflowAPI.TargetPlan,
) (sessionv3Runtime.ArtifactPin, error) {
	ref, found := workflowTargetArtifactRef(target)
	if !found {
		return sessionv3Runtime.ArtifactPin{}, fmt.Errorf(
			"%w: workflow target %q",
			ErrBindingUnavailable,
			target.Name,
		)
	}
	return a.pinArtifact(ctx, ref)
}

func workflowTargetArtifactRef(
	value workflowAPI.TargetPlan,
) (artifactModel.ArtifactRef, bool) {
	if !value.UsesComposition ||
		value.Resolution == nil ||
		value.Resolution.Status != composition.ResolutionAvailable ||
		value.Resolution.Target == nil ||
		value.Resolution.Target.Form != composition.TargetFormArtifact ||
		value.Resolution.Target.Artifact == nil {
		return artifactModel.ArtifactRef{}, false
	}
	return *value.Resolution.Target.Artifact, true
}

func finalizeWorkflowTarget(
	document workflowv1.WorkflowDocument,
	source workflowv1.Target,
	target sessionv3Runtime.WorkflowTarget,
) (sessionv3Runtime.WorkflowTarget, error) {
	if source.Interface != nil &&
		len(source.Interface.Capabilities) != 0 {
		return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
			"%w: target capability requirements are unavailable",
			ErrBindingUnavailable,
		)
	}

	contract, err := compileWorkflowCallableInterface(
		document,
		source.Interface,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowTarget{}, err
	}
	target.Contract = contract

	if source.Interface != nil &&
		source.Interface.Effect != nil &&
		(source.Interface.Effect.Kind == string(sessionv3Runtime.EffectKindKeyed) ||
			source.Interface.Effect.Keyed) {
		target.KeyedCapable = true
	}

	if source.Interface != nil && source.Interface.Effect != nil {
		requested, err := compileWorkflowEffect(source.Interface.Effect)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}
		if target.Effect.Kind != "" && target.Effect.Kind != requested.Kind {
			// A declaration can impose a more restrictive retry classification,
			// but cannot assert safety that the current binding does not supply.
			stricter := requested.Kind == sessionv3Runtime.EffectKindNonIdempotent ||
				(target.Effect.Kind == sessionv3Runtime.EffectKindRead &&
					requested.Kind == sessionv3Runtime.EffectKindIdempotent)
			if !stricter {
				return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
					"%w: declared effect %q exceeds binding guarantee %q",
					ErrBindingUnavailable,
					requested.Kind,
					target.Effect.Kind,
				)
			}
		}
		target.Effect = requested
	}

	if target.Effect.Kind == "" {
		var sourceEffect *workflowv1.EffectContract
		if source.Interface != nil {
			sourceEffect = source.Interface.Effect
		}
		target.Effect, err = compileWorkflowEffect(sourceEffect)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}
	}
	if err := target.Effect.Validate(); err != nil {
		return sessionv3Runtime.WorkflowTarget{}, err
	}
	return target, nil
}

func (a *Admission) pinArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (sessionv3Runtime.ArtifactPin, error) {
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}

	values, err := a.artifacts.GetMany(
		ctx,
		[]artifactModel.ArtifactRef{ref},
	)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if len(values) != 1 || values[0].Ref() != ref {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow Artifact pinning returned an unexpected Artifact",
		)
	}

	value := values[0]
	if value.State != artifactModel.StateAvailable ||
		!value.Enabled {
		return sessionv3Runtime.ArtifactPin{}, fmt.Errorf(
			"workflow Artifact %q is not available and enabled",
			ref.ArtifactID,
		)
	}
	if value.ResolvedDefinition == nil {
		return sessionv3Runtime.ArtifactPin{}, fmt.Errorf(
			"workflow Artifact %q has no resolved Definition",
			ref.ArtifactID,
		)
	}

	return sessionv3Runtime.ArtifactPin{
		RootID:     string(value.RootID),
		ArtifactID: string(value.ID),
		Revision:   value.Revision,
		Digest:     string(*value.ResolvedDefinition),
	}, nil
}
