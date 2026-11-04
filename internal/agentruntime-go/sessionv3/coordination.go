package sessionv3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	maximumCoordinationRecords = 65_536
	maximumCoordinationBytes   = 32 << 20
)

type eventWaitOutput struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Source      string         `json:"source,omitempty"`
	Data        CanonicalValue `json:"data"`
	Correlation CanonicalValue `json:"correlation,omitempty"`
	Time        time.Time      `json:"time"`
}

type channelWaitOutput struct {
	ID          string         `json:"id"`
	Sequence    uint64         `json:"sequence"`
	Channel     string         `json:"channel"`
	Kind        string         `json:"kind,omitempty"`
	Data        CanonicalValue `json:"data"`
	Correlation CanonicalValue `json:"correlation,omitempty"`
	Time        time.Time      `json:"time"`
}

type completionWaitOutput struct {
	WorkItemID WorkItemID     `json:"workItemID,omitempty"`
	FrameID    ProgramFrameID `json:"frameID,omitempty"`
	Output     CanonicalValue `json:"output,omitempty"`
}

func canonicalValuesEqual(left, right CanonicalValue) bool {
	return bytes.Equal(left, right)
}

func containsWaitID(values []WaitID, want WaitID) bool {
	return slices.Contains(values, want)
}

func waitMatchesEvent(
	wait Wait,
	event CanonicalValue,
	source string,
	data CanonicalValue,
	correlation CanonicalValue,
) bool {
	if wait.Request.EventSource != "" &&
		wait.Request.EventSource != source {
		return false
	}
	if len(wait.Request.Correlation) != 0 &&
		!canonicalValuesEqual(wait.Request.Correlation, correlation) {
		return false
	}
	if err := validateWorkflowSchema(wait.Request.DataSchema, data); err != nil {
		return false
	}
	if err := validateWorkflowSchema(
		wait.Request.CorrelationSchema,
		correlation,
	); err != nil {
		return false
	}
	if wait.Request.Filter == nil {
		return true
	}
	matched, err := wait.Request.Filter.Matches(event)
	return err == nil && matched
}

func (r *Runtime) resolveReadyWaitLocked(
	session *Session,
	execution *Execution,
	waitID WaitID,
) error {
	wait := mutableWait(execution, waitID)
	if wait == nil || wait.Status != WaitStatusPending {
		return nil
	}

	switch wait.Request.Kind {
	case WaitKindEvent:
		for _, signal := range execution.Signals {
			if err := r.wakeEventWaitsLocked(session, execution, signal); err != nil {
				return err
			}
			if wait.Status != WaitStatusPending {
				return nil
			}
		}

	case WaitKindChannel:
		for _, channel := range execution.Channels {
			if channel.Name != wait.Request.Channel {
				continue
			}
			for _, message := range channel.Messages {
				if err := r.wakeChannelWaitsLocked(session, execution, message); err != nil {
					return err
				}
				if wait.Status != WaitStatusPending {
					return nil
				}
			}
		}

	case WaitKindChild:
		if wait.Request.Completion == nil {
			return errors.New("completion wait has no completion handle")
		}
		handle := wait.Request.Completion
		if handle.WorkItemID != "" {
			item := executionWorkItem(*execution, handle.WorkItemID)
			if item != nil && item.State.Terminal() {
				r.wakeCompletionWorkWaitsLocked(session, execution, *item)
			}
		}
		if handle.FrameID != "" {
			frame := mutableFrame(execution, handle.FrameID)
			if frame != nil && frame.Status.Terminal() {
				r.wakeCompletionFrameWaitsLocked(session, execution, *frame)
			}
		}
	default:
	}

	return nil
}

func (r *Runtime) dispatchEmit(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	workItemID WorkItemID,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	item := mutableWorkItem(execution, workItemID)
	if item == nil ||
		item.State != WorkItemStateReady ||
		item.Type != WorkItemTypeEmit ||
		execution.Controls.Paused ||
		execution.Controls.CancellationRequested ||
		execution.State.Terminal() {
		return
	}
	if item.Input.Emit == nil {
		now := r.now()
		item.State = WorkItemStateBlocked
		item.Error = &SanitizedError{
			Code:     "emit_payload_missing",
			Category: ErrorCategoryValidation,
			Message:  "emit WorkItem has no payload",
		}
		item.CompletedAt = &now
		item.UpdatedAt = now
		r.touchExecutionLocked(session, execution)
		_ = r.persistLocked(ctx, "emitBlocked", session)
		return
	}

	previous := cloneSession(*session)
	item.State = WorkItemStateBinding
	item.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)

	bound := BoundWork{
		SemanticBinding: SemanticBinding{
			Version:           "session-v3-coordination/v1",
			TargetFingerprint: "emit:" + string(item.ID),
			BoundAt:           r.now(),
		},
	}
	if err := r.startDispatchedLocked(ctx, session, execution, item, bound); err != nil {
		*session = previous
		return
	}

	receipt, signal, channelMessage, emitErr := r.applyEmitLocked(
		execution,
		*item.Input.Emit,
		item.ID,
	)
	if emitErr != nil {
		r.finishWorkLocked(
			session,
			execution,
			item,
			activeAttempt(execution, item.ID),
			WorkExecutionResult{
				Error: &SanitizedError{
					Code:     "emit_failed",
					Category: ErrorCategoryExecution,
					Message:  emitErr.Error(),
				},
				Confirmed: true,
			},
		)
	} else {
		value := MustCanonicalValue(receipt)
		r.finishWorkLocked(
			session,
			execution,
			item,
			activeAttempt(execution, item.ID),
			WorkExecutionResult{
				Output: &WorkItemOutput{
					Value: value,
					Emit:  &receipt,
					Receipt: &ExternalReceipt{
						System:      "session-v3-coordination",
						Reference:   receipt.ID,
						ConfirmedAt: receipt.AcceptedAt,
					},
				},
				Confirmed: true,
			},
		)

		switch {
		case signal != nil:
			if err := r.wakeEventWaitsLocked(session, execution, *signal); err != nil {
				*session = previous
				return
			}
		case channelMessage != nil:
			if err := r.wakeChannelWaitsLocked(
				session,
				execution,
				*channelMessage,
			); err != nil {
				*session = previous
				return
			}
		}
	}

	if err := r.persistLocked(ctx, "emitFinished", session); err != nil {
		return
	}
	r.armExecutionRetriesLocked(ctx, session.ID, execution.ID, *execution)
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
	if item.State != WorkItemStateRetryWaiting {
		r.launchWork(
			ctx,
			session.ID,
			execution.ID,
			r.readyWorkLocked(*execution),
		)
	}
}

func (r *Runtime) applyEmitLocked(
	execution *Execution,
	payload EmitWork,
	workItemID WorkItemID,
) (EmitReceipt, *ExecutionSignal, *ChannelMessage, error) {
	if execution == nil {
		return EmitReceipt{}, nil, nil, errors.New("execution is unavailable")
	}
	if !payload.Data.Valid() || !payload.Correlation.Valid() {
		return EmitReceipt{}, nil, nil, errors.New("emit payload is not canonical JSON")
	}
	if err := validateWorkflowSchema(payload.DataSchema, payload.Data); err != nil {
		return EmitReceipt{}, nil, nil, fmt.Errorf("validate emit data: %w", err)
	}
	if err := validateWorkflowSchema(
		payload.CorrelationSchema,
		payload.Correlation,
	); err != nil {
		return EmitReceipt{}, nil, nil, fmt.Errorf(
			"validate emit correlation: %w",
			err,
		)
	}
	switch {
	case payload.Destination.Channel == "" &&
		payload.Destination.Event == "":
		return EmitReceipt{}, nil, nil, errors.New("emit destination is required")
	case payload.Destination.Channel != "" &&
		payload.Destination.Event != "":
		return EmitReceipt{}, nil, nil, errors.New("emit destination is ambiguous")
	}

	if receipt, found := findEmitReceipt(
		*execution,
		payload.Destination,
		payload.DeduplicationKey,
	); found {
		return receipt, nil, nil, nil
	}
	if len(execution.EmitReceipts) >= maximumCoordinationRecords {
		return EmitReceipt{}, nil, nil, errors.New("execution emit receipt limit exceeded")
	}

	now := r.now()
	receipt := EmitReceipt{
		ID:               string(workItemID),
		Destination:      payload.Destination,
		DeduplicationKey: payload.DeduplicationKey,
		AcceptedAt:       now,
	}

	switch {
	case payload.Destination.Channel != "":
		channel, err := mutableChannelLog(execution, payload.Destination.Channel)
		if err != nil {
			return EmitReceipt{}, nil, nil, err
		}
		maxMessages := maximumCoordinationRecords
		if payload.MaxMessages > 0 && payload.MaxMessages < maxMessages {
			maxMessages = payload.MaxMessages
		}
		if len(channel.Messages) >= maxMessages {
			return EmitReceipt{}, nil, nil, errors.New("channel retention limit exceeded")
		}
		maxBytes := maximumCoordinationBytes
		if payload.MaxBytes > 0 && payload.MaxBytes < maxBytes {
			maxBytes = payload.MaxBytes
		}
		if channelBytes(*channel)+len(payload.Data)+len(payload.Correlation) > maxBytes {
			return EmitReceipt{}, nil, nil, errors.New("channel byte limit exceeded")
		}

		channel.NextSequence++
		message := ChannelMessage{
			ID:               string(workItemID),
			Sequence:         channel.NextSequence,
			Channel:          channel.Name,
			Kind:             payload.Kind,
			Data:             payload.Data.Clone(),
			Correlation:      payload.Correlation.Clone(),
			DeduplicationKey: payload.DeduplicationKey,
			CreatedAt:        now,
		}
		channel.Messages = append(channel.Messages, message)
		receipt.Sequence = message.Sequence
		execution.EmitReceipts = append(execution.EmitReceipts, receipt)
		return receipt, nil, &message, nil

	case payload.Destination.Event != "":
		if len(execution.Signals) >= maximumCoordinationRecords {
			return EmitReceipt{}, nil, nil, errors.New("execution signal retention limit exceeded")
		}
		signal := ExecutionSignal{
			ID:          "emit:" + string(workItemID),
			Name:        payload.Destination.Event,
			Source:      firstNonEmpty(payload.EventSource, "workflow"),
			Data:        payload.Data.Clone(),
			Correlation: payload.Correlation.Clone(),
			CreatedAt:   now,
		}
		execution.Signals = append(execution.Signals, signal)
		execution.EmitReceipts = append(execution.EmitReceipts, receipt)
		return receipt, &signal, nil, nil
	}

	return EmitReceipt{}, nil, nil, errors.New("emit destination is unavailable")
}

func findEmitReceipt(
	execution Execution,
	destination EmitDestination,
	deduplicationKey string,
) (EmitReceipt, bool) {
	if strings.TrimSpace(deduplicationKey) == "" {
		return EmitReceipt{}, false
	}
	for _, receipt := range execution.EmitReceipts {
		if receipt.DeduplicationKey != deduplicationKey {
			continue
		}
		if receipt.Destination == destination {
			return receipt, true
		}
	}
	return EmitReceipt{}, false
}

func mutableChannelLog(
	execution *Execution,
	name string,
) (*ChannelLog, error) {
	if execution == nil || strings.TrimSpace(name) == "" {
		return nil, errors.New("channel name is required")
	}
	for index := range execution.Channels {
		if execution.Channels[index].Name == name {
			return &execution.Channels[index], nil
		}
	}
	execution.Channels = append(execution.Channels, ChannelLog{
		Name:     name,
		Messages: []ChannelMessage{},
	})
	return &execution.Channels[len(execution.Channels)-1], nil
}

func channelBytes(channel ChannelLog) int {
	total := 0
	for _, message := range channel.Messages {
		total += len(message.Data) + len(message.Correlation)
	}
	return total
}

func (r *Runtime) wakeEventWaitsLocked(
	session *Session,
	execution *Execution,
	signal ExecutionSignal,
) error {
	if session == nil || execution == nil {
		return errors.New("event wait owner is unavailable")
	}

	for index := range execution.Waits {
		wait := &execution.Waits[index]
		if wait.Status != WaitStatusPending ||
			wait.Request.Kind != WaitKindEvent ||
			wait.Request.Signal != signal.Name {
			continue
		}
		if wait.Request.Consume &&
			containsWaitID(signal.ConsumedBy, wait.ID) {
			continue
		}

		response := MustCanonicalValue(eventWaitOutput{
			ID:          signal.ID,
			Name:        signal.Name,
			Source:      signal.Source,
			Data:        signal.Data.Clone(),
			Correlation: signal.Correlation.Clone(),
			Time:        signal.CreatedAt,
		})
		if !waitMatchesEvent(
			*wait,
			response,
			signal.Source,
			signal.Data,
			signal.Correlation,
		) {
			continue
		}
		if err := validateWorkflowSchema(wait.Request.InputSchema, response); err != nil {
			return fmt.Errorf("validate event wait response: %w", err)
		}
		if err := r.resolveDurableWaitLocked(
			session,
			execution,
			wait,
			response,
			&ExternalReceipt{
				System:      "session-v3-event",
				Reference:   signal.ID,
				ConfirmedAt: r.now(),
			},
			nil,
			true,
		); err != nil {
			return err
		}

		if wait.Request.Consume {
			for signalIndex := range execution.Signals {
				if execution.Signals[signalIndex].ID != signal.ID {
					continue
				}
				execution.Signals[signalIndex].ConsumedBy = append(
					execution.Signals[signalIndex].ConsumedBy,
					wait.ID,
				)
				break
			}
		}
	}
	return nil
}

func (r *Runtime) wakeChannelWaitsLocked(
	session *Session,
	execution *Execution,
	message ChannelMessage,
) error {
	if session == nil || execution == nil {
		return errors.New("channel wait owner is unavailable")
	}

	for index := range execution.Waits {
		wait := &execution.Waits[index]
		if wait.Status != WaitStatusPending ||
			wait.Request.Kind != WaitKindChannel ||
			wait.Request.Channel != message.Channel ||
			message.Sequence <= wait.Request.Cursor {
			continue
		}
		if wait.Request.Consume &&
			containsWaitID(message.ConsumedBy, wait.ID) {
			continue
		}

		response := MustCanonicalValue(channelWaitOutput{
			ID:          message.ID,
			Sequence:    message.Sequence,
			Channel:     message.Channel,
			Kind:        message.Kind,
			Data:        message.Data.Clone(),
			Correlation: message.Correlation.Clone(),
			Time:        message.CreatedAt,
		})
		if !waitMatchesEvent(
			*wait,
			response,
			"",
			message.Data,
			message.Correlation,
		) {
			continue
		}
		if err := validateWorkflowSchema(wait.Request.InputSchema, response); err != nil {
			return fmt.Errorf("validate channel wait response: %w", err)
		}
		if err := r.resolveDurableWaitLocked(
			session,
			execution,
			wait,
			response,
			&ExternalReceipt{
				System:      "session-v3-channel",
				Reference:   message.ID,
				ConfirmedAt: r.now(),
			},
			nil,
			true,
		); err != nil {
			return err
		}

		if wait.Request.Consume {
			for channelIndex := range execution.Channels {
				channel := &execution.Channels[channelIndex]
				if channel.Name != message.Channel {
					continue
				}
				for messageIndex := range channel.Messages {
					if channel.Messages[messageIndex].ID != message.ID {
						continue
					}
					channel.Messages[messageIndex].ConsumedBy = append(
						channel.Messages[messageIndex].ConsumedBy,
						wait.ID,
					)
					break
				}
			}
		}
	}
	return nil
}

func (r *Runtime) resolveDurableWaitLocked(
	session *Session,
	execution *Execution,
	wait *Wait,
	value CanonicalValue,
	receipt *ExternalReceipt,
	errorValue *SanitizedError,
	confirmed bool,
) error {
	if session == nil || execution == nil || wait == nil {
		return errors.New("wait owner is unavailable")
	}
	item := mutableWorkItem(execution, wait.WorkItemID)
	if item == nil || item.State != WorkItemStateWaiting {
		return errors.New("wait owner WorkItem is unavailable")
	}

	wait.Status = WaitStatusResolved
	wait.Response = value.Clone()
	wait.Error = errorValue
	wait.UpdatedAt = r.now()
	item.WaitID = ""
	r.cancelWaitDeadlineLocked(wait.ID)
	delete(r.privateWaits, wait.ID)

	r.finishWorkLocked(
		session,
		execution,
		item,
		activeAttempt(execution, item.ID),
		WorkExecutionResult{
			Output: &WorkItemOutput{
				Value:   value.Clone(),
				Receipt: receipt,
			},
			Error:     errorValue,
			Confirmed: confirmed,
		},
	)
	return nil
}

func (r *Runtime) wakeCompletionWorkWaitsLocked(
	session *Session,
	execution *Execution,
	completed WorkItem,
) {
	for index := range execution.Waits {
		wait := &execution.Waits[index]
		if wait.Status != WaitStatusPending ||
			wait.Request.Kind != WaitKindChild ||
			wait.Request.Completion == nil ||
			wait.Request.Completion.WorkItemID != completed.ID {
			continue
		}

		value, errorValue, confirmed := completionWorkItemResult(completed)
		if err := validateWorkflowSchema(wait.Request.InputSchema, value); err != nil {
			continue
		}
		_ = r.resolveDurableWaitLocked(
			session,
			execution,
			wait,
			value,
			&ExternalReceipt{
				System:      "session-v3-completion",
				Reference:   string(completed.ID),
				ConfirmedAt: r.now(),
			},
			errorValue,
			confirmed,
		)
	}
}

func (r *Runtime) wakeCompletionFrameWaitsLocked(
	session *Session,
	execution *Execution,
	frame ProgramFrame,
) {
	if !frame.Status.Terminal() {
		return
	}
	for index := range execution.Waits {
		wait := &execution.Waits[index]
		if wait.Status != WaitStatusPending ||
			wait.Request.Kind != WaitKindChild ||
			wait.Request.Completion == nil ||
			wait.Request.Completion.FrameID != frame.ID {
			continue
		}

		value := MustCanonicalValue(completionWaitOutput{
			FrameID: frame.ID,
			Output:  frame.Output.Clone(),
		})
		var errorValue *SanitizedError
		confirmed := frame.Status == ProgramFrameStateCompleted
		if frame.Status == ProgramFrameStateFailed && frame.Error != nil {
			copyValue := *frame.Error
			copyValue.Details = frame.Error.Details.Clone()
			copyValue.Causes = append([]SanitizedError(nil), frame.Error.Causes...)
			errorValue = &copyValue
		}
		if frame.Status == ProgramFrameStateCancelled {
			errorValue = &SanitizedError{
				Code:     "completion_cancelled",
				Category: ErrorCategoryCancellation,
				Message:  "completed frame was cancelled",
			}
		}
		if err := validateWorkflowSchema(wait.Request.InputSchema, value); err != nil {
			continue
		}
		_ = r.resolveDurableWaitLocked(
			session,
			execution,
			wait,
			value,
			&ExternalReceipt{
				System:      "session-v3-completion",
				Reference:   string(frame.ID),
				ConfirmedAt: r.now(),
			},
			errorValue,
			confirmed,
		)
	}
}

func completionWorkItemResult(
	item WorkItem,
) (CanonicalValue, *SanitizedError, bool) {
	output := CanonicalValue(nil)
	if item.Output != nil {
		output = item.Output.Value.Clone()
		if len(output) == 0 && item.Output.Tool != nil {
			output = item.Output.Tool.Structured.Clone()
		}
	}
	if len(output) == 0 {
		output = MustCanonicalValue(completionWaitOutput{
			WorkItemID: item.ID,
		})
	}

	switch item.State {
	case WorkItemStateCompleted:
		return output, nil, true
	case WorkItemStateCancelled:
		return output, &SanitizedError{
			Code:     "completion_cancelled",
			Category: ErrorCategoryCancellation,
			Message:  "completed WorkItem was cancelled",
		}, true
	case WorkItemStateUncertain:
		return output, &SanitizedError{
			Code:     "completion_uncertain",
			Category: ErrorCategoryUncertainty,
			Message:  "completed WorkItem outcome is uncertain",
		}, false
	default:
		if item.Error != nil {
			copyValue := *item.Error
			copyValue.Details = item.Error.Details.Clone()
			copyValue.Causes = append([]SanitizedError(nil), item.Error.Causes...)
			return output, &copyValue, true
		}
		return output, &SanitizedError{
			Code:     "completion_failed",
			Category: ErrorCategoryExecution,
			Message:  "completed WorkItem did not succeed",
		}, true
	}
}
