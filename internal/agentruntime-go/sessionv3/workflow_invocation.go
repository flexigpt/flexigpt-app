package sessionv3

import (
	"encoding/json"
	"errors"
	"slices"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type workflowLLMInvocation struct {
	Activities         []Activity
	Defaults           SessionDefaults
	InputActivityID    ActivityID
	HistoryActivityIDs []ActivityID
	PreparedInvocation *WorkflowPreparedInvocation
}

func (v WorkflowLLMInvocation) Validate() error {
	if len(v.Messages) == 0 && len(v.MessageSegments) == 0 {
		return errors.New("workflow model or agent invocation requires messages")
	}
	if workflowInvocationOverridesTarget(v.Defaults) {
		return errors.New(
			"workflow model or agent defaults cannot replace the selected target",
		)
	}
	for _, segment := range v.MessageSegments {
		if (len(segment.Messages) == 0) == (segment.Collection == nil) {
			return errors.New("workflow model or agent message segment is invalid")
		}
		if segment.Collection != nil {
			if err := segment.Collection.Validate(); err != nil {
				return err
			}
		}
		for index := range segment.Messages {
			if err := segment.Messages[index].Validate(); err != nil {
				return err
			}
		}
	}
	for _, segment := range v.InstructionSegments {
		if (len(segment.Instructions) == 0) == (segment.Collection == nil) {
			return errors.New("workflow model or agent instruction segment is invalid")
		}
		if segment.Collection != nil {
			if err := segment.Collection.Validate(); err != nil {
				return err
			}
		}
		for index := range segment.Instructions {
			if err := segment.Instructions[index].Validate(); err != nil {
				return err
			}
		}
	}
	for index := range v.Messages {
		if err := v.Messages[index].Validate(); err != nil {
			return err
		}
	}
	for index := range v.Instructions {
		if err := v.Instructions[index].Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (v WorkflowLLMMessage) Validate() error {
	switch v.Role {
	case MessageRoleSystem,
		MessageRoleDeveloper,
		MessageRoleUser,
		MessageRoleAssistant:
	default:
		return errors.New("workflow model or agent message role is unsupported")
	}
	if v.Text != nil {
		if err := v.Text.Validate(); err != nil {
			return err
		}
	}
	for _, attachment := range v.Attachments {
		if err := attachment.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (v WorkflowLLMInstruction) Validate() error {
	if v.ID != "" {
		if err := validateID("workflow instruction ID", v.ID); err != nil {
			return err
		}
	}
	return v.Content.Validate()
}

func (i *WorkflowInterpreter) captureWorkflowLLMInvocation(
	program WorkflowProgram,
	state WorkflowState,
	template WorkflowLLMInvocation,
) (WorkflowLLMInvocation, error) {
	if err := template.Validate(); err != nil {
		return WorkflowLLMInvocation{}, err
	}
	output := WorkflowLLMInvocation{
		Defaults:        jsonutil.CloneJSONInValue(template.Defaults),
		PreparedTargets: jsonutil.CloneJSONInValue(template.PreparedTargets),
	}
	messageSegments := template.MessageSegments
	if len(messageSegments) == 0 {
		messageSegments = []WorkflowLLMMessageSegment{{
			Messages: jsonutil.CloneJSONInValue(template.Messages),
		}}
	}
	for _, segment := range messageSegments {
		values := segment.Messages
		if segment.Collection != nil {
			value, err := i.evaluate(*segment.Collection, program, state)
			if err != nil {
				return WorkflowLLMInvocation{}, err
			}
			values, err = decodeWorkflowLLMMessageCollection(value)
			if err != nil {
				return WorkflowLLMInvocation{}, err
			}
		}
		for _, message := range values {
			captured := WorkflowLLMMessage{
				Role:        message.Role,
				Attachments: jsonutil.CloneJSONInValue(message.Attachments),
			}
			if message.Text != nil {
				value, err := i.evaluate(*message.Text, program, state)
				if err != nil {
					return WorkflowLLMInvocation{}, err
				}
				captured.Text = &WorkflowValue{
					Kind:    WorkflowValueLiteral,
					Literal: value.Clone(),
				}
			}
			output.Messages = append(output.Messages, captured)
		}
	}
	instructionSegments := template.InstructionSegments
	if len(instructionSegments) == 0 && len(template.Instructions) != 0 {
		instructionSegments = []WorkflowLLMInstructionSegment{{
			Instructions: jsonutil.CloneJSONInValue(template.Instructions),
		}}
	}
	for _, segment := range instructionSegments {
		values := segment.Instructions
		if segment.Collection != nil {
			value, err := i.evaluate(*segment.Collection, program, state)
			if err != nil {
				return WorkflowLLMInvocation{}, err
			}
			values, err = decodeWorkflowLLMInstructionCollection(value)
			if err != nil {
				return WorkflowLLMInvocation{}, err
			}
		}
		for _, instruction := range values {
			value, err := i.evaluate(instruction.Content, program, state)
			if err != nil {
				return WorkflowLLMInvocation{}, err
			}
			output.Instructions = append(output.Instructions, WorkflowLLMInstruction{
				ID:    instruction.ID,
				Label: instruction.Label,
				Content: WorkflowValue{
					Kind:    WorkflowValueLiteral,
					Literal: value.Clone(),
				},
			})
		}
	}
	if err := output.Validate(); err != nil {
		return WorkflowLLMInvocation{}, err
	}
	return output, nil
}

func decodeWorkflowLLMMessageCollection(
	value jsonutil.CanonicalValue,
) ([]WorkflowLLMMessage, error) {
	var values []struct {
		Role        MessageRole  `json:"role"`
		Text        *string      `json:"text,omitempty"`
		Attachments []Attachment `json:"attachments,omitempty"`
	}
	if err := json.Unmarshal(value, &values); err != nil || len(values) == 0 {
		return nil, errors.New("workflow message collection did not produce typed messages")
	}
	output := make([]WorkflowLLMMessage, 0, len(values))
	for _, value := range values {
		message := WorkflowLLMMessage{
			Role:        value.Role,
			Attachments: jsonutil.CloneJSONInValue(value.Attachments),
		}
		if value.Text != nil {
			message.Text = &WorkflowValue{
				Kind:    WorkflowValueLiteral,
				Literal: jsonutil.MustCanonicalValue(*value.Text),
			}
		}
		if err := message.Validate(); err != nil {
			return nil, err
		}
		output = append(output, message)
	}
	return output, nil
}

func decodeWorkflowLLMInstructionCollection(
	value jsonutil.CanonicalValue,
) ([]WorkflowLLMInstruction, error) {
	var values []struct {
		ID      string `json:"id,omitempty"`
		Label   string `json:"label,omitempty"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(value, &values); err != nil {
		return nil, errors.New("workflow instruction collection did not produce typed instructions")
	}
	output := make([]WorkflowLLMInstruction, 0, len(values))
	for _, value := range values {
		instruction := WorkflowLLMInstruction{
			ID:      value.ID,
			Label:   value.Label,
			Content: WorkflowValue{Kind: WorkflowValueLiteral, Literal: jsonutil.MustCanonicalValue(value.Content)},
		}
		if err := instruction.Validate(); err != nil {
			return nil, err
		}
		output = append(output, instruction)
	}
	return output, nil
}

func (i *WorkflowInterpreter) prepareWorkflowLLMInvocation(
	frame ProgramFrame,
	path string,
	targetName string,
	target WorkflowTarget,
	template WorkflowLLMInvocation,
	program WorkflowProgram,
	state WorkflowState,
) (workflowLLMInvocation, error) {
	captured, err := i.captureWorkflowLLMInvocation(program, state, template)
	if err != nil {
		return workflowLLMInvocation{}, err
	}
	template = captured

	defaults := SessionDefaults{}
	if template.Defaults != nil {
		overrides := jsonutil.CloneJSONInValue(*template.Defaults)
		resolved, err := resolveSessionInvocationDefaults(
			defaults,
			&overrides,
		)
		if err != nil {
			return workflowLLMInvocation{}, err
		}
		defaults = resolved
	}
	for _, instruction := range template.Instructions {
		content, err := i.stringValue(
			instruction.Content,
			program,
			state,
		)
		if err != nil {
			return workflowLLMInvocation{}, err
		}
		defaults.SystemInstructions = append(
			defaults.SystemInstructions,
			Instruction{
				ID:      instruction.ID,
				Label:   instruction.Label,
				Content: content,
			},
		)
	}

	prepared := target.Invocation
	if value, found := template.PreparedTargets[targetName]; found {
		copyValue := value
		prepared = &copyValue
	}
	if prepared != nil {
		if err := prepared.Validate(); err != nil {
			return workflowLLMInvocation{}, err
		}
		if prepared.Model != nil {
			defaults.Model = selectionForPreparedPin(*prepared.Model)
		}
	} else if target.Artifact == nil {
		return workflowLLMInvocation{}, errors.New(
			"workflow model or agent target has no source-backed Artifact pin",
		)
	}

	if prepared == nil && target.Artifact != nil {
		selection := &ArtifactSelection{
			RootID:           target.Artifact.RootID,
			ArtifactID:       target.Artifact.ArtifactID,
			ExpectedRevision: target.Artifact.Revision,
			ExpectedDigest:   target.Artifact.Digest,
		}
		switch target.Kind {
		case WorkflowTargetModel:
			defaults.Model = selection
		case WorkflowTargetAgent:
			defaults.Agent = selection
		default:
			return workflowLLMInvocation{}, errors.New(
				"workflow target is not model or agent",
			)
		}
	}

	activities := make([]Activity, 0, len(template.Messages))
	activityIDs := make([]ActivityID, 0, len(template.Messages))
	for index := range template.Messages {
		message := template.Messages[index]
		text := ""
		if message.Text != nil {
			value, err := i.stringValue(
				*message.Text,
				program,
				state,
			)
			if err != nil {
				return workflowLLMInvocation{}, err
			}
			text = value
		}
		activityID := ActivityID(
			"workflow-llm:" +
				string(frame.ID) +
				":" +
				path +
				":input:" +
				strconv.Itoa(index),
		)
		activities = append(activities, Activity{
			ID:         activityID,
			Kind:       ActivityKindMessage,
			Visibility: ActivityVisibilityAudit,
			Message: &MessageActivity{
				Role: message.Role,
				Text: text,
				Attachments: jsonutil.CloneJSONInValue(
					message.Attachments,
				),
			},
		})
		activityIDs = append(activityIDs, activityID)
	}

	return workflowLLMInvocation{
		Activities:         activities,
		Defaults:           defaults,
		InputActivityID:    activityIDs[len(activityIDs)-1],
		HistoryActivityIDs: append([]ActivityID(nil), activityIDs[:len(activityIDs)-1]...),
		PreparedInvocation: prepared,
	}, nil
}

func selectionForPreparedPin(value ArtifactPin) *ArtifactSelection {
	return &ArtifactSelection{
		RootID:           value.RootID,
		ArtifactID:       value.ArtifactID,
		ExpectedRevision: value.Revision,
		ExpectedDigest:   value.Digest,
	}
}

func workflowInvocationOverridesTarget(
	value *SessionDefaultsOverride,
) bool {
	if value == nil {
		return false
	}
	if value.Model != nil || value.Agent != nil {
		return true
	}
	return slices.Contains(value.Clear, SessionDefaultFieldModel) ||
		slices.Contains(value.Clear, SessionDefaultFieldAgent)
}

func configuredProviderManagedToolChoice(
	value ProviderManagedToolChoice,
	defaults SessionDefaults,
) (ProviderManagedToolChoice, error) {
	for _, selection := range EffectiveToolSelections(defaults) {
		if selection.Artifact.RootID != value.Artifact.RootID ||
			selection.Artifact.ArtifactID != value.Artifact.ArtifactID {
			continue
		}
		value.Configuration = selection.UserArguments.Clone()
		break
	}
	if err := value.Validate(); err != nil {
		return ProviderManagedToolChoice{}, err
	}
	return value, nil
}

func withoutProviderManagedToolSelection(
	input SessionDefaults,
	target ArtifactPin,
) SessionDefaults {
	output := cloneSessionDefaults(input)

	output.ToolSelections = output.ToolSelections[:0]
	for _, selection := range input.ToolSelections {
		if selection.Artifact.RootID == target.RootID &&
			selection.Artifact.ArtifactID == target.ArtifactID {
			continue
		}
		output.ToolSelections = append(
			output.ToolSelections,
			CloneToolSelection(selection),
		)
	}
	if len(output.ToolSelections) == 0 {
		output.ToolSelections = nil
	}
	return output
}

func (i *WorkflowInterpreter) invokeWorkflowLLMTarget(
	frame ProgramFrame,
	state WorkflowState,
	program WorkflowProgram,
	path string,
	step WorkflowStep,
	targetName string,
	target WorkflowTarget,
	call WorkflowCall,
	input jsonutil.CanonicalValue,
	contextValue jsonutil.CanonicalValue,
	effect EffectContract,
	handoff *ConversationHandoffIntent,
	ports []PortPlan,
	budget *ExecutionBudget,
) (ProgramTransition, error) {
	if call.LLMInvocation == nil {
		return workflowFailure(
			state,
			"workflow_invocation_input_invalid",
			"workflow model or agent target requires a typed invocation",
		)
	}
	invocation, err := i.prepareWorkflowLLMInvocation(
		frame,
		path,
		targetName,
		target,
		*call.LLMInvocation,
		program,
		state,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_invocation_input_invalid",
			err.Error(),
		)
	}

	var child ProgramSnapshot
	switch target.Kind {
	case WorkflowTargetModel:
		child, err = NewSingleModelWorkflowInvocationProgram(
			invocation.Defaults,
			invocation.InputActivityID,
			invocation.HistoryActivityIDs,
			nil,
			invocation.PreparedInvocation,
		)
	case WorkflowTargetAgent:
		child, err = NewAgentLoopWorkflowInvocationProgram(
			invocation.Defaults,
			invocation.InputActivityID,
			invocation.HistoryActivityIDs,
			nil,
			invocation.PreparedInvocation,
		)
	default:
		return workflowFailure(
			state,
			"workflow_target_invalid",
			"workflow target is not model or agent",
		)
	}
	if err != nil {
		return workflowFailure(
			state,
			"workflow_invocation_prepare_failed",
			err.Error(),
		)
	}

	transition, err := i.startWorkflowChildCall(
		frame,
		program,
		state,
		path,
		step,
		targetName,
		effect,
		input,
		contextValue,
		child,
		ports,
		budget,
		ProgramFramePortBoundaryNone,
		handoff,
	)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.Activities = append([]Activity(nil), invocation.Activities...)
	return transition, nil
}

func (i *WorkflowInterpreter) handoffWorkflowAgentTarget(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	step WorkflowStep,
	targetName string,
	target WorkflowTarget,
	call WorkflowCall,
	input jsonutil.CanonicalValue,
	contextValue jsonutil.CanonicalValue,
	effect EffectContract,
	ports []PortPlan,
	budget *ExecutionBudget,
) (ProgramTransition, error) {
	if target.Kind != WorkflowTargetAgent || target.Artifact == nil {
		return workflowFailure(
			state,
			"workflow_handoff_invalid",
			"workflow handoff requires an agent target",
		)
	}
	if call.Conversation == nil {
		return workflowFailure(
			state,
			"workflow_handoff_invalid",
			"workflow handoff requires a conversation",
		)
	}

	conversationID, err := i.stringValue(
		*call.Conversation,
		program,
		state,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_handoff_invalid",
			err.Error(),
		)
	}

	owner := ""
	if call.Ownership != nil {
		owner, err = i.stringValue(
			call.Ownership.Owner,
			program,
			state,
		)
		if err != nil {
			return workflowFailure(
				state,
				"workflow_handoff_invalid",
				err.Error(),
			)
		}
	}

	targetAgent := *target.Artifact
	return i.invokeWorkflowLLMTarget(
		frame,
		state,
		program,
		path,
		step,
		targetName,
		target,
		call,
		input,
		contextValue,
		effect,
		&ConversationHandoffIntent{
			ID:                  "workflow-handoff:" + string(frame.ID) + ":" + path,
			ConversationID:      conversationID,
			Owner:               owner,
			TargetAgent:         targetAgent,
			RestoreOnCompletion: call.OnCompletion != "retain",
		},
		ports,
		budget,
	)
}
