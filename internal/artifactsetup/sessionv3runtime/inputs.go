package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) buildInputs(req sessionv3.WorkDispatchRequest) ([]inferenceSpec.InputUnion, error) {
	payload := req.WorkItem.Input.LLM
	if payload == nil {
		return nil, errors.New("LLM payload is missing")
	}
	session := req.Session
	execution := req.Execution
	inputs := []inferenceSpec.InputUnion{}
	history := selectHistory(session, payload)
	for _, activity := range history {
		inputs = append(inputs, activityToInput(activity))
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

func selectHistory(session sessionv3.Session, payload *sessionv3.LLMWork) []sessionv3.Activity {
	_ = payload
	out := []sessionv3.Activity{}
	countUser := 0
	for _, a := range slices.Backward(session.Timeline) {

		if a.Kind != sessionv3.ActivityKindMessage || a.Message == nil {
			continue
		}
		if a.ID == payload.InputActivityID {
			continue
		}
		out = append([]sessionv3.Activity{a}, out...)
		if a.Message.Role == sessionv3.MessageRoleUser {
			countUser++
		}
		if !session.Defaults.History.All && countUser > session.Defaults.History.IncludePreviousUserTurns {
			break
		}
		if len(out) >= 64 {
			break
		}
	}
	return out
}

func findActivity(session sessionv3.Session, id sessionv3.ActivityID) *sessionv3.Activity {
	for i := range session.Timeline {
		if session.Timeline[i].ID == id {
			v := session.Timeline[i]
			return &v
		}
	}
	return nil
}

func findWorkItem(execution sessionv3.Execution, id sessionv3.WorkItemID) *sessionv3.WorkItem {
	for i := range execution.WorkItems {
		if execution.WorkItems[i].ID == id {
			v := execution.WorkItems[i]
			return &v
		}
	}
	return nil
}

func activityToInput(a sessionv3.Activity) inferenceSpec.InputUnion {
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
	if a.Message != nil && a.Message.Role == sessionv3.MessageRoleAssistant {
		role = inferenceSpec.RoleAssistant
	}
	return inferenceSpec.InputUnion{
		Kind: inferenceSpec.InputKindInputMessage,
		InputMessage: &inferenceSpec.InputOutputContent{
			ID:       string(a.ID),
			Role:     role,
			Status:   inferenceSpec.StatusNone,
			Contents: contents,
		},
	}
}

func attachmentToContentItem(a sessionv3.Attachment) (inferenceSpec.InputOutputContentItemUnion, bool) {
	switch a.Kind {
	case sessionv3.AttachmentKindText:
		if strings.TrimSpace(a.Text) == "" {
			return inferenceSpec.InputOutputContentItemUnion{}, false
		}
		return inferenceSpec.InputOutputContentItemUnion{
			Kind:     inferenceSpec.ContentItemKindText,
			TextItem: &inferenceSpec.ContentItemText{Text: a.Text},
		}, true
	case sessionv3.AttachmentKindImage:
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
	case sessionv3.AttachmentKindFile:
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
	case sessionv3.AttachmentKindURL:
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

func toolResultToInput(item sessionv3.WorkItem) inferenceSpec.InputUnion {
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
			IsError:  item.Output.Tool.Kind != sessionv3.ToolOutputKindSuccess,
			Contents: []inferenceSpec.ToolOutputItemUnion{
				{
					Kind:     inferenceSpec.ContentItemKindText,
					TextItem: &inferenceSpec.ContentItemText{Text: text},
				},
			},
		},
	}
}

func (b *Binder) systemPromptText(_ context.Context, req sessionv3.WorkDispatchRequest) string {
	parts := []string{}
	payload := req.WorkItem.Input.LLM
	if payload != nil {
		for _, inst := range payload.SystemInstructions {
			if strings.TrimSpace(inst.Content) != "" {
				parts = append(parts, inst.Content)
			}
		}
	}
	// Built-in programs copy Session defaults into LLMWork. Falling back only
	// when the work item has no explicit instructions prevents duplicated
	// system prompts on every Agent-loop round.
	if len(parts) == 0 {
		for _, inst := range req.Session.Defaults.SystemInstructions {
			if strings.TrimSpace(inst.Content) != "" {
				parts = append(parts, inst.Content)
			}
		}
	}
	return strings.Join(parts, "\n\n")
}
