package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const definitionColumns = `
	root_id, digest, kind, schema_id, schema_version,
	logical_name, logical_version, display_name, description,
	labels_json, body_json, dependencies_json, created_at`

func (s *Store) getDefinition(
	ctx context.Context,
	rootID rootModel.RootID,
	digest cryptoutil.Digest,
) (definitionModel.Definition, error) {
	values, err := s.getDefinitions(ctx, []definitionModel.Key{{RootID: rootID, Digest: digest}})
	if err != nil {
		return definitionModel.Definition{}, err
	}
	return values[0], nil
}

// putDefinitionTx receives a Definition already admitted through
// definition.Admit/Ingress. SQLite enforces immutable-key conflict semantics;
// it deliberately does not canonicalize and hash the same body again.
func putDefinitionTx(
	ctx context.Context,
	tx *sql.Tx,
	rootID rootModel.RootID,
	value definitionModel.Definition,
	createdAt time.Time,
) error {
	canonical := value.Clone()
	if err := definitionModel.ValidateAdmitted(canonical); err != nil {
		return err
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
		string(rootID), string(canonical.Digest), string(canonical.Kind),
		string(canonical.SchemaID), canonical.SchemaVersion,
		string(canonical.LogicalName), string(canonical.LogicalVersion),
		canonical.DisplayName, canonical.Description, labels,
		[]byte(canonical.Body), dependencies, timeValue(createdAt),
	)
	if err != nil {
		return sqliteError(err)
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 1 {
		return nil
	}
	existing, err := getDefinitionTx(ctx, tx, rootID, canonical.Digest)
	if err != nil {
		return err
	}
	if !equalDefinitions(existing, canonical) {
		return fmt.Errorf(
			"%w: Definition digest %q conflicts with existing immutable Definition",
			spec.ErrDigestMismatch,
			canonical.Digest,
		)
	}
	return nil
}

type definitionQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func getDefinitionTx(
	ctx context.Context,
	queryer definitionQueryer,
	rootID rootModel.RootID,
	digest cryptoutil.Digest,
) (definitionModel.Definition, error) {
	return scanDefinition(
		queryer.QueryRowContext(
			ctx,
			`SELECT `+definitionColumns+` FROM artifact_definitions WHERE root_id = ? AND digest = ?`,
			string(rootID),
			string(digest),
		),
	)
}

func scanDefinition(row scanner) (definitionModel.Definition, error) {
	var rootID, digest, kind, schemaID, schemaVersion string
	var logicalName, logicalVersion string
	var displayName, description string
	var labelsRaw, bodyRaw, dependenciesRaw []byte
	var createdAt int64
	if row == nil {
		return definitionModel.Definition{}, fmt.Errorf("%w: Definition row is nil", spec.ErrInvalid)
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
		return definitionModel.Definition{}, err
	}
	var value definitionModel.Definition
	if err := decodeJSON(labelsRaw, &value.Labels); err != nil {
		return definitionModel.Definition{}, err
	}
	if err := decodeJSON(dependenciesRaw, &value.Dependencies); err != nil {
		return definitionModel.Definition{}, err
	}
	value.Digest = cryptoutil.Digest(digest)
	value.Kind = artifactModel.ArtifactKind(kind)
	value.SchemaID = schemaModel.SchemaID(schemaID)
	value.SchemaVersion = schemaVersion
	value.LogicalName = spec.LogicalName(logicalName)
	value.LogicalVersion = spec.LogicalVersion(logicalVersion)
	value.DisplayName = displayName
	value.Description = description
	value.Body = append([]byte(nil), bodyRaw...)
	if err := definitionModel.ValidateAdmitted(value); err != nil {
		return definitionModel.Definition{}, fmt.Errorf("invalid persisted Definition %q/%q: %w", rootID, digest, err)
	}
	return value, nil
}

func equalDefinitions(left, right definitionModel.Definition) bool {
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
