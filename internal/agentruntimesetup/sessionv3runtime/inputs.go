package sessionv3runtime

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) buildInputs(
	req sessionv3Runtime.WorkDispatchRequest,
	injectedContext []string,
) ([]inferenceSpec.InputUnion, error) {
	payload := req.WorkItem.Input.LLM
	if payload == nil {
		return nil, errors.New("LLM payload is missing")
	}
	if payload.Invocation != nil {
		return nil, errors.New(
			"workflow model and agent invocation adapter is unavailable",
		)
	}

	session := req.Session
	execution := req.Execution
	inputs := []inferenceSpec.InputUnion{}
	history := selectHistory(session, payload)
	for _, activity := range history {
		inputs = append(inputs, activityToInput(activity))
		inputs = append(inputs, activityToolResultInputs(activity)...)
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
	inputs = append(inputs, activityToolResultInputs(*current)...)
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
		for _, update := range a.Message.AppContext {
			contents = append(
				contents,
				inferenceSpec.InputOutputContentItemUnion{
					Kind: inferenceSpec.ContentItemKindText,
					TextItem: &inferenceSpec.ContentItemText{
						Text: appContextInputText(update),
					},
				},
			)
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

func activityToolResultInputs(
	activity sessionv3Runtime.Activity,
) []inferenceSpec.InputUnion {
	if activity.Message == nil ||
		len(activity.Message.ToolResults) == 0 {
		return nil
	}
	output := make(
		[]inferenceSpec.InputUnion,
		0,
		len(activity.Message.ToolResults),
	)
	for _, result := range activity.Message.ToolResults {
		text := strings.TrimSpace(result.Text)
		if text == "" {
			text = strings.TrimSpace(result.Error)
		}
		if text == "" {
			text = "Tool completed."
		}
		output = append(output, inferenceSpec.InputUnion{
			Kind: inferenceSpec.InputKindFunctionToolOutput,
			FunctionToolOutput: &inferenceSpec.ToolOutput{
				Type:     inferenceSpec.ToolTypeFunction,
				ID:       string(result.WorkItemID),
				CallID:   firstText(result.ProviderCallID, result.CallID),
				ChoiceID: result.ChoiceID,
				Name:     result.Name,
				Role:     inferenceSpec.RoleTool,
				Status:   inferenceSpec.StatusCompleted,
				IsError:  result.Kind != sessionv3Runtime.ToolOutputKindSuccess,
				Contents: []inferenceSpec.ToolOutputItemUnion{{
					Kind: inferenceSpec.ContentItemKindText,
					TextItem: &inferenceSpec.ContentItemText{
						Text: text,
					},
				}},
			},
		})
	}
	return output
}

func appContextInputText(
	update sessionv3Runtime.AppContextUpdate,
) string {
	value := sessionv3Runtime.MustCanonicalValue(struct {
		Source            string                               `json:"source"`
		ResourceURI       string                               `json:"resourceURI,omitempty"`
		Content           []sessionv3Runtime.AppContextContent `json:"content,omitempty"`
		StructuredContent sessionv3Runtime.CanonicalValue      `json:"structuredContent,omitempty"`
	}{
		Source:            "approved MCP App context",
		ResourceURI:       update.ResourceURI,
		Content:           update.Content,
		StructuredContent: update.StructuredContent.Clone(),
	})
	return "The following context was explicitly approved from a Session-scoped MCP App.\n\n" +
		string(value)
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
