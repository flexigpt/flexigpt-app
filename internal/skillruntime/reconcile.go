package skillruntime

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
	skillstoreSpec "github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
)

type runtimeApplyMode int

const (
	runtimeApplyBestEffort runtimeApplyMode = iota
	runtimeApplyStrict
)

type runtimeDesiredView struct {
	set        map[agentskillsSpec.SkillDef]struct{}
	byTypeName map[string][]agentskillsSpec.SkillDef
}

// ResyncInstalled reconciles only the installed source partition.
// It never removes Workspace definitions.
func (s *SkillRuntime) ResyncInstalled(ctx context.Context) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	view, err := s.installedDesiredView(ctx, true)
	if err != nil {
		return err
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	return s.runtimeApplyDesired(
		ctx,
		view.set,
		view.byTypeName,
		func(definition agentskillsSpec.SkillDef) bool {
			return definition.Type != workspaceSkillProviderType
		},
		runtimeApplyStrict,
	)
}

func (s *SkillRuntime) bestEffortInstalledResync(
	ctx context.Context,
	reason string,
) {
	if s == nil || s.runtime == nil || s.store == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("skill runtime resync: panic", "reason", reason, "panic", recovered)
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, runtimeResyncTimeout)
	defer cancel()
	if err := s.resyncInstalledBestEffort(ctx); err != nil {
		slog.Error("skill runtime resync failed", "reason", reason, "err", err)
	}
}

func (s *SkillRuntime) resyncInstalledBestEffort(ctx context.Context) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	view, err := s.installedDesiredView(ctx, true)
	if err != nil {
		return err
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	return s.runtimeApplyDesired(
		ctx,
		view.set,
		view.byTypeName,
		func(definition agentskillsSpec.SkillDef) bool {
			return definition.Type != workspaceSkillProviderType
		},
		runtimeApplyBestEffort,
	)
}

func (s *SkillRuntime) installedDesiredView(
	ctx context.Context,
	logInvalid bool,
) (runtimeDesiredView, error) {
	bundles := map[bundleitemutils.BundleID]skillstoreSpec.SkillBundle{}
	bundleToken := ""
	for {
		response, err := s.store.ListSkillBundles(ctx, &skillstoreSpec.ListSkillBundlesRequest{
			IncludeDisabled: true,
			PageSize:        256,
			PageToken:       bundleToken,
		})
		if err != nil {
			return runtimeDesiredView{}, err
		}
		if response == nil || response.Body == nil {
			return runtimeDesiredView{}, errors.New("Skill Store returned an empty bundle list response")
		}
		for _, bundle := range response.Body.SkillBundles {
			bundles[bundle.ID] = bundle
		}
		if response.Body.NextPageToken == nil || *response.Body.NextPageToken == "" {
			break
		}
		bundleToken = *response.Body.NextPageToken
	}

	view := runtimeDesiredView{
		set:        map[agentskillsSpec.SkillDef]struct{}{},
		byTypeName: map[string][]agentskillsSpec.SkillDef{},
	}
	skillToken := ""
	for {
		response, err := s.store.ListSkills(ctx, &skillstoreSpec.ListSkillsRequest{
			IncludeDisabled:     true,
			IncludeMissing:      true,
			RecommendedPageSize: 256,
			PageToken:           skillToken,
		})
		if err != nil {
			return runtimeDesiredView{}, err
		}
		if response == nil || response.Body == nil {
			return runtimeDesiredView{}, errors.New("Skill Store returned an empty Skill list response")
		}
		for _, item := range response.Body.SkillListItems {
			bundle, ok := bundles[item.BundleID]
			if !ok || !bundle.IsEnabled || !item.SkillDefinition.IsEnabled {
				continue
			}
			definition, err := s.runtimeDefForStoreSkill(item.SkillDefinition)
			if err != nil {
				if logInvalid {
					slog.Error(
						"runtime desired Skill has invalid definition",
						"bundleID",
						item.BundleID,
						"skill",
						item.SkillSlug,
						"err",
						err,
					)
				}
				continue
			}
			view.set[definition] = struct{}{}
			key := typeNameKey(definition.Type, definition.Name)
			view.byTypeName[key] = append(view.byTypeName[key], definition)
		}
		if response.Body.NextPageToken == nil || *response.Body.NextPageToken == "" {
			break
		}
		skillToken = *response.Body.NextPageToken
	}
	return view, nil
}

func (s *SkillRuntime) runtimeApplyDesired(
	ctx context.Context,
	desired map[agentskillsSpec.SkillDef]struct{},
	desiredByTypeName map[string][]agentskillsSpec.SkillDef,
	owns func(agentskillsSpec.SkillDef) bool,
	mode runtimeApplyMode,
) error {
	currentRecords, err := s.runtime.ListSkills(ctx, nil)
	if err != nil {
		return err
	}
	current := make(map[agentskillsSpec.SkillDef]struct{}, len(currentRecords))
	for _, record := range currentRecords {
		if owns != nil && !owns(record.Def) {
			continue
		}
		current[record.Def] = struct{}{}
	}

	var additions []agentskillsSpec.SkillDef
	for definition := range desired {
		if _, found := current[definition]; !found {
			additions = append(additions, definition)
		}
	}
	sortSkillDefs(additions)
	present := make(map[agentskillsSpec.SkillDef]struct{}, len(current)+len(additions))
	for definition := range current {
		present[definition] = struct{}{}
	}
	for _, definition := range additions {
		if _, err := s.runtime.AddSkill(ctx, definition); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillAlreadyExists) {
				present[definition] = struct{}{}
				continue
			}
			if mode == runtimeApplyStrict {
				return err
			}
			slog.Error(
				"skill runtime add failed",
				"type",
				definition.Type,
				"name",
				definition.Name,
				"location",
				definition.Location,
				"err",
				err,
			)
			continue
		}
		present[definition] = struct{}{}
	}

	desiredPresent := map[string]bool{}
	for definition := range present {
		if _, wanted := desired[definition]; wanted {
			desiredPresent[typeNameKey(definition.Type, definition.Name)] = true
		}
	}
	var removals []agentskillsSpec.SkillDef
	for definition := range current {
		if _, wanted := desired[definition]; !wanted {
			removals = append(removals, definition)
		}
	}
	sortSkillDefs(removals)
	for _, definition := range removals {
		key := typeNameKey(definition.Type, definition.Name)
		if _, replacing := desiredByTypeName[key]; replacing && !desiredPresent[key] {
			if mode == runtimeApplyBestEffort {
				slog.Warn(
					"skill runtime removal skipped because replacement is unavailable",
					"type",
					definition.Type,
					"name",
					definition.Name,
					"location",
					definition.Location,
				)
			}
			continue
		}
		if _, err := s.runtime.RemoveSkill(ctx, definition); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillNotFound) {
				continue
			}
			if mode == runtimeApplyStrict {
				return err
			}
			slog.Error(
				"skill runtime remove failed",
				"type",
				definition.Type,
				"name",
				definition.Name,
				"location",
				definition.Location,
				"err",
				err,
			)
		}
	}
	return nil
}

func (s *SkillRuntime) definitionForSkillRef(
	ctx context.Context,
	ref spec.SkillRef,
) (agentskillsSpec.SkillDef, bool) {
	if ref.Identity != "" {
		switch {
		case strings.HasPrefix(ref.Identity, installedIdentityPrefix):
			installedRef, err := parseInstalledIdentity(ref.Identity)
			if err != nil {
				return agentskillsSpec.SkillDef{}, false
			}
			ref.BundleID = installedRef.BundleID
			ref.SkillSlug = installedRef.SkillSlug
			ref.SkillID = installedRef.SkillID

		case strings.HasPrefix(ref.Identity, workspaceIdentityPrefix):
			return s.workspaceDefinitionForIdentity(ctx, ref.Identity)

		default:
			return agentskillsSpec.SkillDef{}, false
		}
	}
	if strings.TrimSpace(string(ref.BundleID)) == "" || strings.TrimSpace(string(ref.SkillSlug)) == "" {
		return agentskillsSpec.SkillDef{}, false
	}
	response, err := s.store.GetSkill(ctx, &skillstoreSpec.GetSkillRequest{
		BundleID:  ref.BundleID,
		SkillSlug: ref.SkillSlug,
	})
	if err != nil || response == nil || response.Body == nil {
		return agentskillsSpec.SkillDef{}, false
	}
	if response.Body.ID != ref.SkillID {
		return agentskillsSpec.SkillDef{}, false
	}
	definition, err := s.runtimeDefForStoreSkill(*response.Body)
	if err != nil {
		return agentskillsSpec.SkillDef{}, false
	}
	return definition, true
}

func (s *SkillRuntime) runtimeDefForStoreSkill(
	skill skillstoreSpec.Skill,
) (agentskillsSpec.SkillDef, error) {
	source, err := s.store.ResolveSkillSource(skill)
	if err != nil {
		return agentskillsSpec.SkillDef{}, err
	}
	return agentskillsSpec.SkillDef{
		Type:     source.Type,
		Name:     source.Name,
		Location: source.Location,
	}, nil
}

func sortSkillDefs(values []agentskillsSpec.SkillDef) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Type != values[right].Type {
			return values[left].Type < values[right].Type
		}
		if values[left].Name != values[right].Name {
			return values[left].Name < values[right].Name
		}
		return values[left].Location < values[right].Location
	})
}

func typeNameKey(skillType, name string) string {
	return skillType + "\x00" + name
}
