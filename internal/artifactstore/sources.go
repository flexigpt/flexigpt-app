package artifactstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

// SourceUpdate replaces mutable local source-registration fields. SourceID,
// Kind, CreatedAt, and source observations remain store-owned.
type SourceUpdate struct {
	ExpectedModifiedAt time.Time
	DisplayName        string
	Enabled            bool
	ConfigSchemaID     spec.SchemaID
	Config             json.RawMessage
}

// CreateSource creates only app-local source-registration metadata. A source
// kind must have a registered driver, but this method never accesses content.
func (s *Store) CreateSource(ctx context.Context, draft spec.SourceDraft) (spec.ArtifactSource, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	defer finish()
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
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	defer finish()
	return s.repository.GetSource(ctx, sourceID)
}

// ListSources lists app-local source registrations by modification time.
func (s *Store) ListSources(ctx context.Context) ([]spec.ArtifactSource, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	return s.repository.ListSources(ctx)
}

// UpdateSource replaces mutable local registration fields and clears prior
// observations because changing configuration invalidates them.
func (s *Store) UpdateSource(
	ctx context.Context,
	sourceID spec.SourceID,
	update SourceUpdate,
) (spec.ArtifactSource, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	defer finish()

	source, err := s.repository.GetSource(ctx, sourceID)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	if err := requireExpectedModifiedAt(
		"source "+string(sourceID),
		source.ModifiedAt,
		update.ExpectedModifiedAt,
	); err != nil {
		return spec.ArtifactSource{}, err
	}
	previousEnabled := source.Enabled
	previousConfigSchemaID := source.ConfigSchemaID
	previousConfig := append(json.RawMessage(nil), source.Config...)

	source.DisplayName = update.DisplayName
	source.Enabled = update.Enabled
	source.ConfigSchemaID = update.ConfigSchemaID
	source.Config = normalizedJSONObject(update.Config)
	source.ModifiedAt = s.nextModifiedAt(source.ModifiedAt)
	if err := s.validateSource(ctx, &source); err != nil {
		return spec.ArtifactSource{}, err
	}
	observationInvalidated := previousEnabled != source.Enabled ||
		previousConfigSchemaID != source.ConfigSchemaID ||
		!bytes.Equal(previousConfig, source.Config)
	if observationInvalidated {
		if source.ObservationRevision >= spec.MaxObservationRevision {
			return spec.ArtifactSource{}, fmt.Errorf(
				"%w: source observation revision is exhausted",
				spec.ErrConflict,
			)
		}
		source.LastObservedGeneration = nil
		source.LastScannedAt = nil
		source.ObservationRevision++
	}
	if err := validate.ValidateArtifactSource(source); err != nil {
		return spec.ArtifactSource{}, fmt.Errorf(
			"%w: updated source: %w",
			spec.ErrInvalidRequest,
			err,
		)
	}
	if err := s.repository.UpdateSource(ctx, source, update.ExpectedModifiedAt); err != nil {
		return spec.ArtifactSource{}, err
	}
	return source, nil
}

// DeleteSource removes an app-local source registration only when repository
// relationships permit it. It does not mutate source content.
func (s *Store) DeleteSource(
	ctx context.Context,
	sourceID spec.SourceID,
	expectedModifiedAt time.Time,
) error {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer finish()

	source, err := s.repository.GetSource(ctx, sourceID)
	if err != nil {
		return err
	}
	if err := requireExpectedModifiedAt(
		"source "+string(sourceID),
		source.ModifiedAt,
		expectedModifiedAt,
	); err != nil {
		return err
	}
	return s.repository.DeleteSource(ctx, sourceID, expectedModifiedAt)
}

func (s *Store) validateSource(ctx context.Context, source *spec.ArtifactSource) error {
	if source == nil {
		return fmt.Errorf("%w: source is nil", spec.ErrInvalidRequest)
	}
	if len(source.Config) > spec.MaxConfigJSONBytes {
		return fmt.Errorf(
			"%w: source configuration exceeds %d bytes",
			spec.ErrInvalidRequest,
			spec.MaxConfigJSONBytes,
		)
	}
	canonicalConfig, err := baseutils.CanonicalizeJSON(source.Config)
	if err != nil {
		return fmt.Errorf("%w: source configuration: %w", spec.ErrInvalidRequest, err)
	}
	if len(canonicalConfig) == 0 || canonicalConfig[0] != '{' {
		return fmt.Errorf(
			"%w: source configuration must be a JSON object",
			spec.ErrInvalidRequest,
		)
	}
	source.Config = json.RawMessage(canonicalConfig)
	driver, ok := s.driverFor(source.Kind)
	if !ok {
		return fmt.Errorf("%w: source kind %q", spec.ErrDriverUnavailable, source.Kind)
	}
	diagnostics := make([]spec.Diagnostic, 0)
	if normalizer, ok := driver.(spec.SourceConfigNormalizer); ok {
		normalized, normalizationDiagnostics := normalizer.NormalizeConfig(
			ctx,
			append(json.RawMessage(nil), source.Config...),
		)
		if err := errorDiagnostics(
			"source "+string(source.Kind)+" configuration normalization",
			normalizationDiagnostics,
		); err != nil {
			return err
		}
		canonicalConfig, err = baseutils.CanonicalizeJSON(normalized)
		if err != nil {
			return fmt.Errorf(
				"%w: normalized source configuration: %w",
				spec.ErrInvalidRequest,
				err,
			)
		}
		if len(canonicalConfig) == 0 || canonicalConfig[0] != '{' {
			return fmt.Errorf("%w: normalized source configuration must be a JSON object", spec.ErrInvalidRequest)
		}
		source.Config = json.RawMessage(canonicalConfig)
		diagnostics = appendBoundedDiagnostics(diagnostics, normalizationDiagnostics...)
	}
	if err := validate.ValidateArtifactSource(*source); err != nil {
		return fmt.Errorf("%w: source: %w", spec.ErrInvalidRequest, err)
	}
	validationDiagnostics := driver.ValidateConfig(
		ctx,
		append(json.RawMessage(nil), source.Config...),
	)
	if err := errorDiagnostics("source "+string(source.Kind), validationDiagnostics); err != nil {
		return err
	}
	diagnostics = appendBoundedDiagnostics(diagnostics, validationDiagnostics...)
	source.Diagnostics = diagnostics
	if err := validate.ValidateArtifactSource(*source); err != nil {
		return fmt.Errorf("%w: validated source: %w", spec.ErrInvalidRequest, err)
	}
	return nil
}
