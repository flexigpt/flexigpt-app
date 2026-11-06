package sessionv3

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const deterministicCompensationIntentSystem = "session-v3-compensation-intent"

type deterministicCompensationIntentOutput struct {
	EffectID           EffectRecordID `json:"effectID"`
	OriginalWorkItemID WorkItemID     `json:"originalWorkItemID"`
}

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
	case WorkItemTypeTransform,
		WorkItemTypeCondition,
		WorkItemTypeStateUpdate,
		WorkItemTypeFork,
		WorkItemTypeJoin,
		WorkItemTypeReconciliation,
		WorkItemTypeCompensation,
		WorkItemTypeTargetResolver:
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
	case WorkItemTypeFork:
		return deterministicForkResult(item)

	case WorkItemTypeReconciliation:
		return deterministicReconciliationResult(item)

	case WorkItemTypeCompensation:
		if item.Input.Compensation == nil {
			return WorkExecutionResult{}, true, errors.New(
				"deterministic compensation WorkItem has no payload",
			)
		}
		value := MustCanonicalValue(deterministicCompensationIntentOutput{
			EffectID:           item.Input.Compensation.EffectID,
			OriginalWorkItemID: item.Input.Compensation.OriginalWorkItemID,
		})
		return WorkExecutionResult{
			Output: &WorkItemOutput{
				Value: value,
				Receipt: &ExternalReceipt{
					System:      deterministicCompensationIntentSystem,
					Reference:   string(item.ID),
					ConfirmedAt: time.Now().UTC(),
				},
			},
			Confirmed: true,
		}, true, nil

	case WorkItemTypeJoin:
		return deterministicJoinResult(request.Execution, item)

	case WorkItemTypeTransform:
		if item.Input.Transform == nil {
			return WorkExecutionResult{}, true, errors.New(
				"deterministic transform WorkItem has no transform payload",
			)
		}
		value = item.Input.Transform.Input.Clone()
		if err := validateWorkflowSchema(
			item.Input.Transform.OutputSchema,
			value,
		); err != nil {
			return WorkExecutionResult{}, true, fmt.Errorf(
				"validate transform output: %w",
				err,
			)
		}

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

	case WorkItemTypeStateUpdate:
		if item.Input.StateUpdate == nil {
			return WorkExecutionResult{}, true, errors.New(
				"deterministic StateUpdate WorkItem has no payload",
			)
		}
		next, err := applyWorkflowStateUpdate(
			*item.Input.StateUpdate,
		)
		if err != nil {
			return WorkExecutionResult{}, true, err
		}
		if err := validateWorkflowSchema(
			item.Input.StateUpdate.Schema,
			next,
		); err != nil {
			return WorkExecutionResult{}, true, fmt.Errorf(
				"validate state update output: %w",
				err,
			)
		}
		value = next

	case WorkItemTypeTargetResolver:
		if item.Input.TargetResolver == nil {
			return WorkExecutionResult{}, true, errors.New(
				"deterministic target resolver WorkItem has no payload",
			)
		}
		output := TargetResolverOutput{
			TargetSet: item.Input.TargetResolver.TargetSet,
			Members: append(
				[]string(nil),
				item.Input.TargetResolver.Members...,
			),
		}
		if err := output.Validate(); err != nil {
			return WorkExecutionResult{}, true, err
		}
		value = MustCanonicalValue(output)
		return WorkExecutionResult{
			Output: &WorkItemOutput{
				Value:          value,
				TargetResolver: &output,
				Receipt: &ExternalReceipt{
					System:      "session-v3-target-resolver",
					Reference:   string(item.ID),
					ConfirmedAt: time.Now().UTC(),
				},
			},
			Confirmed: true,
		}, true, nil

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

type forkWorkOutput struct {
	ChildFrames []ProgramFrameID `json:"childFrames"`
}

type joinWorkFrame struct {
	ID     ProgramFrameID    `json:"id"`
	Status ProgramFrameState `json:"status"`
	Output CanonicalValue    `json:"output,omitempty"`
	Error  *SanitizedError   `json:"error,omitempty"`
}

type joinWorkOutput struct {
	Policy string          `json:"policy"`
	Frames []joinWorkFrame `json:"frames"`
}

func deterministicForkResult(
	item WorkItem,
) (WorkExecutionResult, bool, error) {
	if item.Input.Fork == nil {
		return WorkExecutionResult{}, true, errors.New(
			"deterministic fork WorkItem has no payload",
		)
	}
	if len(item.Input.Fork.ChildFrames) == 0 {
		return WorkExecutionResult{}, true, errors.New(
			"deterministic fork WorkItem has no child frames",
		)
	}
	value := MustCanonicalValue(forkWorkOutput{
		ChildFrames: append(
			[]ProgramFrameID(nil),
			item.Input.Fork.ChildFrames...,
		),
	})
	return WorkExecutionResult{
		Output: &WorkItemOutput{
			Value: value,
			Receipt: &ExternalReceipt{
				System:      "session-v3-fork",
				Reference:   string(item.ID),
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, true, nil
}

func deterministicReconciliationResult(
	item WorkItem,
) (WorkExecutionResult, bool, error) {
	if item.Input.Reconciliation == nil {
		return WorkExecutionResult{}, true, errors.New(
			"deterministic reconciliation WorkItem has no payload",
		)
	}
	if err := item.Input.Reconciliation.Validate(); err != nil {
		return WorkExecutionResult{}, true, err
	}

	value := MustCanonicalValue(struct {
		OriginalWorkItemID WorkItemID `json:"originalWorkItemID"`
	}{
		OriginalWorkItemID: item.Input.Reconciliation.OriginalWorkItemID,
	})
	return WorkExecutionResult{
		Output: &WorkItemOutput{
			Value: value,
			Receipt: &ExternalReceipt{
				System:      "session-v3-reconciliation",
				Reference:   string(item.ID),
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, true, nil
}

func deterministicJoinResult(
	execution Execution,
	item WorkItem,
) (WorkExecutionResult, bool, error) {
	if item.Input.Join == nil {
		return WorkExecutionResult{}, true, errors.New(
			"deterministic join WorkItem has no payload",
		)
	}
	if len(item.Input.Join.ChildFrames) == 0 {
		return WorkExecutionResult{}, true, errors.New(
			"deterministic join WorkItem has no child frames",
		)
	}

	frames := make([]joinWorkFrame, 0, len(item.Input.Join.ChildFrames))
	for _, id := range item.Input.Join.ChildFrames {
		frame := mutableFrame(&execution, id)
		if frame == nil {
			return WorkExecutionResult{}, true, fmt.Errorf(
				"join WorkItem child ProgramFrame %q is unavailable",
				id,
			)
		}
		if !frame.Status.Terminal() {
			return WorkExecutionResult{}, true, fmt.Errorf(
				"join WorkItem child ProgramFrame %q is not terminal",
				id,
			)
		}
		value := joinWorkFrame{
			ID:     frame.ID,
			Status: frame.Status,
			Output: frame.Output.Clone(),
		}
		if frame.Error != nil {
			copyValue := *frame.Error
			copyValue.Details = frame.Error.Details.Clone()
			copyValue.Causes = append(
				[]SanitizedError(nil),
				frame.Error.Causes...,
			)
			value.Error = &copyValue
		}
		frames = append(frames, value)
	}
	output := joinWorkOutput{
		Policy: item.Input.Join.Policy,
		Frames: frames,
	}
	value := MustCanonicalValue(output)
	return WorkExecutionResult{
		Output: &WorkItemOutput{
			Value: value,
			Receipt: &ExternalReceipt{
				System:      "session-v3-join",
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
	switch request.Wait.Request.Kind {
	case WaitKindTimer:
		return WaitResolutionResult{}, true, errors.New(
			"timer waits cannot be resolved manually",
		)
	case WaitKindEvent:
		return WaitResolutionResult{}, true, errors.New(
			"event waits cannot be resolved manually",
		)
	case WaitKindChannel:
		return WaitResolutionResult{}, true, errors.New(
			"channel waits cannot be resolved manually",
		)
	case WaitKindChild:
		return WaitResolutionResult{}, true, errors.New(
			"completion waits cannot be resolved manually",
		)
	default:
	}

	value := request.Response.Clone()
	if len(value) == 0 {
		value = MustCanonicalValue(nil)
	}
	if err := validateWorkflowSchema(request.Wait.Request.InputSchema, value); err != nil {
		return WaitResolutionResult{}, true, fmt.Errorf(
			"validate wait response: %w",
			err,
		)
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
