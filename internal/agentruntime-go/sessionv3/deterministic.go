package sessionv3

import (
	"context"
	"errors"
	"time"
)

// DeterministicExecutor handles only WorkItem kinds whose behavior is fully
// defined by durable Session state. It has no Artifact, provider, native-path,
// credential, callback, or transport dependency.
//
// Service, Activity, Tool, LLM, Emit, Fork, Join, StateUpdate, and
// Compensation work must be handled by an explicit application-side binder.
// Returning handled=false for these kinds prevents a false successful result.
type DeterministicExecutor struct{}

type deterministicBinding struct{}

func (DeterministicExecutor) Bind(
	_ context.Context,
	request WorkDispatchRequest,
) (BindResult, bool, error) {
	item := request.WorkItem
	binding := SemanticBinding{
		Version:           "session-v3-deterministic/v1",
		TargetFingerprint: string(item.Type) + ":" + item.ProgramPath,
		BoundAt:           time.Now().UTC(),
	}

	switch item.Type {
	case WorkItemTypeTransform, WorkItemTypeCondition:
		return BindResult{
			SemanticBinding: binding,
			Private:         deterministicBinding{},
		}, true, nil

	case WorkItemTypeWait:
		if item.Input.Wait == nil {
			return BindResult{}, true, errors.New(
				"deterministic wait WorkItem has no wait payload",
			)
		}

		wait := item.Input.Wait.Request
		if err := wait.Validate(); err != nil {
			return BindResult{}, true, err
		}

		return BindResult{
			SemanticBinding: binding,
			Wait:            &wait,
			Private:         deterministicBinding{},
		}, true, nil

	default:
		return BindResult{}, false, nil
	}
}

func (DeterministicExecutor) Execute(
	_ context.Context,
	request WorkDispatchRequest,
	work BoundWork,
) (WorkExecutionResult, bool, error) {
	if _, found := work.Private.(deterministicBinding); !found {
		return WorkExecutionResult{}, false, nil
	}

	item := request.WorkItem
	var value CanonicalValue

	switch item.Type {
	case WorkItemTypeTransform:
		if item.Input.Transform == nil {
			return WorkExecutionResult{}, true, errors.New(
				"deterministic transform WorkItem has no transform payload",
			)
		}
		value = item.Input.Transform.Input.Clone()

	case WorkItemTypeCondition:
		if item.Input.Condition == nil {
			return WorkExecutionResult{}, true, errors.New(
				"deterministic condition WorkItem has no condition payload",
			)
		}

		normalized, err := normalizeDeterministicCondition(
			item.Input.Condition.Input,
		)
		if err != nil {
			return WorkExecutionResult{}, true, err
		}
		value = normalized

	default:
		return WorkExecutionResult{}, false, nil
	}

	return WorkExecutionResult{
		Output: &WorkItemOutput{
			Value: value,
			Receipt: &ExternalReceipt{
				System:      "session-v3-deterministic",
				Reference:   string(item.ID),
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, true, nil
}

func (DeterministicExecutor) ResolveWait(
	_ context.Context,
	request WaitResolutionRequest,
) (WaitResolutionResult, bool, error) {
	if request.WorkItem.Type != WorkItemTypeWait {
		return WaitResolutionResult{}, false, nil
	}

	value := request.Response.Clone()
	if len(value) == 0 {
		value = MustCanonicalValue(nil)
	}

	return WaitResolutionResult{
		Output: &WorkItemOutput{
			Value: value,
			Receipt: &ExternalReceipt{
				System:      "session-v3-deterministic",
				Reference:   string(request.Wait.ID),
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, true, nil
}

func normalizeDeterministicCondition(
	value CanonicalValue,
) (CanonicalValue, error) {
	direct, err := DecodeCanonicalValue[bool](value)
	if err == nil {
		return MustCanonicalValue(direct), nil
	}

	type wrappedCondition struct {
		Value *bool `json:"value"`
	}
	wrapped, err := DecodeCanonicalValue[wrappedCondition](value)
	if err == nil &&
		wrapped.Value != nil {
		return MustCanonicalValue(*wrapped.Value), nil
	}

	return nil, errors.New(
		"deterministic condition WorkItem must contain a boolean value",
	)
}
