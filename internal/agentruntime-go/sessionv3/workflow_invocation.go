package sessionv3

import (
	"encoding/json"
	"errors"
	"strconv"
)

// workflowInvocationMessage is a Workflow-owned explicit message transfer.
// It is lowered to an existing Session Activity before a Model or Agent child
// ProgramFrame starts. It is not a Model or Agent domain request envelope.
//
// Workflow values that do not use this typed transfer form cannot become
// fallback chat text.
type workflowInvocationMessage struct {
	Role        MessageRole  `json:"role"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type workflowInvocationInput struct {
	Messages []workflowInvocationMessage `json:"messages"`
}

type workflowInvocationContext struct {
	Instructions []Instruction `json:"instructions,omitempty"`
}

type workflowLLMInvocation struct {
	Activities         []Activity
	Defaults           SessionDefaults
	InputActivityID    ActivityID
	HistoryActivityIDs []ActivityID
}

func prepareWorkflowLLMInvocation(
	frame ProgramFrame,
	path string,
	target WorkflowTarget,
	input CanonicalValue,
	contextValue CanonicalValue,
) (workflowLLMInvocation, error) {
	var request workflowInvocationInput
	if err := json.Unmarshal(input, &request); err != nil {
		return workflowLLMInvocation{}, errors.New(
			"workflow model or agent input must use the explicit messages form",
		)
	}
	if len(request.Messages) == 0 {
		return workflowLLMInvocation{}, errors.New(
			"workflow model or agent input requires at least one message",
		)
	}

	var context workflowInvocationContext
	if len(contextValue) != 0 && string(contextValue) != "null" {
		if err := json.Unmarshal(contextValue, &context); err != nil {
			return workflowLLMInvocation{}, errors.New(
				"workflow model or agent context must use the explicit instructions form",
			)
		}
	}

	if target.Artifact == nil {
		return workflowLLMInvocation{}, errors.New(
			"workflow model or agent target has no artifact pin",
		)
	}

	selection := &ArtifactSelection{
		RootID:           target.Artifact.RootID,
		ArtifactID:       target.Artifact.ArtifactID,
		ExpectedRevision: target.Artifact.Revision,
		ExpectedDigest:   target.Artifact.Digest,
	}
	defaults := SessionDefaults{
		SystemInstructions: context.Instructions,
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

	activities := make([]Activity, 0, len(request.Messages))
	activityIDs := make([]ActivityID, 0, len(request.Messages))
	for index := range request.Messages {
		message := request.Messages[index]
		if message.Role == "" {
			return workflowLLMInvocation{}, errors.New(
				"workflow model or agent message role is required",
			)
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
				Role:        message.Role,
				Text:        message.Text,
				Attachments: append([]Attachment(nil), message.Attachments...),
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

func (i *WorkflowInterpreter) invokeWorkflowLLMTarget(
	frame ProgramFrame,
	state WorkflowState,
	path string,
	targetName string,
	target WorkflowTarget,
	input CanonicalValue,
	contextValue CanonicalValue,
	handoff *ConversationHandoffIntent,
) (ProgramTransition, error) {
	invocation, err := prepareWorkflowLLMInvocation(
		frame,
		path,
		target,
		input,
		contextValue,
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
		child, err = NewSingleModelInvocationProgram(
			invocation.Defaults,
			invocation.InputActivityID,
			invocation.HistoryActivityIDs,
		)
	case WorkflowTargetAgent:
		child, err = NewAgentLoopInvocationProgram(
			invocation.Defaults,
			invocation.InputActivityID,
			invocation.HistoryActivityIDs,
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
		Path:     path,
		Program:  child,
		Deadline: cloneTime(state.Deadline),
		Handoff:  handoff,
	}}
	return transition, nil
}

func (i *WorkflowInterpreter) handoffWorkflowAgentTarget(
	frame ProgramFrame,
	program WorkflowProgram,
	state WorkflowState,
	path string,
	targetName string,
	target WorkflowTarget,
	input CanonicalValue,
	contextValue CanonicalValue,
	call WorkflowCall,
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
		path,
		targetName,
		target,
		input,
		contextValue,
		&ConversationHandoffIntent{
			ID:                  "workflow-handoff:" + string(frame.ID) + ":" + path,
			ConversationID:      conversationID,
			Owner:               owner,
			TargetAgent:         targetAgent,
			RestoreOnCompletion: call.OnCompletion != "retain",
		},
	)
}
