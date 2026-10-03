package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func (s *Store) GetTopologyHydration(
	ctx context.Context,
	installerName string,
) (topology.Hydration, bool, error) {
	if s == nil || s.db == nil {
		return topology.Hydration{}, false, spec.ErrClosed
	}
	if ctx == nil {
		return topology.Hydration{}, false, fmt.Errorf(
			"%w: topology hydration context is nil",
			spec.ErrInvalid,
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
		RootID:        rootModel.RootID(rootID),
		SourceID:      sourceModel.SourceID(sourceID),
		Fingerprint:   cryptoutil.Digest(fingerprint),
	}
	if err := value.Validate(); err != nil {
		return topology.Hydration{}, false, fmt.Errorf(
			"%w: invalid persisted topology hydration: %w",
			spec.ErrInvalid,
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
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology hydration context is nil",
			spec.ErrInvalid,
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

func (s *Store) ListTopologyPackageHydrations(
	ctx context.Context,
) ([]topology.PackageHydration, error) {
	if s == nil || s.db == nil {
		return nil, spec.ErrClosed
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT installer_name, package_scope, root_id, source_id, fingerprint
		 FROM artifact_topology_package_hydrations
		 ORDER BY installer_name, package_scope`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	output := make([]topology.PackageHydration, 0)
	for rows.Next() {
		var installerName, scope, rootID, sourceID, fingerprint string
		if err := rows.Scan(
			&installerName,
			&scope,
			&rootID,
			&sourceID,
			&fingerprint,
		); err != nil {
			return nil, err
		}
		value := topology.PackageHydration{
			Key: topology.PackageHydrationKey{
				InstallerName: installerName,
				Scope:         spec.Locator(scope),
			},
			RootID:      rootModel.RootID(rootID),
			SourceID:    sourceModel.SourceID(sourceID),
			Fingerprint: cryptoutil.Digest(fingerprint),
		}
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf(
				"%w: invalid persisted package hydration: %w",
				spec.ErrInvalid,
				err,
			)
		}
		output = append(output, value)
	}
	return output, rows.Err()
}

func (s *Store) PutTopologyPackageHydration(
	ctx context.Context,
	value topology.PackageHydration,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO artifact_topology_package_hydrations (
			installer_name, package_scope, root_id, source_id, fingerprint, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(installer_name, package_scope) DO UPDATE SET
			root_id = excluded.root_id,
			source_id = excluded.source_id,
			fingerprint = excluded.fingerprint,
			updated_at = excluded.updated_at`,
		value.Key.InstallerName,
		string(value.Key.Scope),
		string(value.RootID),
		string(value.SourceID),
		string(value.Fingerprint),
		time.Now().UTC().UnixNano(),
	)
	return sqliteError(err)
}

func (s *Store) DeleteTopologyPackageHydration(
	ctx context.Context,
	key topology.PackageHydrationKey,
) error {
	if err := key.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		ctx,
		`DELETE FROM artifact_topology_package_hydrations
		 WHERE installer_name = ? AND package_scope = ?`,
		key.InstallerName,
		string(key.Scope),
	)
	return sqliteError(err)
}

// PurgeTopologyRoot removes all metadata owned by one application-owned Root.
//
// This deliberately bypasses ordinary lifecycle transitions. It is only
// called by the trusted startup hydration path after authorization has been
// established by assembly.Components.ResetTopologyHydration.
func (s *Store) PurgeTopologyRoot(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	if s == nil || s.db == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: topology purge context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rootID.Validate(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := purgeRootLocalStateTx(
		ctx,
		tx,
		rootID,
		time.Now().UTC(),
	); err != nil {
		return err
	}

	statements := []string{
		`DELETE FROM artifact_topology_package_hydrations WHERE root_id = ?`,
		`DELETE FROM artifact_source_refresh_state WHERE root_id = ?`,
		`DELETE FROM artifact_artifacts WHERE root_id = ?`,
		`DELETE FROM artifact_definitions WHERE root_id = ?`,
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
