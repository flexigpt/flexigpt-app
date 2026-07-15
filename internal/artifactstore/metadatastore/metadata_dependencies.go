package metadatastore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

const (
	selectDependencyPublicationGenerationSQL = `SELECT generation
		FROM root_catalog_generations
		WHERE root_id = ?
		ORDER BY generation DESC
		LIMIT 1`
	selectDependencyPublicationRecordSQL = `SELECT
			root_id,
			modified_at,
			last_resolved_definition_digest
		FROM artifact_records
		WHERE record_id = ?`
	deleteDependencySnapshotSetSQL = `DELETE FROM artifact_dependencies
		WHERE root_id = ?
		  AND record_id = ?
		  AND catalog_generation = ?
		  AND root_definition_digest = ?`
	insertDependencySnapshotSQL = `INSERT INTO artifact_dependencies (
			root_id,
			record_id,
			catalog_generation,
			root_definition_digest,
			definition_digest,
			selector_index,
			selector_json,
			state,
			candidates_json,
			diagnostics_json,
			modified_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	selectDependencySnapshotsSQL = `SELECT
			root_id,
			record_id,
			catalog_generation,
			root_definition_digest,
			definition_digest,
			selector_index,
			selector_json,
			state,
			candidates_json,
			diagnostics_json,
			modified_at
		FROM artifact_dependencies
		WHERE record_id = ?
		ORDER BY
			catalog_generation DESC,
			root_definition_digest,
			definition_digest,
			selector_index`
)

func (s *MetadataStore) ReplaceDependencySnapshots(
	ctx context.Context,
	publication spec.DependencySnapshotPublication,
) error {
	if publication.RootID == "" ||
		publication.RecordID == "" ||
		publication.RootDefinitionDigest == "" ||
		publication.CatalogGeneration == 0 ||
		publication.ExpectedRecordModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: dependency snapshot publication is incomplete",
			spec.ErrInvalidRequest,
		)
	}
	for _, snapshot := range publication.Snapshots {
		if snapshot.RootID != publication.RootID ||
			snapshot.RecordID != publication.RecordID ||
			snapshot.RootDefinitionDigest != publication.RootDefinitionDigest ||
			snapshot.CatalogGeneration != publication.CatalogGeneration {
			return fmt.Errorf(
				"%w: inconsistent dependency snapshot publication",
				spec.ErrInvalidRequest,
			)
		}
		if err := validate.ValidateArtifactDependencySnapshot(snapshot); err != nil {
			return err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin dependency snapshot publication: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentGeneration uint64
	err = tx.QueryRowContext(
		ctx,
		selectDependencyPublicationGenerationSQL,
		string(publication.RootID),
	).Scan(&currentGeneration)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"%w: root %q has no published catalog",
			spec.ErrConflict,
			publication.RootID,
		)
	}
	if err != nil {
		return err
	}
	if currentGeneration != publication.CatalogGeneration {
		return fmt.Errorf(
			"%w: root catalog changed during dependency resolution",
			spec.ErrConflict,
		)
	}

	var recordRootID, recordModifiedAt string
	var recordDigest sql.NullString
	err = tx.QueryRowContext(
		ctx,
		selectDependencyPublicationRecordSQL,
		string(publication.RecordID),
	).Scan(&recordRootID, &recordModifiedAt, &recordDigest)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"%w: dependency root record no longer exists",
			spec.ErrConflict,
		)
	}
	if err != nil {
		return fmt.Errorf("read dependency publication record: %w", err)
	}
	if recordRootID != string(publication.RootID) ||
		recordModifiedAt != formatTime(publication.ExpectedRecordModifiedAt) ||
		!recordDigest.Valid ||
		recordDigest.String != string(publication.RootDefinitionDigest) {
		return fmt.Errorf(
			"%w: dependency root record changed during dependency resolution",
			spec.ErrConflict,
		)
	}

	if _, err := tx.ExecContext(
		ctx,
		deleteDependencySnapshotSetSQL,
		string(publication.RootID),
		string(publication.RecordID),
		publication.CatalogGeneration,
		string(publication.RootDefinitionDigest),
	); err != nil {
		return sqliteError(fmt.Errorf("delete dependency snapshot set: %w", err))
	}

	for _, snapshot := range publication.Snapshots {
		selectorJSON, err := json.Marshal(snapshot.Selector)
		if err != nil {
			return fmt.Errorf("encode dependency selector: %w", err)
		}
		candidatesJSON, err := json.Marshal(snapshot.Candidates)
		if err != nil {
			return fmt.Errorf("encode dependency candidates: %w", err)
		}
		diagnosticsJSON, err := encodeDiagnostics(snapshot.Diagnostics)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			insertDependencySnapshotSQL,
			string(snapshot.RootID),
			string(snapshot.RecordID),
			snapshot.CatalogGeneration,
			string(snapshot.RootDefinitionDigest),
			string(snapshot.DefinitionDigest),
			snapshot.SelectorIndex,
			selectorJSON,
			string(snapshot.State),
			candidatesJSON,
			diagnosticsJSON,
			formatTime(snapshot.ModifiedAt),
		); err != nil {
			return sqliteError(fmt.Errorf("insert dependency snapshot: %w", err))
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit dependency snapshot publication: %w", err)
	}
	return nil
}

func (s *MetadataStore) ListDependencySnapshots(
	ctx context.Context,
	recordID spec.RecordID,
) ([]spec.ArtifactDependencySnapshot, error) {
	rows, err := s.db.QueryContext(
		ctx,
		selectDependencySnapshotsSQL,
		string(recordID),
	)
	if err != nil {
		return nil, fmt.Errorf("list dependency snapshots: %w", err)
	}
	defer rows.Close()

	out := make([]spec.ArtifactDependencySnapshot, 0)
	for rows.Next() {
		row := artifactDependencyRow{}
		if err := rows.Scan(row.destinations()...); err != nil {
			return nil, err
		}
		modifiedAt, err := parseRequiredTime("dependency.modifiedAt", row.ModifiedAt)
		if err != nil {
			return nil, err
		}
		var selector spec.ArtifactSelector
		if err := json.Unmarshal(row.Selector, &selector); err != nil {
			return nil, fmt.Errorf("decode dependency selector: %w", err)
		}
		var candidates []spec.DependencyCandidateRef
		if err := json.Unmarshal(row.Candidates, &candidates); err != nil {
			return nil, fmt.Errorf("decode dependency candidates: %w", err)
		}
		if candidates == nil {
			candidates = []spec.DependencyCandidateRef{}
		}
		diagnostics, err := decodeDiagnostics(row.Diagnostics)
		if err != nil {
			return nil, err
		}
		snapshot := spec.ArtifactDependencySnapshot{
			RootID:               spec.RootID(row.RootID),
			RecordID:             spec.RecordID(row.RecordID),
			CatalogGeneration:    row.CatalogGeneration,
			RootDefinitionDigest: spec.Digest(row.RootDefinitionDigest),
			DefinitionDigest:     spec.Digest(row.DefinitionDigest),
			SelectorIndex:        row.SelectorIndex,
			Selector:             selector,
			State:                spec.DependencyResolutionState(row.State),
			Candidates:           candidates,
			Diagnostics:          diagnostics,
			ModifiedAt:           modifiedAt,
		}
		if err := validate.ValidateArtifactDependencySnapshot(snapshot); err != nil {
			return nil, fmt.Errorf(
				"invalid persisted dependency snapshot for record %q: %w",
				recordID,
				err,
			)
		}
		out = append(out, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dependency snapshots: %w", err)
	}
	return out, nil
}
