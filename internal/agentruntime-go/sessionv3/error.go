package sessionv3

const (
	workflowChildCancelled            = "workflow_child_cancelled"
	completionCancelled               = "completion_cancelled"
	workflowProgramFrameFailedCode    = "program_frame_failed"
	workflowProgramFrameCancelledCode = "program_frame_cancelled"
	workflowProgramFrameFailedMessage = "a root program frame failed"
	workflowProgramFrameCancelMessage = "a root program frame was cancelled"
	cleanupDeadlineExceededCode       = "cleanup_deadline_elapsed"
	cleanupDeadlineExceededMessage    = "cleanup deadline elapsed"
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
