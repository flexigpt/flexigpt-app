package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
)

// WithRecovery is a helper that recovers from any panic, logs the stack trace,
// and returns an error to the caller.
func WithRecovery(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// Log the panic plus stack trace.
			slog.Error("panic recovered",
				slog.Any("panic", r),
				slog.String("stacktrace", string(debug.Stack())),
			)

			if e, ok := r.(error); ok {
				err = fmt.Errorf("panic recovered: %w", e)
			} else {
				err = fmt.Errorf("panic recovered: %v", r)
			}
		}
	}()

	err = fn()
	if err != nil {
		msg := err.Error()
		slog.Error("response", "error", msg)
		if !isStackTraceSkippable(msg) {
			slog.Error(string(debug.Stack()))
		}
	}
	return err
}

// WithRecoveryResp is a helper that recovers from any panic, logs the stack trace,
// and returns an error to the caller. T must match the response type of your function.
func WithRecoveryResp[T any](fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Log the panic plus stack trace.
			slog.Error("panic recovered",
				slog.Any("panic", r),
				slog.String("stacktrace", string(debug.Stack())),
			)

			// Ensure returned values reflect failure.
			var zero T
			result = zero

			if e, ok := r.(error); ok {
				err = fmt.Errorf("panic recovered: %w", e)
			} else {
				err = fmt.Errorf("panic recovered: %v", r)
			}
		}
	}()

	result, err = fn()
	if err != nil {
		msg := err.Error()
		slog.Error("response", "error", msg)
		if !isStackTraceSkippable(msg) {
			slog.Error(string(debug.Stack()))
		}
	}
	return result, err
}

var stackTraceSkippableErrs = []string{
	"context canceled",
	"context cancelled",
	"context deadline exceeded",
	"request canceled",
	"operation was canceled",
	"operation aborted",
}

func isStackTraceSkippable(msg string) bool {
	msg = strings.ToLower(msg)
	for _, errStr := range stackTraceSkippableErrs {
		if strings.Contains(msg, errStr) {
			return true
		}
	}
	return false
}
