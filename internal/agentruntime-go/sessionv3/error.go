package sessionv3

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

func deadlineError() *SanitizedError {
	return &SanitizedError{
		Code:     "work_deadline_elapsed",
		Category: ErrorCategoryTimeout,
		Message:  "work deadline elapsed",
	}
}
