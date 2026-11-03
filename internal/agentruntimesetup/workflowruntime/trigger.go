package workflowruntime

import (
	"context"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// TriggerManager is backend-only trigger ownership. It never creates another
// persistence root. Trigger occurrences submit ordinary StartWorkflow or
// SignalExecution commands through the common command service.
type TriggerManager struct {
	mu      sync.Mutex
	closed  bool
	cancels map[string]context.CancelFunc
	start   func(context.Context, sessionv3.StartWorkflowCommand) (*sessionv3.CommandReceipt, error)
	signal  func(context.Context, sessionv3.SignalExecutionCommand) (*sessionv3.CommandReceipt, error)
}

func NewTriggerManager(
	start func(context.Context, sessionv3.StartWorkflowCommand) (*sessionv3.CommandReceipt, error),
	signal func(context.Context, sessionv3.SignalExecutionCommand) (*sessionv3.CommandReceipt, error),
) *TriggerManager {
	return &TriggerManager{cancels: map[string]context.CancelFunc{}, start: start, signal: signal}
}

func (m *TriggerManager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.closed = true
	for _, cancel := range m.cancels {
		cancel()
	}
	clear(m.cancels)
	m.mu.Unlock()
}

func (m *TriggerManager) Schedule(key string, at time.Time, command sessionv3.StartWorkflowCommand) {
	if m == nil || key == "" {
		return
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	if cancel := m.cancels[key]; cancel != nil {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancels[key] = cancel
	m.mu.Unlock()
	go func() {
		delay := time.Until(at)
		if delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
		m.mu.Lock()
		delete(m.cancels, key)
		closed := m.closed
		m.mu.Unlock()
		if !closed && m.start != nil {
			_, _ = m.start(context.WithoutCancel(ctx), command)
		}
	}()
}

func (m *TriggerManager) Signal(
	ctx context.Context,
	command sessionv3.SignalExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	if m == nil || m.signal == nil {
		return nil, context.Canceled
	}
	return m.signal(ctx, command)
}
