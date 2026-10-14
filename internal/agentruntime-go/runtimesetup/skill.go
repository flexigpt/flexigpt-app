package runtimesetup

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"strings"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func (c *coordinator) prepareSkills(
	ctx context.Context,
	contributions []skillContribution,
	active map[string]struct{},
	summary *sessionRuntime.OperationResolutionSummary,
) (
	preparedOutput preparedSkills,
	effectiveOutput []sessionRuntime.ArtifactReference,
	activeOutput []sessionRuntime.ArtifactReference,
	instructionOutput []instructionContribution,
	err error,
) {
	if len(contributions) == 0 {
		return preparedSkills{}, nil, nil, nil, nil
	}

	refs := make([]artifactModel.ArtifactRef, 0, len(contributions))
	byKey := make(map[string]skillContribution, len(contributions))

	for _, contribution := range contributions {
		if _, duplicate := byKey[contribution.key]; duplicate {
			continue
		}
		byKey[contribution.key] = contribution

		ref, err := artifactRef(contribution.selection)
		if err != nil {
			return preparedSkills{}, nil, nil, nil, err
		}
		refs = append(refs, ref)
	}

	resolved, err := c.skillResolver.ResolveSkills(ctx, refs)
	if err != nil {
		return preparedSkills{}, nil, nil, nil, operationPreparationError(
			"skill_unavailable",
			err.Error(),
			*summary,
			err,
		)
	}

	output := preparedSkills{
		definitions: make([]provider.SkillDef, 0, len(resolved.Values)),
		byArtifact:  make(map[string]provider.SkillDef, len(resolved.Values)),
		byDefinition: make(
			map[provider.SkillDef]sessionRuntime.ArtifactReference,
			len(resolved.Values),
		),
	}

	effective := make(
		[]sessionRuntime.ArtifactReference,
		0,
		len(resolved.Values),
	)
	activeRefs := make(
		[]sessionRuntime.ArtifactReference,
		0,
		len(resolved.Values),
	)
	usedByDefinition := make(
		map[provider.SkillDef]*sessionRuntime.ResolvedArtifact,
		len(resolved.Values),
	)
	instructionByDefinition := make(
		map[provider.SkillDef]skillContribution,
		len(resolved.Values),
	)

	for _, value := range resolved.Values {
		reference := sessionArtifactReference(value.Artifact)
		contribution, found := byKey[referenceKey(reference)]
		if !found {
			return preparedSkills{}, nil, nil, nil, operationPreparationError(
				"skill_resolution_invalid",
				"skill resolver returned an Artifact that was not requested",
				*summary,
				nil,
			)
		}
		used := resolvedArtifact(
			value.Artifact,
			value.Revision,
			cryptoutil.Digest(value.DefinitionDigest),
		)
		item := contribution.item

		if contribution.owner == sessionRuntime.ConfigurationOwnerSession {
			if err := requireSelectionPin(contribution.selection, used); err != nil {
				item.State = sessionRuntime.ResolutionStateStale
				item.Used = used
				item.Reason = err.Error()
				summary.Skills = append(summary.Skills, item)

				return preparedSkills{}, nil, nil, nil, operationPreparationError(
					"skill_selection_stale",
					err.Error(),
					*summary,
					err,
				)
			}
		}

		if !value.Enabled {
			item.State = sessionRuntime.ResolutionStateUnavailable
			item.Used = used
			item.Reason = "Skill is disabled"
			summary.Skills = append(summary.Skills, item)
			summary.Diagnostics = append(
				summary.Diagnostics,
				sessionRuntime.ResolutionDiagnostic{
					Severity: sessionRuntime.DiagnosticSeverityWarning,
					Code:     "skill_disabled",
					Message:  "Selected Skill is disabled",
					Key:      contribution.key,
				},
			)

			if contribution.owner == sessionRuntime.ConfigurationOwnerSession {
				return preparedSkills{}, nil, nil, nil, operationPreparationError(
					"skill_selection_unavailable",
					"selected Skill is disabled",
					*summary,
					nil,
				)
			}
			continue
		}

		key := artifactKey(value.Artifact)
		output.definitions = append(output.definitions, value.Definition)
		output.byArtifact[key] = value.Definition
		output.byDefinition[value.Definition] = reference
		usedByDefinition[value.Definition] = used
		effective = append(effective, reference)

		item.State = sessionRuntime.ResolutionStateIncluded
		item.Used = used
		summary.Skills = append(summary.Skills, item)

		if _, enabled := active[key]; enabled {
			activeRefs = append(activeRefs, reference)
		}
		if contribution.instruction {
			instructionByDefinition[value.Definition] = contribution
		}
	}

	if len(instructionByDefinition) == 0 {
		return output, effective, activeRefs, nil, nil
	}

	records, err := c.skillRuntime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			AllowSkills: output.definitions,
		},
	)
	if err != nil {
		return preparedSkills{}, nil, nil, nil, operationPreparationError(
			"instruction_skill_unavailable",
			err.Error(),
			*summary,
			err,
		)
	}

	recordByDefinition := make(
		map[provider.SkillDef]agentskillsRuntimeSpec.SkillRecord,
		len(records),
	)
	for _, record := range records {
		recordByDefinition[record.Def] = record
	}

	for _, definition := range output.definitions {
		contribution, needed := instructionByDefinition[definition]
		if !needed {
			continue
		}

		record, found := recordByDefinition[definition]
		if !found {
			return preparedSkills{}, nil, nil, nil, operationPreparationError(
				"instruction_skill_unavailable",
				"Skill Runtime did not index an Agent instruction Skill",
				*summary,
				nil,
			)
		}

		insert, supported := document.NormalizeSkillInsert(record.Insert)
		if !supported || string(insert) != "instructions" {
			return preparedSkills{}, nil, nil, nil, operationPreparationError(
				"instruction_skill_invalid",
				"Agent instruction Skill is not an instructions Skill",
				*summary,
				nil,
			)
		}

		rendered, renderErr := c.skillRuntime.RenderSkill(
			ctx,
			&skillRuntime.RenderSkillRequest{
				Body: &skillRuntime.RenderSkillRequestBody{
					Definition: definition,
					Arguments:  map[string]string{},
				},
			},
		)
		if renderErr != nil {
			return preparedSkills{}, nil, nil, nil, operationPreparationError(
				"instruction_skill_unavailable",
				renderErr.Error(),
				*summary,
				renderErr,
			)
		}
		if rendered == nil || rendered.Body == nil {
			return preparedSkills{}, nil, nil, nil, operationPreparationError(
				"instruction_skill_unavailable",
				"Skill Runtime returned no rendered Agent instruction Skill",
				*summary,
				nil,
			)
		}

		text := strings.TrimSpace(rendered.Body.Text)
		if text == "" {
			summary.Diagnostics = append(
				summary.Diagnostics,
				sessionRuntime.ResolutionDiagnostic{
					Severity: sessionRuntime.DiagnosticSeverityWarning,
					Code:     "instruction_skill_empty",
					Message:  "Agent instruction Skill rendered no instruction text",
					Key:      contribution.key,
				},
			)
			continue
		}

		selected := contribution.selection
		instructionKey := "skill:" + contribution.key + ":instructions"
		instructionOutput = append(
			instructionOutput,
			instructionContribution{
				key:   instructionKey,
				owner: contribution.owner,
				label: selected.ArtifactID,
				text:  text,
				item: sessionRuntime.ResolutionItem{
					Key:      instructionKey,
					Kind:     resolutionItemKindInstruction,
					Owner:    contribution.owner,
					State:    sessionRuntime.ResolutionStateIncluded,
					Selected: &selected,
					Used:     usedByDefinition[definition],
				},
			},
		)
	}

	return output, effective, activeRefs, instructionOutput, nil
}

func (c *coordinator) ensureSkillSession(
	ctx context.Context,
	request sessionRuntime.SkillSessionRequest,
) (sessionRuntime.SkillSessionResult, error) {
	if err := sessionRuntime.RequirePreparedOperation(
		request.Preparation,
	); err != nil {
		return sessionRuntime.SkillSessionResult{}, err
	}

	data, ok := request.Preparation.Private.(*preparedOperation)
	if !ok || data == nil {
		return sessionRuntime.SkillSessionResult{}, errors.New(
			"skill Session request has no operation-owned Skill preparation",
		)
	}
	if len(data.skills.definitions) == 0 {
		return sessionRuntime.SkillSessionResult{}, errors.New(
			"skill Session was requested without effective Skills",
		)
	}

	if !artifactReferencesEqual(
		request.Enabled,
		request.Preparation.EffectiveSkills,
	) {
		return sessionRuntime.SkillSessionResult{}, errors.New(
			"skill Session allow-list differs from immutable operation preparation",
		)
	}

	activeDefinitions := make(
		[]provider.SkillDef,
		0,
		len(request.Active),
	)
	for _, reference := range request.Active {
		definition, found := data.skills.byArtifact[referenceKey(reference)]
		if !found {
			return sessionRuntime.SkillSessionResult{}, errors.New(
				"active Skill is not part of the immutable operation preparation",
			)
		}
		activeDefinitions = append(activeDefinitions, definition)
	}

	response, err := c.skillRuntime.CreateSkillSession(
		ctx,
		&skillRuntime.CreateSkillSessionRequest{
			Body: &skillRuntime.CreateSkillSessionRequestBody{
				CloseSessionID: agentskillsRuntimeSpec.SessionID(
					request.SessionID,
				),
				AllowedSkills: data.skills.definitions,
				ActiveSkills:  activeDefinitions,
			},
		},
	)
	if err != nil {
		return sessionRuntime.SkillSessionResult{}, err
	}
	if response == nil ||
		response.Body == nil ||
		response.Body.SessionID == "" {
		return sessionRuntime.SkillSessionResult{}, errors.New(
			"skill runtime returned no Session identity",
		)
	}

	active := make(
		[]sessionRuntime.ArtifactReference,
		0,
		len(response.Body.ActiveSkills),
	)
	for _, definition := range response.Body.ActiveSkills {
		if reference, found := data.skills.byDefinition[definition]; found {
			active = append(active, reference)
		}
	}

	c.skillMu.Lock()
	if request.SessionID != "" &&
		request.SessionID != string(response.Body.SessionID) {
		delete(c.skillSessions, request.SessionID)
	}
	c.skillSessions[string(response.Body.SessionID)] = preparedSkillSession{
		byDefinition: cloneSkillDefinitions(data.skills.byDefinition),
	}
	c.skillMu.Unlock()

	return sessionRuntime.SkillSessionResult{
		SessionID: string(response.Body.SessionID),
		Active:    active,
	}, nil
}

func (c *coordinator) closeSkillSession(
	ctx context.Context,
	sessionID string,
) error {
	err := c.skillRuntime.CloseSession(
		ctx,
		agentskillsRuntimeSpec.SessionID(sessionID),
	)
	if err != nil {
		return err
	}

	c.skillMu.Lock()
	delete(c.skillSessions, sessionID)
	c.skillMu.Unlock()
	return nil
}

func (c *coordinator) invokeSkillTool(
	ctx context.Context,
	request sessionRuntime.ToolExecutionRequest,
) (sessionRuntime.ToolExecutionResult, error) {
	if strings.TrimSpace(request.SkillSessionID) == "" {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"skill Tool execution requires a Skill Session",
		)
	}

	response, err := c.skillRuntime.InvokeSkillTool(
		ctx,
		&skillRuntime.InvokeSkillToolRequest{
			Body: &skillRuntime.InvokeSkillToolRequestBody{
				SessionID: agentskillsRuntimeSpec.SessionID(
					request.SkillSessionID,
				),
				ToolName: request.ToolCall.Name,
				Args:     string(request.ToolCall.Arguments),
			},
		},
	)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}
	if response == nil || response.Body == nil {
		return sessionRuntime.ToolExecutionResult{}, errors.New(
			"skill Tool invocation returned no response body",
		)
	}

	outputs, err := json.Marshal(response.Body.Outputs)
	if err != nil {
		return sessionRuntime.ToolExecutionResult{}, err
	}

	result := sessionRuntime.ToolExecutionResult{
		HasResult:  true,
		Kind:       sessionRuntime.ToolResultKindSuccess,
		Text:       string(outputs),
		Structured: outputs,
	}
	if response.Body.IsError {
		result.Kind = sessionRuntime.ToolResultKindError
		result.Error = response.Body.ErrorMessage
		if strings.TrimSpace(result.Error) == "" {
			result.Error = "Skill Tool returned an error result"
		}
	}

	if isSkillLifecycleTool(request.ToolCall.Name) {
		active, err := c.activeSkillReferences(
			ctx,
			request.SkillSessionID,
		)
		if err == nil {
			result.SkillSessionActive = active
			result.SkillSessionActiveKnown = true
		}
	}
	return result, nil
}

func (c *coordinator) activeSkillReferences(
	ctx context.Context,
	sessionID string,
) ([]sessionRuntime.ArtifactReference, error) {
	c.skillMu.RLock()
	state := c.skillSessions[sessionID]
	byDefinition := cloneSkillDefinitions(state.byDefinition)
	c.skillMu.RUnlock()

	if len(byDefinition) == 0 {
		return nil, nil
	}

	records, err := c.skillRuntime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			SessionID: agentskillsRuntimeSpec.SessionID(sessionID),
			Activity:  agentskillsRuntimeSpec.SkillActivity("active"),
		},
	)
	if err != nil {
		return nil, err
	}

	output := make(
		[]sessionRuntime.ArtifactReference,
		0,
		len(records),
	)
	for _, record := range records {
		if reference, found := byDefinition[record.Def]; found {
			output = append(output, reference)
		}
	}
	return output, nil
}

func (c *coordinator) hydrateSkillSession(
	ctx context.Context,
	sessionID string,
	skills preparedSkills,
) (string, []inferenceSpec.ToolChoice, preparedBindings, error) {
	if strings.TrimSpace(sessionID) == "" {
		return "", nil, preparedBindings{}, errors.New(
			"prepared operation requires an active Skill Session",
		)
	}

	active, err := c.skillRuntime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			AllowSkills: skills.definitions,
			SessionID: agentskillsRuntimeSpec.SessionID(
				sessionID,
			),
			Activity: agentskillsRuntimeSpec.SkillActivity("active"),
		},
	)
	if err != nil {
		return "", nil, preparedBindings{}, err
	}

	includeAll := len(active) != 0
	prompt, err := c.skillRuntime.SkillsPrompt(
		ctx,
		&agentskillsRuntime.SkillFilter{
			AllowSkills: skills.definitions,
			SessionID: agentskillsRuntimeSpec.SessionID(
				sessionID,
			),
			Activity: agentskillsRuntimeSpec.SkillActivityAny,
		},
	)
	if err != nil {
		return "", nil, preparedBindings{}, err
	}

	choices, bindings, err := skillToolChoices(
		includeAll,
		c.skillRuntime.SupportsRunScript(),
	)
	if err != nil {
		return "", nil, preparedBindings{}, err
	}

	return appendSystemPrompt(
		skillRulesPrompt(
			includeAll,
			c.skillRuntime.SupportsRunScript(),
		),
		prompt,
	), choices, bindings, nil
}

func skillToolChoices(
	includeAll bool,
	includeRunScript bool,
) ([]inferenceSpec.ToolChoice, preparedBindings, error) {
	definitions := []struct {
		name string
		tool llmtoolsSpec.Tool
	}{
		{
			name: skillToolLoadName,
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
				name: skillToolUnloadName,
				tool: agentskillsRuntimeSpec.SkillsUnloadTool(),
			},
			struct {
				name string
				tool llmtoolsSpec.Tool
			}{
				name: skillToolReadResourceName,
				tool: agentskillsRuntimeSpec.SkillsReadResourceTool(),
			},
		)
		if includeRunScript {
			definitions = append(
				definitions,
				struct {
					name string
					tool llmtoolsSpec.Tool
				}{
					name: skillToolRunScriptName,
					tool: agentskillsRuntimeSpec.SkillsRunScriptTool(),
				},
			)
		}
	}

	choices := make(
		[]inferenceSpec.ToolChoice,
		0,
		len(definitions),
	)
	bindings := preparedBindings{
		byChoice: make(
			map[string]sessionRuntime.ToolCallProposal,
			len(definitions),
		),
		byName: make(
			map[string]sessionRuntime.ToolCallProposal,
			len(definitions),
		),
	}

	for _, definition := range definitions {
		schema, err := jsonutil.DecodeJSONStringRawInto[map[string]any](
			jsonutil.JSONRawString(definition.tool.ArgSchema),
		)
		if err != nil {
			return nil, preparedBindings{}, err
		}
		if len(schema) == 0 {
			schema = map[string]any{typeStr: jsonSchemaTypeObject}
		}

		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          "builtin." + definition.name,
			Name:        definition.name,
			Description: definition.tool.Description,
			Arguments:   schema,
		}
		proposal := sessionRuntime.ToolCallProposal{
			Name:     choice.Name,
			ChoiceID: choice.ID,
			Type:     string(choice.Type),
			Required: true,
			Automatic: definition.name == skillToolLoadName ||
				definition.name == skillToolUnloadName ||
				definition.name == skillToolReadResourceName,
			Binding: sessionRuntime.ToolBinding{
				Kind: sessionRuntime.ToolBindingKindSkill,
			},
		}

		choices = append(choices, choice)
		bindings.byChoice[choice.ID] = proposal
		bindings.byName[choice.Name] = proposal
	}

	return choices, bindings, nil
}

func skillRulesPrompt(
	includeAll bool,
	includeRunScript bool,
) string {
	if !includeAll {
		return agentskillsRuntimeSpec.SkillsRulesPromptLoadOnly
	}
	if !includeRunScript {
		return agentskillsRuntimeSpec.SkillsRulesPromptWithoutRunScript
	}
	return agentskillsRuntimeSpec.SkillsRulesPromptAll
}

func isSkillLifecycleTool(name string) bool {
	switch name {
	case skillToolLoadName, skillToolUnloadName:
		return true
	default:
		return false
	}
}

func cloneSkillDefinitions(
	input map[provider.SkillDef]sessionRuntime.ArtifactReference,
) map[provider.SkillDef]sessionRuntime.ArtifactReference {
	output := make(
		map[provider.SkillDef]sessionRuntime.ArtifactReference,
		len(input),
	)
	maps.Copy(output, input)
	return output
}

func artifactReferencesEqual(
	left []sessionRuntime.ArtifactReference,
	right []sessionRuntime.ArtifactReference,
) bool {
	if len(left) != len(right) {
		return false
	}

	seen := make(
		map[string]struct{},
		len(left),
	)
	for _, reference := range left {
		seen[referenceKey(reference)] = struct{}{}
	}
	for _, reference := range right {
		if _, found := seen[referenceKey(reference)]; !found {
			return false
		}
	}
	return true
}

func referenceKey(
	value sessionRuntime.ArtifactReference,
) string {
	return "artifact:" + value.RootID + ":" + value.ArtifactID
}
