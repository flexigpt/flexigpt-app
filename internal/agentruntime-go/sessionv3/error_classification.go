package sessionv3

import (
	"context"
	"errors"
)

// DomainFailure exposes a typed native-domain failure without making Runtime
// infer business meaning from an error message.
type DomainFailure interface {
	RuntimeFailure() *SanitizedError
}

// ClassifyDomainError preserves a supplied domain classification. Runtime only
// classifies its own cancellation and deadline contexts when no domain value
// is available.
func ClassifyDomainError(
	err error,
	fallbackCode string,
	fallbackCategory ErrorCategory,
) *SanitizedError {
	if err == nil {
		return nil
	}

	var supplied *SanitizedError
	if errors.As(err, &supplied) && supplied != nil {
		return cloneWorkflowSanitizedError(supplied)
	}

	var classified DomainFailure
	if errors.As(err, &classified) && classified != nil {
		if value := classified.RuntimeFailure(); value != nil {
			return cloneWorkflowSanitizedError(value)
		}
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return &SanitizedError{
			Code:      fallbackCode,
			Category:  ErrorCategoryTimeout,
			Retryable: true,
			Message:   err.Error(),
		}
	case errors.Is(err, context.Canceled):
		return &SanitizedError{
			Code:     fallbackCode,
			Category: ErrorCategoryCancellation,
			Message:  err.Error(),
		}
	default:
		return &SanitizedError{
			Code:     fallbackCode,
			Category: fallbackCategory,
			Message:  err.Error(),
		}
	}
}

func cloneWaitResolutionIntent(
	value *WaitResolutionIntent,
) *WaitResolutionIntent {
	if value == nil {
		return nil
	}
	output := *value
	output.Response = value.Response.Clone()
	output.Actor = cloneHumanActor(value.Actor)
	return &output
}
