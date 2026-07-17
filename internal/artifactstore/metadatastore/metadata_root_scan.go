package metadatastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

const (
	selectRootScanRootSQL = `SELECT enabled, soft_deleted_at, mount_revision
		FROM artifact_roots
		WHERE root_id = ?`
	selectRootScanAttachmentsSQL = `SELECT source_id, enabled
		FROM root_source_attachments
		WHERE root_id = ?
		ORDER BY source_id`
	selectRootScanSourceSQL = `SELECT observation_revision, enabled
		FROM artifact_sources
		WHERE source_id = ?`
	updateRootScanSourceObservationSQL = `UPDATE artifact_sources
		SET last_observed_generation = ?,
		    last_scanned_at = ?,
		    observation_revision = observation_revision + ?,
		    diagnostics_json = ?,
		    modified_at = CASE
				WHEN modified_at > ? THEN modified_at
				ELSE ?
			END
		WHERE source_id = ?
		  AND observation_revision = ?
		  AND enabled = 1`
	selectCurrentRootCatalogSQL = `SELECT
		c.source_id,
		c.locator,
		c.subresource_locator,
		c.package_manifest_locator,
		c.kind,
		c.logical_name,
		c.logical_version,
		c.current_definition_digest,
		c.source_content_digest,
		c.frontend_id,
		c.state,
		c.first_seen_at,
		c.last_seen_at,
		c.diagnostics_json
		FROM catalog_resources c
		JOIN root_source_attachments a
		  ON a.source_id = c.source_id
		JOIN artifact_sources s
		  ON s.source_id = c.source_id
		WHERE a.root_id = ?
		  AND a.enabled = 1
		  AND s.enabled = 1
		ORDER BY c.source_id, c.locator, c.subresource_locator`
	incrementRootCatalogGenerationSQL = `INSERT INTO root_catalog_generation_counters (
			root_id, generation
		) VALUES (?, 1)
		ON CONFLICT (root_id) DO UPDATE SET
			generation = root_catalog_generation_counters.generation + 1
		RETURNING generation`
	insertRootCatalogGenerationSQL = `INSERT INTO root_catalog_generations (
			root_id,
			generation,
			root_revision,
			source_versions_json,
			scan_plan_digest,
			catalog_digest,
			created_at,
			diagnostics_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	insertRootCatalogResourceSnapshotSQL = `INSERT INTO root_catalog_resource_snapshots (
			root_id,
			generation,
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
			diagnostics_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	selectPublishedRootCatalogSQL = `SELECT
		r.source_id,
		r.locator,
		r.subresource_locator,
		r.package_manifest_locator,
		r.kind,
		r.logical_name,
		r.logical_version,
		r.current_definition_digest,
		r.source_content_digest,
		r.frontend_id,
		r.state,
		r.first_seen_at,
		r.last_seen_at,
		r.diagnostics_json
		FROM root_catalog_resource_snapshots r
		JOIN root_catalog_generation_counters g
		  ON g.root_id = r.root_id
		 AND g.generation = r.generation
		WHERE r.root_id = ?
		ORDER BY r.source_id, r.locator, r.subresource_locator`
	selectRootScanCurrentGenerationSQL = `SELECT
			last_observed_generation,
			observation_revision
		FROM artifact_sources
		WHERE source_id = ?`
)

// PublishRootScan commits all source observations and the resulting root
// generation in one SQLite transaction.
func (s *MetadataStore) PublishRootScan(
	ctx context.Context,
	publication spec.RootScanPublication,
) (spec.RootCatalogGeneration, error) {
	if publication.RootCatalog.RootID == "" ||
		publication.ExpectedRootRevision == 0 ||
		publication.RootCatalog.RootRevision != publication.ExpectedRootRevision ||
		publication.RootCatalog.CreatedAt.IsZero() {
		return spec.RootCatalogGeneration{}, fmt.Errorf(
			"%w: root scan publication is incomplete",
			spec.ErrInvalidRequest,
		)
	}
	if err := validate.ValidateDiagnostics(publication.RootCatalog.Diagnostics); err != nil {
		return spec.RootCatalogGeneration{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return spec.RootCatalogGeneration{}, fmt.Errorf("begin root scan publication: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	attachments, sources, err := validateRootScanExpectations(ctx, tx, publication)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}

	publishedSources := make(map[spec.SourceID]struct{}, len(publication.SourceCatalogs))
	for _, sourcePublication := range publication.SourceCatalogs {
		if _, exists := publishedSources[sourcePublication.SourceID]; exists {
			return spec.RootCatalogGeneration{}, fmt.Errorf(
				"%w: duplicate source publication %q",
				spec.ErrInvalidRequest,
				sourcePublication.SourceID,
			)
		}
		publishedSources[sourcePublication.SourceID] = struct{}{}

		attachment, attached := attachments[sourcePublication.SourceID]
		source, knownSource := sources[sourcePublication.SourceID]
		if !attached || !knownSource || !attachment.Enabled || !source.Enabled {
			return spec.RootCatalogGeneration{}, fmt.Errorf(
				"%w: source %q is not actively attached to root %q",
				spec.ErrSourceNotAttached,
				sourcePublication.SourceID,
				publication.RootCatalog.RootID,
			)
		}
		if sourcePublication.ExpectedObservationRevision != source.ObservationRevision {
			return spec.RootCatalogGeneration{}, fmt.Errorf(
				"%w: source %q publication expectation does not match the scan expectation",
				spec.ErrInvalidRequest,
				sourcePublication.SourceID,
			)
		}
		if err := publishRootScanSourceCatalog(ctx, tx, sourcePublication); err != nil {
			return spec.RootCatalogGeneration{}, err
		}
	}

	if err := validateRootSourceVersions(
		ctx,
		tx,
		publication.RootCatalog.SourceVersions,
		attachments,
		sources,
	); err != nil {
		return spec.RootCatalogGeneration{}, err
	}

	currentResources, err := listCurrentRootCatalogTx(
		ctx,
		tx,
		publication.RootCatalog.RootID,
	)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	if !equivalentCatalogResources(currentResources, publication.CatalogResources) {
		return spec.RootCatalogGeneration{}, fmt.Errorf(
			"%w: calculated root catalog differs from the post-publication source catalog",
			spec.ErrConflict,
		)
	}

	sourceVersions, err := encodeSourceVersions(publication.RootCatalog.SourceVersions)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	diagnostics, err := encodeDiagnostics(publication.RootCatalog.Diagnostics)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}

	var generation uint64
	if err := tx.QueryRowContext(
		ctx,
		incrementRootCatalogGenerationSQL,
		string(publication.RootCatalog.RootID),
	).Scan(&generation); err != nil {
		return spec.RootCatalogGeneration{}, fmt.Errorf(
			"allocate root catalog generation: %w",
			err,
		)
	}

	result := spec.RootCatalogGeneration{
		RootID:         publication.RootCatalog.RootID,
		Generation:     generation,
		RootRevision:   publication.RootCatalog.RootRevision,
		SourceVersions: publication.RootCatalog.SourceVersions,
		ScanPlanDigest: publication.RootCatalog.ScanPlanDigest,
		CatalogDigest:  publication.RootCatalog.CatalogDigest,
		CreatedAt:      publication.RootCatalog.CreatedAt,
		Diagnostics:    publication.RootCatalog.Diagnostics,
	}
	if err := validate.ValidateRootCatalogGeneration(result); err != nil {
		return spec.RootCatalogGeneration{}, err
	}

	if _, err := tx.ExecContext(
		ctx,
		insertRootCatalogGenerationSQL,
		string(result.RootID),
		result.Generation,
		result.RootRevision,
		sourceVersions,
		string(result.ScanPlanDigest),
		string(result.CatalogDigest),
		formatTime(result.CreatedAt),
		diagnostics,
	); err != nil {
		return spec.RootCatalogGeneration{}, sqliteError(
			fmt.Errorf("insert root catalog generation: %w", err),
		)
	}

	for _, resource := range currentResources {
		encodedDiagnostics, err := encodeDiagnostics(resource.Diagnostics)
		if err != nil {
			return spec.RootCatalogGeneration{}, err
		}
		if _, err := tx.ExecContext(
			ctx,
			insertRootCatalogResourceSnapshotSQL,
			string(result.RootID),
			result.Generation,
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
			encodedDiagnostics,
		); err != nil {
			return spec.RootCatalogGeneration{}, sqliteError(
				fmt.Errorf("insert root catalog resource snapshot: %w", err),
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return spec.RootCatalogGeneration{}, fmt.Errorf(
			"commit root scan publication: %w",
			err,
		)
	}
	return result, nil
}

// ListPublishedCatalogResourcesForRoot returns the immutable snapshot belonging
// to the root's latest published generation.
func (s *MetadataStore) ListPublishedCatalogResourcesForRoot(
	ctx context.Context,
	rootID spec.RootID,
) ([]spec.CatalogResource, error) {
	rows, err := s.db.QueryContext(ctx, selectPublishedRootCatalogSQL, string(rootID))
	if err != nil {
		return nil, fmt.Errorf("list published root catalog: %w", err)
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
		return nil, fmt.Errorf("iterate published root catalog: %w", err)
	}
	return resources, nil
}

type rootScanAttachmentState struct {
	Enabled bool
}

type rootScanSourceState struct {
	ObservationRevision uint64
	Enabled             bool
}

func validateRootScanExpectations(
	ctx context.Context,
	tx *sql.Tx,
	publication spec.RootScanPublication,
) (
	attachments map[spec.SourceID]rootScanAttachmentState,
	sources map[spec.SourceID]rootScanSourceState,
	err error,
) {
	var enabled int
	var softDeletedAt sql.NullString
	var mountRevision uint64
	err = tx.QueryRowContext(
		ctx,
		selectRootScanRootSQL,
		string(publication.RootCatalog.RootID),
	).Scan(&enabled, &softDeletedAt, &mountRevision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, fmt.Errorf(
			"%w: root %q",
			spec.ErrNotFound,
			publication.RootCatalog.RootID,
		)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("read root scan root: %w", err)
	}
	if enabled == 0 || softDeletedAt.Valid {
		return nil, nil, fmt.Errorf(
			"%w: root %q is not active",
			spec.ErrConflict,
			publication.RootCatalog.RootID,
		)
	}
	if mountRevision != publication.ExpectedRootRevision {
		return nil, nil, fmt.Errorf(
			"%w: root %q changed while it was being scanned",
			spec.ErrConflict,
			publication.RootCatalog.RootID,
		)
	}

	rows, err := tx.QueryContext(
		ctx,
		selectRootScanAttachmentsSQL,
		string(publication.RootCatalog.RootID),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("read root scan attachments: %w", err)
	}
	attachments = make(map[spec.SourceID]rootScanAttachmentState)
	for rows.Next() {
		var sourceID string
		var attachmentEnabled int
		if err := rows.Scan(&sourceID, &attachmentEnabled); err != nil {
			//nolint:sqlclosecheck // Closing before return.
			_ = rows.Close()
			return nil, nil, err
		}
		id := spec.SourceID(sourceID)
		attachments[id] = rootScanAttachmentState{
			Enabled: attachmentEnabled != 0,
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}

	expectedSources := make(
		map[spec.SourceID]spec.RootScanSourceExpectation,
		len(publication.Sources),
	)
	for _, expected := range publication.Sources {
		if expected.SourceID == "" {
			return nil, nil, fmt.Errorf(
				"%w: invalid root scan source expectation",
				spec.ErrInvalidRequest,
			)
		}
		if _, attached := attachments[expected.SourceID]; !attached {
			return nil, nil, fmt.Errorf(
				"%w: source expectation %q is not attached",
				spec.ErrInvalidRequest,
				expected.SourceID,
			)
		}
		if _, duplicate := expectedSources[expected.SourceID]; duplicate {
			return nil, nil, fmt.Errorf(
				"%w: duplicate source expectation %q",
				spec.ErrInvalidRequest,
				expected.SourceID,
			)
		}
		expectedSources[expected.SourceID] = expected
	}
	if len(expectedSources) != len(attachments) {
		return nil, nil, fmt.Errorf(
			"%w: every attachment requires one source expectation",
			spec.ErrInvalidRequest,
		)
	}

	sources = make(map[spec.SourceID]rootScanSourceState, len(expectedSources))
	for sourceID, expected := range expectedSources {
		var observationRevision uint64
		var sourceEnabled int
		err := tx.QueryRowContext(
			ctx,
			selectRootScanSourceSQL,
			string(sourceID),
		).Scan(&observationRevision, &sourceEnabled)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, fmt.Errorf("%w: source %q", spec.ErrNotFound, sourceID)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read root scan source %q: %w", sourceID, err)
		}
		if observationRevision != expected.ObservationRevision ||
			(sourceEnabled != 0) != expected.Enabled {
			return nil, nil, fmt.Errorf(
				"%w: source %q changed while the root was being scanned",
				spec.ErrConflict,
				sourceID,
			)
		}
		sources[sourceID] = rootScanSourceState{
			ObservationRevision: observationRevision,
			Enabled:             sourceEnabled != 0,
		}
	}
	return attachments, sources, nil
}

func publishRootScanSourceCatalog(
	ctx context.Context,
	tx *sql.Tx,
	publication spec.SourceCatalogPublication,
) error {
	if publication.SourceID == "" ||
		publication.ObservedGeneration == "" ||
		publication.ObservedAt.IsZero() {
		return fmt.Errorf(
			"%w: source catalog publication is incomplete",
			spec.ErrInvalidRequest,
		)
	}
	if publication.AdvanceObservationRevision &&
		publication.ExpectedObservationRevision >= spec.MaxObservationRevision {
		return fmt.Errorf(
			"%w: source %q observation revision is exhausted",
			spec.ErrConflict,
			publication.SourceID,
		)
	}
	if err := validate.ValidateDiagnostics(publication.Diagnostics); err != nil {
		return err
	}
	diagnostics, err := encodeDiagnostics(publication.Diagnostics)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		updateRootScanSourceObservationSQL,
		string(publication.ObservedGeneration),
		formatTime(publication.ObservedAt),
		boolToInt(publication.AdvanceObservationRevision),
		diagnostics,
		formatTime(publication.ObservedAt),
		formatTime(publication.ObservedAt),
		string(publication.SourceID),
		publication.ExpectedObservationRevision,
	)
	if err != nil {
		return sqliteError(fmt.Errorf("publish source observation: %w", err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
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
		if err := validate.ValidateCatalogResource(resource); err != nil {
			return err
		}
		key := string(resource.Locator) + "\x00" + string(resource.SubresourceLocator)
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate catalog resource %q",
				spec.ErrInvalidRequest,
				key,
			)
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
		if err := validate.ValidateCatalogResourceRevision(revision); err != nil {
			return err
		}
		if err := upsertCatalogRevisionTx(ctx, tx, revision); err != nil {
			return err
		}
	}
	return nil
}

func validateRootSourceVersions(
	ctx context.Context,
	tx *sql.Tx,
	versions map[spec.SourceID]spec.SourceCatalogVersion,
	attachments map[spec.SourceID]rootScanAttachmentState,
	sources map[spec.SourceID]rootScanSourceState,
) error {
	expectedCount := 0
	for sourceID, attachment := range attachments {
		source := sources[sourceID]
		if !attachment.Enabled || !source.Enabled {
			if _, exists := versions[sourceID]; exists {
				return fmt.Errorf(
					"%w: disabled source %q appears in source generations",
					spec.ErrInvalidRequest,
					sourceID,
				)
			}
			continue
		}

		var generation sql.NullString
		var observationRevision uint64
		if err := tx.QueryRowContext(
			ctx,
			selectRootScanCurrentGenerationSQL,
			string(sourceID),
		).Scan(&generation, &observationRevision); err != nil {
			return err
		}
		if !generation.Valid || generation.String == "" {
			if _, exists := versions[sourceID]; exists {
				return fmt.Errorf(
					"%w: unobserved source %q has a supplied generation",
					spec.ErrInvalidRequest,
					sourceID,
				)
			}
			continue
		}
		expectedCount++
		supplied, exists := versions[sourceID]
		if !exists ||
			string(supplied.Generation) != generation.String ||
			supplied.ObservationRevision != observationRevision {
			return fmt.Errorf(
				"%w: source catalog version for %q changed during publication",
				spec.ErrConflict,
				sourceID,
			)
		}
	}
	if len(versions) != expectedCount {
		return fmt.Errorf(
			"%w: source generations contain a source outside the active root attachments",
			spec.ErrInvalidRequest,
		)
	}
	return nil
}

func listCurrentRootCatalogTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID spec.RootID,
) ([]spec.CatalogResource, error) {
	rows, err := tx.QueryContext(ctx, selectCurrentRootCatalogSQL, string(rootID))
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return resources, nil
}

func equivalentCatalogResources(left, right []spec.CatalogResource) bool {
	leftCopy := append([]spec.CatalogResource(nil), left...)
	rightCopy := append([]spec.CatalogResource(nil), right...)
	sortCatalogResources(leftCopy)
	sortCatalogResources(rightCopy)
	if len(leftCopy) != len(rightCopy) {
		return false
	}
	for index := range leftCopy {
		l := leftCopy[index]
		r := rightCopy[index]
		if l.Diagnostics == nil {
			l.Diagnostics = []spec.Diagnostic{}
		}
		if r.Diagnostics == nil {
			r.Diagnostics = []spec.Diagnostic{}
		}
		if !reflect.DeepEqual(l, r) {
			return false
		}
	}
	return true
}

func sortCatalogResources(resources []spec.CatalogResource) {
	sort.Slice(resources, func(left, right int) bool {
		if resources[left].SourceID != resources[right].SourceID {
			return resources[left].SourceID < resources[right].SourceID
		}
		if resources[left].Locator != resources[right].Locator {
			return resources[left].Locator < resources[right].Locator
		}
		return resources[left].SubresourceLocator < resources[right].SubresourceLocator
	})
}
