package sessionv3

import (
	"errors"
)

const (
	budgetLabelMessages     = "messages"
	budgetLabelMessageBytes = "message bytes"
)

type budgetMetric uint8

const (
	budgetMetricStepExecutions budgetMetric = iota + 1
	budgetMetricCallAttempts
	budgetMetricModelTokens
	budgetMetricMessageCount
	budgetMetricMessageBytes
	budgetMetricEventCount
	budgetMetricEventBytes
)

type budgetMetricRequest struct {
	Metric    budgetMetric
	Increment int
	Label     string
}

type budgetCounter struct {
	Limit     int
	Current   *int
	Increment int
	Label     string
}

func budgetFrameChain(
	execution *Execution,
	frameID ProgramFrameID,
) []*ProgramFrame {
	if execution == nil || frameID == "" {
		return nil
	}

	output := make([]*ProgramFrame, 0)
	seen := make(map[ProgramFrameID]struct{})
	currentID := frameID
	for currentID != "" {
		if _, duplicate := seen[currentID]; duplicate {
			return nil
		}
		seen[currentID] = struct{}{}

		frame := mutableFrame(execution, currentID)
		if frame == nil {
			return output
		}
		output = append(output, frame)
		currentID = frame.ParentFrameID
	}
	return output
}

func budgetLimit(
	value ExecutionBudget,
	metric budgetMetric,
) int {
	switch metric {
	case budgetMetricStepExecutions:
		return value.StepExecutions
	case budgetMetricCallAttempts:
		return value.CallAttempts
	case budgetMetricModelTokens:
		return value.ModelTokens
	case budgetMetricMessageCount:
		return value.MessageCount
	case budgetMetricMessageBytes:
		return value.MessageBytes
	case budgetMetricEventCount:
		return value.EventCount
	case budgetMetricEventBytes:
		return value.EventBytes
	default:
		return 0
	}
}

func budgetUsagePointer(
	value *ExecutionBudgetUsage,
	metric budgetMetric,
) *int {
	if value == nil {
		return nil
	}
	switch metric {
	case budgetMetricStepExecutions:
		return &value.StepExecutions
	case budgetMetricCallAttempts:
		return &value.CallAttempts
	case budgetMetricModelTokens:
		return &value.ModelTokens
	case budgetMetricMessageCount:
		return &value.MessageCount
	case budgetMetricMessageBytes:
		return &value.MessageBytes
	case budgetMetricEventCount:
		return &value.EventCount
	case budgetMetricEventBytes:
		return &value.EventBytes
	default:
		return nil
	}
}

func mutableBudgetScope(
	execution *Execution,
	id string,
) *ExecutionBudgetScope {
	if execution == nil || id == "" {
		return nil
	}
	for index := range execution.BudgetScopes {
		if execution.BudgetScopes[index].ID == id {
			return &execution.BudgetScopes[index]
		}
	}
	return nil
}

// reserveScopedBudgetLocked applies each metric atomically to the shared
// Execution budget, every enclosing ProgramFrame budget, and an optional
// WorkItem budget. Attached frames therefore share execution aggregate usage
// while retaining their own narrowed counters.
func (r *Runtime) reserveScopedBudgetLocked(
	execution *Execution,
	item *WorkItem,
	frameID ProgramFrameID,
	requests ...budgetMetricRequest,
) error {
	if execution == nil {
		return errors.New("execution budget owner is unavailable")
	}

	var scope *ExecutionBudgetScope
	if item != nil && item.BudgetScopeID != "" {
		scope = mutableBudgetScope(execution, item.BudgetScopeID)
	}

	counters := make([]budgetCounter, 0, len(requests)*4)
	frames := budgetFrameChain(execution, frameID)
	for _, request := range requests {
		if request.Increment < 0 {
			return errors.New("execution budget increment cannot be negative")
		}
		if request.Increment == 0 {
			continue
		}

		counters = append(counters, budgetCounter{
			Limit:     budgetLimit(execution.Budget, request.Metric),
			Current:   budgetUsagePointer(&execution.Usage, request.Metric),
			Increment: request.Increment,
			Label:     request.Label,
		})
		for _, frame := range frames {
			if frame.Budget == nil {
				continue
			}
			counters = append(counters, budgetCounter{
				Limit:     budgetLimit(*frame.Budget, request.Metric),
				Current:   budgetUsagePointer(&frame.Usage, request.Metric),
				Increment: request.Increment,
				Label:     request.Label,
			})
		}
		if scope != nil {
			counters = append(counters, budgetCounter{
				Limit:     budgetLimit(scope.Budget, request.Metric),
				Current:   budgetUsagePointer(&scope.Usage, request.Metric),
				Increment: request.Increment,
				Label:     request.Label,
			})
		}
		if item != nil && item.Budget != nil {
			counters = append(counters, budgetCounter{
				Limit:     budgetLimit(*item.Budget, request.Metric),
				Current:   budgetUsagePointer(&item.Usage, request.Metric),
				Increment: request.Increment,
				Label:     request.Label,
			})
		}
	}

	for _, counter := range counters {
		if counter.Current == nil {
			return errors.New("execution budget metric is unavailable")
		}
		if err := checkBudgetCount(
			counter.Limit,
			*counter.Current,
			counter.Increment,
			counter.Label,
		); err != nil {
			return err
		}
	}
	for _, counter := range counters {
		*counter.Current += counter.Increment
	}
	return nil
}

func (r *Runtime) reserveMessageBudgetForWorkLocked(
	execution *Execution,
	item *WorkItem,
	bytes int,
) error {
	frameID := ProgramFrameID("")
	if item != nil {
		frameID = item.FrameID
	}
	return r.reserveScopedBudgetLocked(
		execution,
		item,
		frameID,
		budgetMetricRequest{
			Metric:    budgetMetricMessageCount,
			Increment: 1,
			Label:     budgetLabelMessages,
		},
		budgetMetricRequest{
			Metric:    budgetMetricMessageBytes,
			Increment: bytes,
			Label:     budgetLabelMessageBytes,
		},
	)
}

func (r *Runtime) reserveMessageBudgetForOwnerLocked(
	execution *Execution,
	workItemID WorkItemID,
	frameID ProgramFrameID,
	bytes int,
) error {
	var item *WorkItem
	if workItemID != "" {
		item = mutableWorkItem(execution, workItemID)
	}
	if item != nil {
		frameID = item.FrameID
	}
	return r.reserveScopedBudgetLocked(
		execution,
		item,
		frameID,
		budgetMetricRequest{
			Metric:    budgetMetricMessageCount,
			Increment: 1,
			Label:     budgetLabelMessages,
		},
		budgetMetricRequest{
			Metric:    budgetMetricMessageBytes,
			Increment: bytes,
			Label:     budgetLabelMessageBytes,
		},
	)
}

func (r *Runtime) reserveEventBudgetForWorkLocked(
	execution *Execution,
	item *WorkItem,
	bytes int,
) error {
	frameID := ProgramFrameID("")
	if item != nil {
		frameID = item.FrameID
	}
	return r.reserveScopedBudgetLocked(
		execution,
		item,
		frameID,
		budgetMetricRequest{
			Metric:    budgetMetricEventCount,
			Increment: 1,
			Label:     "events",
		},
		budgetMetricRequest{
			Metric:    budgetMetricEventBytes,
			Increment: bytes,
			Label:     "event bytes",
		},
	)
}

func (r *Runtime) reserveEventBudgetForOwnerLocked(
	execution *Execution,
	workItemID WorkItemID,
	frameID ProgramFrameID,
	bytes int,
) error {
	var item *WorkItem
	if workItemID != "" {
		item = mutableWorkItem(execution, workItemID)
	}
	if item != nil {
		frameID = item.FrameID
	}
	return r.reserveScopedBudgetLocked(
		execution,
		item,
		frameID,
		budgetMetricRequest{
			Metric:    budgetMetricEventCount,
			Increment: 1,
			Label:     "events",
		},
		budgetMetricRequest{
			Metric:    budgetMetricEventBytes,
			Increment: bytes,
			Label:     "event bytes",
		},
	)
}
