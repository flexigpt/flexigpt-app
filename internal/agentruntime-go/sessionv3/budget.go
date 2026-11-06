package sessionv3

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	budgetErrorCode                 = "budget_limit_exceeded"
	budgetDeadlineErrorCode         = "execution_deadline_elapsed"
	budgetCostUnavailableErrorCode  = "cost_accounting_unavailable"
	budgetCleanupExpiredErrorCode   = "cleanup_deadline_elapsed"
	executionDeadlineTimerOperation = "session-v3 execution deadline"
	cleanupDeadlineTimerOperation   = "session-v3 cleanup deadline"
)

type ExecutionBudgetUsage struct {
	StepExecutions int     `json:"stepExecutions"`
	CallAttempts   int     `json:"callAttempts"`
	ModelTokens    int     `json:"modelTokens"`
	MessageCount   int     `json:"messageCount"`
	MessageBytes   int     `json:"messageBytes"`
	EventCount     int     `json:"eventCount"`
	EventBytes     int     `json:"eventBytes"`
	Cost           float64 `json:"cost"`
}

type CleanupState struct {
	Active    bool            `json:"active"`
	StartedAt *time.Time      `json:"startedAt,omitempty"`
	Deadline  *time.Time      `json:"deadline,omitempty"`
	Expired   bool            `json:"expired,omitempty"`
	Primary   *SanitizedError `json:"primary,omitempty"`
}

func (u ExecutionBudgetUsage) Validate() error {
	for _, value := range []int{
		u.StepExecutions,
		u.CallAttempts,
		u.ModelTokens,
		u.MessageCount,
		u.MessageBytes,
		u.EventCount,
		u.EventBytes,
	} {
		if value < 0 {
			return errors.New("execution budget usage cannot be negative")
		}
	}
	if u.Cost < 0 {
		return errors.New("execution budget cost usage cannot be negative")
	}
	return nil
}

func (c CleanupState) Validate() error {
	if !c.Active {
		if c.StartedAt != nil || c.Deadline != nil ||
			c.Expired || c.Primary != nil {
			return errors.New("inactive cleanup state has durable facts")
		}
		return nil
	}
	if c.StartedAt == nil || c.StartedAt.IsZero() {
		return errors.New("active cleanup state has no start time")
	}
	if c.Deadline != nil && c.Deadline.Before(*c.StartedAt) {
		return errors.New("cleanup deadline precedes cleanup start")
	}
	if c.Primary != nil {
		if strings.TrimSpace(c.Primary.Code) == "" ||
			strings.TrimSpace(c.Primary.Message) == "" ||
			!c.Primary.Details.Valid() {
			return errors.New("cleanup primary error is invalid")
		}
	}
	return nil
}

func initialExecutionBudget(
	budget ExecutionBudget,
	now time.Time,
) (ExecutionBudget, *time.Time, error) {
	if err := budget.Validate(); err != nil {
		return ExecutionBudget{}, nil, err
	}
	var deadline *time.Time
	if budget.Duration != "" {
		value, err := budget.Duration.Deadline(now)
		if err != nil {
			return ExecutionBudget{}, nil, err
		}
		deadline = &value
	}
	return cloneExecutionBudget(budget), deadline, nil
}

func narrowExecutionBudget(
	parent ExecutionBudget,
	child ExecutionBudget,
) (ExecutionBudget, error) {
	if err := parent.Validate(); err != nil {
		return ExecutionBudget{}, err
	}
	if err := child.Validate(); err != nil {
		return ExecutionBudget{}, err
	}

	output := cloneExecutionBudget(parent)
	narrowInteger := func(
		parentValue int,
		childValue int,
	) (int, error) {
		switch {
		case childValue == 0:
			return parentValue, nil
		case parentValue == 0:
			return childValue, nil
		case childValue <= parentValue:
			return childValue, nil
		default:
			return 0, errors.New("child execution budget widens an inherited limit")
		}
	}
	var err error
	for _, pair := range []struct {
		parent *int
		child  int
	}{
		{&output.StepExecutions, child.StepExecutions},
		{&output.CallAttempts, child.CallAttempts},
		{&output.WorkflowDepth, child.WorkflowDepth},
		{&output.Concurrency, child.Concurrency},
		{&output.ModelTokens, child.ModelTokens},
		{&output.MessageCount, child.MessageCount},
		{&output.MessageBytes, child.MessageBytes},
		{&output.EventCount, child.EventCount},
		{&output.EventBytes, child.EventBytes},
	} {
		*pair.parent, err = narrowInteger(*pair.parent, pair.child)
		if err != nil {
			return ExecutionBudget{}, err
		}
	}

	narrowDuration := func(
		parentValue WorkflowDuration,
		childValue WorkflowDuration,
	) (WorkflowDuration, error) {
		if childValue == "" {
			return parentValue, nil
		}
		if parentValue == "" {
			return childValue, nil
		}
		start := time.Unix(0, 0).UTC()
		parentDeadline, parentErr := parentValue.Deadline(start)
		childDeadline, childErr := childValue.Deadline(start)
		if parentErr != nil {
			return "", parentErr
		}
		if childErr != nil {
			return "", childErr
		}
		if childDeadline.After(parentDeadline) {
			return "", errors.New(
				"child execution budget widens an inherited duration",
			)
		}
		return childValue, nil
	}

	output.Duration, err = narrowDuration(parent.Duration, child.Duration)
	if err != nil {
		return ExecutionBudget{}, err
	}
	output.CleanupDuration, err = narrowDuration(
		parent.CleanupDuration,
		child.CleanupDuration,
	)
	if err != nil {
		return ExecutionBudget{}, err
	}

	switch {
	case child.Cost == nil:
	case output.Cost == nil:
		cost := *child.Cost
		output.Cost = &cost
	case child.Cost.Currency != output.Cost.Currency ||
		child.Cost.Amount > output.Cost.Amount:
		return ExecutionBudget{}, errors.New(
			"child execution budget widens an inherited cost limit",
		)
	default:
		cost := *child.Cost
		output.Cost = &cost
	}
	return output, output.Validate()
}

func (r *Runtime) reserveStepExecutionLocked(
	execution *Execution,
	itemBudget *ExecutionBudget,
) error {
	if execution == nil {
		return errors.New("execution budget owner is unavailable")
	}
	if err := checkBudgetCount(
		execution.Budget.StepExecutions,
		execution.Usage.StepExecutions,
		1,
		"step executions",
	); err != nil {
		return err
	}
	if itemBudget != nil {
		if err := checkBudgetCount(
			itemBudget.StepExecutions,
			execution.Usage.StepExecutions,
			1,
			"step executions",
		); err != nil {
			return err
		}
	}
	execution.Usage.StepExecutions++
	return nil
}

func (r *Runtime) reserveCallAttemptLocked(
	execution *Execution,
	item WorkItem,
) error {
	if execution == nil || !workConsumesCallAttempt(item.Type) {
		return nil
	}
	if err := checkBudgetCount(
		execution.Budget.CallAttempts,
		execution.Usage.CallAttempts,
		1,
		"call attempts",
	); err != nil {
		return err
	}
	if item.Budget != nil {
		if err := checkBudgetCount(
			item.Budget.CallAttempts,
			execution.Usage.CallAttempts,
			1,
			"call attempts",
		); err != nil {
			return err
		}
	}
	if item.Type == WorkItemTypeLLM && execution.Budget.Cost != nil {
		return &SanitizedError{
			Code:     budgetCostUnavailableErrorCode,
			Category: ErrorCategoryLimit,
			Message:  "cost budgeting requires provider cost accounting",
		}
	}
	execution.Usage.CallAttempts++
	return nil
}

func workConsumesCallAttempt(kind WorkItemType) bool {
	switch kind {
	case WorkItemTypeLLM,
		WorkItemTypeTool,
		WorkItemTypeService,
		WorkItemTypeActivity,
		WorkItemTypeCompensation,
		WorkItemTypeReconciliation,
		WorkItemTypeTargetResolver:
		return true
	default:
		return false
	}
}

func checkBudgetCount(
	limit int,
	current int,
	increment int,
	label string,
) error {
	if limit == 0 || current+increment <= limit {
		return nil
	}
	return &SanitizedError{
		Code:     budgetErrorCode,
		Category: ErrorCategoryLimit,
		Message:  "execution budget exceeded for " + label,
	}
}

func (r *Runtime) checkDispatchBudgetLocked(
	execution Execution,
	item WorkItem,
) error {
	if execution.Deadline != nil &&
		!r.now().Before(execution.Deadline.UTC()) &&
		!item.Cleanup {
		return &SanitizedError{
			Code:     budgetDeadlineErrorCode,
			Category: ErrorCategoryTimeout,
			Message:  "execution deadline elapsed before work dispatch",
		}
	}
	if execution.Cleanup.Active {
		if !item.Cleanup {
			return &SanitizedError{
				Code:     budgetErrorCode,
				Category: ErrorCategoryCancellation,
				Message:  "normal work is not allowed during cleanup",
			}
		}
		if execution.Cleanup.Deadline != nil &&
			!r.now().Before(execution.Cleanup.Deadline.UTC()) {
			return &SanitizedError{
				Code:     budgetCleanupExpiredErrorCode,
				Category: ErrorCategoryTimeout,
				Message:  "cleanup deadline elapsed before work dispatch",
			}
		}
	}
	if execution.Budget.Concurrency > 0 &&
		!item.Cleanup &&
		activeBudgetWork(execution) >= execution.Budget.Concurrency {
		return &SanitizedError{
			Code:     budgetErrorCode,
			Category: ErrorCategoryLimit,
			Message:  "execution concurrency budget is exhausted",
		}
	}
	return nil
}

func activeBudgetWork(execution Execution) int {
	count := 0
	for _, item := range execution.WorkItems {
		if item.State == WorkItemStateDispatched {
			count++
		}
	}
	return count
}

func (r *Runtime) reserveActivityBudgetLocked(
	execution *Execution,
	activity Activity,
) error {
	if execution == nil {
		return errors.New("execution budget owner is unavailable")
	}
	bytes, err := activityBudgetBytes(activity)
	if err != nil {
		return err
	}
	if err := checkBudgetCount(
		execution.Budget.MessageCount,
		execution.Usage.MessageCount,
		1,
		"messages",
	); err != nil {
		return err
	}
	if err := checkBudgetCount(
		execution.Budget.MessageBytes,
		execution.Usage.MessageBytes,
		bytes,
		"message bytes",
	); err != nil {
		return err
	}
	execution.Usage.MessageCount++
	execution.Usage.MessageBytes += bytes
	return nil
}

func activityBudgetBytes(activity Activity) (int, error) {
	raw, err := json.Marshal(activity)
	if err != nil {
		return 0, err
	}
	return len(raw), nil
}

func (r *Runtime) reserveEventBudgetLocked(
	execution *Execution,
	data CanonicalValue,
	correlation CanonicalValue,
) error {
	if execution == nil {
		return errors.New("execution budget owner is unavailable")
	}
	bytes := len(data) + len(correlation)
	if err := checkBudgetCount(
		execution.Budget.EventCount,
		execution.Usage.EventCount,
		1,
		"events",
	); err != nil {
		return err
	}
	if err := checkBudgetCount(
		execution.Budget.EventBytes,
		execution.Usage.EventBytes,
		bytes,
		"event bytes",
	); err != nil {
		return err
	}
	execution.Usage.EventCount++
	execution.Usage.EventBytes += bytes
	return nil
}

func (r *Runtime) consumeLLMUsageLocked(
	execution *Execution,
	output *WorkItemOutput,
) error {
	if execution == nil || output == nil || output.LLM == nil ||
		output.LLM.Usage == nil {
		return nil
	}

	raw, err := json.Marshal(output.LLM.Usage)
	if err != nil {
		return err
	}
	var usage struct {
		InputTokens      int `json:"inputTokens"`
		OutputTokens     int `json:"outputTokens"`
		PromptTokens     int `json:"promptTokens"`
		CompletionTokens int `json:"completionTokens"`
		TotalTokens      int `json:"totalTokens"`
	}
	if err := json.Unmarshal(raw, &usage); err != nil {
		return err
	}
	tokens := usage.TotalTokens
	if tokens == 0 {
		tokens = usage.InputTokens + usage.OutputTokens
	}
	if tokens == 0 {
		tokens = usage.PromptTokens + usage.CompletionTokens
	}
	if tokens < 0 {
		return errors.New("provider usage has a negative token count")
	}
	if err := checkBudgetCount(
		execution.Budget.ModelTokens,
		execution.Usage.ModelTokens,
		tokens,
		"model tokens",
	); err != nil {
		return err
	}
	execution.Usage.ModelTokens += tokens
	return nil
}

func (r *Runtime) beginCleanupLocked(
	execution *Execution,
	primary *SanitizedError,
) {
	if execution == nil || execution.Cleanup.Active {
		return
	}
	now := r.now()
	execution.Cleanup.Active = true
	execution.Cleanup.StartedAt = &now
	execution.Cleanup.Primary = cloneWorkflowSanitizedError(primary)
	deadline := now
	if execution.Budget.CleanupDuration != "" {
		if value, err := execution.Budget.CleanupDuration.Deadline(now); err == nil {
			deadline = value
		}
	}
	execution.Cleanup.Deadline = &deadline
}

func (r *Runtime) cleanupDispatchAllowedLocked(
	execution Execution,
	item WorkItem,
) bool {
	if !execution.Cleanup.Active {
		return !execution.Controls.CancellationRequested
	}
	if !item.Cleanup {
		return false
	}
	if execution.Cleanup.Deadline == nil {
		return false
	}
	return r.now().Before(execution.Cleanup.Deadline.UTC())
}

func (r *Runtime) armExecutionDeadlineLocked(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	execution Execution,
) {
	if r == nil || execution.Deadline == nil || execution.ID == "" {
		return
	}
	if r.executionTimers == nil {
		r.executionTimers = make(map[ExecutionID]context.CancelFunc)
	}
	if _, found := r.executionTimers[execution.ID]; found {
		return
	}

	timerCtx, cancel := context.WithCancel(r.lifecycleCtx)
	r.executionTimers[execution.ID] = cancel
	deadline := execution.Deadline.UTC()
	r.launchAsync(
		executionDeadlineTimerOperation,
		func() {
			r.waitForExecutionDeadline(
				timerCtx,
				context.WithoutCancel(ctx),
				sessionID,
				executionID,
				deadline,
			)
		},
		nil,
	)
}

func (r *Runtime) cancelExecutionDeadlineLocked(
	executionID ExecutionID,
) {
	if r == nil || r.executionTimers == nil {
		return
	}
	if cancel, found := r.executionTimers[executionID]; found {
		delete(r.executionTimers, executionID)
		cancel()
	}
}

func (r *Runtime) armCleanupDeadlineLocked(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	execution Execution,
) {
	if r == nil ||
		!execution.Cleanup.Active ||
		execution.Cleanup.Deadline == nil ||
		execution.Cleanup.Expired ||
		execution.ID == "" {
		return
	}
	if r.cleanupTimers == nil {
		r.cleanupTimers = make(map[ExecutionID]context.CancelFunc)
	}
	if _, found := r.cleanupTimers[execution.ID]; found {
		return
	}

	timerCtx, cancel := context.WithCancel(r.lifecycleCtx)
	r.cleanupTimers[execution.ID] = cancel
	deadline := execution.Cleanup.Deadline.UTC()
	r.launchAsync(
		cleanupDeadlineTimerOperation,
		func() {
			r.waitForCleanupDeadline(
				timerCtx,
				context.WithoutCancel(ctx),
				sessionID,
				executionID,
				deadline,
			)
		},
		nil,
	)
}

func (r *Runtime) cancelCleanupDeadlineLocked(
	executionID ExecutionID,
) {
	if r == nil || r.cleanupTimers == nil {
		return
	}
	if cancel, found := r.cleanupTimers[executionID]; found {
		delete(r.cleanupTimers, executionID)
		cancel()
	}
}

func (r *Runtime) waitForCleanupDeadline(
	ctx context.Context,
	persistContext context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	deadline time.Time,
) {
	delay := deadline.Sub(r.now())
	timer := time.NewTimer(max(delay, 0))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil ||
		!execution.Cleanup.Active ||
		execution.Cleanup.Deadline == nil ||
		!execution.Cleanup.Deadline.Equal(deadline) ||
		execution.State.Terminal() {
		return
	}
	delete(r.cleanupTimers, executionID)

	now := r.now()
	execution.Cleanup.Expired = true
	expired := &SanitizedError{
		Code:     cleanupDeadlineExceededCode,
		Category: ErrorCategoryTimeout,
		Message:  cleanupDeadlineExceededMessage,
	}
	execution.Cleanup.Primary = appendCleanupCause(
		execution.Cleanup.Primary,
		*expired,
	)

	for index := range execution.WorkItems {
		item := &execution.WorkItems[index]
		if !item.Cleanup || item.State.Terminal() {
			continue
		}
		r.expireCleanupWorkItemLocked(execution, item, now, expired)
	}
	for index := range execution.ProgramFrames {
		frame := &execution.ProgramFrames[index]
		if frame.Status.Terminal() || !workflowFrameRunsCleanup(*frame) {
			continue
		}
		frame.Status = ProgramFrameStateCancelled
		frame.Error = cloneWorkflowSanitizedError(expired)
		frame.UpdatedAt = now
	}

	outcome := ExecutionOutcome{
		Kind:        ExecutionOutcomeFailed,
		Code:        cleanupDeadlineExceededCode,
		Message:     cleanupDeadlineExceededMessage,
		Error:       cloneWorkflowSanitizedError(execution.Cleanup.Primary),
		CompletedAt: now,
	}
	if execution.Controls.CancellationRequested {
		outcome.Kind = ExecutionOutcomeCancelled
	}
	r.completeExecutionLocked(persistContext, session, execution, outcome)
	if err := r.persistLocked(persistContext, "cleanupDeadlineElapsed", session); err != nil {
		return
	}
	r.publishLocked(
		persistContext,
		session,
		execution.ID,
		RuntimeEventExecutionChanged,
	)
}

func (r *Runtime) expireCleanupWorkItemLocked(
	execution *Execution,
	item *WorkItem,
	now time.Time,
	expired *SanitizedError,
) {
	if execution == nil || item == nil || item.State.Terminal() {
		return
	}

	if cancel := r.cancels[item.ID]; cancel != nil {
		cancel()
		delete(r.cancels, item.ID)
	}
	r.cancelResourceWaitLocked(item.ID)
	r.cancelRetryLocked(item.ID)
	delete(r.privateBindings, item.ID)

	if item.WaitID != "" {
		if wait := mutableWait(execution, item.WaitID); wait != nil &&
			(wait.Status == WaitStatusPending || wait.Status == WaitStatusResolving) {
			wait.Status = WaitStatusCancelled
			wait.Error = cloneWorkflowSanitizedError(expired)
			wait.UpdatedAt = now
			r.cancelWaitDeadlineLocked(wait.ID)
			delete(r.privateWaits, wait.ID)
		}
		item.WaitID = ""
	}

	effectState := EffectStateConfirmedNotApplied
	attemptState := AttemptStateCancelled
	if item.State == WorkItemStateDispatched &&
		item.Effect.Kind != EffectKindRead {
		effectState = EffectStateUncertain
		attemptState = AttemptStateUncertain
		item.State = WorkItemStateUncertain
	} else {
		item.State = WorkItemStateCancelled
	}
	if item.Effect.Kind == EffectKindRead {
		effectState = EffectStateNotApplicable
	}
	item.EffectState = effectState
	item.Error = copyEffectError(expired, effectState)
	item.Output = nil
	item.CompletedAt = &now
	item.UpdatedAt = now

	if attempt := activeAttempt(execution, item.ID); attempt != nil {
		attempt.State = attemptState
		attempt.EffectState = effectState
		attempt.Error = copyEffectError(expired, effectState)
		attempt.Receipt = nil
		attempt.FinishedAt = &now
	}

	released := r.releaseWorkResourcesLocked(execution, item.ID)
	r.wakeResourceWaitersLocked(execution, released)
}

func appendCleanupCause(
	primary *SanitizedError,
	cause SanitizedError,
) *SanitizedError {
	if primary == nil {
		return cloneWorkflowSanitizedError(&cause)
	}
	return appendWorkflowErrorCause(primary, cause)
}

func hasOutstandingCleanup(execution Execution) bool {
	for _, item := range execution.WorkItems {
		if item.Cleanup && !item.State.Terminal() {
			return true
		}
	}
	for _, frame := range execution.ProgramFrames {
		if workflowFrameRunsCleanup(frame) && !frame.Status.Terminal() {
			return true
		}
	}
	return false
}

func (r *Runtime) waitForExecutionDeadline(
	ctx context.Context,
	persistContext context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	deadline time.Time,
) {
	delay := deadline.Sub(r.now())
	timer := time.NewTimer(max(delay, 0))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil || execution.Deadline == nil ||
		!execution.Deadline.Equal(deadline) ||
		execution.State.Terminal() {
		return
	}
	delete(r.executionTimers, executionID)

	now := r.now()
	execution.Controls.CancellationRequested = true
	execution.State = ExecutionStateCancelling
	primary := &SanitizedError{
		Code:     budgetDeadlineErrorCode,
		Category: ErrorCategoryTimeout,
		Message:  "execution deadline elapsed",
	}
	r.beginCleanupLocked(execution, primary)

	for index := range execution.WorkItems {
		item := &execution.WorkItems[index]
		if item.Cleanup || item.State.Terminal() {
			continue
		}
		switch item.State {
		case WorkItemStateReady,
			WorkItemStateBinding,
			WorkItemStateResourceWaiting,
			WorkItemStateRetryWaiting,
			WorkItemStateWaiting:
			if cancel := r.cancels[item.ID]; cancel != nil {
				cancel()
				delete(r.cancels, item.ID)
			}
			r.cancelResourceWaitLocked(item.ID)
			r.cancelRetryLocked(item.ID)
			item.State = WorkItemStateCancelled
			item.CompletedAt = &now
			item.UpdatedAt = now
		case WorkItemStateDispatched:
			if cancel := r.cancels[item.ID]; cancel != nil {
				cancel()
				delete(r.cancels, item.ID)
			}
		default:
		}
	}
	frameIDs := make([]ProgramFrameID, 0, len(execution.ProgramFrames))
	for _, frame := range execution.ProgramFrames {
		if frame.Status.Terminal() || workflowFrameRunsCleanup(frame) {
			continue
		}
		frameIDs = append(frameIDs, frame.ID)
	}
	for _, frameID := range frameIDs {
		_ = r.advanceFrameLocked(
			session,
			execution,
			frameID,
			ProgramEvent{Kind: ProgramEventCancelled},
		)
	}
	r.touchExecutionLocked(session, execution)
	if err := r.persistLocked(persistContext, "executionDeadlineElapsed", session); err != nil {
		return
	}
	r.publishLocked(
		persistContext,
		session,
		execution.ID,
		RuntimeEventExecutionChanged,
	)
	r.armCleanupDeadlineLocked(
		persistContext,
		session.ID,
		execution.ID,
		*execution,
	)
	r.launchWork(
		persistContext,
		session.ID,
		execution.ID,
		r.readyWorkLocked(*execution),
	)
}
