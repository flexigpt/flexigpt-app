package builtin

import (
	"log/slog"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// This file provides a tiny utility that takes care of the run-exactly-once-in-the-background-when-stale pattern.
//
// Typical usage:
//
//	reb := builtin.NewAsyncRebuilder(
//	    time.Hour,                            // max snapshot age
//	    func() error { return rebuild() },    // the expensive work
//	)
//	...
//	reb.Trigger() // cheap, may launch the goroutine if needed
//
// The code guarantees that
//
//   - at most one rebuild goroutine is alive at any time,
//   - subsequent Trigger calls are ignored while one is running,
//   - a rebuild is started only when the previous successful run is
//     older than maxAge,
//   - panics inside the rebuild function are caught and logged.

const alwaysStale = time.Duration(1)

// AsyncRebuilder calls fn in a background goroutine when Trigger detects
// that the previous successful run is older than maxAge.
type AsyncRebuilder struct {
	maxAge  time.Duration
	fn      func() error
	lastRun int64         // Unix-nanos of the successful run
	running int32         // 0/1 - guarded with CAS
	done    chan struct{} // closed when the current rebuild finishes
}

// NewAsyncRebuilder returns a ready-to-use AsyncRebuilder.
// If maxAge <= 0 the rebuilder will consider a snapshot stale immediately (i.e. every Trigger may start a rebuild).
func NewAsyncRebuilder(maxAge time.Duration, fn func() error) *AsyncRebuilder {
	if maxAge <= 0 {
		maxAge = alwaysStale
	}
	r := &AsyncRebuilder{
		maxAge: maxAge,
		fn:     fn,
		done:   make(chan struct{}),
	}
	return r
}

func (r *AsyncRebuilder) IsDone() <-chan struct{} { return r.done }

// Trigger starts a rebuild in the background when the stored snapshot is considered stale.
// The call itself is cheap (non-blocking).
func (r *AsyncRebuilder) Trigger() {
	if time.Since(time.Unix(0, atomic.LoadInt64(&r.lastRun))) <= r.maxAge {
		return // snapshot still fresh
	}
	if !atomic.CompareAndSwapInt32(&r.running, 0, 1) {
		return // another rebuild already running
	}

	done := make(chan struct{})
	r.done = done

	go func() {
		defer func() {
			rec := recover()

			// Always reset state first; logging can be slow.
			atomic.StoreInt32(&r.running, 0)
			close(done)

			if rec != nil {
				slog.Error("panic in async rebuild",
					"err", rec,
					"stack", debug.Stack())
			}
		}()

		if err := r.fn(); err != nil {
			slog.Error("async rebuild failed", "error", err)
			return
		}

		r.MarkFresh()
	}()
}

// Force executes fn synchronously and updates the timestamp on success.
// It is exported mainly for unit tests.
func (r *AsyncRebuilder) Force() error {
	if err := r.fn(); err != nil {
		return err
	}
	r.MarkFresh()
	return nil
}

func (r *AsyncRebuilder) MarkFresh() {
	atomic.StoreInt64(&r.lastRun, time.Now().UnixNano())
}
