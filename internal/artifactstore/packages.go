package artifactstore

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

func (s *Store) GetArtifactPackage(
	ctx context.Context,
	sourceID spec.SourceID,
	manifestLocator spec.SourceLocator,
) (spec.ArtifactPackage, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactPackage{}, err
	}
	defer finish()
	return s.repository.GetArtifactPackage(ctx, sourceID, manifestLocator)
}

func (s *Store) ListArtifactPackagesForSource(
	ctx context.Context,
	sourceID spec.SourceID,
) ([]spec.ArtifactPackage, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	return s.repository.ListArtifactPackagesForSource(ctx, sourceID)
}

// PublishArtifactPackage persists app-local discovery metadata for a portable
// package manifest. The caller owns source parsing and never leaks local paths
// into the portable manifest itself.
func (s *Store) PublishArtifactPackage(ctx context.Context, artifactPackage spec.ArtifactPackage) error {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer finish()
	if err := validate.ValidateArtifactPackage(artifactPackage); err != nil {
		return fmt.Errorf("%w: artifact package: %w", spec.ErrInvalidRequest, err)
	}
	if _, err := s.repository.GetSource(ctx, artifactPackage.SourceID); err != nil {
		return err
	}
	return s.repository.UpsertArtifactPackage(ctx, artifactPackage)
}
