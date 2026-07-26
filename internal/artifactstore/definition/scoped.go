package definition

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

// RootValidator establishes whether a Root remains reachable for definition
// access. The system composition supplies an active-Root validator without
// coupling the definition package to one metadata implementation.
type RootValidator func(
	ctx context.Context,
	rootID artifactstore.RootID,
) error

// RootScopedRepository enforces Root reachability before delegating immutable
// definition reads and writes to a content repository.
//
// This prevents a digest and a guessed RootID from becoming an authorization
// bypass after a Root has been retired or purged.
type RootScopedRepository struct {
	repository   Repository
	validateRoot RootValidator
}

func NewRootScopedRepository(
	repository Repository,
	validateRoot RootValidator,
) (*RootScopedRepository, error) {
	if repository == nil || validateRoot == nil {
		return nil, fmt.Errorf(
			"%w: root-scoped definition repository dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &RootScopedRepository{
		repository:   repository,
		validateRoot: validateRoot,
	}, nil
}

func (r *RootScopedRepository) Get(
	ctx context.Context,
	rootID artifactstore.RootID,
	digest artifactstore.Digest,
) (Definition, error) {
	if err := r.validateRequest(ctx, rootID); err != nil {
		return Definition{}, err
	}
	if err := artifactstore.ValidateDigest(digest); err != nil {
		return Definition{}, err
	}

	value, err := r.repository.Get(ctx, rootID, digest)
	if err != nil {
		return Definition{}, err
	}
	canonical, err := Canonicalize(value)
	if err != nil {
		return Definition{}, fmt.Errorf(
			"canonicalize root-scoped definition read: %w",
			err,
		)
	}
	if canonical.Digest != digest {
		return Definition{}, fmt.Errorf(
			"%w: requested definition %q, repository returned %q",
			artifactstore.ErrDigestMismatch,
			digest,
			canonical.Digest,
		)
	}
	return canonical, nil
}

func (r *RootScopedRepository) Put(
	ctx context.Context,
	rootID artifactstore.RootID,
	value Definition,
) (Definition, error) {
	if err := r.validateRequest(ctx, rootID); err != nil {
		return Definition{}, err
	}
	canonicalInput, err := Canonicalize(value)
	if err != nil {
		return Definition{}, err
	}

	stored, err := r.repository.Put(ctx, rootID, canonicalInput)
	if err != nil {
		return Definition{}, err
	}
	canonicalStored, err := Canonicalize(stored)
	if err != nil {
		return Definition{}, fmt.Errorf(
			"canonicalize root-scoped definition write result: %w",
			err,
		)
	}
	if canonicalStored.Digest != canonicalInput.Digest {
		return Definition{}, fmt.Errorf(
			"%w: requested definition %q, repository stored %q",
			artifactstore.ErrDigestMismatch,
			canonicalInput.Digest,
			canonicalStored.Digest,
		)
	}
	return canonicalStored, nil
}

func (r *RootScopedRepository) validateRequest(
	ctx context.Context,
	rootID artifactstore.RootID,
) error {
	if r == nil || r.repository == nil || r.validateRoot == nil {
		return fmt.Errorf("%w: definition repository is unavailable", artifactstore.ErrClosed)
	}
	if ctx == nil {
		return fmt.Errorf("%w: definition repository context is nil", artifactstore.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := artifactstore.ValidateRootID(rootID); err != nil {
		return err
	}
	return r.validateRoot(ctx, rootID)
}
