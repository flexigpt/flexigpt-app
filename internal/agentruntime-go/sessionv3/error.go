package sessionv3

const (
	workflowChildCancelled = "workflow_child_cancelled"
	completionCancelled    = "completion_cancelled"
)

// ErrorCategory identifies runtime-owned failure classification. Callable
// domains can supply additional categories through their declared contract.
type ErrorCategory string

const (
	ErrorCategoryValidation    ErrorCategory = "validation"
	ErrorCategoryRouting       ErrorCategory = "routing"
	ErrorCategoryExecution     ErrorCategory = "execution"
	ErrorCategoryTransient     ErrorCategory = "transient"
	ErrorCategoryTimeout       ErrorCategory = "timeout"
	ErrorCategoryAuthorization ErrorCategory = "authorization"
	ErrorCategoryLimit         ErrorCategory = "limit"
	ErrorCategoryCancellation  ErrorCategory = "cancellation"
	ErrorCategoryUncertainty   ErrorCategory = "uncertainty"
)

func (e *SanitizedError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func deadlineError() *SanitizedError {
	return &SanitizedError{
		Code:     "work_deadline_elapsed",
		Category: ErrorCategoryTimeout,
		Message:  "work deadline elapsed",
	}
}
