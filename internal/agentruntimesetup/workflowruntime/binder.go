package workflowruntime

import (
	"context"
	"errors"
	"fmt"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// NewInterpreter returns the runtime-owned interpreter. Workflow Artifact
// admission remains in this package, but execution semantics live in
// agentruntime-go/sessionv3 and do not import Artifact contracts.
func NewInterpreter() (*sessionv3Runtime.WorkflowInterpreter, error) {
	return sessionv3Runtime.NewWorkflowInterpreter()
}

// Binder composes the existing provider and Tool binder with Workflow-local
// deterministic execution plus registered Service and Activity bindings. All
// private binding values stay in BoundWork and are never checkpointed.
type Binder struct {
	Base          sessionv3Runtime.WorkBinder
	Bindings      *BindingRegistry
	Pins          ArtifactPinVerifier
	Deterministic sessionv3Runtime.DeterministicExecutor
}

// ArtifactPinVerifier validates a durable Workflow Artifact pin against
// current Artifact state. It is implemented by setup, never by Session core.
type ArtifactPinVerifier interface {
	VerifyArtifactPin(
		ctx context.Context,
		pin sessionv3Runtime.ArtifactPin,
	) error
}

type boundExternal struct {
	binding WorkBinding
	private any
}

func NewBinder(
	base sessionv3Runtime.WorkBinder,
	bindings *BindingRegistry,
	pins ...ArtifactPinVerifier,
) (*Binder, error) {
	if base == nil || bindings == nil {
		return nil, errors.New("workflow binder dependencies are incomplete")
	}
	if len(pins) > 1 {
		return nil, errors.New("workflow binder has more than one Artifact pin verifier")
	}
	var verifier ArtifactPinVerifier
	if len(pins) == 1 {
		verifier = pins[0]
	}
	return &Binder{Base: base, Bindings: bindings, Pins: verifier}, nil
}

func (b *Binder) BindWork(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
	if result, handled, err := b.Deterministic.Bind(ctx, request); handled || err != nil {
		return result, err
	}
	if err := b.verifyWorkflowToolPins(ctx, request.WorkItem); err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	switch request.WorkItem.Type {
	case sessionv3Runtime.WorkItemTypeService:
		return b.bindService(ctx, request)
	case sessionv3Runtime.WorkItemTypeActivity:
		return b.bindActivity(ctx, request)
	default:
		return b.Base.BindWork(ctx, request)
	}
}

func (b *Binder) ExecuteWork(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
	bound sessionv3Runtime.BoundWork,
) (sessionv3Runtime.WorkExecutionResult, error) {
	if result, handled, err := b.Deterministic.Execute(ctx, request, bound); handled || err != nil {
		return result, err
	}

	switch value := bound.Private.(type) {
	case boundExternal:
		input, err := externalWorkInput(request.WorkItem)
		if err != nil {
			return sessionv3Runtime.WorkExecutionResult{}, err
		}
		return value.binding.Execute(ctx, input, value.private)
	default:
		return b.Base.ExecuteWork(ctx, request, bound)
	}
}

func (b *Binder) ResolveWait(
	ctx context.Context,
	request sessionv3Runtime.WaitResolutionRequest,
) (sessionv3Runtime.WaitResolutionResult, error) {
	if result, handled, err := b.Deterministic.ResolveWait(ctx, request); handled || err != nil {
		return result, err
	}

	switch value := request.Private.(type) {
	case boundExternal:
		return value.binding.ResolveWait(ctx, request.Wait, request.Resolution, request.Response, value.private)
	default:
		return b.Base.ResolveWait(ctx, request)
	}
}

func (b *Binder) CloseExecutionBinding(ctx context.Context, executionID sessionv3Runtime.ExecutionID) error {
	if closer, ok := b.Base.(sessionv3Runtime.ExecutionBindingCloser); ok {
		return closer.CloseExecutionBinding(ctx, executionID)
	}
	return nil
}

func (b *Binder) ValidatePortPlans(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) error {
	if len(request.WorkItem.Ports) == 0 {
		return nil
	}
	validator, found := b.Base.(sessionv3Runtime.PortPlanValidator)
	if !found {
		return errors.New(
			"port adapter is unavailable for the selected workflow target",
		)
	}
	return validator.ValidatePortPlans(ctx, request)
}

func (b *Binder) verifyWorkflowToolPins(
	ctx context.Context,
	item sessionv3Runtime.WorkItem,
) error {
	if item.Type != sessionv3Runtime.WorkItemTypeTool || item.Input.Tool == nil {
		return nil
	}
	payload := item.Input.Tool
	if payload.Target == nil && payload.MCP == nil {
		return nil
	}
	if b == nil || b.Pins == nil {
		return fmt.Errorf("%w: Workflow Artifact pin verifier", ErrBindingUnavailable)
	}
	if payload.Target != nil {
		if err := b.Pins.VerifyArtifactPin(ctx, *payload.Target); err != nil {
			return err
		}
	}
	if payload.MCP != nil {
		if err := b.Pins.VerifyArtifactPin(ctx, payload.MCP.Server); err != nil {
			return err
		}
	}
	return nil
}

func (b *Binder) bindService(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
	payload := request.WorkItem.Input.Service
	if payload == nil {
		return sessionv3Runtime.BindResult{}, errors.New("Service WorkItem payload is missing")
	}
	binding, found := b.Bindings.Service(payload.ServiceID)
	if !found || binding.Version() != payload.BindingVersion {
		return sessionv3Runtime.BindResult{}, fmt.Errorf(
			"%w: service %q version %q",
			ErrBindingUnavailable,
			payload.ServiceID,
			payload.BindingVersion,
		)
	}
	return bindExternal(ctx, binding, payload.BindingVersion, payload.Input)
}

func (b *Binder) bindActivity(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
	payload := request.WorkItem.Input.Activity
	if payload == nil {
		return sessionv3Runtime.BindResult{}, errors.New("activity WorkItem payload is missing")
	}
	binding, found := b.Bindings.Activity(payload.ActivityID)
	if !found || binding.Version() != payload.BindingVersion {
		return sessionv3Runtime.BindResult{}, fmt.Errorf(
			"%w: activity %q version %q",
			ErrBindingUnavailable,
			payload.ActivityID,
			payload.BindingVersion,
		)
	}
	return bindExternal(ctx, binding, payload.BindingVersion, payload.Input)
}

func bindExternal(
	ctx context.Context,
	binding WorkBinding,
	version string,
	input sessionv3Runtime.CanonicalValue,
) (sessionv3Runtime.BindResult, error) {
	if binding == nil || binding.Version() != version {
		return sessionv3Runtime.BindResult{}, fmt.Errorf(
			"%w: external binding version %q",
			ErrBindingUnavailable,
			version,
		)
	}

	semantic, private, err := binding.Bind(ctx, input)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	return sessionv3Runtime.BindResult{
		SemanticBinding: semantic,
		Private:         boundExternal{binding: binding, private: private},
	}, nil
}

func externalWorkInput(
	item sessionv3Runtime.WorkItem,
) (sessionv3Runtime.CanonicalValue, error) {
	switch item.Type {
	case sessionv3Runtime.WorkItemTypeService:
		if item.Input.Service == nil {
			return nil, errors.New("Service WorkItem payload is missing")
		}
		return item.Input.Service.Input.Clone(), nil

	case sessionv3Runtime.WorkItemTypeActivity:
		if item.Input.Activity == nil {
			return nil, errors.New("activity WorkItem payload is missing")
		}
		return item.Input.Activity.Input.Clone(), nil

	default:
		return nil, fmt.Errorf(
			"unsupported external WorkItem type %q",
			item.Type,
		)
	}
}
