package main

import (
	"context"
	"errors"
	"log/slog"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/sessionstore"
)

type SessionStoreWrapper struct {
	store *sessionstore.Store
}

func InitSessionStoreWrapper(
	wrapper *SessionStoreWrapper,
	baseDir string,
) error {
	if wrapper == nil {
		return errors.New("session store wrapper is required")
	}
	if wrapper.store != nil {
		return errors.New("session store wrapper is already initialized")
	}

	store, err := sessionstore.New(
		baseDir,
		sessionstore.WithFTS(true),
	)
	if err != nil {
		return err
	}

	wrapper.store = store
	return nil
}

func (w *SessionStoreWrapper) ReadSession(
	sessionID sessionRuntime.SessionID,
	forceFetch bool,
) (sessionRuntime.Session, error) {
	return withSessionStore(
		w,
		func(store *sessionstore.Store) (sessionRuntime.Session, error) {
			return store.ReadSession(
				context.Background(),
				sessionID,
				forceFetch,
			)
		},
	)
}

func (w *SessionStoreWrapper) ReadSessionConversation(
	sessionID sessionRuntime.SessionID,
	forceFetch bool,
) (sessionstore.ConversationProjection, error) {
	return withSessionStore(
		w,
		func(store *sessionstore.Store) (
			sessionstore.ConversationProjection,
			error,
		) {
			return store.ReadConversation(
				context.Background(),
				sessionID,
				forceFetch,
			)
		},
	)
}

func (w *SessionStoreWrapper) ListSessions(
	request sessionstore.ListSessionsRequest,
) (sessionstore.SessionPage, error) {
	return withSessionStore(
		w,
		func(store *sessionstore.Store) (sessionstore.SessionPage, error) {
			return store.ListSessions(context.Background(), request)
		},
	)
}

func (w *SessionStoreWrapper) SearchSessions(
	request sessionstore.SearchSessionsRequest,
) (sessionstore.SessionPage, error) {
	return withSessionStore(
		w,
		func(store *sessionstore.Store) (sessionstore.SessionPage, error) {
			return store.SearchSessions(context.Background(), request)
		},
	)
}

func (w *SessionStoreWrapper) ReadSessionRun(
	sessionID sessionRuntime.SessionID,
	runID sessionRuntime.RunID,
	forceFetch bool,
) (sessionRuntime.RunSnapshot, error) {
	return withSessionStore(
		w,
		func(store *sessionstore.Store) (
			sessionRuntime.RunSnapshot,
			error,
		) {
			return store.ReadRun(
				context.Background(),
				sessionID,
				runID,
				forceFetch,
			)
		},
	)
}

func (w *SessionStoreWrapper) ListSessionRuns(
	request sessionstore.ListSessionRunsRequest,
) ([]sessionstore.RunListItem, error) {
	return withSessionStore(
		w,
		func(store *sessionstore.Store) ([]sessionstore.RunListItem, error) {
			return store.ListSessionRuns(context.Background(), request)
		},
	)
}

// ExportSession returns the Session-store-owned semantic export. It is a read
// operation only; frontend callers cannot mutate transcript or execution data
// through this wrapper.
func (w *SessionStoreWrapper) ExportSession(
	sessionID sessionRuntime.SessionID,
) (string, error) {
	return withSessionStore(
		w,
		func(store *sessionstore.Store) (string, error) {
			raw, err := store.ExportSession(
				context.Background(),
				sessionID,
			)
			if err != nil {
				return "", err
			}
			return string(raw), nil
		},
	)
}

func withSessionStore[T any](
	wrapper *SessionStoreWrapper,
	callback func(*sessionstore.Store) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		if wrapper == nil || wrapper.store == nil {
			var zero T
			return zero, sessionstore.ErrStoreClosed
		}
		return callback(wrapper.store)
	})
}

func (w *SessionStoreWrapper) close() {
	if w == nil || w.store == nil {
		return
	}

	if err := w.store.Close(); err != nil {
		slog.Error("close session store", "error", err)
	}
	w.store = nil
}
