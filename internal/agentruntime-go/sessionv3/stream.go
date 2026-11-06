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
		execution.Controls.CancellationRequested ||
		execution.State.Terminal() {
		return context.Canceled
	}

	now := r.now()
	if item.LLMStream == nil {
		item.LLMStream = &LLMStreamState{}
	}
	if thinking {
		item.LLMStream.Thinking += value
	} else {
		item.LLMStream.Text += value
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
	item.UpdatedAt = now
	r.touchExecutionLocked(session, execution)

	if err := r.persistLocked(ctx, "llmStream", session); err != nil {
		return err
	}

	r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
	return nil
}

func mergeLLMStreamResult(
	item WorkItem,
	result WorkExecutionResult,
) WorkExecutionResult {
	if item.LLMStream == nil ||
		result.Output == nil ||
		result.Output.LLM == nil {
		return result
	}

	output := *result.Output
	llm := *output.LLM
	llm.Text = mergeLLMStreamText(llm.Text, item.LLMStream.Text)
	llm.Thinking = mergeLLMStreamText(
		llm.Thinking,
		item.LLMStream.Thinking,
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
