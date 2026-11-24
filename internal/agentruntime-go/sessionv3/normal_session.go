package sessionv3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	manualToolResultReceiptSystem      = "session-v3-manual-tool-result"
	defaultSkippedToolMessage          = "the tool call was intentionally skipped by the user."
	defaultErrorToolMessage            = "the tool call failed without a usable result."
	errContinuationNoOwner             = "continuation input has no active conversation owner"
	errContinuationRevisionMismatchFmt = "continuation input revision mismatch: expected %d, current %d"
	errTerminalToolControl             = "terminal or cancelling executions cannot mutate tool calls"
	errRecoveryPendingToolControl      = "recovery review must be resolved before mutating tool calls"
	errActiveLLMToolControl            = "tool calls cannot change while a model operation is active"
	errNoReadyToolWorkItems            = "no ready tool work items are available"
	errRequestedToolNotReady           = "requested tool work item is not ready"
	errToolWorkItemUnavailable         = "tool work item is unavailable"
	errProviderToolManualMutation      = "provider-managed tool calls cannot be changed manually"
	errToolResultAlreadyConsumed       = "tool work item result was already consumed by a later model operation"
	errToolOutsideCurrentBatch         = "tool work item does not belong to the current unconsumed tool batch"
	errToolRetryDisallowed             = "tool work item cannot retry without confirmed non-application"
	errActiveToolManualCompletion      = "active tool work item cannot be completed manually"
	errSuccessfulToolOverwrite         = "successful tool work item cannot be overwritten"
	errTerminalAutomationControl       = "terminal or cancelling executions cannot change automation"
	errAutomationTargetRequired        = "automation hold must target tools, continuation, or both"
	errAutomationScopeUnavailable      = "automation hold scope is unavailable"
	errEditSafeBoundaryRequired        = "editing requires a safe terminal conversation boundary"
	errReplaceUserMessageOnly          = "only a user message activity can be replaced"
	errEditInputRequired               = "edit-and-resend requires text, attachments, or carried tool results"
	errEditExecutionUnavailable        = "edit-and-resend execution is unavailable"
	errConversationExecDisappeared     = "conversation execution disappeared during edit-and-resend"
	errContinuationExecDisappeared     = "conversation execution disappeared while consuming continuation input"
	errConversationFrameUnavailable    = "conversation frame is unavailable"
	errAdvanceFrameExecDisappeared     = "conversation execution disappeared while advancing its message frame"
	errConversationExecutionMissing    = "conversation execution is unavailable"
	errCarriedToolIDRequired           = "carried tool work item identity is required"
	errCarriedToolUnconfirmedFmt       = "carried tool work item %q has no confirmed tool result"
)

func (r *Runtime) UpdateContinuationInput(
	ctx context.Context,
	command UpdateContinuationInputCommand,
) (*CommandReceipt, error) {
	if command.Meta.ID == "" {
		return nil, errors.New(errCommandIDRequired)
	}
	if err := command.Input.Validate(); err != nil {
		return &CommandReceipt{
			CommandID:   command.Meta.ID,
			SessionID:   command.SessionID,
			ExecutionID: command.ExecutionID,
			Message:     err.Error(),
		}, nil
	}
	if receipt, found, lookupErr := r.FindCommandReceipt(
		ctx,
		command.SessionID,
		command.Meta,
		command,
	); lookupErr != nil {
		return nil, lookupErr
	} else if found {
		return receipt, nil
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

	identity := command
	r.mu.RLock()
	if !r.closed {
		if session := r.sessions[command.SessionID]; session != nil {
			command.ExecutionID = r.resolveConversationControlExecutionIDLocked(
				session,
				command.ExecutionID,
			)
		}
	}
	r.mu.RUnlock()

	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		identity,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			active := r.findConversationOwnerExecutionLocked(session)
			if active == nil || active.ID != execution.ID {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     errContinuationNoOwner,
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
						errContinuationRevisionMismatchFmt,
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
	identity := command
	r.mu.RLock()
	if !r.closed {
		if session := r.sessions[command.SessionID]; session != nil {
			command.ExecutionID = r.resolveConversationControlExecutionIDLocked(
				session,
				command.ExecutionID,
			)
		}
	}
	r.mu.RUnlock()

	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		identity,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if message := validateManualToolControlExecution(*execution); message != "" {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     message,
				}, nil
			}

			ids := uniqueNormalWorkItemIDs(command.WorkItemIDs)
			if len(ids) == 0 {
				ownerFrame := activeConversationAgentFrame(execution)
				if ownerFrame == nil {
					ownerFrame = liveConversationAgentFrame(session, *execution)
				}
				if ownerFrame != nil && frameHasActiveLLMWork(*execution, ownerFrame.ID) {
					return &CommandReceipt{
						CommandID:   command.Meta.ID,
						SessionID:   session.ID,
						ExecutionID: execution.ID,
						Message:     errActiveLLMToolControl,
					}, nil
				}
				ids = readyCurrentBatchToolIDs(*execution, ownerFrame)
			}
			if len(ids) == 0 {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     errNoReadyToolWorkItems,
				}, nil
			}

			for _, id := range ids {
				item := mutableWorkItem(execution, id)
				if item == nil || item.Type != WorkItemTypeTool {
					return &CommandReceipt{
						CommandID:   command.Meta.ID,
						SessionID:   session.ID,
						ExecutionID: execution.ID,
						Message:     errToolWorkItemUnavailable,
					}, nil
				}
				if message := validateManualToolBatchItem(*execution, *item); message != "" {
					return &CommandReceipt{
						CommandID:   command.Meta.ID,
						SessionID:   session.ID,
						ExecutionID: execution.ID,
						Message:     message,
					}, nil
				}
				if item.State != WorkItemStateReady {
					return &CommandReceipt{
						CommandID:   command.Meta.ID,
						SessionID:   session.ID,
						ExecutionID: execution.ID,
						Message:     errRequestedToolNotReady,
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
	identity := command
	r.mu.RLock()
	if !r.closed {
		if session := r.sessions[command.SessionID]; session != nil {
			command.ExecutionID = r.resolveConversationControlExecutionIDLocked(
				session,
				command.ExecutionID,
			)
		}
	}
	r.mu.RUnlock()

	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		identity,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if message := validateManualToolControlExecution(*execution); message != "" {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     message,
				}, nil
			}
			item := mutableWorkItem(execution, command.WorkItemID)
			if item == nil || item.Type != WorkItemTypeTool {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     errToolWorkItemUnavailable,
				}, nil
			}
			if message := validateManualToolBatchItem(*execution, *item); message != "" {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     message,
				}, nil
			}
			if !manualToolRetryAllowed(*item) {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     errToolRetryDisallowed,
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
		defaultSkippedToolMessage,
	)
}

func (r *Runtime) CompleteToolCallAsError(
	ctx context.Context,
	command CompleteToolCallAsErrorCommand,
) (*CommandReceipt, error) {
	message := strings.TrimSpace(command.Error)
	if message == "" {
		message = defaultErrorToolMessage
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
	identity := command
	r.mu.RLock()
	if !r.closed {
		if session := r.sessions[command.SessionID]; session != nil {
			command.ExecutionID = r.resolveConversationControlExecutionIDLocked(
				session,
				command.ExecutionID,
			)
		}
	}
	r.mu.RUnlock()

	return r.changeExecution(
		ctx,
		command.Meta,
		command.SessionID,
		command.ExecutionID,
		identity,
		func(session *Session, execution *Execution) (*CommandReceipt, error) {
			if execution.State.Terminal() ||
				execution.Controls.CancellationRequested {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     errTerminalAutomationControl,
				}, nil
			}
			if !command.Tools && !command.Continuation {
				return &CommandReceipt{
					CommandID:   command.Meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     errAutomationTargetRequired,
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
					Message:     errAutomationScopeUnavailable,
				}, nil
			}

			ownerFrame := activeConversationAgentFrame(execution)
			if ownerFrame == nil {
				ownerFrame = liveConversationAgentFrame(session, *execution)
			}
			hold := AutomationHold{
				Enabled:          command.Enabled,
				Scope:            scope,
				TargetWorkItemID: latestScopedLLMWorkItemID(*execution, ownerFrame),
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
			if !command.Enabled && command.Continuation {
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
		return nil, errors.New(errCommandIDRequired)
	}
	if err := command.Input.Validate(); err != nil {
		return &CommandReceipt{
			CommandID: command.Meta.ID,
			SessionID: command.SessionID,
			Message:   err.Error(),
		}, nil
	}
	if receipt, found, lookupErr := r.FindCommandReceipt(
		ctx,
		command.SessionID,
		command.Meta,
		command,
	); lookupErr != nil {
		return nil, lookupErr
	} else if found {
		return receipt, nil
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
			errEditSafeBoundaryRequired,
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
			errReplaceUserMessageOnly,
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
			errEditInputRequired,
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
				errEditExecutionUnavailable,
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
		return nil, errors.New(errConversationExecDisappeared)
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
	if err := r.persistLocked(ctx, checkpointReasonMessageEditedAndResent, session); err != nil {
		*session = previous
		r.mu.Unlock()
		return nil, err
	}
	r.publishLocked(ctx, session, current.ID, RuntimeEventExecutionChanged)
	r.mu.Unlock()
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
	appContext, appSources := stagedConversationAppContextLocked(
		session,
		*execution,
	)
	frame := activeConversationAgentFrame(execution)
	if frame == nil {
		frame = liveConversationAgentFrame(session, *execution)
	}
	if frame == nil {
		return nil, false, errors.New(errConversationFrameUnavailable)
	}
	var (
		activity Activity
		err      error
	)

	activity, err = r.submitNormalMessageToFrameWithDefaultsLocked(
		session,
		execution,
		frame,
		input,
		nil,
		staged.InvocationDefaults,
		appContext,
	)
	if err != nil {
		return nil, false, err
	}

	execution = mutableSessionExecution(session, execution.ID)
	if execution == nil {
		return nil, false, errors.New(errContinuationExecDisappeared)
	}
	execution.Continuation.ConsumedRevision = staged.Revision
	execution.Continuation.Text = ""
	execution.Continuation.Attachments = nil
	execution.Continuation.AppContext = nil
	execution.Continuation.Metadata = nil
	execution.Continuation.InvocationDefaults = nil
	execution.Continuation.InvocationBudget = nil
	execution.Continuation.UpdatedAt = r.now()
	r.clearConsumedAppContextLocked(session, execution.ID, appSources)
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
		nil,
	)
}

func (r *Runtime) submitNormalMessageWithDefaultsLocked(
	session *Session,
	execution *Execution,
	input MessageInput,
	carried []ToolResultActivity,
	resolvedDefaults *SessionDefaults,
	appContext []AppContextUpdate,
) (Activity, error) {
	frame := normalConversationFrame(execution)
	if frame == nil {
		frame = liveConversationAgentFrame(session, *execution)
	}
	if frame == nil || frame.Status.Terminal() {
		return Activity{}, errors.New(errConversationFrameUnavailable)
	}
	return r.submitNormalMessageToFrameWithDefaultsLocked(
		session,
		execution,
		frame,
		input,
		carried,
		resolvedDefaults,
		appContext,
	)
}

func (r *Runtime) submitNormalMessageToFrameWithDefaultsLocked(
	session *Session,
	execution *Execution,
	frame *ProgramFrame,
	input MessageInput,
	carried []ToolResultActivity,
	resolvedDefaults *SessionDefaults,
	appContext []AppContextUpdate,
) (Activity, error) {
	if session == nil || execution == nil || frame == nil ||
		frame.Status.Terminal() {
		return Activity{}, errors.New(errConversationFrameUnavailable)
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

	resolvedValue := cloneSessionDefaults(*resolvedDefaults)
	applySessionConversationOwnerDefaults(
		session,
		&resolvedValue,
	)
	resolvedDefaults = &resolvedValue

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
			Status:             MessageStatusAccepted,
			Text:               input.Text,
			Attachments:        append([]Attachment(nil), input.Attachments...),
			ToolResults:        append([]ToolResultActivity(nil), carried...),
			AppContext:         cloneAppContextUpdates(appContext),
			Metadata:           input.Metadata.Clone(),
			InvocationDefaults: cloneInvocationDefaults(resolvedDefaults),
			InvocationBudget:   cloneBudget(input.InvocationBudget),
		},
	}
	if err := r.reserveActivityBudgetLocked(execution, activity); err != nil {
		return Activity{}, err
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
		session.Timeline = session.Timeline[:len(session.Timeline)-1]
		return Activity{}, err
	}
	if current := mutableSessionExecution(session, execution.ID); current != nil {
		execution = current
	} else {
		return Activity{}, errors.New(errAdvanceFrameExecDisappeared)
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
	resolvedExecutionID := executionID
	r.mu.RLock()
	if !r.closed {
		if session := r.sessions[sessionID]; session != nil {
			resolvedExecutionID = r.resolveConversationControlExecutionIDLocked(
				session,
				executionID,
			)
		}
	}
	r.mu.RUnlock()

	return r.changeExecution(
		ctx,
		meta,
		sessionID,
		resolvedExecutionID,
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
			if controlMessage := validateManualToolControlExecution(*execution); controlMessage != "" {
				return &CommandReceipt{
					CommandID:   meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     controlMessage,
				}, nil
			}
			item := mutableWorkItem(execution, workItemID)
			if item == nil || item.Type != WorkItemTypeTool {
				return &CommandReceipt{
					CommandID:   meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     errToolWorkItemUnavailable,
				}, nil
			}
			if batchMessage := validateManualToolBatchItem(*execution, *item); batchMessage != "" {
				return &CommandReceipt{
					CommandID:   meta.ID,
					SessionID:   session.ID,
					ExecutionID: execution.ID,
					Message:     batchMessage,
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
					Message:     errActiveToolManualCompletion,
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
					Message:     errSuccessfulToolOverwrite,
				}, nil
			}

			now := r.now()
			effect := EffectStateConfirmedNotApplied
			if item.Effect.Kind == EffectKindRead {
				effect = EffectStateNotApplicable
			}
			value := jsonutil.MustCanonicalValue(struct {
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
					System:      manualToolResultReceiptSystem,
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
			execution = mutableSessionExecution(session, execution.ID)
			if execution == nil {
				return nil, ErrExecutionNotFound
			}
			item = mutableWorkItem(execution, workItemID)
			if item == nil {
				return nil, ErrExecutionNotFound
			}
			r.wakeCompletionWorkWaitsLocked(session, execution, *item)
			execution, err := r.finalizeExecutionSettlementLocked(
				ctx,
				session,
				execution.ID,
				nil,
			)
			if err != nil || execution == nil {
				return nil, ErrExecutionNotFound
			}
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
		return errors.New(errConversationExecutionMissing)
	}

	frame := activeConversationAgentFrame(execution)
	if frame == nil {
		frame = liveConversationAgentFrame(session, *execution)
	}
	if frame == nil {
		return nil
	}
	return r.advanceFrameLocked(
		session,
		execution,
		frame.ID,
		ProgramEvent{Kind: ProgramEventContinue},
	)
}

func (r *Runtime) findConversationOwnerExecutionLocked(
	session *Session,
) *Execution {
	if session == nil {
		return nil
	}
	for _, ownership := range session.ConversationOwnership {
		if ownership.ConversationID != string(session.ID) ||
			ownership.ActiveHandoffID == "" ||
			ownership.Owner.ExecutionID == "" ||
			ownership.Owner.FrameID == "" {
			continue
		}
		execution := mutableSessionExecution(
			session,
			ownership.Owner.ExecutionID,
		)
		if execution == nil || execution.State.Terminal() {
			continue
		}
		frame := mutableFrame(execution, ownership.Owner.FrameID)
		if frame == nil ||
			frame.Status.Terminal() ||
			frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID {
			continue
		}
		return execution
	}
	for index := range session.Executions {
		execution := &session.Executions[index]
		if !execution.State.Terminal() && isNormalConversationExecution(*execution) {
			return execution
		}
	}
	return nil
}

func (r *Runtime) resolveConversationControlExecutionIDLocked(
	session *Session,
	requested ExecutionID,
) ExecutionID {
	if session == nil {
		return requested
	}
	owner := r.findConversationOwnerExecutionLocked(session)
	if owner == nil {
		return requested
	}
	if requested == "" {
		return owner.ID
	}
	target := mutableSessionExecution(session, requested)
	if target != nil && isNormalConversationExecution(*target) {
		return owner.ID
	}
	return requested
}

func hasPendingRecoveryReviewWait(execution Execution) bool {
	for _, wait := range execution.Waits {
		if wait.Request.Kind != WaitKindRecovery {
			continue
		}
		if wait.Status == WaitStatusPending ||
			wait.Status == WaitStatusResolving {
			return true
		}
	}
	return false
}

func frameHasActiveLLMWork(
	execution Execution,
	frameID ProgramFrameID,
) bool {
	for _, item := range execution.WorkItems {
		if item.FrameID != frameID || item.Type != WorkItemTypeLLM {
			continue
		}
		if item.State == WorkItemStateBinding ||
			item.State == WorkItemStateDispatched {
			return true
		}
	}
	return false
}

func validateManualToolControlExecution(execution Execution) string {
	if execution.State.Terminal() ||
		execution.Controls.CancellationRequested {
		return errTerminalToolControl
	}
	if hasPendingRecoveryReviewWait(execution) {
		return errRecoveryPendingToolControl
	}
	return ""
}

func validateManualToolBatchItem(
	execution Execution,
	item WorkItem,
) string {
	if item.Type != WorkItemTypeTool || item.Input.Tool == nil {
		return errToolWorkItemUnavailable
	}
	if item.Input.Tool.Call.Binding.Kind == ToolBindingKindProvider {
		return errProviderToolManualMutation
	}
	if item.ConsumedByWorkItemID != "" {
		return errToolResultAlreadyConsumed
	}
	frame := mutableFrame(&execution, item.FrameID)
	if frame == nil ||
		frame.Status.Terminal() ||
		frame.CancellationRequested ||
		frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID {
		return errToolOutsideCurrentBatch
	}
	if frameHasActiveLLMWork(execution, frame.ID) {
		return errActiveLLMToolControl
	}
	var state agentLoopProgramState
	if err := json.Unmarshal(frame.State.Data, &state); err != nil {
		return errToolOutsideCurrentBatch
	}
	if len(state.PendingToolIDs) == 0 ||
		!containsWorkItemID(state.PendingToolIDs, item.ID) {
		return errToolOutsideCurrentBatch
	}
	if state.LastLLMWorkItemID != "" &&
		item.OriginWorkItemID != state.LastLLMWorkItemID {
		return errToolOutsideCurrentBatch
	}
	return ""
}

func readyCurrentBatchToolIDs(
	execution Execution,
	ownerFrame *ProgramFrame,
) []WorkItemID {
	output := make([]WorkItemID, 0)
	collectFromFrame := func(frame ProgramFrame) {
		if frame.Status.Terminal() ||
			frame.CancellationRequested ||
			frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID ||
			frameHasActiveLLMWork(execution, frame.ID) {
			return
		}
		var state agentLoopProgramState
		if err := json.Unmarshal(frame.State.Data, &state); err != nil {
			return
		}
		for _, id := range state.PendingToolIDs {
			item := executionWorkItem(execution, id)
			if item == nil ||
				item.Type != WorkItemTypeTool ||
				item.State != WorkItemStateReady ||
				item.ConsumedByWorkItemID != "" {
				continue
			}
			if state.LastLLMWorkItemID != "" &&
				item.OriginWorkItemID != state.LastLLMWorkItemID {
				continue
			}
			output = append(output, item.ID)
		}
	}
	if ownerFrame != nil {
		collectFromFrame(*ownerFrame)
		return uniqueNormalWorkItemIDs(output)
	}
	for _, frame := range execution.ProgramFrames {
		collectFromFrame(frame)
	}
	return uniqueNormalWorkItemIDs(output)
}

// normalToolBatchDispatchAllowed keeps automatic Agent-loop Tool batches
// sequential. Explicitly requested work uses the same durable batch order,
// but only selected Tool WorkItems participate.
func normalToolBatchDispatchAllowed(
	execution Execution,
	item WorkItem,
) bool {
	if item.Type != WorkItemTypeTool {
		return true
	}
	frame := mutableFrame(&execution, item.FrameID)
	if frame == nil ||
		frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID {
		return true
	}
	var state agentLoopProgramState
	if json.Unmarshal(frame.State.Data, &state) != nil ||
		len(state.PendingToolIDs) == 0 {
		return true
	}

	requested := false
	for _, id := range state.PendingToolIDs {
		candidate := executionWorkItem(execution, id)
		if candidate != nil && candidate.Requested && !candidate.State.Terminal() {
			requested = true
			break
		}
	}
	for _, id := range state.PendingToolIDs {
		candidate := executionWorkItem(execution, id)
		if candidate == nil || candidate.State.Terminal() {
			if !requested &&
				candidate != nil &&
				(candidate.Output == nil ||
					candidate.Output.Tool == nil ||
					candidate.Output.Tool.Kind != ToolOutputKindSuccess) {
				return false
			}
			continue
		}
		if requested {
			if !candidate.Requested {
				continue
			}
			return candidate.ID == item.ID
		}
		if !candidate.Automatic ||
			automationHoldApplies(
				execution.Controls.AutomaticTools,
				candidate.OriginWorkItemID,
			) {
			return false
		}
		return candidate.ID == item.ID
	}
	return false
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

func liveConversationAgentFrame(
	session *Session,
	execution Execution,
) *ProgramFrame {
	if session == nil {
		return nil
	}
	for _, ownership := range session.ConversationOwnership {
		if ownership.ConversationID != string(session.ID) ||
			ownership.ActiveHandoffID == "" ||
			ownership.Owner.ExecutionID != execution.ID ||
			ownership.Owner.FrameID == "" {
			continue
		}
		frame := mutableFrame(&execution, ownership.Owner.FrameID)
		if frame == nil ||
			frame.Status.Terminal() ||
			frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID {
			return nil
		}
		return frame
	}
	return nil
}

func activeConversationAgentFrame(
	execution *Execution,
) *ProgramFrame {
	root := normalConversationFrame(execution)
	if root == nil || root.Status.Terminal() {
		return nil
	}
	state, err := jsonutil.DecodeCanonicalValue[conversationProgramState](
		root.State.Data,
	)
	if err != nil || state.ActiveFrameID == "" {
		return nil
	}
	frame := mutableFrame(execution, state.ActiveFrameID)
	if frame == nil ||
		frame.Status.Terminal() ||
		frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID {
		return nil
	}
	return frame
}

func (r *Runtime) conversationMessageExecutionLocked(
	session *Session,
) (*Execution, error) {
	if session == nil {
		return nil, ErrSessionNotFound
	}
	for _, ownership := range session.ConversationOwnership {
		if ownership.ConversationID != string(session.ID) ||
			ownership.ActiveHandoffID == "" ||
			ownership.Owner.ExecutionID == "" ||
			ownership.Owner.FrameID == "" {
			continue
		}
		execution := mutableSessionExecution(
			session,
			ownership.Owner.ExecutionID,
		)
		if execution == nil || execution.State.Terminal() {
			continue
		}
		frame := mutableFrame(execution, ownership.Owner.FrameID)
		if frame == nil ||
			frame.Status.Terminal() ||
			frame.Program.InterpreterID != BuiltinAgentLoopInterpreterID {
			continue
		}
		return execution, nil
	}
	return r.conversationExecutionLocked(session)
}

func applySessionConversationOwnerDefaults(
	session *Session,
	defaults *SessionDefaults,
) {
	if session == nil || defaults == nil {
		return
	}
	for _, ownership := range session.ConversationOwnership {
		if ownership.ConversationID != string(session.ID) ||
			ownership.Owner.Agent == nil {
			continue
		}
		agent := ownership.Owner.Agent
		defaults.Agent = &ArtifactSelection{
			RootID:           agent.RootID,
			ArtifactID:       agent.ArtifactID,
			ExpectedRevision: agent.Revision,
			ExpectedDigest:   agent.Digest,
		}
		return
	}
}

func manualToolRetryAllowed(item WorkItem) bool {
	if item.State == WorkItemStateUncertain ||
		item.EffectState == EffectStateUncertain {
		return false
	}
	if item.ConsumedByWorkItemID != "" {
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

type stagedAppContextSource struct {
	ExecutionID ExecutionID
	Revision    Revision
}

// stagedConversationAppContextLocked transfers App context only from the
// active conversation owner or from the durable previous owner of its active
// handoff. Other Executions do not implicitly contribute context to a normal
// conversation message.
func stagedConversationAppContextLocked(
	session *Session,
	owner Execution,
) ([]AppContextUpdate, []stagedAppContextSource) {
	if session == nil || owner.ID == "" {
		return nil, nil
	}
	values := make([]AppContextUpdate, 0)
	sources := make([]stagedAppContextSource, 0)
	seen := make(map[string]struct{})
	for _, execution := range session.Executions {
		if !appContextSourceRoutesToConversationOwner(
			session,
			owner,
			execution,
		) {
			continue
		}
		if execution.Continuation.Revision <= execution.Continuation.ConsumedRevision ||
			len(execution.Continuation.AppContext) == 0 {
			continue
		}
		copied := false
		for _, update := range execution.Continuation.AppContext {
			key := string(jsonutil.MustCanonicalValue(update))
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			values = append(values, cloneAppContextUpdate(update))
			copied = true
		}
		if copied {
			sources = append(sources, stagedAppContextSource{
				ExecutionID: execution.ID,
				Revision:    execution.Continuation.Revision,
			})
		}
	}
	return values, sources
}

func appContextSourceRoutesToConversationOwner(
	session *Session,
	owner Execution,
	source Execution,
) bool {
	if session == nil || owner.ID == "" || source.ID == "" {
		return false
	}
	if owner.ID == source.ID {
		return true
	}
	for _, ownership := range session.ConversationOwnership {
		if ownership.ConversationID != string(session.ID) ||
			ownership.ActiveHandoffID == "" ||
			ownership.Owner.ExecutionID != owner.ID ||
			ownership.PreviousOwner == nil ||
			ownership.PreviousOwner.ExecutionID != source.ID {
			continue
		}
		return true
	}
	return false
}

func (r *Runtime) clearConsumedAppContextLocked(
	session *Session,
	owner ExecutionID,
	sources []stagedAppContextSource,
) {
	if r == nil || session == nil {
		return
	}
	now := r.now()
	for _, source := range sources {
		execution := mutableSessionExecution(session, source.ExecutionID)
		if execution == nil ||
			execution.Continuation.Revision != source.Revision {
			continue
		}
		execution.Continuation.AppContext = nil
		execution.Continuation.UpdatedAt = now
		if execution.ID != owner &&
			strings.TrimSpace(execution.Continuation.Text) == "" &&
			len(execution.Continuation.Attachments) == 0 {
			execution.Continuation.ConsumedRevision = execution.Continuation.Revision
		}
		if execution.ID != owner {
			r.touchExecutionLocked(session, execution)
		}
	}
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

func latestScopedLLMWorkItemID(
	execution Execution,
	ownerFrame *ProgramFrame,
) WorkItemID {
	if ownerFrame != nil {
		for _, item := range slices.Backward(execution.WorkItems) {
			if item.FrameID == ownerFrame.ID && item.Type == WorkItemTypeLLM {
				return item.ID
			}
		}
	}
	return latestNormalLLMWorkItemID(execution)
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
			return nil, errors.New(errCarriedToolIDRequired)
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
				errCarriedToolUnconfirmedFmt,
				id,
			)
		}

		output = append(output, ToolResultActivity{
			WorkItemID:     item.ID,
			CallID:         item.Input.Tool.Call.ID,
			ProviderCallID: item.Input.Tool.Call.ProviderCallID,
			ChoiceID:       item.Input.Tool.Call.ChoiceID,
			Name:           item.Input.Tool.Call.Name,
			Type:           item.Input.Tool.Call.Type,
			Kind:           item.Output.Tool.Kind,
			Text:           item.Output.Tool.Text,
			Contents:       jsonutil.CloneJSONInValue(item.Output.Tool.Contents),
			Structured:     item.Output.Tool.Structured.Clone(),
			Error:          item.Output.Tool.Error,
		})
	}
	return output, nil
}
