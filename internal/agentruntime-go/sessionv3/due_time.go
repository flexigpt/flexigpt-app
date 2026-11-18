package sessionv3

import "context"

// dueTimeKind identifies a Runtime-owned due-time lifecycle. The durable
// Session stores due-time facts. This private registry stores only cancellation
// functions for the corresponding process-local waits.
type dueTimeKind string

const (
	dueTimeKindWait              dueTimeKind = "wait"
	dueTimeKindRetry             dueTimeKind = "retry"
	dueTimeKindResource          dueTimeKind = "resource"
	dueTimeKindExecutionDeadline dueTimeKind = "executionDeadline"
	dueTimeKindCleanupDeadline   dueTimeKind = "cleanupDeadline"
	dueTimeKindTrigger           dueTimeKind = "trigger"
)

type dueTimeKey struct {
	kind     dueTimeKind
	identity string
}

func waitDueTimeKey(id WaitID) dueTimeKey {
	return dueTimeKey{
		kind:     dueTimeKindWait,
		identity: string(id),
	}
}

func retryDueTimeKey(id WorkItemID) dueTimeKey {
	return dueTimeKey{
		kind:     dueTimeKindRetry,
		identity: string(id),
	}
}

func resourceDueTimeKey(id WorkItemID) dueTimeKey {
	return dueTimeKey{
		kind:     dueTimeKindResource,
		identity: string(id),
	}
}

func executionDeadlineDueTimeKey(id ExecutionID) dueTimeKey {
	return dueTimeKey{
		kind:     dueTimeKindExecutionDeadline,
		identity: string(id),
	}
}

func cleanupDeadlineDueTimeKey(id ExecutionID) dueTimeKey {
	return dueTimeKey{
		kind:     dueTimeKindCleanupDeadline,
		identity: string(id),
	}
}

func triggerDueTimeKey(
	sessionID SessionID,
	triggerID TriggerID,
) dueTimeKey {
	return dueTimeKey{
		kind:     dueTimeKindTrigger,
		identity: string(sessionID) + "\x00" + string(triggerID),
	}
}

// armDueTimeLocked creates one process-private cancellation context for one
// durable due-time fact. Callers validate the persisted due-time fact again
// when their callback fires.
func (r *Runtime) armDueTimeLocked(
	key dueTimeKey,
) (context.Context, bool) {
	if r == nil || r.closed || r.lifecycleCtx == nil {
		return nil, false
	}
	if r.dueTimers == nil {
		r.dueTimers = make(map[dueTimeKey]context.CancelFunc)
	}
	if _, found := r.dueTimers[key]; found {
		return nil, false
	}

	timerCtx, cancel := context.WithCancel(r.lifecycleCtx)
	r.dueTimers[key] = cancel
	return timerCtx, true
}

func (r *Runtime) cancelDueTimeLocked(key dueTimeKey) {
	if r == nil || r.dueTimers == nil {
		return
	}
	cancel, found := r.dueTimers[key]
	if !found {
		return
	}
	delete(r.dueTimers, key)
	cancel()
}

// consumeDueTimeLocked removes the private timer lifecycle after its durable
// callback reaches its settlement path.
func (r *Runtime) consumeDueTimeLocked(key dueTimeKey) {
	r.cancelDueTimeLocked(key)
}

func (r *Runtime) takeAllDueTimesLocked() []context.CancelFunc {
	if r == nil || len(r.dueTimers) == 0 {
		return nil
	}

	output := make([]context.CancelFunc, 0, len(r.dueTimers))
	for _, cancel := range r.dueTimers {
		output = append(output, cancel)
	}
	clear(r.dueTimers)
	return output
}
