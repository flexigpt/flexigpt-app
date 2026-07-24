package skillruntime

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"sort"
	"strings"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
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
	definitions map[agentskillsSpec.SkillDef]string
}

func newRuntimeDesiredView() runtimeDesiredView {
	return runtimeDesiredView{
		definitions: map[agentskillsSpec.SkillDef]string{},
	}
}

func (v *runtimeDesiredView) add(
	definition agentskillsSpec.SkillDef,
	version string,
) {
	if v.definitions == nil {
		v.definitions = map[agentskillsSpec.SkillDef]string{}
	}
	current, exists := v.definitions[definition]
	if !exists {
		v.definitions[definition] = version
		return
	}
	v.definitions[definition] = mergeRuntimeVersions(current, version)
}

func cloneRuntimeDesiredView(
	input runtimeDesiredView,
) runtimeDesiredView {
	output := newRuntimeDesiredView()
	maps.Copy(output.definitions, input.definitions)
	return output
}

func mergeRuntimeVersions(left, right string) string {
	values := map[string]struct{}{}
	for _, value := range append(
		strings.Split(left, "\x00"),
		strings.Split(right, "\x00")...,
	) {
		if value != "" {
			values[value] = struct{}{}
		}
	}
	output := make([]string, 0, len(values))
	for value := range values {
		output = append(output, value)
	}
	sort.Strings(output)
	return strings.Join(output, "\x00")
}

// ResyncInstalled updates only the installed desired partition. Runtime
// ownership is tracked independently from provider type because installed and
// Workspace filesystem skills both intentionally use provider type "fs".
func (s *SkillRuntime) ResyncInstalled(ctx context.Context) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	view, err := s.installedDesiredView(ctx, true)
	if err != nil {
		return err
	}

	return s.reconcilePartitionsLocked(
		ctx,
		view,
		cloneWorkspaceDesiredViews(s.managedWorkspaces),
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
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	view, err := s.installedDesiredView(ctx, true)
	if err != nil {
		return err
	}

	return s.reconcilePartitionsLocked(
		ctx,
		view,
		cloneWorkspaceDesiredViews(s.managedWorkspaces),
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

	view := newRuntimeDesiredView()
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
			view.add(
				definition,
				"installed:"+item.SkillDefinition.Digest+
					":"+item.SkillDefinition.ModifiedAt.UTC().Format(time.RFC3339Nano),
			)
		}
		if response.Body.NextPageToken == nil || *response.Body.NextPageToken == "" {
			break
		}
		skillToken = *response.Body.NextPageToken
	}
	return view, nil
}

func cloneWorkspaceDesiredViews(
	input map[artifactstore.RootID]runtimeDesiredView,
) map[artifactstore.RootID]runtimeDesiredView {
	output := make(
		map[artifactstore.RootID]runtimeDesiredView,
		len(input),
	)
	for rootID, value := range input {
		output[rootID] = cloneRuntimeDesiredView(value)
	}
	return output
}

func mergeDesiredPartitions(
	installed runtimeDesiredView,
	workspaces map[artifactstore.RootID]runtimeDesiredView,
) runtimeDesiredView {
	output := cloneRuntimeDesiredView(installed)
	for _, workspace := range workspaces {
		for definition, version := range workspace.definitions {
			output.add(definition, version)
		}
	}
	return output
}

func (s *SkillRuntime) reconcilePartitionsLocked(
	ctx context.Context,
	installed runtimeDesiredView,
	workspaces map[artifactstore.RootID]runtimeDesiredView,
	mode runtimeApplyMode,
) error {
	desired := mergeDesiredPartitions(installed, workspaces)
	managed, err := s.runtimeApplyDesired(
		ctx,
		s.managedRuntime,
		desired,
		mode,
	)
	// Agent Skills has no transaction spanning remove/add operations. Keep
	// desired state and observed runtime state separately even after a partial
	// failure so the next reconciliation converges instead of starting from
	// stale bookkeeping.
	s.managedInstalled = cloneRuntimeDesiredView(installed)
	s.managedWorkspaces = cloneWorkspaceDesiredViews(workspaces)
	s.managedRuntime = managed
	if err != nil {
		return err
	}
	return nil
}

// runtimeApplyDesired reconciles only definitions previously registered by
// this SkillRuntime. It never claims all definitions of a provider type.
func (s *SkillRuntime) runtimeApplyDesired(
	ctx context.Context,
	current map[agentskillsSpec.SkillDef]string,
	desired runtimeDesiredView,
	mode runtimeApplyMode,
) (map[agentskillsSpec.SkillDef]string, error) {
	present := make(map[agentskillsSpec.SkillDef]string, len(current))
	maps.Copy(present, current)

	var additions []agentskillsSpec.SkillDef
	for definition := range desired.definitions {
		if _, found := present[definition]; !found {
			additions = append(additions, definition)
		}
	}
	sortSkillDefs(additions)

	for _, definition := range additions {
		if _, err := s.runtime.AddSkill(ctx, definition); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillAlreadyExists) {
				present[definition] = desired.definitions[definition]
				continue
			}
			if mode == runtimeApplyStrict {
				return present, err
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
		present[definition] = desired.definitions[definition]
	}

	var reindexes []agentskillsSpec.SkillDef
	for definition, version := range desired.definitions {
		if currentVersion, found := present[definition]; found &&
			currentVersion != version {
			reindexes = append(reindexes, definition)
		}
	}
	sortSkillDefs(reindexes)
	for _, definition := range reindexes {
		if _, err := s.runtime.RemoveSkill(ctx, definition); err != nil &&
			!errors.Is(err, agentskillsSpec.ErrSkillNotFound) {
			if mode == runtimeApplyStrict {
				return present, err
			}
			slog.Error(
				"skill runtime reindex removal failed",
				"type", definition.Type,
				"name", definition.Name,
				"location", definition.Location,
				"err", err,
			)
			continue
		}
		delete(present, definition)
		if _, err := s.runtime.AddSkill(ctx, definition); err != nil {
			if mode == runtimeApplyStrict {
				return present, err
			}
			slog.Error(
				"skill runtime reindex add failed",
				"type", definition.Type,
				"name", definition.Name,
				"location", definition.Location,
				"err", err,
			)
			continue
		}
		present[definition] = desired.definitions[definition]
	}

	var removals []agentskillsSpec.SkillDef
	for definition := range present {
		if _, wanted := desired.definitions[definition]; !wanted {
			removals = append(removals, definition)
		}
	}
	sortSkillDefs(removals)
	for _, definition := range removals {
		if _, err := s.runtime.RemoveSkill(ctx, definition); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillNotFound) {
				delete(present, definition)
				continue
			}
			if mode == runtimeApplyStrict {
				return present, err
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
			continue
		}
		delete(present, definition)
	}
	return present, nil
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
