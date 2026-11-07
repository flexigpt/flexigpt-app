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
		domainRequest, err := externalWorkRequest(
			request.WorkItem,
			request.PortStream,
		)
		if err != nil {
			return sessionv3Runtime.WorkExecutionResult{}, err
		}
		return value.binding.Execute(ctx, domainRequest, value.private)
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
		domainRequest, err := externalWaitRequest(request)
		if err != nil {
			return sessionv3Runtime.WaitResolutionResult{}, err
		}
		return value.binding.ResolveWait(
			ctx,
			domainRequest,
			value.private,
		)
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
	switch request.WorkItem.Type {
	case sessionv3Runtime.WorkItemTypeService,
		sessionv3Runtime.WorkItemTypeActivity:
		return nil
	default:
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
	return bindExternal(ctx, binding, payload.BindingVersion, request)
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
	return bindExternal(ctx, binding, payload.BindingVersion, request)
}

func bindExternal(
	ctx context.Context,
	binding WorkBinding,
	version string,
	request sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
	if binding == nil || binding.Version() != version {
		return sessionv3Runtime.BindResult{}, fmt.Errorf(
			"%w: external binding version %q",
			ErrBindingUnavailable,
			version,
		)
	}

	domainRequest, err := externalWorkRequest(
		request.WorkItem,
		request.PortStream,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	semantic, private, err := binding.Bind(ctx, domainRequest)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	semantic, err = externalSemanticBinding(
		request.WorkItem,
		semantic,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	return sessionv3Runtime.BindResult{
		SemanticBinding: semantic,
		Private:         boundExternal{binding: binding, private: private},
	}, nil
}

func externalWorkRequest(
	item sessionv3Runtime.WorkItem,
	stream sessionv3Runtime.WorkPortStream,
) (WorkBindingRequest, error) {
	input, err := externalWorkInput(item)
	if err != nil {
		return WorkBindingRequest{}, err
	}
	contextValue, err := externalWorkContext(item)
	if err != nil {
		return WorkBindingRequest{}, err
	}
	return WorkBindingRequest{
		Input:   input,
		Context: contextValue,
		Ports:   cloneExternalPortInputs(item.PortInputs),
		Stream:  stream,
	}, nil
}

func externalWaitRequest(
	request sessionv3Runtime.WaitResolutionRequest,
) (WorkBindingWaitRequest, error) {
	input, err := externalWorkInput(request.WorkItem)
	if err != nil {
		return WorkBindingWaitRequest{}, err
	}
	contextValue, err := externalWorkContext(request.WorkItem)
	if err != nil {
		return WorkBindingWaitRequest{}, err
	}
	return WorkBindingWaitRequest{
		Wait:       request.Wait,
		Resolution: request.Resolution,
		Response:   request.Response.Clone(),
		Input:      input,
		Context:    contextValue,
		Ports:      cloneExternalPortInputs(request.WorkItem.PortInputs),
	}, nil
}

func externalWorkContext(
	item sessionv3Runtime.WorkItem,
) (sessionv3Runtime.CanonicalValue, error) {
	switch item.Type {
	case sessionv3Runtime.WorkItemTypeService:
		if item.Input.Service == nil {
			return nil, errors.New("service workItem payload is missing")
		}
		return item.Input.Service.Context.Clone(), nil

	case sessionv3Runtime.WorkItemTypeActivity:
		if item.Input.Activity == nil {
			return nil, errors.New("activity workItem payload is missing")
		}
		return item.Input.Activity.Context.Clone(), nil

	default:
		return nil, fmt.Errorf(
			"unsupported external workItem type %q",
			item.Type,
		)
	}
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

func externalSemanticBinding(
	item sessionv3Runtime.WorkItem,
	binding sessionv3Runtime.SemanticBinding,
) (sessionv3Runtime.SemanticBinding, error) {
	if binding.Service != nil || binding.Activity != nil {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"external binding returned an unexpected dispatch plan",
		)
	}

	switch item.Type {
	case sessionv3Runtime.WorkItemTypeService:
		if item.Input.Service == nil {
			return sessionv3Runtime.SemanticBinding{}, errors.New(
				"service work item payload is missing",
			)
		}
		binding.Service = &sessionv3Runtime.ServiceDispatchPlan{
			ID:             item.Input.Service.ServiceID,
			BindingVersion: item.Input.Service.BindingVersion,
			Input:          item.Input.Service.Input.Clone(),
			Context:        item.Input.Service.Context.Clone(),
			Ports:          cloneExternalPortPlans(item.Ports),
			PortInputs:     cloneExternalPortInputs(item.PortInputs),
			Effect:         item.Effect,
		}

	case sessionv3Runtime.WorkItemTypeActivity:
		if item.Input.Activity == nil {
			return sessionv3Runtime.SemanticBinding{}, errors.New(
				"activity work item payload is missing",
			)
		}
		binding.Activity = &sessionv3Runtime.ActivityDispatchPlan{
			ID:             item.Input.Activity.ActivityID,
			BindingVersion: item.Input.Activity.BindingVersion,
			Input:          item.Input.Activity.Input.Clone(),
			Context:        item.Input.Activity.Context.Clone(),
			Ports:          cloneExternalPortPlans(item.Ports),
			PortInputs:     cloneExternalPortInputs(item.PortInputs),
			Effect:         item.Effect,
		}

	default:
		return sessionv3Runtime.SemanticBinding{}, fmt.Errorf(
			"unsupported external work item type %q",
			item.Type,
		)
	}

	if err := binding.Validate(); err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}
	return binding, nil
}

func cloneExternalPortPlans(
	values []sessionv3Runtime.PortPlan,
) []sessionv3Runtime.PortPlan {
	if len(values) == 0 {
		return nil
	}
	output := make([]sessionv3Runtime.PortPlan, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Schema = values[index].Schema.Clone()
		output[index].ChannelSchema = values[index].ChannelSchema.Clone()
	}
	return output
}

func cloneExternalPortInputs(
	values []sessionv3Runtime.PortInput,
) []sessionv3Runtime.PortInput {
	if len(values) == 0 {
		return nil
	}
	output := make([]sessionv3Runtime.PortInput, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Values = make(
			[]sessionv3Runtime.CanonicalValue,
			len(values[index].Values),
		)
		for valueIndex := range values[index].Values {
			output[index].Values[valueIndex] = values[index].Values[valueIndex].Clone()
		}
	}
	return output
}
