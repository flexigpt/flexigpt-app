package sessionv3runtime

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) buildInputs(
	req sessionv3Runtime.WorkDispatchRequest,
	defaults sessionv3Runtime.SessionDefaults,
	injectedContext []string,
	preparedActivities []sessionv3Runtime.Activity,
) ([]inferenceSpec.InputUnion, error) {
	payload := req.WorkItem.Input.LLM
	if payload == nil {
		return nil, errors.New("LLM payload is missing")
	}

	execution := req.Execution
	activities := preparedActivities
	if len(activities) == 0 {
		var err error
		activities, err = llmInputActivitySequence(
			req.Session,
			req.Execution,
			req.WorkItem,
			payload,
			defaults.History,
		)
		if err != nil {
			return nil, err
		}
	}
	if len(activities) == 0 {
		return nil, errors.New("LLM has no semantic activities")
	}

	inputs := make([]inferenceSpec.InputUnion, 0, len(activities))
	emittedToolResults := make(map[sessionv3Runtime.WorkItemID]struct{})
	appendActivity := func(activity sessionv3Runtime.Activity) error {
		value, inputErr := durableActivityInputs(req.Session, execution, activity)
		if inputErr != nil {
			return inputErr
		}
		inputs = append(inputs, value...)
		for _, item := range completedToolResultItemsForAssistant(
			req.Session,
			execution,
			activity,
		) {
			if _, found := emittedToolResults[item.ID]; found {
				continue
			}
			inputs = append(inputs, toolResultToInput(item))
			emittedToolResults[item.ID] = struct{}{}
		}
		return nil
	}

	contextInserted := false
	for _, activity := range activities {
		if activity.ID == payload.InputActivityID {
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
			contextInserted = true
		}
		if activity.ID != payload.InputActivityID {
			contextValues, contextErr := durableActivityContext(
				req.Session,
				activity,
			)
			if contextErr != nil {
				return nil, contextErr
			}
			for index, value := range contextValues {
				inputs = append(inputs, injectedContextInput(
					activity.ID,
					index,
					value,
				))
			}
		}
		if err := appendActivity(activity); err != nil {
			return nil, err
		}
	}
	if !contextInserted {
		return nil, errors.New("LLM input activity is absent from its semantic sequence")
	}

	for _, id := range payload.ContinuationWorkItemIDs {
		if _, found := emittedToolResults[id]; found {
			continue
		}
		item := findSessionWorkItem(req.Session, execution, id)
		if item == nil || !completedSemanticToolOutput(*item) {
			continue
		}
		inputs = append(inputs, toolResultToInput(*item))
		emittedToolResults[id] = struct{}{}
	}
	if len(inputs) == 0 {
		return nil, errors.New("LLM has no inputs")
	}
	return inputs, nil
}

func llmInputActivitySequence(
	session sessionv3Runtime.Session,
	execution sessionv3Runtime.Execution,
	workItem sessionv3Runtime.WorkItem,
	payload *sessionv3Runtime.LLMWork,
	historySelection sessionv3Runtime.HistorySelection,
) ([]sessionv3Runtime.Activity, error) {
	currentIndex := -1
	for index := range session.Timeline {
		if session.Timeline[index].ID == payload.InputActivityID {
			currentIndex = index
			break
		}
	}
	if currentIndex < 0 {
		return nil, fmt.Errorf(
			"LLM input activity %q not found",
			payload.InputActivityID,
		)
	}
	current := session.Timeline[currentIndex]
	if payload.IsolatedInput {
		output := make(
			[]sessionv3Runtime.Activity,
			0,
			len(payload.HistoryActivityIDs)+1,
		)
		for _, activityID := range payload.HistoryActivityIDs {
			activity := findActivity(session, activityID)
			if activity == nil {
				return nil, fmt.Errorf(
					"isolated LLM history activity %q not found",
					activityID,
				)
			}
			output = append(output, *activity)
		}
		for _, activity := range session.Timeline[currentIndex+1:] {
			if activity.Kind != sessionv3Runtime.ActivityKindMessage ||
				activity.Message == nil ||
				activity.Message.Role != sessionv3Runtime.MessageRoleAssistant ||
				activity.ExecutionID != execution.ID ||
				activity.FrameID != workItem.FrameID {
				continue
			}
			output = append(output, activity)
		}
		output = append(output, current)
		for _, activity := range session.Timeline[currentIndex+1:] {
			if activity.Kind != sessionv3Runtime.ActivityKindMessage ||
				activity.Message == nil ||
				activity.Message.Role != sessionv3Runtime.MessageRoleAssistant ||
				activity.ExecutionID != execution.ID ||
				activity.FrameID != workItem.FrameID {
				continue
			}
			output = append(output, activity)
		}
		return output, nil
	}

	history := selectHistory(
		session,
		currentIndex,
		historySelection,
	)
	history = append(history, current)
	for _, activity := range session.Timeline[currentIndex+1:] {
		if activity.Kind != sessionv3Runtime.ActivityKindMessage ||
			activity.Message == nil ||
			activity.Message.Role != sessionv3Runtime.MessageRoleAssistant ||
			activity.ExecutionID != execution.ID ||
			activity.FrameID != workItem.FrameID {
			continue
		}
		history = append(history, activity)
	}
	return history, nil
}

func selectHistory(
	session sessionv3Runtime.Session,
	currentIndex int,
	selection sessionv3Runtime.HistorySelection,
) []sessionv3Runtime.Activity {
	if currentIndex <= 0 {
		return nil
	}
	prefix := session.Timeline[:currentIndex]
	if selection.All {
		output := make([]sessionv3Runtime.Activity, 0, len(prefix))
		for _, activity := range prefix {
			if activity.Kind != sessionv3Runtime.ActivityKindMessage ||
				activity.Message == nil ||
				activity.Visibility == sessionv3Runtime.ActivityVisibilityAudit {
				continue
			}
			output = append(output, activity)
		}
		return output
	}

	requestedTurns := max(selection.IncludePreviousUserTurns, 0)
	if requestedTurns == 0 {
		return durableSystemHistory(prefix)
	}

	reversed := make([]sessionv3Runtime.Activity, 0)
	selectedStart := len(prefix)
	userTurns := 0
	for index, activity := range slices.Backward(prefix) {
		if activity.Kind != sessionv3Runtime.ActivityKindMessage ||
			activity.Message == nil ||
			activity.Visibility == sessionv3Runtime.ActivityVisibilityAudit {
			continue
		}
		reversed = append(reversed, activity)
		selectedStart = index
		if activity.Message.Role != sessionv3Runtime.MessageRoleUser {
			continue
		}
		userTurns++
		if userTurns >= requestedTurns {
			break
		}
	}

	output := durableSystemHistory(prefix[:selectedStart])
	for _, r := range slices.Backward(reversed) {
		output = append(output, r)
	}
	return output
}

func durableSystemHistory(
	activities []sessionv3Runtime.Activity,
) []sessionv3Runtime.Activity {
	output := make([]sessionv3Runtime.Activity, 0)
	for _, activity := range activities {
		if activity.Kind != sessionv3Runtime.ActivityKindMessage ||
			activity.Message == nil ||
			activity.Visibility == sessionv3Runtime.ActivityVisibilityAudit {
			continue
		}
		switch activity.Message.Role {
		case sessionv3Runtime.MessageRoleSystem,
			sessionv3Runtime.MessageRoleDeveloper:
			output = append(output, activity)
		default:
		}
	}
	return output
}

func completedToolResultItemsForAssistant(
	session sessionv3Runtime.Session,
	execution sessionv3Runtime.Execution,
	activity sessionv3Runtime.Activity,
) []sessionv3Runtime.WorkItem {
	if activity.Kind != sessionv3Runtime.ActivityKindMessage ||
		activity.Message == nil ||
		activity.Message.Role != sessionv3Runtime.MessageRoleAssistant ||
		activity.WorkItemID == "" {
		return nil
	}
	owner := execution
	for _, candidate := range session.Executions {
		if candidate.ID == activity.ExecutionID {
			owner = candidate
			break
		}
	}
	return completedToolResultItemsForOrigin(owner, activity.WorkItemID)
}

func completedToolResultItemsForOrigin(
	execution sessionv3Runtime.Execution,
	originID sessionv3Runtime.WorkItemID,
) []sessionv3Runtime.WorkItem {
	candidates := make([]sessionv3Runtime.WorkItem, 0)
	for _, item := range execution.WorkItems {
		if item.Type != sessionv3Runtime.WorkItemTypeTool ||
			item.OriginWorkItemID != originID ||
			!completedSemanticToolOutput(item) {
			continue
		}
		candidates = append(candidates, item)
	}
	if len(candidates) == 0 {
		return nil
	}

	ordered := make([]sessionv3Runtime.WorkItem, 0, len(candidates))
	used := make(map[sessionv3Runtime.WorkItemID]struct{}, len(candidates))
	origin := findWorkItem(execution, originID)
	if origin != nil &&
		origin.Output != nil &&
		origin.Output.LLM != nil {
		for _, call := range origin.Output.LLM.ToolCalls {
			for _, item := range candidates {
				if _, found := used[item.ID]; found ||
					item.Input.Tool == nil ||
					!sameToolCallIdentity(item.Input.Tool.Call, call) {
					continue
				}
				ordered = append(ordered, item)
				used[item.ID] = struct{}{}
				break
			}
		}
	}
	for _, item := range candidates {
		if _, found := used[item.ID]; found {
			continue
		}
		ordered = append(ordered, item)
	}
	return ordered
}

func completedSemanticToolOutput(item sessionv3Runtime.WorkItem) bool {
	switch item.State {
	case sessionv3Runtime.WorkItemStateCompleted,
		sessionv3Runtime.WorkItemStateFailed:
	default:
		return false
	}
	return item.EffectState != sessionv3Runtime.EffectStateUncertain &&
		item.Output != nil &&
		item.Output.Tool != nil
}

func sameToolCallIdentity(
	left sessionv3Runtime.ToolCall,
	right sessionv3Runtime.ToolCall,
) bool {
	if left.ID != "" && left.ID == right.ID {
		return true
	}
	return left.ProviderCallID != "" &&
		left.ProviderCallID == right.ProviderCallID
}

func findActivity(
	session sessionv3Runtime.Session,
	id sessionv3Runtime.ActivityID,
) *sessionv3Runtime.Activity {
	for i := range session.Timeline {
		if session.Timeline[i].ID == id {
			v := session.Timeline[i]
			return &v
		}
	}
	return nil
}

func findWorkItem(
	execution sessionv3Runtime.Execution,
	id sessionv3Runtime.WorkItemID,
) *sessionv3Runtime.WorkItem {
	for i := range execution.WorkItems {
		if execution.WorkItems[i].ID == id {
			v := execution.WorkItems[i]
			return &v
		}
	}
	return nil
}

func findSessionWorkItem(
	session sessionv3Runtime.Session,
	execution sessionv3Runtime.Execution,
	id sessionv3Runtime.WorkItemID,
) *sessionv3Runtime.WorkItem {
	if item := findWorkItem(execution, id); item != nil {
		return item
	}
	for _, candidate := range session.Executions {
		if candidate.ID == execution.ID {
			continue
		}
		if item := findWorkItem(candidate, id); item != nil {
			return item
		}
	}
	return nil
}

func durableActivityContext(
	session sessionv3Runtime.Session,
	activity sessionv3Runtime.Activity,
) ([]string, error) {
	if activity.Kind != sessionv3Runtime.ActivityKindMessage ||
		activity.Message == nil ||
		activity.Message.Role != sessionv3Runtime.MessageRoleUser {
		return nil, nil
	}

	for _, execution := range session.Executions {
		for _, item := range execution.WorkItems {
			if item.Type != sessionv3Runtime.WorkItemTypeLLM ||
				item.Input.LLM == nil ||
				item.Input.LLM.InputActivityID != activity.ID ||
				item.Binding == nil ||
				item.Binding.Plan == nil {
				continue
			}
			values := make([]string, 0, len(item.Binding.Plan.Context))
			for _, value := range item.Binding.Plan.Context {
				text, err := jsonutil.DecodeCanonicalValue[string](value.Value)
				if err != nil {
					text = string(value.Value)
				}
				text = strings.TrimSpace(text)
				if text != "" {
					values = append(values, text)
				}
			}
			return values, nil
		}
	}
	return nil, nil
}

func durableActivityInputs(
	session sessionv3Runtime.Session,
	execution sessionv3Runtime.Execution,
	activity sessionv3Runtime.Activity,
) ([]inferenceSpec.InputUnion, error) {
	if activity.Message != nil &&
		activity.Message.Role == sessionv3Runtime.MessageRoleAssistant &&
		activity.WorkItemID != "" {
		item := findSessionWorkItem(session, execution, activity.WorkItemID)
		if item != nil &&
			item.Type == sessionv3Runtime.WorkItemTypeLLM &&
			item.Output != nil &&
			item.Output.LLM != nil &&
			len(item.Output.LLM.ProtocolOutputs) != 0 {
			return providerOutputsAsInputs(item.Output.LLM.ProtocolOutputs)
		}
	}

	output := make([]inferenceSpec.InputUnion, 0, 1)
	output = append(output, activityToInput(activity))
	output = append(output, activityToolResultInputs(activity)...)
	return output, nil
}

func providerOutputsAsInputs(
	values []jsonutil.CanonicalValue,
) ([]inferenceSpec.InputUnion, error) {
	output := make([]inferenceSpec.InputUnion, 0, len(values))
	for index := range values {
		value, err := jsonutil.DecodeCanonicalValue[inferenceSpec.OutputUnion](
			values[index],
		)
		if err != nil {
			return nil, fmt.Errorf(
				"decode durable provider output %d: %w",
				index,
				err,
			)
		}
		if input := providerOutputAsInput(value, index); input != nil {
			output = append(output, *input)
		}
	}
	return output, nil
}

func providerOutputAsInput(
	value inferenceSpec.OutputUnion,
	fallbackIndex int,
) *inferenceSpec.InputUnion {
	switch value.Kind {
	case inferenceSpec.OutputKindOutputMessage:
		return &inferenceSpec.InputUnion{
			Kind:          inferenceSpec.InputKindOutputMessage,
			OutputMessage: value.OutputMessage,
		}
	case inferenceSpec.OutputKindReasoningMessage:
		return &inferenceSpec.InputUnion{
			Kind:             inferenceSpec.InputKindReasoningMessage,
			ReasoningMessage: value.ReasoningMessage,
		}
	case inferenceSpec.OutputKindFunctionToolCall:
		return &inferenceSpec.InputUnion{
			Kind:             inferenceSpec.InputKindFunctionToolCall,
			FunctionToolCall: value.FunctionToolCall,
		}
	case inferenceSpec.OutputKindCustomToolCall:
		return &inferenceSpec.InputUnion{
			Kind:           inferenceSpec.InputKindCustomToolCall,
			CustomToolCall: value.CustomToolCall,
		}
	case inferenceSpec.OutputKindWebSearchToolCall:
		return &inferenceSpec.InputUnion{
			Kind:              inferenceSpec.InputKindWebSearchToolCall,
			WebSearchToolCall: value.WebSearchToolCall,
		}
	case inferenceSpec.OutputKindWebSearchToolOutput:
		return &inferenceSpec.InputUnion{
			Kind:                inferenceSpec.InputKindWebSearchToolOutput,
			WebSearchToolOutput: value.WebSearchToolOutput,
		}
	default:
		return &inferenceSpec.InputUnion{
			Kind: inferenceSpec.InputKindOutputMessage,
			OutputMessage: &inferenceSpec.InputOutputContent{
				ID:     fmt.Sprintf("runtime-unavailable-output:%d", fallbackIndex),
				Role:   inferenceSpec.RoleAssistant,
				Status: inferenceSpec.StatusCompleted,
				Contents: []inferenceSpec.InputOutputContentItemUnion{{
					Kind: inferenceSpec.ContentItemKindText,
					TextItem: &inferenceSpec.ContentItemText{
						Text: fmt.Sprintf(
							"[Provider output kind %q is not available in replay.]",
							value.Kind,
						),
					},
				}},
			},
		}
	}
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
		output = append(output, toolOutputInput(
			result.WorkItemID,
			sessionv3Runtime.ToolCall{
				ID:             result.CallID,
				ProviderCallID: result.ProviderCallID,
				ChoiceID:       result.ChoiceID,
				Name:           result.Name,
				Type:           result.Type,
			},
			sessionv3Runtime.ToolOutput{
				Kind:       result.Kind,
				Text:       result.Text,
				Contents:   cloneToolOutputContents(result.Contents),
				Structured: result.Structured.Clone(),
				Error:      result.Error,
			},
		))
	}
	return output
}

func appContextInputText(
	update sessionv3Runtime.AppContextUpdate,
) string {
	value := jsonutil.MustCanonicalValue(struct {
		Source            string                               `json:"source"`
		ResourceURI       string                               `json:"resourceURI,omitempty"`
		Content           []sessionv3Runtime.AppContextContent `json:"content,omitempty"`
		StructuredContent jsonutil.CanonicalValue              `json:"structuredContent,omitempty"`
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
	return toolOutputInput(
		item.ID,
		item.Input.Tool.Call,
		*item.Output.Tool,
	)
}

func toolOutputInput(
	resultID sessionv3Runtime.WorkItemID,
	call sessionv3Runtime.ToolCall,
	result sessionv3Runtime.ToolOutput,
) inferenceSpec.InputUnion {
	contents := cloneToolOutputContents(result.Contents)
	if len(contents) == 0 {
		contents = []inferenceSpec.ToolOutputItemUnion{{
			Kind: inferenceSpec.ContentItemKindText,
			TextItem: &inferenceSpec.ContentItemText{
				Text: toolOutputText(result),
			},
		}}
	}

	toolType := call.Type
	if toolType == "" {
		toolType = inferenceSpec.ToolTypeFunction
	}
	output := &inferenceSpec.ToolOutput{
		Type:     toolType,
		ID:       string(resultID),
		CallID:   firstText(call.ProviderCallID, call.ID),
		ChoiceID: call.ChoiceID,
		Name:     call.Name,
		Role:     inferenceSpec.RoleTool,
		Status:   inferenceSpec.StatusCompleted,
		IsError:  result.Kind != sessionv3Runtime.ToolOutputKindSuccess,
		Contents: contents,
	}
	if toolType == inferenceSpec.ToolTypeCustom {
		return inferenceSpec.InputUnion{
			Kind:             inferenceSpec.InputKindCustomToolOutput,
			CustomToolOutput: output,
		}
	}
	return inferenceSpec.InputUnion{
		Kind:               inferenceSpec.InputKindFunctionToolOutput,
		FunctionToolOutput: output,
	}
}

func cloneToolOutputContents(
	values []inferenceSpec.ToolOutputItemUnion,
) []inferenceSpec.ToolOutputItemUnion {
	return jsonutil.CloneJSONInValue(values)
}

func toolOutputText(result sessionv3Runtime.ToolOutput) string {
	if strings.TrimSpace(result.Text) != "" {
		return result.Text
	}
	if strings.TrimSpace(result.Error) != "" {
		return result.Error
	}
	if len(result.Structured) != 0 {
		return string(result.Structured)
	}
	return "Tool completed."
}
