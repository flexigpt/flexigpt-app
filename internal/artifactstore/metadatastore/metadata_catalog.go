package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

const catalogResourceColumns = `
	source_id,
	locator,
	subresource_locator,
	package_manifest_locator,
	kind,
	logical_name,
	logical_version,
	current_definition_digest,
	source_content_digest,
	frontend_id,
	state,
	first_seen_at,
	last_seen_at,
	diagnostics_json`

const catalogResourceRevisionColumns = `
	source_id,
	locator,
	subresource_locator,
	definition_digest,
	source_content_digest,
	kind,
	frontend_id,
	first_seen_at,
	last_seen_at`

func (s *MetadataStore) GetCatalogResource(
	ctx context.Context,
	key spec.CatalogResourceKey,
) (spec.CatalogResource, error) {
	if err := spec.ValidateCatalogResourceKey(key); err != nil {
		return spec.CatalogResource{}, fmt.Errorf("validate catalog resource key: %w", err)
	}
	resource, err := scanCatalogResource(s.db.QueryRowContext(
		ctx,
		`SELECT `+catalogResourceColumns+`
		   FROM catalog_resources
		  WHERE source_id = ? AND locator = ? AND subresource_locator = ?`,
		string(key.SourceID),
		string(key.Locator),
		string(key.SubresourceLocator),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return spec.CatalogResource{}, fmt.Errorf(
			"%w: catalog resource %q/%q/%q",
			spec.ErrNotFound,
			key.SourceID,
			key.Locator,
			key.SubresourceLocator,
		)
	}
	if err != nil {
		return spec.CatalogResource{}, err
	}
	return resource, nil
}

func (s *MetadataStore) ListCatalogResourcesForSource(
	ctx context.Context,
	sourceID spec.SourceID,
) ([]spec.CatalogResource, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+catalogResourceColumns+`
		   FROM catalog_resources
		  WHERE source_id = ?
		  ORDER BY locator ASC, subresource_locator ASC`,
		string(sourceID),
	)
	if err != nil {
		return nil, fmt.Errorf("list catalog resources: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog resources: %w", err)
	}
	return resources, nil
}

func (s *MetadataStore) UpsertCatalogResource(ctx context.Context, resource spec.CatalogResource) error {
	if err := spec.ValidateCatalogResource(resource); err != nil {
		return fmt.Errorf("validate catalog resource for persistence: %w", err)
	}
	diagnostics, err := encodeDiagnostics(resource.Diagnostics)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO catalog_resources (
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
	if err != nil {
		return sqliteError(fmt.Errorf("upsert catalog resource: %w", err))
	}
	return nil
}

func (s *MetadataStore) UpsertCatalogResourceRevision(
	ctx context.Context,
	revision spec.CatalogResourceRevision,
) error {
	if err := spec.ValidateCatalogResourceRevision(revision); err != nil {
		return fmt.Errorf("validate catalog resource revision for persistence: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO catalog_resource_revisions (
			source_id, locator, subresource_locator, definition_digest,
			source_content_digest, kind, frontend_id, first_seen_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (source_id, locator, subresource_locator, definition_digest) DO UPDATE SET
			source_content_digest = excluded.source_content_digest,
			kind = excluded.kind,
			frontend_id = excluded.frontend_id,
			last_seen_at = excluded.last_seen_at`,
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
	if err != nil {
		return sqliteError(fmt.Errorf("upsert catalog resource revision: %w", err))
	}
	return nil
}

func (s *MetadataStore) ListCatalogResourceRevisions(
	ctx context.Context,
	key spec.CatalogResourceKey,
) ([]spec.CatalogResourceRevision, error) {
	if err := spec.ValidateCatalogResourceKey(key); err != nil {
		return nil, fmt.Errorf("validate catalog resource key: %w", err)
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+catalogResourceRevisionColumns+`
		   FROM catalog_resource_revisions
		  WHERE source_id = ? AND locator = ? AND subresource_locator = ?
		  ORDER BY last_seen_at DESC, definition_digest ASC`,
		string(key.SourceID),
		string(key.Locator),
		string(key.SubresourceLocator),
	)
	if err != nil {
		return nil, fmt.Errorf("list catalog resource revisions: %w", err)
	}
	defer rows.Close()

	revisions := make([]spec.CatalogResourceRevision, 0)
	for rows.Next() {
		revision, err := scanCatalogResourceRevision(rows)
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog resource revisions: %w", err)
	}
	return revisions, nil
}

func scanCatalogResource(scanner sqlScanner) (spec.CatalogResource, error) {
	row := catalogResourceRow{}
	if err := scanner.Scan(row.destinations()...); err != nil {
		return spec.CatalogResource{}, err
	}
	firstSeen, err := parseRequiredTime("catalog resource.firstSeenAt", row.FirstSeenAt)
	if err != nil {
		return spec.CatalogResource{}, err
	}
	lastSeen, err := parseRequiredTime("catalog resource.lastSeenAt", row.LastSeenAt)
	if err != nil {
		return spec.CatalogResource{}, err
	}
	decodedDiagnostics, err := decodeDiagnostics(row.Diagnostics)
	if err != nil {
		return spec.CatalogResource{}, err
	}
	resource := spec.CatalogResource{
		SourceID:                spec.SourceID(row.SourceID),
		Locator:                 spec.SourceLocator(row.Locator),
		SubresourceLocator:      spec.SubresourceLocator(row.SubresourceLocator),
		PackageManifestLocator:  spec.SourceLocator(row.PackageManifestLocator),
		Kind:                    spec.ArtifactKind(row.Kind),
		LogicalName:             spec.LogicalName(row.LogicalName),
		LogicalVersion:          spec.LogicalVersion(row.LogicalVersion),
		CurrentDefinitionDigest: optionalDigest(row.CurrentDefinitionDigest),
		SourceContentDigest:     optionalDigest(row.SourceContentDigest),
		FrontendID:              spec.FrontendID(row.FrontendID),
		State:                   spec.CatalogState(row.State),
		FirstSeenAt:             firstSeen,
		LastSeenAt:              lastSeen,
		Diagnostics:             decodedDiagnostics,
	}
	if err := spec.ValidateCatalogResource(resource); err != nil {
		return spec.CatalogResource{}, fmt.Errorf(
			"invalid persisted catalog resource %q/%q/%q: %w",
			row.SourceID,
			row.Locator,
			row.SubresourceLocator,
			err,
		)
	}
	return resource, nil
}

func scanCatalogResourceRevision(scanner sqlScanner) (spec.CatalogResourceRevision, error) {
	row := catalogResourceRevisionRow{}
	if err := scanner.Scan(row.destinations()...); err != nil {
		return spec.CatalogResourceRevision{}, err
	}
	firstSeen, err := parseRequiredTime("catalog revision.firstSeenAt", row.FirstSeenAt)
	if err != nil {
		return spec.CatalogResourceRevision{}, err
	}
	lastSeen, err := parseRequiredTime("catalog revision.lastSeenAt", row.LastSeenAt)
	if err != nil {
		return spec.CatalogResourceRevision{}, err
	}
	revision := spec.CatalogResourceRevision{
		SourceID:            spec.SourceID(row.SourceID),
		Locator:             spec.SourceLocator(row.Locator),
		SubresourceLocator:  spec.SubresourceLocator(row.SubresourceLocator),
		DefinitionDigest:    spec.Digest(row.DefinitionDigest),
		SourceContentDigest: spec.Digest(row.SourceContentDigest),
		Kind:                spec.ArtifactKind(row.Kind),
		FrontendID:          spec.FrontendID(row.FrontendID),
		FirstSeenAt:         firstSeen,
		LastSeenAt:          lastSeen,
	}
	if err := spec.ValidateCatalogResourceRevision(revision); err != nil {
		return spec.CatalogResourceRevision{}, fmt.Errorf(
			"invalid persisted catalog resource revision %q/%q/%q/%q: %w",
			row.SourceID,
			row.Locator,
			row.SubresourceLocator,
			row.DefinitionDigest,
			err,
		)
	}
	return revision, nil
}
