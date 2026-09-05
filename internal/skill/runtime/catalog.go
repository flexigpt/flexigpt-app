package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
)

type catalogView map[provider.SkillDef]string

// SyncCatalog pulls the current registrations from the configured
// runtime-owned CatalogSource. No catalog data is accepted from Wails callers.
func (s *Service) SyncCatalog(
	ctx context.Context,
	id CatalogID,
) error {
	if ctx == nil {
		return fmt.Errorf("%w: catalog context is nil", ErrInvalidRequest)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("%w: catalog ID is required", ErrInvalidRequest)
	}

	source, generation, err := s.beginCatalogSync(id)
	if err != nil {
		return err
	}

	values, err := source.Skills(ctx, id)
	if err != nil {
		cleanupErr := s.removeCatalogAtGeneration(
			context.WithoutCancel(ctx),
			id,
			generation,
		)
		return errors.Join(err, cleanupErr)
	}

	return s.reconcileCatalogAtGeneration(ctx, id, generation, values)
}

func (s *Service) reconcileCatalogAtGeneration(
	ctx context.Context,
	id CatalogID,
	generation uint64,
	values []SkillRegistration,
) error {
	if ctx == nil {
		return fmt.Errorf("%w: catalog context is nil", ErrInvalidRequest)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("%w: catalog ID is required", ErrInvalidRequest)
	}

	view, err := catalogViewFrom(values)
	if err != nil {
		cleanupErr := s.removeCatalogAtGeneration(
			context.WithoutCancel(ctx),
			id,
			generation,
		)
		return errors.Join(err, cleanupErr)
	}

	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()

	s.lifecycleMu.RLock()
	defer s.lifecycleMu.RUnlock()
	if s.closed || s.agentRuntime == nil {
		return ErrClosed
	}
	if s.generation[id] != generation {
		return nil
	}

	next := cloneCatalogViews(s.catalogs)
	next[id] = view
	s.catalogs = next

	return s.reconcileCatalogsLocked(ctx)
}

func (s *Service) RemoveCatalog(
	ctx context.Context,
	id CatalogID,
) error {
	if ctx == nil {
		return fmt.Errorf("%w: catalog context is nil", ErrInvalidRequest)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("%w: catalog ID is required", ErrInvalidRequest)
	}

	generation, err := s.beginCatalogRemoval(id)
	if err != nil {
		return err
	}
	return s.removeCatalogAtGeneration(ctx, id, generation)
}

func (s *Service) removeCatalogAtGeneration(
	ctx context.Context,
	id CatalogID,
	generation uint64,
) error {
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()

	s.lifecycleMu.RLock()
	defer s.lifecycleMu.RUnlock()
	if s.closed || s.agentRuntime == nil {
		return ErrClosed
	}
	if s.generation[id] != generation {
		return nil
	}

	next := cloneCatalogViews(s.catalogs)
	delete(next, id)
	s.catalogs = next

	return s.reconcileCatalogsLocked(ctx)
}

func (s *Service) IsRegistered(value SkillRegistration) bool {
	if s == nil {
		return false
	}

	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()

	s.lifecycleMu.RLock()
	defer s.lifecycleMu.RUnlock()
	if s.closed {
		return false
	}
	revision, found := s.registered[value.Definition]
	return found && revision == value.Revision
}

func (s *Service) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("%w: close context is nil", ErrInvalidRequest)
	}

	s.lifecycleMu.Lock()
	if s.closed {
		s.lifecycleMu.Unlock()
		return nil
	}
	s.closed = true
	s.lifecycleMu.Unlock()

	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()

	s.catalogs = map[CatalogID]catalogView{}
	err := s.reconcileCatalogsLocked(ctx)
	s.generation = nil
	return err
}

func (s *Service) reconcileCatalogsLocked(ctx context.Context) error {
	desired, mergeErr := mergeCatalogViews(s.catalogs)
	present, applyErr := s.applyDesiredLocked(
		ctx,
		s.registered,
		desired,
	)
	s.registered = present
	return errors.Join(mergeErr, applyErr)
}

func (s *Service) applyDesiredLocked(
	ctx context.Context,
	current catalogView,
	desired catalogView,
) (catalogView, error) {
	present := make(catalogView, len(current))
	maps.Copy(present, current)

	var applyErr error

	removals := make([]provider.SkillDef, 0)
	for def, currentRevision := range present {
		desiredRevision, wanted := desired[def]
		if !wanted || desiredRevision != currentRevision {
			removals = append(removals, def)
		}
	}
	sortSkillDefs(removals)

	for _, def := range removals {
		if err := ctx.Err(); err != nil {
			applyErr = errors.Join(applyErr, err)
			break
		}

		_, err := s.agentRuntime.RemoveSkill(ctx, def)
		if err != nil &&
			!errors.Is(err, agentskillsRuntimeSpec.ErrSkillNotFound) {
			applyErr = errors.Join(
				applyErr,
				fmt.Errorf(
					"remove runtime Skill %q: %w",
					def.Name,
					err,
				),
			)
			continue
		}
		delete(present, def)
	}

	additions := make([]provider.SkillDef, 0)
	for def := range desired {
		if _, found := present[def]; !found {
			additions = append(additions, def)
		}
	}
	sortSkillDefs(additions)

	for _, def := range additions {
		if err := ctx.Err(); err != nil {
			applyErr = errors.Join(applyErr, err)
			break
		}

		if _, err := s.agentRuntime.AddSkill(ctx, def); err != nil {
			applyErr = errors.Join(
				applyErr,
				fmt.Errorf(
					"add runtime Skill %q: %w",
					def.Name,
					err,
				),
			)
			continue
		}
		present[def] = desired[def]
	}

	return present, applyErr
}

// mergeCatalogViews permits Agent Skills to perform its native handle
// disambiguation for equal names at different definitions. Only an exact
// SkillDef requested at conflicting revisions is rejected.
func mergeCatalogViews(
	catalogs map[CatalogID]catalogView,
) (catalogView, error) {
	output := catalogView{}
	conflicts := map[provider.SkillDef]struct{}{}

	for _, catalog := range catalogs {
		for def, revision := range catalog {
			if _, conflict := conflicts[def]; conflict {
				continue
			}

			previous, found := output[def]
			if found && previous != revision {
				delete(output, def)
				conflicts[def] = struct{}{}
				continue
			}
			output[def] = revision
		}
	}

	if len(conflicts) == 0 {
		return output, nil
	}

	definitions := make([]provider.SkillDef, 0, len(conflicts))
	for def := range conflicts {
		definitions = append(definitions, def)
	}
	sortSkillDefs(definitions)

	names := make([]string, 0, len(definitions))
	for _, def := range definitions {
		names = append(names, fmt.Sprintf(
			"%s:%s@%s",
			def.Type,
			def.Name,
			def.Location,
		))
	}
	return output, fmt.Errorf(
		"%w: conflicting revisions for runtime Skills: %s",
		ErrInvalidRequest,
		strings.Join(names, ", "),
	)
}

func catalogViewFrom(
	values []SkillRegistration,
) (catalogView, error) {
	output := make(catalogView, len(values))
	for _, value := range values {
		previous, duplicate := output[value.Definition]
		if duplicate && previous != value.Revision {
			return nil, fmt.Errorf(
				"%w: duplicate Skill def has conflicting revisions: %+v",
				ErrInvalidRequest,
				value.Definition,
			)
		}
		output[value.Definition] = value.Revision
	}
	return output, nil
}

func cloneCatalogViews(
	input map[CatalogID]catalogView,
) map[CatalogID]catalogView {
	output := make(map[CatalogID]catalogView, len(input))
	for id, view := range input {
		cloned := make(catalogView, len(view))
		maps.Copy(cloned, view)
		output[id] = cloned
	}
	return output
}

func sortSkillDefs(values []provider.SkillDef) {
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
