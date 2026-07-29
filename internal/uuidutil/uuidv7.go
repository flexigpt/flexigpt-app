package uuidutil

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
)

var errInvalid = errors.New("invalid uui")

var uuidV7Pattern = regexp.MustCompile(
	`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

// Generator creates identifier strings for domain aggregates.
type Generator interface {
	NewID(ctx context.Context) (string, error)
}

// UUIDv7Generator creates canonical UUIDv7 identifier strings.
type UUIDv7Generator struct{}

// NewID creates one UUIDv7 after checking whether the caller context was
// canceled before generation began.
func (UUIDv7Generator) NewID(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	value, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate UUIDv7: %w", err)
	}
	return value.String(), nil
}

func ValidateUUIDv7(label, value string) error {
	if !uuidV7Pattern.MatchString(value) {
		return fmt.Errorf("%w: %s must be a canonical UUIDv7", errInvalid, label)
	}
	return nil
}
