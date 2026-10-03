// Package value contains the pluggable physical secret-value store
// contract used by Artifact Store.
//
// Implementations store only secret values. Artifact references, SHA-256
// metadata, slots, namespaces, lifecycle state, and cleanup state belong in
// Artifact Store metadata repositories.
package value

import (
	"context"
	"fmt"

	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ValueStore persists actual secret values by opaque Artifact Store refs.
//
// Put must create a new value for a previously unused ref. It must not
// overwrite a value already stored under that ref.
type ValueStore interface {
	Name() string

	Put(
		ctx context.Context,
		ref secretModel.Ref,
		value string,
	) error

	Get(
		ctx context.Context,
		ref secretModel.Ref,
	) (string, error)

	// Delete must be idempotent. Deleting an already absent ref returns nil.
	Delete(
		ctx context.Context,
		ref secretModel.Ref,
	) error

	// Close must be safe to call more than once.
	Close() error
}

func ValidateValueStore(value ValueStore) error {
	if value == nil {
		return fmt.Errorf(
			"%w: secret value store is nil",
			spec.ErrInvalid,
		)
	}
	return spec.ValidateIdentifier(
		"secret value store name",
		value.Name(),
		spec.MaxKindBytes,
	)
}
