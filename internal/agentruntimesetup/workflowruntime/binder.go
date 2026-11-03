package workflowruntime

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// Binder composes the existing provider and Tool binder with Workflow-local
// deterministic execution plus registered Service and Activity bindings. All
// private binding values stay in BoundWork and are never checkpointed.
type Binder struct {
	Base          sessionv3.WorkBinder
	Bindings      *BindingRegistry
	Deterministic sessionv3.DeterministicExecutor
}

type boundService struct {
	binding ServiceBinding
	private any
}

type boundActivity struct {
	binding ActivityBinding
	private any
}

func NewBinder(base sessionv3.WorkBinder, bindings *BindingRegistry) (*Binder, error) {
	if base == nil || bindings == nil {
		return nil, errors.New("workflow binder dependencies are incomplete")
	}
	return &Binder{Base: base, Bindings: bindings}, nil
}

func (b *Binder) BindWork(ctx context.Context, request sessionv3.WorkDispatchRequest) (sessionv3.BindResult, error) {
	if result, handled, err := b.Deterministic.Bind(ctx, request); handled || err != nil {
		return result, err
	}
	switch request.WorkItem.Type {
	case sessionv3.WorkItemTypeService:
		return b.bindService(ctx, request)
	case sessionv3.WorkItemTypeActivity:
		return b.bindActivity(ctx, request)
	default:
		return b.Base.BindWork(ctx, request)
	}
}

func (b *Binder) ExecuteWork(
	ctx context.Context,
	request sessionv3.WorkDispatchRequest,
	bound sessionv3.BoundWork,
) (sessionv3.WorkExecutionResult, error) {
	if result, handled, err := b.Deterministic.Execute(ctx, request, bound); handled || err != nil {
		return result, err
	}
	switch value := bound.Private.(type) {
	case boundService:
		return value.binding.Execute(ctx, request.WorkItem.Input.Service.Input, value.private)
	case boundActivity:
		return value.binding.Execute(ctx, request.WorkItem.Input.Activity.Input, value.private)
	default:
		return b.Base.ExecuteWork(ctx, request, bound)
	}
}

func (b *Binder) ResolveWait(
	ctx context.Context,
	request sessionv3.WaitResolutionRequest,
) (sessionv3.WaitResolutionResult, error) {
	if result, handled, err := b.Deterministic.ResolveWait(ctx, request); handled || err != nil {
		return result, err
	}
	switch value := request.Private.(type) {
	case boundService:
		return value.binding.ResolveWait(ctx, request.Wait, request.Resolution, request.Response, value.private)
	case boundActivity:
		return value.binding.ResolveWait(ctx, request.Wait, request.Resolution, request.Response, value.private)
	default:
		return b.Base.ResolveWait(ctx, request)
	}
}

func (b *Binder) CloseExecutionBinding(ctx context.Context, executionID sessionv3.ExecutionID) error {
	if closer, ok := b.Base.(sessionv3.ExecutionBindingCloser); ok {
		return closer.CloseExecutionBinding(ctx, executionID)
	}
	return nil
}

func (b *Binder) bindService(ctx context.Context, request sessionv3.WorkDispatchRequest) (sessionv3.BindResult, error) {
	payload := request.WorkItem.Input.Service
	if payload == nil {
		return sessionv3.BindResult{}, errors.New("Service WorkItem payload is missing")
	}
	binding, found := b.Bindings.Service(payload.ServiceID)
	if !found || binding.Version() != payload.BindingVersion {
		return sessionv3.BindResult{}, fmt.Errorf(
			"%w: service %q version %q",
			ErrBindingUnavailable,
			payload.ServiceID,
			payload.BindingVersion,
		)
	}
	semantic, private, err := binding.Bind(ctx, payload.Input)
	if err != nil {
		return sessionv3.BindResult{}, err
	}
	return sessionv3.BindResult{
		SemanticBinding: semantic,
		Private:         boundService{binding: binding, private: private},
	}, nil
}

func (b *Binder) bindActivity(
	ctx context.Context,
	request sessionv3.WorkDispatchRequest,
) (sessionv3.BindResult, error) {
	payload := request.WorkItem.Input.Activity
	if payload == nil {
		return sessionv3.BindResult{}, errors.New("activity WorkItem payload is missing")
	}
	binding, found := b.Bindings.Activity(payload.ActivityID)
	if !found || binding.Version() != payload.BindingVersion {
		return sessionv3.BindResult{}, fmt.Errorf(
			"%w: activity %q version %q",
			ErrBindingUnavailable,
			payload.ActivityID,
			payload.BindingVersion,
		)
	}
	semantic, private, err := binding.Bind(ctx, payload.Input)
	if err != nil {
		return sessionv3.BindResult{}, err
	}
	return sessionv3.BindResult{
		SemanticBinding: semantic,
		Private:         boundActivity{binding: binding, private: private},
	}, nil
}
