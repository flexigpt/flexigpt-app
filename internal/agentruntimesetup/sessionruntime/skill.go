package sessionruntime

import (
	"context"
	"errors"
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
	activeSelections map[string]sessionRuntime.ArtifactSelection,
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
	instructionDefinitions := make(
		[]provider.SkillDef,
		0,
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
		used := resolvedArtifactWithDetails(
			value.Artifact,
			value.Revision,
			cryptoutil.Digest(value.DefinitionDigest),
			firstNonEmpty(
				value.DisplayName,
				value.Definition.Name,
				"Skill",
			),
			value.Description,
			firstNonEmpty(
				value.SourceLocation,
				value.Definition.Location,
			),
			nil,
		)
		item := contribution.item
		key := artifactKey(value.Artifact)
		activeSelection, activeSelected := activeSelections[key]
		_, activeNow := active[key]
		item.Used = used
		item.Skill = &sessionRuntime.SkillResolutionState{
			Enabled:          value.Enabled,
			Active:           activeNow,
			SessionAvailable: false,
		}

		if contribution.owner == sessionRuntime.ConfigurationOwnerSession {
			if err := requireSelectionPin(contribution.selection, used); err != nil {
				item.State = sessionRuntime.ResolutionStateStale
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

		if activeSelected {
			if err := requireSelectionPin(activeSelection, used); err != nil {
				selected := activeSelection
				summary.Skills = append(
					summary.Skills,
					sessionRuntime.ResolutionItem{
						Key:      key,
						Kind:     resolutionItemKindSkill,
						Owner:    sessionRuntime.ConfigurationOwnerSession,
						State:    sessionRuntime.ResolutionStateStale,
						Selected: &selected,
						Used:     used,
						Reason:   err.Error(),
					},
				)
				return preparedSkills{}, nil, nil, nil, operationPreparationError(
					"active_skill_selection_stale", err.Error(), *summary, err,
				)
			}
		}

		if !value.Enabled {
			item.State = sessionRuntime.ResolutionStateUnavailable
			item.Reason = "Skill is disabled"
			summary.Skills = append(summary.Skills, item)
			severity := sessionRuntime.DiagnosticSeverityWarning
			if contribution.owner == sessionRuntime.ConfigurationOwnerSession ||
				contribution.required ||
				activeSelected {
				severity = sessionRuntime.DiagnosticSeverityError
			}
			summary.Diagnostics = append(
				summary.Diagnostics,
				sessionRuntime.ResolutionDiagnostic{
					Severity: severity,
					Code:     "skill_disabled",
					Message:  "Selected Skill is disabled",
					Key:      contribution.key,
				},
			)

			if contribution.owner == sessionRuntime.ConfigurationOwnerSession ||
				contribution.required ||
				activeSelected {
				code := "skill_selection_unavailable"
				message := "selected Skill is disabled"
				switch {
				case activeSelected:
					code = "active_skill_selection_unavailable"
					message = "selected active Skill is disabled"
				case contribution.required:
					code = "agent_required_skill_unavailable"
					message = "required Agent Skill is disabled"
				}
				return preparedSkills{}, nil, nil, nil, operationPreparationError(
					code,
					message,
					*summary,
					nil,
				)
			}
			continue
		}

		/*
		 * An Agent contribution whose use mode is "instructions" is not a
		 * Skill-session contribution by itself.
		 *
		 * It is rendered into the system prompt below, but it must not enter
		 * EffectiveSkills, create a Skill session, or inject Skill lifecycle
		 * tools. A deliberate Session Skill Add override for the same Artifact
		 * replaces this contribution with instruction=false before this point,
		 * which is how an explicit "enable as Skill too" request remains
		 * supported.
		 */
		if contribution.instruction {
			if _, requestedActive := active[key]; requestedActive {
				return preparedSkills{}, nil, nil, nil, operationPreparationError(
					"instruction_skill_active_invalid",
					"an instruction-only Skill must be explicitly enabled before it can be active",
					*summary,
					nil,
				)
			}

			if _, alreadyRecorded := instructionByDefinition[value.Definition]; !alreadyRecorded {
				instructionDefinitions = append(instructionDefinitions, value.Definition)
			}
			instructionByDefinition[value.Definition] = contribution
			usedByDefinition[value.Definition] = used
			continue
		}

		output.definitions = append(output.definitions, value.Definition)
		output.byArtifact[key] = value.Definition
		output.byDefinition[value.Definition] = reference
		usedByDefinition[value.Definition] = used
		effective = append(effective, reference)

		item.State = sessionRuntime.ResolutionStateIncluded
		item.Skill.SessionAvailable = true
		summary.Skills = append(summary.Skills, item)

		if _, enabled := active[key]; enabled {
			activeRefs = append(activeRefs, reference)
		}
	}

	if len(instructionByDefinition) == 0 {
		return output, effective, activeRefs, nil, nil
	}

	records, err := c.skillRuntime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			AllowSkills: instructionDefinitions,
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

	for _, definition := range instructionDefinitions {
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
			Name:        choice.Name,
			ChoiceID:    choice.ID,
			Type:        choice.Type,
			Required:    true,
			DisplayName: skillToolDisplayName(definition.name),
			Description: strings.TrimSpace(definition.tool.Description),
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

func skillToolDisplayName(name string) string {
	switch name {
	case skillToolLoadName:
		return "Load Skill"
	case skillToolUnloadName:
		return "Unload Skill"
	case skillToolReadResourceName:
		return "Read Skill resource"
	case skillToolRunScriptName:
		return "Run Skill script"
	default:
		return "Skill Tool"
	}
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

func referenceKey(
	value sessionRuntime.ArtifactReference,
) string {
	return "artifact:" + value.RootID + ":" + value.ArtifactID
}
