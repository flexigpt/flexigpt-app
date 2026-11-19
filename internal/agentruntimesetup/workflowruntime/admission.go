// Package workflowruntime compiles planning-owned Workflow Artifact closures
// into runtime-neutral Session v3 WorkflowProgram snapshots. It does not
// execute Workflow behavior.
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
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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
	Input   jsonutil.CanonicalValue
	Context jsonutil.CanonicalValue
	Ports   []sessionv3Runtime.PortInput
	Stream  sessionv3Runtime.WorkPortStream
}

// WorkBindingWaitRequest is the typed domain boundary for a Service or
// Activity wait resolution. It contains no Runtime private handles.
type WorkBindingWaitRequest struct {
	Wait       sessionv3Runtime.Wait
	Resolution sessionv3Runtime.WaitResolution
	Response   jsonutil.CanonicalValue
	Input      jsonutil.CanonicalValue
	Context    jsonutil.CanonicalValue
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
	Context          bool
	InputPorts       bool
	FinalOutputPorts bool
	OutputStream     bool
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

// PreparationRequest is the setup-owned request for one input-bound Workflow
// ProgramSnapshot. Session Runtime never receives this selection request.
type PreparationRequest struct {
	Workflow           sessionv3Runtime.ArtifactSelection
	Input              jsonutil.CanonicalValue
	InvocationDefaults *sessionv3Runtime.SessionDefaults
}

// TriggerPreparationRequest is the setup-owned request for one input-unbound
// trigger ProgramSnapshot.
type TriggerPreparationRequest struct {
	Workflow           sessionv3Runtime.ArtifactSelection
	InvocationDefaults *sessionv3Runtime.SessionDefaults
}

// PrepareWorkflow resolves the static plan, acquires current pins, validates
// invocation input, prepares current target bindings, and lowers one immutable
// ProgramSnapshot. It does not mutate Session state.
func (a *Admission) PrepareWorkflow(
	ctx context.Context,
	request PreparationRequest,
) (sessionv3Runtime.ProgramSnapshot, error) {
	if request.Workflow.RootID == "" ||
		request.Workflow.ArtifactID == "" ||
		!request.Input.Valid() {
		return sessionv3Runtime.ProgramSnapshot{}, errors.New(
			"invalid Workflow preparation request",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(request.Workflow.RootID),
		ArtifactID: artifactModel.ArtifactID(request.Workflow.ArtifactID),
	}
	result, err := a.admitRoot(
		ctx,
		ref,
		request.Workflow,
		request.Input,
		true,
		request.InvocationDefaults,
	)
	if err != nil {
		return sessionv3Runtime.ProgramSnapshot{}, err
	}
	return result.Program, nil
}

// PrepareWorkflowTriggers prepares an input-unbound ProgramSnapshot for
// explicit Runtime trigger registration.
func (a *Admission) PrepareWorkflowTriggers(
	ctx context.Context,
	request TriggerPreparationRequest,
) (sessionv3Runtime.ProgramSnapshot, error) {
	return a.prepareWorkflowTriggers(
		ctx,
		request.Workflow,
		request.InvocationDefaults,
	)
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
	program, err := a.PrepareWorkflow(
		ctx,
		PreparationRequest{
			Workflow: request.Workflow,
			Input:    request.Input,
		},
	)
	return sessionv3Runtime.WorkflowAdmissionResult{Program: program}, err
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

// prepareWorkflowTriggers compiles an input-unbound Workflow program for
// Session-owned trigger registration. Trigger occurrence input is bound only
// when a durable occurrence is accepted.
func (a *Admission) prepareWorkflowTriggers(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
	defaults *sessionv3Runtime.SessionDefaults,
) (sessionv3Runtime.ProgramSnapshot, error) {
	if selection.RootID == "" || selection.ArtifactID == "" {
		return sessionv3Runtime.ProgramSnapshot{}, errors.New(
			"invalid workflow trigger registration request",
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(selection.RootID),
		ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
	}
	result, err := a.admitRoot(
		ctx,
		ref,
		selection,
		nil,
		false,
		defaults,
	)
	if err != nil {
		return sessionv3Runtime.ProgramSnapshot{}, err
	}

	program, err := jsonutil.DecodeCanonicalValue[sessionv3Runtime.WorkflowProgram](result.Program.Payload)
	if err != nil {
		return sessionv3Runtime.ProgramSnapshot{}, err
	}
	if len(program.Triggers) == 0 {
		return sessionv3Runtime.ProgramSnapshot{}, errors.New(
			"workflow declares no triggers",
		)
	}
	return result.Program, nil
}

type workflowAdmissionArtifact struct {
	revision         uint64
	definitionDigest cryptoutil.Digest
}

// workflowAdmissionPlan is one setup-owned read projection of a
// planning-owned Workflow closure node. Its Workflow facts are already
// admitted. Setup must not parse or validate source again.
type workflowAdmissionPlan struct {
	artifact         artifactModel.ArtifactRef
	revision         uint64
	definitionDigest cryptoutil.Digest
	facts            workflowAPI.WorkflowPlanNode
}

// workflowAdmissionClosure is an immutable request-local lookup index over a
// trusted Workflow plan closure. Setup reads it but does not revalidate source
// grammar, recompute composition, or resolve nested Workflow plans.
type workflowAdmissionClosure struct {
	artifacts map[artifactModel.ArtifactRef]workflowAdmissionArtifact
	workflows map[artifactModel.ArtifactRef]workflowAdmissionPlan
}

// workflowAdmissionClosureFromPlan indexes planning-owned closure facts.
//
// "workflow.ResolvePlan" already validates source, consistency, closure
// completeness, ordering, nested Workflow facts, and unsupported direct
// composition rejection. Setup keeps only the structural checks required to avoid silently
// selecting a missing or conflicting closure node.
func workflowAdmissionClosureFromPlan(
	plan workflowAPI.Plan,
) (*workflowAdmissionClosure, workflowAdmissionPlan, error) {
	closure := &workflowAdmissionClosure{
		artifacts: make(
			map[artifactModel.ArtifactRef]workflowAdmissionArtifact,
			len(plan.Closure.Nodes),
		),
		workflows: make(
			map[artifactModel.ArtifactRef]workflowAdmissionPlan,
		),
	}
	var (
		root      workflowAdmissionPlan
		rootFound bool
	)

	for index := range plan.Closure.Nodes {
		node := plan.Closure.Nodes[index]
		if _, duplicate := closure.artifacts[node.Artifact]; duplicate {
			return nil, workflowAdmissionPlan{}, fmt.Errorf(
				"workflow Artifact closure repeats Artifact %q",
				node.Artifact.ArtifactID,
			)
		}
		closure.artifacts[node.Artifact] = workflowAdmissionArtifact{
			revision:         node.Revision,
			definitionDigest: node.DefinitionDigest,
		}
		if node.Workflow == nil {
			continue
		}

		facts := *node.Workflow
		current := workflowAdmissionPlan{
			artifact:         node.Artifact,
			revision:         node.Revision,
			definitionDigest: node.DefinitionDigest,
			facts:            facts,
		}
		closure.workflows[node.Artifact] = current
		if node.ID == plan.Closure.Root {
			root = current
			rootFound = true
		}
	}

	if !rootFound {
		return nil, workflowAdmissionPlan{}, errors.New(
			"workflow artifact closure root has no typed Workflow facts",
		)
	}
	if plan.Workflow != root.artifact ||
		plan.Revision != root.revision ||
		plan.DefinitionDigest != root.definitionDigest {
		return nil, workflowAdmissionPlan{}, errors.New(
			"workflow plan root differs from its static closure root",
		)
	}

	return closure, root, nil
}

func (c *workflowAdmissionClosure) artifact(
	ref artifactModel.ArtifactRef,
) (workflowAdmissionArtifact, bool) {
	if c == nil {
		return workflowAdmissionArtifact{}, false
	}
	value, found := c.artifacts[ref]
	return value, found
}

func (c *workflowAdmissionClosure) workflow(
	ref artifactModel.ArtifactRef,
) (workflowAdmissionPlan, bool) {
	if c == nil {
		return workflowAdmissionPlan{}, false
	}
	value, found := c.workflows[ref]
	return value, found
}

// admitRoot resolves one root Workflow plan, then compiles it from its
// planning-owned closure. Nested Workflows use typed closure facts and never
// call ResolvePlan recursively.
func (a *Admission) admitRoot(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	selection sessionv3Runtime.ArtifactSelection,
	input jsonutil.CanonicalValue,
	inputBound bool,
	defaults *sessionv3Runtime.SessionDefaults,
) (sessionv3Runtime.WorkflowAdmissionResult, error) {
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
	closure, root, err := workflowAdmissionClosureFromPlan(plan)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}

	return a.admitPlanned(
		ctx,
		root,
		selection,
		input,
		closure,
		inputBound,
		defaults,
		nil,
	)
}

func (a *Admission) admitPlanned(
	ctx context.Context,
	plan workflowAdmissionPlan,
	selection sessionv3Runtime.ArtifactSelection,
	input jsonutil.CanonicalValue,
	closure *workflowAdmissionClosure,
	inputBound bool,
	defaults *sessionv3Runtime.SessionDefaults,
	inheritedOwnerModel *sessionv3Runtime.ArtifactPin,
) (sessionv3Runtime.WorkflowAdmissionResult, error) {
	if selection.RootID != string(plan.artifact.RootID) ||
		selection.ArtifactID != string(plan.artifact.ArtifactID) {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"selected Workflow differs from the static Artifact closure",
		)
	}
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != plan.revision {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"selected Workflow revision changed",
		)
	}
	if selection.ExpectedDigest != "" &&
		selection.ExpectedDigest != string(plan.definitionDigest) {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New(
			"selected Workflow definition changed",
		)
	}

	pin, err := a.pinClosureArtifact(ctx, closure, plan.artifact)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}

	var program sessionv3Runtime.WorkflowProgram
	if inputBound {
		program, err = compileWorkflowProgram(
			plan.facts.Document,
			plan.facts.Inventory,
			input,
		)
	} else {
		program, err = compileWorkflowProgramTemplate(
			plan.facts.Document,
			plan.facts.Inventory,
		)
	}
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	ownerModel := inheritedOwnerModel
	if ownerModel != nil {
		copyValue := *ownerModel
		ownerModel = &copyValue
	} else {
		ownerModel, err = a.workflowOwnerModel(ctx, defaults)
	}
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	requirements, err := a.prepareRequirements(
		ctx,
		plan,
		closure,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	program.Requirements = requirements
	if err := a.bindCallableTargets(
		ctx,
		plan,
		&program,
		closure,
		ownerModel,
	); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := prepareWorkflowLLMInvocations(
		plan.facts.Inventory,
		&program,
	); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := validatePreparedTargetTransport(
		plan.facts.Inventory,
		&program,
	); err != nil {
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
			Payload:            jsonutil.MustCanonicalValue(program),
		},
	}, nil
}

func (a *Admission) bindCallableTargets(
	ctx context.Context,
	plan workflowAdmissionPlan,
	program *sessionv3Runtime.WorkflowProgram,
	closure *workflowAdmissionClosure,
	ownerModel *sessionv3Runtime.ArtifactPin,
) error {
	if program == nil {
		return errors.New("workflow program is unavailable")
	}

	for _, targetPlan := range plan.facts.Inventory.CallableTargets {
		if _, duplicate := program.Targets[targetPlan.Name]; duplicate {
			return fmt.Errorf(
				"workflow target %q is declared more than once",
				targetPlan.Name,
			)
		}

		bound, err := a.bindTarget(
			ctx,
			plan.facts.Document,
			targetPlan,
			closure,
			ownerModel,
		)
		if err != nil {
			return err
		}
		bound, err = a.prepareBoundTarget(
			ctx,
			targetPlan,
			bound,
		)
		if err != nil {
			return err
		}
		bound.DeclarationPath = targetPlan.Path
		program.Targets[targetPlan.Name] = bound
	}
	return nil
}

// prepareRequirements retains root requirements separately from callable
// targets. A requirement can establish readiness facts but can never be
// selected by Workflow calls or target sets.
func (a *Admission) prepareRequirements(
	ctx context.Context,
	plan workflowAdmissionPlan,
	closure *workflowAdmissionClosure,
) ([]sessionv3Runtime.WorkflowRequirementFact, error) {
	output := make([]sessionv3Runtime.WorkflowRequirementFact, 0, len(plan.facts.Inventory.Requirements))
	for _, requirement := range plan.facts.Inventory.Requirements {
		fact, err := a.prepareRequirement(
			ctx,
			requirement,
			closure,
		)
		if err != nil {
			return nil, fmt.Errorf("workflow requirement %q: %w", requirement.Name, err)
		}
		output = append(output, fact)
	}
	return output, nil
}

func (a *Admission) bindTarget(
	ctx context.Context,
	document workflowv1.WorkflowDocument,
	targetPlan workflowAPI.TargetPlan,
	closure *workflowAdmissionClosure,
	ownerModel *sessionv3Runtime.ArtifactPin,
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
		ref, err := requireWorkflowTargetArtifactRef(
			targetPlan,
			"Tool target",
		)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}
		pin, err := a.pinClosureArtifact(ctx, closure, ref)
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
			planner, found := a.tools.(ProviderManagedWorkflowToolPlanner)
			if !found {
				return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
					"%w: provider-managed tool target %q has no typed Model planner",
					ErrBindingUnavailable,
					target.Ref,
				)
			}
			if ownerModel == nil {
				return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
					"%w: provider-managed tool target %q requires a prepared Model selection",
					ErrBindingUnavailable,
					target.Ref,
				)
			}
			choice, prepareErr := planner.PrepareProviderManagedWorkflowTool(ctx, pin)
			if prepareErr != nil {
				return sessionv3Runtime.WorkflowTarget{}, prepareErr
			}
			return finalizeWorkflowTarget(document, target, sessionv3Runtime.WorkflowTarget{
				Kind: sessionv3Runtime.WorkflowTargetProviderTool,
				ProviderTool: &sessionv3Runtime.ProviderManagedToolTarget{
					OwnerModel: *ownerModel,
					Choice:     choice,
				},
				Effect: sessionv3Runtime.EffectContract{
					Kind: sessionv3Runtime.EffectKindRead,
				},
			})
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
		ref, err := requireWorkflowTargetArtifactRef(
			targetPlan,
			"nested Workflow target",
		)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}
		nestedPlan, found := closure.workflow(ref)
		if !found {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: nested Workflow %q is absent from the static Artifact closure",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		nested, err := a.admitPlanned(
			ctx,
			nestedPlan,
			workflowSelectionForAdmissionPlan(nestedPlan),
			nil,
			closure,
			false,
			nil,
			ownerModel,
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
		pin, err := a.pinInvocationTarget(ctx, targetPlan, closure)
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
			DeclarationPath: targetPlan.Path,
		})

	case workflowv1.TargetModel:
		if a.invocations == nil {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: workflow model target %q requires a typed workflow invocation adapter",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		pin, err := a.pinInvocationTarget(ctx, targetPlan, closure)
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
			DeclarationPath: targetPlan.Path,
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
	closure *workflowAdmissionClosure,
) (sessionv3Runtime.ArtifactPin, error) {
	ref, err := requireWorkflowTargetArtifactRef(
		target,
		"Workflow invocation target",
	)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	return a.pinClosureArtifact(ctx, closure, ref)
}

func workflowSelectionForAdmissionPlan(
	plan workflowAdmissionPlan,
) sessionv3Runtime.ArtifactSelection {
	return sessionv3Runtime.ArtifactSelection{
		RootID:           string(plan.artifact.RootID),
		ArtifactID:       string(plan.artifact.ArtifactID),
		ExpectedRevision: plan.revision,
		ExpectedDigest:   string(plan.definitionDigest),
	}
}

func requireWorkflowTargetArtifactRef(
	value workflowAPI.TargetPlan,
	targetLabel string,
) (artifactModel.ArtifactRef, error) {
	if ref, found := workflowTargetArtifactRef(value); found {
		return ref, nil
	}

	if value.Resolution != nil &&
		value.Resolution.Status == composition.ResolutionAvailable {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: %s %q has a non-Artifact composition result; Workflow direct capabilities are unsupported",
			ErrBindingUnavailable,
			targetLabel,
			value.Name,
		)
	}

	return artifactModel.ArtifactRef{}, fmt.Errorf(
		"%w: %s %q has no resolved source-backed Artifact",
		ErrBindingUnavailable,
		targetLabel,
		value.Name,
	)
}

func workflowTargetArtifactRef(
	value workflowAPI.TargetPlan,
) (artifactModel.ArtifactRef, bool) {
	if !value.UsesComposition ||
		value.Resolution == nil ||
		value.Resolution.Status != composition.ResolutionAvailable ||
		value.Resolution.ResolvedArtifactRef == nil {
		return artifactModel.ArtifactRef{}, false
	}
	return *value.Resolution.ResolvedArtifactRef, true
}

func finalizeWorkflowTarget(
	document workflowv1.WorkflowDocument,
	source workflowv1.Target,
	target sessionv3Runtime.WorkflowTarget,
) (sessionv3Runtime.WorkflowTarget, error) {
	contract, err := compileWorkflowCallableInterface(
		document,
		source.Interface,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowTarget{}, err
	}
	target.Contract = contract
	target.DeclarationPath = ""

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

func (a *Admission) pinClosureArtifact(
	ctx context.Context,
	closure *workflowAdmissionClosure,
	ref artifactModel.ArtifactRef,
) (sessionv3Runtime.ArtifactPin, error) {
	expected, found := closure.artifact(ref)
	if !found {
		return sessionv3Runtime.ArtifactPin{}, fmt.Errorf(
			"%w: Artifact %q is absent from the static Workflow closure",
			ErrBindingUnavailable,
			ref.ArtifactID,
		)
	}

	pin, err := a.pinArtifact(ctx, ref)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if pin.Revision != expected.revision ||
		pin.Digest != string(expected.definitionDigest) {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"workflow artifact closure changed since planning",
		)
	}
	return pin, nil
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
