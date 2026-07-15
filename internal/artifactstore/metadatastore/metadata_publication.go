package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *MetadataStore) PublishSourceCatalog(ctx context.Context, publication spec.SourceCatalogPublication) error {
	if publication.SourceID == "" ||
		publication.ExpectedSourceModifiedAt.IsZero() ||
		publication.ObservedGeneration == "" ||
		publication.ObservedAt.IsZero() {
		return fmt.Errorf(
			"%w: source catalog publication requires source, expected version, generation, and observation time",
			spec.ErrInvalidRequest,
		)
	}
	if err := spec.ValidateDiagnostics(publication.Diagnostics); err != nil {
		return err
	}
	diagnostics, err := encodeDiagnostics(publication.Diagnostics)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `
		UPDATE artifact_sources
		   SET last_observed_generation = ?, last_scanned_at = ?, diagnostics_json = ?, modified_at = ?
		 WHERE source_id = ? AND modified_at = ?`,
		string(publication.ObservedGeneration), formatTime(publication.ObservedAt), diagnostics,
		formatTime(publication.ObservedAt), string(publication.SourceID),
		formatTime(publication.ExpectedSourceModifiedAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	changed, err := result.RowsAffected()
	if err != nil || changed == 0 {
		if err != nil {
			return err
		}
		return fmt.Errorf(
			"%w: source %q changed while catalog publication was pending",
			spec.ErrConflict,
			publication.SourceID,
		)
	}
	seen := make(map[string]struct{}, len(publication.Resources))
	for _, resource := range publication.Resources {
		if resource.SourceID != publication.SourceID {
			return fmt.Errorf("%w: catalog resource source mismatch", spec.ErrInvalidRequest)
		}
		if err := spec.ValidateCatalogResource(resource); err != nil {
			return err
		}
		key := string(resource.Locator) + "\x00" + string(resource.SubresourceLocator)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: duplicate catalog resource %q", spec.ErrInvalidRequest, key)
		}
		seen[key] = struct{}{}
		if err := upsertCatalogResourceTx(ctx, tx, resource); err != nil {
			return err
		}
	}
	for _, revision := range publication.Revisions {
		if revision.SourceID != publication.SourceID {
			return fmt.Errorf("%w: catalog revision source mismatch", spec.ErrInvalidRequest)
		}
		if err := spec.ValidateCatalogResourceRevision(revision); err != nil {
			return err
		}
		if err := upsertCatalogRevisionTx(ctx, tx, revision); err != nil {
			return err
		}
	}
	if publication.Authoritative {
		rows, err := tx.QueryContext(
			ctx,
			`SELECT locator, subresource_locator FROM catalog_resources WHERE source_id = ?`,
			string(publication.SourceID),
		)
		if err != nil {
			return err
		}
		missing := make([]spec.CatalogResourceKey, 0)
		for rows.Next() {
			var locator, subresource string
			if err := rows.Scan(&locator, &subresource); err != nil {
				//nolint:sqlclosecheck // Closing before return.
				_ = rows.Close()
				return err
			}
			if _, ok := seen[locator+"\x00"+subresource]; ok {
				continue
			}
			missing = append(missing, spec.CatalogResourceKey{
				SourceID:           publication.SourceID,
				Locator:            spec.SourceLocator(locator),
				SubresourceLocator: spec.SubresourceLocator(subresource),
			})
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		missingDiagnostics, err := encodeDiagnostics([]spec.Diagnostic{})
		if err != nil {
			return err
		}
		for _, key := range missing {
			if _, err := tx.ExecContext(
				ctx,
				`UPDATE catalog_resources
				    SET state = ?, diagnostics_json = ?
				  WHERE source_id = ? AND locator = ? AND subresource_locator = ?`,
				string(spec.CatalogStateMissing),
				missingDiagnostics,
				string(key.SourceID),
				string(key.Locator),
				string(key.SubresourceLocator),
			); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *MetadataStore) ListCatalogResourcesForRoot(
	ctx context.Context,
	rootID spec.RootID,
) ([]spec.CatalogResource, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+catalogResourceColumns+`
		  FROM catalog_resources c
		  JOIN root_source_attachments a ON a.source_id = c.source_id
		  JOIN artifact_sources src ON src.source_id = c.source_id
		 WHERE a.root_id = ? AND a.enabled = 1 AND src.enabled = 1
		 ORDER BY c.source_id, c.locator, c.subresource_locator`, string(rootID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resources := make([]spec.CatalogResource, 0)
	for rows.Next() {
		resource, err := scanCatalogResource(rows)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, rows.Err()
}

func (s *MetadataStore) PublishRootCatalogGeneration(
	ctx context.Context,
	publication spec.RootCatalogPublication,
) (spec.RootCatalogGeneration, error) {
	if publication.RootID == "" || publication.CreatedAt.IsZero() {
		return spec.RootCatalogGeneration{}, fmt.Errorf(
			"%w: root catalog publication is incomplete",
			spec.ErrInvalidRequest,
		)
	}
	sourceGenerations, err := encodeSourceGenerations(publication.SourceGenerations)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	diagnostics, err := encodeDiagnostics(publication.Diagnostics)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var generation uint64
	if err := tx.QueryRowContext(
		ctx,
		`INSERT INTO root_catalog_generation_counters (root_id, generation)
		 VALUES (?, 1)
		 ON CONFLICT (root_id) DO UPDATE SET
			generation = root_catalog_generation_counters.generation + 1
		 RETURNING generation`,
		string(publication.RootID),
	).
		Scan(&generation); err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	result := spec.RootCatalogGeneration{
		RootID:            publication.RootID,
		Generation:        generation,
		SourceGenerations: publication.SourceGenerations,
		ScanPlanDigest:    publication.ScanPlanDigest,
		CatalogDigest:     publication.CatalogDigest,
		CreatedAt:         publication.CreatedAt,
		Diagnostics:       publication.Diagnostics,
	}
	if err := spec.ValidateRootCatalogGeneration(result); err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO root_catalog_generations (root_id, generation, source_generations_json, scan_plan_digest, catalog_digest, created_at, diagnostics_json) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(result.RootID),
		result.Generation,
		sourceGenerations,
		string(result.ScanPlanDigest),
		string(result.CatalogDigest),
		formatTime(result.CreatedAt),
		diagnostics,
	); err != nil {
		return spec.RootCatalogGeneration{}, sqliteError(err)
	}
	if err := tx.Commit(); err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	return result, nil
}

func (s *MetadataStore) GetRootCatalogGeneration(
	ctx context.Context,
	rootID spec.RootID,
) (spec.RootCatalogGeneration, error) {
	var sourceGenerations, diagnostics []byte
	var generation uint64
	var scanPlanDigest, catalogDigest, createdAt string
	err := s.db.QueryRowContext(ctx, `SELECT generation, source_generations_json, scan_plan_digest, catalog_digest, created_at, diagnostics_json FROM root_catalog_generations WHERE root_id = ? ORDER BY generation DESC LIMIT 1`, string(rootID)).
		Scan(&generation, &sourceGenerations, &scanPlanDigest, &catalogDigest, &createdAt, &diagnostics)
	if errors.Is(err, sql.ErrNoRows) {
		return spec.RootCatalogGeneration{}, fmt.Errorf("%w: root catalog generation %q", spec.ErrNotFound, rootID)
	}
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	generations, err := decodeSourceGenerations(sourceGenerations)
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
		RootID:            rootID,
		Generation:        generation,
		SourceGenerations: generations,
		ScanPlanDigest:    spec.Digest(scanPlanDigest),
		CatalogDigest:     spec.Digest(catalogDigest),
		CreatedAt:         created,
		Diagnostics:       decodedDiagnostics,
	}
	if err := spec.ValidateRootCatalogGeneration(result); err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	return result, nil
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
