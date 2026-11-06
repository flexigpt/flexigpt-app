package workflowruntime

import (
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowChannelsAndTargetSets(
	document workflowv1.WorkflowDocument,
) (
	[]sessionv3Runtime.WorkflowChannel,
	map[string]sessionv3Runtime.WorkflowTargetSet,
	error,
) {
	channels, err := compileWorkflowChannels(document)
	if err != nil {
		return nil, nil, err
	}
	targetSets, err := compileWorkflowTargetSets(document)
	if err != nil {
		return nil, nil, err
	}
	return channels, targetSets, nil
}

func compileWorkflowTargetSets(
	document workflowv1.WorkflowDocument,
) (map[string]sessionv3Runtime.WorkflowTargetSet, error) {
	if len(document.TargetSets) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(document.TargetSets))
	for name := range document.TargetSets {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make(
		map[string]sessionv3Runtime.WorkflowTargetSet,
		len(names),
	)
	for _, name := range names {
		input := document.TargetSets[name]
		value := sessionv3Runtime.WorkflowTargetSet{
			Members: append([]string(nil), input.Members...),
		}
		contract, err := compileWorkflowCallableInterface(
			document,
			input.Interface,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"workflow target set %q interface: %w",
				name,
				err,
			)
		}
		value.Contract = contract

		if input.Resolved != nil {
			resolvedValue, valueErr := compileWorkflowValue(
				input.Resolved.Value,
			)
			if valueErr != nil {
				return nil, fmt.Errorf(
					"workflow target set %q resolver: %w",
					name,
					valueErr,
				)
			}
			resolvedContract, contractErr := compileWorkflowCallableInterface(
				document,
				&input.Resolved.Interface,
			)
			if contractErr != nil {
				return nil, fmt.Errorf(
					"workflow target set %q resolver interface: %w",
					name,
					contractErr,
				)
			}
			if resolvedContract == nil {
				return nil, fmt.Errorf(
					"workflow target set %q resolver has no interface",
					name,
				)
			}
			resolved := &sessionv3Runtime.WorkflowResolvedTargetSet{
				Value:    resolvedValue,
				Contract: *resolvedContract,
			}
			if len(input.Resolved.Constraints) != 0 {
				constraints, constraintsErr := compileWorkflowValue(input.Resolved.Constraints)
				if constraintsErr != nil {
					return nil, fmt.Errorf(
						"workflow target set %q constraints: %w",
						name,
						constraintsErr,
					)
				}
				resolved.Constraints = &constraints
			}
			value.Resolved = resolved
		}

		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf(
				"workflow target set %q: %w",
				name,
				err,
			)
		}
		output[name] = value
	}
	return output, nil
}
