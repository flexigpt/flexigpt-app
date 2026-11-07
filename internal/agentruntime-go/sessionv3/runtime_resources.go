package sessionv3

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

func initialExecutionResources(
	values []WorkflowResourceDefinition,
) ([]WorkflowResourceDefinition, error) {
	if len(values) == 0 {
		return nil, nil
	}

	output := make(
		[]WorkflowResourceDefinition,
		0,
		len(values),
	)
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[value.Name]; duplicate {
			return nil, errors.New("workflow resource definition is duplicated")
		}
		seen[value.Name] = struct{}{}
		output = append(output, cloneWorkflowResourceDefinition(value))
	}
	return output, nil
}

func cloneWorkflowResourceDefinition(
	value WorkflowResourceDefinition,
) WorkflowResourceDefinition {
	output := value
	if value.Rate != nil {
		rate := *value.Rate
		output.Rate = &rate
	}
	return output
}

func (r *Runtime) mergeExecutionResourcesLocked(
	execution *Execution,
	values []WorkflowResourceDefinition,
) error {
	if execution == nil || len(values) == 0 {
		return nil
	}

	for _, incoming := range values {
		if err := incoming.Validate(); err != nil {
			return err
		}
		found := false
		for _, existing := range execution.Resources {
			if existing.Name != incoming.Name {
				continue
			}
			if !workflowResourceDefinitionsEqual(existing, incoming) {
				return fmt.Errorf(
					"workflow resource %q conflicts with an inherited resource",
					incoming.Name,
				)
			}
			found = true
			break
		}
		if !found {
			execution.Resources = append(
				execution.Resources,
				cloneWorkflowResourceDefinition(incoming),
			)
		}
	}
	return nil
}

func workflowResourceDefinitionsEqual(
	left WorkflowResourceDefinition,
	right WorkflowResourceDefinition,
) bool {
	if left.Name != right.Name ||
		left.Concurrency != right.Concurrency ||
		left.Keyed != right.Keyed {
		return false
	}
	switch {
	case left.Rate == nil && right.Rate == nil:
		return true
	case left.Rate == nil || right.Rate == nil:
		return false
	default:
		return *left.Rate == *right.Rate
	}
}

func resourceDefinition(
	execution Execution,
	name string,
) (WorkflowResourceDefinition, bool) {
	for _, value := range execution.Resources {
		if value.Name == name {
			return value, true
		}
	}
	return WorkflowResourceDefinition{}, false
}

func (r *Runtime) acquireWorkResourcesLocked(
	execution *Execution,
	item *WorkItem,
) (
	bool,
	ResourceWaitReason,
	*time.Time,
	error,
) {
	if execution == nil || item == nil || len(item.Claims) == 0 {
		return true, "", nil, nil
	}
	if resourceLeaseCount(*execution, item.ID) != 0 {
		return true, "", nil, nil
	}

	now := r.now()
	nextLeases := append(
		[]ResourceLease(nil),
		execution.ResourceLeases...,
	)
	nextRates := append(
		[]ResourceRateWindow(nil),
		execution.ResourceRates...,
	)

	var (
		waitReason ResourceWaitReason
		wakeAt     *time.Time
	)
	for _, claim := range item.Claims {
		if err := claim.Validate(); err != nil {
			return false, "", nil, err
		}
		resource, found := resourceDefinition(*execution, claim.Resource)
		if !found {
			return false, "", nil, fmt.Errorf(
				"resource claim %q has no definition",
				claim.Resource,
			)
		}
		if resource.Keyed && strings.TrimSpace(claim.Key) == "" {
			return false, "", nil, fmt.Errorf(
				"keyed resource claim %q has no key",
				claim.Resource,
			)
		}
		if !resource.Keyed && claim.Key != "" {
			return false, "", nil, fmt.Errorf(
				"non-keyed resource claim %q has a key",
				claim.Resource,
			)
		}
		if resource.Concurrency > 0 &&
			resourceLeaseUnits(nextLeases, claim.Resource)+claim.Units >
				resource.Concurrency {
			waitReason = ResourceWaitReasonConcurrency
			continue
		}
		if resource.Keyed &&
			resourceKeyHeld(nextLeases, claim.Resource, claim.Key) {
			waitReason = ResourceWaitReasonKeyed
			continue
		}

		if resource.Rate != nil {
			allowed, nextWake, err := consumeResourceRate(
				&nextRates,
				resource,
				claim.Units,
				now,
			)
			if err != nil {
				return false, "", nil, err
			}
			if !allowed {
				waitReason = ResourceWaitReasonRate
				wakeAt = laterResourceWakeAt(wakeAt, nextWake)
				continue
			}
		}

		nextLeases = append(nextLeases, ResourceLease{
			WorkItemID: item.ID,
			Resource:   claim.Resource,
			Units:      claim.Units,
			Key:        claim.Key,
			AcquiredAt: now,
		})
	}

	if waitReason != "" {
		return false, waitReason, wakeAt, nil
	}
	execution.ResourceLeases = nextLeases
	execution.ResourceRates = nextRates
	return true, "", nil, nil
}

func consumeResourceRate(
	values *[]ResourceRateWindow,
	resource WorkflowResourceDefinition,
	units int,
	now time.Time,
) (bool, *time.Time, error) {
	if values == nil || resource.Rate == nil {
		return true, nil, nil
	}
	rate := resource.Rate
	capacity := rate.Units + rate.Burst
	if capacity < units {
		return false, nil, fmt.Errorf(
			"resource rate %q cannot satisfy claim units",
			resource.Name,
		)
	}

	index := resourceRateWindowIndex(*values, resource.Name)
	if index < 0 {
		*values = append(*values, ResourceRateWindow{
			Resource:      resource.Name,
			WindowStarted: now,
		})
		index = len(*values) - 1
	}
	window := &(*values)[index]
	windowEnd, err := rate.Per.Deadline(window.WindowStarted)
	if err != nil {
		return false, nil, err
	}
	if !now.Before(windowEnd) {
		window.WindowStarted = now
		window.UnitsConsumed = 0
		windowEnd, err = rate.Per.Deadline(window.WindowStarted)
		if err != nil {
			return false, nil, err
		}
	}
	if window.UnitsConsumed+units > capacity {
		return false, &windowEnd, nil
	}
	window.UnitsConsumed += units
	return true, nil, nil
}

func resourceRateWindowIndex(
	values []ResourceRateWindow,
	resource string,
) int {
	for index := range values {
		if values[index].Resource == resource {
			return index
		}
	}
	return -1
}

func resourceLeaseCount(
	execution Execution,
	workItemID WorkItemID,
) int {
	count := 0
	for _, lease := range execution.ResourceLeases {
		if lease.WorkItemID == workItemID {
			count++
		}
	}
	return count
}

func resourceLeaseUnits(
	leases []ResourceLease,
	resource string,
) int {
	units := 0
	for _, lease := range leases {
		if lease.Resource == resource {
			units += lease.Units
		}
	}
	return units
}

func resourceKeyHeld(
	leases []ResourceLease,
	resource string,
	key string,
) bool {
	for _, lease := range leases {
		if lease.Resource == resource && lease.Key == key {
			return true
		}
	}
	return false
}

func laterResourceWakeAt(
	current *time.Time,
	candidate *time.Time,
) *time.Time {
	switch {
	case current == nil:
		return cloneTime(candidate)
	case candidate == nil:
		return cloneTime(current)
	case current.Before(*candidate):
		return cloneTime(candidate)
	default:
		return cloneTime(current)
	}
}

func (r *Runtime) setResourceWaitingLocked(
	item *WorkItem,
	reason ResourceWaitReason,
	wakeAt *time.Time,
) {
	if item == nil {
		return
	}
	r.resourceWaitSequence++
	item.State = WorkItemStateResourceWaiting
	item.ResourceWait = &ResourceWaitState{
		Reason:     reason,
		WakeAt:     cloneTime(wakeAt),
		Generation: r.resourceWaitSequence,
	}
	item.UpdatedAt = r.now()
}

func (r *Runtime) releaseWorkResourcesLocked(
	execution *Execution,
	workItemID WorkItemID,
) []string {
	if execution == nil || workItemID == "" {
		return nil
	}
	released := make(map[string]struct{})
	next := execution.ResourceLeases[:0]
	for _, lease := range execution.ResourceLeases {
		if lease.WorkItemID == workItemID {
			released[lease.Resource] = struct{}{}
			continue
		}
		next = append(next, lease)
	}
	execution.ResourceLeases = next

	output := make([]string, 0, len(released))
	for name := range released {
		output = append(output, name)
	}
	return output
}

func (r *Runtime) transferWorkResourcesToFrameLocked(
	execution *Execution,
	workItemID WorkItemID,
	frameID ProgramFrameID,
) {
	if execution == nil || workItemID == "" || frameID == "" {
		return
	}
	for index := range execution.ResourceLeases {
		lease := &execution.ResourceLeases[index]
		if lease.WorkItemID != workItemID {
			continue
		}
		lease.WorkItemID = ""
		lease.FrameID = frameID
	}
}

func (r *Runtime) releaseFrameResourcesLocked(
	execution *Execution,
	frameID ProgramFrameID,
) []string {
	if execution == nil || frameID == "" {
		return nil
	}

	released := make(map[string]struct{})
	next := execution.ResourceLeases[:0]
	for _, lease := range execution.ResourceLeases {
		if lease.FrameID == frameID {
			released[lease.Resource] = struct{}{}
			continue
		}
		next = append(next, lease)
	}
	execution.ResourceLeases = next

	output := make([]string, 0, len(released))
	for resource := range released {
		output = append(output, resource)
	}
	return output
}

func (r *Runtime) wakeResourceWaitersLocked(
	execution *Execution,
	released []string,
) {
	if execution == nil || len(released) == 0 {
		return
	}
	for index := range execution.WorkItems {
		item := &execution.WorkItems[index]
		if item.State != WorkItemStateResourceWaiting ||
			item.ResourceWait == nil ||
			item.ResourceWait.WakeAt != nil {
			continue
		}
		if !workItemUsesResource(*item, released) {
			continue
		}
		r.cancelResourceWaitLocked(item.ID)
		item.State = WorkItemStateReady
		item.ResourceWait = nil
		item.UpdatedAt = r.now()
	}
}

func workItemUsesResource(
	item WorkItem,
	resources []string,
) bool {
	for _, claim := range item.Claims {
		if slices.Contains(resources, claim.Resource) {
			return true
		}
	}
	return false
}

func (r *Runtime) armResourceWaitLocked(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	item WorkItem,
) {
	if r == nil ||
		item.ID == "" ||
		item.State != WorkItemStateResourceWaiting ||
		item.ResourceWait == nil ||
		item.ResourceWait.WakeAt == nil {
		return
	}
	if r.resourceTimers == nil {
		r.resourceTimers = make(map[WorkItemID]context.CancelFunc)
	}
	if _, found := r.resourceTimers[item.ID]; found {
		return
	}

	waitCtx, cancel := context.WithCancel(r.lifecycleCtx)
	r.resourceTimers[item.ID] = cancel
	wakeAt := item.ResourceWait.WakeAt.UTC()
	generation := item.ResourceWait.Generation
	r.launchAsync(
		"session-v3 resource wait timer",
		func() {
			r.waitForResource(
				waitCtx,
				context.WithoutCancel(ctx),
				sessionID,
				executionID,
				item.ID,
				generation,
				wakeAt,
			)
		},
		nil,
	)
}

func (r *Runtime) armExecutionResourceWaitsLocked(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	execution Execution,
) {
	for _, item := range execution.WorkItems {
		r.armResourceWaitLocked(ctx, sessionID, executionID, item)
	}
}

func (r *Runtime) cancelResourceWaitLocked(
	workItemID WorkItemID,
) {
	if r == nil || r.resourceTimers == nil {
		return
	}
	if cancel, found := r.resourceTimers[workItemID]; found {
		delete(r.resourceTimers, workItemID)
		cancel()
	}
}

func (r *Runtime) waitForResource(
	ctx context.Context,
	persistContext context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	workItemID WorkItemID,
	generation uint64,
	wakeAt time.Time,
) {
	delay := wakeAt.Sub(r.now())
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		return
	}
	item := mutableWorkItem(execution, workItemID)
	if item == nil ||
		item.State != WorkItemStateResourceWaiting ||
		item.ResourceWait == nil ||
		item.ResourceWait.Generation != generation ||
		item.ResourceWait.WakeAt == nil ||
		!item.ResourceWait.WakeAt.Equal(wakeAt) {
		return
	}
	delete(r.resourceTimers, workItemID)

	if execution.Controls.CancellationRequested ||
		execution.State.Terminal() {
		item.State = WorkItemStateCancelled
		item.ResourceWait = nil
		item.UpdatedAt = r.now()
		r.touchExecutionLocked(session, execution)
		_ = r.persistLocked(persistContext, "resourceWaitCancelled", session)
		return
	}

	item.State = WorkItemStateReady
	item.ResourceWait = nil
	item.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)
	if err := r.persistLocked(persistContext, "resourceReady", session); err != nil {
		return
	}
	r.publishLocked(
		persistContext,
		session,
		execution.ID,
		RuntimeEventWorkChanged,
	)
	r.launchWork(
		persistContext,
		session.ID,
		execution.ID,
		r.readyWorkLocked(*execution),
	)
}

func normalizeRecoveredResources(
	execution *Execution,
) {
	if execution == nil {
		return
	}

	active := make(map[WorkItemID]struct{}, len(execution.WorkItems))
	activeFrames := make(map[ProgramFrameID]struct{}, len(execution.ProgramFrames))
	for _, item := range execution.WorkItems {
		switch item.State {
		case WorkItemStateBinding,
			WorkItemStateDispatched,
			WorkItemStateRetryWaiting:
			active[item.ID] = struct{}{}
		default:
		}
	}
	for _, frame := range execution.ProgramFrames {
		if !frame.Status.Terminal() {
			activeFrames[frame.ID] = struct{}{}
		}
	}

	leases := execution.ResourceLeases[:0]
	for _, lease := range execution.ResourceLeases {
		if _, found := active[lease.WorkItemID]; found {
			leases = append(leases, lease)
			continue
		}
		if _, found := activeFrames[lease.FrameID]; found {
			leases = append(leases, lease)
		}
	}
	execution.ResourceLeases = leases
}
