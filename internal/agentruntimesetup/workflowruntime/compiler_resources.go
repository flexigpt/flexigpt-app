package workflowruntime

import (
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowResources(
	document workflowv1.WorkflowDocument,
) ([]sessionv3Runtime.WorkflowResourceDefinition, error) {
	if len(document.Resources) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(document.Resources))
	for name := range document.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make(
		[]sessionv3Runtime.WorkflowResourceDefinition,
		0,
		len(names),
	)
	for _, name := range names {
		source := document.Resources[name]
		value := sessionv3Runtime.WorkflowResourceDefinition{
			Name:  name,
			Keyed: source.Keyed,
		}
		if source.Concurrency != nil {
			value.Concurrency = *source.Concurrency
		}
		if source.Rate != nil {
			per, err := sessionv3Runtime.ParseWorkflowDuration(
				source.Rate.Per,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"workflow resource %q rate: %w",
					name,
					err,
				)
			}
			rate := &sessionv3Runtime.WorkflowResourceRate{
				Units: source.Rate.Units,
				Per:   per,
			}
			if source.Rate.Burst != nil {
				rate.Burst = *source.Rate.Burst
			}
			value.Rate = rate
		}
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf(
				"workflow resource %q: %w",
				name,
				err,
			)
		}
		output = append(output, value)
	}
	return output, nil
}

func compileWorkflowResourceClaims(
	values []workflowv1.Claim,
) ([]sessionv3Runtime.WorkflowResourceClaim, error) {
	if len(values) == 0 {
		return nil, nil
	}

	output := make(
		[]sessionv3Runtime.WorkflowResourceClaim,
		0,
		len(values),
	)
	seen := make(map[string]struct{}, len(values))
	for _, source := range values {
		if _, duplicate := seen[source.Resource]; duplicate {
			return nil, fmt.Errorf(
				"workflow resource claim %q is duplicated",
				source.Resource,
			)
		}
		seen[source.Resource] = struct{}{}

		claim := sessionv3Runtime.WorkflowResourceClaim{
			Resource: source.Resource,
			Units:    1,
		}
		if source.Units != nil {
			claim.Units = *source.Units
		}
		if len(source.Key) != 0 {
			key, err := compileWorkflowValue(source.Key)
			if err != nil {
				return nil, err
			}
			claim.Key = &key
		}
		if err := claim.Validate(); err != nil {
			return nil, err
		}
		output = append(output, claim)
	}
	return output, nil
}
