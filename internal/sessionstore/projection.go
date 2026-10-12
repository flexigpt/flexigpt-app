package sessionstore

import (
	"encoding/json"
	"slices"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// storedSessionFromCheckpoint is the store-owned projection boundary.
//
// The runtime checkpoint remains live runtime state. This conversion keeps the
// durable Session transcript and execution record semantic while explicitly
// omitting hydrated inference payloads, generated attachment content, provider
// protocol output payloads, App render payloads, and debug metadata.
func storedSessionFromCheckpoint(
	checkpoint sessionRuntime.RuntimeCheckpointView,
) storedSession {
	document := storedSession{
		SchemaVersion: SessionSchemaVersion,
		Session:       projectSession(checkpoint.Session),
	}

	if checkpoint.Run != nil {
		document.Runs = []sessionRuntime.Run{
			projectRun(*checkpoint.Run),
		}
	}

	return document
}

func cloneStoredSession(value storedSession) storedSession {
	return jsonutil.CloneJSONInValue(value)
}

func projectSession(
	value sessionRuntime.Session,
) sessionRuntime.Session {
	output := value
	output.Configuration = projectSessionConfiguration(value.Configuration)
	output.Transcript = projectTranscript(value.Transcript)
	output.RunIDs = slices.Clone(value.RunIDs)
	return output
}

func projectSessionConfiguration(
	value sessionRuntime.SessionConfiguration,
) sessionRuntime.SessionConfiguration {
	output := value

	output.RequestPatch = cloneRawJSON(value.RequestPatch)
	output.MCPContext = cloneRawJSON(value.MCPContext)
	output.WorkspaceSelection = cloneRawJSON(value.WorkspaceSelection)
	output.Metadata = cloneRawJSON(value.Metadata)

	output.ToolSelections = make(
		[]sessionRuntime.ModelToolSelection,
		len(value.ToolSelections),
	)
	for index := range value.ToolSelections {
		selection := value.ToolSelections[index]
		selection.UserArguments = cloneRawJSON(
			value.ToolSelections[index].UserArguments,
		)

		if selection.Target.Artifact != nil {
			artifact := *selection.Target.Artifact
			selection.Target.Artifact = &artifact
		}

		output.ToolSelections[index] = selection
	}

	output.InstalledSkills = slices.Clone(value.InstalledSkills)
	output.WorkspaceSkills = slices.Clone(value.WorkspaceSkills)

	if value.Agent != nil {
		agent := *value.Agent
		output.Agent = &agent
	}

	return output
}

func projectTranscript(
	values []sessionRuntime.TranscriptMessage,
) []sessionRuntime.TranscriptMessage {
	if len(values) == 0 {
		return nil
	}

	output := make([]sessionRuntime.TranscriptMessage, len(values))
	for index := range values {
		output[index] = projectTranscriptMessage(values[index])
	}

	return output
}

func projectTranscriptMessage(
	value sessionRuntime.TranscriptMessage,
) sessionRuntime.TranscriptMessage {
	output := value

	output.Attachments = slices.Clone(value.Attachments)
	output.ProtocolInputs = nil
	output.ProtocolOutputs = nil
	output.ToolResults = projectToolResults(value.ToolResults)
	output.MCPAppContext = nil
	output.Metadata = nil

	if value.Error != nil {
		errorValue := *value.Error
		output.Error = &errorValue
	}

	return output
}

func projectToolResults(
	values []sessionRuntime.ToolResult,
) []sessionRuntime.ToolResult {
	if len(values) == 0 {
		return nil
	}

	output := make([]sessionRuntime.ToolResult, len(values))
	for index := range values {
		result := values[index]
		result.Structured = nil
		result.App = nil
		output[index] = result
	}

	return output
}

func projectRun(
	value sessionRuntime.Run,
) sessionRuntime.Run {
	output := value

	output.Driver.Data = cloneRawJSON(value.Driver.Data)
	output.Configuration = projectSessionConfiguration(value.Configuration)
	output.SkillSession.Enabled = slices.Clone(value.SkillSession.Enabled)
	output.SkillSession.Active = slices.Clone(value.SkillSession.Active)
	output.Continuation = projectContinuation(value.Continuation)
	output.ModelOperations = projectModelOperations(value.ModelOperations)
	output.ToolCalls = projectToolCalls(value.ToolCalls)
	output.WaitingRequests = projectWaitingRequests(value.WaitingRequests)
	output.RequestedCallIDs = slices.Clone(value.RequestedCallIDs)

	if value.Outcome != nil {
		outcome := *value.Outcome
		output.Outcome = &outcome
	}

	return output
}

func projectContinuation(
	value sessionRuntime.ContinuationInput,
) sessionRuntime.ContinuationInput {
	output := value

	output.Attachments = slices.Clone(value.Attachments)
	output.MCPAppContext = nil
	output.Metadata = nil

	if value.Configuration != nil {
		configuration := projectSessionConfiguration(*value.Configuration)
		output.Configuration = &configuration
	}

	return output
}

func projectModelOperations(
	values []sessionRuntime.ModelOperation,
) []sessionRuntime.ModelOperation {
	if len(values) == 0 {
		return nil
	}

	output := make([]sessionRuntime.ModelOperation, len(values))
	for index := range values {
		operation := values[index]
		operation.Request.Configuration = projectSessionConfiguration(
			values[index].Request.Configuration,
		)
		operation.Request.History = projectTranscript(
			values[index].Request.History,
		)
		operation.Request.Current = projectTranscriptMessage(
			values[index].Request.Current,
		)
		operation.ProtocolOutputs = nil
		operation.ToolCallIDs = slices.Clone(values[index].ToolCallIDs)
		operation.Metadata = nil

		if values[index].Error != nil {
			errorValue := *values[index].Error
			operation.Error = &errorValue
		}

		output[index] = operation
	}

	return output
}

func projectToolCalls(
	values []sessionRuntime.ToolCall,
) []sessionRuntime.ToolCall {
	if len(values) == 0 {
		return nil
	}

	output := make([]sessionRuntime.ToolCall, len(values))
	for index := range values {
		call := values[index]
		call.Arguments = cloneRawJSON(values[index].Arguments)
		call.Binding.Data = cloneRawJSON(values[index].Binding.Data)
		call.Attempts = slices.Clone(values[index].Attempts)
		call.Results = projectToolResults(values[index].Results)
		call.Authorization = nil
		output[index] = call
	}

	return output
}

func projectWaitingRequests(
	values []sessionRuntime.WaitingRequest,
) []sessionRuntime.WaitingRequest {
	if len(values) == 0 {
		return nil
	}

	output := make([]sessionRuntime.WaitingRequest, len(values))
	for index := range values {
		wait := values[index]
		wait.Summary = cloneRawJSON(values[index].Summary)
		wait.InputSchema = cloneRawJSON(values[index].InputSchema)
		wait.AllowedResolutions = slices.Clone(
			values[index].AllowedResolutions,
		)

		if values[index].ExpiresAt != nil {
			expiresAt := *values[index].ExpiresAt
			wait.ExpiresAt = &expiresAt
		}

		output[index] = wait
	}

	return output
}

func cloneRawJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}

	return append(json.RawMessage(nil), value...)
}
