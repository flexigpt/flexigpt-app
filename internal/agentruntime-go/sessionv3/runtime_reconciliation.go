package sessionv3

import (
	"errors"
	"fmt"
)

func (r *Runtime) reconciliationRequiredLocked(
	execution *Execution,
	item *WorkItem,
) bool {
	if execution == nil ||
		item == nil ||
		item.Effect.Kind == EffectKindRead ||
		!item.Effect.Reconcile ||
		item.EffectPlan == nil ||
		item.EffectPlan.Reconcile == nil {
		return false
	}
	frame := mutableFrame(execution, item.FrameID)
	return frame != nil &&
		frame.Program.InterpreterID == WorkflowInterpreterID
}

func (r *Runtime) applyReconciliationSettlementsLocked(
	session *Session,
	execution *Execution,
	values []ReconciliationSettlement,
) ([]WorkItemID, error) {
	if len(values) == 0 {
		return nil, nil
	}
	if session == nil || execution == nil {
		return nil, errors.New("reconciliation settlement owner is unavailable")
	}

	failed := make([]WorkItemID, 0)
	for _, settlement := range values {
		if err := settlement.Validate(); err != nil {
			return nil, err
		}
		item := mutableWorkItem(execution, settlement.OriginalWorkItemID)
		if item == nil {
			return nil, errors.New(
				"reconciliation settlement original work item is unavailable",
			)
		}
		if item.State != WorkItemStateReconciling {
			return nil, fmt.Errorf(
				"reconciliation settlement original work item %q is not reconciling",
				item.ID,
			)
		}

		attempt := mutableLatestAttemptForWorkItem(execution, item.ID)
		switch settlement.Decision {
		case ReconciliationDecisionApplied:
			output, err := reconciliationAppliedOutput(
				*item,
				settlement,
			)
			if err != nil {
				return nil, err
			}
			r.settleReconciledAppliedLocked(
				execution,
				item,
				attempt,
				output,
			)

		case ReconciliationDecisionNotApplied:
			if r.settleReconciledNotAppliedLocked(
				execution,
				item,
				attempt,
			) {
				continue
			}
			failed = append(failed, item.ID)

		case ReconciliationDecisionUnknown:
			r.settleReconciledUnknownLocked(
				item,
				attempt,
				settlement.Error,
			)

		default:
			return nil, errors.New("reconciliation settlement decision is unavailable")
		}
	}
	return failed, nil
}

func reconciliationAppliedOutput(
	item WorkItem,
	settlement ReconciliationSettlement,
) (*WorkItemOutput, error) {
	if settlement.Receipt == nil {
		return nil, errors.New("applied reconciliation settlement has no receipt")
	}

	output := &WorkItemOutput{
		Value:   settlement.Output.Clone(),
		Receipt: cloneReconciliationReceipt(settlement.Receipt),
	}
	if item.Type == WorkItemTypeTool {
		output.Tool = &ToolOutput{
			Kind:       ToolOutputKindSuccess,
			Text:       string(settlement.Output),
			Structured: settlement.Output.Clone(),
		}
	}
	if err := output.Validate(item.Type); err != nil {
		return nil, err
	}
	return output, nil
}

func (r *Runtime) settleReconciledAppliedLocked(
	execution *Execution,
	item *WorkItem,
	attempt *Attempt,
	output *WorkItemOutput,
) {
	now := r.now()
	item.State = WorkItemStateCompleted
	item.Output = output
	item.Error = nil
	item.CompletedAt = &now
	item.UpdatedAt = now
	item.WaitID = ""

	if attempt != nil {
		attempt.State = AttemptStateConfirmed
		attempt.FinishedAt = &now
		attempt.Error = nil
		attempt.Receipt = cloneReconciliationReceipt(output.Receipt)
	}

	recordEffectSettlement(
		item,
		attempt,
		WorkExecutionResult{
			Output:    output,
			Confirmed: true,
		},
	)
}

func (r *Runtime) settleReconciledNotAppliedLocked(
	execution *Execution,
	item *WorkItem,
	attempt *Attempt,
) bool {
	now := r.now()
	failure := reconciledNonApplicationError(item.Error)
	item.State = WorkItemStateFailed
	item.Output = nil
	item.Error = failure
	item.CompletedAt = &now
	item.UpdatedAt = now
	item.WaitID = ""

	if attempt != nil {
		attempt.State = AttemptStateFailed
		attempt.FinishedAt = &now
		attempt.Error = cloneReconciliationError(failure)
		attempt.Receipt = nil
	}

	recordEffectSettlement(
		item,
		attempt,
		WorkExecutionResult{
			Error:     failure,
			Confirmed: true,
		},
	)
	return r.prepareRetryLocked(execution, item, attempt, now)
}

func (r *Runtime) settleReconciledUnknownLocked(
	item *WorkItem,
	attempt *Attempt,
	cause *SanitizedError,
) {
	now := r.now()
	item.State = WorkItemStateUncertain
	item.Output = nil
	item.Error = cloneReconciliationError(cause)
	if item.Error == nil {
		item.Error = &SanitizedError{
			Code:     "reconciliation_unknown",
			Category: ErrorCategoryUncertainty,
			Effect:   EffectStateUncertain,
			Message:  "reconciliation did not confirm the external effect",
		}
	}
	item.CompletedAt = &now
	item.UpdatedAt = now
	item.WaitID = ""

	if attempt != nil {
		attempt.State = AttemptStateUncertain
		attempt.FinishedAt = &now
		attempt.Error = cloneReconciliationError(item.Error)
		attempt.Receipt = nil
	}
	recordEffectSettlement(
		item,
		attempt,
		WorkExecutionResult{
			Error: item.Error,
		},
	)
}

func reconciledNonApplicationError(
	previous *SanitizedError,
) *SanitizedError {
	if previous == nil {
		return &SanitizedError{
			Code:      "reconciled_not_applied",
			Category:  ErrorCategoryExecution,
			Retryable: true,
			Effect:    EffectStateConfirmedNotApplied,
			Message:   "reconciliation confirmed that the external effect was not applied",
		}
	}

	output := cloneReconciliationError(previous)
	output.Effect = EffectStateConfirmedNotApplied
	output.Message = "reconciliation confirmed that the external effect was not applied" + ":" + output.Message

	return output
}

func mutableLatestAttemptForWorkItem(
	execution *Execution,
	workItemID WorkItemID,
) *Attempt {
	if execution == nil {
		return nil
	}
	for index := len(execution.Attempts) - 1; index >= 0; index-- {
		if execution.Attempts[index].WorkItemID == workItemID {
			return &execution.Attempts[index]
		}
	}
	return nil
}

func reconciliationResumeRequired(
	frame ProgramFrame,
	execution Execution,
) bool {
	if frame.Program.InterpreterID != WorkflowInterpreterID ||
		frame.Status.Terminal() {
		return false
	}
	_, state, err := decodeWorkflowProgram(frame)
	if err != nil ||
		state.Pending == nil ||
		state.Pending.Kind != WorkflowPendingReconciliation ||
		state.Pending.Reconciliation == nil {
		return false
	}

	progress := state.Pending.Reconciliation
	switch progress.Phase {
	case WorkflowReconciliationPhaseIntent:
		return executionWorkItem(execution, progress.IntentWorkItemID) == nil
	case WorkflowReconciliationPhaseReview:
		return progress.ReviewWorkItemID == "" ||
			executionWorkItem(execution, progress.ReviewWorkItemID) == nil
	default:
		return false
	}
}
