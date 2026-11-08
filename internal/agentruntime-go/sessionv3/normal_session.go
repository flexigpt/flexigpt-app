package sessionv3

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

func (r *Runtime) UpdateContinuationInput(
	ctx context.Context,
	command UpdateContinuationInputCommand,
) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	if err := command.Input.Validate(); err != nil {
		return &CommandReceipt{
			CommandID:   command.Meta.ID,
			SessionID:   command.SessionID,
			ExecutionID: command.ExecutionID,
			Message:     err.Error(),
		}, nil
	}

	admitted, err := r.admitMessageAttachments(
		ctx,
		command.SessionID,
		command.Input.AttachmentSources,
	)
	if err != nil {
		return &CommandReceipt{
			CommandID:   command.Meta.ID,
			SessionID:   command.SessionID,
			ExecutionID: command.ExecutionID,
			Message:     err.Error(),
		}, nil
	}

	input := command.Input
	input.Attachments = append(
		append([]Attachment(nil), input.Attachments...),
		admitted...,
	)
	input.AttachmentSources = nil

	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if !isNormalConversationExecution(*execution) {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "continuation input requires a conversation execution",
				}, nil
			}
			if command.ExpectedInputRevision != nil &&
				execution.Continuation.Revision != *command.ExpectedInputRevision {
				return &CommandReceipt{
					CommandID:         command.Meta.ID,
					SessionID:         session.ID,
					ExecutionID:       execution.ID,
					SessionRevision:   session.Revision,
					ExecutionRevision: execution.Revision,
					Message: fmt.Sprintf(
						"continuation input revision mismatch: expected %d, current %d",
						*command.ExpectedInputRevision,
						execution.Continuation.Revision,
					),
				}, nil
			}

			continuation, inputErr := continuationFromMessageInput(
				execution.Continuation,
				input,
				session.Defaults,
				r.now(),
			)
			if inputErr != nil {
				return &CommandReceipt{
					CommandID:         command.Meta.ID,
					SessionID:         session.ID,
					ExecutionID:       execution.ID,
					SessionRevision:   session.Revision,
					ExecutionRevision: execution.Revision,
					Message:           inputErr.Error(),
				}, nil
			}
			execution.Continuation = continuation
			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
			}, nil
		},
	)
}

func (r *Runtime) ExecuteToolCalls(
	ctx context.Context,
	command ExecuteToolCallsCommand,
) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if execution.State.Terminal() ||
				execution.Controls.CancellationRequested {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "terminal or cancelling executions cannot execute Tool calls",
				}, nil
			}

			ids := uniqueNormalWorkItemIDs(command.WorkItemIDs)
			if len(ids) == 0 {
				for _, item := range execution.WorkItems {
					if item.Type == WorkItemTypeTool &&
						item.State == WorkItemStateReady {
						ids = append(ids, item.ID)
					}
				}
			}
			if len(ids) == 0 {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "no ready Tool WorkItems are available",
				}, nil
			}

			for _, id := range ids {
				item := mutableWorkItem(execution, id)
				if item == nil ||
					item.Type != WorkItemTypeTool ||
					item.State != WorkItemStateReady {
					return &CommandReceipt{
						CommandID:   command.Meta.ID,
						SessionID:   session.ID,
						ExecutionID: execution.ID,
						Message:     "requested Tool WorkItem is not ready",
					}, nil
				}
				item.Requested = true
				item.UpdatedAt = r.now()
			}

			if command.Continue {
				if err := r.requestAgentContinuationLocked(
					session,
					execution,
				); err != nil {
					return nil, err
				}
			}
			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
				WorkItemIDs:       ids,
			}, nil
		},
	)
}

func (r *Runtime) RetryToolCall(
	ctx context.Context,
	command RetryToolCallCommand,
) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			item := mutableWorkItem(execution, command.WorkItemID)
			if item == nil || item.Type != WorkItemTypeTool {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "Tool WorkItem is unavailable",
				}, nil
			}
			if !manualToolRetryAllowed(*item) {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "Tool WorkItem cannot retry without confirmed non-application",
				}, nil
			}

			r.cancelRetryLocked(item.ID)
			r.cancelResourceWaitLocked(item.ID)
			delete(r.privateBindings, item.ID)
			item.State = WorkItemStateReady
			item.Requested = true
			item.Binding = nil
			item.Output = nil
			item.Error = nil
			item.LLMStream = nil
			item.RetryState = nil
			item.ResourceWait = nil
			item.WaitID = ""
			item.DispatchedAt = nil
			item.CompletedAt = nil
			item.EffectState = undispatchedEffectState(item.Effect)
			item.PortStreams = nil
			item.UpdatedAt = r.now()

			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
				WorkItemIDs:       []WorkItemID{item.ID},
			}, nil
		},
	)
}

func (r *Runtime) SkipToolCall(
	ctx context.Context,
	command SkipToolCallCommand,
) (*CommandReceipt, error) {
	return r.completeNormalToolCall(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command.WorkItemID,
		ToolOutputKindSkipped,
		"The Tool call was intentionally skipped by the user.",
	)
}

func (r *Runtime) CompleteToolCallAsError(
	ctx context.Context,
	command CompleteToolCallAsErrorCommand,
) (*CommandReceipt, error) {
	message := strings.TrimSpace(command.Error)
	if message == "" {
		message = "The Tool call failed without a usable result."
	}
	return r.completeNormalToolCall(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command.WorkItemID,
		ToolOutputKindError,
		message,
	)
}

func (r *Runtime) SetAutomationHold(
	ctx context.Context,
	command SetAutomationHoldCommand,
) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		command,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if execution.State.Terminal() ||
				execution.Controls.CancellationRequested {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "terminal or cancelling executions cannot change automation",
				}, nil
			}
			if !command.Tools && !command.Continuation {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "automation hold must target tools, continuation, or both",
				}, nil
			}

			scope := command.Scope
			if scope == "" {
				scope = AutomationHoldOneBatch
			}
			if scope != AutomationHoldOneBatch &&
				scope != AutomationHoldIndefinite {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "automation hold scope is unavailable",
				}, nil
			}

			hold := AutomationHold{
				Enabled:          command.Enabled,
				Scope:            scope,
				TargetWorkItemID: latestNormalLLMWorkItemID(*execution),
				SetByCommandID:   command.Meta.ID,
			}
			if !command.Enabled {
				hold = AutomationHold{}
			}
			if command.Tools {
				execution.Controls.AutomaticTools = hold
			}
			if command.Continuation {
				execution.Controls.AutomaticContinuation = hold
			}
			if !command.Enabled {
				if err := r.requestAgentContinuationLocked(
					session,
					execution,
				); err != nil {
					return nil, err
				}
			}

			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         command.Meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
			}, nil
		},
	)
}

func (r *Runtime) EditAndResend(
	ctx context.Context,
	command EditAndResendCommand,
) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New("command ID is required")
	}
	if err := command.Input.Validate(); err != nil {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: command.SessionID,
			Message:   err.Error(),
		}, nil
	}

	admitted, err := r.admitMessageAttachments(
		ctx,
		command.SessionID,
		command.Input.AttachmentSources,
	)
	if err != nil {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: command.SessionID,
			Message:   err.Error(),
		}, nil
	}
	input := command.Input
	input.Attachments = append(
		append([]Attachment(nil), input.Attachments...),
		admitted...,
	)
	input.AttachmentSources = nil

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	session := r.sessions[command.SessionID]
	if session == nil {
		r.mu.Unlock()
		return nil, ErrSessionNotFound
	}
	if receipt, found, receiptErr := r.commandReceiptLocked(
		session,
		command.Meta,
		command,
	); receiptErr != nil || found {
		r.mu.Unlock()
		return receipt, receiptErr
	}
	if message := expectedSessionRevision(session, command.Meta); message != "" {
		receipt := r.rejectLocked(session, command.Meta, command, message)
		r.mu.Unlock()
		return receipt, nil
	}
	if !normalSessionEditBoundary(*session) {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			"editing requires a safe terminal conversation boundary",
		)
		r.mu.Unlock()
		return receipt, nil
	}

	index := -1
	for current := range session.Timeline {
		activity := session.Timeline[current]
		if activity.ID == command.ReplaceActivityID {
			index = current
			break
		}
	}
	if index < 0 ||
		session.Timeline[index].Kind != ActivityKindMessage ||
		session.Timeline[index].Message == nil ||
		session.Timeline[index].Message.Role != MessageRoleUser {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			"only a user message activity can be replaced",
		)
		r.mu.Unlock()
		return receipt, nil
	}
	if strings.TrimSpace(input.Text) == "" &&
		len(input.Attachments) == 0 &&
		len(command.CarryForwardWorkItemIDs) == 0 {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			"edit-and-resend requires text, attachments, or carried Tool results",
		)
		r.mu.Unlock()
		return receipt, nil
	}

	carried, carryErr := carriedNormalToolResults(
		*session,
		command.CarryForwardWorkItemIDs,
	)
	if carryErr != nil {
		receipt := r.rejectLocked(
			session,
			command.Meta,
			command,
			carryErr.Error(),
		)
		r.mu.Unlock()
		return receipt, nil
	}

	if command.ExecutionID != "" {
		execution := mutableSessionExecution(session, command.ExecutionID)
		if execution == nil || !isNormalConversationExecution(*execution) {
			receipt := r.rejectLocked(
				session,
				command.Meta,
				command,
				"edit-and-resend execution is unavailable",
			)
			r.mu.Unlock()
			return receipt, nil
		}
	}

	previous := cloneSession(*session)
	session.Timeline = append(
		[]Activity(nil),
		session.Timeline[:index]...,
	)

	execution, executionErr := r.conversationExecutionLocked(session)
	if executionErr != nil {
		*session = previous
		r.mu.Unlock()
		return nil, executionErr
	}
	activity, submitErr := r.submitNormalMessageLocked(
		session,
		execution,
		input,
		carried,
	)
	if submitErr != nil {
		*session = previous
		r.mu.Unlock()
		return nil, submitErr
	}

	current := mutableSessionExecution(session, execution.ID)
	if current == nil {
		*session = previous
		r.mu.Unlock()
		return nil, errors.New("conversation execution disappeared during edit-and-resend")
	}
	r.refreshExecutionStateLocked(current)
	receipt := CommandReceipt{
		CommandID:         command.Meta.ID,
		Accepted:          true,
		SessionID:         session.ID,
		ExecutionID:       current.ID,
		SessionRevision:   session.Revision,
		ExecutionRevision: current.Revision,
		ActivityIDs:       []ActivityID{activity.ID},
	}
	r.rememberCommandLocked(session, command.Meta, command, receipt)
	if err := r.persistLocked(ctx, "messageEditedAndResent", session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	ready := r.readyWorkForSessionLocked(session)
	r.publishLocked(ctx, session, current.ID, RuntimeEventExecutionChanged)
	r.mu.Unlock()
	r.launchReadyWork(ctx, session.ID, ready)
	return &receipt, nil
}

func continuationFromMessageInput(
	previous ContinuationInput,
	input MessageInput,
	defaults SessionDefaults,
	now time.Time,
) (ContinuationInput, error) {
	resolved, err := resolveSessionInvocationDefaults(defaults, input.Defaults)
	if err != nil {
		return ContinuationInput{}, err
	}
	return ContinuationInput{
		Revision:           previous.Revision + 1,
		ConsumedRevision:   previous.ConsumedRevision,
		Text:               input.Text,
		Attachments:        append([]Attachment(nil), input.Attachments...),
		AppContext:         cloneAppContextUpdates(previous.AppContext),
		Metadata:           input.Metadata.Clone(),
		InvocationDefaults: cloneInvocationDefaults(&resolved),
		InvocationBudget:   cloneBudget(input.InvocationBudget),
		UpdatedAt:          now,
	}, nil
}

func hasContinuationInput(value ContinuationInput) bool {
	return strings.TrimSpace(value.Text) != "" ||
		len(value.Attachments) != 0 ||
		len(value.AppContext) != 0
}

func (r *Runtime) consumeContinuationLocked(
	session *Session,
	execution *Execution,
) ([]ActivityID, bool, error) {
	if session == nil || execution == nil ||
		!isNormalConversationExecution(*execution) ||
		!hasContinuationInput(execution.Continuation) {
		return nil, false, nil
	}

	staged := execution.Continuation
	input := MessageInput{
		Text:             staged.Text,
		Attachments:      append([]Attachment(nil), staged.Attachments...),
		Metadata:         staged.Metadata.Clone(),
		InvocationBudget: cloneBudget(staged.InvocationBudget),
	}
	activity, err := r.submitNormalMessageWithDefaultsLocked(
		session,
		execution,
		input,
		nil,
		staged.InvocationDefaults,
	)
	if err != nil {
		return nil, false, err
	}

	execution = mutableSessionExecution(session, execution.ID)
	if execution == nil {
		return nil, false, errors.New("conversation execution disappeared while consuming continuation input")
	}
	execution.Continuation.ConsumedRevision = staged.Revision
	execution.Continuation.Text = ""
	execution.Continuation.Attachments = nil
	execution.Continuation.AppContext = nil
	execution.Continuation.Metadata = nil
	execution.Continuation.InvocationDefaults = nil
	execution.Continuation.InvocationBudget = nil
	execution.Continuation.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)
	return []ActivityID{activity.ID}, true, nil
}

func (r *Runtime) submitNormalMessageLocked(
	session *Session,
	execution *Execution,
	input MessageInput,
	carried []ToolResultActivity,
) (Activity, error) {
	return r.submitNormalMessageWithDefaultsLocked(
		session,
		execution,
		input,
		carried,
		nil,
	)
}

func (r *Runtime) submitNormalMessageWithDefaultsLocked(
	session *Session,
	execution *Execution,
	input MessageInput,
	carried []ToolResultActivity,
	resolvedDefaults *SessionDefaults,
) (Activity, error) {
	frame := normalConversationFrame(execution)
	if frame == nil || frame.Status.Terminal() {
		return Activity{}, errors.New("conversation frame is unavailable")
	}
	if resolvedDefaults == nil {
		value, err := resolveSessionInvocationDefaults(
			session.Defaults,
			input.Defaults,
		)
		if err != nil {
			return Activity{}, err
		}
		resolvedDefaults = &value
	}

	now := r.now()
	activity := Activity{
		ID:          ActivityID(r.newID()),
		Kind:        ActivityKindMessage,
		Visibility:  ActivityVisibilityBoth,
		CreatedAt:   now,
		ExecutionID: execution.ID,
		FrameID:     frame.ID,
		Message: &MessageActivity{
			Role:               MessageRoleUser,
			Status:             "accepted",
			Text:               input.Text,
			Attachments:        append([]Attachment(nil), input.Attachments...),
			ToolResults:        append([]ToolResultActivity(nil), carried...),
			Metadata:           input.Metadata.Clone(),
			InvocationDefaults: cloneInvocationDefaults(resolvedDefaults),
			InvocationBudget:   cloneBudget(input.InvocationBudget),
		},
	}
	session.Timeline = append(session.Timeline, activity)
	if err := r.advanceFrameLocked(
		session,
		execution,
		frame.ID,
		ProgramEvent{
			Kind:               ProgramEventActivity,
			ActivityID:         activity.ID,
			InvocationDefaults: cloneInvocationDefaults(resolvedDefaults),
			InvocationBudget:   cloneBudget(input.InvocationBudget),
		},
	); err != nil {
		return Activity{}, err
	}
	r.touchExecutionLocked(session, execution)
	return activity, nil
}

func (r *Runtime) completeNormalToolCall(
	ctx context.Context,
	meta CommandMeta,
	sessionID SessionID,
	executionID ExecutionID,
	workItemID WorkItemID,
	kind ToolOutputKind,
	message string,
) (*CommandReceipt, error) {
	return r.changeExecution(
		ctx,
		meta,
		sessionID,
		executionID,
		struct {
			Meta        CommandMeta
			SessionID   SessionID
			ExecutionID ExecutionID
			WorkItemID  WorkItemID
			Kind        ToolOutputKind
			Message     string
		}{
			Meta:        meta,
			SessionID:   sessionID,
			ExecutionID: executionID,
			WorkItemID:  workItemID,
			Kind:        kind,
			Message:     message,
		},
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			item := mutableWorkItem(execution, workItemID)
			if item == nil || item.Type != WorkItemTypeTool {
				return &CommandReceipt{
					CommandID:   meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "Tool WorkItem is unavailable",
				}, nil
			}
			switch item.State {
			case WorkItemStateReady,
				WorkItemStateFailed,
				WorkItemStateBlocked,
				WorkItemStateCompleted:
			default:
				return &CommandReceipt{
					CommandID:   meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "active Tool WorkItem cannot be completed manually",
				}, nil
			}
			if item.State == WorkItemStateCompleted &&
				item.Output != nil &&
				item.Output.Tool != nil &&
				item.Output.Tool.Kind == ToolOutputKindSuccess {
				return &CommandReceipt{
					CommandID:   meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     "successful Tool WorkItem cannot be overwritten",
				}, nil
			}

			now := r.now()
			effect := EffectStateConfirmedNotApplied
			if item.Effect.Kind == EffectKindRead {
				effect = EffectStateNotApplicable
			}
			value := MustCanonicalValue(struct {
				WorkItemID WorkItemID     `json:"workItemID"`
				Kind       ToolOutputKind `json:"kind"`
				Message    string         `json:"message"`
			}{
				WorkItemID: item.ID,
				Kind:       kind,
				Message:    message,
			})
			item.State = WorkItemStateCompleted
			item.EffectState = effect
			item.Output = &WorkItemOutput{
				Value: value,
				Tool: &ToolOutput{
					Kind:       kind,
					Text:       message,
					Structured: value.Clone(),
					Error:      map[ToolOutputKind]string{ToolOutputKindError: message}[kind],
				},
				Receipt: &ExternalReceipt{
					System:      "session-v3-manual-tool-result",
					Reference:   string(item.ID),
					ConfirmedAt: now,
					EffectState: effect,
				},
			}
			item.Error = nil
			item.WaitID = ""
			item.CompletedAt = &now
			item.UpdatedAt = now
			delete(r.privateBindings, item.ID)
			r.cancelRetryLocked(item.ID)
			r.cancelResourceWaitLocked(item.ID)
			released := r.releaseWorkResourcesLocked(execution, item.ID)
			r.wakeResourceWaitersLocked(execution, released)

			r.advanceSettledWorkLocked(
				session,
				execution,
				item.FrameID,
				item.ID,
			)
			r.wakeCompletionWorkWaitsLocked(session, execution, *item)
			r.refreshExecutionStateLocked(execution)
			r.settleTerminalExecutionLocked(session, execution)
			r.touchExecutionLocked(session, execution)
			return &CommandReceipt{
				CommandID:         meta.ID,
				Accepted:          true,
				SessionID:         session.ID,
				ExecutionID:       execution.ID,
				SessionRevision:   session.Revision,
				ExecutionRevision: execution.Revision,
				WorkItemIDs:       []WorkItemID{item.ID},
			}, nil
		},
	)
}

func (r *Runtime) requestAgentContinuationLocked(
	session *Session,
	execution *Execution,
) error {
	if session == nil || execution == nil {
		return errors.New("conversation execution is unavailable")
	}
	frames := make([]ProgramFrameID, 0)
	for _, frame := range execution.ProgramFrames {
		if frame.Status.Terminal() ||
			frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID {
			continue
		}
		frames = append(frames, frame.ID)
	}
	for _, frameID := range frames {
		if err := r.advanceFrameLocked(
			session,
			execution,
			frameID,
			ProgramEvent{Kind: ProgramEventContinue},
		); err != nil {
			return err
		}
	}
	return nil
}

func isNormalConversationExecution(execution Execution) bool {
	return normalConversationFrame(&execution) != nil
}

func normalConversationFrame(execution *Execution) *ProgramFrame {
	if execution == nil {
		return nil
	}
	for index := range execution.ProgramFrames {
		frame := &execution.ProgramFrames[index]
		if frame.ParentFrameID == "" &&
			frame.Program.InterpreterID == BuiltinConversationInterpreterID {
			return frame
		}
	}
	return nil
}

func manualToolRetryAllowed(item WorkItem) bool {
	if item.State == WorkItemStateUncertain ||
		item.EffectState == EffectStateUncertain {
		return false
	}
	if item.State == WorkItemStateCompleted &&
		(item.Output == nil ||
			item.Output.Tool == nil ||
			item.Output.Tool.Kind == ToolOutputKindSuccess) {
		return false
	}
	if item.State != WorkItemStateFailed &&
		item.State != WorkItemStateBlocked &&
		item.State != WorkItemStateCompleted {
		return false
	}
	if effectReplaySafe(item.Effect) {
		return true
	}
	return item.EffectState == EffectStateConfirmedNotApplied
}

func undispatchedEffectState(effect EffectContract) EffectState {
	if effect.Kind == EffectKindRead {
		return EffectStateNotApplicable
	}
	return EffectStateNotStarted
}

func latestNormalLLMWorkItemID(execution Execution) WorkItemID {
	for _, v := range slices.Backward(execution.WorkItems) {
		if v.Type == WorkItemTypeLLM {
			return v.ID
		}
	}
	return ""
}

func automationHoldApplies(
	hold AutomationHold,
	origin WorkItemID,
) bool {
	if !hold.Enabled {
		return false
	}
	if hold.Scope == AutomationHoldIndefinite {
		return true
	}
	return hold.TargetWorkItemID == "" ||
		hold.TargetWorkItemID == origin
}

func uniqueNormalWorkItemIDs(values []WorkItemID) []WorkItemID {
	seen := make(map[WorkItemID]struct{}, len(values))
	output := make([]WorkItemID, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	return output
}

func normalSessionEditBoundary(session Session) bool {
	for _, execution := range session.Executions {
		for _, item := range execution.WorkItems {
			if !item.State.Terminal() {
				return false
			}
		}
		for _, frame := range execution.ProgramFrames {
			if frame.ParentFrameID == "" &&
				frame.Program.InterpreterID ==
					BuiltinConversationInterpreterID {
				continue
			}
			if !frame.Status.Terminal() {
				return false
			}
		}
		if !execution.State.Terminal() &&
			!isNormalConversationExecution(execution) {
			return false
		}
	}
	return true
}

func carriedNormalToolResults(
	session Session,
	ids []WorkItemID,
) ([]ToolResultActivity, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	output := make([]ToolResultActivity, 0, len(ids))
	seen := make(map[WorkItemID]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			return nil, errors.New("carried Tool WorkItem identity is required")
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}

		var item *WorkItem
		for _, execution := range session.Executions {
			for index := range execution.WorkItems {
				if execution.WorkItems[index].ID == id {
					value := execution.WorkItems[index]
					item = &value
					break
				}
			}
			if item != nil {
				break
			}
		}
		if item == nil ||
			item.Type != WorkItemTypeTool ||
			item.State != WorkItemStateCompleted ||
			item.Output == nil ||
			item.Output.Tool == nil ||
			item.Input.Tool == nil {
			return nil, fmt.Errorf(
				"carried Tool WorkItem %q has no confirmed Tool result",
				id,
			)
		}

		output = append(output, ToolResultActivity{
			WorkItemID:     item.ID,
			CallID:         item.Input.Tool.Call.ID,
			ProviderCallID: item.Input.Tool.Call.ProviderCallID,
			ChoiceID:       item.Input.Tool.Call.ChoiceID,
			Name:           item.Input.Tool.Call.Name,
			Kind:           item.Output.Tool.Kind,
			Text:           item.Output.Tool.Text,
			Structured:     item.Output.Tool.Structured.Clone(),
			Error:          item.Output.Tool.Error,
		})
	}
	return output, nil
}
