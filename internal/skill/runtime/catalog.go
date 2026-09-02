package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
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

	source, err := s.catalogSourceForSync()
	if err != nil {
		return err
	}

	values, err := source.Skills(ctx, id)
	if err != nil {
		cleanupErr := s.RemoveCatalog(context.WithoutCancel(ctx), id)
		return errors.Join(err, cleanupErr)
	}

	return s.reconcileCatalog(ctx, id, values)
}

func (s *Service) reconcileCatalog(
	ctx context.Context,
	id CatalogID,
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
		cleanupErr := s.RemoveCatalog(
			context.WithoutCancel(ctx),
			id,
		)
		return errors.Join(err, cleanupErr)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.agentRuntime == nil {
		return ErrClosed
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
	if id == "" {
		return fmt.Errorf("%w: catalog ID is required", ErrInvalidRequest)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.agentRuntime == nil {
		return ErrClosed
	}

	next := cloneCatalogViews(s.catalogs)
	delete(next, id)
	s.catalogs = next

	return s.reconcileCatalogsLocked(ctx)
}

func (s *Service) CatalogIDs() []CatalogID {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	output := make([]CatalogID, 0, len(s.catalogs))
	for id := range s.catalogs {
		output = append(output, id)
	}
	slices.Sort(output)
	return output
}

func (s *Service) IsRegistered(value SkillRegistration) bool {
	if s == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.catalogs = map[CatalogID]catalogView{}
	err := s.reconcileCatalogsLocked(ctx)
	s.closed = true
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
	for definition, currentRevision := range present {
		desiredRevision, wanted := desired[definition]
		if !wanted || desiredRevision != currentRevision {
			removals = append(removals, definition)
		}
	}
	sortSkillDefs(removals)

	for _, definition := range removals {
		if err := ctx.Err(); err != nil {
			applyErr = errors.Join(applyErr, err)
			break
		}

		_, err := s.agentRuntime.RemoveSkill(ctx, definition)
		if err != nil &&
			!errors.Is(err, agentskillsRuntimeSpec.ErrSkillNotFound) {
			applyErr = errors.Join(
				applyErr,
				fmt.Errorf(
					"remove runtime Skill %q: %w",
					definition.Name,
					err,
				),
			)
			continue
		}
		delete(present, definition)
	}

	additions := make([]provider.SkillDef, 0)
	for definition := range desired {
		if _, found := present[definition]; !found {
			additions = append(additions, definition)
		}
	}
	sortSkillDefs(additions)

	for _, definition := range additions {
		if err := ctx.Err(); err != nil {
			applyErr = errors.Join(applyErr, err)
			break
		}

		if _, err := s.agentRuntime.AddSkill(ctx, definition); err != nil {
			applyErr = errors.Join(
				applyErr,
				fmt.Errorf(
					"add runtime Skill %q: %w",
					definition.Name,
					err,
				),
			)
			continue
		}
		present[definition] = desired[definition]
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
		for definition, revision := range catalog {
			if _, conflict := conflicts[definition]; conflict {
				continue
			}

			previous, found := output[definition]
			if found && previous != revision {
				delete(output, definition)
				conflicts[definition] = struct{}{}
				continue
			}
			output[definition] = revision
		}
	}

	if len(conflicts) == 0 {
		return output, nil
	}

	definitions := make([]provider.SkillDef, 0, len(conflicts))
	for definition := range conflicts {
		definitions = append(definitions, definition)
	}
	sortSkillDefs(definitions)

	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, fmt.Sprintf(
			"%s:%s@%s",
			definition.Type,
			definition.Name,
			definition.Location,
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
				"%w: duplicate Skill definition has conflicting revisions: %+v",
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
