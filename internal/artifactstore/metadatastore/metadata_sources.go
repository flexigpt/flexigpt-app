package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

const sourceColumns = `
	source_id,
	kind,
	display_name,
	enabled,
	config_schema_id,
	config_json,
	last_observed_generation,
	last_scanned_at,
	diagnostics_json,
	created_at,
	modified_at`

func (s *MetadataStore) CreateSource(ctx context.Context, source spec.ArtifactSource) error {
	if err := spec.ValidateArtifactSource(source); err != nil {
		return fmt.Errorf("validate source for persistence: %w", err)
	}
	diagnostics, err := encodeDiagnostics(source.Diagnostics)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO artifact_sources (
			source_id, kind, display_name, enabled, config_schema_id, config_json,
			last_observed_generation, last_scanned_at, diagnostics_json, created_at, modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(source.SourceID),
		string(source.Kind),
		source.DisplayName,
		boolToInt(source.Enabled),
		string(source.ConfigSchemaID),
		[]byte(source.Config),
		nullableSourceGeneration(source.LastObservedGeneration),
		nullableTime(source.LastScannedAt),
		diagnostics,
		formatTime(source.CreatedAt),
		formatTime(source.ModifiedAt),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("insert source: %w", err))
	}
	return nil
}

func (s *MetadataStore) GetSource(ctx context.Context, sourceID spec.SourceID) (spec.ArtifactSource, error) {
	source, err := scanSource(
		s.db.QueryRowContext(
			ctx,
			`SELECT `+sourceColumns+` FROM artifact_sources WHERE source_id = ?`,
			string(sourceID),
		),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return spec.ArtifactSource{}, fmt.Errorf("%w: source %q", spec.ErrNotFound, sourceID)
	}
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	return source, nil
}

func (s *MetadataStore) ListSources(ctx context.Context) ([]spec.ArtifactSource, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+sourceColumns+` FROM artifact_sources ORDER BY modified_at DESC, source_id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()

	sources := make([]spec.ArtifactSource, 0)
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sources: %w", err)
	}
	return sources, nil
}

func (s *MetadataStore) UpdateSource(ctx context.Context, source spec.ArtifactSource) error {
	if err := spec.ValidateArtifactSource(source); err != nil {
		return fmt.Errorf("validate source for persistence: %w", err)
	}
	diagnostics, err := encodeDiagnostics(source.Diagnostics)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE artifact_sources
		   SET display_name = ?,
		       enabled = ?,
		       config_schema_id = ?,
		       config_json = ?,
		       last_observed_generation = ?,
		       last_scanned_at = ?,
		       diagnostics_json = ?,
		       modified_at = ?
		 WHERE source_id = ?`,
		source.DisplayName,
		boolToInt(source.Enabled),
		string(source.ConfigSchemaID),
		[]byte(source.Config),
		nullableSourceGeneration(source.LastObservedGeneration),
		nullableTime(source.LastScannedAt),
		diagnostics,
		formatTime(source.ModifiedAt),
		string(source.SourceID),
	)
	if err != nil {
		return sqliteError(fmt.Errorf("update source: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect source update: %w", err)
	}
	if changed == 0 {
		return fmt.Errorf("%w: source %q", spec.ErrNotFound, source.SourceID)
	}
	return nil
}

func (s *MetadataStore) DeleteSource(ctx context.Context, sourceID spec.SourceID) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM artifact_sources WHERE source_id = ?`, string(sourceID))
	if err != nil {
		return sqliteError(fmt.Errorf("delete source: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect source deletion: %w", err)
	}
	if changed == 0 {
		return fmt.Errorf("%w: source %q", spec.ErrNotFound, sourceID)
	}
	return nil
}

func scanSource(scanner sqlScanner) (spec.ArtifactSource, error) {
	var (
		sourceID       string
		kind           string
		displayName    string
		enabled        int
		configSchemaID string
		config         []byte
		lastGeneration sql.NullString
		lastScannedAt  sql.NullString
		diagnostics    []byte
		createdAt      string
		modifiedAt     string
	)
	if err := scanner.Scan(
		&sourceID,
		&kind,
		&displayName,
		&enabled,
		&configSchemaID,
		&config,
		&lastGeneration,
		&lastScannedAt,
		&diagnostics,
		&createdAt,
		&modifiedAt,
	); err != nil {
		return spec.ArtifactSource{}, err
	}
	created, err := parseRequiredTime("source.createdAt", createdAt)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	modified, err := parseRequiredTime("source.modifiedAt", modifiedAt)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	scanned, err := parseNullableTime("source.lastScannedAt", lastScannedAt)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	decodedDiagnostics, err := decodeDiagnostics(diagnostics)
	if err != nil {
		return spec.ArtifactSource{}, err
	}
	source := spec.ArtifactSource{
		SourceID:               spec.SourceID(sourceID),
		Kind:                   spec.SourceKind(kind),
		DisplayName:            displayName,
		Enabled:                enabled != 0,
		ConfigSchemaID:         spec.SchemaID(configSchemaID),
		Config:                 append([]byte(nil), config...),
		LastObservedGeneration: optionalSourceGeneration(lastGeneration),
		LastScannedAt:          scanned,
		Diagnostics:            decodedDiagnostics,
		CreatedAt:              created,
		ModifiedAt:             modified,
	}
	if err := spec.ValidateArtifactSource(source); err != nil {
		return spec.ArtifactSource{}, fmt.Errorf("invalid persisted source %q: %w", sourceID, err)
	}
	return source, nil
}

func nullableSourceGeneration(value *spec.SourceGeneration) any {
	if value == nil {
		return nil
	}
	return string(*value)
}
