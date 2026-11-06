package sessionv3

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

func (r *Runtime) acceptSignalForSubscriptionsLocked(
	execution *Execution,
	signal ExecutionSignal,
) error {
	if execution == nil {
		return errors.New("subscription signal owner is unavailable")
	}

	now := r.now()
	for frameIndex := range execution.ProgramFrames {
		frame := &execution.ProgramFrames[frameIndex]
		if frame.Status.Terminal() ||
			frame.Program.InterpreterID != WorkflowInterpreterID {
			continue
		}

		_, state, err := decodeWorkflowProgram(*frame)
		if err != nil {
			return fmt.Errorf(
				"decode workflow subscription state for frame %q: %w",
				frame.ID,
				err,
			)
		}

		changed := false
		for subscriptionIndex := range state.Subscriptions {
			subscription := &state.Subscriptions[subscriptionIndex]
			if changedByPurge, err := purgeWorkflowSubscription(
				subscription,
				now,
			); err != nil {
				return err
			} else if changedByPurge {
				changed = true
			}

			matched, err := workflowSubscriptionMatchesSignal(
				*subscription,
				signal,
			)
			if err != nil {
				return err
			}
			if !matched {
				continue
			}

			if changedByAccept, err := appendWorkflowSubscriptionEvent(
				subscription,
				signal,
				now,
			); err != nil {
				return fmt.Errorf(
					"retain workflow subscription %q: %w",
					subscription.Name,
					err,
				)
			} else if changedByAccept {
				changed = true
			}
		}

		if !changed {
			continue
		}
		next, err := encodeWorkflowProgramState(state)
		if err != nil {
			return err
		}
		frame.State = next
		frame.UpdatedAt = now
	}
	return nil
}

func workflowSubscriptionMatchesSignal(
	subscription WorkflowSubscriptionState,
	signal ExecutionSignal,
) (bool, error) {
	if subscription.Event != signal.Name ||
		subscription.Source != signal.Source {
		return false, nil
	}
	if !subscription.ActivatedAt.IsZero() &&
		signal.CreatedAt.Before(subscription.ActivatedAt) {
		return false, nil
	}
	if len(subscription.Correlation) != 0 &&
		!canonicalValuesEqual(subscription.Correlation, signal.Correlation) {
		return false, nil
	}
	if err := validateWorkflowSchema(
		subscription.DataSchema,
		signal.Data,
	); err != nil {
		return false, nil
	}
	if err := validateWorkflowSchema(
		subscription.CorrelationSchema,
		signal.Correlation,
	); err != nil {
		return false, nil
	}
	if subscription.Filter == nil {
		return true, nil
	}

	event := MustCanonicalValue(eventWaitOutput{
		ID:          signal.ID,
		Name:        signal.Name,
		Source:      signal.Source,
		Data:        signal.Data.Clone(),
		Correlation: signal.Correlation.Clone(),
		Time:        signal.CreatedAt,
	})
	return subscription.Filter.Matches(event)
}

func appendWorkflowSubscriptionEvent(
	subscription *WorkflowSubscriptionState,
	signal ExecutionSignal,
	now time.Time,
) (bool, error) {
	if subscription == nil {
		return false, errors.New("workflow subscription is unavailable")
	}
	if subscription.Capacity < 1 ||
		subscription.MaxBytes < 1 {
		return false, errors.New("workflow subscription retention facts are invalid")
	}

	for _, existing := range subscription.Accepted {
		if existing.Source == signal.Source && existing.ID == signal.ID {
			return false, nil
		}
	}

	bytes := len(signal.Data) + len(signal.Correlation)
	if len(subscription.Accepted) >= subscription.Capacity {
		return false, errors.New("workflow subscription capacity is exceeded")
	}
	if subscription.Bytes+bytes > subscription.MaxBytes {
		return false, errors.New("workflow subscription byte limit is exceeded")
	}

	subscription.Accepted = append(
		subscription.Accepted,
		WorkflowSubscriptionEvent{
			Source:     signal.Source,
			ID:         signal.ID,
			AcceptedAt: now.UTC(),
			Bytes:      bytes,
		},
	)
	subscription.Bytes += bytes
	return true, nil
}

func purgeWorkflowSubscription(
	subscription *WorkflowSubscriptionState,
	now time.Time,
) (bool, error) {
	if subscription == nil || subscription.Retention == "" {
		return false, nil
	}

	retained := make(
		[]WorkflowSubscriptionEvent,
		0,
		len(subscription.Accepted),
	)
	bytes := 0
	changed := false
	for _, event := range subscription.Accepted {
		expiresAt, err := subscription.Retention.Deadline(event.AcceptedAt)
		if err != nil {
			return false, err
		}
		if !now.Before(expiresAt.UTC()) {
			changed = true
			continue
		}
		retained = append(retained, event)
		bytes += event.Bytes
	}

	if !changed {
		return false, nil
	}
	subscription.Accepted = retained
	subscription.Bytes = bytes
	return true, nil
}

func (r *Runtime) subscriptionAcceptedForWaitLocked(
	execution *Execution,
	wait Wait,
	signal ExecutionSignal,
) (bool, error) {
	if wait.Request.Subscription == "" {
		return true, nil
	}
	if execution == nil {
		return false, errors.New("subscription wait owner is unavailable")
	}

	item := mutableWorkItem(execution, wait.WorkItemID)
	if item == nil {
		return false, errors.New("subscription wait work item is unavailable")
	}
	frame := mutableFrame(execution, item.FrameID)
	if frame == nil || frame.Program.InterpreterID != WorkflowInterpreterID {
		return false, nil
	}
	if wait.Request.SubscriptionFrameID != "" &&
		wait.Request.SubscriptionFrameID != frame.ID {
		return false, nil
	}

	_, state, err := decodeWorkflowProgram(*frame)
	if err != nil {
		return false, err
	}

	now := r.now()
	changed := false
	for index := range state.Subscriptions {
		subscription := &state.Subscriptions[index]
		if subscription.Name != wait.Request.Subscription {
			continue
		}
		if changedByPurge, purgeErr := purgeWorkflowSubscription(
			subscription,
			now,
		); purgeErr != nil {
			return false, purgeErr
		} else if changedByPurge {
			changed = true
		}

		for _, accepted := range subscription.Accepted {
			if accepted.Source == signal.Source &&
				accepted.ID == signal.ID {
				if changed {
					next, encodeErr := encodeWorkflowProgramState(state)
					if encodeErr != nil {
						return false, encodeErr
					}
					frame.State = next
					frame.UpdatedAt = now
				}
				return true, nil
			}
		}
		break
	}

	if changed {
		next, encodeErr := encodeWorkflowProgramState(state)
		if encodeErr != nil {
			return false, encodeErr
		}
		frame.State = next
		frame.UpdatedAt = now
	}
	return false, nil
}

func subscriptionConsumptionForWait(
	execution *Execution,
	wait Wait,
) (SubscriptionConsumption, bool) {
	if execution == nil || wait.Request.Subscription == "" {
		return SubscriptionConsumption{}, false
	}

	item := mutableWorkItem(execution, wait.WorkItemID)
	if item == nil {
		return SubscriptionConsumption{}, false
	}
	frameID := wait.Request.SubscriptionFrameID
	if frameID == "" {
		frameID = item.FrameID
	}
	if frameID == "" {
		return SubscriptionConsumption{}, false
	}
	return workflowSubscriptionIdentity(frameID, wait.Request.Subscription), true
}

func signalConsumedBySubscription(
	signal ExecutionSignal,
	consumption SubscriptionConsumption,
) bool {
	return slices.Contains(signal.SubscriptionConsumptions, consumption)
}
