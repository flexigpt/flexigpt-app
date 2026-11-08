package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

type llmExecutionPlan struct {
	model              *sessionv3Runtime.ArtifactSelection
	defaults           sessionv3Runtime.SessionDefaults
	systemInstructions []sessionv3Runtime.Instruction
	texts              []sessionv3Runtime.TextDispatchPlan
	context            []string
	artifacts          []sessionv3Runtime.ArtifactPin
}

func (b *Binder) prepareLLMPlan(
	ctx context.Context,
	request sessionv3Runtime.WorkDispatchRequest,
) (llmExecutionPlan, error) {
	if b == nil {
		return llmExecutionPlan{}, errors.New("session v3 binder is unavailable")
	}

	payload := request.WorkItem.Input.LLM
	if payload == nil {
		return llmExecutionPlan{}, errors.New("llm payload is missing")
	}

	defaults := jsonutil.CloneJSONInValue(request.Session.Defaults)
	if payload.InvocationDefaults != nil {
		defaults = *payload.InvocationDefaults
	}
	if payload.Model != nil {
		defaults.Model = cloneLLMArtifactSelection(payload.Model)
	}
	if payload.Agent != nil {
		defaults.Agent = cloneLLMArtifactSelection(payload.Agent)
	}

	plan := llmExecutionPlan{
		model:    cloneLLMArtifactSelection(defaults.Model),
		defaults: defaults,
	}
	for _, instruction := range defaults.SystemInstructions {
		plan.addInstruction(instruction)
	}
	for _, instruction := range payload.SystemInstructions {
		plan.addInstruction(instruction)
	}

	if defaults.Agent != nil {
		if err := b.applyAgentPlan(ctx, &plan, *defaults.Agent); err != nil {
			return llmExecutionPlan{}, err
		}
	}
	if defaults.Workspace != nil {
		if err := b.applyWorkspacePlan(ctx, &plan, *defaults.Workspace); err != nil {
			return llmExecutionPlan{}, err
		}
	}

	if plan.model == nil {
		return llmExecutionPlan{}, errors.New("no effective model selection is available")
	}
	plan.defaults.Model = cloneLLMArtifactSelection(plan.model)

	var err error
	var pins []sessionv3Runtime.ArtifactPin

	plan.defaults.Tools, pins, err = b.pinSelections(
		ctx,
		plan.defaults.Tools,
	)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	for _, pin := range pins {
		if err := plan.addArtifact(pin); err != nil {
			return llmExecutionPlan{}, err
		}
	}

	plan.defaults.Skills, pins, err = b.pinSelections(
		ctx,
		plan.defaults.Skills,
	)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	for _, pin := range pins {
		if err := plan.addArtifact(pin); err != nil {
			return llmExecutionPlan{}, err
		}
	}

	plan.defaults.ActiveSkills, pins, err = b.pinSelections(
		ctx,
		plan.defaults.ActiveSkills,
	)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	for _, pin := range pins {
		if err := plan.addArtifact(pin); err != nil {
			return llmExecutionPlan{}, err
		}
	}

	plan.defaults.MCP, pins, err = b.pinMCPSelections(
		ctx,
		plan.defaults.MCP,
	)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	for _, pin := range pins {
		if err := plan.addArtifact(pin); err != nil {
			return llmExecutionPlan{}, err
		}
	}

	if len(plan.defaults.MCP) != 0 {
		if b.mcpTools == nil {
			return llmExecutionPlan{}, errors.New(
				"session v3 mcp runtime is unavailable",
			)
		}
		contextValues, contextErr := b.mcpTools.Context(
			ctx,
			plan.defaults.MCP,
		)
		if contextErr != nil {
			return llmExecutionPlan{}, contextErr
		}
		for _, value := range contextValues {
			plan.addContext(value)
		}
	}

	plan.defaults.SystemInstructions = append(
		[]sessionv3Runtime.Instruction(nil),
		plan.systemInstructions...,
	)
	return plan, nil
}

func (b *Binder) applyAgentPlan(
	ctx context.Context,
	plan *llmExecutionPlan,
	selection sessionv3Runtime.ArtifactSelection,
) error {
	if b.deps.Agents == nil {
		return errors.New("session v3 agent runtime is unavailable")
	}

	pin, err := b.pinSelection(ctx, selection)
	if err != nil {
		return err
	}
	if err := plan.addArtifact(pin); err != nil {
		return err
	}

	ref := artifactRefForSelection(selection)
	if err := ref.Validate(); err != nil {
		return err
	}

	resolved, err := b.deps.Agents.ResolveAgent(ctx, ref)
	if err != nil {
		return err
	}

	for _, occurrence := range resolved.Capabilities.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil {
			if occurrence.Required {
				return fmt.Errorf(
					"required agent capability %q is unavailable",
					occurrence.Path,
				)
			}
			continue
		}

		switch occurrence.Type {
		case declaration.TypeModel:
			if plan.model != nil {
				continue
			}
			value, modelPin, selectionErr := b.selectionForCapabilityTarget(
				ctx,
				occurrence.Target,
			)
			if selectionErr != nil {
				if occurrence.Required {
					return selectionErr
				}
				continue
			}
			plan.model = &value
			if err := plan.addArtifact(modelPin); err != nil {
				return err
			}

		case declaration.TypeText:
			value, textPin, selectionErr := b.selectionForCapabilityTarget(
				ctx,
				occurrence.Target,
			)
			if selectionErr != nil {
				if occurrence.Required {
					return selectionErr
				}
				continue
			}
			if err := plan.addArtifact(textPin); err != nil {
				return err
			}
			if err := b.appendAgentTextInstruction(
				ctx,
				plan,
				value,
				textPin,
			); err != nil && occurrence.Required {
				return err
			}

		case declaration.TypeTool:
			value, toolPin, selectionErr := b.selectionForCapabilityTarget(
				ctx,
				occurrence.Target,
			)
			if selectionErr != nil {
				if occurrence.Required {
					return selectionErr
				}
				continue
			}
			plan.defaults.Tools = appendUniqueLLMSelection(
				plan.defaults.Tools,
				value,
			)
			if err := plan.addArtifact(toolPin); err != nil {
				return err
			}

		case declaration.TypeSkill:
			value, skillPin, selectionErr := b.selectionForCapabilityTarget(
				ctx,
				occurrence.Target,
			)
			if selectionErr != nil {
				if occurrence.Required {
					return selectionErr
				}
				continue
			}
			if err := plan.addArtifact(skillPin); err != nil {
				return err
			}

			if occurrence.SkillUseMode ==
				agentAPI.AgentSkillUseModeInstructions {
				if b.skillTools == nil {
					return errors.New(
						"session v3 skill runtime is unavailable",
					)
				}
				instruction, instructionErr := b.skillTools.Instruction(
					ctx,
					value,
				)
				if instructionErr != nil {
					if occurrence.Required {
						return instructionErr
					}
					continue
				}
				plan.addInstruction(sessionv3Runtime.Instruction{
					ID: "agent-skill:" +
						value.RootID + ":" +
						value.ArtifactID,
					Label:   value.ArtifactID,
					Content: instruction,
				})
				continue
			}

			plan.defaults.Skills = appendUniqueLLMSelection(
				plan.defaults.Skills,
				value,
			)
			if occurrence.SkillUseMode ==
				agentAPI.AgentSkillUseModeActive {
				plan.defaults.ActiveSkills = appendUniqueLLMSelection(
					plan.defaults.ActiveSkills,
					value,
				)
			}

		case declaration.TypeMCP:
			value, serverPin, selectionErr := b.selectionForCapabilityTarget(
				ctx,
				occurrence.Target,
			)
			if selectionErr != nil {
				if occurrence.Required {
					return selectionErr
				}
				continue
			}
			if err := plan.addArtifact(serverPin); err != nil {
				return err
			}
			plan.defaults.MCP = appendUniqueMCPSelection(
				plan.defaults.MCP,
				sessionv3Runtime.MCPServerSelection{
					Server:                    value,
					IncludeServerInstructions: true,
					ToolExposure:              sessionv3Runtime.MCPToolExposureAll,
				},
			)

		case declaration.TypeWorkspace:
			value, workspacePin, selectionErr := b.selectionForCapabilityTarget(
				ctx,
				occurrence.Target,
			)
			if selectionErr != nil {
				if occurrence.Required {
					return selectionErr
				}
				continue
			}
			if plan.defaults.Workspace == nil {
				plan.defaults.Workspace = &value
			}
			if err := plan.addArtifact(workspacePin); err != nil {
				return err
			}
		default:
		}
	}

	return nil
}

func (b *Binder) appendAgentTextInstruction(
	ctx context.Context,
	plan *llmExecutionPlan,
	selection sessionv3Runtime.ArtifactSelection,
	pin sessionv3Runtime.ArtifactPin,
) error {
	if b.deps.Texts == nil {
		return errors.New("session v3 text runtime is unavailable")
	}

	ref := artifactRefForSelection(selection)
	if err := ref.Validate(); err != nil {
		return err
	}

	materialized, err := b.deps.Texts.Materialize(ctx, ref)
	if err != nil {
		return err
	}
	if materialized.Artifact != ref {
		return errors.New(
			"agent text materialization returned another artifact",
		)
	}
	if materialized.ArtifactRevision != pin.Revision ||
		string(materialized.DefinitionDigest) != pin.Digest {
		return errors.New(
			"agent text materialization differs from its resolved artifact pin",
		)
	}

	insert, err := textDispatchInsert(string(materialized.Insert))
	if err != nil {
		return err
	}
	plan.addText(sessionv3Runtime.TextDispatchPlan{
		Artifact: pin,
		Insert:   insert,
		Content:  materialized.Content,
	})

	switch materialized.Insert {
	case "instructions":
		plan.addInstruction(sessionv3Runtime.Instruction{
			ID: "agent-text:" +
				selection.RootID + ":" +
				selection.ArtifactID,
			Label:   string(materialized.Name),
			Content: materialized.Content,
		})
		return nil

	case "user-message":
		// Agent user-message Text remains Composer starter material. V3 has
		// no frontend adoption yet, so it must not become an implicit hidden
		// user message in a backend invocation.
		return nil

	default:
		return fmt.Errorf(
			"agent text has unsupported insertion mode %q",
			materialized.Insert,
		)
	}
}

func (b *Binder) applyWorkspacePlan(
	ctx context.Context,
	plan *llmExecutionPlan,
	selection sessionv3Runtime.ArtifactSelection,
) error {
	if b.deps.Workspaces == nil {
		return errors.New("session v3 workspace runtime is unavailable")
	}

	workspacePin, err := b.pinSelection(ctx, selection)
	if err != nil {
		return err
	}
	if err := plan.addArtifact(workspacePin); err != nil {
		return err
	}

	workspaceRef := artifactRefForSelection(selection)
	if err := workspaceRef.Validate(); err != nil {
		return err
	}

	capabilities, err := b.deps.Workspaces.ResolveWorkspaceCapabilities(
		ctx,
		workspaceRef,
	)
	if err != nil {
		return err
	}

	promptArtifacts := v3ArtifactRefsForCapabilities(
		capabilities,
		declaration.TypeText,
	)
	skillArtifacts := v3ArtifactRefsForCapabilities(
		capabilities,
		declaration.TypeSkill,
	)
	mcpArtifacts := v3ArtifactRefsForCapabilities(
		capabilities,
		declaration.TypeMCP,
	)

	runtimePlan, err := b.deps.Workspaces.ResolveWorkspaceRuntimePlan(
		ctx,
		workspaceRef,
		workspaceAPI.WorkspaceRuntimeSelection{
			PromptArtifacts: promptArtifacts,
			SkillArtifacts:  skillArtifacts,
			MCPArtifacts:    mcpArtifacts,
		},
	)
	if err != nil {
		return err
	}

	for _, contribution := range runtimePlan.Prompt.Contributions {
		selection, pin, selectionErr := b.selectionForWorkspaceArtifact(
			ctx,
			contribution.Artifact,
			contribution.ArtifactRevision,
			contribution.DefinitionDigest,
		)
		if selectionErr != nil {
			return selectionErr
		}
		insert, insertErr := textDispatchInsert(string(contribution.Insert))
		if insertErr != nil {
			return insertErr
		}
		plan.addText(sessionv3Runtime.TextDispatchPlan{
			Artifact: pin,
			Insert:   insert,
			Content:  "",
		})
		_ = selection
	}

	plan.addInstruction(sessionv3Runtime.Instruction{
		ID:      "workspace:" + selection.RootID + ":" + selection.ArtifactID,
		Label:   selection.ArtifactID,
		Content: runtimePlan.Prompt.Instructions,
	})
	plan.addContext(runtimePlan.Prompt.UserMessage)

	for _, skill := range runtimePlan.Skills.Skills {
		value, pin, selectionErr := b.selectionForWorkspaceArtifact(
			ctx,
			skill.Artifact,
			skill.ArtifactRevision,
			skill.DefinitionDigest,
		)
		if selectionErr != nil {
			return selectionErr
		}
		plan.defaults.Skills = appendUniqueLLMSelection(
			plan.defaults.Skills,
			value,
		)
		if err := plan.addArtifact(pin); err != nil {
			return err
		}
	}

	for _, server := range runtimePlan.MCPServers.Servers {
		value, pin, selectionErr := b.selectionForWorkspaceArtifact(
			ctx,
			server.Artifact,
			server.ArtifactRevision,
			server.DefinitionDigest,
		)
		if selectionErr != nil {
			return selectionErr
		}
		plan.defaults.MCP = appendUniqueMCPSelection(
			plan.defaults.MCP,
			sessionv3Runtime.MCPServerSelection{
				Server:                    value,
				IncludeServerInstructions: true,
				ToolExposure:              sessionv3Runtime.MCPToolExposureAll,
			},
		)
		if err := plan.addArtifact(pin); err != nil {
			return err
		}
	}

	for _, ref := range v3ArtifactRefsForCapabilities(
		capabilities,
		declaration.TypeTool,
	) {
		value, pin, selectionErr := b.selectionForArtifactRef(ctx, ref)
		if selectionErr != nil {
			return selectionErr
		}
		plan.defaults.Tools = appendUniqueLLMSelection(
			plan.defaults.Tools,
			value,
		)
		if err := plan.addArtifact(pin); err != nil {
			return err
		}
	}

	return nil
}

func (b *Binder) selectionForWorkspaceArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	revision uint64,
	digest cryptoutil.Digest,
) (
	sessionv3Runtime.ArtifactSelection,
	sessionv3Runtime.ArtifactPin,
	error,
) {
	selection, pin, err := b.selectionForArtifactRef(ctx, ref)
	if err != nil {
		return sessionv3Runtime.ArtifactSelection{},
			sessionv3Runtime.ArtifactPin{},
			err
	}

	if revision != 0 && revision != pin.Revision {
		return sessionv3Runtime.ArtifactSelection{},
			sessionv3Runtime.ArtifactPin{},
			errors.New(
				"workspace runtime plan artifact revision changed",
			)
	}
	if digest != "" && string(digest) != pin.Digest {
		return sessionv3Runtime.ArtifactSelection{},
			sessionv3Runtime.ArtifactPin{},
			errors.New(
				"workspace runtime plan artifact definition changed",
			)
	}
	return selection, pin, nil
}

func (b *Binder) selectionForCapabilityTarget(
	ctx context.Context,
	target *composition.CapabilityTarget,
) (
	sessionv3Runtime.ArtifactSelection,
	sessionv3Runtime.ArtifactPin,
	error,
) {
	if target == nil ||
		target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil {
		return sessionv3Runtime.ArtifactSelection{},
			sessionv3Runtime.ArtifactPin{},
			errors.New(
				"agent capability target is not artifact-backed",
			)
	}
	return b.selectionForArtifactRef(ctx, *target.Artifact)
}

func (b *Binder) selectionForArtifactRef(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (
	sessionv3Runtime.ArtifactSelection,
	sessionv3Runtime.ArtifactPin,
	error,
) {
	pin, err := b.pinArtifactRef(ctx, ref)
	if err != nil {
		return sessionv3Runtime.ArtifactSelection{},
			sessionv3Runtime.ArtifactPin{},
			err
	}
	return sessionv3Runtime.ArtifactSelection{
		RootID:           pin.RootID,
		ArtifactID:       pin.ArtifactID,
		ExpectedRevision: pin.Revision,
		ExpectedDigest:   pin.Digest,
	}, pin, nil
}

func (b *Binder) pinSelections(
	ctx context.Context,
	selections []sessionv3Runtime.ArtifactSelection,
) ([]sessionv3Runtime.ArtifactSelection, []sessionv3Runtime.ArtifactPin, error) {
	if len(selections) == 0 {
		return nil, nil, nil
	}

	output := make(
		[]sessionv3Runtime.ArtifactSelection,
		0,
		len(selections),
	)
	pins := make(
		[]sessionv3Runtime.ArtifactPin,
		0,
		len(selections),
	)
	for _, selection := range selections {
		pin, err := b.pinSelection(ctx, selection)
		if err != nil {
			return nil, nil, err
		}
		output = appendUniqueLLMSelection(
			output,
			sessionv3Runtime.ArtifactSelection{
				RootID:           pin.RootID,
				ArtifactID:       pin.ArtifactID,
				ExpectedRevision: pin.Revision,
				ExpectedDigest:   pin.Digest,
			},
		)
		pins = append(pins, pin)
	}
	return output, pins, nil
}

func (b *Binder) pinMCPSelections(
	ctx context.Context,
	selections []sessionv3Runtime.MCPServerSelection,
) ([]sessionv3Runtime.MCPServerSelection, []sessionv3Runtime.ArtifactPin, error) {
	if len(selections) == 0 {
		return nil, nil, nil
	}

	output := make(
		[]sessionv3Runtime.MCPServerSelection,
		0,
		len(selections),
	)
	pins := make(
		[]sessionv3Runtime.ArtifactPin,
		0,
		len(selections),
	)
	for _, selection := range selections {
		pin, err := b.pinSelection(ctx, selection.Server)
		if err != nil {
			return nil, nil, err
		}
		value := jsonutil.CloneJSONInValue(selection)
		value.Server = sessionv3Runtime.ArtifactSelection{
			RootID:           pin.RootID,
			ArtifactID:       pin.ArtifactID,
			ExpectedRevision: pin.Revision,
			ExpectedDigest:   pin.Digest,
		}
		output = appendUniqueMCPSelection(output, value)
		pins = append(pins, pin)
	}
	return output, pins, nil
}

func (b *Binder) pinSelection(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
) (sessionv3Runtime.ArtifactPin, error) {
	ref := artifactRefForSelection(selection)
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	pin, err := b.pinArtifactRef(ctx, ref)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != pin.Revision {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected artifact revision changed",
		)
	}
	if selection.ExpectedDigest != "" &&
		selection.ExpectedDigest != pin.Digest {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected artifact definition changed",
		)
	}
	return pin, nil
}

func (b *Binder) pinArtifactRef(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (sessionv3Runtime.ArtifactPin, error) {
	if b == nil || b.deps.Artifacts == nil {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"session v3 artifact runtime is unavailable",
		)
	}
	if err := ref.Validate(); err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}

	values, err := b.deps.Artifacts.GetMany(
		ctx,
		[]artifactModel.ArtifactRef{ref},
	)
	if err != nil {
		return sessionv3Runtime.ArtifactPin{}, err
	}
	if len(values) != 1 || values[0].Ref() != ref {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"artifact runtime returned an unexpected artifact",
		)
	}

	value := values[0]
	if value.State != artifactModel.StateAvailable || !value.Enabled {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected artifact is unavailable",
		)
	}
	if value.ResolvedDefinition == nil ||
		strings.TrimSpace(string(*value.ResolvedDefinition)) == "" {
		return sessionv3Runtime.ArtifactPin{}, errors.New(
			"selected artifact has no resolved definition",
		)
	}

	return sessionv3Runtime.ArtifactPin{
		RootID:     string(value.RootID),
		ArtifactID: string(value.ID),
		Revision:   value.Revision,
		Digest:     string(*value.ResolvedDefinition),
	}, nil
}

func (b *Binder) verifyArtifactPin(
	ctx context.Context,
	pin sessionv3Runtime.ArtifactPin,
) error {
	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(pin.RootID),
		ArtifactID: artifactModel.ArtifactID(pin.ArtifactID),
	}
	current, err := b.pinArtifactRef(ctx, ref)
	if err != nil {
		return err
	}
	if current != pin {
		return errors.New("artifact pin changed since work item binding")
	}
	return nil
}

func (p *llmExecutionPlan) addArtifact(
	pin sessionv3Runtime.ArtifactPin,
) error {
	for _, current := range p.artifacts {
		if current.RootID != pin.RootID ||
			current.ArtifactID != pin.ArtifactID {
			continue
		}
		if current != pin {
			return errors.New(
				"model execution plan contains conflicting artifact pins",
			)
		}
		return nil
	}
	p.artifacts = append(p.artifacts, pin)
	return nil
}

func (p *llmExecutionPlan) addInstruction(
	instruction sessionv3Runtime.Instruction,
) {
	instruction.Content = strings.TrimSpace(instruction.Content)
	if instruction.Content == "" {
		return
	}
	for _, current := range p.systemInstructions {
		if instruction.ID != "" &&
			instruction.ID == current.ID {
			return
		}
		if instruction.Content == current.Content {
			return
		}
	}
	p.systemInstructions = append(
		p.systemInstructions,
		instruction,
	)
}

func (p *llmExecutionPlan) addText(
	value sessionv3Runtime.TextDispatchPlan,
) {
	if p == nil {
		return
	}
	for index := range p.texts {
		current := &p.texts[index]
		if current.Artifact.RootID != value.Artifact.RootID ||
			current.Artifact.ArtifactID != value.Artifact.ArtifactID ||
			current.Insert != value.Insert {
			continue
		}
		if current.Content == "" {
			current.Content = value.Content
		}
		return
	}
	p.texts = append(p.texts, value)
}

func textDispatchInsert(
	value string,
) (sessionv3Runtime.TextDispatchInsert, error) {
	switch value {
	case string(sessionv3Runtime.TextDispatchInsertInstructions):
		return sessionv3Runtime.TextDispatchInsertInstructions, nil
	case string(sessionv3Runtime.TextDispatchInsertUserMessage):
		return sessionv3Runtime.TextDispatchInsertUserMessage, nil
	default:
		return "", fmt.Errorf("unsupported text dispatch insert %q", value)
	}
}

func (p *llmExecutionPlan) addContext(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if slices.Contains(p.context, value) {
		return
	}
	p.context = append(p.context, value)
}

func (p *llmExecutionPlan) systemPrompt() string {
	parts := make([]string, 0, len(p.systemInstructions))
	for _, instruction := range p.systemInstructions {
		if strings.TrimSpace(instruction.Content) != "" {
			parts = append(parts, instruction.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

func cloneLLMArtifactSelection(
	value *sessionv3Runtime.ArtifactSelection,
) *sessionv3Runtime.ArtifactSelection {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}

func appendUniqueLLMSelection(
	values []sessionv3Runtime.ArtifactSelection,
	value sessionv3Runtime.ArtifactSelection,
) []sessionv3Runtime.ArtifactSelection {
	for index := range values {
		if values[index].RootID != value.RootID ||
			values[index].ArtifactID != value.ArtifactID {
			continue
		}
		values[index] = value
		return values
	}
	return append(values, value)
}

func appendUniqueMCPSelection(
	values []sessionv3Runtime.MCPServerSelection,
	value sessionv3Runtime.MCPServerSelection,
) []sessionv3Runtime.MCPServerSelection {
	for index := range values {
		if values[index].Server.RootID != value.Server.RootID ||
			values[index].Server.ArtifactID != value.Server.ArtifactID {
			continue
		}
		values[index] = value
		return values
	}
	return append(values, value)
}

func v3ArtifactRefsForCapabilities(
	plan composition.CapabilityPlan,
	kind declaration.Type,
) []artifactModel.ArtifactRef {
	seen := make(map[artifactModel.ArtifactRef]struct{})
	output := make([]artifactModel.ArtifactRef, 0)

	for _, occurrence := range plan.Occurrences {
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Type != kind ||
			occurrence.Target == nil ||
			occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			continue
		}

		ref := *occurrence.Target.Artifact
		if _, duplicate := seen[ref]; duplicate {
			continue
		}
		seen[ref] = struct{}{}
		output = append(output, ref)
	}
	return output
}
