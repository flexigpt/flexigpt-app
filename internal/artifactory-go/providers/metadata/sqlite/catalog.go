package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const artifactCatalogColumns = `
	a.id,
	a.root_id,
	a.source_id,
	a.locator,
	a.subresource_locator,
	a.kind,
	a.logical_name,
	a.logical_version,
	a.state,
	a.display_name,
	a.enabled,
	a.revision,

	s.kind,
	s.storage_key,
	s.enabled,

	d.digest,
	d.schema_id,
	d.schema_version,
	d.description`

const catalogListFilterSQL = `
	  AND (? = '' OR a.kind = ?)
	  AND (? < 0 OR a.enabled = ?)
	  AND (? = 0 OR a.logical_name IN (
		SELECT value FROM json_each(?)
	  ))`

const listArtifactCatalogByRootSQL = `
	SELECT ` + artifactCatalogColumns + `
	FROM artifact_artifacts a
	JOIN artifact_roots r
	  ON r.id = a.root_id
	JOIN artifact_sources s
	  ON s.root_id = a.root_id
	 AND s.id = a.source_id
	LEFT JOIN artifact_definitions d
	  ON d.root_id = a.root_id
	 AND d.digest = a.resolved_definition_digest
	 AND a.state = 'available'
	WHERE a.root_id = ?
	  AND r.retired_at IS NULL
` + catalogListFilterSQL + `
	ORDER BY a.logical_name, a.root_id, a.id`

const listArtifactCatalogBySourceSQL = `
	SELECT ` + artifactCatalogColumns + `
	FROM artifact_artifacts a
	JOIN artifact_roots r
	  ON r.id = a.root_id
	JOIN artifact_sources s
	  ON s.root_id = a.root_id
	 AND s.id = a.source_id
	LEFT JOIN artifact_definitions d
	  ON d.root_id = a.root_id
	 AND d.digest = a.resolved_definition_digest
	 AND a.state = 'available'
	WHERE a.root_id = ?
	  AND a.source_id = ?
	  AND r.retired_at IS NULL
` + catalogListFilterSQL + `
	ORDER BY a.locator, a.subresource_locator, a.kind, a.id`

const findArtifactCatalogByIdentitySQL = `
	SELECT ` + artifactCatalogColumns + `
	FROM artifact_artifacts a
	JOIN artifact_roots r
	  ON r.id = a.root_id
	JOIN artifact_sources s
	  ON s.root_id = a.root_id
	 AND s.id = a.source_id
	LEFT JOIN artifact_definitions d
	  ON d.root_id = a.root_id
	 AND d.digest = a.resolved_definition_digest
	 AND a.state = 'available'
	WHERE a.root_id = ?
	  AND a.kind = ?
	  AND a.logical_name = ?
	  AND r.retired_at IS NULL
` + catalogListFilterSQL + `
	ORDER BY a.source_id, a.locator, a.subresource_locator, a.id`

const artifactQualifiedColumns = `
	a.id,
	a.root_id,
	a.source_id,
	a.locator,
	a.subresource_locator,
	a.kind,
	a.logical_name,
	a.logical_version,
	a.resolved_definition_digest,
	a.source_content_digest,
	a.state,
	a.diagnostics_json,
	a.display_name,
	a.enabled,
	a.data_json,
	a.revision,
	a.created_at,
	a.modified_at`

const getArtifactsByReferencesSQL = `
	WITH requested AS (
		SELECT
			json_extract(value, '$.rootID') AS root_id,
			json_extract(value, '$.artifactID') AS artifact_id,
			CAST(key AS INTEGER) AS ordinal
		FROM json_each(?)
	)
	SELECT ` + artifactQualifiedColumns + `
	FROM requested q
	JOIN artifact_artifacts a
	  ON a.root_id = q.root_id
	 AND a.id = q.artifact_id
	JOIN artifact_roots r
	  ON r.id = a.root_id
	WHERE r.retired_at IS NULL
	ORDER BY q.ordinal`

const artifactGetManyChunkSize = 256

func (s *Store) listArtifactCatalogByRoot(
	ctx context.Context,
	rootID root.RootID,
	options catalog.ListOptions,
) ([]catalog.Entry, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}
	filter, err := catalogListFilterArguments(options)
	if err != nil {
		return nil, err
	}

	args := append([]any{string(rootID)}, filter...)
	rows, err := s.db.QueryContext(
		ctx,
		listArtifactCatalogByRootSQL,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanArtifactCatalogEntries(rows)
}

func (s *Store) listArtifactCatalogBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	options catalog.ListOptions,
) ([]catalog.Entry, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := sourceID.Validate(); err != nil {
		return nil, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}
	filter, err := catalogListFilterArguments(options)
	if err != nil {
		return nil, err
	}

	args := append(
		[]any{string(rootID), string(sourceID)},
		filter...,
	)
	rows, err := s.db.QueryContext(
		ctx,
		listArtifactCatalogBySourceSQL,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanArtifactCatalogEntries(rows)
}

func (s *Store) findArtifactCatalogByIdentity(
	ctx context.Context,
	rootID root.RootID,
	kind artifact.ArtifactKind,
	logicalName model.LogicalName,
	options catalog.ListOptions,
) ([]catalog.Entry, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := kind.Validate(); err != nil {
		return nil, err
	}
	if err := logicalName.Validate(); err != nil {
		return nil, err
	}
	if err := s.requireActiveRoot(ctx, rootID); err != nil {
		return nil, err
	}
	filter, err := catalogListFilterArguments(options)
	if err != nil {
		return nil, err
	}

	args := append(
		[]any{string(rootID), string(kind), string(logicalName)},
		filter...,
	)
	rows, err := s.db.QueryContext(
		ctx,
		findArtifactCatalogByIdentitySQL,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanArtifactCatalogEntries(rows)
}

func (s *Store) getArtifactsByReferences(
	ctx context.Context,
	refs []artifact.ArtifactRef,
) ([]artifact.Artifact, error) {
	if len(refs) == 0 {
		return []artifact.Artifact{}, nil
	}

	output := make([]artifact.Artifact, 0, len(refs))
	for start := 0; start < len(refs); start += artifactGetManyChunkSize {
		end := min(start+artifactGetManyChunkSize, len(refs))

		raw, err := json.Marshal(refs[start:end])
		if err != nil {
			return nil, err
		}
		rows, err := s.db.QueryContext(
			ctx,
			getArtifactsByReferencesSQL,
			string(raw),
		)
		if err != nil {
			return nil, err
		}

		values, scanErr := scanArtifacts(rows)
		//nolint:sqlclosecheck // Inline.
		closeErr := rows.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(values) != end-start {
			return nil, fmt.Errorf(
				"%w: one or more requested Artifacts are unavailable",
				model.ErrArtifactNotFound,
			)
		}
		output = append(output, values...)
	}
	return output, nil
}

func catalogListFilterArguments(
	options catalog.ListOptions,
) ([]any, error) {
	if options.Kind != "" {
		if err := options.Kind.Validate(); err != nil {
			return nil, err
		}
	}

	names := make(map[string]struct{}, len(options.LogicalNames))
	for _, name := range options.LogicalNames {
		if err := name.Validate(); err != nil {
			return nil, err
		}
		names[string(name)] = struct{}{}
	}
	orderedNames := make([]string, 0, len(names))
	for name := range names {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)
	namesJSON, err := json.Marshal(orderedNames)
	if err != nil {
		return nil, err
	}

	enabledFilter := -1
	enabledValue := 0
	if options.Enabled != nil {
		enabledFilter = 1
		enabledValue = boolInt(*options.Enabled)
	}

	hasNames := 0
	if len(orderedNames) != 0 {
		hasNames = 1
	}

	kind := string(options.Kind)
	return []any{
		kind,
		kind,
		enabledFilter,
		enabledValue,
		hasNames,
		string(namesJSON),
	}, nil
}

func scanArtifactCatalogEntries(
	rows *sql.Rows,
) ([]catalog.Entry, error) {
	output := make([]catalog.Entry, 0)
	for rows.Next() {
		value, err := scanArtifactCatalogEntry(rows)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return output, nil
}

func scanArtifactCatalogEntry(
	row scanner,
) (catalog.Entry, error) {
	var (
		id, rootID, sourceID, locator, subresource string
		kind, logicalName, logicalVersion          string
		state, displayName                         string
		enabled                                    int
		revision                                   uint64

		sourceKind, sourceStorageKey string
		sourceEnabled                int

		definitionDigest, schemaID, schemaVersion sql.NullString
		description                               sql.NullString
	)

	if err := row.Scan(
		&id,
		&rootID,
		&sourceID,
		&locator,
		&subresource,
		&kind,
		&logicalName,
		&logicalVersion,
		&state,
		&displayName,
		&enabled,
		&revision,
		&sourceKind,
		&sourceStorageKey,
		&sourceEnabled,
		&definitionDigest,
		&schemaID,
		&schemaVersion,
		&description,
	); err != nil {
		return catalog.Entry{}, err
	}

	value := catalog.Entry{
		ID:     artifact.ArtifactID(id),
		RootID: root.RootID(rootID),
		Binding: artifact.SourceBinding{
			SourceID:           source.SourceID(sourceID),
			Locator:            model.Locator(locator),
			SubresourceLocator: model.SubresourceLocator(subresource),
		},
		Kind:           artifact.ArtifactKind(kind),
		LogicalName:    model.LogicalName(logicalName),
		LogicalVersion: model.LogicalVersion(logicalVersion),
		DisplayName:    displayName,
		State:          artifact.State(state),
		Enabled:        enabled != 0,
		Revision:       revision,
		Source: catalog.SourceMetadata{
			ID:         source.SourceID(sourceID),
			Kind:       source.SourceKind(sourceKind),
			StorageKey: model.StorageKey(sourceStorageKey),
			Enabled:    sourceEnabled != 0,
		},
	}
	if definitionDigest.Valid {
		value.Definition = &catalog.DefinitionMetadata{
			Digest:        cryptoutil.Digest(definitionDigest.String),
			SchemaID:      schema.SchemaID(schemaID.String),
			SchemaVersion: schemaVersion.String,
			Description:   description.String,
		}
	}
	return value, nil
}
