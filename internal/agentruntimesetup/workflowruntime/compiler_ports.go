package workflowruntime

import (
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

func compileWorkflowPorts(
	document workflowv1.WorkflowDocument,
) ([]sessionv3Runtime.WorkflowPort, error) {
	names := make([]string, 0, len(document.Ports))
	for name := range document.Ports {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make([]sessionv3Runtime.WorkflowPort, 0, len(names))
	for _, name := range names {
		value := document.Ports[name]
		port := sessionv3Runtime.WorkflowPort{
			Name:    name,
			Channel: value.Channel,
			Access:  sessionv3Runtime.WorkflowPortAccess(value.Access),
		}
		output = append(output, port)
	}
	return output, nil
}

func compileWorkflowPortBindings(
	values map[string]workflowv1.PortBinding,
) ([]sessionv3Runtime.WorkflowPortBinding, error) {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make(
		[]sessionv3Runtime.WorkflowPortBinding,
		0,
		len(names),
	)
	for _, name := range names {
		value := values[name]
		binding := sessionv3Runtime.WorkflowPortBinding{
			Name:    name,
			Channel: value.Channel,
			Access:  sessionv3Runtime.WorkflowPortAccess(value.Access),
		}
		output = append(output, binding)
	}
	return output, nil
}

func compileWorkflowContextRequirements(
	document workflowv1.WorkflowDocument,
	values map[string]workflowv1.Schema,
) ([]sessionv3Runtime.WorkflowContextRequirement, error) {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make(
		[]sessionv3Runtime.WorkflowContextRequirement,
		0,
		len(names),
	)
	for _, name := range names {
		schema, err := compileWorkflowSchema(document, values[name])
		if err != nil {
			return nil, fmt.Errorf(
				"workflow callable context %q: %w",
				name,
				err,
			)
		}
		value := sessionv3Runtime.WorkflowContextRequirement{
			Name:   name,
			Schema: schema,
		}
		output = append(output, value)
	}
	return output, nil
}

func compileWorkflowCallablePorts(
	document workflowv1.WorkflowDocument,
	values map[string]workflowv1.CallablePort,
) ([]sessionv3Runtime.WorkflowCallablePort, error) {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make(
		[]sessionv3Runtime.WorkflowCallablePort,
		0,
		len(names),
	)
	for _, name := range names {
		source := values[name]
		schema, err := compileWorkflowSchema(document, source.Schema)
		if err != nil {
			return nil, fmt.Errorf(
				"workflow callable port %q: %w",
				name,
				err,
			)
		}
		value := sessionv3Runtime.WorkflowCallablePort{
			Name:      name,
			Direction: sessionv3Runtime.WorkflowPortDirection(source.Direction),
			Schema:    schema,
			Stream:    source.Stream,
		}
		output = append(output, value)
	}
	return output, nil
}

func compileWorkflowOptionalExecutionBudget(
	input *workflowv1.Limits,
) (*sessionv3Runtime.ExecutionBudget, error) {
	if input == nil {
		return nil, nil
	}
	value, err := compileWorkflowExecutionBudget(*input)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
