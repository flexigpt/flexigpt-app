package sessionv3runtime

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/contributionruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type llmExecutionPlan struct {
	model                         *sessionv3Runtime.ArtifactSelection
	defaults                      sessionv3Runtime.SessionDefaults
	systemInstructions            []sessionv3Runtime.Instruction
	texts                         []sessionv3Runtime.TextDispatchPlan
	context                       []string
	artifacts                     []sessionv3Runtime.ArtifactPin
	activities                    []sessionv3Runtime.Activity
	agentModelIncludeSystemPrompt *bool
}

func (p *llmExecutionPlan) applySpecificInstructionPolicy(
	policy sessionv3Runtime.CollectionPolicy,
) {
	removed := removedCollectionKeys(policy)
	filtered := p.systemInstructions[:0]
	for _, instruction := range p.systemInstructions {
		if collectionMode(policy) == sessionv3Runtime.CollectionOverrideReplace &&
			(instruction.Owner == sessionv3Runtime.ConfigurationOwnerAgent ||
				instruction.Owner == sessionv3Runtime.ConfigurationOwnerWorkspace) {
			continue
		}
		if _, removedValue := removed[instruction.SourceKey]; removedValue {
			continue
		}
		filtered = append(filtered, instruction)
	}
	p.systemInstructions = filtered
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

	specificCollections := jsonutil.CloneJSONInValue(defaults)
	defaults.SystemInstructions = nil
	defaults.InstructionArtifacts = nil
	defaults.ToolSelections = nil
	defaults.Skills = nil
	defaults.ActiveSkills = nil
	defaults.MCP = nil

	plan := llmExecutionPlan{
		model:    cloneLLMArtifactSelection(defaults.Model),
		defaults: defaults,
	}
	workflowInstructions := append(
		[]sessionv3Runtime.Instruction(nil),
		payload.SystemInstructions...,
	)

	normalizedMCP, err := normalizeMCPSelections(
		specificCollections.MCP,
	)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	specificCollections.MCP = normalizedMCP

	if payload.PreparedInvocation != nil {
		if err := b.applyPreparedContributionPlan(
			ctx,
			&plan,
			*payload.PreparedInvocation,
		); err != nil {
			return llmExecutionPlan{}, err
		}
	} else if defaults.Agent != nil || defaults.Workspace != nil {
		if b.contributions == nil {
			return llmExecutionPlan{}, errors.New(
				"session v3 contribution runtime is unavailable",
			)
		}
		prepared, prepareErr := b.contributions.Prepare(
			ctx,
			contributionruntime.Request{
				Agent:              cloneLLMArtifactSelection(defaults.Agent),
				Workspace:          cloneLLMArtifactSelection(defaults.Workspace),
				WorkspaceOverrides: defaults.WorkspaceOverrides,
			},
		)
		if prepareErr != nil {
			return llmExecutionPlan{}, prepareErr
		}
		if err := b.applyPreparedContributionPlan(ctx, &plan, prepared); err != nil {
			return llmExecutionPlan{}, err
		}
	}
	applySpecificCollections(&plan.defaults, specificCollections)
	plan.applySpecificInstructionPolicy(
		specificCollections.CollectionPolicies.SystemInstructions,
	)
	for _, instruction := range plan.defaults.SystemInstructions {
		instruction.Owner = sessionv3Runtime.ConfigurationOwnerSession
		instruction.SourceKey = instructionKey(instruction)
		plan.addInstruction(instruction)
	}
	plan.defaults.MCP, err = normalizeMCPSelections(plan.defaults.MCP)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	for _, instruction := range workflowInstructions {
		plan.addInstruction(instruction)
	}

	if plan.model == nil {
		return llmExecutionPlan{}, errors.New("no effective model selection is available")
	}
	plan.defaults.Model = cloneLLMArtifactSelection(plan.model)

	var pins []sessionv3Runtime.ArtifactPin

	plan.defaults.InstructionArtifacts, pins, err = b.pinSelections(
		ctx,
		plan.defaults.InstructionArtifacts,
	)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	for _, pin := range pins {
		if err := plan.addArtifact(pin); err != nil {
			return llmExecutionPlan{}, err
		}
	}

	toolSelections, pins, err := b.pinToolSelections(
		ctx,
		sessionv3Runtime.EffectiveToolSelections(plan.defaults),
	)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	for _, pin := range pins {
		if err := plan.addArtifact(pin); err != nil {
			return llmExecutionPlan{}, err
		}
	}
	plan.defaults.ToolSelections = toolSelections

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
	if len(plan.defaults.Skills) != 0 {
		if b.skillTools == nil {
			return llmExecutionPlan{}, errors.New(
				"session v3 Skill runtime is unavailable",
			)
		}
		active, activeErr := b.skillTools.EffectiveActiveSelectionsForFrame(
			request.Execution,
			request.WorkItem.FrameID,
			plan.defaults.Skills,
			plan.defaults.ActiveSkills,
		)
		if activeErr != nil {
			return llmExecutionPlan{}, activeErr
		}
		plan.defaults.ActiveSkills = active
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

	plan.defaults.MCP, err = normalizeMCPSelections(plan.defaults.MCP)
	if err != nil {
		return llmExecutionPlan{}, err
	}

	if err := b.applyInstructionArtifacts(ctx, &plan); err != nil {
		return llmExecutionPlan{}, err
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

	activities, err := llmInputActivitySequence(
		request.Session,
		request.Execution,
		request.WorkItem,
		payload,
		plan.defaults.History,
	)
	if err != nil {
		return llmExecutionPlan{}, err
	}
	plan.activities = append(
		[]sessionv3Runtime.Activity(nil),
		activities...,
	)

	plan.defaults.SystemInstructions = append(
		[]sessionv3Runtime.Instruction(nil),
		plan.systemInstructions...,
	)
	return plan, nil
}

type preparedLLMTextContribution struct {
	Dispatch sessionv3Runtime.TextDispatchPlan
	Label    string
}

func (b *Binder) prepareLLMTextContribution(
	ctx context.Context,
	selection sessionv3Runtime.ArtifactSelection,
	pin sessionv3Runtime.ArtifactPin,
) (preparedLLMTextContribution, error) {
	if b.deps.Texts == nil {
		return preparedLLMTextContribution{}, errors.New(
			"session v3 text runtime is unavailable",
		)
	}

	ref := artifactRefForSelection(selection)
	if err := ref.Validate(); err != nil {
		return preparedLLMTextContribution{}, err
	}

	materialized, err := b.deps.Texts.Materialize(ctx, ref)
	if err != nil {
		return preparedLLMTextContribution{}, err
	}
	if materialized.Artifact != ref {
		return preparedLLMTextContribution{}, errors.New(
			"agent text materialization returned another artifact",
		)
	}
	if materialized.ArtifactRevision != pin.Revision ||
		string(materialized.DefinitionDigest) != pin.Digest {
		return preparedLLMTextContribution{}, errors.New(
			"agent text materialization differs from its resolved artifact pin",
		)
	}

	insert, err := textDispatchInsert(string(materialized.Insert))
	if err != nil {
		return preparedLLMTextContribution{}, err
	}
	return preparedLLMTextContribution{
		Dispatch: sessionv3Runtime.TextDispatchPlan{
			Artifact: pin,
			Insert:   insert,
			Content:  materialized.Content,
		},
		Label: string(materialized.Name),
	}, nil
}

func (b *Binder) applyInstructionArtifacts(
	ctx context.Context,
	plan *llmExecutionPlan,
) error {
	if plan == nil {
		return errors.New("llm execution plan is unavailable")
	}

	for _, selection := range plan.defaults.InstructionArtifacts {
		pin, found := pinForSelection(plan.artifacts, selection)
		if !found {
			return fmt.Errorf(
				"instruction artifact %q has no durable pin",
				selection.ArtifactID,
			)
		}
		contribution, err := b.prepareLLMTextContribution(
			ctx,
			selection,
			pin,
		)
		if err != nil {
			return err
		}
		if contribution.Dispatch.Insert !=
			sessionv3Runtime.TextDispatchInsertInstructions {
			return fmt.Errorf(
				"instruction artifact %q does not insert instructions",
				selection.ArtifactID,
			)
		}

		plan.addText(contribution.Dispatch)
		plan.addInstruction(sessionv3Runtime.Instruction{
			ID: "session-text:" +
				selection.RootID + ":" +
				selection.ArtifactID,
			Label:     contribution.Label,
			Content:   contribution.Dispatch.Content,
			Owner:     sessionv3Runtime.ConfigurationOwnerSession,
			SourceKey: selection.RootID + "\x00" + selection.ArtifactID,
		})
	}
	return nil
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

func (b *Binder) pinToolSelections(
	ctx context.Context,
	selections []sessionv3Runtime.ToolSelection,
) ([]sessionv3Runtime.ToolSelection, []sessionv3Runtime.ArtifactPin, error) {
	if len(selections) == 0 {
		return nil, nil, nil
	}

	output := make(
		[]sessionv3Runtime.ToolSelection,
		0,
		len(selections),
	)
	pins := make(
		[]sessionv3Runtime.ArtifactPin,
		0,
		len(selections),
	)
	seen := make(map[string]struct{}, len(selections))
	for _, selection := range selections {
		if err := selection.Validate(); err != nil {
			return nil, nil, err
		}
		pin, err := b.pinSelection(ctx, selection.Artifact)
		if err != nil {
			return nil, nil, err
		}
		key := pin.RootID + "\x00" + pin.ArtifactID
		if _, duplicate := seen[key]; duplicate {
			return nil, nil, errors.New("tool selection is duplicated")
		}
		seen[key] = struct{}{}
		output = append(output, sessionv3Runtime.ToolSelection{
			Artifact: sessionv3Runtime.ArtifactSelection{
				RootID:           pin.RootID,
				ArtifactID:       pin.ArtifactID,
				ExpectedRevision: pin.Revision,
				ExpectedDigest:   pin.Digest,
			},
			UserArguments: selection.UserArguments.Clone(),
			AutoExecute:   cloneBool(selection.AutoExecute),
		})
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
	if instruction.Owner == "" {
		instruction.Owner = sessionv3Runtime.ConfigurationOwnerSession
	}
	instruction.SourceKey = firstText(instruction.SourceKey, instructionKey(instruction))
	instruction.Content = strings.TrimSpace(instruction.Content)
	if instruction.Content == "" {
		return
	}
	for _, current := range p.systemInstructions {
		if instruction.ID != "" &&
			instruction.ID == current.ID {
			return
		}
		if instruction.SourceKey == current.SourceKey {
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

func appendUniqueLLMToolSelection(
	values []sessionv3Runtime.ToolSelection,
	value sessionv3Runtime.ToolSelection,
) []sessionv3Runtime.ToolSelection {
	for index := range values {
		current := values[index]
		if current.Artifact.RootID != value.Artifact.RootID ||
			current.Artifact.ArtifactID != value.Artifact.ArtifactID {
			continue
		}

		// Earlier effective values are more specific. A Session or invocation
		// selection must not be overwritten by an inherited Agent or Workspace
		// contribution.
		return values
	}
	return append(values, sessionv3Runtime.CloneToolSelection(value))
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

		// Earlier effective values are more specific. Invocation and Session
		// selections must not be replaced by later Agent or Workspace defaults.
		return values
	}
	return append(values, value)
}
