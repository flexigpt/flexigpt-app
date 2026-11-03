// Package workflowruntime compiles Artifact-backed Workflow declarations into
// Session ProgramFrames. It does not own a persistence root.
package workflowruntime

import (
	"context"
	"encoding/json"
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
	InterpreterID      = "artifact.workflow.v1"
	InterpreterVersion = uint32(1)
)

var ErrBindingUnavailable = errors.New("workflow target binding is unavailable")

// WorkBinding is the common private runtime binding for Workflow service and
// activity targets. The target kind remains durable Workflow declaration data.
type WorkBinding interface {
	ID() string
	Version() string
	Effect() sessionv3Runtime.EffectContract
	Bind(ctx context.Context, input sessionv3Runtime.CanonicalValue) (sessionv3Runtime.SemanticBinding, any, error)
	Execute(
		ctx context.Context,
		input sessionv3Runtime.CanonicalValue,
		state any,
	) (sessionv3Runtime.WorkExecutionResult, error)
	ResolveWait(
		ctx context.Context,
		wait sessionv3Runtime.Wait,
		resolution sessionv3Runtime.WaitResolution,
		input sessionv3Runtime.CanonicalValue,
		state any,
	) (sessionv3Runtime.WaitResolutionResult, error)
}

type (
	ServiceBinding  = WorkBinding
	ActivityBinding = WorkBinding
)

type BindingRegistry struct {
	mu       sync.RWMutex
	bindings map[string]WorkBinding
}

func NewBindingRegistry() *BindingRegistry {
	return &BindingRegistry{bindings: map[string]WorkBinding{}}
}

func (r *BindingRegistry) RegisterService(v ServiceBinding) error {
	if r == nil || v == nil || v.ID() == "" || v.Version() == "" {
		return errors.New("invalid Workflow service binding")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.registerLocked("service", v)
}

func (r *BindingRegistry) RegisterActivity(v ActivityBinding) error {
	if r == nil || v == nil || v.ID() == "" || v.Version() == "" {
		return errors.New("invalid Workflow activity binding")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.registerLocked("activity", v)
}

func (r *BindingRegistry) Service(id string) (ServiceBinding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.bindings["service:"+id]
	return v, ok
}

func (r *BindingRegistry) Activity(id string) (ActivityBinding, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.bindings["activity:"+id]
	return v, ok
}

func (r *BindingRegistry) registerLocked(
	kind string,
	value WorkBinding,
) error {
	key := kind + ":" + value.ID()
	if _, found := r.bindings[key]; found {
		return fmt.Errorf("duplicate Workflow %s binding %q", kind, value.ID())
	}
	r.bindings[key] = value
	return nil
}

type Admission struct {
	workflows *workflowAPI.Service
	artifacts artifact.API
	bindings  *BindingRegistry
}

func NewAdmission(
	workflows *workflowAPI.Service,
	artifacts artifact.API,
	bindings *BindingRegistry,
) (*Admission, error) {
	if workflows == nil || artifacts == nil || bindings == nil {
		return nil, errors.New("workflow admission dependencies are incomplete")
	}
	return &Admission{
		workflows: workflows,
		artifacts: artifacts,
		bindings:  bindings,
	}, nil
}

type TargetBinding struct {
	Name           string                            `json:"name"`
	Kind           workflowv1.TargetKind             `json:"kind"`
	Artifact       *sessionv3Runtime.ArtifactPin     `json:"artifact,omitempty"`
	ServiceID      string                            `json:"serviceID,omitempty"`
	ActivityID     string                            `json:"activityID,omitempty"`
	BindingVersion string                            `json:"bindingVersion,omitempty"`
	NestedProgram  *sessionv3Runtime.ProgramSnapshot `json:"nestedProgram,omitempty"`
}

type Program struct {
	Version  uint32                          `json:"version"`
	Workflow sessionv3Runtime.ArtifactPin    `json:"workflow"`
	Document workflowv1.WorkflowDocument     `json:"document"`
	Input    sessionv3Runtime.CanonicalValue `json:"input"`
	Targets  map[string]TargetBinding        `json:"targets"`
}

func (a *Admission) AdmitWorkflow(
	ctx context.Context,
	request sessionv3Runtime.WorkflowAdmissionRequest,
) (sessionv3Runtime.WorkflowAdmissionResult, error) {
	if request.Workflow.RootID == "" || request.Workflow.ArtifactID == "" || !request.Input.Valid() {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New("invalid Workflow admission request")
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
	)
}

func (a *Admission) admit(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	selection sessionv3Runtime.ArtifactSelection,
	input sessionv3Runtime.CanonicalValue,
	active map[artifactModel.ArtifactRef]struct{},
) (sessionv3Runtime.WorkflowAdmissionResult, error) {
	if _, exists := active[ref]; exists {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New("nested Workflow cycle")
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
	if selection.ExpectedRevision != 0 && selection.ExpectedRevision != plan.Revision {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New("selected Workflow revision changed")
	}
	if selection.ExpectedDigest != "" && selection.ExpectedDigest != string(plan.Digest) {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New("selected Workflow definition changed")
	}
	pin, err := a.pinArtifact(ctx, plan.Workflow)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if pin.Revision != plan.Revision || pin.Digest != string(plan.Digest) {
		return sessionv3Runtime.WorkflowAdmissionResult{}, errors.New("workflow Artifact changed during admission")
	}
	program := Program{
		Version:  InterpreterVersion,
		Workflow: pin,
		Document: plan.Document,
		Input:    input.Clone(),
		Targets:  map[string]TargetBinding{},
	}
	if err := a.bindTargets(ctx, plan, &program, active); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := validateProgram(program); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
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
	program *Program,
	active map[artifactModel.ArtifactRef]struct{},
) error {
	refs := map[string]artifactModel.ArtifactRef{}
	for _, occurrence := range plan.Capabilities.Occurrences {
		if occurrence.Status == composition.ResolutionAvailable && occurrence.Target != nil &&
			occurrence.Target.Artifact != nil {
			refs[occurrence.Path] = *occurrence.Target.Artifact
		}
	}
	for name, target := range plan.Document.Targets {
		b, err := a.bindTarget(ctx, "targets/"+name, name, target, refs, active)
		if err != nil {
			return err
		}
		program.Targets[name] = b
	}
	for name, target := range plan.Document.Requirements {
		b, err := a.bindTarget(ctx, "requirements/"+name, name, target, refs, active)
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
	active map[artifactModel.ArtifactRef]struct{},
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
		pin, err := a.pinArtifact(ctx, ref)
		if err != nil {
			return TargetBinding{}, err
		}
		out.Artifact = &pin
		if target.Kind == workflowv1.TargetWorkflow {
			nested, err := a.admit(
				ctx,
				ref,
				sessionv3Runtime.ArtifactSelection{
					RootID:           pin.RootID,
					ArtifactID:       pin.ArtifactID,
					ExpectedRevision: pin.Revision,
					ExpectedDigest:   pin.Digest,
				},
				sessionv3Runtime.MustCanonicalValue(map[string]any{}),
				active,
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
	if value.State != artifactModel.StateAvailable || !value.Enabled {
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
	for _, handler := range block.Catch {
		if err := validateBlockTargets(handler.Do, targets); err != nil {
			return err
		}
	}
	if block.Finally != nil {
		if err := validateBlockTargets(*block.Finally, targets); err != nil {
			return err
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
	if step.Machine != nil {
		for _, state := range step.Machine.States {
			if state.Entry != nil {
				out = append(out, *state.Entry)
			}
			if state.Exit != nil {
				out = append(out, *state.Exit)
			}
			for _, transition := range state.Transitions {
				if transition.Do != nil {
					out = append(out, *transition.Do)
				}
			}
		}
	}
	return out
}
