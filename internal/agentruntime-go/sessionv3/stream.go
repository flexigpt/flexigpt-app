package sessionv3

import (
	"context"
	"strings"
)

func (r *Runtime) appendLLMStream(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	workItemID WorkItemID,
	value string,
	thinking bool,
) error {
	if value == "" {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrRuntimeClosed
	}

	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		return err
	}

	item := mutableWorkItem(execution, workItemID)
	if item == nil ||
		item.Type != WorkItemTypeLLM ||
		item.State != WorkItemStateDispatched ||
		execution.State.Terminal() ||
		(execution.Controls.CancellationRequested &&
			!r.cleanupDispatchAllowedLocked(*execution, *item)) {
		return context.Canceled
	}

	if err := r.reserveScopedBudgetLocked(
		execution,
		item,
		item.FrameID,
		budgetMetricRequest{
			Metric:    budgetMetricMessageBytes,
			Increment: len(value),
			Label:     budgetLabelMessageBytes,
		},
	); err != nil {
		return err
	}

	now := r.now()
	if item.LLMStream == nil {
		item.LLMStream = &LLMStreamState{}
	}
	attempt := activeAttempt(execution, item.ID)
	if attempt != nil {
		if attempt.LLMStream == nil ||
			attempt.LLMStream.AttemptID != attempt.ID {
			attempt.LLMStream = &LLMStreamState{
				AttemptID: attempt.ID,
			}
		}
		item.LLMStream.AttemptID = attempt.ID
	}
	if thinking {
		item.LLMStream.Thinking += value
		if attempt != nil {
			attempt.LLMStream.Thinking += value
		}
	} else {
		item.LLMStream.Text += value
		if attempt != nil {
			attempt.LLMStream.Text += value
		}
	}
	if !thinking {
		if err := r.deliverLLMStreamPortsLocked(
			session,
			execution,
			item,
			value,
		); err != nil {
			return err
		}
	}
	item.LLMStream.UpdatedAt = now
	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return ErrExecutionNotFound
	}
	item = mutableWorkItem(execution, workItemID)
	if item == nil {
		return ErrExecutionNotFound
	}
	item.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)

	if err := r.persistLocked(ctx, checkpointReasonLLMStream, session); err != nil {
		return err
	}

	r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
	return nil
}

func mergeLLMStreamResult(
	item WorkItem,
	attempt *Attempt,
	result WorkExecutionResult,
) WorkExecutionResult {
	stream := item.LLMStream
	if attempt != nil && attempt.LLMStream != nil {
		stream = attempt.LLMStream
	}
	if stream == nil ||
		result.Output == nil ||
		result.Output.LLM == nil {
		return result
	}

	output := *result.Output
	llm := *output.LLM
	llm.Text = mergeLLMStreamText(llm.Text, stream.Text)
	llm.Thinking = mergeLLMStreamText(
		llm.Thinking,
		stream.Thinking,
	)
	output.LLM = &llm
	result.Output = &output

	if strings.TrimSpace(llm.Text) != "" {
		result.Confirmed = true
	}
	return result
}

func mergeLLMStreamText(
	finalValue string,
	streamedValue string,
) string {
	if streamedValue == "" {
		return finalValue
	}
	if finalValue == "" {
		return streamedValue
	}
	if strings.HasPrefix(finalValue, streamedValue) {
		return finalValue
	}
	if strings.HasPrefix(streamedValue, finalValue) {
		return streamedValue
	}
	return finalValue
}
