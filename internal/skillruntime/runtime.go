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
	"github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
)

var errSkillInvalidRequest = errors.New("invalid request")

func (s *SkillRuntime) CreateSkillSession(
	ctx context.Context,
	req *spec.CreateSkillSessionRequest,
) (*spec.CreateSkillSessionResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if len(req.Body.AllowSkillRefs) == 0 {
		return nil, fmt.Errorf("%w: allowSkillRefs required", errSkillInvalidRequest)
	}
	for _, ref := range req.Body.AllowSkillRefs {
		if err := validateSkillRef(ref); err != nil {
			return nil, fmt.Errorf("%w: invalid allowSkillRef: %w", errSkillInvalidRequest, err)
		}
	}
	for _, ref := range req.Body.ActiveSkillRefs {
		if err := validateSkillRef(ref); err != nil {
			return nil, fmt.Errorf("%w: invalid activeSkillRef: %w", errSkillInvalidRequest, err)
		}
	}
	if sessionID := strings.TrimSpace(string(req.Body.CloseSessionID)); sessionID != "" {
		_ = s.runtime.CloseSession(ctx, agentskillsSpec.SessionID(sessionID))
	}

	activeRefs := normalizeActiveRefsSubsetOfAllow(req.Body.AllowSkillRefs, req.Body.ActiveSkillRefs)
	resolved := s.resolveAllowSkillRefs(ctx, req.Body.AllowSkillRefs)
	if len(resolved.AllowDefs) == 0 {
		options := []agentskills.SessionOption{}
		if req.Body.MaxActivePerSession > 0 {
			options = append(options, agentskills.WithSessionMaxActivePerSession(req.Body.MaxActivePerSession))
		}
		sessionID, _, err := s.runtime.NewSession(ctx, options...)
		if err != nil {
			return nil, err
		}
		return &spec.CreateSkillSessionResponse{Body: &spec.CreateSkillSessionResponseBody{
			SessionID:       sessionID,
			ActiveSkillRefs: []spec.SkillRef{},
		}}, nil
	}

	activeDefinitions := map[agentskillsSpec.SkillDef]struct{}{}
	for _, ref := range activeRefs {
		if definition, ok := resolved.RefToDef[refKey(ref)]; ok {
			activeDefinitions[definition] = struct{}{}
		}
	}
	activeDefs := make([]agentskillsSpec.SkillDef, 0, len(activeDefinitions))
	for definition := range activeDefinitions {
		activeDefs = append(activeDefs, definition)
	}
	sortSkillDefs(activeDefs)

	if len(activeDefs) > 0 {
		records, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
			AllowSkills: resolved.AllowDefs,
			Activity:    agentskillsSpec.SkillActivityAny,
		})
		if err != nil {
			activeDefs = nil
		} else {
			knownInstruction := map[agentskillsSpec.SkillDef]struct{}{}
			for _, record := range records {
				insert := record.Insert
				if insert == "" {
					insert = agentskillsSpec.SkillInsertInstructions
				}
				if insert == agentskillsSpec.SkillInsertInstructions {
					knownInstruction[record.Def] = struct{}{}
				}
			}
			filtered := make([]agentskillsSpec.SkillDef, 0, len(activeDefs))
			for _, definition := range activeDefs {
				if _, ok := knownInstruction[definition]; ok {
					filtered = append(filtered, definition)
				}
			}
			activeDefs = filtered
		}
	}

	options := []agentskills.SessionOption{}
	if req.Body.MaxActivePerSession > 0 {
		options = append(options, agentskills.WithSessionMaxActivePerSession(req.Body.MaxActivePerSession))
	}
	if len(activeDefs) > 0 {
		options = append(options, agentskills.WithSessionActiveSkills(activeDefs))
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
		if errors.Is(err, agentskillsSpec.ErrSessionNotFound) {
			return nil, err
		}
		records = nil
	}
	active := map[agentskillsSpec.SkillDef]struct{}{}
	for _, record := range records {
		active[record.Def] = struct{}{}
	}
	output := buildActiveSkillRefs(resolved.DefToRefs, active)
	if output == nil {
		output = []spec.SkillRef{}
	}
	return &spec.CreateSkillSessionResponse{Body: &spec.CreateSkillSessionResponseBody{
		SessionID:       sessionID,
		ActiveSkillRefs: output,
	}}, nil
}

func (s *SkillRuntime) CloseSkillSession(
	ctx context.Context,
	req *spec.CloseSkillSessionRequest,
) (*spec.CloseSkillSessionResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if err := s.runtime.CloseSession(ctx, req.SessionID); err != nil {
		return nil, err
	}
	return &spec.CloseSkillSessionResponse{}, nil
}

func (s *SkillRuntime) GetSkillsPrompt(
	ctx context.Context,
	req *spec.GetSkillsPromptRequest,
) (*spec.GetSkillsPromptResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	var filter *agentskills.SkillFilter
	if req != nil && req.Body != nil && req.Body.Filter != nil {
		value := req.Body.Filter
		var allowed []agentskillsSpec.SkillDef
		if len(value.AllowSkillRefs) > 0 {
			for _, ref := range value.AllowSkillRefs {
				if err := validateSkillRef(ref); err != nil {
					return nil, fmt.Errorf(
						"%w: invalid filter.allowSkillRefs: %w",
						errSkillInvalidRequest,
						err,
					)
				}
			}
			resolved := s.resolveAllowSkillRefs(ctx, value.AllowSkillRefs)
			if len(resolved.AllowDefs) == 0 {
				return &spec.GetSkillsPromptResponse{Body: &spec.GetSkillsPromptResponseBody{}}, nil
			}
			allowed = resolved.AllowDefs
		}
		if len(value.Inserts) > 0 && !slices.Contains(value.Inserts, agentskillsSpec.SkillInsertInstructions) {
			return &spec.GetSkillsPromptResponse{Body: &spec.GetSkillsPromptResponseBody{}}, nil
		}
		filter = &agentskills.SkillFilter{
			Types:          append([]string(nil), value.Types...),
			LocationPrefix: value.LocationPrefix,
			AllowSkills:    allowed,
			SessionID:      value.SessionID,
			Activity:       value.Activity,
		}
	}
	prompt, err := s.runtime.SkillsPrompt(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &spec.GetSkillsPromptResponse{Body: &spec.GetSkillsPromptResponseBody{Prompt: prompt}}, nil
}

func (s *SkillRuntime) ListRuntimeSkills(
	ctx context.Context,
	req *spec.ListRuntimeSkillsRequest,
) (*spec.ListRuntimeSkillsResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil, fmt.Errorf("%w: missing filter", errSkillInvalidRequest)
	}
	filterRequest := req.Body.Filter
	if len(filterRequest.AllowSkillRefs) == 0 {
		return &spec.ListRuntimeSkillsResponse{
			Body: &spec.ListRuntimeSkillsResponseBody{Skills: []spec.RuntimeSkillListItem{}},
		}, nil
	}
	for _, ref := range filterRequest.AllowSkillRefs {
		if err := validateSkillRef(ref); err != nil {
			return nil, fmt.Errorf("%w: invalid filter.allowSkillRefs: %w", errSkillInvalidRequest, err)
		}
	}
	activity := filterRequest.Activity
	if activity == "" {
		activity = agentskillsSpec.SkillActivityAny
	}
	if activity == agentskillsSpec.SkillActivityActive && strings.TrimSpace(string(filterRequest.SessionID)) == "" {
		return nil, fmt.Errorf("%w: activity=active requires sessionID", errSkillInvalidRequest)
	}

	resolved := s.resolveAllowSkillRefs(ctx, filterRequest.AllowSkillRefs)
	if len(resolved.AllowDefs) == 0 {
		return &spec.ListRuntimeSkillsResponse{
			Body: &spec.ListRuntimeSkillsResponseBody{Skills: []spec.RuntimeSkillListItem{}},
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
	if filterRequest.SessionID != "" && activity == agentskillsSpec.SkillActivityAny {
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

	items := make([]spec.RuntimeSkillListItem, 0, len(records))
	seen := map[string]struct{}{}
	for _, record := range records {
		for _, ref := range resolved.DefToRefs[record.Def] {
			key := refKey(ref)
			if _, found := seen[key]; found {
				continue
			}
			seen[key] = struct{}{}
			_, isActive := active[record.Def]
			items = append(items, spec.RuntimeSkillListItem{
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
	}
	sort.Slice(items, func(left, right int) bool {
		return refKey(items[left].SkillRef) < refKey(items[right].SkillRef)
	})
	return &spec.ListRuntimeSkillsResponse{Body: &spec.ListRuntimeSkillsResponseBody{Skills: items}}, nil
}

func (s *SkillRuntime) RenderSkill(
	ctx context.Context,
	req *spec.RenderSkillRequest,
) (*spec.RenderSkillResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if err := validateSkillRef(req.Body.SkillRef); err != nil {
		return nil, fmt.Errorf("%w: invalid skillRef: %w", errSkillInvalidRequest, err)
	}
	definition, ok := s.definitionForSkillRef(ctx, req.Body.SkillRef)
	if !ok {
		return nil, errors.New("skill not found")
	}
	out, err := s.runtime.RenderSkill(
		ctx,
		agentskills.RenderSkillParams{Def: definition, Arguments: req.Body.Arguments},
	)
	if err != nil {
		return nil, err
	}
	return &spec.RenderSkillResponse{Body: &spec.RenderSkillResponseBody{
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
	}}, nil
}

type resolvedAllowSkillRefs struct {
	DefToRefs map[agentskillsSpec.SkillDef][]spec.SkillRef
	RefToDef  map[string]agentskillsSpec.SkillDef
	AllowDefs []agentskillsSpec.SkillDef
}

func (s *SkillRuntime) resolveAllowSkillRefs(
	ctx context.Context,
	refs []spec.SkillRef,
) resolvedAllowSkillRefs {
	output := resolvedAllowSkillRefs{
		DefToRefs: map[agentskillsSpec.SkillDef][]spec.SkillRef{},
		RefToDef:  map[string]agentskillsSpec.SkillDef{},
	}
	seenRefs := map[string]struct{}{}
	seenDefs := map[agentskillsSpec.SkillDef]struct{}{}
	for _, ref := range refs {
		key := refKey(ref)
		if key == "||" {
			continue
		}
		if _, found := seenRefs[key]; found {
			continue
		}
		seenRefs[key] = struct{}{}
		definition, ok := s.definitionForSkillRef(ctx, ref)
		if !ok {
			continue
		}
		output.DefToRefs[definition] = append(output.DefToRefs[definition], ref)
		output.RefToDef[key] = definition
		if _, found := seenDefs[definition]; !found {
			seenDefs[definition] = struct{}{}
			output.AllowDefs = append(output.AllowDefs, definition)
		}
	}
	sortSkillDefs(output.AllowDefs)
	return output
}

func buildActiveSkillRefs(
	defToRefs map[agentskillsSpec.SkillDef][]spec.SkillRef,
	active map[agentskillsSpec.SkillDef]struct{},
) []spec.SkillRef {
	seen := map[string]struct{}{}
	output := make([]spec.SkillRef, 0)
	for definition := range active {
		for _, ref := range defToRefs[definition] {
			key := refKey(ref)
			if _, found := seen[key]; found {
				continue
			}
			seen[key] = struct{}{}
			output = append(output, ref)
		}
	}
	sort.Slice(output, func(left, right int) bool {
		return refKey(output[left]) < refKey(output[right])
	})
	return output
}

func normalizeActiveRefsSubsetOfAllow(
	allow,
	active []spec.SkillRef,
) []spec.SkillRef {
	allowed := map[string]struct{}{}
	for _, ref := range allow {
		allowed[refKey(ref)] = struct{}{}
	}
	seen := map[string]struct{}{}
	output := make([]spec.SkillRef, 0, len(active))
	for _, ref := range active {
		key := refKey(ref)
		if _, ok := allowed[key]; !ok {
			continue
		}
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, ref)
	}
	return output
}

func refKey(ref spec.SkillRef) string {
	if ref.Identity != "" {
		return ref.Identity
	}
	return string(ref.BundleID) + "|" + string(ref.SkillSlug) + "|" + string(ref.SkillID)
}

func validateSkillRef(ref spec.SkillRef) error {
	if strings.TrimSpace(ref.Identity) != "" {
		if ref.BundleID != "" || ref.SkillSlug != "" || ref.SkillID != "" {
			return errors.New("identity and installed reference fields are mutually exclusive")
		}
		switch {
		case strings.HasPrefix(ref.Identity, installedIdentityPrefix):
			_, err := parseInstalledIdentity(ref.Identity)
			return err
		case strings.HasPrefix(ref.Identity, workspaceIdentityPrefix):
			_, _, err := parseWorkspaceIdentity(ref.Identity)
			return err
		default:
			return errors.New(
				"Skill identity has an unsupported source",
			)
		}
	}
	if strings.TrimSpace(string(ref.BundleID)) == "" {
		return errors.New("bundleID is empty")
	}
	if strings.TrimSpace(string(ref.SkillSlug)) == "" {
		return errors.New("skillSlug is empty")
	}
	if strings.TrimSpace(string(ref.SkillID)) == "" {
		return errors.New("skillID is empty")
	}
	return nil
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

func cloneSkillResourceInfo(input agentskillsSpec.SkillResourceInfo) agentskillsSpec.SkillResourceInfo {
	input.Locations = append([]string(nil), input.Locations...)
	return input
}
