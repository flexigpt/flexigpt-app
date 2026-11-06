package workflowruntime

import (
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowTriggers(
	document workflowv1.WorkflowDocument,
) ([]sessionv3Runtime.WorkflowTrigger, error) {
	if len(document.Triggers) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(document.Triggers))
	for name := range document.Triggers {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make([]sessionv3Runtime.WorkflowTrigger, 0, len(names))
	for _, name := range names {
		source := document.Triggers[name]
		if (source.Event == nil) == (source.Schedule == nil) {
			return nil, fmt.Errorf(
				"workflow trigger %q must contain exactly one trigger form",
				name,
			)
		}
		value := sessionv3Runtime.WorkflowTrigger{Name: name}
		if source.Event != nil {
			event, err := compileWorkflowEventTrigger(*source.Event)
			if err != nil {
				return nil, fmt.Errorf(
					"workflow event trigger %q: %w",
					name,
					err,
				)
			}
			value.Event = &event
		}
		if source.Schedule != nil {
			schedule, err := compileWorkflowScheduleTrigger(*source.Schedule)
			if err != nil {
				return nil, fmt.Errorf(
					"workflow schedule trigger %q: %w",
					name,
					err,
				)
			}
			value.Schedule = &schedule
		}
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("workflow trigger %q: %w", name, err)
		}
		output = append(output, value)
	}
	return output, nil
}

func compileWorkflowEventTrigger(
	input workflowv1.EventTrigger,
) (sessionv3Runtime.WorkflowEventTrigger, error) {
	value, err := compileWorkflowValue(input.Input)
	if err != nil {
		return sessionv3Runtime.WorkflowEventTrigger{}, err
	}
	output := sessionv3Runtime.WorkflowEventTrigger{
		Event:    input.Event,
		Input:    value,
		Behavior: sessionv3Runtime.WorkflowTriggerBehavior(input.Behavior),
	}
	if len(input.Filter) != 0 {
		filter, filterErr := compileWorkflowValue(input.Filter)
		if filterErr != nil {
			return sessionv3Runtime.WorkflowEventTrigger{}, filterErr
		}
		output.Filter = &filter
	}
	if len(input.InstanceKey) != 0 {
		key, keyErr := compileWorkflowValue(input.InstanceKey)
		if keyErr != nil {
			return sessionv3Runtime.WorkflowEventTrigger{}, keyErr
		}
		output.InstanceKey = &key
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowEventTrigger{}, err
	}
	return output, nil
}

func compileWorkflowScheduleTrigger(
	input workflowv1.Schedule,
) (sessionv3Runtime.WorkflowScheduleTrigger, error) {
	value, err := compileWorkflowValue(input.Input)
	if err != nil {
		return sessionv3Runtime.WorkflowScheduleTrigger{}, err
	}
	output := sessionv3Runtime.WorkflowScheduleTrigger{
		Calendar: input.Calendar,
		TimeZone: input.TimeZone,
		Input:    value,
		Overlap:  sessionv3Runtime.WorkflowTriggerOverlap(input.Overlap),
	}
	if input.Interval != "" {
		interval, intervalErr := sessionv3Runtime.ParseWorkflowDuration(
			input.Interval,
		)
		if intervalErr != nil {
			return sessionv3Runtime.WorkflowScheduleTrigger{}, intervalErr
		}
		output.Interval = interval
	}
	if input.Missed != nil {
		output.Missed = sessionv3Runtime.WorkflowTriggerMissed{
			Mode: sessionv3Runtime.WorkflowTriggerMissedMode(
				input.Missed.Mode,
			),
		}
		if input.Missed.MaxOccurrences != nil {
			output.Missed.MaxOccurrences = *input.Missed.MaxOccurrences
		}
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowScheduleTrigger{}, err
	}
	return output, nil
}
