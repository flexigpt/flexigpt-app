package main

import (
	"context"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/sessionstorev3"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const sessionV3RuntimeEventName = "session-v3-runtime:event"

type SessionV3Wrapper struct {
	mu      sync.RWMutex
	runtime *sessionv3.Runtime
	store   *sessionstorev3.Store
	desktop *application.App
	closed  bool
}

var _ sessionv3.EventPublisher = (*SessionV3Wrapper)(nil)

func InitSessionV3Wrapper(wrapper *SessionV3Wrapper, runtime *sessionv3.Runtime, store *sessionstorev3.Store) error {
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
	event sessionv3.RuntimeEvent,
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

func (w *SessionV3Wrapper) CreateSession(command sessionv3.CreateSessionCommand) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.CreateSession(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) SubmitMessage(command sessionv3.SubmitMessageCommand) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.SubmitMessage(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) StartWorkflow(command sessionv3.StartWorkflowCommand) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.StartWorkflow(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) SignalExecution(
	command sessionv3.SignalExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.SignalExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ExecuteWorkItems(
	command sessionv3.ExecuteWorkItemsCommand,
) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.ExecuteWorkItems(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ContinueExecution(
	command sessionv3.ContinueExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.ContinueExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) PauseExecution(
	command sessionv3.PauseExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.PauseExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ResumeExecution(
	command sessionv3.ResumeExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.ResumeExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) CancelExecution(
	command sessionv3.CancelExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.CancelExecution(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) ResolveWait(command sessionv3.ResolveWaitCommand) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.ResolveWait(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) RenameSession(command sessionv3.RenameSessionCommand) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.RenameSession(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) UpdateSessionDefaults(
	command sessionv3.UpdateSessionDefaultsCommand,
) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.UpdateSessionDefaults(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) DeleteSession(command sessionv3.DeleteSessionCommand) (*sessionv3.CommandReceipt, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (*sessionv3.CommandReceipt, error) {
		return r.DeleteSession(context.Background(), command)
	})
}

func (w *SessionV3Wrapper) Conversation(id sessionv3.SessionID) (sessionv3.ConversationProjection, error) {
	w.mu.RLock()
	store := w.store
	w.mu.RUnlock()
	if store == nil {
		return sessionv3.ConversationProjection{}, errSessionV3Closed
	}
	return store.ReadConversation(id)
}

func (w *SessionV3Wrapper) Activities(
	id sessionv3.SessionID,
) (sessionv3.ActivityProjection, error) {
	w.mu.RLock()
	store := w.store
	w.mu.RUnlock()
	if store == nil {
		return sessionv3.ActivityProjection{}, errSessionV3Closed
	}
	return store.ReadActivities(id)
}

func (w *SessionV3Wrapper) Workflow(
	id sessionv3.SessionID,
) (sessionv3.WorkflowProjection, error) {
	w.mu.RLock()
	store := w.store
	w.mu.RUnlock()
	if store == nil {
		return sessionv3.WorkflowProjection{}, errSessionV3Closed
	}
	return store.ReadWorkflow(id)
}

func (w *SessionV3Wrapper) Execution(
	sessionID sessionv3.SessionID,
	executionID sessionv3.ExecutionID,
) (sessionv3.Execution, error) {
	w.mu.RLock()
	store := w.store
	w.mu.RUnlock()
	if store == nil {
		return sessionv3.Execution{}, errSessionV3Closed
	}
	return store.ReadExecution(sessionID, executionID)
}

func (w *SessionV3Wrapper) Session(
	id sessionv3.SessionID,
) (sessionv3.Session, error) {
	return withSessionV3(w, func(r *sessionv3.Runtime) (sessionv3.Session, error) {
		return r.GetSession(context.Background(), id)
	})
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
