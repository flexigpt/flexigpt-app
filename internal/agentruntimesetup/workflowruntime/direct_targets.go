package workflowruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// DirectTargetPreparationRequest contains only runtime-neutral direct
// capability evidence and declared callable requirements. The owning domain
// adapter resolves current facts and native readiness.
type DirectTargetPreparationRequest struct {
	Target       sessionv3Runtime.DirectCapabilityTarget
	Requirements []sessionv3Runtime.WorkflowCapabilityRequirement
}

// DirectTargetPreparation is the durable result returned by an owning target
// domain. Private clients, credentials, native handles, and callbacks remain
// in the adapter and never enter this value.
type DirectTargetPreparation struct {
	Facts        sessionv3Runtime.CanonicalValue
	Transport    sessionv3Runtime.WorkflowTargetTransport
	Capabilities []sessionv3Runtime.WorkflowCapabilityFact
}

// DirectTargetAdapter owns preparation and execution of one direct capability
// provider. Workflow only routes typed values to this boundary.
type DirectTargetAdapter interface {
	ID() string
	SupportsWorkflowDirectTarget(
		target sessionv3Runtime.DirectCapabilityTarget,
	) bool
	PrepareWorkflowDirectTarget(
		ctx context.Context,
		request DirectTargetPreparationRequest,
	) (DirectTargetPreparation, error)
	BindWorkflowDirectWork(
		ctx context.Context,
		request sessionv3Runtime.WorkDispatchRequest,
		plan sessionv3Runtime.DirectCapabilityPlan,
	) (sessionv3Runtime.BindResult, error)
	ExecuteWorkflowDirectWork(
		ctx context.Context,
		request sessionv3Runtime.WorkDispatchRequest,
		plan sessionv3Runtime.DirectCapabilityPlan,
		private any,
	) (sessionv3Runtime.WorkExecutionResult, error)
	ResolveWorkflowDirectWait(
		ctx context.Context,
		request sessionv3Runtime.WaitResolutionRequest,
		plan sessionv3Runtime.DirectCapabilityPlan,
		private any,
	) (sessionv3Runtime.WaitResolutionResult, error)
}

type directBoundWork struct {
	adapterID string
	plan      sessionv3Runtime.DirectCapabilityPlan
	private   any
}

type directBoundWait struct {
	adapterID string
	plan      sessionv3Runtime.DirectCapabilityPlan
	private   any
}

// DirectTargetRegistry is setup-owned. It contains typed owning-domain
// adapters and never fabricates Artifact identities or generic request text.
type DirectTargetRegistry struct {
	mu       sync.RWMutex
	adapters map[string]DirectTargetAdapter
	order    []string
}

func NewDirectTargetRegistry() *DirectTargetRegistry {
	return &DirectTargetRegistry{
		adapters: map[string]DirectTargetAdapter{},
		order:    []string{},
	}
}

func (r *DirectTargetRegistry) Register(
	adapter DirectTargetAdapter,
) error {
	if r == nil || adapter == nil {
		return errors.New("direct target adapter is required")
	}
	if adapter.ID() == "" {
		return errors.New("direct target adapter ID is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.adapters[adapter.ID()]; found {
		return fmt.Errorf(
			"direct target adapter %q is already registered",
			adapter.ID(),
		)
	}
	r.adapters[adapter.ID()] = adapter
	r.order = append(r.order, adapter.ID())
	sort.Strings(r.order)
	return nil
}

func (r *DirectTargetRegistry) PrepareWorkflowDirectTarget(
	ctx context.Context,
	request DirectTargetPreparationRequest,
) (sessionv3Runtime.DirectCapabilityPlan, error) {
	if r == nil {
		return sessionv3Runtime.DirectCapabilityPlan{}, ErrBindingUnavailable
	}
	if err := request.Target.Validate(); err != nil {
		return sessionv3Runtime.DirectCapabilityPlan{}, err
	}

	adapter, err := r.adapterForTarget(request.Target)
	if err != nil {
		return sessionv3Runtime.DirectCapabilityPlan{}, err
	}
	if adapter == nil {
		return sessionv3Runtime.DirectCapabilityPlan{}, fmt.Errorf(
			"%w: direct %s capability provider %q",
			ErrBindingUnavailable,
			request.Target.Kind,
			request.Target.ProviderIdentity,
		)
	}

	prepared, err := adapter.PrepareWorkflowDirectTarget(ctx, request)
	if err != nil {
		return sessionv3Runtime.DirectCapabilityPlan{}, err
	}
	if err := prepared.Transport.Validate(); err != nil {
		return sessionv3Runtime.DirectCapabilityPlan{}, err
	}
	if err := sessionv3Runtime.ValidateWorkflowCapabilityFacts(
		request.Requirements,
		prepared.Capabilities,
	); err != nil {
		return sessionv3Runtime.DirectCapabilityPlan{}, err
	}

	plan := sessionv3Runtime.DirectCapabilityPlan{
		AdapterID:    adapter.ID(),
		Target:       request.Target,
		Facts:        prepared.Facts.Clone(),
		Transport:    prepared.Transport,
		Capabilities: cloneDirectCapabilityFacts(prepared.Capabilities),
	}
	if err := plan.Validate(); err != nil {
		return sessionv3Runtime.DirectCapabilityPlan{}, err
	}
	return plan, nil
}

// BindWorkflowDirectWork satisfies the narrow setup binder interface used by
// Session Runtime v3. A direct WorkItem has no generic fallback route.
func (r *DirectTargetRegistry) BindWorkflowDirectWork(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) (sessionv3Runtime.BindResult, bool, error) {
	plan := sessionv3Runtime.DirectCapabilityPlanForWorkItem(
		request.WorkItem,
	)
	if plan == nil {
		return sessionv3Runtime.BindResult{}, false, nil
	}

	adapter := r.adapterByID(plan.AdapterID)
	if adapter == nil {
		return sessionv3Runtime.BindResult{}, true, fmt.Errorf(
			"%w: direct target adapter %q",
			ErrBindingUnavailable,
			plan.AdapterID,
		)
	}

	result, err := adapter.BindWorkflowDirectWork(ctx, request, *plan)
	if err != nil {
		return sessionv3Runtime.BindResult{}, true, err
	}
	if err := result.SemanticBinding.Validate(); err != nil {
		return sessionv3Runtime.BindResult{}, true, err
	}

	if result.SemanticBinding.Direct != nil {
		return sessionv3Runtime.BindResult{}, true, errors.New(
			"direct target adapter returned conflicting direct target facts",
		)
	}
	result.SemanticBinding.Direct = plan
	result.SemanticBinding.Capabilities = cloneDirectCapabilityFacts(
		plan.Capabilities,
	)

	switch {
	case result.Wait != nil:
		if result.Private == nil {
			return sessionv3Runtime.BindResult{}, true, errors.New(
				"direct target wait has no private resolver state",
			)
		}
		result.Private = directBoundWait{
			adapterID: adapter.ID(),
			plan:      *plan,
			private:   result.Private,
		}
	case result.ImmediateOutput != nil:
		result.Private = nil
	default:
		if result.Private == nil {
			return sessionv3Runtime.BindResult{}, true, errors.New(
				"direct target binding has no private execution state",
			)
		}
		result.Private = directBoundWork{
			adapterID: adapter.ID(),
			plan:      *plan,
			private:   result.Private,
		}
	}
	return result, true, nil
}

func (r *DirectTargetRegistry) ExecuteWorkflowDirectWork(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
	bound sessionv3Runtime.BoundWork,
) (sessionv3Runtime.WorkExecutionResult, bool, error) {
	value, found := bound.Private.(directBoundWork)
	if !found {
		return sessionv3Runtime.WorkExecutionResult{}, false, nil
	}
	adapter := r.adapterByID(value.adapterID)
	if adapter == nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, fmt.Errorf(
			"%w: direct target adapter %q",
			ErrBindingUnavailable,
			value.adapterID,
		)
	}
	result, err := adapter.ExecuteWorkflowDirectWork(
		ctx,
		request,
		value.plan,
		value.private,
	)
	return result, true, err
}

func (r *DirectTargetRegistry) ResolveWorkflowDirectWait(
	ctx context.Context,
	request sessionv3Runtime.WaitResolutionRequest,
) (sessionv3Runtime.WaitResolutionResult, bool, error) {
	value, found := request.Private.(directBoundWait)
	if !found {
		return sessionv3Runtime.WaitResolutionResult{}, false, nil
	}
	adapter := r.adapterByID(value.adapterID)
	if adapter == nil {
		return sessionv3Runtime.WaitResolutionResult{}, true, fmt.Errorf(
			"%w: direct target adapter %q",
			ErrBindingUnavailable,
			value.adapterID,
		)
	}

	result, err := adapter.ResolveWorkflowDirectWait(
		ctx,
		request,
		value.plan,
		value.private,
	)
	if err != nil {
		return sessionv3Runtime.WaitResolutionResult{}, true, err
	}
	if !result.Resume {
		return result, true, nil
	}
	if result.Bound == nil || result.Bound.Private == nil {
		return sessionv3Runtime.WaitResolutionResult{}, true, errors.New(
			"direct target wait resume has no bound execution state",
		)
	}
	result.Bound.Private = directBoundWork{
		adapterID: value.adapterID,
		plan:      value.plan,
		private:   result.Bound.Private,
	}
	return result, true, nil
}

func (r *DirectTargetRegistry) adapterForTarget(
	target sessionv3Runtime.DirectCapabilityTarget,
) (DirectTargetAdapter, error) {
	if r == nil {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	var selected DirectTargetAdapter
	for _, id := range r.order {
		adapter := r.adapters[id]
		if adapter.SupportsWorkflowDirectTarget(target) {
			if selected != nil {
				return nil, fmt.Errorf(
					"%w: direct capability provider %q has multiple owning adapters",
					ErrBindingUnavailable,
					target.ProviderIdentity,
				)
			}
			selected = adapter
		}
	}
	return selected, nil
}

func (r *DirectTargetRegistry) adapterByID(
	id string,
) DirectTargetAdapter {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	value := r.adapters[id]
	r.mu.RUnlock()
	return value
}

func cloneDirectCapabilityFacts(
	values []sessionv3Runtime.WorkflowCapabilityFact,
) []sessionv3Runtime.WorkflowCapabilityFact {
	if len(values) == 0 {
		return nil
	}
	output := make(
		[]sessionv3Runtime.WorkflowCapabilityFact,
		len(values),
	)
	for index := range values {
		output[index] = values[index]
		output[index].Details = values[index].Details.Clone()
	}
	return output
}
