package metadatastore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

type sqlScanner interface {
	Scan(destinations ...any) error
}

func encodeDiagnostics(value []spec.Diagnostic) ([]byte, error) {
	if value == nil {
		value = []spec.Diagnostic{}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode diagnostics: %w", err)
	}
	return encoded, nil
}

func decodeDiagnostics(raw []byte) ([]spec.Diagnostic, error) {
	if len(raw) == 0 {
		return []spec.Diagnostic{}, nil
	}
	var value []spec.Diagnostic
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode diagnostics: %w", err)
	}
	if err := spec.ValidateDiagnostics(value); err != nil {
		return nil, fmt.Errorf("validate persisted diagnostics: %w", err)
	}
	return value, nil
}

func encodeSourceGenerations(value map[spec.SourceID]spec.SourceGeneration) ([]byte, error) {
	if value == nil {
		value = map[spec.SourceID]spec.SourceGeneration{}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode source generations: %w", err)
	}
	return encoded, nil
}

func decodeSourceGenerations(raw []byte) (map[spec.SourceID]spec.SourceGeneration, error) {
	if len(raw) == 0 {
		return map[spec.SourceID]spec.SourceGeneration{}, nil
	}
	var value map[spec.SourceID]spec.SourceGeneration
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode source generations: %w", err)
	}
	if value == nil {
		value = map[spec.SourceID]spec.SourceGeneration{}
	}
	return value, nil
}

func encodeCatalogResourceKey(value *spec.CatalogResourceKey) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode catalog resource key: %w", err)
	}
	return encoded, nil
}

func decodeCatalogResourceKey(raw []byte) (*spec.CatalogResourceKey, error) {
	if len(raw) == 0 {
		//nolint:nilnil // Nil nil required.
		return nil, nil
	}

	value := spec.CatalogResourceKey{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode catalog resource key: %w", err)
	}
	if err := spec.ValidateCatalogResourceKey(value); err != nil {
		return nil, fmt.Errorf("validate persisted catalog resource key: %w", err)
	}
	return &value, nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return formatTime(*value)
}

func parseNullableTime(label string, value sql.NullString) (*time.Time, error) {
	if !value.Valid || value.String == "" {
		//nolint:nilnil // Nil nil required.
		return nil, nil
	}
	parsed, err := parseRequiredTime(label, value.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseRequiredTime(label, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s: %w", label, err)
	}
	return parsed.UTC(), nil
}

func nullableDigest(value *spec.Digest) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func nullableID[T ~string](value *T) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func optionalDigest(value sql.NullString) *spec.Digest {
	if !value.Valid || value.String == "" {
		return nil
	}
	result := spec.Digest(value.String)
	return &result
}

func optionalSourceGeneration(value sql.NullString) *spec.SourceGeneration {
	if !value.Valid || value.String == "" {
		return nil
	}
	result := spec.SourceGeneration(value.String)
	return &result
}

func optionalCollectionID(value sql.NullString) *spec.CollectionID {
	if !value.Valid || value.String == "" {
		return nil
	}
	result := spec.CollectionID(value.String)
	return &result
}

func optionalRecordID(value sql.NullString) *spec.RecordID {
	if !value.Valid || value.String == "" {
		return nil
	}
	result := spec.RecordID(value.String)
	return &result
}

func sqliteError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed"):
		return fmt.Errorf("%w: %w", spec.ErrConflict, err)
	case strings.Contains(message, "foreign key constraint failed"):
		return fmt.Errorf("%w: related app-local metadata exists or is missing", spec.ErrConflict)
	default:
		return err
	}
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
