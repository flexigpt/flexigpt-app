// Package backgroundutil provides the common panic boundary for application-owned
// background work.
package backgroundutil

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

const PanicFailureCode = "runtime_panic"

// PanicFailureError identifies an application-owned operation that panicked.
//
// Panic values are deliberately not retained or formatted. External provider,
// MCP, and domain implementations can panic with values containing credentials
// or request content.
//
// Stack information is process-local diagnostic material. Callers must use
// Error(), rather than Stack, in public or durable failure records.
type PanicFailureError struct {
	Operation string `json:"-"`
	Stack     []byte `json:"-"`
}

func (f PanicFailureError) Error() string {
	return "runtime operation panicked: " + f.Operation
}

// RunSafely executes work synchronously behind the common panic boundary.
//
// The failure handler runs before RunSafely returns. A failure handler is itself
// protected by the same boundary, without recursively invoking that handler.
func RunSafely(
	operation string,
	work func(),
	onPanic func(PanicFailureError),
) {
	defer recoverFailure(operation, onPanic)
	work()
}

// Launch starts application-owned background work behind the common panic
// boundary.
//
// When group is supplied, completion includes both work and its panic handler.
// The owner must serialize admission against shutdown before calling Launch;
// Launch does not own the lifecycle of the supplied WaitGroup.
func Launch(
	group *sync.WaitGroup,
	operation string,
	work func(),
	onPanic func(PanicFailureError),
) {
	if group != nil {
		group.Add(1)
	}

	go func() {
		if group != nil {
			defer group.Done()
		}
		RunSafely(operation, work, onPanic)
	}()
}

// AfterFunc is the common panic-safe boundary for timer-owned background work.
func AfterFunc(
	delay time.Duration,
	operation string,
	work func(),
	onPanic func(PanicFailureError),
) *time.Timer {
	return time.AfterFunc(delay, func() {
		RunSafely(operation, work, onPanic)
	})
}

// AfterContext is the common panic-safe boundary for context-owned callbacks.
func AfterContext(
	ctx context.Context,
	operation string,
	work func(),
	onPanic func(PanicFailureError),
) func() bool {
	return context.AfterFunc(ctx, func() {
		RunSafely(operation, work, onPanic)
	})
}

func recoverFailure(
	operation string,
	onPanic func(PanicFailureError),
) {
	if recovered := recover(); recovered != nil {
		failure := PanicFailureError{
			Operation: operation,
			Stack:     debug.Stack(),
		}

		slog.Error(
			"runtime panic recovered",
			"code", PanicFailureCode,
			"operation", failure.Operation,
			"stack", string(failure.Stack),
		)

		if onPanic != nil {
			RunSafely(
				operation+" panic handler",
				func() {
					onPanic(failure)
				},
				nil,
			)
		}
	}
}
