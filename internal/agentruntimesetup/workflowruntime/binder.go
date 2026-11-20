package workflowruntime

import (
	"context"
	"errors"
	"fmt"
	"slices"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
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
	Base     sessionv3Runtime.WorkBinder
	Bindings *BindingRegistry
}

type boundExternal struct {
	binding WorkBinding
	private any
}

func NewBinder(
	base sessionv3Runtime.WorkBinder,
	bindings *BindingRegistry,
) (*Binder, error) {
	if base == nil || bindings == nil {
		return nil, errors.New("workflow binder dependencies are incomplete")
	}
	return &Binder{Base: base, Bindings: bindings}, nil
}

func (b *Binder) BindWork(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, error) {
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
	switch value := bound.Private.(type) {
	case boundExternal:
		domainRequest, err := externalWorkRequest(
			request,
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
	switch value := request.Private.(type) {
	case boundExternal:
		domainRequest, err := externalWaitRequest(request)
		if err != nil {
			return sessionv3Runtime.WaitResolutionResult{}, err
		}
		result, err := value.binding.ResolveWait(
			ctx,
			domainRequest,
			value.private,
		)
		if err != nil {
			return sessionv3Runtime.WaitResolutionResult{}, err
		}
		return wrapExternalWaitResolution(
			request.WorkItem,
			value,
			result,
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

func (b *Binder) PrepareAppTool(
	ctx context.Context,
	request sessionv3Runtime.AppToolPreparationRequest,
) (sessionv3Runtime.ToolCall, error) {
	executor, found := b.Base.(sessionv3Runtime.AppExecutor)
	if !found || executor == nil {
		return sessionv3Runtime.ToolCall{}, errors.New(
			"session v3 App runtime is unavailable",
		)
	}
	return executor.PrepareAppTool(ctx, request)
}

func (b *Binder) ReadAppResource(
	ctx context.Context,
	request sessionv3Runtime.AppResourceReadRequest,
) (sessionv3Runtime.AppResourceReadResult, error) {
	executor, found := b.Base.(sessionv3Runtime.AppExecutor)
	if !found || executor == nil {
		return sessionv3Runtime.AppResourceReadResult{}, errors.New(
			"session v3 App runtime is unavailable",
		)
	}
	return executor.ReadAppResource(ctx, request)
}

func (b *Binder) ValidateAppContext(
	ctx context.Context,
	request sessionv3Runtime.AppContextStagingRequest,
) (sessionv3Runtime.AppContextUpdate, error) {
	executor, found := b.Base.(sessionv3Runtime.AppExecutor)
	if !found || executor == nil {
		return sessionv3Runtime.AppContextUpdate{}, errors.New(
			"session v3 App runtime is unavailable",
		)
	}
	return executor.ValidateAppContext(ctx, request)
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
		return b.validateExternalPortPlans(request)
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
		request,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	if err := requireExternalPortCapabilities(binding, request.WorkItem, domainRequest); err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	rb, err := binding.Bind(ctx, domainRequest)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}
	semantic, err := externalSemanticBinding(
		request.WorkItem,
		rb.SemanticBinding,
	)
	if err != nil {
		return sessionv3Runtime.BindResult{}, err
	}

	if rb.Wait != nil && rb.ImmediateOutput != nil {
		return sessionv3Runtime.BindResult{}, errors.New(
			"external binding returned both a durable wait and immediate output",
		)
	}
	result := sessionv3Runtime.BindResult{
		SemanticBinding:      semantic,
		Wait:                 rb.Wait,
		ImmediateOutput:      rb.ImmediateOutput,
		ImmediateEffectState: rb.ImmediateEffectState,
	}
	switch {
	case rb.Wait != nil:
		result.Private = boundExternal{
			binding: binding,
			private: rb.Private,
		}
	case rb.ImmediateOutput != nil:
		// Immediate domain output is durable at the Runtime boundary. It has
		// no subsequent domain wait or private execution state.
		result.Private = nil
	default:
		result.Private = boundExternal{
			binding: binding,
			private: rb.Private,
		}
	}
	return result, nil
}

func wrapExternalWaitResolution(
	item sessionv3Runtime.WorkItem,
	current boundExternal,
	result sessionv3Runtime.WaitResolutionResult,
) (sessionv3Runtime.WaitResolutionResult, error) {
	if !result.Resume {
		return result, nil
	}
	if result.Bound == nil {
		return sessionv3Runtime.WaitResolutionResult{}, errors.New(
			"external wait resolution returned no bound work",
		)
	}

	semantic, err := externalSemanticBinding(
		item,
		result.Bound.SemanticBinding,
	)
	if err != nil {
		return sessionv3Runtime.WaitResolutionResult{}, err
	}
	result.Bound = &sessionv3Runtime.BoundWork{
		SemanticBinding: semantic,
		Private: boundExternal{
			binding: current.binding,
			private: result.Bound.Private,
		},
	}
	return result, nil
}

func externalWorkRequest(
	request sessionv3Runtime.WorkDispatchRequest,
) (WorkBindingRequest, error) {
	item := request.WorkItem
	input, err := externalWorkInput(item)
	if err != nil {
		return WorkBindingRequest{}, err
	}
	contextValue, err := externalWorkContext(item)
	if err != nil {
		return WorkBindingRequest{}, err
	}
	return WorkBindingRequest{
		ExecutionID:    request.Execution.ID,
		WorkItemID:     item.ID,
		AttemptID:      externalAttemptID(request.Execution, item.ID),
		Effect:         item.Effect,
		IdempotencyKey: item.Effect.IdempotencyKey,
		Input:          input,
		Context:        contextValue,
		Ports:          cloneExternalPortInputs(item.PortInputs),
		Stream:         request.PortStream,
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
		Wait:           request.Wait,
		Resolution:     request.Resolution,
		Response:       request.Response.Clone(),
		ExecutionID:    request.Execution.ID,
		WorkItemID:     request.WorkItem.ID,
		AttemptID:      externalAttemptID(request.Execution, request.WorkItem.ID),
		Effect:         request.WorkItem.Effect,
		IdempotencyKey: request.WorkItem.Effect.IdempotencyKey,
		Input:          input,
		Context:        contextValue,
		Ports:          cloneExternalPortInputs(request.WorkItem.PortInputs),
	}, nil
}

func externalAttemptID(
	execution sessionv3Runtime.Execution,
	workItemID sessionv3Runtime.WorkItemID,
) sessionv3Runtime.AttemptID {
	for _, attempt := range slices.Backward(execution.Attempts) {
		if attempt.WorkItemID == workItemID &&
			attempt.State == sessionv3Runtime.AttemptStateDispatched {
			return attempt.ID
		}
	}
	return ""
}

func externalWorkContext(
	item sessionv3Runtime.WorkItem,
) (jsonutil.CanonicalValue, error) {
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
) (jsonutil.CanonicalValue, error) {
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

func requireExternalPortCapabilities(
	binding WorkBinding,
	item sessionv3Runtime.WorkItem,
	request WorkBindingRequest,
) error {
	needsContext := len(request.Context) != 0
	needsInputPorts := len(request.Ports) != 0
	needsOutputStream := false
	for _, plan := range item.Ports {
		if plan.Direction != sessionv3Runtime.WorkflowPortDirectionOutput {
			continue
		}
		if plan.Stream {
			needsOutputStream = true
		}
	}
	if !needsContext && !needsInputPorts && !needsOutputStream {
		return nil
	}

	capability, found := binding.(WorkBindingPortCapability)
	if !found {
		return errors.New(
			"external binding does not support workflow port transport",
		)
	}
	support := capability.WorkflowPortCapabilities()
	switch {
	case needsContext && !support.Context:
		return errors.New(
			"external binding does not support workflow context",
		)
	case needsInputPorts && !support.InputPorts:
		return errors.New(
			"external binding does not support workflow input ports",
		)
	case needsOutputStream && !support.OutputStream:
		return errors.New(
			"external binding does not support workflow output streams",
		)
	default:
		return nil
	}
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
			[]jsonutil.CanonicalValue,
			len(values[index].Values),
		)
		for valueIndex := range values[index].Values {
			output[index].Values[valueIndex] = values[index].Values[valueIndex].Clone()
		}
	}
	return output
}

func (b *Binder) validateExternalPortPlans(
	request sessionv3Runtime.WorkDispatchRequest,
) error {
	switch request.WorkItem.Type {
	case sessionv3Runtime.WorkItemTypeService:
		if request.WorkItem.Input.Service == nil {
			return errors.New("service WorkItem payload is missing")
		}
		binding, found := b.Bindings.Service(
			request.WorkItem.Input.Service.ServiceID,
		)
		if !found {
			return ErrBindingUnavailable
		}
		domainRequest, err := externalWorkRequest(
			request,
		)
		if err != nil {
			return err
		}
		return requireExternalPortCapabilities(
			binding,
			request.WorkItem,
			domainRequest,
		)

	case sessionv3Runtime.WorkItemTypeActivity:
		if request.WorkItem.Input.Activity == nil {
			return errors.New("activity WorkItem payload is missing")
		}
		binding, found := b.Bindings.Activity(
			request.WorkItem.Input.Activity.ActivityID,
		)
		if !found {
			return ErrBindingUnavailable
		}
		domainRequest, err := externalWorkRequest(
			request,
		)
		if err != nil {
			return err
		}
		return requireExternalPortCapabilities(binding, request.WorkItem, domainRequest)
	default:
		return errors.New("external port validation has another work item type")
	}
}
