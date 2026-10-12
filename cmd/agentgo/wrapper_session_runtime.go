package main

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

const sessionRuntimeEventName = "session-runtime:event"

type SessionRuntimeWrapper struct {
	runtime          *sessionRuntime.Runtime
	eventUnsubscribe func()
	eventDone        chan struct{}
}

func InitSessionRuntimeWrapper(
	wrapper *SessionRuntimeWrapper,
	runtimeValue *sessionRuntime.Runtime,
) error {
	if wrapper == nil || runtimeValue == nil {
		return errors.New("session runtime wrapper dependencies are incomplete")
	}

	if wrapper.runtime != nil {
		return errors.New("session runtime wrapper is already initialized")
	}

	wrapper.runtime = runtimeValue
	return nil
}

func (w *SessionRuntimeWrapper) CreateSession(
	command sessionRuntime.CreateSessionCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.CreateSession(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) UpdateSessionConfiguration(
	command sessionRuntime.UpdateSessionConfigurationCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.UpdateSessionConfiguration(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) ApplyAgent(
	command sessionRuntime.ApplyAgentCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.ApplyAgent(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) StartRun(
	command sessionRuntime.StartRunCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.StartRun(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) UpdateContinuationInput(
	command sessionRuntime.UpdateContinuationInputCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.UpdateContinuationInput(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) ContinueRun(
	command sessionRuntime.ContinueRunCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.ContinueRun(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) ExecuteCalls(
	command sessionRuntime.ExecuteCallsCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.ExecuteCalls(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) RetryCall(
	command sessionRuntime.RetryCallCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.RetryCall(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) SkipCall(
	command sessionRuntime.SkipCallCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.SkipCall(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) CompleteCallAsError(
	command sessionRuntime.CompleteCallAsErrorCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.CompleteCallAsError(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) ResolveWait(
	command sessionRuntime.ResolveWaitCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.ResolveWait(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) SetAutomationHold(
	command sessionRuntime.SetAutomationHoldCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.SetAutomationHold(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) PauseRun(
	command sessionRuntime.PauseRunCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.PauseRun(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) ResumeRun(
	command sessionRuntime.ResumeRunCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.ResumeRun(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) CancelRun(
	command sessionRuntime.CancelRunCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.CancelRun(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) EditAndResend(
	command sessionRuntime.EditAndResendCommand,
) (*sessionRuntime.CommandReceipt, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (*sessionRuntime.CommandReceipt, error) {
		return runtimeValue.EditAndResend(context.Background(), command)
	})
}

func (w *SessionRuntimeWrapper) GetSessionSnapshot(
	sessionID sessionRuntime.SessionID,
) (sessionRuntime.SessionSnapshot, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (sessionRuntime.SessionSnapshot, error) {
		return runtimeValue.GetSessionSnapshot(context.Background(), sessionID)
	})
}

func (w *SessionRuntimeWrapper) GetRunSnapshot(
	runID sessionRuntime.RunID,
) (sessionRuntime.RunSnapshot, error) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) (sessionRuntime.RunSnapshot, error) {
		return runtimeValue.GetRunSnapshot(context.Background(), runID)
	})
}

func (w *SessionRuntimeWrapper) ListRuntimeSessions() (
	[]sessionRuntime.SessionListItem,
	error,
) {
	return withSessionRuntime(w, func(runtimeValue *sessionRuntime.Runtime) ([]sessionRuntime.SessionListItem, error) {
		return runtimeValue.ListSessions(context.Background())
	})
}

func withSessionRuntime[T any](
	wrapper *SessionRuntimeWrapper,
	callback func(*sessionRuntime.Runtime) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if wrapper == nil {
			return zero, sessionRuntime.ErrRuntimeClosed
		}

		return callback(wrapper.runtime)
	})
}

// setAppContext binds the core subscription to the Wails transport after the
// application context is available. Runtime snapshots remain authoritative:
// the event is only a notification to refresh attached Session/Run views.
func (w *SessionRuntimeWrapper) setAppContext(ctx context.Context) {
	if w == nil || ctx == nil {
		return
	}

	previousUnsubscribe := w.eventUnsubscribe
	previousDone := w.eventDone
	w.eventUnsubscribe = nil
	w.eventDone = nil
	runtimeValue := w.runtime

	if previousUnsubscribe != nil {
		previousUnsubscribe()
	}
	if previousDone != nil {
		<-previousDone
	}
	if runtimeValue == nil {
		return
	}

	events, unsubscribe := runtimeValue.Subscribe(256)
	done := make(chan struct{})

	if w.runtime != runtimeValue {
		unsubscribe()
		return
	}
	w.eventUnsubscribe = unsubscribe
	w.eventDone = done

	go func() {
		defer close(done)
		for event := range events {
			runtime.EventsEmit(ctx, sessionRuntimeEventName, event)
		}
	}()
}

func (w *SessionRuntimeWrapper) close() {
	if w == nil {
		return
	}

	runtimeValue := w.runtime
	unsubscribe := w.eventUnsubscribe
	done := w.eventDone
	w.runtime = nil
	w.eventUnsubscribe = nil
	w.eventDone = nil

	if unsubscribe != nil {
		unsubscribe()
	}
	if done != nil {
		<-done
	}
	if runtimeValue != nil {
		runtimeValue.Close()
	}
}
