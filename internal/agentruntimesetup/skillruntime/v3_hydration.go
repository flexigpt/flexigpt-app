package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/toolruntime"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

type V3LLMSkillHydration struct {
	Prompt  string
	Choices []toolruntime.V3PreparedChoice
}

type v3ResolvedSkillSelections struct {
	definitions  []provider.SkillDef
	active       []provider.SkillDef
	state        sessionv3Runtime.SkillSelectionState
	byDefinition map[provider.SkillDef]sessionv3Runtime.ArtifactPin
}

// EffectiveActiveSelections reconstructs the latest semantic active-Skill
// selection for one enabled Skill set. It uses only completed durable Tool
// output. It never reads a private Skill Runtime session or current Session
// defaults from a generated Tool origin.
func (e *V3Executor) EffectiveActiveSelections(
	execution sessionv3Runtime.Execution,
	enabled []sessionv3Runtime.ArtifactSelection,
	active []sessionv3Runtime.ArtifactSelection,
) ([]sessionv3Runtime.ArtifactSelection, error) {
	return e.effectiveActiveSelections(
		execution,
		"",
		enabled,
		active,
	)
}

func (e *V3Executor) effectiveActiveSelections(
	execution sessionv3Runtime.Execution,
	frameID sessionv3Runtime.ProgramFrameID,
	enabled []sessionv3Runtime.ArtifactSelection,
	active []sessionv3Runtime.ArtifactSelection,
) ([]sessionv3Runtime.ArtifactSelection, error) {
	output := append([]sessionv3Runtime.ArtifactSelection(nil), active...)
	if len(enabled) == 0 {
		return nil, nil
	}

	var latest *sessionv3Runtime.WorkItem
	for index := range execution.WorkItems {
		item := execution.WorkItems[index]
		if item.Type != sessionv3Runtime.WorkItemTypeTool ||
			item.Input.Tool == nil ||
			item.Input.Tool.Call.Binding.Kind != sessionv3Runtime.ToolBindingKindSkill ||
			item.Output == nil ||
			item.Output.Tool == nil ||
			item.Output.Tool.SkillState == nil {
			continue
		}
		if frameID != "" && item.FrameID != frameID {
			continue
		}
		switch item.State {
		case sessionv3Runtime.WorkItemStateCompleted,
			sessionv3Runtime.WorkItemStateFailed:
		default:
			continue
		}
		if item.EffectState == sessionv3Runtime.EffectStateUncertain {
			continue
		}
		if latest == nil ||
			latest.CompletedAt == nil ||
			(item.CompletedAt != nil &&
				(item.CompletedAt.After(*latest.CompletedAt) ||
					(item.CompletedAt.Equal(*latest.CompletedAt) &&
						item.ID > latest.ID))) {
			copyValue := item
			latest = &copyValue
		}
	}
	if latest == nil {
		return output, nil
	}

	state := *latest.Output.Tool.SkillState
	if err := state.Validate(); err != nil {
		return nil, err
	}
	if !v3SkillStateMatchesEnabledSelections(state, enabled) {
		return output, nil
	}
	return v3SkillActiveSelections(state, enabled)
}

// HydrateLLM creates or reuses the process-private Skill Runtime session for
// the exact durable selection, then loads the Skill rules and SkillsPrompt
// before a provider model invocation.
func (e *V3Executor) HydrateLLM(
	ctx context.Context,
	executionID sessionv3Runtime.ExecutionID,
	frameID sessionv3Runtime.ProgramFrameID,
	enabled []sessionv3Runtime.ArtifactSelection,
	active []sessionv3Runtime.ArtifactSelection,
) (V3LLMSkillHydration, error) {
	if len(enabled) == 0 {
		return V3LLMSkillHydration{}, nil
	}
	if e == nil || e.runtime == nil {
		return V3LLMSkillHydration{}, errors.New(
			"session v3 Skill runtime is unavailable",
		)
	}

	resolved, err := e.resolveV3Selections(ctx, enabled, active)
	if err != nil {
		return V3LLMSkillHydration{}, err
	}
	key := V3SkillSessionKey{
		ExecutionID: executionID,
		FrameID:     frameID,
	}
	runtimeSession, err := e.ensureResolvedSkillSession(
		ctx,
		key,
		resolved,
	)
	if err != nil {
		return V3LLMSkillHydration{}, err
	}

	hydrated, err := e.runtime.HydrateSkillSession(
		ctx,
		agentskillsRuntimeSpec.SessionID(runtimeSession.id),
		runtimeSession.definitions,
	)
	if err != nil {
		return V3LLMSkillHydration{}, err
	}

	actual, err := v3SkillStateFromDefinitions(
		runtimeSession,
		hydrated.ActiveSkills,
	)
	if err != nil {
		return V3LLMSkillHydration{}, err
	}
	if !sameV3SkillState(actual, resolved.state) {
		return V3LLMSkillHydration{}, errors.New(
			"private Skill Runtime active selection differs from durable Skill state",
		)
	}
	e.storeV3SkillSession(key, runtimeSession, actual)

	choices, err := v3SkillChoices(
		len(actual.Active) != 0,
		e.runtime.SupportsRunScript(),
	)
	if err != nil {
		return V3LLMSkillHydration{}, err
	}

	parts := []string{
		SkillRulesPrompt(
			len(actual.Active) != 0,
			e.runtime.SupportsRunScript(),
		),
		strings.TrimSpace(hydrated.Prompt),
	}
	prompt := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			prompt = append(prompt, part)
		}
	}

	return V3LLMSkillHydration{
		Prompt:  strings.Join(prompt, "\n\n"),
		Choices: choices,
	}, nil
}

func (e *V3Executor) resolveV3Selections(
	ctx context.Context,
	selections []sessionv3Runtime.ArtifactSelection,
	activeSelections []sessionv3Runtime.ArtifactSelection,
) (v3ResolvedSkillSelections, error) {
	if e == nil || e.resolver == nil {
		return v3ResolvedSkillSelections{}, errors.New(
			"session v3 Skill resolver is unavailable",
		)
	}
	if len(selections) == 0 {
		return v3ResolvedSkillSelections{}, errors.New(
			"enabled Skill selection is required",
		)
	}

	refs := make([]artifactModel.ArtifactRef, 0, len(selections))
	byKey := make(
		map[string]sessionv3Runtime.ArtifactSelection,
		len(selections),
	)
	for _, selection := range selections {
		ref := artifactModel.ArtifactRef{
			RootID:     rootModel.RootID(selection.RootID),
			ArtifactID: artifactModel.ArtifactID(selection.ArtifactID),
		}
		if err := ref.Validate(); err != nil {
			return v3ResolvedSkillSelections{}, err
		}
		key := v3SkillArtifactKey(selection.RootID, selection.ArtifactID)
		if _, duplicate := byKey[key]; duplicate {
			return v3ResolvedSkillSelections{}, fmt.Errorf(
				"duplicate selected Skill %s/%s",
				selection.RootID,
				selection.ArtifactID,
			)
		}
		byKey[key] = selection
		refs = append(refs, ref)
	}

	activeByKey := make(
		map[string]sessionv3Runtime.ArtifactSelection,
		len(activeSelections),
	)
	for _, selection := range activeSelections {
		key := v3SkillArtifactKey(selection.RootID, selection.ArtifactID)
		if _, duplicate := activeByKey[key]; duplicate {
			return v3ResolvedSkillSelections{}, fmt.Errorf(
				"duplicate active Skill %s/%s",
				selection.RootID,
				selection.ArtifactID,
			)
		}
		if _, enabled := byKey[key]; !enabled {
			return v3ResolvedSkillSelections{}, errors.New(
				"active Skill is not part of the enabled Skill selection",
			)
		}
		activeByKey[key] = selection
	}

	values, err := e.resolver.ResolveSkills(ctx, refs)
	if err != nil {
		return v3ResolvedSkillSelections{}, err
	}
	if len(values.Values) != len(refs) {
		return v3ResolvedSkillSelections{}, errors.New(
			"skill resolver returned an unexpected selected Skill count",
		)
	}

	output := v3ResolvedSkillSelections{
		definitions: make([]provider.SkillDef, 0, len(values.Values)),
		state: sessionv3Runtime.SkillSelectionState{
			Enabled: make([]sessionv3Runtime.ArtifactPin, 0, len(values.Values)),
		},
		byDefinition: make(
			map[provider.SkillDef]sessionv3Runtime.ArtifactPin,
			len(values.Values),
		),
	}
	pinsByKey := make(
		map[string]sessionv3Runtime.ArtifactPin,
		len(values.Values),
	)
	definitionsByKey := make(
		map[string]provider.SkillDef,
		len(values.Values),
	)

	for _, value := range values.Values {
		key := v3SkillArtifactKey(
			string(value.Artifact.RootID),
			string(value.Artifact.ArtifactID),
		)
		selection, found := byKey[key]
		if !found {
			return v3ResolvedSkillSelections{}, errors.New(
				"skill resolver returned an Artifact that was not requested",
			)
		}
		if selection.ExpectedRevision != 0 &&
			selection.ExpectedRevision != value.Revision {
			return v3ResolvedSkillSelections{}, errors.New(
				"selected Skill revision changed",
			)
		}
		if selection.ExpectedDigest != "" &&
			selection.ExpectedDigest != value.DefinitionDigest {
			return v3ResolvedSkillSelections{}, errors.New(
				"selected Skill Definition changed",
			)
		}
		if !value.Enabled {
			return v3ResolvedSkillSelections{}, fmt.Errorf(
				"selected Skill %q is disabled",
				value.Artifact.ArtifactID,
			)
		}

		pin := sessionv3Runtime.ArtifactPin{
			RootID:     string(value.Artifact.RootID),
			ArtifactID: string(value.Artifact.ArtifactID),
			Revision:   value.Revision,
			Digest:     value.DefinitionDigest,
		}

		output.definitions = append(output.definitions, value.Definition)
		output.state.Enabled = append(output.state.Enabled, pin)
		output.byDefinition[value.Definition] = pin
		pinsByKey[key] = pin
		definitionsByKey[key] = value.Definition
	}

	for _, selection := range activeSelections {
		key := v3SkillArtifactKey(selection.RootID, selection.ArtifactID)
		pin, found := pinsByKey[key]
		if !found {
			return v3ResolvedSkillSelections{}, errors.New(
				"active Skill is not part of the enabled Skill selection",
			)
		}
		if selection.ExpectedRevision != 0 &&
			selection.ExpectedRevision != pin.Revision {
			return v3ResolvedSkillSelections{}, errors.New(
				"selected active Skill revision changed",
			)
		}
		if selection.ExpectedDigest != "" &&
			selection.ExpectedDigest != pin.Digest {
			return v3ResolvedSkillSelections{}, errors.New(
				"selected active Skill Definition changed",
			)
		}
		output.active = append(output.active, definitionsByKey[key])
		output.state.Active = append(output.state.Active, pin)
	}
	if err := output.state.Validate(); err != nil {
		return v3ResolvedSkillSelections{}, err
	}
	return output, nil
}

func (e *V3Executor) ensureResolvedSkillSession(
	ctx context.Context,
	key V3SkillSessionKey,
	resolved v3ResolvedSkillSelections,
) (v3SkillSession, error) {
	if key.ExecutionID == "" || key.FrameID == "" {
		return v3SkillSession{}, errors.New(
			"skill runtime session requires an invocation identity",
		)
	}
	if current := e.executionSession(key); current.id != "" &&
		sameV3SkillState(current.state, resolved.state) {
		return current, nil
	}

	created, err := e.runtime.CreateSkillSession(
		ctx,
		&skillRuntime.CreateSkillSessionRequest{
			Body: &skillRuntime.CreateSkillSessionRequestBody{
				AllowedSkills: append([]provider.SkillDef(nil), resolved.definitions...),
				ActiveSkills:  append([]provider.SkillDef(nil), resolved.active...),
			},
		},
	)
	if err != nil {
		return v3SkillSession{}, err
	}
	if created == nil ||
		created.Body == nil ||
		created.Body.SessionID == "" {
		return v3SkillSession{}, errors.New(
			"skill runtime returned no Session identity",
		)
	}
	if !sameV3SkillDefinitions(
		created.Body.ActiveSkills,
		resolved.active,
	) {
		_ = e.runtime.CloseSession(
			context.WithoutCancel(ctx),
			created.Body.SessionID,
		)
		return v3SkillSession{}, errors.New(
			"skill runtime returned an unexpected active Skill selection",
		)
	}

	next := v3SkillSession{
		id:           string(created.Body.SessionID),
		definitions:  append([]provider.SkillDef(nil), resolved.definitions...),
		state:        cloneV3SkillState(resolved.state),
		byDefinition: maps.Clone(resolved.byDefinition),
	}

	e.mu.Lock()
	current, found := e.sessions[key]
	if found && sameV3SkillState(current.state, resolved.state) {
		e.mu.Unlock()
		_ = e.runtime.CloseSession(
			context.WithoutCancel(ctx),
			created.Body.SessionID,
		)
		return cloneV3SkillSession(current), nil
	}
	e.sessions[key] = next
	e.mu.Unlock()

	if found && current.id != "" {
		_ = e.runtime.CloseSession(
			context.WithoutCancel(ctx),
			agentskillsRuntimeSpec.SessionID(current.id),
		)
	}
	return cloneV3SkillSession(next), nil
}

func (e *V3Executor) activeSkillSelectionState(
	ctx context.Context,
	key V3SkillSessionKey,
	session v3SkillSession,
) (sessionv3Runtime.SkillSelectionState, error) {
	records, err := e.runtime.ListAgentSkills(
		ctx,
		&agentskillsRuntime.SkillListFilter{
			AllowSkills: append([]provider.SkillDef(nil), session.definitions...),
			SessionID:   agentskillsRuntimeSpec.SessionID(session.id),
			Activity:    agentskillsRuntimeSpec.SkillActivity("active"),
		},
	)
	if err != nil {
		return sessionv3Runtime.SkillSelectionState{}, err
	}

	active := make([]provider.SkillDef, 0, len(records))
	for _, record := range records {
		active = append(active, record.Def)
	}
	state, err := v3SkillStateFromDefinitions(session, active)
	if err != nil {
		return sessionv3Runtime.SkillSelectionState{}, err
	}
	e.storeV3SkillSession(key, session, state)
	return state, nil
}

func (e *V3Executor) storeV3SkillSession(
	key V3SkillSessionKey,
	session v3SkillSession,
	state sessionv3Runtime.SkillSelectionState,
) {
	if e == nil || key.ExecutionID == "" || key.FrameID == "" {
		return
	}
	e.mu.Lock()
	current, found := e.sessions[key]
	if found && current.id == session.id {
		current.state = cloneV3SkillState(state)
		e.sessions[key] = current
	}
	e.mu.Unlock()
}

func v3SkillStateFromDefinitions(
	session v3SkillSession,
	active []provider.SkillDef,
) (sessionv3Runtime.SkillSelectionState, error) {
	activeByKey := make(map[string]struct{}, len(active))
	for _, definition := range active {
		pin, found := session.byDefinition[definition]
		if !found {
			return sessionv3Runtime.SkillSelectionState{}, errors.New(
				"skill runtime returned an active Skill outside its durable allow-list",
			)
		}
		activeByKey[pin.RootID+"\x00"+pin.ArtifactID] = struct{}{}
	}

	state := cloneV3SkillState(session.state)
	state.Active = nil
	for _, pin := range state.Enabled {
		if _, activeNow := activeByKey[pin.RootID+"\x00"+pin.ArtifactID]; activeNow {
			state.Active = append(state.Active, pin)
		}
	}
	if err := state.Validate(); err != nil {
		return sessionv3Runtime.SkillSelectionState{}, err
	}
	return state, nil
}

func v3SkillStateMatchesEnabledSelections(
	state sessionv3Runtime.SkillSelectionState,
	selections []sessionv3Runtime.ArtifactSelection,
) bool {
	if len(state.Enabled) != len(selections) {
		return false
	}
	expected := make(
		map[string]sessionv3Runtime.ArtifactPin,
		len(selections),
	)
	for _, selection := range selections {
		pin, found := v3SkillSelectionPin(selection)
		if !found {
			return false
		}
		expected[pin.RootID+"\x00"+pin.ArtifactID] = pin
	}
	for _, pin := range state.Enabled {
		current, found := expected[pin.RootID+"\x00"+pin.ArtifactID]
		if !found || current != pin {
			return false
		}
	}
	return true
}

func v3SkillActiveSelections(
	state sessionv3Runtime.SkillSelectionState,
	enabled []sessionv3Runtime.ArtifactSelection,
) ([]sessionv3Runtime.ArtifactSelection, error) {
	byKey := make(
		map[string]sessionv3Runtime.ArtifactSelection,
		len(enabled),
	)
	for _, selection := range enabled {
		byKey[v3SkillArtifactKey(selection.RootID, selection.ArtifactID)] = selection
	}

	output := make([]sessionv3Runtime.ArtifactSelection, 0, len(state.Active))
	for _, pin := range state.Active {
		key := v3SkillArtifactKey(pin.RootID, pin.ArtifactID)
		selection, found := byKey[key]
		if !found {
			return nil, errors.New(
				"durable active Skill state contains an unavailable Skill",
			)
		}
		expected, found := v3SkillSelectionPin(selection)
		if !found || expected != pin {
			return nil, errors.New(
				"durable active Skill state differs from the current enabled Skill pin",
			)
		}
		output = append(output, selection)
	}
	return output, nil
}

func v3SkillSelectionPin(
	selection sessionv3Runtime.ArtifactSelection,
) (sessionv3Runtime.ArtifactPin, bool) {
	if selection.ExpectedRevision == 0 ||
		strings.TrimSpace(selection.ExpectedDigest) == "" {
		return sessionv3Runtime.ArtifactPin{}, false
	}
	return sessionv3Runtime.ArtifactPin{
		RootID:     selection.RootID,
		ArtifactID: selection.ArtifactID,
		Revision:   selection.ExpectedRevision,
		Digest:     selection.ExpectedDigest,
	}, true
}

func sameV3SkillDefinitions(
	left []provider.SkillDef,
	right []provider.SkillDef,
) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[provider.SkillDef]struct{}, len(left))
	for _, value := range left {
		seen[value] = struct{}{}
	}
	for _, value := range right {
		if _, found := seen[value]; !found {
			return false
		}
	}
	return true
}

func sameV3SkillState(
	left sessionv3Runtime.SkillSelectionState,
	right sessionv3Runtime.SkillSelectionState,
) bool {
	if len(left.Enabled) != len(right.Enabled) ||
		len(left.Active) != len(right.Active) {
		return false
	}
	for index := range left.Enabled {
		if left.Enabled[index] != right.Enabled[index] {
			return false
		}
	}
	for index := range left.Active {
		if left.Active[index] != right.Active[index] {
			return false
		}
	}
	return true
}

func cloneV3SkillState(
	value sessionv3Runtime.SkillSelectionState,
) sessionv3Runtime.SkillSelectionState {
	return sessionv3Runtime.SkillSelectionState{
		Enabled: append([]sessionv3Runtime.ArtifactPin(nil), value.Enabled...),
		Active:  append([]sessionv3Runtime.ArtifactPin(nil), value.Active...),
	}
}

func cloneV3SkillSession(value v3SkillSession) v3SkillSession {
	return v3SkillSession{
		id:           value.id,
		definitions:  append([]provider.SkillDef(nil), value.definitions...),
		state:        cloneV3SkillState(value.state),
		byDefinition: maps.Clone(value.byDefinition),
	}
}
