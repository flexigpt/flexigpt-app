package definition

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Reader interface {
	Get(
		ctx context.Context,
		rootID artifactstore.RootID,
		digest artifactstore.Digest,
	) (Definition, error)
}

type Writer interface {
	Put(
		ctx context.Context,
		rootID artifactstore.RootID,
		value Definition,
	) (Definition, error)
}

type Repository interface {
	Reader
	Writer
}

// ReadCanonical reads a definition through a Reader and establishes the
// ownership and integrity guarantees required by consumers.
//
// The returned definition is canonical, independently owned, valid, and has
// the digest requested by the caller.
func ReadCanonical(
	ctx context.Context,
	reader Reader,
	rootID artifactstore.RootID,
	digest artifactstore.Digest,
) (Definition, error) {
	if ctx == nil {
		return Definition{}, fmt.Errorf(
			"%w: definition read context is nil",
			artifactstore.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return Definition{}, err
	}
	if reader == nil {
		return Definition{}, fmt.Errorf(
			"%w: definition reader is nil",
			artifactstore.ErrInvalid,
		)
	}
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return Definition{}, err
	}
	if err := artifactstore.ValidateDigest(digest); err != nil {
		return Definition{}, err
	}

	value, err := reader.Get(ctx, rootID, digest)
	if err != nil {
		return Definition{}, err
	}
	canonical, err := Canonicalize(value)
	if err != nil {
		return Definition{}, fmt.Errorf("canonicalize read definition: %w", err)
	}
	if canonical.Digest != digest {
		return Definition{}, fmt.Errorf(
			"%w: requested definition %q, reader returned %q",
			artifactstore.ErrDigestMismatch,
			digest,
			canonical.Digest,
		)
	}
	return canonical, nil
}
