package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func (s *Store) GetTopologyHydration(
	ctx context.Context,
	installerName string,
) (topology.Hydration, bool, error) {
	if s == nil || s.db == nil {
		return topology.Hydration{}, false, basespec.ErrClosed
	}
	if ctx == nil {
		return topology.Hydration{}, false, fmt.Errorf(
			"%w: topology hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return topology.Hydration{}, false, err
	}
	if err := topology.ValidateHydrationInstallerName(installerName); err != nil {
		return topology.Hydration{}, false, err
	}

	var rootID, sourceID, fingerprint string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT root_id, source_id, fingerprint
		 FROM artifact_topology_hydrations
		 WHERE installer_name = ?`,
		installerName,
	).Scan(&rootID, &sourceID, &fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return topology.Hydration{}, false, nil
	}
	if err != nil {
		return topology.Hydration{}, false, err
	}

	value := topology.Hydration{
		InstallerName: installerName,
		RootID:        basespec.RootID(rootID),
		SourceID:      basespec.SourceID(sourceID),
		Fingerprint:   cryptoutil.Digest(fingerprint),
	}
	if err := value.Validate(); err != nil {
		return topology.Hydration{}, false, fmt.Errorf(
			"%w: invalid persisted topology hydration: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	return value, true, nil
}

func (s *Store) PutTopologyHydration(
	ctx context.Context,
	value topology.Hydration,
) error {
	if s == nil || s.db == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO artifact_topology_hydrations (
			installer_name, root_id, source_id, fingerprint, updated_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(installer_name) DO UPDATE SET
			root_id = excluded.root_id,
			source_id = excluded.source_id,
			fingerprint = excluded.fingerprint,
			updated_at = excluded.updated_at`,
		value.InstallerName,
		string(value.RootID),
		string(value.SourceID),
		string(value.Fingerprint),
		time.Now().UTC().UnixNano(),
	)
	return sqliteError(err)
}

// PurgeTopologyRoot removes all metadata owned by one application-owned Root.
//
// This deliberately bypasses ordinary lifecycle transitions. It is only
// called by the trusted startup hydration path after authorization has been
// established by system.Components.ResetTopologyHydration.
func (s *Store) PurgeTopologyRoot(
	ctx context.Context,
	rootID basespec.RootID,
) error {
	if s == nil || s.db == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology purge context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	statements := []string{
		`DELETE FROM artifact_artifacts WHERE root_id = ?`,
		`DELETE FROM artifact_suppressions WHERE root_id = ?`,
		`DELETE FROM artifact_current_occurrences WHERE root_id = ?`,
		`DELETE FROM artifact_current_catalogs WHERE root_id = ?`,
		`DELETE FROM artifact_collection_attachments WHERE root_id = ?`,
		`DELETE FROM artifact_collections WHERE root_id = ?`,
		`DELETE FROM artifact_sources WHERE root_id = ?`,
		`DELETE FROM artifact_roots WHERE id = ?`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement, string(rootID)); err != nil {
			return sqliteError(err)
		}
	}

	// Do not remove artifact_topology_hydrations here. The previous record is
	// the authorization and retry marker until the new hydration succeeds.
	return sqliteError(tx.Commit())
}
