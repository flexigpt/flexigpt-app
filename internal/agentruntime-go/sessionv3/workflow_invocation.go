package sessionv3

import (
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
}

func (v WorkflowLLMInvocation) Validate() error {
	if len(v.Messages) == 0 {
		return errors.New("workflow model or agent invocation requires messages")
	}
	if workflowInvocationOverridesTarget(v.Defaults) {
		return errors.New(
			"workflow model or agent defaults cannot replace the selected target",
		)
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

func (i *WorkflowInterpreter) prepareWorkflowLLMInvocation(
	frame ProgramFrame,
	path string,
	target WorkflowTarget,
	template WorkflowLLMInvocation,
	program WorkflowProgram,
	state WorkflowState,
) (workflowLLMInvocation, error) {
	if err := template.Validate(); err != nil {
		return workflowLLMInvocation{}, err
	}

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

	if target.Artifact == nil {
		return workflowLLMInvocation{}, errors.New(
			"workflow model or agent target has no source-backed Artifact pin",
		)
	}

	if target.Artifact != nil {
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
	}, nil
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

func (i *WorkflowInterpreter) invokeWorkflowLLMTarget(
	frame ProgramFrame,
	state WorkflowState,
	program WorkflowProgram,
	path string,
	step WorkflowStep,
	targetName string,
	target WorkflowTarget,
	call WorkflowCall,
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
	claims, err := i.workflowResourceClaims(
		program,
		state,
		step,
	)
	if err != nil {
		return workflowFailure(
			state,
			"workflow_resource_claim_invalid",
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
		)
	case WorkflowTargetAgent:
		child, err = NewAgentLoopWorkflowInvocationProgram(
			invocation.Defaults,
			invocation.InputActivityID,
			invocation.HistoryActivityIDs,
			nil,
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

	state.PendingPath = path
	state.PendingTarget = targetName
	transition, err := workflowTransition(state, nil, nil, nil)
	if err != nil {
		return ProgramTransition{}, err
	}
	transition.Activities = invocation.Activities
	transition.Frames = []ProgramFrameIntent{{
		Path:            path,
		DeclarationPath: state.PendingDeclarationPath,
		Program:         child,
		Deadline:        cloneTime(state.Deadline),
		Budget:          cloneBudget(budget),
		Claims:          claims,
		Ports:           clonePortPlans(ports),
		Handoff:         handoff,
	}}
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
