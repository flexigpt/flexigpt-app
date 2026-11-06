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
	resources, err := compileWorkflowResources(document)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	ports, err := compileWorkflowPorts(document)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	limits, err := compileWorkflowOptionalExecutionBudget(document.Limits)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	events, err := compileWorkflowEvents(document)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	channels, targetSets, err := compileWorkflowChannelsAndTargetSets(document)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	triggers, err := compileWorkflowTriggers(document)
	if err != nil {
		return sessionv3Runtime.WorkflowProgram{}, err
	}
	inputSchema, err := compileWorkflowSchema(document, document.InputSchema)
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
		TargetSets:   targetSets,
		Events:       events,
		Resources:    resources,
		Channels:     channels,
		Ports:        ports,
		Triggers:     triggers,
	}
	if limits != nil {
		output.Limits = *limits
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
	_ = document
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
	output, err := compileWorkflowBlockCore(document, block)
	if err != nil {
		return sessionv3Runtime.WorkflowBlock{}, err
	}
	output.Listen, err = compileWorkflowSubscriptions(document, block.Listen)
	if err != nil {
		return sessionv3Runtime.WorkflowBlock{}, err
	}
	output.Catch, err = compileWorkflowCatchHandlers(
		document,
		block.Catch,
	)
	if err != nil {
		return sessionv3Runtime.WorkflowBlock{}, err
	}
	if block.Finally != nil {
		finallyBlock, finallyErr := compileWorkflowBlock(
			document,
			*block.Finally,
		)
		if finallyErr != nil {
			return sessionv3Runtime.WorkflowBlock{}, finallyErr
		}
		output.Finally = &finallyBlock
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
	output := sessionv3Runtime.WorkflowStep{ID: input.ID}
	limits, err := compileWorkflowOptionalExecutionBudget(input.Limits)
	if err != nil {
		return sessionv3Runtime.WorkflowStep{}, err
	}
	output.Limits = limits
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
	if input.Effect != nil {
		effect, err := compileWorkflowStepEffect(
			document,
			*input.Effect,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Effect = &effect
	}
	claims, err := compileWorkflowResourceClaims(input.Claims)
	if err != nil {
		return sessionv3Runtime.WorkflowStep{}, err
	}
	output.Claims = claims

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
		input.Machine != nil,
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
		saga, err := compileWorkflowSaga(document, *input.Saga)
		if err != nil {
			return sessionv3Runtime.WorkflowStep{}, err
		}
		output.Saga = &saga

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
	ports, err := compileWorkflowPortBindings(input.Ports)
	if err != nil {
		return sessionv3Runtime.WorkflowCall{}, err
	}
	output := sessionv3Runtime.WorkflowCall{
		Target:       target,
		Input:        callInput,
		Context:      contextValue,
		Ports:        ports,
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
	if len(set.Members) == 0 && set.Resolved == nil {
		return sessionv3Runtime.WorkflowCall{}, errors.New("workflow target set has no members")
	}
	output.AllowedTargetSet = input.Allowed
	if set.Resolved == nil {
		output.AllowedTargets = append([]string(nil), set.Members...)
	}
	interfaceValue := set.Interface
	if set.Resolved != nil {
		interfaceValue = &set.Resolved.Interface
	}
	output.SetContract, err = compileWorkflowCallableInterface(document, interfaceValue)
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
	contextRequirements, err := compileWorkflowContextRequirements(
		document,
		input.Context,
	)
	if err != nil {
		return nil, err
	}
	ports, err := compileWorkflowCallablePorts(document, input.Ports)
	if err != nil {
		return nil, err
	}
	limits, err := compileWorkflowOptionalExecutionBudget(input.Limits)
	if err != nil {
		return nil, err
	}
	capabilities, err := compileWorkflowCapabilityRequirements(
		input.Capabilities,
	)
	if err != nil {
		return nil, err
	}

	return &sessionv3Runtime.WorkflowCallableInterface{
		Modes:         append([]sessionv3Runtime.WorkflowCallMode(nil), workflowCallModes(input.Modes)...),
		InputSchema:   inputSchema,
		OutputSchema:  outputSchema,
		ErrorSchema:   errorSchema,
		ReceiptSchema: receiptSchema,
		Context:       contextRequirements,
		Ports:         ports,
		Limits:        limits,
		Capabilities:  capabilities,
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

	output := sessionv3Runtime.EffectContract{
		Reconcile:  input.Reconcile,
		Compensate: input.Compensate,
	}
	switch input.Kind {
	case string(sessionv3Runtime.EffectKindRead):
		output.Kind = sessionv3Runtime.EffectKindRead
	case string(sessionv3Runtime.EffectKindIdempotent):
		output.Kind = sessionv3Runtime.EffectKindIdempotent
	case string(sessionv3Runtime.EffectKindNonIdempotent):
		output.Kind = sessionv3Runtime.EffectKindNonIdempotent
	case string(sessionv3Runtime.EffectKindKeyed):
		// A declaration-level target contract describes capability. The
		// concrete key is a call-level stable semantic value.
		output.Kind = sessionv3Runtime.EffectKindIdempotent
	default:
		return sessionv3Runtime.EffectContract{}, unsupportedWorkflowOperation(
			"target.interface.effect",
			"unknown effect contract",
		)
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.EffectContract{}, err
	}
	return output, nil
}

func compileWorkflowStepEffect(
	document workflowv1.WorkflowDocument,
	input workflowv1.Effect,
) (sessionv3Runtime.WorkflowStepEffect, error) {
	output := sessionv3Runtime.WorkflowStepEffect{}
	switch input.Kind {
	case string(sessionv3Runtime.EffectKindRead):
		output.Kind = sessionv3Runtime.EffectKindRead
	case string(sessionv3Runtime.EffectKindIdempotent):
		output.Kind = sessionv3Runtime.EffectKindIdempotent
	case string(sessionv3Runtime.EffectKindKeyed):
		output.Kind = sessionv3Runtime.EffectKindKeyed
	case string(sessionv3Runtime.EffectKindNonIdempotent):
		output.Kind = sessionv3Runtime.EffectKindNonIdempotent
	default:
		return sessionv3Runtime.WorkflowStepEffect{},
			unsupportedWorkflowOperation(
				"step.effect",
				"unknown effect contract",
			)
	}

	if len(input.Key) != 0 {
		if output.Kind != sessionv3Runtime.EffectKindKeyed {
			return sessionv3Runtime.WorkflowStepEffect{},
				errors.New(
					"non-keyed workflow effect cannot declare an operation key",
				)
		}
		key, err := compileWorkflowValue(input.Key)
		if err != nil {
			return sessionv3Runtime.WorkflowStepEffect{}, err
		}
		output.Key = &key
	}
	if input.Reconcile != nil {
		block, err := compileWorkflowBlock(
			document,
			*input.Reconcile,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowStepEffect{}, err
		}
		output.Reconcile = &block
	}
	if input.Compensate != nil {
		block, err := compileWorkflowBlock(
			document,
			*input.Compensate,
		)
		if err != nil {
			return sessionv3Runtime.WorkflowStepEffect{}, err
		}
		output.Compensate = &block
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowStepEffect{}, err
	}
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
