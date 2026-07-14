package artifactstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

// SourceUpdate replaces mutable local source-registration fields. SourceID,
// Kind, CreatedAt, and source observations remain store-owned.
type SourceUpdate struct {
	DisplayName    string
	Enabled        bool
	ConfigSchemaID spec.SchemaID
	Config         json.RawMessage
}

// CreateSource creates only app-local source-registration metadata. A source
// kind must have a registered driver, but this method never accesses content.
func (s *Store) CreateSource(ctx context.Context, draft spec.SourceDraft) (spec.ArtifactSource, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactSource{}, err
	}
	if err := ctx.Err(); err != nil {
		return spec.ArtifactSource{}, err
	}
	id, err := s.newID()
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	now := s.nowUTC()
	source := spec.ArtifactSource{
		SourceID:       spec.SourceID(id),
		Kind:           draft.Kind,
		DisplayName:    draft.DisplayName,
		Enabled:        draft.Enabled,
		ConfigSchemaID: draft.ConfigSchemaID,
		Config:         normalizedJSONObject(draft.Config),
		CreatedAt:      now,
		ModifiedAt:     now,
	}
	if err := s.validateSource(ctx, &source); err != nil {
		return spec.ArtifactSource{}, err
	}
	if err := s.repository.CreateSource(ctx, source); err != nil {
		return spec.ArtifactSource{}, err
	}
	return source, nil
}

// GetSource returns one app-local source registration.
func (s *Store) GetSource(ctx context.Context, sourceID spec.SourceID) (spec.ArtifactSource, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactSource{}, err
	}
	return s.repository.GetSource(ctx, sourceID)
}

// ListSources lists app-local source registrations by modification time.
func (s *Store) ListSources(ctx context.Context) ([]spec.ArtifactSource, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.repository.ListSources(ctx)
}

// UpdateSource replaces mutable local registration fields and clears prior
// observations because changing configuration invalidates them.
func (s *Store) UpdateSource(
	ctx context.Context,
	sourceID spec.SourceID,
	update SourceUpdate,
) (spec.ArtifactSource, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactSource{}, err
	}
	source, err := s.repository.GetSource(ctx, sourceID)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	source.DisplayName = update.DisplayName
	source.Enabled = update.Enabled
	source.ConfigSchemaID = update.ConfigSchemaID
	source.Config = normalizedJSONObject(update.Config)
	source.LastObservedGeneration = nil
	source.LastScannedAt = nil
	source.Diagnostics = nil
	source.ModifiedAt = s.nowUTC()
	if err := s.validateSource(ctx, &source); err != nil {
		return spec.ArtifactSource{}, err
	}
	if err := s.repository.UpdateSource(ctx, source); err != nil {
		return spec.ArtifactSource{}, err
	}
	return source, nil
}

// DeleteSource removes an app-local source registration only when SQLite
// foreign-key relationships permit it. It does not mutate source content.
func (s *Store) DeleteSource(ctx context.Context, sourceID spec.SourceID) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	return s.repository.DeleteSource(ctx, sourceID)
}

func (s *Store) validateSource(ctx context.Context, source *spec.ArtifactSource) error {
	if source == nil {
		return fmt.Errorf("%w: source is nil", spec.ErrInvalidRequest)
	}
	if err := spec.ValidateArtifactSource(*source); err != nil {
		return fmt.Errorf("%w: source: %w", spec.ErrInvalidRequest, err)
	}
	driver, ok := s.driverFor(source.Kind)
	if !ok {
		return fmt.Errorf("%w: source kind %q", spec.ErrDriverUnavailable, source.Kind)
	}
	diagnostics := driver.ValidateConfig(ctx, append(json.RawMessage(nil), source.Config...))
	if err := errorDiagnostics("source "+string(source.Kind), diagnostics); err != nil {
		return err
	}
	source.Diagnostics = append([]spec.Diagnostic(nil), diagnostics...)
	if err := spec.ValidateArtifactSource(*source); err != nil {
		return fmt.Errorf("%w: validated source: %w", spec.ErrInvalidRequest, err)
	}
	return nil
}
