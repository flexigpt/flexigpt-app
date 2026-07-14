package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *MetadataStore) GetArtifactPackage(
	ctx context.Context,
	sourceID spec.SourceID,
	locator spec.SourceLocator,
) (spec.ArtifactPackage, error) {
	value, err := scanArtifactPackage(
		s.db.QueryRowContext(
			ctx,
			`SELECT source_id, manifest_locator, name, version, display_name, description, current_manifest_digest, state, diagnostics_json, first_seen_at, last_seen_at FROM artifact_packages WHERE source_id = ? AND manifest_locator = ?`,
			string(sourceID),
			string(locator),
		),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return spec.ArtifactPackage{}, fmt.Errorf("%w: package %q/%q", spec.ErrNotFound, sourceID, locator)
	}
	return value, err
}

func (s *MetadataStore) ListArtifactPackagesForSource(
	ctx context.Context,
	sourceID spec.SourceID,
) ([]spec.ArtifactPackage, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT source_id, manifest_locator, name, version, display_name, description, current_manifest_digest, state, diagnostics_json, first_seen_at, last_seen_at FROM artifact_packages WHERE source_id = ? ORDER BY manifest_locator`,
		string(sourceID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []spec.ArtifactPackage{}
	for rows.Next() {
		value, err := scanArtifactPackage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (s *MetadataStore) UpsertArtifactPackage(ctx context.Context, value spec.ArtifactPackage) error {
	if err := spec.ValidateArtifactPackage(value); err != nil {
		return err
	}
	diagnostics, err := encodeDiagnostics(value.Diagnostics)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO artifact_packages (source_id, manifest_locator, name, version, display_name, description, current_manifest_digest, state, diagnostics_json, first_seen_at, last_seen_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (source_id, manifest_locator) DO UPDATE SET name = excluded.name, version = excluded.version, display_name = excluded.display_name, description = excluded.description, current_manifest_digest = excluded.current_manifest_digest, state = excluded.state, diagnostics_json = excluded.diagnostics_json, last_seen_at = excluded.last_seen_at`,
		string(value.SourceID),
		string(value.ManifestLocator),
		string(value.Name),
		string(value.Version),
		value.DisplayName,
		value.Description,
		nullableDigest(value.CurrentManifestDigest),
		string(value.State),
		diagnostics,
		formatTime(value.FirstSeenAt),
		formatTime(value.LastSeenAt),
	)
	return sqliteError(err)
}

func scanArtifactPackage(scanner sqlScanner) (spec.ArtifactPackage, error) {
	var sourceID, locator, name, version, displayName, description, state, firstRaw, lastRaw string
	var digest sql.NullString
	var diagnostics []byte
	if err := scanner.Scan(
		&sourceID,
		&locator,
		&name,
		&version,
		&displayName,
		&description,
		&digest,
		&state,
		&diagnostics,
		&firstRaw,
		&lastRaw,
	); err != nil {
		return spec.ArtifactPackage{}, err
	}
	first, err := parseRequiredTime("package.firstSeenAt", firstRaw)
	if err != nil {
		return spec.ArtifactPackage{}, err
	}
	last, err := parseRequiredTime("package.lastSeenAt", lastRaw)
	if err != nil {
		return spec.ArtifactPackage{}, err
	}
	decoded, err := decodeDiagnostics(diagnostics)
	if err != nil {
		return spec.ArtifactPackage{}, err
	}
	value := spec.ArtifactPackage{
		SourceID:              spec.SourceID(sourceID),
		ManifestLocator:       spec.SourceLocator(locator),
		Name:                  spec.LogicalName(name),
		Version:               spec.LogicalVersion(version),
		DisplayName:           displayName,
		Description:           description,
		CurrentManifestDigest: optionalDigest(digest),
		State:                 spec.CatalogState(state),
		Diagnostics:           decoded,
		FirstSeenAt:           first,
		LastSeenAt:            last,
	}
	if err := spec.ValidateArtifactPackage(value); err != nil {
		return spec.ArtifactPackage{}, err
	}
	return value, nil
}
