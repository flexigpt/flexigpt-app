// Package workflowruntime compiles Artifact-backed Workflow declarations into
// Session ProgramFrames. It does not own a persistence root.
package workflowruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

const (
	InterpreterID      = "artifact.workflow.v1"
	InterpreterVersion = uint32(1)
)

var ErrBindingUnavailable = errors.New("workflow target binding is unavailable")

type ArtifactPinResolver interface {
	PinArtifact(ctx context.Context, ref artifactModel.ArtifactRef) (sessionv3.ArtifactPin, error)
}

type ServiceBinding interface {
	ID() string
	Version() string
	Effect() sessionv3.EffectContract
	Bind(ctx context.Context, input sessionv3.CanonicalValue) (sessionv3.SemanticBinding, any, error)
	Execute(ctx context.Context, input sessionv3.CanonicalValue, state any) (sessionv3.WorkExecutionResult, error)
	ResolveWait(
		ctx context.Context,
		wait sessionv3.Wait,
		resolution sessionv3.WaitResolution,
		input sessionv3.CanonicalValue,
		state any,
	) (sessionv3.WaitResolutionResult, error)
}

type ActivityBinding interface {
	ID() string
	Version() string
	Effect() sessionv3.EffectContract
	Bind(ctx context.Context, input sessionv3.CanonicalValue) (sessionv3.SemanticBinding, any, error)
	Execute(ctx context.Context, input sessionv3.CanonicalValue, state any) (sessionv3.WorkExecutionResult, error)
	ResolveWait(
		ctx context.Context,
		wait sessionv3.Wait,
		resolution sessionv3.WaitResolution,
		input sessionv3.CanonicalValue,
		state any,
	) (sessionv3.WaitResolutionResult, error)
}

type BindingRegistry struct {
	mu         sync.RWMutex
	services   map[string]ServiceBinding
	activities map[string]ActivityBinding
}

func NewBindingRegistry() *BindingRegistry {
	return &BindingRegistry{services: map[string]ServiceBinding{}, activities: map[string]ActivityBinding{}}
}

func (r *BindingRegistry) RegisterService(v ServiceBinding) error {
	if r == nil || v == nil || v.ID() == "" || v.Version() == "" {
		return errors.New("invalid Workflow service binding")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.services[v.ID()]; ok {
		return fmt.Errorf("duplicate Workflow service binding %q", v.ID())
	}
	r.services[v.ID()] = v
	return nil
}

func (r *BindingRegistry) RegisterActivity(v ActivityBinding) error {
	if r == nil || v == nil || v.ID() == "" || v.Version() == "" {
		return errors.New("invalid Workflow activity binding")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.activities[v.ID()]; ok {
		return fmt.Errorf("duplicate Workflow activity binding %q", v.ID())
	}
	r.activities[v.ID()] = v
	return nil
}

func (r *BindingRegistry) Service(id string) (ServiceBinding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.services[id]
	return v, ok
}

func (r *BindingRegistry) Activity(id string) (ActivityBinding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.activities[id]
	return v, ok
}

type Admission struct {
	workflows *workflowAPI.Service
	pins      ArtifactPinResolver
	bindings  *BindingRegistry
	mu        sync.Mutex
	active    map[artifactModel.ArtifactRef]struct{}
}

func NewAdmission(
	workflows *workflowAPI.Service,
	pins ArtifactPinResolver,
	bindings *BindingRegistry,
) (*Admission, error) {
	if workflows == nil || pins == nil || bindings == nil {
		return nil, errors.New("workflow admission dependencies are incomplete")
	}
	return &Admission{
		workflows: workflows,
		pins:      pins,
		bindings:  bindings,
		active:    map[artifactModel.ArtifactRef]struct{}{},
	}, nil
}

type TargetBinding struct {
	Name           string                     `json:"name"`
	Kind           workflowv1.TargetKind      `json:"kind"`
	Artifact       *sessionv3.ArtifactPin     `json:"artifact,omitempty"`
	ServiceID      string                     `json:"serviceID,omitempty"`
	ActivityID     string                     `json:"activityID,omitempty"`
	BindingVersion string                     `json:"bindingVersion,omitempty"`
	NestedProgram  *sessionv3.ProgramSnapshot `json:"nestedProgram,omitempty"`
}

type Program struct {
	Version  uint32                      `json:"version"`
	Workflow sessionv3.ArtifactPin       `json:"workflow"`
	Document workflowv1.WorkflowDocument `json:"document"`
	Input    sessionv3.CanonicalValue    `json:"input"`
	Targets  map[string]TargetBinding    `json:"targets"`
}

func (a *Admission) AdmitWorkflow(
	ctx context.Context,
	request sessionv3.WorkflowAdmissionRequest,
) (sessionv3.WorkflowAdmissionResult, error) {
	if request.Workflow.RootID == "" || request.Workflow.ArtifactID == "" || !request.Input.Valid() {
		return sessionv3.WorkflowAdmissionResult{}, errors.New("invalid Workflow admission request")
	}
	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(request.Workflow.RootID),
		ArtifactID: artifactModel.ArtifactID(request.Workflow.ArtifactID),
	}
	return a.admit(ctx, ref, request.Workflow, request.Input)
}

func (a *Admission) admit(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	selection sessionv3.ArtifactSelection,
	input sessionv3.CanonicalValue,
) (sessionv3.WorkflowAdmissionResult, error) {
	a.mu.Lock()
	if _, exists := a.active[ref]; exists {
		a.mu.Unlock()
		return sessionv3.WorkflowAdmissionResult{}, errors.New("nested Workflow cycle")
	}
	a.active[ref] = struct{}{}
	a.mu.Unlock()
	defer func() { a.mu.Lock(); delete(a.active, ref); a.mu.Unlock() }()
	plan, err := a.workflows.ResolvePlan(ctx, ref)
	if err != nil {
		return sessionv3.WorkflowAdmissionResult{}, err
	}
	if !plan.CompositionComplete {
		return sessionv3.WorkflowAdmissionResult{}, fmt.Errorf(
			"%w: unresolved required Workflow capability",
			ErrBindingUnavailable,
		)
	}
	if selection.ExpectedRevision != 0 && selection.ExpectedRevision != plan.Revision {
		return sessionv3.WorkflowAdmissionResult{}, errors.New("selected Workflow revision changed")
	}
	if selection.ExpectedDigest != "" && selection.ExpectedDigest != string(plan.Digest) {
		return sessionv3.WorkflowAdmissionResult{}, errors.New("selected Workflow definition changed")
	}
	pin, err := a.pins.PinArtifact(ctx, plan.Workflow)
	if err != nil {
		return sessionv3.WorkflowAdmissionResult{}, err
	}
	if pin.Revision != plan.Revision || pin.Digest != string(plan.Digest) {
		return sessionv3.WorkflowAdmissionResult{}, errors.New("workflow Artifact changed during admission")
	}
	program := Program{
		Version:  InterpreterVersion,
		Workflow: pin,
		Document: plan.Document,
		Input:    input.Clone(),
		Targets:  map[string]TargetBinding{},
	}
	if err := a.bindTargets(ctx, plan, &program); err != nil {
		return sessionv3.WorkflowAdmissionResult{}, err
	}
	if err := validateProgram(program); err != nil {
		return sessionv3.WorkflowAdmissionResult{}, err
	}
	return sessionv3.WorkflowAdmissionResult{
		Program: sessionv3.ProgramSnapshot{
			InterpreterID:      InterpreterID,
			InterpreterVersion: InterpreterVersion,
			Artifact:           &pin,
			DefinitionDigest:   pin.Digest,
			Payload:            sessionv3.MustCanonicalValue(program),
		},
	}, nil
}

func (a *Admission) bindTargets(ctx context.Context, plan workflowAPI.Plan, program *Program) error {
	refs := map[string]artifactModel.ArtifactRef{}
	for _, occurrence := range plan.Capabilities.Occurrences {
		if occurrence.Status == composition.ResolutionAvailable && occurrence.Target != nil &&
			occurrence.Target.Artifact != nil {
			refs[occurrence.Path] = *occurrence.Target.Artifact
		}
	}
	for name, target := range plan.Document.Targets {
		b, err := a.bindTarget(ctx, "targets/"+name, name, target, refs)
		if err != nil {
			return err
		}
		program.Targets[name] = b
	}
	for name, target := range plan.Document.Requirements {
		b, err := a.bindTarget(ctx, "requirements/"+name, name, target, refs)
		if err != nil {
			return err
		}
		program.Targets[name] = b
	}
	return validateBlockTargets(plan.Document.Block, program.Targets)
}

func (a *Admission) bindTarget(
	ctx context.Context,
	path, name string,
	target workflowv1.Target,
	refs map[string]artifactModel.ArtifactRef,
) (TargetBinding, error) {
	out := TargetBinding{Name: name, Kind: target.Kind}
	switch target.Kind {
	case workflowv1.TargetService:
		v, ok := a.bindings.Service(target.Ref)
		if !ok {
			return TargetBinding{}, fmt.Errorf("%w: service %q", ErrBindingUnavailable, target.Ref)
		}
		out.ServiceID = v.ID()
		out.BindingVersion = v.Version()
	case workflowv1.TargetActivity:
		v, ok := a.bindings.Activity(target.Ref)
		if !ok {
			return TargetBinding{}, fmt.Errorf("%w: activity %q", ErrBindingUnavailable, target.Ref)
		}
		out.ActivityID = v.ID()
		out.BindingVersion = v.Version()
	case workflowv1.TargetAgent, workflowv1.TargetModel, workflowv1.TargetTool, workflowv1.TargetWorkflow:
		ref, ok := refs[path]
		if !ok {
			return TargetBinding{}, fmt.Errorf("%w: Artifact target %q", ErrBindingUnavailable, name)
		}
		pin, err := a.pins.PinArtifact(ctx, ref)
		if err != nil {
			return TargetBinding{}, err
		}
		out.Artifact = &pin
		if target.Kind == workflowv1.TargetWorkflow {
			nested, err := a.admit(
				ctx,
				ref,
				sessionv3.ArtifactSelection{
					RootID:           pin.RootID,
					ArtifactID:       pin.ArtifactID,
					ExpectedRevision: pin.Revision,
					ExpectedDigest:   pin.Digest,
				},
				sessionv3.MustCanonicalValue(map[string]any{}),
			)
			if err != nil {
				return TargetBinding{}, err
			}
			out.NestedProgram = &nested.Program
		}
	default:
		return TargetBinding{}, fmt.Errorf("unsupported Workflow target kind %q", target.Kind)
	}
	return out, nil
}

func validateProgram(p Program) error {
	if p.Version != InterpreterVersion || p.Workflow.RootID == "" || p.Workflow.ArtifactID == "" ||
		p.Workflow.Revision == 0 ||
		p.Workflow.Digest == "" ||
		!p.Input.Valid() {
		return errors.New("invalid Workflow Program payload")
	}
	if err := p.Document.Validate(); err != nil {
		return err
	}
	return validateBlockTargets(p.Document.Block, p.Targets)
}

func validateBlockTargets(block workflowv1.Block, targets map[string]TargetBinding) error {
	for _, step := range block.Steps {
		if step.Call != nil {
			var name string
			if json.Unmarshal(step.Call.Target, &name) == nil && name != "" {
				if _, ok := targets[name]; !ok {
					return fmt.Errorf("workflow call target %q is not bound", name)
				}
			}
		}
		for _, child := range childBlocks(step) {
			if err := validateBlockTargets(child, targets); err != nil {
				return err
			}
		}
	}
	return nil
}

func childBlocks(step workflowv1.Step) []workflowv1.Block {
	out := []workflowv1.Block{}
	if step.Scope != nil {
		out = append(out, *step.Scope)
	}
	if step.Choose != nil {
		for _, c := range step.Choose.Cases {
			out = append(out, c.Do)
		}
		if step.Choose.Default != nil {
			out = append(out, *step.Choose.Default)
		}
	}
	if step.Parallel != nil {
		for _, b := range step.Parallel.Branches {
			out = append(out, b.Do)
		}
	}
	if step.Foreach != nil {
		out = append(out, step.Foreach.Do)
	}
	if step.Repeat != nil {
		out = append(out, step.Repeat.Do)
	}
	if step.Saga != nil {
		out = append(out, step.Saga.Do)
	}
	if step.Effect != nil {
		if step.Effect.Reconcile != nil {
			out = append(out, *step.Effect.Reconcile)
		}
		if step.Effect.Compensate != nil {
			out = append(out, *step.Effect.Compensate)
		}
	}
	return out
}
