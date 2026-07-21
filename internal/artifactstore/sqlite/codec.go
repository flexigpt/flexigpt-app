package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return timeValue(*value)
}

func timeValue(value time.Time) int64 {
	return value.UTC().UnixNano()
}

func parseNullableTime(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed := parseTime(value.Int64)
	return &parsed
}

func parseTime(value int64) time.Time {
	return time.Unix(0, value).UTC()
}

func nullableDigest(value *artifactstore.Digest) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func parseDigest(value sql.NullString) *artifactstore.Digest {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed := artifactstore.Digest(value.String)
	return &parsed
}

func encodeJSON(value any) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func decodeJSON(raw []byte, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func sqliteError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed"):
		return fmt.Errorf("%w: metadata already exists", artifactstore.ErrConflict)
	case strings.Contains(message, "foreign key constraint failed"):
		return fmt.Errorf("%w: related metadata is missing or still referenced", artifactstore.ErrConflict)
	case strings.Contains(message, "database is locked"),
		strings.Contains(message, "database is busy"),
		strings.Contains(message, "sqlite_busy"),
		strings.Contains(message, "sqlite_locked"):
		return fmt.Errorf("%w: metadata database is busy", artifactstore.ErrConflict)
	default:
		return err
	}
}
