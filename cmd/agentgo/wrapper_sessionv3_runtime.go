package main

import (
	"context"
	"sync"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/sessionstorev3"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const sessionV3RuntimeEventName = "session-v3-runtime:event"

type SessionV3Wrapper struct {
	mu      sync.RWMutex
	runtime *sessionv3Runtime.Runtime
	store   *sessionstorev3.Store
	desktop *application.App
	closed  bool
}

func InitSessionV3Wrapper(
	wrapper *SessionV3Wrapper,
	runtime *sessionv3Runtime.Runtime,
	store *sessionstorev3.Store,
) error {
	if wrapper == nil || runtime == nil || store == nil {
		return errSessionV3Deps
	}
	wrapper.mu.Lock()
	defer wrapper.mu.Unlock()
	if wrapper.closed || wrapper.runtime != nil {
		return errSessionV3AlreadyInit
	}
	wrapper.runtime = runtime
	wrapper.store = store
	return nil
}

func (w *SessionV3Wrapper) PublishRuntimeEvent(
	_ context.Context,
	event sessionv3Runtime.RuntimeEvent,
) error {
	if w == nil {
		return nil
	}

	w.mu.RLock()
	desktop := w.desktop
	closed := w.closed
	w.mu.RUnlock()
	if closed || desktop == nil {
		return nil
	}

	desktop.Event.Emit(sessionV3RuntimeEventName, event)
	return nil
}

func (w *SessionV3Wrapper) CreateSession(
	command sessionv3Runtime.CreateSessionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.CreateSession(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) SubmitMessage(
	command sessionv3Runtime.SubmitMessageCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.SubmitMessage(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) StartWorkflow(
	command sessionv3Runtime.StartWorkflowCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.StartWorkflow(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) RegisterWorkflowTriggers(
	command sessionv3Runtime.RegisterWorkflowTriggersCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.RegisterWorkflowTriggers(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) SetTriggerRegistrationState(
	command sessionv3Runtime.SetTriggerRegistrationStateCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.SetTriggerRegistrationState(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) AcceptTriggerEvent(
	command sessionv3Runtime.AcceptTriggerEventCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(
		w,
		func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
			return r.AcceptTriggerEvent(context.Background(), command)
		},
	)
}

func (w *SessionV3Wrapper) SignalExecution(
	command sessionv3Runtime.SignalExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.SignalExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ExecuteWorkItems(
	command sessionv3Runtime.ExecuteWorkItemsCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.ExecuteWorkItems(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ContinueExecution(
	command sessionv3Runtime.ContinueExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.ContinueExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) PauseExecution(
	command sessionv3Runtime.PauseExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.PauseExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ResumeExecution(
	command sessionv3Runtime.ResumeExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.ResumeExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) CancelExecution(
	command sessionv3Runtime.CancelExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.CancelExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ResolveWait(
	command sessionv3Runtime.ResolveWaitCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.ResolveWait(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) RenameSession(
	command sessionv3Runtime.RenameSessionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.RenameSession(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) UpdateSessionDefaults(
	command sessionv3Runtime.UpdateSessionDefaultsCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.UpdateSessionDefaults(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) PrepareUserMessageSkillTemplate(
	request sessionv3Runtime.UserMessageSkillTemplatePreparationRequest,
) (sessionv3Runtime.UserMessageSkillTemplateResult, error) {
	return withSessionV3(
		w,
		func(r *sessionv3Runtime.Runtime) (
			sessionv3Runtime.UserMessageSkillTemplateResult,
			error,
		) {
			return r.PrepareUserMessageSkillTemplate(context.Background(), request)
		},
	)
}

func (w *SessionV3Wrapper) PreviewInstructionSkill(
	request sessionv3Runtime.InstructionSkillPreviewRequest,
) (sessionv3Runtime.InstructionSkillPreviewResult, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (sessionv3Runtime.InstructionSkillPreviewResult, error) {
		return r.PreviewInstructionSkill(context.Background(), request)
	})
}

func (w *SessionV3Wrapper) UpdateContinuationInput(
	command sessionv3Runtime.UpdateContinuationInputCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.UpdateContinuationInput(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ExecuteToolCalls(
	command sessionv3Runtime.ExecuteToolCallsCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.ExecuteToolCalls(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) RetryToolCall(
	command sessionv3Runtime.RetryToolCallCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.RetryToolCall(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) SkipToolCall(
	command sessionv3Runtime.SkipToolCallCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.SkipToolCall(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) CompleteToolCallAsError(
	command sessionv3Runtime.CompleteToolCallAsErrorCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.CompleteToolCallAsError(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) SetAutomationHold(
	command sessionv3Runtime.SetAutomationHoldCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.SetAutomationHold(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) OpenAppView(
	command sessionv3Runtime.OpenAppViewCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.OpenAppView(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ExecuteAppTool(
	command sessionv3Runtime.ExecuteAppToolCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.ExecuteAppTool(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) StageAppContext(
	command sessionv3Runtime.StageAppContextCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.StageAppContext(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ReadAppResource(
	request sessionv3Runtime.AppResourceReadRequest,
) (sessionv3Runtime.AppResourceReadResult, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (sessionv3Runtime.AppResourceReadResult, error) {
		return r.ReadAppResource(context.Background(), request)
	})
}

func (w *SessionV3Wrapper) EditAndResend(
	command sessionv3Runtime.EditAndResendCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.EditAndResend(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) DeleteSession(
	command sessionv3Runtime.DeleteSessionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (*sessionv3Runtime.CommandReceipt, error) {
		return r.DeleteSession(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) Conversation(
	id sessionv3Runtime.SessionID,
) (sessionv3Runtime.ConversationProjection, error) {
	store, err := w.projectionStore()
	if err != nil {
		return sessionv3Runtime.ConversationProjection{}, err
	}
	return store.ReadConversation(id)
}

func (w *SessionV3Wrapper) Activities(
	id sessionv3Runtime.SessionID,
) (sessionv3Runtime.ActivityProjection, error) {
	store, err := w.projectionStore()
	if err != nil {
		return sessionv3Runtime.ActivityProjection{}, err
	}
	return store.ReadActivities(id)
}

func (w *SessionV3Wrapper) SessionRuntimeProjection(
	id sessionv3Runtime.SessionID,
) (sessionv3Runtime.SessionRuntimeProjection, error) {
	store, err := w.projectionStore()
	if err != nil {
		return sessionv3Runtime.SessionRuntimeProjection{}, err
	}
	return store.ReadSessionRuntimeProjection(id)
}

func (w *SessionV3Wrapper) ModelOperations(
	id sessionv3Runtime.SessionID,
) ([]sessionv3Runtime.ModelOperationProjection, error) {
	projection, err := w.SessionRuntimeProjection(id)
	if err != nil {
		return nil, err
	}
	return projection.ModelOperations, nil
}

func (w *SessionV3Wrapper) ToolCalls(
	id sessionv3Runtime.SessionID,
) ([]sessionv3Runtime.ToolCallProjection, error) {
	projection, err := w.SessionRuntimeProjection(id)
	if err != nil {
		return nil, err
	}
	return projection.ToolCalls, nil
}

func (w *SessionV3Wrapper) ToolAttempts(
	id sessionv3Runtime.SessionID,
) ([]sessionv3Runtime.ToolAttemptProjection, error) {
	projection, err := w.SessionRuntimeProjection(id)
	if err != nil {
		return nil, err
	}
	return projection.Attempts, nil
}

func (w *SessionV3Wrapper) Waits(
	id sessionv3Runtime.SessionID,
) ([]sessionv3Runtime.WaitProjection, error) {
	projection, err := w.SessionRuntimeProjection(id)
	if err != nil {
		return nil, err
	}
	return projection.Waits, nil
}

func (w *SessionV3Wrapper) Apps(
	id sessionv3Runtime.SessionID,
) ([]sessionv3Runtime.AppInstance, error) {
	projection, err := w.SessionRuntimeProjection(id)
	if err != nil {
		return nil, err
	}
	return projection.Apps, nil
}

func (w *SessionV3Wrapper) Usage(
	id sessionv3Runtime.SessionID,
) ([]sessionv3Runtime.ExecutionUsageProjection, error) {
	projection, err := w.SessionRuntimeProjection(id)
	if err != nil {
		return nil, err
	}
	return projection.Usage, nil
}

func (w *SessionV3Wrapper) Workflow(
	id sessionv3Runtime.SessionID,
) (sessionv3Runtime.WorkflowProjection, error) {
	store, err := w.projectionStore()
	if err != nil {
		return sessionv3Runtime.WorkflowProjection{}, err
	}
	return store.ReadWorkflow(id)
}

func (w *SessionV3Wrapper) Execution(
	sessionID sessionv3Runtime.SessionID,
	executionID sessionv3Runtime.ExecutionID,
) (sessionv3Runtime.Execution, error) {
	store, err := w.projectionStore()
	if err != nil {
		return sessionv3Runtime.Execution{}, err
	}
	return store.ReadExecution(sessionID, executionID)
}

func (w *SessionV3Wrapper) ListSessions(
	request sessionstorev3.ListSessionsRequest,
) (sessionstorev3.SessionPage, error) {
	store, err := w.projectionStore()
	if err != nil {
		return sessionstorev3.SessionPage{}, err
	}
	return store.ListSessions(request)
}

func (w *SessionV3Wrapper) SearchSessions(
	request sessionstorev3.SearchSessionsRequest,
) (sessionstorev3.SessionPage, error) {
	store, err := w.projectionStore()
	if err != nil {
		return sessionstorev3.SessionPage{}, err
	}
	return store.SearchSessions(context.Background(), request)
}

func (w *SessionV3Wrapper) ListExecutions(
	request sessionstorev3.ListExecutionsRequest,
) ([]sessionstorev3.ExecutionListItem, error) {
	store, err := w.projectionStore()
	if err != nil {
		return nil, err
	}
	return store.ListExecutions(request)
}

func (w *SessionV3Wrapper) Session(
	id sessionv3Runtime.SessionID,
) (sessionv3Runtime.Session, error) {
	return withSessionV3(w, func(r *sessionv3Runtime.Runtime) (sessionv3Runtime.Session, error) {
		return r.GetSession(context.Background(), id)
	})
}

func (w *SessionV3Wrapper) projectionStore() (*sessionstorev3.Store, error) {
	if w == nil {
		return nil, errSessionV3Closed
	}
	w.mu.RLock()
	store := w.store
	closed := w.closed
	w.mu.RUnlock()
	if closed || store == nil {
		return nil, errSessionV3Closed
	}
	return store, nil
}

func (w *SessionV3Wrapper) setApplication(desktop *application.App) {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.desktop = desktop
	w.mu.Unlock()
}

func (w *SessionV3Wrapper) close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	runtime := w.runtime
	store := w.store
	w.runtime = nil
	w.store = nil
	w.desktop = nil
	w.closed = true
	w.mu.Unlock()

	if runtime != nil {
		runtime.Close()
	}
	if store != nil {
		_ = store.Close()
	}
}
