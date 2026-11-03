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
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

const (
	InterpreterID      = sessionv3Runtime.WorkflowInterpreterID
	InterpreterVersion = sessionv3Runtime.WorkflowInterpreterVersion
)

var ErrBindingUnavailable = errors.New("workflow target binding is unavailable")

// WorkBinding is an application-owned Service or Activity runtime binding.
// Artifact Workflow declarations only persist its stable ID and version.
type WorkBinding interface {
	ID() string
	Version() string
	Effect() sessionv3Runtime.EffectContract
	Bind(
		ctx context.Context,
		input sessionv3Runtime.CanonicalValue,
	) (sessionv3Runtime.SemanticBinding, any, error)
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
		return nil, errors.New(
			"workflow admission dependencies are incomplete",
		)
	}
	return &Admission{
		workflows: workflows,
		artifacts: artifacts,
		bindings:  bindings,
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
	)
}

func (a *Admission) admit(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	selection sessionv3Runtime.ArtifactSelection,
	input sessionv3Runtime.CanonicalValue,
	active map[artifactModel.ArtifactRef]struct{},
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

	program, err := compileWorkflowProgram(plan.Document, input)
	if err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := a.bindTargets(ctx, plan, &program, active); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := program.Validate(); err != nil {
		return sessionv3Runtime.WorkflowAdmissionResult{}, err
	}
	if err := program.ValidateInput(); err != nil {
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
	program *sessionv3Runtime.WorkflowProgram,
	active map[artifactModel.ArtifactRef]struct{},
) error {
	if program == nil {
		return errors.New("workflow program is unavailable")
	}

	refs := make(map[string]artifactModel.ArtifactRef)
	for _, occurrence := range plan.Capabilities.Occurrences {
		if occurrence.Target == nil ||
			occurrence.Target.Artifact == nil {
			continue
		}
		refs[occurrence.Path] = *occurrence.Target.Artifact
	}

	for _, group := range []struct {
		path   string
		values map[string]workflowv1.Target
	}{
		{
			path:   "targets",
			values: plan.Document.Targets,
		},
		{
			path:   "requirements",
			values: plan.Document.Requirements,
		},
	} {
		for name, target := range group.values {
			if _, duplicate := program.Targets[name]; duplicate {
				return fmt.Errorf(
					"workflow target %q is declared more than once",
					name,
				)
			}

			bound, err := a.bindTarget(
				ctx,
				group.path+"/"+name,
				target,
				refs,
				active,
			)
			if err != nil {
				return err
			}
			program.Targets[name] = bound
		}
	}
	return nil
}

func (a *Admission) bindTarget(
	ctx context.Context,
	path string,
	target workflowv1.Target,
	refs map[string]artifactModel.ArtifactRef,
	active map[artifactModel.ArtifactRef]struct{},
) (sessionv3Runtime.WorkflowTarget, error) {
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
		return sessionv3Runtime.WorkflowTarget{
			Kind:           sessionv3Runtime.WorkflowTargetService,
			ServiceID:      value.ID(),
			BindingVersion: value.Version(),
		}, nil

	case workflowv1.TargetActivity:
		value, found := a.bindings.Activity(target.Ref)
		if !found {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: activity %q",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		return sessionv3Runtime.WorkflowTarget{
			Kind:           sessionv3Runtime.WorkflowTargetActivity,
			ActivityID:     value.ID(),
			BindingVersion: value.Version(),
		}, nil

	case workflowv1.TargetTool:
		ref, found := refs[path]
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
			return sessionv3Runtime.WorkflowTarget{
				Kind: sessionv3Runtime.WorkflowTargetMCPTool,
				MCP: &sessionv3Runtime.WorkflowMCP{
					Server:   pin,
					ToolName: target.Ref,
				},
			}, nil
		}

		return sessionv3Runtime.WorkflowTarget{
			Kind:     sessionv3Runtime.WorkflowTargetTool,
			Artifact: &pin,
			ToolName: target.Ref,
		}, nil

	case workflowv1.TargetWorkflow:
		ref, found := refs[path]
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
			sessionv3Runtime.MustCanonicalValue(map[string]any{}),
			active,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}

		return sessionv3Runtime.WorkflowTarget{
			Kind:          sessionv3Runtime.WorkflowTargetWorkflow,
			NestedProgram: &nested.Program,
		}, nil

	case workflowv1.TargetAgent,
		workflowv1.TargetModel:
		ref, found := refs[path]
		if !found {
			return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
				"%w: Artifact target %q",
				ErrBindingUnavailable,
				target.Ref,
			)
		}
		pin, err := a.pinArtifact(ctx, ref)
		if err != nil {
			return sessionv3Runtime.WorkflowTarget{}, err
		}

		kind := sessionv3Runtime.WorkflowTargetAgent
		if target.Kind == workflowv1.TargetModel {
			kind = sessionv3Runtime.WorkflowTargetModel
		}
		return sessionv3Runtime.WorkflowTarget{
			Kind:     kind,
			Artifact: &pin,
		}, nil

	default:
		return sessionv3Runtime.WorkflowTarget{}, fmt.Errorf(
			"unsupported Workflow target kind %q",
			target.Kind,
		)
	}
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
