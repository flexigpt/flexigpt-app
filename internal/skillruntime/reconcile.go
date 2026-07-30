package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"sort"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

type runtimeApplyMode int

const (
	runtimeApplyBestEffort runtimeApplyMode = iota
	runtimeApplyStrict
)

type runtimeDesiredView struct {
	definitions map[agentskillsSpec.SkillDef]string
	artifacts   map[agentskillsSpec.SkillDef]string
}

func newRuntimeDesiredView() runtimeDesiredView {
	return runtimeDesiredView{
		definitions: map[agentskillsSpec.SkillDef]string{},
		artifacts:   map[agentskillsSpec.SkillDef]string{},
	}
}

func (v *runtimeDesiredView) add(value ResolvedArtifactSkill) {
	if v.definitions == nil {
		v.definitions = map[agentskillsSpec.SkillDef]string{}
	}
	if v.artifacts == nil {
		v.artifacts = map[agentskillsSpec.SkillDef]string{}
	}
	v.definitions[value.Definition] = value.Version
	v.artifacts[value.Definition] = artifactRefKey(value.Artifact)
}

func cloneRuntimeDesiredView(input runtimeDesiredView) runtimeDesiredView {
	output := newRuntimeDesiredView()
	maps.Copy(output.definitions, input.definitions)
	maps.Copy(output.artifacts, input.artifacts)
	return output
}

func cloneCollectionDesiredViews(
	input map[collection.CollectionRef]runtimeDesiredView,
) map[collection.CollectionRef]runtimeDesiredView {
	output := make(
		map[collection.CollectionRef]runtimeDesiredView,
		len(input),
	)
	for ref, value := range input {
		output[ref] = cloneRuntimeDesiredView(value)
	}
	return output
}

// mergeDesiredCollections deterministically fails closed on any same-name
// collision. The Agent Skills runtime has global name semantics, so retaining
// either arbitrary source would turn map or scheduling order into policy.
func mergeDesiredCollections(
	collections map[collection.CollectionRef]runtimeDesiredView,
) runtimeDesiredView {
	byName := map[string]map[string]struct{}{}
	for _, view := range collections {
		for definition := range view.definitions {
			if byName[definition.Name] == nil {
				byName[definition.Name] = map[string]struct{}{}
			}
			byName[definition.Name][view.artifacts[definition]] = struct{}{}
		}
	}

	collided := map[string]struct{}{}
	for name, artifacts := range byName {
		if len(artifacts) > 1 {
			collided[name] = struct{}{}
			slog.Error(
				"skill runtime same-name collision rejected",
				"name",
				name,
				"artifacts",
				len(artifacts),
			)
		}
	}

	output := newRuntimeDesiredView()
	for _, view := range collections {
		for definition, version := range view.definitions {
			if _, collision := collided[definition.Name]; collision {
				continue
			}
			output.definitions[definition] = version
			output.artifacts[definition] = view.artifacts[definition]
		}
	}
	return output
}

func (s *SkillRuntime) ResyncCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}

	values, err := s.resolver.ListCollectionSkills(ctx, ref)
	if err != nil {
		s.rtResyncMu.Lock()
		defer s.rtResyncMu.Unlock()
		return s.failClosedCollectionLocked(ctx, ref, err)
	}

	desired := newRuntimeDesiredView()
	for _, value := range values {
		if err := value.Validate(); err != nil {
			s.rtResyncMu.Lock()
			nErr := s.failClosedCollectionLocked(ctx, ref, err)
			s.rtResyncMu.Unlock()
			return nErr
		}
		desired.add(value)
	}

	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	collections := cloneCollectionDesiredViews(s.managedCollections)
	collections[ref] = desired
	return s.reconcileCollectionsLocked(ctx, collections, runtimeApplyStrict)
}

func (s *SkillRuntime) RemoveCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}

	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	collections := cloneCollectionDesiredViews(s.managedCollections)
	delete(collections, ref)
	return s.reconcileCollectionsLocked(ctx, collections, runtimeApplyStrict)
}

func (s *SkillRuntime) failClosedCollectionLocked(
	ctx context.Context,
	ref collection.CollectionRef,
	cause error,
) error {
	collections := cloneCollectionDesiredViews(s.managedCollections)
	collections[ref] = newRuntimeDesiredView()

	cleanupContext, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		runtimeForegroundValidateTimeout,
	)
	defer cancel()

	cleanupErr := s.reconcileCollectionsLocked(
		cleanupContext,
		collections,
		runtimeApplyBestEffort,
	)
	if cleanupErr != nil {
		return errors.Join(
			cause,
			fmt.Errorf("fail-closed Skill runtime cleanup: %w", cleanupErr),
		)
	}
	return cause
}

func (s *SkillRuntime) reconcileCollectionsLocked(
	ctx context.Context,
	collections map[collection.CollectionRef]runtimeDesiredView,
	mode runtimeApplyMode,
) error {
	desired := mergeDesiredCollections(collections)
	managed, err := s.runtimeApplyDesired(
		ctx,
		s.managedRuntime,
		desired,
		mode,
	)

	// Runtime registration is inherently non-transactional. Retain the desired
	// feature partitions even after a partial provider operation so the next
	// reconciliation converges from actual process-local state.
	s.managedCollections = cloneCollectionDesiredViews(collections)
	s.managedRuntime = managed
	return err
}

func (s *SkillRuntime) runtimeApplyDesired(
	ctx context.Context,
	current map[agentskillsSpec.SkillDef]string,
	desired runtimeDesiredView,
	mode runtimeApplyMode,
) (map[agentskillsSpec.SkillDef]string, error) {
	present := make(map[agentskillsSpec.SkillDef]string, len(current))
	maps.Copy(present, current)

	removals := make([]agentskillsSpec.SkillDef, 0)
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
			slog.Error("skill runtime remove failed", "name", definition.Name, "err", err)
			continue
		}
		delete(present, definition)
	}

	reindexes := make([]agentskillsSpec.SkillDef, 0)
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
			slog.Error("skill runtime reindex removal failed", "name", definition.Name, "err", err)
			continue
		}
		delete(present, definition)
	}

	additions := make([]agentskillsSpec.SkillDef, 0)
	for definition := range desired.definitions {
		if _, found := present[definition]; !found {
			additions = append(additions, definition)
		}
	}
	sortSkillDefs(additions)

	for _, definition := range additions {
		if _, err := s.runtime.AddSkill(ctx, definition); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillAlreadyExists) {
				err = fmt.Errorf(
					"runtime Skill name collision remained after fail-closed reconciliation: %q",
					definition.Name,
				)
			}
			if mode == runtimeApplyStrict {
				return present, err
			}
			slog.Error("skill runtime add failed", "name", definition.Name, "err", err)
			continue
		}
		present[definition] = desired.definitions[definition]
	}
	return present, nil
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

func artifactRefKey(ref artifact.ArtifactRef) string {
	return string(ref.RootID) + "\x00" + string(ref.ArtifactID)
}
