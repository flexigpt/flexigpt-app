package skillruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	v3SkillToolLoadName         = "skills-load"
	v3SkillToolUnloadName       = "skills-unload"
	v3SkillToolReadResourceName = "skills-readresource"
	v3SkillToolRunScriptName    = "skills-runscript"
)

// V3Executor owns source-backed Skill selection and per-Execution Skill
// Runtime sessions. Session v3 only stores Artifact selections and generic
// Tool WorkItems.
type V3Executor struct {
	resolver *Resolver
	runtime  *skillRuntime.Service

	mu       sync.Mutex
	sessions map[sessionv3Runtime.ExecutionID]v3SkillSession
}

type v3SkillSession struct {
	id          string
	definitions []provider.SkillDef
}

func NewV3Executor(
	resolver *Resolver,
	runtime *skillRuntime.Service,
) (*V3Executor, error) {
	if resolver == nil || runtime == nil {
		return nil, errors.New("session v3 Skill runtime dependencies are incomplete")
	}

	return &V3Executor{
		resolver: resolver,
		runtime:  runtime,
		sessions: make(map[sessionv3Runtime.ExecutionID]v3SkillSession),
	}, nil
}

// Choices exposes Skill lifecycle Tools. When an existing runtime session has
// active Skills, the full lifecycle set is visible; otherwise only loading is
// available.
func (e *V3Executor) Choices(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) ([]toolruntime.V3PreparedChoice, error) {
	if len(request.Session.Defaults.Skills) == 0 {
		return nil, nil
	}

	includeAll := len(request.Session.Defaults.ActiveSkills) != 0
	session := e.executionSession(request.Execution.ID)
	if session.id != "" {
		active, err := e.runtime.ListAgentSkills(
			ctx,
			&agentskillsRuntime.SkillListFilter{
				AllowSkills: session.definitions,
				SessionID: agentskillsRuntimeSpec.SessionID(
					session.id,
				),
				Activity: agentskillsRuntimeSpec.SkillActivity("active"),
			},
		)
		if err != nil {
			return nil, err
		}
		includeAll = len(active) != 0
	}

	return e.v3SkillChoices(
		includeAll,
		e.runtime.SupportsRunScript(),
	)
}

// Instruction resolves and renders an instruction-mode Skill through the
// Skill domain. It does not create an Execution-owned Skill session.
func (e *V3Executor) Instruction(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
) (string, error) {
	if e == nil || e.resolver == nil || e.runtime == nil {
		return "", errors.New("session v3 skill runtime is unavailable")
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(selection.RootID),
		ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
	}
	if err := ref.Validate(); err != nil {
		return "", err
	}

	resolved, err := e.resolver.ResolveSkills(
		ctx,
		[]artifactModel.ArtifactRef{ref},
	)
	if err != nil {
		return "", err
	}
	if len(resolved.Values) != 1 {
		return "", errors.New(
			"skill resolver returned an unexpected instruction skill count",
		)
	}

	value := resolved.Values[0]
	if value.Artifact != ref {
		return "", errors.New(
			"skill resolver returned another instruction skill",
		)
	}
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != value.Revision {
		return "", errors.New("selected skill revision changed")
	}
	if selection.ExpectedDigest != "" &&
		selection.ExpectedDigest != value.DefinitionDigest {
		return "", errors.New("selected skill definition changed")
	}
	if !value.Enabled {
		return "", errors.New("selected instruction skill is disabled")
	}

	records, err := e.runtime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			AllowSkills: []provider.SkillDef{value.Definition},
		},
	)
	if err != nil {
		return "", err
	}
	for _, record := range records {
		if record.Def != value.Definition {
			continue
		}
		insert, supported := document.NormalizeSkillInsert(record.Insert)
		if !supported || insert != document.SkillInsertInstructions {
			return "", errors.New(
				"selected skill is not an instruction skill",
			)
		}

		rendered, err := e.runtime.RenderSkill(
			ctx,
			&skillRuntime.RenderSkillRequest{
				Body: &skillRuntime.RenderSkillRequestBody{
					Definition: value.Definition,
					Arguments:  map[string]string{},
				},
			},
		)
		if err != nil {
			return "", err
		}
		if rendered == nil || rendered.Body == nil {
			return "", errors.New(
				"skill runtime returned no rendered instruction skill",
			)
		}
		return strings.TrimSpace(rendered.Body.Text), nil
	}

	return "", errors.New("skill runtime did not index the selected instruction skill")
}

// Bind verifies the Skill allow-list and active-skill selections before a
// Skill Tool WorkItem is dispatched. Runtime Session allocation remains lazy
// and occurs only inside Execute.
func (e *V3Executor) Bind(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.SemanticBinding, error) {
	if e == nil || e.resolver == nil || e.runtime == nil {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"session v3 Skill runtime is unavailable",
		)
	}
	if payload == nil ||
		payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindSkill ||
		payload.Call.Binding.Skill == nil ||
		payload.Call.Binding.Skill.Name != payload.Call.Name {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"skill tool binding does not match the requested Tool",
		)
	}
	if err := payload.Call.Validate(); err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}
	invocation := v3SkillWorkflowInvocation(request, payload)
	if invocation != nil &&
		(!payload.Call.Binding.Skill.WorkflowPorts ||
			!e.runtime.SupportsWorkflowToolInvocation()) {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"skill tool does not support workflow port transport",
		)
	}
	if len(request.Session.Defaults.Skills) == 0 {
		return sessionv3Runtime.SemanticBinding{}, errors.New(
			"skill tool was requested without enabled Skills",
		)
	}
	if _, _, err := e.resolveSelections(
		ctx,
		request.Session.Defaults.Skills,
		request.Session.Defaults.ActiveSkills,
	); err != nil {
		return sessionv3Runtime.SemanticBinding{}, err
	}

	artifacts := make(
		[]sessionv3Runtime.ArtifactPin,
		0,
		len(request.Session.Defaults.Skills),
	)
	for _, selection := range request.Session.Defaults.Skills {
		if selection.ExpectedRevision == 0 ||
			strings.TrimSpace(selection.ExpectedDigest) == "" {
			continue
		}
		artifacts = append(artifacts, sessionv3Runtime.ArtifactPin{
			RootID:     selection.RootID,
			ArtifactID: selection.ArtifactID,
			Revision:   selection.ExpectedRevision,
			Digest:     selection.ExpectedDigest,
		})
	}

	return sessionv3Runtime.SemanticBinding{
		Version:           "session-v3-skill/v1",
		Artifacts:         artifacts,
		TargetFingerprint: "skill:" + payload.Call.Name,
		BoundAt:           time.Now().UTC(),
	}, nil
}

// Execute handles only Skill Tool bindings.
func (e *V3Executor) Execute(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) (sessionv3Runtime.WorkExecutionResult, bool, error) {
	if payload == nil ||
		payload.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindSkill {
		return sessionv3Runtime.WorkExecutionResult{}, false, nil
	}
	if payload.Call.Binding.Skill == nil ||
		payload.Call.Binding.Skill.Name != payload.Call.Name {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "invalid_skill_binding",
				Message: "Skill Tool binding does not match the requested Tool",
			},
		}, true, nil
	}

	invocation := v3SkillWorkflowInvocation(request, payload)
	if invocation != nil && !e.runtime.SupportsWorkflowToolInvocation() {
		return sessionv3Runtime.WorkExecutionResult{}, true, errors.New(
			"skill tool does not support workflow port transport",
		)
	}
	sessionID, err := e.ensureSession(ctx, request)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "skill_session_failed",
				Message: err.Error(),
			},
		}, true, nil
	}

	arguments := payload.Call.Arguments.Clone()
	if len(arguments) == 0 {
		arguments = sessionv3Runtime.MustCanonicalValue(map[string]any{})
	}
	if !arguments.Valid() {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "invalid_skill_arguments",
				Message: "Skill Tool arguments are not canonical JSON",
			},
		}, true, nil
	}

	response, err := e.runtime.InvokeSkillTool(
		ctx,
		&skillRuntime.InvokeSkillToolRequest{
			Body: &skillRuntime.InvokeSkillToolRequestBody{
				SessionID: agentskillsRuntimeSpec.SessionID(sessionID),
				ToolName:  payload.Call.Name,
				Args:      string(arguments),
				Workflow:  invocation,
			},
		},
	)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "skill_failed",
				Message: err.Error(),
			},
		}, true, nil
	}
	if response == nil || response.Body == nil {
		return sessionv3Runtime.WorkExecutionResult{
			Error: &sessionv3Runtime.SanitizedError{
				Code:    "skill_empty",
				Message: "Skill Tool returned no response",
			},
		}, true, nil
	}

	text, structured, err := toolruntime.V3ToolOutputs(response.Body.Outputs)
	if err != nil {
		return sessionv3Runtime.WorkExecutionResult{}, true, err
	}

	kind := sessionv3Runtime.ToolOutputKindSuccess
	var executionError *sessionv3Runtime.SanitizedError
	if response.Body.IsError {
		kind = sessionv3Runtime.ToolOutputKindError
		message := firstV3SkillText(
			response.Body.ErrorMessage,
			"Skill Tool returned an error result",
		)
		executionError = &sessionv3Runtime.SanitizedError{
			Code:    "skill_tool_error",
			Message: message,
		}
		if strings.TrimSpace(text) == "" {
			text = message
		}
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: structured,
			Tool: &sessionv3Runtime.ToolOutput{
				Kind:       kind,
				Text:       text,
				Structured: structured,
				Error:      firstV3SkillText(response.Body.ErrorMessage),
			},
			Receipt: &sessionv3Runtime.ExternalReceipt{
				System:      "skill",
				Reference:   sessionID + ":" + payload.Call.Name,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Error:     executionError,
		Confirmed: true,
	}, true, nil
}

// CloseExecution closes the process-local Skill Runtime session associated
// with one v3 Execution.
func (e *V3Executor) CloseExecution(
	ctx context.Context,
	executionID sessionv3Runtime.ExecutionID,
) error {
	if e == nil || executionID == "" {
		return nil
	}

	e.mu.Lock()
	session, found := e.sessions[executionID]
	delete(e.sessions, executionID)
	e.mu.Unlock()

	if !found || session.id == "" {
		return nil
	}
	return e.runtime.CloseSession(
		ctx,
		agentskillsRuntimeSpec.SessionID(session.id),
	)
}

func (e *V3Executor) ensureSession(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) (string, error) {
	if existing := e.executionSession(request.Execution.ID); existing.id != "" {
		return existing.id, nil
	}
	if len(request.Session.Defaults.Skills) == 0 {
		return "", errors.New("skill tool was requested without enabled Skills")
	}

	definitions, active, err := e.resolveSelections(
		ctx,
		request.Session.Defaults.Skills,
		request.Session.Defaults.ActiveSkills,
	)
	if err != nil {
		return "", err
	}

	created, err := e.runtime.CreateSkillSession(
		ctx,
		&skillRuntime.CreateSkillSessionRequest{
			Body: &skillRuntime.CreateSkillSessionRequestBody{
				AllowedSkills: definitions,
				ActiveSkills:  active,
			},
		},
	)
	if err != nil {
		return "", err
	}
	if created == nil ||
		created.Body == nil ||
		created.Body.SessionID == "" {
		return "", errors.New("skill Runtime returned no Session identity")
	}

	next := v3SkillSession{
		id:          string(created.Body.SessionID),
		definitions: append([]provider.SkillDef(nil), definitions...),
	}

	e.mu.Lock()
	existing, found := e.sessions[request.Execution.ID]
	if !found {
		e.sessions[request.Execution.ID] = next
	}
	e.mu.Unlock()

	if found {
		_ = e.runtime.CloseSession(
			context.WithoutCancel(ctx),
			created.Body.SessionID,
		)
		return existing.id, nil
	}
	return next.id, nil
}

func (e *V3Executor) resolveSelections(
	ctx context.Context,
	selections []sessionv3Runtime.ArtifactSelection,
	activeSelections []sessionv3Runtime.ArtifactSelection,
) (defs, active []provider.SkillDef, err error) {
	refs := make(
		[]artifactModel.ArtifactRef,
		0,
		len(selections),
	)
	selectionByKey := make(
		map[string]sessionv3Runtime.ArtifactSelection,
		len(selections),
	)

	for _, selection := range selections {
		ref := artifactModel.ArtifactRef{
			RootID:     rootModel.RootID(selection.RootID),
			ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
		}
		if err := ref.Validate(); err != nil {
			return nil, nil, err
		}
		key := v3SkillArtifactKey(selection.RootID, selection.ArtifactID)
		if _, duplicate := selectionByKey[key]; duplicate {
			return nil, nil, fmt.Errorf(
				"duplicate selected Skill %s/%s",
				selection.RootID,
				selection.ArtifactID,
			)
		}
		selectionByKey[key] = selection
		refs = append(refs, ref)
	}

	resolved, err := e.resolver.ResolveSkills(ctx, refs)
	if err != nil {
		return nil, nil, err
	}
	if len(resolved.Values) != len(refs) {
		return nil, nil, errors.New(
			"skill resolver returned an unexpected selected Skill count",
		)
	}

	definitions := make([]provider.SkillDef, 0, len(resolved.Values))
	definitionByKey := make(
		map[string]provider.SkillDef,
		len(resolved.Values),
	)
	for _, value := range resolved.Values {
		key := v3SkillArtifactKey(
			string(value.Artifact.RootID),
			string(value.Artifact.ArtifactID),
		)
		selection, found := selectionByKey[key]
		if !found {
			return nil, nil, errors.New(
				"skill resolver returned an Artifact that was not requested",
			)
		}
		if selection.ExpectedRevision != 0 &&
			selection.ExpectedRevision != value.Revision {
			return nil, nil, errors.New("selected Skill revision changed")
		}
		if selection.ExpectedDigest != "" &&
			selection.ExpectedDigest != value.DefinitionDigest {
			return nil, nil, errors.New("selected Skill Definition changed")
		}
		if !value.Enabled {
			return nil, nil, fmt.Errorf(
				"selected Skill %q is disabled",
				value.Artifact.ArtifactID,
			)
		}

		definitions = append(definitions, value.Definition)
		definitionByKey[key] = value.Definition
	}

	active = make([]provider.SkillDef, 0, len(activeSelections))
	for _, selection := range activeSelections {
		key := v3SkillArtifactKey(selection.RootID, selection.ArtifactID)
		definition, found := definitionByKey[key]
		if !found {
			return nil, nil, errors.New(
				"active Skill is not part of the enabled Skill selection",
			)
		}
		active = append(active, definition)
	}
	return definitions, active, nil
}

func (e *V3Executor) executionSession(
	executionID sessionv3Runtime.ExecutionID,
) v3SkillSession {
	if e == nil || executionID == "" {
		return v3SkillSession{}
	}

	e.mu.Lock()
	value := e.sessions[executionID]
	e.mu.Unlock()
	value.definitions = append([]provider.SkillDef(nil), value.definitions...)
	return value
}

func (e *V3Executor) v3SkillChoices(
	includeAll bool,
	includeRunScript bool,
) ([]toolruntime.V3PreparedChoice, error) {
	definitions := []struct {
		name string
		tool llmtoolsSpec.Tool
	}{
		{
			name: v3SkillToolLoadName,
			tool: agentskillsRuntimeSpec.SkillsLoadTool(),
		},
	}
	if includeAll {
		definitions = append(
			definitions,
			struct {
				name string
				tool llmtoolsSpec.Tool
			}{
				name: v3SkillToolUnloadName,
				tool: agentskillsRuntimeSpec.SkillsUnloadTool(),
			},
			struct {
				name string
				tool llmtoolsSpec.Tool
			}{
				name: v3SkillToolReadResourceName,
				tool: agentskillsRuntimeSpec.SkillsReadResourceTool(),
			},
		)
		if includeRunScript {
			definitions = append(definitions, struct {
				name string
				tool llmtoolsSpec.Tool
			}{
				name: v3SkillToolRunScriptName,
				tool: agentskillsRuntimeSpec.SkillsRunScriptTool(),
			})
		}
	}

	output := make([]toolruntime.V3PreparedChoice, 0, len(definitions))
	for _, definition := range definitions {
		raw, err := jsonutil.DecodeJSONStringRaw(
			jsonutil.JSONRawString(definition.tool.ArgSchema),
		)
		if err != nil {
			return nil, err
		}
		arguments, err := toolruntime.InputSchema(raw)
		if err != nil {
			return nil, err
		}

		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          "builtin." + definition.name,
			Name:        definition.name,
			Description: definition.tool.Description,
			Arguments:   arguments,
		}
		call := sessionv3Runtime.ToolCall{
			Name:        definition.name,
			ChoiceID:    choice.ID,
			DisplayName: v3SkillToolDisplayName(definition.name),
			Description: definition.tool.Description,
			Automatic: definition.name == v3SkillToolLoadName ||
				definition.name == v3SkillToolUnloadName ||
				definition.name == v3SkillToolReadResourceName,
			Binding: sessionv3Runtime.ToolBinding{
				Kind: sessionv3Runtime.ToolBindingKindSkill,
				Skill: &sessionv3Runtime.SkillToolBinding{
					Name:          definition.name,
					WorkflowPorts: e.runtime.SupportsWorkflowToolInvocation(),
				},
			},
		}
		output = append(output, toolruntime.V3PreparedChoice{
			Choice: choice,
			Call:   call,
		})
	}
	return output, nil
}

func v3SkillToolDisplayName(name string) string {
	switch name {
	case v3SkillToolLoadName:
		return "Load Skill"
	case v3SkillToolUnloadName:
		return "Unload Skill"
	case v3SkillToolReadResourceName:
		return "Read Skill resource"
	case v3SkillToolRunScriptName:
		return "Run Skill script"
	default:
		return "Skill Tool"
	}
}

func v3SkillWorkflowInvocation(
	request sessionv3Runtime.WorkDispatchRequest,
	payload *sessionv3Runtime.ToolWork,
) *skillRuntime.WorkflowToolInvocation {
	if payload == nil {
		return nil
	}

	outputPorts := make([]string, 0)
	for _, plan := range request.WorkItem.Ports {
		if plan.Direction == sessionv3Runtime.WorkflowPortDirectionOutput &&
			plan.Stream {
			outputPorts = append(outputPorts, plan.Name)
		}
	}
	if len(payload.Context) == 0 &&
		len(request.WorkItem.PortInputs) == 0 &&
		len(outputPorts) == 0 {
		return nil
	}

	output := &skillRuntime.WorkflowToolInvocation{
		Context: append(json.RawMessage(nil), payload.Context...),
		PortInputs: make(
			[]skillRuntime.WorkflowPortInput,
			0,
			len(request.WorkItem.PortInputs),
		),
		OutputPorts: outputPorts,
	}
	for _, input := range request.WorkItem.PortInputs {
		value := skillRuntime.WorkflowPortInput{
			Name:     input.Name,
			Channel:  input.Channel,
			Cursor:   input.Cursor,
			Streamed: input.Streamed,
			Values:   make([]json.RawMessage, 0, len(input.Values)),
		}
		for _, current := range input.Values {
			value.Values = append(
				value.Values,
				json.RawMessage(current.Clone()),
			)
		}
		output.PortInputs = append(output.PortInputs, value)
	}
	if request.PortStream.Emit != nil {
		output.Emit = func(
			name string,
			value json.RawMessage,
		) error {
			canonical, err := sessionv3Runtime.CanonicalValueFrom(value)
			if err != nil {
				return err
			}
			return request.PortStream.Emit(name, canonical)
		}
	}
	return output
}

func v3SkillArtifactKey(rootID, artifactID string) string {
	return rootID + "\x00" + artifactID
}

func firstV3SkillText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
