package workflowruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

var ErrUnsupportedOperation = errors.New(
	"workflow operation is unsupported by the typed runtime",
)

// UnsupportedOperationError is returned only during Workflow admission. It
// prevents a syntactically valid declaration from being accepted and later
// ignored or changed by Session Runtime.
type UnsupportedOperationError struct {
	Path      string
	Operation string
}

func (e *UnsupportedOperationError) Error() string {
	if e == nil {
		return ErrUnsupportedOperation.Error()
	}
	return fmt.Sprintf("%s: %s", ErrUnsupportedOperation, e.Operation+" at "+e.Path)
}

func (*UnsupportedOperationError) Unwrap() error {
	return ErrUnsupportedOperation
}

func compileWorkflowProgram(
	document workflowv1.WorkflowDocument,
	input sessionv3Runtime.CanonicalValue,
) (sessionv3Runtime.WorkflowProgram, error) {
	return compileWorkflowProgramWithInput(document, input, true)
}

func compileWorkflowProgramTemplate(
	document workflowv1.WorkflowDocument,
) (sessionv3Runtime.WorkflowProgram, error) {
	return compileWorkflowProgramWithInput(document, nil, false)
}

func compileWorkflowProgramWithInput(
	document workflowv1.WorkflowDocument,
	input sessionv3Runtime.CanonicalValue,
	inputBound bool,
) (sessionv3Runtime.WorkflowProgram, error) {
	if err := rejectUnsupportedWorkflowDocument(document); err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	events, err := compileWorkflowEvents(document)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	channels, err := compileWorkflowChannels(document)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
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
		InputBound:   inputBound,
		Root:         root,
		Targets:      make(map[string]sessionv3Runtime.WorkflowTarget),
		Events:       events,
		Channels:     channels,
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	if inputBound {
		if err := output.ValidateInput(); err != nil {
			return sessionv3Runtime.WorkflowProgram{}, err
		}
	}
	return output, nil
}

func rejectUnsupportedWorkflowDocument(
	document workflowv1.WorkflowDocument,
) error {
	switch {
	case len(document.Ports) != 0:
		return unsupportedWorkflowOperation("workflow", "ports")
	case len(document.Triggers) != 0:
		return unsupportedWorkflowOperation("workflow", "triggers")
	case len(document.Resources) != 0:
		return unsupportedWorkflowOperation("workflow", "resources")
	case document.Limits != nil:
		return unsupportedWorkflowOperation("workflow", "limits")
	}

	for name, targetSet := range document.TargetSets {
		if targetSet.Resolved != nil {
			return unsupportedWorkflowOperation(
				"targetSets/"+name,
				"resolver-backed dynamic target set",
			)
		}
	}
	return nil
}

func unsupportedWorkflowOperation(path, operation string) error {
	return &UnsupportedOperationError{
		Path:      path,
		Operation: operation,
	}
}

func compileWorkflowBlock(
	document workflowv1.WorkflowDocument,
	block workflowv1.Block,
) (sessionv3Runtime.WorkflowBlock, error) {
	if len(block.Catch) != 0 {
		return sessionv3Runtime.WorkflowBlock{}, unsupportedWorkflowOperation(
			"block",
			"catch handlers",
		)
	}
	if block.Finally != nil {
		return sessionv3Runtime.WorkflowBlock{}, unsupportedWorkflowOperation(
			"block",
			"finally block",
		)
	}
	output, err := compileWorkflowBlockCore(document, block)
	if err != nil {
		return sessionv3Runtime.WorkflowBlock{}, err
	}
	output.Listen, err = compileWorkflowSubscriptions(document, block.Listen)
	if err != nil {
		return sessionv3Runtime.WorkflowBlock{}, err
	}
	return output, nil
}

func compileWorkflowBlockCore(
	document workflowv1.WorkflowDocument,
	block workflowv1.Block,
) (sessionv3Runtime.WorkflowBlock, error) {
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
	switch {
	case len(input.Claims) != 0:
		return sessionv3Runtime.WorkflowStep{}, unsupportedWorkflowOperation(
			"steps/"+input.ID,
			"resource claims",
		)
	case input.Limits != nil:
		return sessionv3Runtime.WorkflowStep{}, unsupportedWorkflowOperation(
			"steps/"+input.ID,
			"limits",
		)
	}

	output := sessionv3Runtime.WorkflowStep{ID: input.ID}
	if input.Timeout != "" {
		timeout, err := sessionv3Runtime.ParseWorkflowDuration(input.Timeout)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Timeout = timeout
	}
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
	if input.Retry != nil {
		if !workflowStepSupportsRetry(document, input) {
			return sessionv3Runtime.WorkflowStep{}, unsupportedWorkflowOperation(
				"steps/"+input.ID,
				"retry for this operation or target kind",
			)
		}
		retry, err := compileWorkflowRetry(*input.Retry)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Retry = &retry
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

	case input.Scope != nil,
		input.Choose != nil,
		input.Parallel != nil,
		input.Foreach != nil,
		input.Repeat != nil,
		input.Break != nil,
		input.Continue != nil:
		if err := compileWorkflowStructuredOperation(
			document,
			input,
			&output,
		); err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}

	case input.Return != nil:
		value, err := compileWorkflowValue(input.Return.Value)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Return = &sessionv3Runtime.WorkflowReturn{
			To:    sessionv3Runtime.WorkflowReturnDestination(input.Return.To),
			Value: value,
		}

	case input.Fail != nil:
		message, err := compileWorkflowValue(input.Fail.Message)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Fail = &sessionv3Runtime.WorkflowFail{
			Code:     input.Fail.Code,
			Category: sessionv3Runtime.ErrorCategory(input.Fail.Category),
			Message:  message,
		}
		if input.Fail.Retryable != nil {
			output.Fail.Retryable = *input.Fail.Retryable
		}
		if len(input.Fail.Details) != 0 {
			details, err := compileWorkflowValue(input.Fail.Details)
			if err != nil {
				return sessionv3Runtime.WorkflowStep{}, err
			}
			output.Fail.Details = &details
		}
	case input.Wait != nil:
		wait, err := compileWorkflowWait(document, *input.Wait)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Wait = &wait

	case input.Emit != nil:
		emit, err := compileWorkflowEmit(*input.Emit)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Emit = &emit

	case input.Cancel != nil:
		handle, err := compileWorkflowValue(input.Cancel.Handle)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		cancel := sessionv3Runtime.WorkflowCancel{
			Handle: handle,
		}
		if len(input.Cancel.Reason) != 0 {
			reason, reasonErr := compileWorkflowValue(input.Cancel.Reason)
			if reasonErr != nil {
				return sessionv3Runtime.WorkflowStep{}, reasonErr
			}
			cancel.Reason = &reason
		}
		output.Cancel = &cancel

	case input.Saga != nil:
		return sessionv3Runtime.WorkflowStep{}, unsupportedWorkflowOperation(
			"steps/"+input.ID,
			"saga",
		)

	default:
		return sessionv3Runtime.WorkflowStep{}, unsupportedWorkflowOperation(
			"steps/"+input.ID,
			"unknown operation",
		)
	}

	return output, nil
}

func compileWorkflowCall(
	document workflowv1.WorkflowDocument,
	input workflowv1.Call,
) (sessionv3Runtime.WorkflowCall, error) {
	mode := sessionv3Runtime.WorkflowCallModeInvoke
	if input.Mode != "" {
		mode = sessionv3Runtime.WorkflowCallMode(input.Mode)
	}
	switch mode {
	case sessionv3Runtime.WorkflowCallModeInvoke,
		sessionv3Runtime.WorkflowCallModeStart,
		sessionv3Runtime.WorkflowCallModeHandoff:
	default:
		return sessionv3Runtime.WorkflowCall{}, unsupportedWorkflowOperation(
			"call",
			"unknown call mode",
		)
	}

	lifetime := sessionv3Runtime.WorkflowChildLifetime("")
	if input.Lifetime != "" {
		lifetime = sessionv3Runtime.WorkflowChildLifetime(input.Lifetime)
	}
	if mode != sessionv3Runtime.WorkflowCallModeStart &&
		mode != sessionv3Runtime.WorkflowCallModeHandoff &&
		lifetime != "" {
		return sessionv3Runtime.WorkflowCall{}, unsupportedWorkflowOperation(
			"call",
			"lifetime outside start mode",
		)
	}
	if len(input.Ports) != 0 {
		return sessionv3Runtime.WorkflowCall{}, unsupportedWorkflowOperation(
			"call",
			"ports",
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
	contextValue, err := compileWorkflowValueObject(input.Context)
	if err != nil {
		return sessionv3Runtime.WorkflowCall{}, err
	}

	contract, err := compileWorkflowCallableInterface(
		document,
		input.Interface,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowCall{}, err
	}
	output := sessionv3Runtime.WorkflowCall{
		Target:       target,
		Input:        callInput,
		Context:      contextValue,
		Contract:     contract,
		Mode:         mode,
		Lifetime:     lifetime,
		OnCompletion: input.OnCompletion,
	}
	if len(input.Conversation) != 0 {
		conversation, err := compileWorkflowValue(input.Conversation)
		if err != nil {
			return sessionv3Runtime.WorkflowCall{}, err
		}
		output.Conversation = &conversation
	}
	if input.Ownership != nil {
		ownership, err := compileWorkflowChildOwnership(*input.Ownership)
		if err != nil {
			return sessionv3Runtime.WorkflowCall{}, err
		}
		output.Ownership = &ownership
	}
	if input.Allowed == "" {
		if err := output.Validate(); err != nil {
			return sessionv3Runtime.WorkflowCall{}, err
		}
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
		return sessionv3Runtime.WorkflowCall{}, unsupportedWorkflowOperation(
			"targetSets/"+input.Allowed,
			"resolver-backed dynamic target set",
		)
	}
	if len(set.Members) == 0 {
		return sessionv3Runtime.WorkflowCall{}, errors.New("workflow target set has no members")
	}
	output.AllowedTargets = append([]string(nil), set.Members...)
	output.SetContract, err = compileWorkflowCallableInterface(
		document,
		set.Interface,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowCall{}, err
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowCall{}, err
	}
	return output, nil
}

func compileWorkflowCallableInterface(
	document workflowv1.WorkflowDocument,
	input *workflowv1.CallableInterface,
) (*sessionv3Runtime.WorkflowCallableInterface, error) {
	if input == nil {
		return nil, nil
	}

	// These fields must not be admitted and silently discarded. Their
	// execution paths remain explicit development gaps.
	if len(input.Context) != 0 || len(input.Ports) != 0 ||
		input.Limits != nil || len(input.Capabilities) != 0 {
		return nil, unsupportedWorkflowOperation(
			"interface",
			"context, ports, limits, or capability requirements",
		)
	}

	inputSchema, err := compileWorkflowSchema(document, input.InputSchema)
	if err != nil {
		return nil, err
	}
	outputSchema, err := compileWorkflowSchema(document, input.OutputSchema)
	if err != nil {
		return nil, err
	}
	errorSchema, err := compileWorkflowSchema(document, input.ErrorSchema)
	if err != nil {
		return nil, err
	}
	receiptSchema, err := compileWorkflowSchema(document, input.ReceiptSchema)
	if err != nil {
		return nil, err
	}
	if len(inputSchema) == 0 || len(outputSchema) == 0 {
		return nil, errors.New("workflow callable interface schemas are incomplete")
	}

	return &sessionv3Runtime.WorkflowCallableInterface{
		Modes:         append([]sessionv3Runtime.WorkflowCallMode(nil), workflowCallModes(input.Modes)...),
		InputSchema:   inputSchema,
		OutputSchema:  outputSchema,
		ErrorSchema:   errorSchema,
		ReceiptSchema: receiptSchema,
	}, nil
}

func compileWorkflowEffect(
	input *workflowv1.EffectContract,
) (sessionv3Runtime.EffectContract, error) {
	if input == nil {
		return sessionv3Runtime.EffectContract{
			Kind: sessionv3Runtime.EffectKindNonIdempotent,
		}, nil
	}

	if input.Keyed || input.Reconcile || input.Compensate {
		return sessionv3Runtime.EffectContract{}, unsupportedWorkflowOperation(
			"target.interface.effect",
			"keying, reconciliation, or compensation capability",
		)
	}

	switch input.Kind {
	case "read":
		return sessionv3Runtime.EffectContract{
			Kind: sessionv3Runtime.EffectKindRead,
		}, nil
	case "idempotent":
		return sessionv3Runtime.EffectContract{
			Kind: sessionv3Runtime.EffectKindIdempotent,
		}, nil
	case "non-idempotent":
		return sessionv3Runtime.EffectContract{
			Kind: sessionv3Runtime.EffectKindNonIdempotent,
		}, nil
	case "keyed":
		return sessionv3Runtime.EffectContract{}, unsupportedWorkflowOperation(
			"target.interface.effect",
			"keyed effect without a typed operation key",
		)
	default:
		return sessionv3Runtime.EffectContract{}, unsupportedWorkflowOperation(
			"target.interface.effect",
			"unknown effect contract",
		)
	}
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

	value, err := jsonutil.DecodeJSONRawNumbers[any](
		json.RawMessage(input),
	)
	if err != nil {
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
			raw, err := jsonutil.EncodeToJSONRaw(literal)
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
		raw, err := jsonutil.EncodeToJSONRaw(typed)
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
