package sessionstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	textStr  = "text"
	fileStr  = "file"
	imageStr = "image"
)

// Projection is checked and deep-copying. It never returns its input on clone
// failure, and never persists a live executor authorization or hydrated input.
func storedSessionFromCheckpoint(
	checkpoint sessionRuntime.RuntimeCheckpointView,
) (storedSession, error) {
	sessionValue, err := projectSession(checkpoint.Session)
	if err != nil {
		return storedSession{}, err
	}

	document := storedSession{
		SchemaVersion: SessionSchemaVersion,
		Kind:          documentKindSession,
		Session:       &sessionValue,
	}
	if checkpoint.Run != nil {
		run, err := projectRun(*checkpoint.Run)
		if err != nil {
			return storedSession{}, err
		}
		document.Runs = []sessionRuntime.Run{run}
	}
	return document, nil
}

func projectSession(
	input sessionRuntime.Session,
) (sessionRuntime.Session, error) {
	output, err := jsonutil.CloneJSON(input)
	if err != nil {
		return sessionRuntime.Session{}, fmt.Errorf("clone semantic Session: %w", err)
	}

	output.Transcript, err = projectTranscript(output.Transcript)
	if err != nil {
		return sessionRuntime.Session{}, err
	}
	if err := normalizeConfiguration(&output.Configuration); err != nil {
		return sessionRuntime.Session{}, err
	}
	return output, nil
}

func projectRun(
	input sessionRuntime.Run,
) (sessionRuntime.Run, error) {
	output, err := jsonutil.CloneJSON(input)
	if err != nil {
		return sessionRuntime.Run{}, fmt.Errorf("clone semantic Run: %w", err)
	}

	output.SkillSession.ID = ""
	output.SkillSession.EstablishedGeneration = 0
	output.Continuation.Metadata = nil
	output.Apps = projectApps(output.Apps)

	if err := normalizeConfiguration(&output.Configuration); err != nil {
		return sessionRuntime.Run{}, err
	}
	if output.Continuation.Configuration != nil {
		if err := normalizeConfiguration(output.Continuation.Configuration); err != nil {
			return sessionRuntime.Run{}, err
		}
	}

	for index := range output.ModelOperations {
		operation := &output.ModelOperations[index]
		operation.Metadata = nil
		operation.RuntimeMetadata = nil
		operation.Preparation.Private = nil

		if err := normalizeConfiguration(&operation.Request.Configuration); err != nil {
			return sessionRuntime.Run{}, err
		}
		operation.Request.History, err = projectTranscript(operation.Request.History)
		if err != nil {
			return sessionRuntime.Run{}, err
		}
		operation.Request.Current, err = projectTranscriptMessage(operation.Request.Current)
		if err != nil {
			return sessionRuntime.Run{}, err
		}
		operation.ProtocolOutputs, err = semanticProtocolOutputs(operation.ProtocolOutputs)
		if err != nil {
			return sessionRuntime.Run{}, err
		}
	}

	for index := range output.ToolCalls {
		call := &output.ToolCalls[index]
		call.Authorization = nil

		call.Arguments, err = canonicalJSONValue(call.Arguments)
		if err != nil {
			return sessionRuntime.Run{}, fmt.Errorf("tool arguments: %w", err)
		}
		call.Binding.Data, err = semanticBindingData(call.Binding)
		if err != nil {
			return sessionRuntime.Run{}, err
		}
		call.Results, err = projectToolResults(call.Results)
		if err != nil {
			return sessionRuntime.Run{}, err
		}
	}

	for index := range output.WaitingRequests {
		wait := &output.WaitingRequests[index]
		wait.Summary, err = canonicalJSONValue(wait.Summary)
		if err != nil {
			return sessionRuntime.Run{}, fmt.Errorf("wait summary: %w", err)
		}
		wait.InputSchema, err = canonicalJSONValue(wait.InputSchema)
		if err != nil {
			return sessionRuntime.Run{}, fmt.Errorf("wait input schema: %w", err)
		}
	}

	return output, nil
}

func projectTranscript(
	input []sessionRuntime.TranscriptMessage,
) ([]sessionRuntime.TranscriptMessage, error) {
	output := make([]sessionRuntime.TranscriptMessage, len(input))
	for index := range input {
		message, err := projectTranscriptMessage(input[index])
		if err != nil {
			return nil, fmt.Errorf("transcript message %d: %w", index, err)
		}
		output[index] = message
	}
	return output, nil
}

func projectTranscriptMessage(
	input sessionRuntime.TranscriptMessage,
) (sessionRuntime.TranscriptMessage, error) {
	output, err := jsonutil.CloneJSON(input)
	if err != nil {
		return sessionRuntime.TranscriptMessage{}, err
	}

	output.ProtocolInputs = nil
	output.AttachmentsHydrated = false
	output.Metadata = nil
	output.RuntimeMetadata = nil

	output.ProtocolOutputs, err = semanticProtocolOutputs(output.ProtocolOutputs)
	if err != nil {
		return sessionRuntime.TranscriptMessage{}, err
	}
	output.ToolResults, err = projectToolResults(output.ToolResults)
	if err != nil {
		return sessionRuntime.TranscriptMessage{}, err
	}
	return output, nil
}

func projectToolResults(
	input []sessionRuntime.ToolResult,
) ([]sessionRuntime.ToolResult, error) {
	output := make([]sessionRuntime.ToolResult, len(input))
	for index := range input {
		value := input[index]

		if len(value.Contents) != 0 {
			if text := semanticToolResultContentsText(value.Contents); text != "" {
				value.Text = text
			}
		} else if len(value.Structured) != 0 {
			// Legacy compatibility path for results created before typed
			// inference-go ToolOutputItemUnion content was persisted.
			text, err := semanticToolOutputText(value.Structured)
			if err != nil {
				return nil, fmt.Errorf("tool result %q: %w", value.ID, err)
			}
			value.Text = text
		}
		value.Structured = nil
		value.App = nil
		output[index] = value
	}
	return output, nil
}

func semanticToolResultContentsText(
	contents []sessionRuntime.InferenceToolOutputItem,
) string {
	parts := make([]string, 0, len(contents))
	for _, content := range contents {
		switch content.Kind {
		case inferenceSpec.ContentItemKindText:
			if content.TextItem != nil &&
				strings.TrimSpace(content.TextItem.Text) != "" {
				parts = append(parts, content.TextItem.Text)
			}

		case inferenceSpec.ContentItemKindImage:
			name := imageStr
			if content.ImageItem != nil &&
				strings.TrimSpace(content.ImageItem.ImageName) != "" {
				name = content.ImageItem.ImageName
			}
			parts = append(parts, "[Tool image omitted: "+name+"]")

		case inferenceSpec.ContentItemKindFile:
			name := fileStr
			if content.FileItem != nil &&
				strings.TrimSpace(content.FileItem.FileName) != "" {
				name = content.FileItem.FileName
			}
			parts = append(parts, "[Tool file omitted: "+name+"]")
		default:
		}
	}

	return strings.Join(parts, "\n\n")
}

func normalizeConfiguration(
	configuration *sessionRuntime.SessionConfiguration,
) error {
	metadata, err := canonicalJSONValue(configuration.Metadata)
	if err != nil {
		return fmt.Errorf("session presentation metadata: %w", err)
	}
	configuration.Metadata = metadata

	for index := range configuration.Tools {
		override := &configuration.Tools[index]
		if override.Value == nil {
			continue
		}
		arguments, err := canonicalJSONValue(
			override.Value.UserArguments,
		)
		if err != nil {
			return fmt.Errorf("tool override arguments: %w", err)
		}
		override.Value.UserArguments = arguments

	}
	for key, raw := range configuration.ModelOverrides.AdapterParameters {
		canonical, err := canonicalJSONValue(raw)
		if err != nil {
			return fmt.Errorf("model adapter parameter %q: %w", key, err)
		}
		configuration.ModelOverrides.AdapterParameters[key] = canonical
	}
	return nil
}

func canonicalJSONValue(input json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(input)) == 0 {
		return nil, nil
	}
	if len(input) > maximumJSONValueBytes {
		return nil, fmt.Errorf("%w: JSON value exceeds size limit", ErrInvalidDocument)
	}
	canonical, err := jsonutil.Canonicalize(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	return json.RawMessage(canonical), nil
}

type semanticGoBinding struct {
	Target    sessionRuntime.CapabilityTarget `json:"target"`
	Used      sessionRuntime.ResolvedArtifact `json:"used"`
	TimeoutMS int                             `json:"timeoutMS,omitempty"`
}

type semanticMCPBinding struct {
	Server           string   `json:"server"`
	ProviderToolName string   `json:"providerToolName"`
	ChoiceID         string   `json:"choiceID"`
	ToolName         string   `json:"toolName"`
	ToolDigest       string   `json:"toolDigest"`
	ApprovalRule     string   `json:"approvalRule"`
	ExecutionMode    string   `json:"executionMode"`
	AppResourceURI   string   `json:"appResourceUri,omitempty"`
	Visibility       []string `json:"visibility,omitempty"`
}

func semanticBindingData(
	binding sessionRuntime.ToolBinding,
) (json.RawMessage, error) {
	switch binding.Kind {
	case sessionRuntime.ToolBindingKindGo:
		value, err := jsonutil.DecodeJSONRaw[semanticGoBinding](binding.Data)
		if err != nil {
			return nil, fmt.Errorf("go tool binding: %w", err)
		}
		if value.TimeoutMS < 0 {
			return nil, fmt.Errorf("%w: negative Tool timeout", ErrInvalidDocument)
		}
		if err := validateCapabilityTarget(value.Target); err != nil {
			return nil, err
		}
		return json.Marshal(value)

	case sessionRuntime.ToolBindingKindMCP:
		value, err := jsonutil.DecodeJSONRaw[semanticMCPBinding](binding.Data)
		if err != nil {
			return nil, fmt.Errorf("MCP Tool binding: %w", err)
		}
		for _, identity := range []string{
			value.Server,
			value.ProviderToolName,
			value.ChoiceID,
			value.ToolName,
			value.ToolDigest,
		} {
			if err := validateIdentity("MCP binding identity", identity); err != nil {
				return nil, err
			}
		}
		value.Visibility = slices.Clone(value.Visibility)
		return json.Marshal(value)

	case sessionRuntime.ToolBindingKindSkill,
		sessionRuntime.ToolBindingKindProvider,
		sessionRuntime.ToolBindingKindUnresolved:
		return nil, nil

	default:
		return nil, fmt.Errorf(
			"%w: unsupported Tool binding kind %q",
			ErrInvalidDocument,
			binding.Kind,
		)
	}
}

// The following types are a closed semantic projection of provider outputs.
// Their marshaled values preserve replayable messages and Tool call identities;
// arbitrary provider transport, debugging, binary, or private fields are not
// copied into persistence.
type semanticOutput struct {
	Kind                string              `json:"kind"`
	OutputMessage       *semanticMessage    `json:"outputMessage,omitempty"`
	ReasoningMessage    *semanticReasoning  `json:"reasoningMessage,omitempty"`
	FunctionToolCall    *semanticCall       `json:"functionToolCall,omitempty"`
	CustomToolCall      *semanticCall       `json:"customToolCall,omitempty"`
	WebSearchToolCall   *semanticCall       `json:"webSearchToolCall,omitempty"`
	WebSearchToolOutput *semanticToolOutput `json:"webSearchToolOutput,omitempty"`
}

type semanticMessage struct {
	ID       string            `json:"id,omitempty"`
	Role     string            `json:"role"`
	Status   string            `json:"status,omitempty"`
	Contents []semanticContent `json:"contents"`
}

type semanticContent struct {
	Kind      string                  `json:"kind"`
	TextItem  *semanticText           `json:"textItem,omitempty"`
	ImageItem *semanticImageReference `json:"imageItem,omitempty"`
	FileItem  *semanticFileReference  `json:"fileItem,omitempty"`
}

type semanticText struct {
	Text string `json:"text"`
}

type semanticImageReference struct {
	ImageName string `json:"imageName,omitempty"`
	ImageMIME string `json:"imageMIME,omitempty"`
}

type semanticFileReference struct {
	FileName string `json:"fileName,omitempty"`
	FileMIME string `json:"fileMIME,omitempty"`
}

type semanticReasoning struct {
	ID       string   `json:"id,omitempty"`
	Thinking []string `json:"thinking,omitempty"`
}

type semanticCall struct {
	ID        string          `json:"id,omitempty"`
	CallID    string          `json:"callID,omitempty"`
	ChoiceID  string          `json:"choiceID,omitempty"`
	Name      string          `json:"name"`
	Type      string          `json:"type,omitempty"`
	Role      string          `json:"role,omitempty"`
	Status    string          `json:"status,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type semanticToolOutput struct {
	ID       string            `json:"id,omitempty"`
	CallID   string            `json:"callID,omitempty"`
	ChoiceID string            `json:"choiceID,omitempty"`
	Name     string            `json:"name,omitempty"`
	Type     string            `json:"type,omitempty"`
	Role     string            `json:"role,omitempty"`
	Status   string            `json:"status,omitempty"`
	IsError  bool              `json:"isError,omitempty"`
	Contents []semanticContent `json:"contents,omitempty"`
}

func semanticProtocolOutputs(
	input []json.RawMessage,
) ([]json.RawMessage, error) {
	output := make([]json.RawMessage, 0, len(input))
	for index, raw := range input {
		canonical, err := canonicalJSONValue(raw)
		if err != nil {
			return nil, fmt.Errorf("protocol output %d: %w", index, err)
		}

		var value semanticOutput
		if err := json.Unmarshal(canonical, &value); err != nil {
			return nil, fmt.Errorf("adapt protocol output %d: %w", index, err)
		}

		switch value.Kind {
		case "outputMessage":
			if value.OutputMessage == nil {
				return nil, fmt.Errorf("%w: missing output message", ErrInvalidDocument)
			}
			value.OutputMessage.Contents = semanticContents(value.OutputMessage.Contents)

		case "reasoningMessage":
			if value.ReasoningMessage == nil {
				return nil, fmt.Errorf("%w: missing reasoning message", ErrInvalidDocument)
			}

		case "functionToolCall":
			if err := normalizeSemanticCall(value.FunctionToolCall); err != nil {
				return nil, err
			}

		case "customToolCall":
			if err := normalizeSemanticCall(value.CustomToolCall); err != nil {
				return nil, err
			}

		case "webSearchToolCall":
			if err := normalizeSemanticCall(value.WebSearchToolCall); err != nil {
				return nil, err
			}

		case "webSearchToolOutput":
			if value.WebSearchToolOutput == nil {
				return nil, fmt.Errorf("%w: missing provider Tool output", ErrInvalidDocument)
			}
			value.WebSearchToolOutput.Contents = semanticContents(
				value.WebSearchToolOutput.Contents,
			)

		default:
			// Unknown provider output cannot be faithfully replayed. Its
			// omission is explicit in durable visible history.
			value = semanticOutput{
				Kind: "outputMessage",
				OutputMessage: &semanticMessage{
					Role:   "assistant",
					Status: "completed",
					Contents: []semanticContent{{
						Kind: textStr,
						TextItem: &semanticText{
							Text: fmt.Sprintf(
								"[Provider output kind %q is not available in durable history.]",
								value.Kind,
							),
						},
					}},
				},
			}
		}

		projected, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		output = append(output, projected)
	}
	return output, nil
}

func normalizeSemanticCall(call *semanticCall) error {
	if call == nil {
		return fmt.Errorf("%w: missing provider Tool call", ErrInvalidDocument)
	}
	if err := validateIdentity("provider Tool name", call.Name); err != nil {
		return err
	}
	arguments, err := canonicalJSONValue(call.Arguments)
	if err != nil {
		return err
	}
	call.Arguments = arguments
	return nil
}

func semanticContents(input []semanticContent) []semanticContent {
	output := make([]semanticContent, 0, len(input))
	for _, value := range input {
		switch value.Kind {
		case textStr:
			if value.TextItem != nil {
				output = append(output, semanticContent{
					Kind:     textStr,
					TextItem: &semanticText{Text: value.TextItem.Text},
				})
			}
		case imageStr:
			name := ""
			if value.ImageItem != nil {
				name = value.ImageItem.ImageName
			}
			output = append(output, semanticContent{
				Kind: textStr,
				TextItem: &semanticText{
					Text: fmt.Sprintf("[Image content omitted from durable history: %s]", name),
				},
			})
		case fileStr:
			name := ""
			if value.FileItem != nil {
				name = value.FileItem.FileName
			}
			output = append(output, semanticContent{
				Kind: textStr,
				TextItem: &semanticText{
					Text: fmt.Sprintf("[File content omitted from durable history: %s]", name),
				},
			})
		default:
			output = append(output, semanticContent{
				Kind: textStr,
				TextItem: &semanticText{
					Text: fmt.Sprintf("[Content kind %q omitted from durable history.]", value.Kind),
				},
			})
		}
	}
	return output
}

func projectApps(
	values []sessionRuntime.AppInstance,
) []sessionRuntime.AppInstance {
	output := make([]sessionRuntime.AppInstance, len(values))
	for index := range values {
		value := values[index]
		value.Route = nil
		value.Views = append([]sessionRuntime.AppViewID(nil), value.Views...)
		if value.UnavailableReason == "" {
			value.UnavailableReason = "App route is not durable"
		}
		output[index] = value
	}
	return output
}

type toolOutputWire struct {
	Kind      string                  `json:"kind"`
	TextItem  *semanticText           `json:"textItem,omitempty"`
	ImageItem *semanticImageReference `json:"imageItem,omitempty"`
	FileItem  *semanticFileReference  `json:"fileItem,omitempty"`
}

type mcpResultWire struct {
	Content []mcpContentWire `json:"content,omitempty"`
}

type mcpContentWire struct {
	Type     string           `json:"type"`
	Text     string           `json:"text,omitempty"`
	URI      string           `json:"uri,omitempty"`
	Name     string           `json:"name,omitempty"`
	MIMEType string           `json:"mimeType,omitempty"`
	Resource *mcpResourceWire `json:"resource,omitempty"`
}

type mcpResourceWire struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

func semanticToolOutputText(raw json.RawMessage) (string, error) {
	canonical, err := canonicalJSONValue(raw)
	if err != nil {
		return "", err
	}
	if len(canonical) == 0 {
		return "", nil
	}

	var parts []string
	switch canonical[0] {
	case '[':
		var values []toolOutputWire
		if err := json.Unmarshal(canonical, &values); err != nil {
			return "", err
		}
		for _, value := range values {
			switch value.Kind {
			case textStr:
				if value.TextItem != nil {
					parts = append(parts, value.TextItem.Text)
				}
			case imageStr:
				name := ""
				if value.ImageItem != nil {
					name = value.ImageItem.ImageName
				}
				parts = append(parts, "[Tool image omitted: "+name+"]")
			case fileStr:
				name := ""
				if value.FileItem != nil {
					name = value.FileItem.FileName
				}
				parts = append(parts, "[Tool file omitted: "+name+"]")
			default:
				parts = append(parts, "[Structured Tool output omitted from durable history.]")
			}
		}
	case '{':
		var value mcpResultWire
		if err := json.Unmarshal(canonical, &value); err != nil {
			return "", err
		}
		for _, content := range value.Content {
			switch content.Type {
			case textStr:
				parts = append(parts, content.Text)
			case "resource":
				if content.Resource != nil {
					parts = append(parts, content.Resource.Text)
					if content.Resource.Text == "" {
						parts = append(parts, "[MCP resource omitted: "+content.Resource.URI+"]")
					}
				}
			case "resource_link":
				parts = append(parts, strings.TrimSpace(content.Name+" "+content.URI))
			default:
				parts = append(parts, "[MCP binary content omitted from durable history.]")
			}
		}
	}

	if len(parts) == 0 {
		return "[Structured Tool content is not available in durable history.]", nil
	}
	return strings.Join(parts, "\n\n"), nil
}
