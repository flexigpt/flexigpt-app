package sessionv3runtime

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const workflowPortContextPrefix = "workflow port context"

type workflowInvocationRequest struct {
	Messages []workflowInvocationMessage `json:"messages"`
}

type workflowInvocationContext struct {
	Messages     []workflowInvocationMessage `json:"messages,omitempty"`
	Instructions []string                    `json:"instructions,omitempty"`
}

type workflowInvocationMessage struct {
	Role        sessionv3Runtime.MessageRole  `json:"role"`
	Text        string                        `json:"text,omitempty"`
	Attachments []sessionv3Runtime.Attachment `json:"attachments,omitempty"`
}

type workflowInvocationPortContext struct {
	Port   string                            `json:"port"`
	Values []sessionv3Runtime.CanonicalValue `json:"values"`
}

func (b *Binder) buildInputs(
	req sessionv3Runtime.WorkDispatchRequest,
	injectedContext []string,
) ([]inferenceSpec.InputUnion, error) {
	payload := req.WorkItem.Input.LLM
	if payload == nil {
		return nil, errors.New("LLM payload is missing")
	}
	if payload.Invocation != nil {
		return workflowInvocationInputs(
			*payload.Invocation,
			injectedContext,
			req.WorkItem.PortInputs,
		)
	}
	session := req.Session
	execution := req.Execution
	inputs := []inferenceSpec.InputUnion{}
	history := selectHistory(session, payload)
	for _, activity := range history {
		inputs = append(inputs, activityToInput(activity))
	}
	for index, value := range injectedContext {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		inputs = append(inputs, injectedContextInput(
			payload.InputActivityID,
			index,
			value,
		))
	}
	current := findActivity(session, payload.InputActivityID)
	if current == nil {
		return nil, fmt.Errorf("LLM input activity %q not found", payload.InputActivityID)
	}
	inputs = append(inputs, activityToInput(*current))
	for _, id := range payload.ContinuationWorkItemIDs {
		item := findWorkItem(execution, id)
		if item == nil || item.Output == nil || item.Output.Tool == nil {
			continue
		}
		inputs = append(inputs, toolResultToInput(*item))
	}
	if len(inputs) == 0 {
		return nil, errors.New("LLM has no inputs")
	}
	return inputs, nil
}

func workflowInvocationInputs(
	plan sessionv3Runtime.LLMInvocationPlan,
	injectedContext []string,
	portInputs []sessionv3Runtime.PortInput,
) ([]inferenceSpec.InputUnion, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}

	request, err := sessionv3Runtime.DecodeCanonicalValue[workflowInvocationRequest](
		plan.Input,
	)
	if err != nil {
		return nil, fmt.Errorf("decode workflow model request: %w", err)
	}
	if len(request.Messages) == 0 {
		return nil, errors.New("workflow model request has no messages")
	}

	contextValue := workflowInvocationContext{}
	if len(plan.Context) != 0 {
		contextValue, err = sessionv3Runtime.DecodeCanonicalValue[workflowInvocationContext](
			plan.Context,
		)
		if err != nil {
			return nil, fmt.Errorf("decode workflow model context: %w", err)
		}
	}

	output := make(
		[]inferenceSpec.InputUnion,
		0,
		len(injectedContext)+
			len(portInputs)+
			len(contextValue.Instructions)+len(contextValue.Messages)+len(request.Messages),
	)
	nextID := 0
	appendMessage := func(message workflowInvocationMessage) error {
		input, err := workflowInvocationMessageInput(nextID, message)
		if err != nil {
			return err
		}
		nextID++
		output = append(output, input)
		return nil
	}

	for _, instruction := range contextValue.Instructions {
		if strings.TrimSpace(instruction) == "" {
			continue
		}
		if err := appendMessage(workflowInvocationMessage{
			Role: sessionv3Runtime.MessageRoleSystem,
			Text: instruction,
		}); err != nil {
			return nil, err
		}
	}
	for _, value := range injectedContext {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if err := appendMessage(workflowInvocationMessage{
			Role: sessionv3Runtime.MessageRoleUser,
			Text: value,
		}); err != nil {
			return nil, err
		}
	}
	for _, input := range portInputs {
		if len(input.Values) == 0 {
			continue
		}
		value := sessionv3Runtime.MustCanonicalValue(
			workflowInvocationPortContext{
				Port:   input.Name,
				Values: input.Values,
			},
		)
		if err := appendMessage(workflowInvocationMessage{
			Role: sessionv3Runtime.MessageRoleDeveloper,
			Text: workflowPortContextPrefix + "\n" + string(value),
		}); err != nil {
			return nil, err
		}
	}
	for _, message := range contextValue.Messages {
		if err := appendMessage(message); err != nil {
			return nil, err
		}
	}
	for _, message := range request.Messages {
		if err := appendMessage(message); err != nil {
			return nil, err
		}
	}

	if len(output) == 0 {
		return nil, errors.New("workflow model request produced no provider inputs")
	}
	return output, nil
}

func workflowInvocationMessageInput(
	index int,
	message workflowInvocationMessage,
) (inferenceSpec.InputUnion, error) {
	switch message.Role {
	case sessionv3Runtime.MessageRoleSystem,
		sessionv3Runtime.MessageRoleDeveloper,
		sessionv3Runtime.MessageRoleUser,
		sessionv3Runtime.MessageRoleAssistant:
	default:
		return inferenceSpec.InputUnion{}, errors.New("workflow model message role is invalid")
	}

	activity := sessionv3Runtime.Activity{
		ID: sessionv3Runtime.ActivityID(
			fmt.Sprintf("workflow-invocation-%d", index),
		),
		Kind: sessionv3Runtime.ActivityKindMessage,
		Message: &sessionv3Runtime.MessageActivity{
			Role:        message.Role,
			Text:        message.Text,
			Attachments: append([]sessionv3Runtime.Attachment(nil), message.Attachments...),
		},
	}
	return activityToInput(activity), nil
}

func selectHistory(session sessionv3Runtime.Session, payload *sessionv3Runtime.LLMWork) []sessionv3Runtime.Activity {
	reversed := make([]sessionv3Runtime.Activity, 0)
	countUser := 0
	for _, a := range slices.Backward(session.Timeline) {
		if a.Kind != sessionv3Runtime.ActivityKindMessage || a.Message == nil {
			continue
		}
		if a.ID == payload.InputActivityID {
			continue
		}
		reversed = append(reversed, a)
		if a.Message.Role == sessionv3Runtime.MessageRoleUser {
			countUser++
		}
		if !session.Defaults.History.All && countUser > session.Defaults.History.IncludePreviousUserTurns {
			break
		}
	}

	out := make([]sessionv3Runtime.Activity, len(reversed))
	for index := range reversed {
		out[len(reversed)-1-index] = reversed[index]
	}
	return out
}

func findActivity(session sessionv3Runtime.Session, id sessionv3Runtime.ActivityID) *sessionv3Runtime.Activity {
	for i := range session.Timeline {
		if session.Timeline[i].ID == id {
			v := session.Timeline[i]
			return &v
		}
	}
	return nil
}

func findWorkItem(execution sessionv3Runtime.Execution, id sessionv3Runtime.WorkItemID) *sessionv3Runtime.WorkItem {
	for i := range execution.WorkItems {
		if execution.WorkItems[i].ID == id {
			v := execution.WorkItems[i]
			return &v
		}
	}
	return nil
}

func activityToInput(a sessionv3Runtime.Activity) inferenceSpec.InputUnion {
	text := ""
	if a.Message != nil {
		text = a.Message.Text
	}
	contents := []inferenceSpec.InputOutputContentItemUnion{}
	if strings.TrimSpace(text) != "" {
		contents = append(
			contents,
			inferenceSpec.InputOutputContentItemUnion{
				Kind:     inferenceSpec.ContentItemKindText,
				TextItem: &inferenceSpec.ContentItemText{Text: text},
			},
		)
	}
	if a.Message != nil {
		for _, att := range a.Message.Attachments {
			if item, ok := attachmentToContentItem(att); ok {
				contents = append(contents, item)
			}
		}
	}

	role := inferenceSpec.RoleUser
	if a.Message != nil && a.Message.Role != "" {
		role = inferenceSpec.RoleEnum(a.Message.Role)
	}
	message := &inferenceSpec.InputOutputContent{
		ID:       string(a.ID),
		Role:     role,
		Status:   inferenceSpec.StatusNone,
		Contents: contents,
	}

	if role == inferenceSpec.RoleAssistant {
		message.Status = inferenceSpec.StatusCompleted
		return inferenceSpec.InputUnion{
			Kind:          inferenceSpec.InputKindOutputMessage,
			OutputMessage: message,
		}
	}

	return inferenceSpec.InputUnion{
		Kind:         inferenceSpec.InputKindInputMessage,
		InputMessage: message,
	}
}

func injectedContextInput(
	activityID sessionv3Runtime.ActivityID,
	index int,
	text string,
) inferenceSpec.InputUnion {
	return inferenceSpec.InputUnion{
		Kind: inferenceSpec.InputKindInputMessage,
		InputMessage: &inferenceSpec.InputOutputContent{
			ID: fmt.Sprintf(
				"runtime-context:%s:%d",
				activityID,
				index,
			),
			Role:   inferenceSpec.RoleUser,
			Status: inferenceSpec.StatusNone,
			Contents: []inferenceSpec.InputOutputContentItemUnion{{
				Kind: inferenceSpec.ContentItemKindText,
				TextItem: &inferenceSpec.ContentItemText{
					Text: text,
				},
			}},
		},
	}
}

func attachmentToContentItem(a sessionv3Runtime.Attachment) (inferenceSpec.InputOutputContentItemUnion, bool) {
	switch a.Kind {
	case sessionv3Runtime.AttachmentKindText:
		if strings.TrimSpace(a.Text) == "" {
			return inferenceSpec.InputOutputContentItemUnion{}, false
		}
		return inferenceSpec.InputOutputContentItemUnion{
			Kind:     inferenceSpec.ContentItemKindText,
			TextItem: &inferenceSpec.ContentItemText{Text: a.Text},
		}, true
	case sessionv3Runtime.AttachmentKindImage:
		image := &inferenceSpec.ContentItemImage{ImageMIME: "image/png"}
		if a.MIMEType != "" {
			image.ImageMIME = a.MIMEType
		}
		if a.Base64Data != "" {
			image.ImageData = a.Base64Data
		}
		if a.URL != "" {
			image.ImageURL = a.URL
		}
		if image.ImageData == "" && image.ImageURL == "" {
			return inferenceSpec.InputOutputContentItemUnion{}, false
		}
		return inferenceSpec.InputOutputContentItemUnion{
			Kind:      inferenceSpec.ContentItemKindImage,
			ImageItem: image,
		}, true
	case sessionv3Runtime.AttachmentKindFile:
		file := &inferenceSpec.ContentItemFile{FileMIME: "application/octet-stream"}
		if a.MIMEType != "" {
			file.FileMIME = a.MIMEType
		}
		if a.Label != "" {
			file.FileName = a.Label
		}
		if a.Base64Data != "" {
			file.FileData = a.Base64Data
		}
		if a.URL != "" {
			file.FileURL = a.URL
		}
		if file.FileData == "" && file.FileURL == "" && strings.TrimSpace(a.Text) != "" {
			return inferenceSpec.InputOutputContentItemUnion{
				Kind:     inferenceSpec.ContentItemKindText,
				TextItem: &inferenceSpec.ContentItemText{Text: a.Text},
			}, true
		}
		if file.FileData == "" && file.FileURL == "" {
			return inferenceSpec.InputOutputContentItemUnion{}, false
		}
		return inferenceSpec.InputOutputContentItemUnion{Kind: inferenceSpec.ContentItemKindFile, FileItem: file}, true
	case sessionv3Runtime.AttachmentKindURL:
		if strings.TrimSpace(a.URL) == "" && strings.TrimSpace(a.Text) == "" {
			return inferenceSpec.InputOutputContentItemUnion{}, false
		}
		text := strings.TrimSpace(a.URL)
		if text == "" {
			text = a.Text
		}
		return inferenceSpec.InputOutputContentItemUnion{
			Kind:     inferenceSpec.ContentItemKindText,
			TextItem: &inferenceSpec.ContentItemText{Text: text},
		}, true
	default:
		return inferenceSpec.InputOutputContentItemUnion{}, false
	}
}

func toolResultToInput(item sessionv3Runtime.WorkItem) inferenceSpec.InputUnion {
	text := ""
	if item.Output != nil && item.Output.Tool != nil {
		text = item.Output.Tool.Text
	}
	if strings.TrimSpace(text) == "" {
		text = "Tool completed."
	}
	call := item.Input.Tool.Call
	return inferenceSpec.InputUnion{
		Kind: inferenceSpec.InputKindFunctionToolOutput,
		FunctionToolOutput: &inferenceSpec.ToolOutput{
			Type:     inferenceSpec.ToolTypeFunction,
			ID:       string(item.ID),
			CallID:   firstText(call.ProviderCallID, call.ID),
			ChoiceID: call.ChoiceID,
			Name:     call.Name,
			Role:     inferenceSpec.RoleTool,
			Status:   inferenceSpec.StatusCompleted,
			IsError:  item.Output.Tool.Kind != sessionv3Runtime.ToolOutputKindSuccess,
			Contents: []inferenceSpec.ToolOutputItemUnion{
				{
					Kind:     inferenceSpec.ContentItemKindText,
					TextItem: &inferenceSpec.ContentItemText{Text: text},
				},
			},
		},
	}
}
