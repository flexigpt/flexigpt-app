package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const definitionColumns = `
	root_id, digest, kind, schema_id, schema_version,
	logical_name, logical_version, display_name, description,
	labels_json, body_json, dependencies_json, created_at`

func (s *Store) getDefinition(
	ctx context.Context,
	rootID root.RootID,
	digest cryptoutil.Digest,
) (definition.Definition, error) {
	values, err := s.getDefinitions(
		ctx,
		[]definition.Key{{
			RootID: rootID,
			Digest: digest,
		}},
	)
	if err != nil {
		return definition.Definition{}, err
	}
	return values[0], nil
}

func putDefinitionTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID root.RootID,
	value definition.Definition,
	createdAt time.Time,
) error {
	canonical, err := definition.Canonicalize(value)
	if err != nil {
		return err
	}
	if canonical.Digest != value.Digest {
		return fmt.Errorf(
			"%w: Definition is not canonical",
			model.ErrInvalid,
		)
	}

	labels, err := encodeJSON(canonical.Labels)
	if err != nil {
		return err
	}
	dependencies, err := encodeJSON(canonical.Dependencies)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO artifact_definitions (
			root_id, digest, kind, schema_id, schema_version,
			logical_name, logical_version, display_name, description,
			labels_json, body_json, dependencies_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(root_id, digest) DO NOTHING`,
		string(rootID),
		string(canonical.Digest),
		string(canonical.Kind),
		string(canonical.SchemaID),
		canonical.SchemaVersion,
		string(canonical.LogicalName),
		string(canonical.LogicalVersion),
		canonical.DisplayName,
		canonical.Description,
		labels,
		[]byte(canonical.Body),
		dependencies,
		timeValue(createdAt),
	)
	if err != nil {
		return sqliteError(err)
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 1 {
		// The exact admitted payload was inserted by this transaction.
		return nil
	}

	existing, err := getDefinitionTx(
		ctx,
		tx,
		rootID,
		canonical.Digest,
	)
	if err != nil {
		return err
	}
	if !equalDefinitions(existing, canonical) {
		return fmt.Errorf(
			"%w: Definition digest %q conflicts with existing immutable Definition",
			model.ErrDigestMismatch,
			canonical.Digest,
		)
	}
	return nil
}

type definitionQueryer interface {
	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

func getDefinitionTx(
	ctx context.Context,
	queryer definitionQueryer,
	rootID root.RootID,
	digest cryptoutil.Digest,
) (definition.Definition, error) {
	return scanDefinition(queryer.QueryRowContext(
		ctx,
		`SELECT `+definitionColumns+`
		 FROM artifact_definitions
		 WHERE root_id = ? AND digest = ?`,
		string(rootID),
		string(digest),
	))
}

func scanDefinition(
	row scanner,
) (definition.Definition, error) {
	var (
		rootID, digest, kind, schemaID, schemaVersion string
		logicalName, logicalVersion                   string
		displayName, description                      string
		labelsRaw, bodyRaw, dependenciesRaw           []byte
		createdAt                                     int64
	)
	if row == nil {
		return definition.Definition{}, fmt.Errorf(
			"%w: Definition row is nil",
			model.ErrInvalid,
		)
	}
	if err := row.Scan(
		&rootID,
		&digest,
		&kind,
		&schemaID,
		&schemaVersion,
		&logicalName,
		&logicalVersion,
		&displayName,
		&description,
		&labelsRaw,
		&bodyRaw,
		&dependenciesRaw,
		&createdAt,
	); err != nil {
		return definition.Definition{}, err
	}

	var value definition.Definition
	if err := decodeJSON(labelsRaw, &value.Labels); err != nil {
		return definition.Definition{}, err
	}
	if err := decodeJSON(
		dependenciesRaw,
		&value.Dependencies,
	); err != nil {
		return definition.Definition{}, err
	}
	value.Digest = cryptoutil.Digest(digest)
	value.Kind = artifact.ArtifactKind(kind)
	value.SchemaID = schema.SchemaID(schemaID)
	value.SchemaVersion = schemaVersion
	value.LogicalName = model.LogicalName(logicalName)
	value.LogicalVersion = model.LogicalVersion(logicalVersion)
	value.DisplayName = displayName
	value.Description = description
	value.Body = append([]byte(nil), bodyRaw...)
	canonical, err := definition.Canonicalize(value)
	if err != nil {
		return definition.Definition{}, fmt.Errorf(
			"invalid persisted Definition %q/%q: %w",
			rootID,
			digest,
			err,
		)
	}
	return canonical, nil
}

func equalDefinitions(
	left definition.Definition,
	right definition.Definition,
) bool {
	return left.Digest == right.Digest &&
		left.Kind == right.Kind &&
		left.SchemaID == right.SchemaID &&
		left.SchemaVersion == right.SchemaVersion &&
		left.LogicalName == right.LogicalName &&
		left.LogicalVersion == right.LogicalVersion &&
		left.DisplayName == right.DisplayName &&
		left.Description == right.Description &&
		reflect.DeepEqual(left.Labels, right.Labels) &&
		bytes.Equal(left.Body, right.Body) &&
		reflect.DeepEqual(left.Dependencies, right.Dependencies)
}
