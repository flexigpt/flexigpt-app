package definition

import (
	"context"
	"fmt"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// Service owns Definition lookup validation, liveness-facing read semantics,
// ordering, cardinality, and ownership of returned mutable values.
type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: Definition repository is nil", spec.ErrInvalid)
	}
	return &Service{repository: repository}, nil
}

func (s *Service) GetDefinition(
	ctx context.Context,
	rootID rootModel.RootID,
	digest cryptoutil.Digest,
) (definitionModel.Definition, error) {
	if s == nil || s.repository == nil {
		return definitionModel.Definition{}, spec.ErrClosed
	}
	if ctx == nil {
		return definitionModel.Definition{}, fmt.Errorf("%w: Definition read context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return definitionModel.Definition{}, err
	}
	if err := rootID.Validate(); err != nil {
		return definitionModel.Definition{}, err
	}
	if err := cryptoutil.ValidateDigest(digest); err != nil {
		return definitionModel.Definition{}, err
	}
	value, err := s.repository.GetDefinition(ctx, rootID, digest)
	if err != nil {
		return definitionModel.Definition{}, err
	}
	// Repository values are contractually independently owned. Avoid a second
	// deep copy on every immutable read after its persistence boundary has
	// already established ownership.
	return value, nil
}

func (s *Service) GetDefinitions(
	ctx context.Context,
	keys []definitionModel.Key,
) ([]definitionModel.Definition, error) {
	if s == nil || s.repository == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: definition batch read context is nil", spec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return []definitionModel.Definition{}, nil
	}
	ownedKeys := append([]definitionModel.Key(nil), keys...)
	for index, key := range ownedKeys {
		if err := key.Validate(); err != nil {
			return nil, fmt.Errorf("definition keys[%d]: %w", index, err)
		}
	}
	values, err := s.repository.GetDefinitions(ctx, ownedKeys)
	if err != nil {
		return nil, err
	}
	if len(values) != len(ownedKeys) {
		return nil, fmt.Errorf(
			"%w: definition repository returned %d values for %d keys",
			spec.ErrInvalid,
			len(values),
			len(ownedKeys),
		)
	}
	return values, nil
}
