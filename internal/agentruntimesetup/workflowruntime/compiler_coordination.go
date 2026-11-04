package workflowruntime

import (
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowEvents(
	document workflowv1.WorkflowDocument,
) ([]sessionv3Runtime.WorkflowEventContract, error) {
	names := make([]string, 0, len(document.Events))
	for name := range document.Events {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make([]sessionv3Runtime.WorkflowEventContract, 0, len(names))
	for _, name := range names {
		value := document.Events[name]
		schema, err := compileWorkflowSchema(document, value.Schema)
		if err != nil {
			return nil, err
		}
		correlation, err := compileWorkflowSchema(
			document,
			value.CorrelationSchema,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, sessionv3Runtime.WorkflowEventContract{
			Name:              name,
			Source:            value.Source,
			Schema:            schema,
			CorrelationSchema: correlation,
		})
	}
	return output, nil
}

func compileWorkflowChannels(
	document workflowv1.WorkflowDocument,
) ([]sessionv3Runtime.WorkflowChannel, error) {
	names := make([]string, 0, len(document.Channels))
	for name := range document.Channels {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make([]sessionv3Runtime.WorkflowChannel, 0, len(names))
	for _, name := range names {
		value := document.Channels[name]
		schema, err := compileWorkflowSchema(document, value.Schema)
		if err != nil {
			return nil, err
		}
		channel := sessionv3Runtime.WorkflowChannel{
			Name:    name,
			Schema:  schema,
			Readers: append([]string(nil), value.Readers...),
			Writers: append([]string(nil), value.Writers...),
		}
		if value.MaxMessages != nil {
			channel.MaxMessages = *value.MaxMessages
		}
		if value.MaxBytes != nil {
			channel.MaxBytes = *value.MaxBytes
		}
		output = append(output, channel)
	}
	return output, nil
}

func compileWorkflowSubscriptions(
	document workflowv1.WorkflowDocument,
	values map[string]workflowv1.Subscription,
) ([]sessionv3Runtime.WorkflowSubscription, error) {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make([]sessionv3Runtime.WorkflowSubscription, 0, len(names))
	for _, name := range names {
		value := values[name]
		if _, found := document.Events[value.Event]; !found {
			return nil, fmt.Errorf(
				"workflow subscription %q references unknown event %q",
				name,
				value.Event,
			)
		}
		if value.Retention != "" ||
			value.Capacity != nil ||
			value.MaxBytes != nil {
			return nil, unsupportedWorkflowOperation(
				"listen/"+name,
				"subscription-specific retention limits",
			)
		}

		subscription := sessionv3Runtime.WorkflowSubscription{
			Name:  name,
			Event: value.Event,
		}
		if len(value.Correlation) != 0 {
			correlation, err := compileWorkflowValue(value.Correlation)
			if err != nil {
				return nil, err
			}
			subscription.Correlation = &correlation
		}
		if len(value.Filter) != 0 {
			filter, err := compileWorkflowValue(value.Filter)
			if err != nil {
				return nil, err
			}
			subscription.Filter = &filter
		}
		output = append(output, subscription)
	}
	return output, nil
}

func compileWorkflowEmit(
	input workflowv1.Emit,
) (sessionv3Runtime.WorkflowEmit, error) {
	data, err := compileWorkflowValue(input.Data)
	if err != nil {
		return sessionv3Runtime.WorkflowEmit{}, err
	}
	output := sessionv3Runtime.WorkflowEmit{
		Destination: sessionv3Runtime.EmitDestination{
			Channel: input.To.Channel,
			Event:   input.To.Event,
		},
		Kind: input.Kind,
		Data: data,
	}
	if len(input.Correlation) != 0 {
		value, err := compileWorkflowValue(input.Correlation)
		if err != nil {
			return sessionv3Runtime.WorkflowEmit{}, err
		}
		output.Correlation = &value
	}
	if len(input.DeduplicationKey) != 0 {
		value, err := compileWorkflowValue(input.DeduplicationKey)
		if err != nil {
			return sessionv3Runtime.WorkflowEmit{}, err
		}
		output.DeduplicationKey = &value
	}
	return output, nil
}

func compileWorkflowRetry(
	input workflowv1.Retry,
) (sessionv3Runtime.RetryPolicy, error) {
	output := sessionv3Runtime.RetryPolicy{
		MaxAttempts: input.MaxAttempts,
	}
	if input.Match != nil {
		output.Match.Codes = append([]string(nil), input.Match.Codes...)
		output.Match.Retryable = input.Match.Retryable
		for _, category := range input.Match.Categories {
			value, err := compileWorkflowErrorCategory(category)
			if err != nil {
				return sessionv3Runtime.RetryPolicy{}, err
			}
			output.Match.Categories = append(output.Match.Categories, value)
		}
	}
	if input.Backoff != nil {
		initial, err := sessionv3Runtime.ParseWorkflowDuration(input.Backoff.Initial)
		if err != nil {
			return sessionv3Runtime.RetryPolicy{}, err
		}
		output.Backoff.Initial = initial
		if input.Backoff.Maximum != "" {
			maximum, err := sessionv3Runtime.ParseWorkflowDuration(input.Backoff.Maximum)
			if err != nil {
				return sessionv3Runtime.RetryPolicy{}, err
			}
			output.Backoff.Maximum = maximum
		}
		if input.Backoff.Multiplier != nil {
			output.Backoff.Multiplier = *input.Backoff.Multiplier
		}
		if input.Backoff.Jitter != nil {
			output.Backoff.Jitter = *input.Backoff.Jitter
		}
	} else if input.MaxAttempts > 1 {
		output.Backoff.Initial = sessionv3Runtime.WorkflowDuration("PT1S")
	}
	if input.PerAttemptTimeout != "" {
		value, err := sessionv3Runtime.ParseWorkflowDuration(
			input.PerAttemptTimeout,
		)
		if err != nil {
			return sessionv3Runtime.RetryPolicy{}, err
		}
		output.PerAttemptTimeout = value
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.RetryPolicy{}, err
	}
	return output, nil
}

func compileWorkflowErrorCategory(
	value string,
) (sessionv3Runtime.ErrorCategory, error) {
	category := sessionv3Runtime.ErrorCategory(value)
	switch category {
	case sessionv3Runtime.ErrorCategoryValidation,
		sessionv3Runtime.ErrorCategoryRouting,
		sessionv3Runtime.ErrorCategoryExecution,
		sessionv3Runtime.ErrorCategoryTransient,
		sessionv3Runtime.ErrorCategoryTimeout,
		sessionv3Runtime.ErrorCategoryAuthorization,
		sessionv3Runtime.ErrorCategoryLimit,
		sessionv3Runtime.ErrorCategoryCancellation,
		sessionv3Runtime.ErrorCategoryUncertainty:
		return category, nil
	default:
		return "", fmt.Errorf("unsupported workflow retry error category %q", value)
	}
}
