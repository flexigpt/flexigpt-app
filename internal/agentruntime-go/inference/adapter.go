package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	attachmentRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/attachment"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"

	inferenceRuntimeSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inference/spec"
	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const (
	emptyCompletionResponseCode = "empty_completion_response"

	wireKindField          = "kind"
	wireInputMessageField  = "inputMessage"
	wireOutputMessageField = "outputMessage"
	wireTextItemField      = "textItem"
	wireContentsField      = "contents"
	wireIDField            = "id"
	wireRoleField          = "role"
	wireStatusField        = "status"
	wireTextField          = "text"
	wireTypeField          = "type"
	wireChoiceIDField      = "choiceID"
	wireCallIDField        = "callID"
	wireNameField          = "name"
	wireIsErrorField       = "isError"

	wireInputMessageKind         = "inputMessage"
	wireOutputMessageKind        = "outputMessage"
	wireTextKind                 = "text"
	wireFunctionToolOutputKind   = "functionToolOutput"
	wireFunctionToolOutputField  = "functionToolOutput"
	wireCustomToolOutputKind     = "customToolOutput"
	wireCustomToolOutputField    = "customToolOutput"
	wireWebSearchToolOutputKind  = "webSearchToolOutput"
	wireWebSearchToolOutputField = "webSearchToolOutput"
	wireToolRole                 = "tool"
	wireCompletedStatus          = "completed"
)

type CompletionProvider interface {
	FetchCompletion(
		ctx context.Context,
		request *inferenceRuntimeSpec.CompletionRequest,
	) (*inferenceRuntimeSpec.CompletionResponse, error)
}

type AttachmentMaterializer interface {
	Materialize(
		ctx context.Context,
		references []sessionRuntime.AttachmentReference,
	) ([]attachmentRuntime.Attachment, error)
}

// Adapter translates Session-owned runtime records into the existing
// inferencewrapper request and response model. It deliberately owns the
// conversion boundary so the session core does not import conversation,
// artifact, attachment, or inferencewrapper packages.
type Adapter struct {
	provider    CompletionProvider
	attachments AttachmentMaterializer
}

func NewAdapter(
	provider CompletionProvider,
	attachments AttachmentMaterializer,
) (*Adapter, error) {
	if provider == nil {
		return nil, errors.New("session inference adapter requires a completion provider")
	}
	if attachments == nil {
		return nil, errors.New("session inference adapter requires an attachment materializer")
	}

	return &Adapter{
		provider:    provider,
		attachments: attachments,
	}, nil
}

func (a *Adapter) ExecuteModel(
	ctx context.Context,
	request sessionRuntime.ModelExecutionRequest,
	stream sessionRuntime.ModelStream,
) (sessionRuntime.ModelExecutionResult, error) {
	modelRef, err := artifactReference(request.Configuration.Model)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, err
	}

	history, err := a.conversationMessages(
		ctx,
		request.History,
		true,
		true,
	)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, err
	}

	currentValues, err := a.conversationMessages(
		ctx,
		[]sessionRuntime.TranscriptMessage{request.Current},
		true,
		false,
	)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, err
	}
	if len(currentValues) != 1 {
		return sessionRuntime.ModelExecutionResult{}, errors.New(
			"session inference adapter did not build a current conversation message",
		)
	}
	current := currentValues[0]

	toolSelections, err := convertToolSelections(
		request.Configuration.ToolSelections,
	)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, err
	}

	requestPatch, err := convertRequestPatch(
		request.Configuration.RequestPatch,
	)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, err
	}

	mcpContext, err := decodeOptional[conversationSpec.MCPConversationContext](
		request.Configuration.MCPContext,
	)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, fmt.Errorf(
			"decode session MCP context: %w",
			err,
		)
	}

	workspaceSelection, err := decodeOptional[conversationSpec.WorkspaceConversationSelection](
		request.Configuration.WorkspaceSelection,
	)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, fmt.Errorf(
			"decode session Workspace selection: %w",
			err,
		)
	}
	current.WorkspaceSelection = workspaceSelection

	enabledSkills, err := artifactReferences(
		request.Configuration.InstalledSkills,
		request.Configuration.WorkspaceSkills,
	)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, err
	}
	current.EnabledSkillRefs = enabledSkills

	mcpAppContext, err := decodeOptional[[]conversationSpec.MCPAppModelContextUpdate](request.Current.MCPAppContext)
	if err != nil {
		return sessionRuntime.ModelExecutionResult{}, fmt.Errorf(
			"decode session MCP App context: %w",
			err,
		)
	}
	if mcpAppContext != nil {
		current.MCPAppContextUpdates = *mcpAppContext
	}

	completionRequest := &inferenceRuntimeSpec.CompletionRequest{
		Model: inferenceRuntimeSpec.RuntimeModelRequest{
			Model:        modelRef,
			RequestPatch: requestPatch,
		},
		History:        history,
		Current:        current,
		ToolSelections: toolSelections,
		MCPContext:     mcpContext,
		SkillSessionID: request.SkillSessionID,
		OnStreamText: func(text string) error {
			if stream.OnText == nil {
				return nil
			}
			return stream.OnText(text)
		},
		OnStreamThinking: func(thinking string) error {
			if stream.OnThinking == nil {
				return nil
			}
			return stream.OnThinking(thinking)
		},
	}

	response, fetchErr := a.provider.FetchCompletion(ctx, completionRequest)
	result := sessionRuntime.ModelExecutionResult{}

	if response != nil && response.Body != nil {
		body := response.Body

		result.HydratedCurrentInputs = marshalInputUnions(
			body.HydratedCurrentInputs,
		)
		result.Metadata = marshalMetadata(
			body.MCPToolMappings,
			body.WorkspaceUsage,
		)

		if body.InferenceResponse != nil {
			inferenceResponse := body.InferenceResponse
			result.ProtocolOutputs = marshalOutputUnions(
				inferenceResponse.Outputs,
			)
			result.Error = inferenceError(
				inferenceResponse.Error,
			)

			text, thinking, calls := decodeOutputs(
				result.ProtocolOutputs,
				request.Configuration.ToolSelections,
				body.MCPToolMappings,
			)
			result.Text = text
			result.Thinking = thinking
			result.ToolCalls = calls
			result.Usable = strings.TrimSpace(text) != "" ||
				len(calls) != 0 ||
				hasUsableNonTextOutput(result.ProtocolOutputs)
		}
	}

	if fetchErr != nil {
		if result.Error == nil {
			result.Error = &sessionRuntime.OperationError{
				Code:    "completion_failed",
				Message: fetchErr.Error(),
			}
		}
		if response != nil {
			return result, nil
		}
		return result, fetchErr
	}

	if response == nil || response.Body == nil {
		return sessionRuntime.ModelExecutionResult{
			Error: &sessionRuntime.OperationError{
				Code:    emptyCompletionResponseCode,
				Message: "completion provider returned no response",
			},
		}, nil
	}

	if result.Error != nil && !result.Usable && len(result.ToolCalls) == 0 {
		return result, nil
	}

	return result, nil
}

func (a *Adapter) conversationMessages(
	ctx context.Context,
	messages []sessionRuntime.TranscriptMessage,
	materializeAttachments bool,
	embedAttachments bool,
) ([]conversationSpec.ConversationMessage, error) {
	output := make([]conversationSpec.ConversationMessage, 0, len(messages))

	for _, message := range messages {
		value, err := a.conversationMessage(
			ctx,
			message,
			materializeAttachments,
			embedAttachments,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}

	return output, nil
}

func (a *Adapter) conversationMessage(
	ctx context.Context,
	message sessionRuntime.TranscriptMessage,
	materializeAttachments bool,
	embedAttachments bool,
) (conversationSpec.ConversationMessage, error) {
	inputs, err := decodeInputUnions(message.ProtocolInputs)
	if err != nil {
		return conversationSpec.ConversationMessage{}, err
	}
	outputs, err := decodeOutputUnions(message.ProtocolOutputs)
	if err != nil {
		return conversationSpec.ConversationMessage{}, err
	}

	if len(inputs) == 0 && message.Role != sessionRuntime.MessageRoleAssistant {
		inputs, err = buildInputsFromTranscriptMessage(message)
		if err != nil {
			return conversationSpec.ConversationMessage{}, err
		}
	}
	if len(outputs) == 0 && message.Role == sessionRuntime.MessageRoleAssistant {
		outputs, err = buildOutputsFromTranscriptMessage(message)
		if err != nil {
			return conversationSpec.ConversationMessage{}, err
		}
	}

	value := conversationSpec.ConversationMessage{
		ID:        message.ID,
		CreatedAt: message.CreatedAt,
		Role:      inferenceSpec.RoleEnum(message.Role),
		Status:    inferenceSpec.Status(message.Status),
		Inputs:    inputs,
		Outputs:   outputs,
	}

	if materializeAttachments && len(message.Attachments) != 0 {
		attachments, materializeErr := a.attachments.Materialize(
			ctx,
			message.Attachments,
		)
		if materializeErr != nil {
			return conversationSpec.ConversationMessage{}, fmt.Errorf(
				"materialize session attachments: %w",
				materializeErr,
			)
		}

		if embedAttachments && message.Role == sessionRuntime.MessageRoleUser {
			value.Inputs, err = appendMaterializedAttachmentInputs(
				ctx,
				value.Inputs,
				attachments,
				message.ID,
			)
			if err != nil {
				return conversationSpec.ConversationMessage{}, err
			}
		} else {
			value.Attachments = attachments
		}
	}

	return value, nil
}

func appendMaterializedAttachmentInputs(
	ctx context.Context,
	inputs []inferenceSpec.InputUnion,
	attachments []attachmentRuntime.Attachment,
	messageID string,
) ([]inferenceSpec.InputUnion, error) {
	if len(attachments) == 0 || hasMaterializedAttachmentInput(inputs) {
		return inputs, nil
	}

	items, err := buildContentItemsFromAttachments(ctx, attachments)
	if err != nil || len(items) == 0 {
		return inputs, err
	}

	output := append([]inferenceSpec.InputUnion(nil), inputs...)
	for index := len(output) - 1; index >= 0; index-- {
		input := &output[index]
		if input.Kind != inferenceSpec.InputKindInputMessage ||
			input.InputMessage == nil ||
			input.InputMessage.Role != inferenceSpec.RoleUser {
			continue
		}

		input.InputMessage.Contents = append(
			input.InputMessage.Contents,
			items...,
		)
		return output, nil
	}

	output = append(output, inferenceSpec.InputUnion{
		Kind: inferenceSpec.InputKindInputMessage,
		InputMessage: &inferenceSpec.InputOutputContent{
			ID:       messageID,
			Role:     inferenceSpec.RoleUser,
			Status:   inferenceSpec.StatusNone,
			Contents: items,
		},
	})
	return output, nil
}

func hasMaterializedAttachmentInput(
	inputs []inferenceSpec.InputUnion,
) bool {
	for _, input := range inputs {
		if input.Kind != inferenceSpec.InputKindInputMessage ||
			input.InputMessage == nil {
			continue
		}

		for _, content := range input.InputMessage.Contents {
			switch content.Kind {
			case inferenceSpec.ContentItemKindImage,
				inferenceSpec.ContentItemKindFile:
				return true

			case inferenceSpec.ContentItemKindText:
				if content.TextItem != nil &&
					strings.HasPrefix(
						strings.TrimSpace(content.TextItem.Text),
						"<<<FILE",
					) {
					return true
				}
			default:
			}
		}
	}

	return false
}

func buildInputsFromTranscriptMessage(
	message sessionRuntime.TranscriptMessage,
) ([]inferenceSpec.InputUnion, error) {
	userInput := map[string]any{
		wireKindField: wireInputMessageKind,
		wireInputMessageField: map[string]any{
			wireIDField:     message.ID,
			wireRoleField:   string(message.Role),
			wireStatusField: message.Status,
			wireContentsField: []map[string]any{
				{
					wireKindField: wireTextKind,
					wireTextItemField: map[string]any{
						wireTextField: message.Text,
					},
				},
			},
		},
	}
	if strings.TrimSpace(message.Text) == "" {
		inputMessage, ok := userInput[wireInputMessageField].(map[string]any)
		if !ok {
			return nil, errors.New("session inference adapter could not build an input message")
		}
		inputMessage[wireContentsField] = []map[string]any{}
	}

	raw, err := json.Marshal(userInput)
	if err != nil {
		return nil, err
	}

	var base inferenceSpec.InputUnion
	if err := json.Unmarshal(raw, &base); err != nil {
		return nil, err
	}
	output := []inferenceSpec.InputUnion{base}

	for _, result := range message.ToolResults {
		wire := map[string]any{
			wireTypeField:     result.Type,
			wireChoiceIDField: result.ChoiceID,
			wireIDField:       string(result.ID),
			wireCallIDField:   string(result.CallID),
			wireRoleField:     wireToolRole,
			wireStatusField:   wireCompletedStatus,
			wireNameField:     result.Name,
			wireIsErrorField:  result.Kind != sessionRuntime.ToolResultKindSuccess,
			wireContentsField: []map[string]any{
				{
					wireKindField: wireTextKind,
					wireTextItemField: map[string]any{
						wireTextField: toolResultText(result),
					},
				},
			},
		}
		kind := wireFunctionToolOutputKind
		field := wireFunctionToolOutputField
		if result.Type == "custom" {
			kind = wireCustomToolOutputKind
			field = wireCustomToolOutputField
		}
		if result.Type == "webSearch" {
			kind = wireWebSearchToolOutputKind
			field = wireWebSearchToolOutputField
		}

		unionRaw, marshalErr := json.Marshal(map[string]any{
			wireKindField: kind,
			field:         wire,
		})
		if marshalErr != nil {
			return nil, marshalErr
		}

		var union inferenceSpec.InputUnion
		if err := json.Unmarshal(unionRaw, &union); err != nil {
			return nil, err
		}
		output = append(output, union)
	}

	return output, nil
}

func buildOutputsFromTranscriptMessage(
	message sessionRuntime.TranscriptMessage,
) ([]inferenceSpec.OutputUnion, error) {
	raw, err := json.Marshal(map[string]any{
		wireKindField: wireOutputMessageKind,
		wireOutputMessageField: map[string]any{
			wireIDField:     message.ID,
			wireRoleField:   string(message.Role),
			wireStatusField: message.Status,
			wireContentsField: []map[string]any{
				{
					wireKindField: wireTextKind,
					wireTextItemField: map[string]any{
						wireTextField: message.Text,
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var output inferenceSpec.OutputUnion
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, err
	}
	return []inferenceSpec.OutputUnion{output}, nil
}

func toolResultText(result sessionRuntime.ToolResult) string {
	if strings.TrimSpace(result.Text) != "" {
		return result.Text
	}
	if len(result.Structured) != 0 {
		return string(result.Structured)
	}
	if strings.TrimSpace(result.Error) != "" {
		return result.Error
	}
	return "Tool completed without textual output."
}

func convertToolSelections(
	selections []sessionRuntime.ModelToolSelection,
) ([]conversationSpec.ToolSelection, error) {
	output := make(
		[]conversationSpec.ToolSelection,
		0,
		len(selections),
	)

	for _, selection := range selections {
		rawTarget, err := json.Marshal(selection.Target)
		if err != nil {
			return nil, err
		}

		var target composition.CapabilityTarget
		if err := json.Unmarshal(rawTarget, &target); err != nil {
			return nil, fmt.Errorf(
				"decode session tool target %q: %w",
				selection.ChoiceID,
				err,
			)
		}

		output = append(output, conversationSpec.ToolSelection{
			ChoiceID:              selection.ChoiceID,
			Target:                target,
			AutoExecute:           selection.AutoExecute,
			UserArgSchemaInstance: jsonutil.JSONRawString(selection.UserArguments),
		})
	}

	return output, nil
}

func convertRequestPatch(
	raw json.RawMessage,
) (*inferenceRuntimeSpec.RuntimeRequestPatch, error) {
	if len(raw) == 0 {
		return nil, nil //nolint:nilnil // nil preserves the absence of a Session request patch.
	}

	var patch inferenceRuntimeSpec.RuntimeRequestPatch
	if err := json.Unmarshal(raw, &patch); err != nil {
		return nil, err
	}

	return &patch, nil
}

func artifactReference(
	reference sessionRuntime.ArtifactReference,
) (artifactModel.ArtifactRef, error) {
	if reference.RootID == "" || reference.ArtifactID == "" {
		return artifactModel.ArtifactRef{}, errors.New(
			"session artifact reference is incomplete",
		)
	}

	raw, err := json.Marshal(reference)
	if err != nil {
		return artifactModel.ArtifactRef{}, err
	}

	var output artifactModel.ArtifactRef
	if err := json.Unmarshal(raw, &output); err != nil {
		return artifactModel.ArtifactRef{}, err
	}

	return output, nil
}

func artifactReferences(
	groups ...[]sessionRuntime.ArtifactReference,
) ([]artifactModel.ArtifactRef, error) {
	seen := make(map[string]struct{})
	output := make([]artifactModel.ArtifactRef, 0)

	for _, group := range groups {
		for _, reference := range group {
			value, err := artifactReference(reference)
			if err != nil {
				return nil, err
			}
			key := reference.RootID + "\x00" + reference.ArtifactID
			if _, found := seen[key]; found {
				continue
			}
			seen[key] = struct{}{}
			output = append(output, value)
		}
	}

	return output, nil
}

func decodeInputUnions(
	rawValues []json.RawMessage,
) ([]inferenceSpec.InputUnion, error) {
	if len(rawValues) == 0 {
		return nil, nil
	}

	output := make([]inferenceSpec.InputUnion, 0, len(rawValues))
	for _, raw := range rawValues {
		var value inferenceSpec.InputUnion
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode protocol input: %w", err)
		}
		output = append(output, value)
	}

	return output, nil
}

func decodeOutputUnions(
	rawValues []json.RawMessage,
) ([]inferenceSpec.OutputUnion, error) {
	if len(rawValues) == 0 {
		return nil, nil
	}

	output := make([]inferenceSpec.OutputUnion, 0, len(rawValues))
	for _, raw := range rawValues {
		var value inferenceSpec.OutputUnion
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("decode protocol output: %w", err)
		}
		output = append(output, value)
	}

	return output, nil
}

func marshalInputUnions(
	values []inferenceSpec.InputUnion,
) []json.RawMessage {
	if len(values) == 0 {
		return nil
	}

	output := make([]json.RawMessage, 0, len(values))
	for _, value := range values {
		raw, err := json.Marshal(value)
		if err == nil {
			output = append(output, raw)
		}
	}
	return output
}

func marshalOutputUnions(
	values []inferenceSpec.OutputUnion,
) []json.RawMessage {
	if len(values) == 0 {
		return nil
	}

	output := make([]json.RawMessage, 0, len(values))
	for _, value := range values {
		raw, err := json.Marshal(value)
		if err == nil {
			output = append(output, raw)
		}
	}
	return output
}

func marshalMetadata(
	mappings []conversationSpec.MCPProviderToolMapping,
	workspaceUsage *conversationSpec.WorkspaceConversationUsage,
) json.RawMessage {
	if len(mappings) == 0 && workspaceUsage == nil {
		return nil
	}

	raw, err := json.Marshal(map[string]any{
		"mcpToolMappings": mappings,
		"workspaceUsage":  workspaceUsage,
	})
	if err != nil {
		return nil
	}
	return raw
}

func inferenceError(
	value *inferenceSpec.Error,
) *sessionRuntime.OperationError {
	if value == nil {
		return nil
	}

	return &sessionRuntime.OperationError{
		Code:    value.Code,
		Message: value.Message,
	}
}

func decodeOptional[T any](
	raw json.RawMessage,
) (*T, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil //nolint:nilnil // nil represents an omitted optional Session field.
	}

	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

type outputWire struct {
	Kind              string         `json:"kind"`
	OutputMessage     *messageWire   `json:"outputMessage"`
	ReasoningMessage  *reasoningWire `json:"reasoningMessage"`
	FunctionToolCall  *toolCallWire  `json:"functionToolCall"`
	CustomToolCall    *toolCallWire  `json:"customToolCall"`
	WebSearchToolCall *toolCallWire  `json:"webSearchToolCall"`
}

type messageWire struct {
	Contents []contentWire `json:"contents"`
}

type contentWire struct {
	Kind     string `json:"kind"`
	TextItem *struct {
		Text string `json:"text"`
	} `json:"textItem"`
}

type reasoningWire struct {
	Thinking []string `json:"thinking"`
}

type toolCallWire struct {
	ID        string          `json:"id"`
	CallID    string          `json:"callID"`
	Name      string          `json:"name"`
	ChoiceID  string          `json:"choiceID"`
	Type      string          `json:"type"`
	Arguments json.RawMessage `json:"arguments"`
}

func decodeOutputs(
	outputs []json.RawMessage,
	selections []sessionRuntime.ModelToolSelection,
	mappings []conversationSpec.MCPProviderToolMapping,
) (
	text string,
	thinking string,
	calls []sessionRuntime.ToolCallProposal,
) {
	selectionByChoice := make(map[string]sessionRuntime.ModelToolSelection)
	selectionByName := make(map[string]sessionRuntime.ModelToolSelection)
	for _, selection := range selections {
		selectionByChoice[selection.ChoiceID] = selection
		selectionByName[selection.Target.Name] = selection
	}

	mappingByChoice := make(map[string]conversationSpec.MCPProviderToolMapping)
	mappingByName := make(map[string]conversationSpec.MCPProviderToolMapping)
	for _, mapping := range mappings {
		mappingByChoice[mapping.ChoiceID] = mapping
		mappingByName[mapping.ProviderToolName] = mapping
	}

	var textBuilder strings.Builder
	var thinkingBuilder strings.Builder
	calls = make([]sessionRuntime.ToolCallProposal, 0)

	for _, raw := range outputs {
		var output outputWire
		if err := json.Unmarshal(raw, &output); err != nil {
			continue
		}

		if output.OutputMessage != nil {
			for _, content := range output.OutputMessage.Contents {
				if content.Kind == wireTextKind && content.TextItem != nil {
					textBuilder.WriteString(content.TextItem.Text)
				}
			}
		}
		if output.ReasoningMessage != nil {
			for _, value := range output.ReasoningMessage.Thinking {
				thinkingBuilder.WriteString(value)
			}
		}

		call := output.FunctionToolCall
		if call == nil {
			call = output.CustomToolCall
		}
		if call == nil {
			call = output.WebSearchToolCall
		}
		if call == nil {
			continue
		}

		calls = append(calls, buildToolCallProposal(
			*call,
			selectionByChoice,
			selectionByName,
			mappingByChoice,
			mappingByName,
		))
	}

	return textBuilder.String(), thinkingBuilder.String(), calls
}

func buildToolCallProposal(
	call toolCallWire,
	selectionByChoice map[string]sessionRuntime.ModelToolSelection,
	selectionByName map[string]sessionRuntime.ModelToolSelection,
	mappingByChoice map[string]conversationSpec.MCPProviderToolMapping,
	mappingByName map[string]conversationSpec.MCPProviderToolMapping,
) sessionRuntime.ToolCallProposal {
	arguments := normalizedArguments(call.Arguments)
	proposal := sessionRuntime.ToolCallProposal{
		ID:             sessionRuntime.ToolCallID(firstNonEmpty(call.ID, call.CallID)),
		ProviderCallID: firstNonEmpty(call.CallID, call.ID),
		Name:           call.Name,
		ChoiceID:       call.ChoiceID,
		Type:           call.Type,
		Arguments:      arguments,
		Required:       true,
		Binding: sessionRuntime.ToolBinding{
			Kind: sessionRuntime.ToolBindingKindUnresolved,
		},
	}

	mapping, mapped := mappingByChoice[call.ChoiceID]
	if !mapped {
		mapping, mapped = mappingByName[call.Name]
	}
	if mapped {
		rawMapping, _ := json.Marshal(mapping)
		proposal.Binding = sessionRuntime.ToolBinding{
			Kind: sessionRuntime.ToolBindingKindMCP,
			Data: rawMapping,
		}
		proposal.Automatic = string(mapping.ExecutionMode) == "auto"
		return proposal
	}

	if strings.HasPrefix(call.Name, "skills-") {
		proposal.Binding = sessionRuntime.ToolBinding{
			Kind: sessionRuntime.ToolBindingKindSkill,
		}
		proposal.Automatic = isAutomaticSkillCall(call.ChoiceID)
		return proposal
	}

	selection, found := selectionByChoice[call.ChoiceID]
	if !found {
		selection, found = selectionByName[call.Name]
	}
	if found {
		if selection.ProviderManaged ||
			selection.ImplementationKind != "" &&
				selection.ImplementationKind != "go" {
			proposal.ProviderManaged = true
			proposal.Required = false
			proposal.Binding.Kind = sessionRuntime.ToolBindingKindProvider
			return proposal
		}

		rawBinding, _ := json.Marshal(map[string]any{
			"target": selection.Target,
		})
		proposal.Binding = sessionRuntime.ToolBinding{
			Kind: sessionRuntime.ToolBindingKindGo,
			Data: rawBinding,
		}
		proposal.Automatic = selection.AutoExecute
		return proposal
	}

	if call.Type == "webSearch" {
		proposal.ProviderManaged = true
		proposal.Required = false
		proposal.Binding.Kind = sessionRuntime.ToolBindingKindProvider
		return proposal
	}

	proposal.Error = "the tool call cannot be matched to the originating model operation bindings"
	return proposal
}

func normalizedArguments(
	raw json.RawMessage,
) json.RawMessage {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return json.RawMessage(value)
	}
	return append(json.RawMessage(nil), raw...)
}

func hasUsableNonTextOutput(
	outputs []json.RawMessage,
) bool {
	for _, raw := range outputs {
		var output outputWire
		if err := json.Unmarshal(raw, &output); err != nil {
			continue
		}
		if output.FunctionToolCall != nil ||
			output.CustomToolCall != nil ||
			output.WebSearchToolCall != nil {
			return true
		}
	}
	return false
}

func isAutomaticSkillCall(choiceID string) bool {
	switch choiceID {
	case "builtin.skills-load",
		"builtin.skills-unload",
		"builtin.skills-readresource":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
