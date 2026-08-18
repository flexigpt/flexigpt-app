package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"sort"
	"strings"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
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
	if ctx == nil {
		return fmt.Errorf(
			"%w: Skill runtime resync context is nil",
			basespec.ErrInvalid,
		)
	}

	// Resolve only the requested Collection and do slow source verification
	// without holding the process-local runtime mutation lock.
	values, err := s.resolver.ListCollectionSkills(ctx, ref)
	if err != nil {
		return s.failClosedCollection(ctx, ref, err)
	}
	desired, err := desiredCollectionView(ref, values)
	if err != nil {
		return s.failClosedCollection(ctx, ref, err)
	}

	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()
	if s.isClosed() {
		return basespec.ErrClosed
	}

	collections := cloneCollectionDesiredViews(s.managedCollections)
	collections[ref] = desired

	return s.reconcileCollectionsLocked(
		ctx,
		collections,
		runtimeApplyStrict,
	)
}

// WarmCollections best-effort warms process-local runtime registrations for
// known Collections. It is intended for optional background startup warmup.
//
// Durable Artifact Store state remains authoritative. Failed Collections are
// omitted from this pass and can be resolved again through normal lazy
// foreground reconciliation.
func (s *SkillRuntime) WarmCollections(
	ctx context.Context,
	refs []collection.CollectionRef,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Skill runtime warmup context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	normalized, err := normalizeWarmCollectionRefs(refs)
	if err != nil {
		return err
	}

	// Catalog and filesystem work must not hold rtResyncMu. Otherwise the
	// first foreground Skill request waits behind the complete warmup.
	warmed := make(map[collection.CollectionRef]runtimeDesiredView, len(normalized))
	var warmErr error

	for _, ref := range normalized {
		if err := ctx.Err(); err != nil {
			warmErr = errors.Join(warmErr, err)
			break
		}

		values, err := s.resolver.ListCollectionSkills(ctx, ref)
		if err != nil {
			warmErr = errors.Join(
				warmErr,
				fmt.Errorf(
					"warm Skill runtime Collection %q: %w",
					ref.CollectionID,
					err,
				),
			)
			continue
		}

		desired, err := desiredCollectionView(ref, values)
		if err != nil {
			warmErr = errors.Join(
				warmErr,
				fmt.Errorf(
					"build warmed Skill runtime Collection %q: %w",
					ref.CollectionID,
					err,
				),
			)
			continue
		}
		warmed[ref] = desired
	}

	if len(warmed) == 0 {
		return warmErr
	}

	s.rtResyncMu.Lock()
	if s.isClosed() {
		s.rtResyncMu.Unlock()
		return errors.Join(warmErr, basespec.ErrClosed)
	}

	// Preserve a view already loaded by a foreground request while this
	// background operation was resolving source state.
	collections := cloneCollectionDesiredViews(s.managedCollections)
	for ref, desired := range warmed {
		if _, foregroundLoaded := collections[ref]; foregroundLoaded {
			continue
		}
		collections[ref] = desired
	}

	applyErr := s.reconcileCollectionsLocked(
		ctx,
		collections,
		runtimeApplyBestEffort,
	)
	s.rtResyncMu.Unlock()
	return errors.Join(warmErr, applyErr)
}

func normalizeWarmCollectionRefs(
	refs []collection.CollectionRef,
) ([]collection.CollectionRef, error) {
	seen := make(map[collection.CollectionRef]struct{}, len(refs))
	output := make([]collection.CollectionRef, 0, len(refs))
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[ref]; duplicate {
			continue
		}
		seen[ref] = struct{}{}
		output = append(output, ref)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].CollectionID < output[right].CollectionID
	})
	return output, nil
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

	if s.isClosed() {
		return basespec.ErrClosed
	}

	collections := cloneCollectionDesiredViews(s.managedCollections)
	delete(collections, ref)
	return s.reconcileCollectionsLocked(ctx, collections, runtimeApplyStrict)
}

func (s *SkillRuntime) failClosedCollection(
	ctx context.Context,
	ref collection.CollectionRef,
	cause error,
) error {
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	if s.isClosed() {
		return errors.Join(cause, basespec.ErrClosed)
	}
	return s.failClosedCollectionLocked(ctx, ref, cause)
}

func (s *SkillRuntime) failClosedCollectionLocked(
	ctx context.Context,
	ref collection.CollectionRef,
	cause error,
) error {
	collections := cloneCollectionDesiredViews(s.managedCollections)
	delete(collections, ref)

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
	if s.isClosed() {
		return basespec.ErrClosed
	}

	desired, mergeErr := mergeDesiredCollections(collections)
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
	return errors.Join(mergeErr, err)
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

// mergeDesiredCollections deterministically fails closed on any same-name
// collision. The Agent Skills runtime has global name semantics, so retaining
// either arbitrary source would turn map or scheduling order into policy.
func mergeDesiredCollections(
	collections map[collection.CollectionRef]runtimeDesiredView,
) (runtimeDesiredView, error) {
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
	if len(collided) == 0 {
		return output, nil
	}

	names := make([]string, 0, len(collided))
	for name := range collided {
		names = append(names, name)
	}
	sort.Strings(names)
	return output, fmt.Errorf(
		"%w: runtime Skill names are ambiguous across Collections: %s",
		basespec.ErrConflict,
		strings.Join(names, ", "),
	)
}

func desiredCollectionView(
	ref collection.CollectionRef,
	values []ResolvedArtifactSkill,
) (runtimeDesiredView, error) {
	desired := newRuntimeDesiredView()
	names := make(map[string]artifact.ArtifactRef, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return runtimeDesiredView{}, err
		}
		if value.Collection != ref {
			return runtimeDesiredView{}, fmt.Errorf(
				"%w: runtime Skill belongs to another Collection",
				basespec.ErrInvalid,
			)
		}
		if previous, exists := names[value.Definition.Name]; exists &&
			previous != value.Artifact {
			return runtimeDesiredView{}, fmt.Errorf(
				"%w: collection %q has multiple runtime Skills named %q",
				basespec.ErrConflict,
				ref.CollectionID,
				value.Definition.Name,
			)
		}
		names[value.Definition.Name] = value.Artifact
		desired.add(value)
	}
	return desired, nil
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

func cloneRuntimeDesiredView(input runtimeDesiredView) runtimeDesiredView {
	output := newRuntimeDesiredView()
	maps.Copy(output.definitions, input.definitions)
	maps.Copy(output.artifacts, input.artifacts)
	return output
}

func newRuntimeDesiredView() runtimeDesiredView {
	return runtimeDesiredView{
		definitions: map[agentskillsSpec.SkillDef]string{},
		artifacts:   map[agentskillsSpec.SkillDef]string{},
	}
}

func artifactRefKey(ref artifact.ArtifactRef) string {
	return string(ref.RootID) + "\x00" + string(ref.ArtifactID)
}
