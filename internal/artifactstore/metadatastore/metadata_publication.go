package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

func (s *MetadataStore) GetRootCatalogGeneration(
	ctx context.Context,
	rootID spec.RootID,
) (spec.RootCatalogGeneration, error) {
	var sourceVersions, diagnostics []byte
	var generation, rootRevision uint64
	var scanPlanDigest, catalogDigest, createdAt string
	err := s.db.QueryRowContext(ctx, `SELECT generation, root_revision, source_versions_json, scan_plan_digest, catalog_digest, created_at, diagnostics_json FROM root_catalog_generations WHERE root_id = ? ORDER BY generation DESC LIMIT 1`, string(rootID)).
		Scan(
			&generation,
			&rootRevision,
			&sourceVersions,
			&scanPlanDigest,
			&catalogDigest,
			&createdAt,
			&diagnostics,
		)
	if errors.Is(err, sql.ErrNoRows) {
		return spec.RootCatalogGeneration{}, fmt.Errorf("%w: root catalog generation %q", spec.ErrNotFound, rootID)
	}
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	versions, err := decodeSourceVersions(sourceVersions)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	decodedDiagnostics, err := decodeDiagnostics(diagnostics)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	created, err := parseRequiredTime("root catalog generation.createdAt", createdAt)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	result := spec.RootCatalogGeneration{
		RootID:         rootID,
		Generation:     generation,
		RootRevision:   rootRevision,
		SourceVersions: versions,
		ScanPlanDigest: spec.Digest(scanPlanDigest),
		CatalogDigest:  spec.Digest(catalogDigest),
		CreatedAt:      created,
		Diagnostics:    decodedDiagnostics,
	}
	if err := validate.ValidateRootCatalogGeneration(result); err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	return result, nil
}

func ensureRootCatalogCurrentTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID spec.RootID,
	generation uint64,
) error {
	var rootRevision uint64
	var sourceVersionsJSON []byte
	err := tx.QueryRowContext(
		ctx,
		`SELECT root_revision, source_versions_json
		   FROM root_catalog_generations
		  WHERE root_id = ? AND generation = ?`,
		string(rootID),
		generation,
	).Scan(&rootRevision, &sourceVersionsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: root catalog generation no longer exists", spec.ErrConflict)
	}
	if err != nil {
		return err
	}
	expected, err := decodeSourceVersions(sourceVersionsJSON)
	if err != nil {
		return err
	}

	var enabled int
	var softDeleted sql.NullString
	var currentRootRevision uint64
	err = tx.QueryRowContext(
		ctx,
		`SELECT enabled, soft_deleted_at, mount_revision
		   FROM artifact_roots
		  WHERE root_id = ?`,
		string(rootID),
	).Scan(&enabled, &softDeleted, &currentRootRevision)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: root no longer exists", spec.ErrConflict)
	}
	if err != nil {
		return err
	}
	if enabled == 0 || softDeleted.Valid || currentRootRevision != rootRevision {
		return fmt.Errorf("%w: root mount changed after catalog publication", spec.ErrConflict)
	}

	rows, err := tx.QueryContext(
		ctx,
		`SELECT s.source_id, s.last_observed_generation, s.observation_revision
		   FROM root_source_attachments a
		   JOIN artifact_sources s ON s.source_id = a.source_id
		  WHERE a.root_id = ? AND a.enabled = 1 AND s.enabled = 1
		  ORDER BY s.source_id`,
		string(rootID),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	current := make(map[spec.SourceID]spec.SourceCatalogVersion)
	for rows.Next() {
		var sourceID string
		var sourceGeneration sql.NullString
		var observationRevision uint64
		if err := rows.Scan(&sourceID, &sourceGeneration, &observationRevision); err != nil {
			return err
		}
		if !sourceGeneration.Valid || sourceGeneration.String == "" {
			return fmt.Errorf("%w: active source is no longer observed", spec.ErrConflict)
		}
		current[spec.SourceID(sourceID)] = spec.SourceCatalogVersion{
			Generation:          spec.SourceGeneration(sourceGeneration.String),
			ObservationRevision: observationRevision,
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !maps.Equal(current, expected) {
		return fmt.Errorf("%w: source catalogs changed after root catalog publication", spec.ErrConflict)
	}
	return nil
}

func upsertCatalogResourceTx(ctx context.Context, tx *sql.Tx, resource spec.CatalogResource) error {
	diagnostics, err := encodeDiagnostics(resource.Diagnostics)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO catalog_resources (
			source_id, locator, subresource_locator, package_manifest_locator,
			kind, logical_name, logical_version, current_definition_digest,
			source_content_digest, frontend_id, state, first_seen_at, last_seen_at,
			diagnostics_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (source_id, locator, subresource_locator) DO UPDATE SET
			package_manifest_locator = CASE
				WHEN excluded.package_manifest_locator <> '' THEN excluded.package_manifest_locator
				ELSE catalog_resources.package_manifest_locator
			END,
			kind = CASE
				WHEN excluded.kind <> '' THEN excluded.kind
				ELSE catalog_resources.kind
			END,
			logical_name = CASE
				WHEN excluded.logical_name <> '' THEN excluded.logical_name
				ELSE catalog_resources.logical_name
			END,
			logical_version = CASE
				WHEN excluded.logical_version <> '' THEN excluded.logical_version
				ELSE catalog_resources.logical_version
			END,
			current_definition_digest = COALESCE(
				excluded.current_definition_digest,
				catalog_resources.current_definition_digest
			),
			source_content_digest = COALESCE(
				excluded.source_content_digest,
				catalog_resources.source_content_digest
			),
			frontend_id = CASE
				WHEN excluded.frontend_id <> '' THEN excluded.frontend_id
				ELSE catalog_resources.frontend_id
			END,
			state = excluded.state,
			last_seen_at = excluded.last_seen_at,
			diagnostics_json = excluded.diagnostics_json`,
		string(resource.SourceID),
		string(resource.Locator),
		string(resource.SubresourceLocator),
		string(resource.PackageManifestLocator),
		string(resource.Kind),
		string(resource.LogicalName),
		string(resource.LogicalVersion),
		nullableDigest(resource.CurrentDefinitionDigest),
		nullableDigest(resource.SourceContentDigest),
		string(resource.FrontendID),
		string(resource.State),
		formatTime(resource.FirstSeenAt),
		formatTime(resource.LastSeenAt),
		diagnostics,
	)
	return sqliteError(err)
}

func upsertCatalogRevisionTx(ctx context.Context, tx *sql.Tx, revision spec.CatalogResourceRevision) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO catalog_resource_revisions (source_id, locator, subresource_locator, definition_digest, source_content_digest, kind, frontend_id, first_seen_at, last_seen_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (source_id, locator, subresource_locator, definition_digest) DO UPDATE SET source_content_digest = excluded.source_content_digest, kind = excluded.kind, frontend_id = excluded.frontend_id, last_seen_at = excluded.last_seen_at`,
		string(revision.SourceID),
		string(revision.Locator),
		string(revision.SubresourceLocator),
		string(revision.DefinitionDigest),
		string(revision.SourceContentDigest),
		string(revision.Kind),
		string(revision.FrontendID),
		formatTime(revision.FirstSeenAt),
		formatTime(revision.LastSeenAt),
	)
	return sqliteError(err)
}
