package sessionv3

import (
	"errors"
	"fmt"
)

func (s EffectState) Validate() error {
	switch s {
	case "",
		EffectStateNotApplicable,
		EffectStateNotStarted,
		EffectStateDispatched,
		EffectStateConfirmedApplied,
		EffectStateConfirmedNotApplied,
		EffectStateUncertain,
		EffectStateCompensating,
		EffectStateCompensated,
		EffectStateCompensationFailed:
		return nil
	default:
		return fmt.Errorf("unsupported effect state %q", s)
	}
}

func effectStateForDispatch(effect EffectContract) EffectState {
	if effect.Kind == EffectKindRead {
		return EffectStateNotApplicable
	}
	return EffectStateDispatched
}

func effectStateForSettlement(
	item WorkItem,
	result WorkExecutionResult,
) EffectState {
	if item.Effect.Kind == EffectKindRead {
		return EffectStateNotApplicable
	}

	switch item.State {
	case WorkItemStateCompleted:
		return EffectStateConfirmedApplied

	case WorkItemStateCancelled:
		if result.Confirmed {
			return EffectStateConfirmedNotApplied
		}
		return EffectStateUncertain

	case WorkItemStateBlocked:
		return EffectStateConfirmedNotApplied

	case WorkItemStateFailed:
		if result.Confirmed {
			if result.Output != nil &&
				result.Output.Receipt != nil {
				return EffectStateConfirmedApplied
			}
			return EffectStateConfirmedNotApplied
		}
		return EffectStateUncertain

	case WorkItemStateUncertain:
		return EffectStateUncertain

	default:
		return EffectStateUncertain
	}
}

func copyEffectError(
	value *SanitizedError,
	effect EffectState,
) *SanitizedError {
	if value == nil {
		return nil
	}

	output := *value
	output.Effect = effect
	output.Details = value.Details.Clone()
	output.Causes = make([]SanitizedError, len(value.Causes))
	for index := range value.Causes {
		output.Causes[index] = cloneEffectCause(
			value.Causes[index],
		)
	}
	return &output
}

func cloneEffectCause(value SanitizedError) SanitizedError {
	output := value
	output.Details = value.Details.Clone()
	output.Causes = make([]SanitizedError, len(value.Causes))
	for index := range value.Causes {
		output.Causes[index] = cloneEffectCause(value.Causes[index])
	}
	return output
}

func recordEffectSettlement(
	item *WorkItem,
	attempt *Attempt,
	result WorkExecutionResult,
) {
	if item == nil {
		return
	}

	effect := effectStateForSettlement(*item, result)
	item.EffectState = effect
	item.Error = copyEffectError(item.Error, effect)

	if item.Output != nil &&
		item.Output.Receipt != nil {
		item.Output.Receipt.EffectState = effect
	}

	if attempt == nil {
		return
	}
	attempt.EffectState = effect
	attempt.Error = copyEffectError(attempt.Error, effect)
	if attempt.Receipt != nil {
		attempt.Receipt.EffectState = effect
	}
}

func effectReplaySafe(effect EffectContract) bool {
	switch effect.Kind {
	case EffectKindRead,
		EffectKindIdempotent,
		EffectKindKeyed:
		return true
	default:
		return false
	}
}

func retryEffectSafe(effect EffectContract) bool {
	return effectReplaySafe(effect)
}

func validateEffectStateForWork(
	item WorkItem,
) error {
	if err := item.EffectState.Validate(); err != nil {
		return err
	}

	switch item.State {
	case WorkItemStateReady,
		WorkItemStateBinding,
		WorkItemStateRetryWaiting:
		switch item.EffectState {
		case "",
			EffectStateNotApplicable,
			EffectStateNotStarted:
		default:
			return errors.New(
				"undispatched work item has a settled effect state",
			)
		}

	case WorkItemStateDispatched:
		if item.Effect.Kind != EffectKindRead &&
			item.EffectState != EffectStateDispatched {
			return errors.New(
				"dispatched mutating work item has no dispatched effect state",
			)
		}

	case WorkItemStateUncertain:
		if item.Effect.Kind != EffectKindRead &&
			item.EffectState != EffectStateUncertain {
			return errors.New(
				"uncertain mutating work item has no uncertain effect state",
			)
		}
	default:
	}

	return nil
}
