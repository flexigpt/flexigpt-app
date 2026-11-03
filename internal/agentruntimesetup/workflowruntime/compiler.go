package workflowruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

const maxWorkflowDefinitionBytes = 16 << 20

func compileWorkflowProgram(
	document workflowv1.WorkflowDocument,
	input sessionv3Runtime.CanonicalValue,
) (sessionv3Runtime.WorkflowProgram, error) {
	inputSchema, err := compileWorkflowSchema(
		document,
		document.InputSchema,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	outputSchema, err := compileWorkflowSchema(
		document,
		document.OutputSchema,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	root, err := compileWorkflowBlock(document, document.Block)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}

	output := sessionv3Runtime.WorkflowProgram{
		Version:      sessionv3Runtime.WorkflowInterpreterVersion,
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
		Input:        input.Clone(),
		Root:         root,
		Targets:      make(map[string]sessionv3Runtime.WorkflowTarget),
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	if err := output.ValidateInput(); err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	return output, nil
}

func compileWorkflowBlock(
	document workflowv1.WorkflowDocument,
	block workflowv1.Block,
) (sessionv3Runtime.WorkflowBlock, error) {
	if len(block.Listen) != 0 ||
		len(block.Catch) != 0 ||
		block.Finally != nil {
		return sessionv3Runtime.WorkflowBlock{}, errors.New(
			"workflow listeners, catch handlers, and finally blocks are not executable by runtime v3 yet",
		)
	}

	output, err := compileWorkflowValue(block.Output)
	if err != nil {
		return sessionv3Runtime.WorkflowBlock{}, err
	}

	result := sessionv3Runtime.WorkflowBlock{
		Steps:  make([]sessionv3Runtime.WorkflowStep, 0, len(block.Steps)),
		Output: output,
	}
	if block.State != nil {
		schema, err := compileWorkflowSchema(
			document,
			block.State.Schema,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowBlock{}, err
		}
		initial, err := compileWorkflowValue(block.State.Initial)
		if err != nil {
			return sessionv3Runtime.WorkflowBlock{}, err
		}
		result.State = &sessionv3Runtime.WorkflowStateSpec{
			Schema:  schema,
			Initial: initial,
		}
	}

	for index := range block.Steps {
		step, err := compileWorkflowStep(document, block.Steps[index])
		if err != nil {
			return sessionv3Runtime.WorkflowBlock{}, fmt.Errorf(
				"compile Workflow step %q: %w",
				block.Steps[index].ID,
				err,
			)
		}
		result.Steps = append(result.Steps, step)
	}
	return result, nil
}

func compileWorkflowStep(
	document workflowv1.WorkflowDocument,
	input workflowv1.Step,
) (sessionv3Runtime.WorkflowStep, error) {
	if input.Timeout != "" ||
		input.Retry != nil ||
		input.Effect != nil ||
		len(input.Claims) != 0 ||
		input.Limits != nil {
		return sessionv3Runtime.WorkflowStep{}, fmt.Errorf(
			"workflow step %q uses retry, effect, claims, timeout, or limits that runtime v3 does not execute yet",
			input.ID,
		)
	}

	output := sessionv3Runtime.WorkflowStep{ID: input.ID}
	if len(input.When) != 0 {
		when, err := compileWorkflowValue(input.When)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.When = &when
	}
	if len(input.OutputSchema) != 0 {
		schema, err := compileWorkflowSchema(
			document,
			input.OutputSchema,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.OutputSchema = schema
	}

	switch {
	case input.Call != nil:
		call, err := compileWorkflowCall(document, *input.Call)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Call = &call

	case input.Transform != nil:
		value, err := compileWorkflowValue(input.Transform.Value)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		transform := sessionv3Runtime.WorkflowTransform{
			Value: value,
		}
		if len(input.Transform.OutputSchema) != 0 {
			schema, err := compileWorkflowSchema(
				document,
				input.Transform.OutputSchema,
			)
			if err != nil {
				return sessionv3Runtime.WorkflowStep{}, err
			}
			transform.OutputSchema = schema
		}
		output.Transform = &transform

	case input.Set != nil:
		values := make(
			[]sessionv3Runtime.WorkflowValuePair,
			0,
			len(input.Set.Values),
		)
		keys := make([]string, 0, len(input.Set.Values))
		for key := range input.Set.Values {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value, err := compileWorkflowValue(input.Set.Values[key])
			if err != nil {
				return sessionv3Runtime.WorkflowStep{}, err
			}
			values = append(values, sessionv3Runtime.WorkflowValuePair{
				Key:   key,
				Value: value,
			})
		}
		output.Set = &sessionv3Runtime.WorkflowSet{
			Values: values,
			Remove: append([]string(nil), input.Set.Remove...),
		}

	case input.Return != nil:
		value, err := compileWorkflowValue(input.Return.Value)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Return = &sessionv3Runtime.WorkflowReturn{
			Value: value,
		}

	case input.Fail != nil:
		message, err := compileWorkflowValue(input.Fail.Message)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Fail = &sessionv3Runtime.WorkflowFail{
			Code:    input.Fail.Code,
			Message: message,
		}

	default:
		return sessionv3Runtime.WorkflowStep{}, errors.New(
			"workflow operation is not executable by runtime v3 yet",
		)
	}

	return output, nil
}

func compileWorkflowCall(
	document workflowv1.WorkflowDocument,
	input workflowv1.Call,
) (sessionv3Runtime.WorkflowCall, error) {
	if input.Mode != "" && input.Mode != "invoke" ||
		input.Lifetime != "" ||
		input.Ownership != nil ||
		len(input.Conversation) != 0 ||
		input.OnCompletion != "" ||
		len(input.Context) != 0 ||
		len(input.Ports) != 0 ||
		input.Interface != nil {
		return sessionv3Runtime.WorkflowCall{}, errors.New(
			"workflow call mode, ownership, context, ports, and inline interfaces are not executable by runtime v3 yet",
		)
	}

	target, err := compileWorkflowValue(input.Target)
	if err != nil {
		return sessionv3Runtime.WorkflowCall{}, err
	}
	callInput, err := compileWorkflowValue(input.Input)
	if err != nil {
		return sessionv3Runtime.WorkflowCall{}, err
	}

	output := sessionv3Runtime.WorkflowCall{
		Target: target,
		Input:  callInput,
	}
	if input.Allowed == "" {
		return output, nil
	}

	set, found := document.TargetSets[input.Allowed]
	if !found {
		return sessionv3Runtime.WorkflowCall{}, fmt.Errorf(
			"workflow target set %q is unavailable",
			input.Allowed,
		)
	}
	if set.Resolved != nil {
		return sessionv3Runtime.WorkflowCall{}, errors.New(
			"workflow dynamic target sets are not executable by runtime v3 yet",
		)
	}
	output.AllowedTargets = append([]string(nil), set.Members...)
	return output, nil
}

func compileWorkflowSchema(
	document workflowv1.WorkflowDocument,
	schema workflowv1.Schema,
) (sessionv3Runtime.CanonicalValue, error) {
	if len(schema) == 0 {
		return nil, nil
	}

	raw, err := document.CanonicalSchema(schema)
	if err != nil {
		return nil, err
	}
	return sessionv3Runtime.CanonicalValueFrom(raw)
}

func compileWorkflowValue(
	input workflowv1.Value,
) (sessionv3Runtime.WorkflowValue, error) {
	if len(bytes.TrimSpace(input)) == 0 {
		return sessionv3Runtime.WorkflowValue{
			Kind:    sessionv3Runtime.WorkflowValueLiteral,
			Literal: sessionv3Runtime.MustCanonicalValue(nil),
		}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return sessionv3Runtime.WorkflowValue{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return sessionv3Runtime.WorkflowValue{}, errors.New(
				"workflow value contains multiple JSON values",
			)
		}
		return sessionv3Runtime.WorkflowValue{}, err
	}

	return compileWorkflowValueNode(value)
}

func compileWorkflowValueNode(
	value any,
) (sessionv3Runtime.WorkflowValue, error) {
	switch typed := value.(type) {
	case map[string]any:
		if expression, found := typed["$expr"]; found {
			if len(typed) != 1 {
				return sessionv3Runtime.WorkflowValue{}, errors.New(
					"workflow expression has fields other than $expr",
				)
			}
			source, ok := expression.(string)
			if !ok {
				return sessionv3Runtime.WorkflowValue{}, errors.New(
					"workflow expression source is not a string",
				)
			}
			return sessionv3Runtime.WorkflowValue{
				Kind:       sessionv3Runtime.WorkflowValueExpression,
				Expression: source,
			}, nil
		}

		if literal, found := typed["$literal"]; found {
			if len(typed) != 1 {
				return sessionv3Runtime.WorkflowValue{}, errors.New(
					"workflow literal has fields other than $literal",
				)
			}
			raw, err := jsonutil.MarshalCanonicalObject(
				literal,
				maxWorkflowDefinitionBytes,
			)
			if err != nil {
				return sessionv3Runtime.WorkflowValue{}, err
			}
			value, err := sessionv3Runtime.CanonicalValueFrom(raw)
			if err != nil {
				return sessionv3Runtime.WorkflowValue{}, err
			}
			return sessionv3Runtime.WorkflowValue{
				Kind:    sessionv3Runtime.WorkflowValueLiteral,
				Literal: value,
			}, nil
		}

		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		fields := make(
			[]sessionv3Runtime.WorkflowValuePair,
			0,
			len(keys),
		)
		for _, key := range keys {
			child, err := compileWorkflowValueNode(typed[key])
			if err != nil {
				return sessionv3Runtime.WorkflowValue{}, err
			}
			fields = append(fields, sessionv3Runtime.WorkflowValuePair{
				Key:   key,
				Value: child,
			})
		}
		return sessionv3Runtime.WorkflowValue{
			Kind:   sessionv3Runtime.WorkflowValueObject,
			Object: fields,
		}, nil

	case []any:
		values := make(
			[]sessionv3Runtime.WorkflowValue,
			len(typed),
		)
		for index := range typed {
			child, err := compileWorkflowValueNode(typed[index])
			if err != nil {
				return sessionv3Runtime.WorkflowValue{}, err
			}
			values[index] = child
		}
		return sessionv3Runtime.WorkflowValue{
			Kind:  sessionv3Runtime.WorkflowValueArray,
			Array: values,
		}, nil

	default:
		raw, err := jsonutil.MarshalCanonicalObject(
			typed,
			maxWorkflowDefinitionBytes,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowValue{}, err
		}
		literal, err := sessionv3Runtime.CanonicalValueFrom(raw)
		if err != nil {
			return sessionv3Runtime.WorkflowValue{}, err
		}
		return sessionv3Runtime.WorkflowValue{
			Kind:    sessionv3Runtime.WorkflowValueLiteral,
			Literal: literal,
		}, nil
	}
}
