package workflowruntime

import (
	"errors"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowStructuredOperation(
	document workflowv1.WorkflowDocument,
	input workflowv1.Step,
	output *sessionv3Runtime.WorkflowStep,
) error {
	if output == nil {
		return errors.New("workflow structured step output is unavailable")
	}

	switch {
	case input.Scope != nil:
		block, err := compileWorkflowBlock(document, *input.Scope)
		if err != nil {
			return err
		}
		output.Scope = &block
		return nil

	case input.Choose != nil:
		value, err := compileWorkflowChoose(document, *input.Choose)
		if err != nil {
			return err
		}
		output.Choose = &value
		return nil

	case input.Parallel != nil:
		value, err := compileWorkflowParallel(document, *input.Parallel)
		if err != nil {
			return err
		}
		output.Parallel = &value
		return nil

	case input.Foreach != nil:
		value, err := compileWorkflowForeach(document, *input.Foreach)
		if err != nil {
			return err
		}
		output.Foreach = &value
		return nil

	case input.Repeat != nil:
		value, err := compileWorkflowRepeat(document, *input.Repeat)
		if err != nil {
			return err
		}
		output.Repeat = &value
		return nil

	case input.Break != nil:
		output.Break = true
		return nil

	case input.Continue != nil:
		output.Continue = true
		return nil

	default:
		return errors.New("workflow structured operation is unavailable")
	}
}

func compileWorkflowChoose(
	document workflowv1.WorkflowDocument,
	input workflowv1.Choose,
) (sessionv3Runtime.WorkflowChoose, error) {
	output := sessionv3Runtime.WorkflowChoose{
		Cases: make([]sessionv3Runtime.WorkflowCase, 0, len(input.Cases)),
	}
	for index := range input.Cases {
		when, err := compileWorkflowValue(input.Cases[index].When)
		if err != nil {
			return sessionv3Runtime.WorkflowChoose{}, err
		}
		block, err := compileWorkflowBlock(document, input.Cases[index].Do)
		if err != nil {
			return sessionv3Runtime.WorkflowChoose{}, err
		}
		output.Cases = append(output.Cases, sessionv3Runtime.WorkflowCase{
			When: when,
			Do:   block,
		})
	}
	if input.Default != nil {
		block, err := compileWorkflowBlock(document, *input.Default)
		if err != nil {
			return sessionv3Runtime.WorkflowChoose{}, err
		}
		output.Default = &block
	}
	return output, nil
}

func compileWorkflowCompletion(
	input *workflowv1.Completion,
) sessionv3Runtime.WorkflowCompletion {
	if input == nil {
		return sessionv3Runtime.WorkflowCompletion{Mode: "all"}
	}

	value := sessionv3Runtime.WorkflowCompletion{
		Mode: input.Mode,
	}
	if input.Quorum != nil {
		value.Quorum = *input.Quorum
	}
	return value
}

func compileWorkflowParallel(
	document workflowv1.WorkflowDocument,
	input workflowv1.Parallel,
) (sessionv3Runtime.WorkflowParallel, error) {
	names := make([]string, 0, len(input.Branches))
	for name := range input.Branches {
		names = append(names, name)
	}
	sort.Strings(names)

	output := sessionv3Runtime.WorkflowParallel{
		Completion: compileWorkflowCompletion(input.Completion),
		Errors:     input.Errors,
		Remaining:  input.Remaining,
		Branches:   make([]sessionv3Runtime.WorkflowBranch, 0, len(names)),
	}
	for _, name := range names {
		branch := input.Branches[name]
		value := sessionv3Runtime.WorkflowBranch{
			Name: name,
		}
		if len(branch.When) != 0 {
			when, err := compileWorkflowValue(branch.When)
			if err != nil {
				return sessionv3Runtime.WorkflowParallel{}, err
			}
			value.When = &when
		}
		block, err := compileWorkflowBlock(document, branch.Do)
		if err != nil {
			return sessionv3Runtime.WorkflowParallel{}, err
		}
		value.Do = block
		output.Branches = append(output.Branches, value)
	}
	return output, nil
}

func compileWorkflowForeach(
	document workflowv1.WorkflowDocument,
	input workflowv1.Foreach,
) (sessionv3Runtime.WorkflowForeach, error) {
	items, err := compileWorkflowValue(input.Items)
	if err != nil {
		return sessionv3Runtime.WorkflowForeach{}, err
	}
	itemSchema, err := compileWorkflowSchema(document, input.ItemSchema)
	if err != nil {
		return sessionv3Runtime.WorkflowForeach{}, err
	}
	block, err := compileWorkflowBlock(document, input.Do)
	if err != nil {
		return sessionv3Runtime.WorkflowForeach{}, err
	}

	output := sessionv3Runtime.WorkflowForeach{
		Items:      items,
		ItemSchema: itemSchema,
		Completion: compileWorkflowCompletion(input.Completion),
		Errors:     input.Errors,
		Remaining:  input.Remaining,
		Do:         block,
	}
	if input.Concurrency != nil {
		output.Concurrency = *input.Concurrency
	}
	if input.MaxItems != nil {
		output.MaxItems = *input.MaxItems
	}
	return output, nil
}

func compileWorkflowRepeatState(
	document workflowv1.WorkflowDocument,
	input *workflowv1.State,
) (*sessionv3Runtime.WorkflowStateSpec, error) {
	if input == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	schema, err := compileWorkflowSchema(document, input.Schema)
	if err != nil {
		return nil, err
	}
	initial, err := compileWorkflowValue(input.Initial)
	if err != nil {
		return nil, err
	}
	return &sessionv3Runtime.WorkflowStateSpec{
		Schema:  schema,
		Initial: initial,
	}, nil
}

func compileWorkflowRepeat(
	document workflowv1.WorkflowDocument,
	input workflowv1.Repeat,
) (sessionv3Runtime.WorkflowRepeat, error) {
	block, err := compileWorkflowBlock(document, input.Do)
	if err != nil {
		return sessionv3Runtime.WorkflowRepeat{}, err
	}
	state, err := compileWorkflowRepeatState(document, input.State)
	if err != nil {
		return sessionv3Runtime.WorkflowRepeat{}, err
	}

	output := sessionv3Runtime.WorkflowRepeat{
		Lifetime:    input.Lifetime,
		State:       state,
		OnExhausted: input.OnExhausted,
		Do:          block,
	}
	if input.MaxIterations != nil {
		output.MaxIterations = *input.MaxIterations
	}
	if len(input.Count) != 0 {
		value, err := compileWorkflowValue(input.Count)
		if err != nil {
			return sessionv3Runtime.WorkflowRepeat{}, err
		}
		output.Count = &value
	}
	if len(input.While) != 0 {
		value, err := compileWorkflowValue(input.While)
		if err != nil {
			return sessionv3Runtime.WorkflowRepeat{}, err
		}
		output.While = &value
	}
	if len(input.Until) != 0 {
		value, err := compileWorkflowValue(input.Until)
		if err != nil {
			return sessionv3Runtime.WorkflowRepeat{}, err
		}
		output.Until = &value
	}

	if input.Timeout != "" {
		return sessionv3Runtime.WorkflowRepeat{}, unsupportedWorkflowOperation(
			"repeat",
			"repeat timeout",
		)
	}
	return output, nil
}
