package workflowruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	workflowAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

const (
	workflowLLMContextDefaults     = "defaults"
	workflowLLMContextInstructions = "instructions"
	workflowLLMContextMessages     = "messages"

	workflowLLMFieldAttachments = "attachments"
	workflowLLMFieldContent     = "content"
	workflowLLMFieldID          = "id"
	workflowLLMFieldLabel       = "label"
	workflowLLMFieldMessages    = "messages"
	workflowLLMFieldRole        = "role"
	workflowLLMFieldText        = "text"
)

// compileWorkflowLLMInvocation converts the supported explicit Workflow
// Model and Agent invocation form into typed runtime templates. It is called
// only after setup determines that a call can select a Model, Agent, or
// provider-managed Tool target.
func compileWorkflowLLMInvocation(
	call workflowv1.Call,
) (*sessionv3Runtime.WorkflowLLMInvocation, error) {
	inputSegment, err := compileWorkflowLLMMessageSegment(
		call.Input,
		"workflow model or agent input messages",
	)
	if err != nil {
		return nil, err
	}

	output := &sessionv3Runtime.WorkflowLLMInvocation{
		MessageSegments: []sessionv3Runtime.WorkflowLLMMessageSegment{
			inputSegment,
		},
	}
	for name := range call.Context {
		switch name {
		case workflowLLMContextInstructions,
			workflowLLMContextMessages,
			workflowLLMContextDefaults:
		default:
			return nil, fmt.Errorf(
				"workflow model or agent context form %q is unsupported",
				name,
			)
		}
	}

	if value, found := call.Context[workflowLLMContextInstructions]; found {
		segment, instructionErr := compileWorkflowLLMInstructionSegment(
			value,
		)
		if instructionErr != nil {
			return nil, instructionErr
		}
		output.InstructionSegments = append(
			output.InstructionSegments,
			segment,
		)
	}
	if value, found := call.Context[workflowLLMContextMessages]; found {
		segment, messageErr := compileWorkflowLLMMessageSegment(
			value,
			"workflow model or agent context messages",
		)
		if messageErr != nil {
			return nil, messageErr
		}
		output.MessageSegments = append(
			[]sessionv3Runtime.WorkflowLLMMessageSegment{segment},
			output.MessageSegments...,
		)
	}
	if value, found := call.Context[workflowLLMContextDefaults]; found {
		defaults, defaultsErr := compileWorkflowLLMDefaults(value)
		if defaultsErr != nil {
			return nil, defaultsErr
		}
		output.Defaults = defaults
	}

	if err := output.Validate(); err != nil {
		return nil, err
	}
	return output, nil
}

func compileWorkflowLLMMessageSegment(
	value workflowv1.Value,
	label string,
) (sessionv3Runtime.WorkflowLLMMessageSegment, error) {
	template, err := compileWorkflowValue(value)
	if err != nil {
		return sessionv3Runtime.WorkflowLLMMessageSegment{}, err
	}
	if template.Kind == sessionv3Runtime.WorkflowValueExpression {
		return sessionv3Runtime.WorkflowLLMMessageSegment{
			Collection: &template,
		}, nil
	}
	if template.Kind == sessionv3Runtime.WorkflowValueLiteral {
		value = workflowv1.Value(template.Literal)
	}
	values, err := compileWorkflowLLMMessageList(value, label)
	if err != nil {
		return sessionv3Runtime.WorkflowLLMMessageSegment{}, err
	}
	return sessionv3Runtime.WorkflowLLMMessageSegment{Messages: values}, nil
}

func compileWorkflowLLMMessageList(
	value workflowv1.Value,
	label string,
) ([]sessionv3Runtime.WorkflowLLMMessage, error) {
	var values []json.RawMessage
	if err := json.Unmarshal(value, &values); err != nil {
		return nil, fmt.Errorf("%s must be an array: %w", label, err)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("%s must not be empty", label)
	}

	output := make([]sessionv3Runtime.WorkflowLLMMessage, 0, len(values))
	for index := range values {
		message, err := compileWorkflowLLMMessage(values[index], label)
		if err != nil {
			return nil, fmt.Errorf("%s item %d: %w", label, index, err)
		}
		output = append(output, message)
	}
	return output, nil
}

func compileWorkflowLLMMessage(
	raw json.RawMessage,
	label string,
) (sessionv3Runtime.WorkflowLLMMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return sessionv3Runtime.WorkflowLLMMessage{}, fmt.Errorf(
			"%s item must be an object",
			label,
		)
	}
	for name := range object {
		switch name {
		case workflowLLMFieldRole, workflowLLMFieldText, workflowLLMFieldAttachments:
		default:
			return sessionv3Runtime.WorkflowLLMMessage{}, fmt.Errorf(
				"%s item has unsupported field %q",
				label,
				name,
			)
		}
	}

	roleRaw, found := object[workflowLLMFieldRole]
	if !found {
		return sessionv3Runtime.WorkflowLLMMessage{}, fmt.Errorf(
			"%s item requires role",
			label,
		)
	}
	var role sessionv3Runtime.MessageRole
	if err := json.Unmarshal(roleRaw, &role); err != nil {
		return sessionv3Runtime.WorkflowLLMMessage{}, errors.New(
			"workflow model or agent message role must be a string",
		)
	}

	output := sessionv3Runtime.WorkflowLLMMessage{Role: role}
	if textRaw, found := object[workflowLLMFieldText]; found {
		value, err := compileWorkflowValue(workflowv1.Value(textRaw))
		if err != nil {
			return sessionv3Runtime.WorkflowLLMMessage{}, err
		}
		output.Text = &value
	}
	if attachmentsRaw, found := object[workflowLLMFieldAttachments]; found {
		var attachments []sessionv3Runtime.Attachment
		if err := json.Unmarshal(attachmentsRaw, &attachments); err != nil {
			return sessionv3Runtime.WorkflowLLMMessage{}, errors.New(
				"workflow model or agent message attachments are invalid",
			)
		}
		output.Attachments = attachments
	}
	if err := output.Validate(); err != nil {
		return sessionv3Runtime.WorkflowLLMMessage{}, err
	}
	return output, nil
}

func compileWorkflowLLMInstructionSegment(
	value workflowv1.Value,
) (sessionv3Runtime.WorkflowLLMInstructionSegment, error) {
	template, err := compileWorkflowValue(value)
	if err != nil {
		return sessionv3Runtime.WorkflowLLMInstructionSegment{}, err
	}
	if template.Kind == sessionv3Runtime.WorkflowValueExpression {
		return sessionv3Runtime.WorkflowLLMInstructionSegment{
			Collection: &template,
		}, nil
	}
	if template.Kind == sessionv3Runtime.WorkflowValueLiteral {
		value = workflowv1.Value(template.Literal)
	}

	var values []json.RawMessage
	if err := json.Unmarshal(value, &values); err != nil {
		return sessionv3Runtime.WorkflowLLMInstructionSegment{}, errors.New(
			"workflow Model or Agent instructions must be an array",
		)
	}

	output := make(
		[]sessionv3Runtime.WorkflowLLMInstruction,
		0,
		len(values),
	)
	for index := range values {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(values[index], &object); err != nil || object == nil {
			return sessionv3Runtime.WorkflowLLMInstructionSegment{}, fmt.Errorf(
				"workflow Model or Agent instruction %d must be an object",
				index,
			)
		}
		for name := range object {
			switch name {
			case workflowLLMFieldID,
				workflowLLMFieldLabel,
				workflowLLMFieldContent:
			default:
				return sessionv3Runtime.WorkflowLLMInstructionSegment{}, fmt.Errorf(
					"workflow model or agent instruction %d has unsupported field %q",
					index,
					name,
				)
			}
		}

		contentRaw, found := object[workflowLLMFieldContent]
		if !found {
			return sessionv3Runtime.WorkflowLLMInstructionSegment{}, fmt.Errorf(
				"workflow model or agent instruction %d requires content",
				index,
			)
		}
		content, err := compileWorkflowValue(
			workflowv1.Value(contentRaw),
		)
		if err != nil {
			return sessionv3Runtime.WorkflowLLMInstructionSegment{}, err
		}

		instruction := sessionv3Runtime.WorkflowLLMInstruction{
			Content: content,
		}
		if idRaw, found := object[workflowLLMFieldID]; found {
			if err := json.Unmarshal(idRaw, &instruction.ID); err != nil {
				return sessionv3Runtime.WorkflowLLMInstructionSegment{}, errors.New(
					"workflow model or agent instruction ID must be a string",
				)
			}
		}
		if labelRaw, found := object[workflowLLMFieldLabel]; found {
			if err := json.Unmarshal(labelRaw, &instruction.Label); err != nil {
				return sessionv3Runtime.WorkflowLLMInstructionSegment{}, errors.New(
					"workflow model or agent instruction label must be a string",
				)
			}
		}
		if err := instruction.Validate(); err != nil {
			return sessionv3Runtime.WorkflowLLMInstructionSegment{}, err
		}
		output = append(output, instruction)
	}
	return sessionv3Runtime.WorkflowLLMInstructionSegment{
		Instructions: output,
	}, nil
}

func compileWorkflowLLMDefaults(
	value workflowv1.Value,
) (*sessionv3Runtime.SessionDefaultsOverride, error) {
	raw := bytes.TrimSpace(value)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var output sessionv3Runtime.SessionDefaultsOverride
	if err := decoder.Decode(&output); err != nil {
		return nil, errors.New(
			"workflow model or agent defaults must be a typed defaults object",
		)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New(
			"workflow model or agent defaults contain more than one value",
		)
	}
	if output.Model != nil || output.Agent != nil ||
		slicesContainTargetClear(output.Clear) {
		return nil, errors.New(
			"workflow model or agent defaults cannot replace the selected target",
		)
	}
	return &output, nil
}

func slicesContainTargetClear(
	values []sessionv3Runtime.SessionDefaultField,
) bool {
	for _, value := range values {
		if value == sessionv3Runtime.SessionDefaultFieldModel ||
			value == sessionv3Runtime.SessionDefaultFieldAgent {
			return true
		}
	}
	return false
}

func workflowCallAtDeclarationPath(
	block *sessionv3Runtime.WorkflowBlock,
	path string,
) *sessionv3Runtime.WorkflowCall {
	if block == nil || strings.TrimSpace(path) == "" {
		return nil
	}
	for index := range block.Steps {
		step := &block.Steps[index]
		if step.Call != nil && step.Call.DeclarationPath == path {
			return step.Call
		}
		if step.Effect != nil {
			if value := workflowCallAtDeclarationPath(
				step.Effect.Reconcile,
				path,
			); value != nil {
				return value
			}
			if value := workflowCallAtDeclarationPath(
				step.Effect.Compensate,
				path,
			); value != nil {
				return value
			}
		}
		if step.Scope != nil {
			if value := workflowCallAtDeclarationPath(step.Scope, path); value != nil {
				return value
			}
		}
		if step.Choose != nil {
			for caseIndex := range step.Choose.Cases {
				if value := workflowCallAtDeclarationPath(
					&step.Choose.Cases[caseIndex].Do,
					path,
				); value != nil {
					return value
				}
			}
			if value := workflowCallAtDeclarationPath(
				step.Choose.Default,
				path,
			); value != nil {
				return value
			}
		}
		if step.Parallel != nil {
			for branchIndex := range step.Parallel.Branches {
				if value := workflowCallAtDeclarationPath(
					&step.Parallel.Branches[branchIndex].Do,
					path,
				); value != nil {
					return value
				}
			}
		}
		if step.Foreach != nil {
			if value := workflowCallAtDeclarationPath(&step.Foreach.Do, path); value != nil {
				return value
			}
		}
		if step.Repeat != nil {
			if value := workflowCallAtDeclarationPath(&step.Repeat.Do, path); value != nil {
				return value
			}
		}
		if step.Machine != nil {
			for stateIndex := range step.Machine.States {
				machineState := &step.Machine.States[stateIndex]
				if value := workflowCallAtDeclarationPath(machineState.Entry, path); value != nil {
					return value
				}
				if value := workflowCallAtDeclarationPath(machineState.Exit, path); value != nil {
					return value
				}
				for transitionIndex := range machineState.Transitions {
					if value := workflowCallAtDeclarationPath(
						machineState.Transitions[transitionIndex].Do,
						path,
					); value != nil {
						return value
					}
				}
			}
		}
		if step.Saga != nil {
			if value := workflowCallAtDeclarationPath(&step.Saga.Do, path); value != nil {
				return value
			}
		}
	}
	for index := range block.Catch {
		if value := workflowCallAtDeclarationPath(&block.Catch[index].Do, path); value != nil {
			return value
		}
	}
	return workflowCallAtDeclarationPath(block.Finally, path)
}

func prepareWorkflowLLMInvocations(
	inventory workflowAPI.Inventory,
	program *sessionv3Runtime.WorkflowProgram,
) error {
	if program == nil {
		return errors.New("workflow program is unavailable")
	}
	for _, callSite := range inventory.Calls {
		requiresTypedInvocation := false
		for _, use := range callSite.TargetUses {
			target, found := program.Targets[use.Name]
			if !found {
				return fmt.Errorf(
					"workflow call %q target %q is unavailable",
					callSite.Path,
					use.Name,
				)
			}
			switch target.Kind {
			case sessionv3Runtime.WorkflowTargetModel,
				sessionv3Runtime.WorkflowTargetAgent,
				sessionv3Runtime.WorkflowTargetProviderTool:
				requiresTypedInvocation = true
			default:
			}
		}
		if !requiresTypedInvocation {
			continue
		}
		call := workflowCallAtDeclarationPath(&program.Root, callSite.Path)
		if call == nil {
			return fmt.Errorf(
				"workflow call %q has no lowered runtime call",
				callSite.Path,
			)
		}
		invocation, err := compileWorkflowLLMInvocation(callSite.Call)
		if err != nil {
			return fmt.Errorf(
				"workflow call %q typed Model or Agent invocation: %w",
				callSite.Path,
				err,
			)
		}
		invocation.PreparedTargets = make(
			map[string]sessionv3Runtime.WorkflowPreparedInvocation,
		)
		for _, use := range callSite.TargetUses {
			target, found := program.Targets[use.Name]
			if !found || target.Invocation == nil {
				continue
			}
			invocation.PreparedTargets[use.Name] = jsonutil.CloneJSONInValue(*target.Invocation)
		}
		if len(invocation.PreparedTargets) == 0 {
			invocation.PreparedTargets = nil
		}
		call.LLMInvocation = invocation
	}
	return nil
}
