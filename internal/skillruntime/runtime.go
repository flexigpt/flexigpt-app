package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	skillruntimeSpec "github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
)

var errSkillInvalidRequest = errors.New("invalid request")

func (s *SkillRuntime) CreateSkillSession(
	ctx context.Context,
	req *skillruntimeSpec.CreateSkillSessionRequest,
) (*skillruntimeSpec.CreateSkillSessionResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if len(req.Body.AllowArtifacts) == 0 {
		return nil, fmt.Errorf("%w: allowArtifacts required", errSkillInvalidRequest)
	}
	if err := validateArtifactRefs(req.Body.AllowArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid allowArtifacts: %w", errSkillInvalidRequest, err)
	}
	if err := validateArtifactRefs(req.Body.ActiveArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid activeArtifacts: %w", errSkillInvalidRequest, err)
	}

	if sessionID := strings.TrimSpace(string(req.Body.CloseSessionID)); sessionID != "" {
		_ = s.runtime.CloseSession(ctx, agentskillsSpec.SessionID(sessionID))
	}

	resolved := s.resolveAllowArtifacts(ctx, req.Body.AllowArtifacts)
	activeRefs := subsetArtifacts(
		req.Body.AllowArtifacts,
		req.Body.ActiveArtifacts,
	)
	activeDefinitions := map[agentskillsSpec.SkillDef]struct{}{}
	for _, ref := range activeRefs {
		if definition, found := resolved.RefToDef[artifactRefKey(ref)]; found {
			activeDefinitions[definition] = struct{}{}
		}
	}

	activeDefs := make([]agentskillsSpec.SkillDef, 0, len(activeDefinitions))
	for definition := range activeDefinitions {
		activeDefs = append(activeDefs, definition)
	}
	sortSkillDefs(activeDefs)

	options := []agentskills.SessionOption{}
	if req.Body.MaxActivePerSession > 0 {
		options = append(
			options,
			agentskills.WithSessionMaxActivePerSession(
				req.Body.MaxActivePerSession,
			),
		)
	}
	if len(activeDefs) > 0 {
		options = append(
			options,
			agentskills.WithSessionActiveSkills(activeDefs),
		)
	}

	sessionID, _, err := s.runtime.NewSession(ctx, options...)
	if err != nil {
		return nil, err
	}

	records, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
		SessionID:   sessionID,
		Activity:    agentskillsSpec.SkillActivityActive,
		AllowSkills: resolved.AllowDefs,
	})
	if err != nil {
		closeErr := s.runtime.CloseSession(
			context.WithoutCancel(ctx),
			sessionID,
		)
		return nil, errors.Join(err, closeErr)
	}

	active := map[agentskillsSpec.SkillDef]struct{}{}
	for _, record := range records {
		active[record.Def] = struct{}{}
	}
	return &skillruntimeSpec.CreateSkillSessionResponse{
		Body: &skillruntimeSpec.CreateSkillSessionResponseBody{
			SessionID:       sessionID,
			ActiveArtifacts: buildActiveArtifacts(resolved.DefToArtifacts, active),
		},
	}, nil
}

func (s *SkillRuntime) CloseSkillSession(
	ctx context.Context,
	req *skillruntimeSpec.CloseSkillSessionRequest,
) (*skillruntimeSpec.CloseSkillSessionResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if err := s.runtime.CloseSession(ctx, req.SessionID); err != nil {
		return nil, err
	}
	return &skillruntimeSpec.CloseSkillSessionResponse{}, nil
}

func (s *SkillRuntime) GetSkillsPrompt(
	ctx context.Context,
	req *skillruntimeSpec.GetSkillsPromptRequest,
) (*skillruntimeSpec.GetSkillsPromptResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil, fmt.Errorf("%w: Skill prompt filter is required", errSkillInvalidRequest)
	}

	filterRequest := req.Body.Filter
	if len(filterRequest.AllowArtifacts) == 0 {
		return &skillruntimeSpec.GetSkillsPromptResponse{
			Body: &skillruntimeSpec.GetSkillsPromptResponseBody{},
		}, nil
	}
	if err := validateArtifactRefs(filterRequest.AllowArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid allowArtifacts: %w", errSkillInvalidRequest, err)
	}
	if len(filterRequest.Inserts) > 0 &&
		!containsInstructionInsert(filterRequest.Inserts) {
		return &skillruntimeSpec.GetSkillsPromptResponse{
			Body: &skillruntimeSpec.GetSkillsPromptResponseBody{},
		}, nil
	}

	resolved := s.resolveAllowArtifacts(ctx, filterRequest.AllowArtifacts)
	if len(resolved.AllowDefs) == 0 {
		return &skillruntimeSpec.GetSkillsPromptResponse{
			Body: &skillruntimeSpec.GetSkillsPromptResponseBody{},
		}, nil
	}

	prompt, err := s.runtime.SkillsPrompt(ctx, &agentskills.SkillFilter{
		Types:          append([]string(nil), filterRequest.Types...),
		LocationPrefix: filterRequest.LocationPrefix,
		AllowSkills:    resolved.AllowDefs,
		SessionID:      filterRequest.SessionID,
		Activity:       filterRequest.Activity,
	})
	if err != nil {
		return nil, err
	}
	return &skillruntimeSpec.GetSkillsPromptResponse{
		Body: &skillruntimeSpec.GetSkillsPromptResponseBody{Prompt: prompt},
	}, nil
}

func (s *SkillRuntime) ListRuntimeSkills(
	ctx context.Context,
	req *skillruntimeSpec.ListRuntimeSkillsRequest,
) (*skillruntimeSpec.ListRuntimeSkillsResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil, fmt.Errorf("%w: missing filter", errSkillInvalidRequest)
	}

	filterRequest := req.Body.Filter
	if len(filterRequest.AllowArtifacts) == 0 {
		return &skillruntimeSpec.ListRuntimeSkillsResponse{
			Body: &skillruntimeSpec.ListRuntimeSkillsResponseBody{
				Skills: []skillruntimeSpec.RuntimeSkillListItem{},
			},
		}, nil
	}
	if err := validateArtifactRefs(filterRequest.AllowArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid allowArtifacts: %w", errSkillInvalidRequest, err)
	}

	activity := filterRequest.Activity
	if activity == "" {
		activity = agentskillsSpec.SkillActivityAny
	}
	if activity == agentskillsSpec.SkillActivityActive &&
		strings.TrimSpace(string(filterRequest.SessionID)) == "" {
		return nil, fmt.Errorf(
			"%w: activity=active requires sessionID",
			errSkillInvalidRequest,
		)
	}

	resolved := s.resolveAllowArtifacts(ctx, filterRequest.AllowArtifacts)
	if len(resolved.AllowDefs) == 0 {
		return &skillruntimeSpec.ListRuntimeSkillsResponse{
			Body: &skillruntimeSpec.ListRuntimeSkillsResponseBody{
				Skills: []skillruntimeSpec.RuntimeSkillListItem{},
			},
		}, nil
	}

	records, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
		Types:          append([]string(nil), filterRequest.Types...),
		LocationPrefix: filterRequest.LocationPrefix,
		AllowSkills:    resolved.AllowDefs,
		Inserts:        append([]agentskillsSpec.SkillInsert(nil), filterRequest.Inserts...),
		SessionID:      filterRequest.SessionID,
		Activity:       activity,
	})
	if err != nil {
		return nil, err
	}

	active := map[agentskillsSpec.SkillDef]struct{}{}
	if filterRequest.SessionID != "" &&
		activity == agentskillsSpec.SkillActivityAny {
		current, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
			SessionID:   filterRequest.SessionID,
			Activity:    agentskillsSpec.SkillActivityActive,
			AllowSkills: resolved.AllowDefs,
		})
		if err != nil {
			return nil, err
		}
		for _, record := range current {
			active[record.Def] = struct{}{}
		}
	}

	items := make([]skillruntimeSpec.RuntimeSkillListItem, 0, len(records))
	for _, record := range records {
		ref, found := resolved.DefToArtifacts[record.Def]
		if !found {
			continue
		}
		_, isActive := active[record.Def]
		items = append(items, skillruntimeSpec.RuntimeSkillListItem{
			SkillRef:       ref,
			Type:           record.Def.Type,
			Name:           record.Def.Name,
			DisplayName:    record.DisplayName,
			Description:    record.Description,
			Digest:         record.Digest,
			Insert:         record.Insert,
			Arguments:      append([]agentskillsSpec.SkillArgument(nil), record.Arguments...),
			SourceTags:     append([]string(nil), record.Tags...),
			Resources:      cloneSkillResourceInfo(record.Resources),
			RawFrontmatter: cloneAnyMap(record.RawFrontmatter),
			Warnings:       append([]string(nil), record.Warnings...),
			IsActive: activity == agentskillsSpec.SkillActivityActive ||
				(activity == agentskillsSpec.SkillActivityAny && isActive),
		})
	}
	sort.Slice(items, func(left, right int) bool {
		return artifactRefKey(items[left].SkillRef) <
			artifactRefKey(items[right].SkillRef)
	})
	return &skillruntimeSpec.ListRuntimeSkillsResponse{
		Body: &skillruntimeSpec.ListRuntimeSkillsResponseBody{Skills: items},
	}, nil
}

func (s *SkillRuntime) RenderSkill(
	ctx context.Context,
	req *skillruntimeSpec.RenderSkillRequest,
) (*skillruntimeSpec.RenderSkillResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if err := req.Body.Artifact.Validate(); err != nil {
		return nil, fmt.Errorf("%w: invalid artifact: %w", errSkillInvalidRequest, err)
	}

	resolved, found := s.resolveArtifactSkill(ctx, req.Body.Artifact)
	if !found {
		return nil, skillruntimeSpec.ErrSkillNotFound
	}
	out, err := s.runtime.RenderSkill(ctx, agentskills.RenderSkillParams{
		Def:       resolved.Definition,
		Arguments: req.Body.Arguments,
	})
	if err != nil {
		return nil, err
	}
	return &skillruntimeSpec.RenderSkillResponse{
		Body: &skillruntimeSpec.RenderSkillResponseBody{
			Text:             out.Text,
			Insert:           out.Insert,
			Name:             out.Name,
			Description:      out.Description,
			DisplayName:      out.DisplayName,
			SourceTags:       append([]string(nil), out.Tags...),
			Resources:        cloneSkillResourceInfo(out.Resources),
			Arguments:        append([]agentskillsSpec.SkillArgument(nil), out.Arguments...),
			AppliedArguments: cloneStringMap(out.AppliedArguments),
			RawFrontmatter:   cloneAnyMap(out.RawFrontmatter),
			Warnings:         append([]string(nil), out.Warnings...),
		},
	}, nil
}

type resolvedAllowArtifacts struct {
	DefToArtifacts map[agentskillsSpec.SkillDef]artifact.ArtifactRef
	RefToDef       map[string]agentskillsSpec.SkillDef
	AllowDefs      []agentskillsSpec.SkillDef
}

func (s *SkillRuntime) resolveAllowArtifacts(
	ctx context.Context,
	refs []artifact.ArtifactRef,
) resolvedAllowArtifacts {
	output := resolvedAllowArtifacts{
		DefToArtifacts: map[agentskillsSpec.SkillDef]artifact.ArtifactRef{},
		RefToDef:       map[string]agentskillsSpec.SkillDef{},
	}

	seenRefs := map[string]struct{}{}
	byName := map[string][]ResolvedArtifactSkill{}
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			continue
		}
		key := artifactRefKey(ref)
		if _, exists := seenRefs[key]; exists {
			continue
		}
		seenRefs[key] = struct{}{}

		value, found := s.resolveArtifactSkill(ctx, ref)
		if !found {
			continue
		}
		byName[value.Definition.Name] = append(
			byName[value.Definition.Name],
			value,
		)
	}

	for _, values := range byName {
		if len(values) != 1 {
			continue
		}
		value := values[0]
		output.DefToArtifacts[value.Definition] = value.Artifact
		output.RefToDef[artifactRefKey(value.Artifact)] = value.Definition
		output.AllowDefs = append(output.AllowDefs, value.Definition)
	}
	sortSkillDefs(output.AllowDefs)
	return output
}

func (s *SkillRuntime) resolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedArtifactSkill, bool) {
	value, err := s.resolver.ResolveArtifactSkill(ctx, ref)
	if err != nil {
		return ResolvedArtifactSkill{}, false
	}
	if err := s.ResyncCollection(ctx, value.Collection); err != nil {
		return ResolvedArtifactSkill{}, false
	}
	value, err = s.resolver.ResolveArtifactSkill(ctx, ref)
	if err != nil {
		return ResolvedArtifactSkill{}, false
	}

	s.rtResyncMu.Lock()
	registeredVersion, registered := s.managedRuntime[value.Definition]
	s.rtResyncMu.Unlock()
	if !registered || registeredVersion != value.Version {
		return ResolvedArtifactSkill{}, false
	}
	return value, true
}

func validateArtifactRefs(values []artifact.ArtifactRef) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		key := artifactRefKey(value)
		if _, duplicate := seen[key]; duplicate {
			return errors.New("duplicate ArtifactRef")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func subsetArtifacts(
	allow []artifact.ArtifactRef,
	active []artifact.ArtifactRef,
) []artifact.ArtifactRef {
	allowed := map[string]struct{}{}
	for _, ref := range allow {
		allowed[artifactRefKey(ref)] = struct{}{}
	}

	seen := map[string]struct{}{}
	output := make([]artifact.ArtifactRef, 0, len(active))
	for _, ref := range active {
		key := artifactRefKey(ref)
		if _, exists := allowed[key]; !exists {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, ref)
	}
	return output
}

func buildActiveArtifacts(
	definitions map[agentskillsSpec.SkillDef]artifact.ArtifactRef,
	active map[agentskillsSpec.SkillDef]struct{},
) []artifact.ArtifactRef {
	output := make([]artifact.ArtifactRef, 0)
	for definition := range active {
		ref, found := definitions[definition]
		if found {
			output = append(output, ref)
		}
	}
	sort.Slice(output, func(left, right int) bool {
		return artifactRefKey(output[left]) < artifactRefKey(output[right])
	})
	return output
}

func containsInstructionInsert(
	values []agentskillsSpec.SkillInsert,
) bool {
	return slices.Contains(values, agentskillsSpec.SkillInsertInstructions)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	maps.Copy(output, input)
	return output
}

func cloneAnyMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	maps.Copy(output, input)
	return output
}

func cloneSkillResourceInfo(
	input agentskillsSpec.SkillResourceInfo,
) agentskillsSpec.SkillResourceInfo {
	input.Locations = append([]string(nil), input.Locations...)
	return input
}
