package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
	rootID rootModel.RootID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
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
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
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
	rootID rootModel.RootID,
	kind artifactModel.ArtifactKind,
	logicalName spec.LogicalName,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
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
	refs []artifactModel.ArtifactRef,
) ([]artifactModel.Artifact, error) {
	if len(refs) == 0 {
		return []artifactModel.Artifact{}, nil
	}

	output := make([]artifactModel.Artifact, 0, len(refs))
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
				spec.ErrArtifactNotFound,
			)
		}
		output = append(output, values...)
	}
	return output, nil
}

func catalogListFilterArguments(
	options catalogModel.ListOptions,
) ([]any, error) {
	names := make(map[string]struct{}, len(options.LogicalNames))
	for _, name := range options.LogicalNames {
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
) ([]catalogModel.Entry, error) {
	output := make([]catalogModel.Entry, 0)
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
) (catalogModel.Entry, error) {
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
		return catalogModel.Entry{}, err
	}

	value := catalogModel.Entry{
		ID:     artifactModel.ArtifactID(id),
		RootID: rootModel.RootID(rootID),
		Binding: artifactModel.SourceBinding{
			SourceID:           sourceModel.SourceID(sourceID),
			Locator:            spec.Locator(locator),
			SubresourceLocator: spec.SubresourceLocator(subresource),
		},
		Kind:           artifactModel.ArtifactKind(kind),
		LogicalName:    spec.LogicalName(logicalName),
		LogicalVersion: spec.LogicalVersion(logicalVersion),
		DisplayName:    displayName,
		State:          artifactModel.State(state),
		Enabled:        enabled != 0,
		Revision:       revision,
		Source: catalogModel.SourceMetadata{
			ID:         sourceModel.SourceID(sourceID),
			Kind:       sourceModel.SourceKind(sourceKind),
			StorageKey: spec.StorageKey(sourceStorageKey),
			Enabled:    sourceEnabled != 0,
		},
	}
	if definitionDigest.Valid {
		value.Definition = &catalogModel.DefinitionMetadata{
			Digest:        cryptoutil.Digest(definitionDigest.String),
			SchemaID:      schemaModel.SchemaID(schemaID.String),
			SchemaVersion: schemaVersion.String,
			Description:   description.String,
		}
	}
	return value, nil
}
