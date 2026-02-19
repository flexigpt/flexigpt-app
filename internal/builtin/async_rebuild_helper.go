package builtin

import (
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// This file provides a tiny utility that takes care of the run-exactly-once-in-the-background-when-stale pattern.
//
// Typical usage:
//
//  reb := builtin.NewAsyncRebuilder(
//      time.Hour,                            // max snapshot age
//      func() error { return rebuild() },    // the expensive work
//  )
//  ...
//  reb.Trigger() // cheap, may launch the goroutine if needed
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
	lastRun int64 // Unix-nanos of the successful run

	mu        sync.Mutex    // protects running, done and closed
	running   bool          // true when a rebuild goroutine is alive
	done      chan struct{} // closed when the current rebuild finishes
	closed    bool          // set by Close() to prevent further triggers
	closeOnce sync.Once     // ensures Close action runs only once
}

// NewAsyncRebuilder returns a ready-to-use AsyncRebuilder.
// If maxAge <= 0 the rebuilder will consider a snapshot stale immediately (i.e. every Trigger may start a rebuild).
func NewAsyncRebuilder(maxAge time.Duration, fn func() error) *AsyncRebuilder {
	if maxAge <= 0 {
		maxAge = alwaysStale
	}
	// When nothing is running, done should be a closed channel so IsDone() is ready.
	initDone := make(chan struct{})
	close(initDone)

	r := &AsyncRebuilder{
		maxAge: maxAge,
		fn:     fn,
		done:   initDone,
	}
	return r
}

func (r *AsyncRebuilder) IsDone() <-chan struct{} {
	r.mu.Lock()
	ch := r.done
	r.mu.Unlock()
	return ch
}

// Trigger starts a rebuild in the background when the stored snapshot is considered stale.
// The call itself is cheap (non-blocking).
func (r *AsyncRebuilder) Trigger() {
	// Fast-path: if the last successful run is still fresh, return quickly.
	last := atomic.LoadInt64(&r.lastRun)
	if last != 0 && time.Since(time.Unix(0, last)) <= r.maxAge {
		return // snapshot still fresh
	}

	r.mu.Lock()
	if r.closed || r.running {
		r.mu.Unlock()
		return // closed or already running
	}
	// Re-check freshness while holding the lock (read lastRun atomically).
	if time.Since(time.Unix(0, atomic.LoadInt64(&r.lastRun))) <= r.maxAge {
		r.mu.Unlock()
		return // snapshot fresh now
	}

	r.running = true
	done := make(chan struct{})
	r.done = done
	r.mu.Unlock()

	go func() {
		defer func() {
			rec := recover()

			// Reset state under lock, then close the done channel.
			r.mu.Lock()
			r.running = false
			close(done)
			r.mu.Unlock()

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

// MarkFresh updates the last successful-run timestamp.
func (r *AsyncRebuilder) MarkFresh() {
	atomic.StoreInt64(&r.lastRun, time.Now().UnixNano())
}

// Close prevents any new rebuilds from starting and waits for any running rebuild to finish.
// The Close action is performed only once (sync.Once); subsequent Close calls return immediately.
func (r *AsyncRebuilder) Close() {
	r.closeOnce.Do(func() {
		r.mu.Lock()
		r.closed = true
		ch := r.done
		running := r.running
		r.mu.Unlock()

		if running {
			<-ch
		}
	})
}
