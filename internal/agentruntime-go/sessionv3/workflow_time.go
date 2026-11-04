package sessionv3

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WorkflowDuration is an ISO 8601 duration, not an arbitrary JSON value.
// Calendar components remain separate from elapsed-time components.
type WorkflowDuration string

var workflowDurationPattern = regexp.MustCompile(
	`^P(?:([0-9]+)Y)?(?:([0-9]+)M)?(?:([0-9]+)W)?(?:([0-9]+)D)?(?:T(?:([0-9]+)H)?(?:([0-9]+)M)?(?:([0-9]+(?:\.[0-9]+)?)S)?)?$`,
)

func ParseWorkflowDuration(value string) (WorkflowDuration, error) {
	if value == "" || len(value) > 256 ||
		value == "P" || strings.HasSuffix(value, "T") ||
		!workflowDurationPattern.MatchString(value) {
		return "", errors.New("invalid workflow iso 8601 duration")
	}
	return WorkflowDuration(value), nil
}

// Deadline applies calendar components in UTC, then elapsed components.
// Positive sub-nanosecond durations round up so a timer cannot fire early.
// Values outside the supported timestamp range fail before work dispatch.
func (d WorkflowDuration) Deadline(start time.Time) (time.Time, error) {
	if start.IsZero() {
		return time.Time{}, errors.New("workflow duration has no logical start time")
	}
	if _, err := ParseWorkflowDuration(string(d)); err != nil {
		return time.Time{}, err
	}

	parts := workflowDurationPattern.FindStringSubmatch(string(d))
	calendar := make([]int, 4)
	maximum := []int64{9999, 119988, 521722, 3652059}
	for index := range calendar {
		if parts[index+1] == "" {
			continue
		}
		value, err := strconv.ParseInt(parts[index+1], 10, 64)
		if err != nil || value > maximum[index] {
			return time.Time{}, errors.New("workflow duration exceeds the supported clock range")
		}
		calendar[index] = int(value)
	}

	next := start.UTC().AddDate(
		calendar[0],
		calendar[1],
		calendar[2]*7+calendar[3],
	)
	if next.Year() < 1 || next.Year() > 9999 {
		return time.Time{}, errors.New("workflow duration exceeds the supported clock range")
	}

	elapsed := new(big.Rat)
	units := []int64{
		int64(time.Hour),
		int64(time.Minute),
		int64(time.Second),
	}
	for index, unit := range units {
		raw := parts[index+5]
		if raw == "" {
			continue
		}
		component, valid := new(big.Rat).SetString(raw)
		if !valid {
			return time.Time{}, errors.New("invalid workflow duration component")
		}
		component.Mul(component, new(big.Rat).SetInt64(unit))
		elapsed.Add(elapsed, component)
	}

	nanoseconds := new(big.Int)
	remainder := new(big.Int)
	nanoseconds.QuoRem(elapsed.Num(), elapsed.Denom(), remainder)
	if remainder.Sign() != 0 {
		nanoseconds.Add(nanoseconds, big.NewInt(1))
	}
	if !nanoseconds.IsInt64() {
		return time.Time{}, errors.New("workflow elapsed duration exceeds the supported clock range")
	}

	next = next.Add(time.Duration(nanoseconds.Int64()))
	if next.Year() < 1 || next.Year() > 9999 {
		return time.Time{}, errors.New("workflow duration exceeds the supported clock range")
	}
	return next, nil
}

type TimerWait struct {
	ScheduledAt time.Time `json:"scheduledAt"`
	FireAt      time.Time `json:"fireAt"`
}

type TimerWaitOutput struct {
	ScheduledAt time.Time `json:"scheduledAt"`
	FireAt      time.Time `json:"fireAt"`
	FiredAt     time.Time `json:"firedAt"`
}

func (t TimerWait) Validate() error {
	if t.ScheduledAt.IsZero() || t.FireAt.IsZero() {
		return errors.New("timer wait timestamps are required")
	}
	return nil
}

func earlierDeadline(left, right *time.Time) *time.Time {
	switch {
	case left == nil:
		return cloneTime(right)
	case right == nil:
		return cloneTime(left)
	case left.Before(*right):
		return cloneTime(left)
	default:
		return cloneTime(right)
	}
}

func validateWorkflowTimeout(value WorkflowDuration) error {
	if value == "" {
		return nil
	}
	if _, err := ParseWorkflowDuration(string(value)); err != nil {
		return err
	}
	if !strings.ContainsAny(string(value), "123456789") {
		return errors.New("workflow timeout must be positive")
	}
	return nil
}

func (i *WorkflowInterpreter) timerWait(
	program WorkflowProgram,
	state WorkflowState,
	path string,
	spec WorkflowWait,
) (ProgramTransition, error) {
	var (
		fireAt time.Time
		err    error
	)
	switch {
	case spec.After != nil:
		var raw string
		raw, err = i.stringValue(*spec.After, program, state)
		if err == nil {
			var duration WorkflowDuration
			duration, err = ParseWorkflowDuration(raw)
			if err == nil {
				fireAt, err = duration.Deadline(state.LogicalTime)
			}
		}

	case spec.At != nil:
		var raw string
		raw, err = i.stringValue(*spec.At, program, state)
		if err == nil {
			fireAt, err = time.Parse(time.RFC3339Nano, raw)
		}

	default:
		err = errors.New("timer wait requires after or at")
	}
	if err != nil {
		return workflowFailure(
			state,
			"workflow_timer_invalid",
			fmt.Sprintf("prepare workflow timer: %v", err),
		)
	}

	return i.workflowStepWork(program, state, path, WorkItemIntent{
		ProgramPath: path,
		Type:        WorkItemTypeWait,
		Automatic:   true,
		Effect:      EffectContract{Kind: EffectKindRead},
		Input: WorkItemPayload{
			Wait: &WaitWork{
				Request: WaitRequest{
					Kind:     WaitKindTimer,
					Deadline: cloneTime(state.Deadline),
					Timer: &TimerWait{
						ScheduledAt: state.LogicalTime,
						FireAt:      fireAt.UTC(),
					},
				},
			},
		},
	})
}
