package artifactstore

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *Store) GetArtifactPackage(
	ctx context.Context,
	sourceID spec.SourceID,
	manifestLocator spec.SourceLocator,
) (spec.ArtifactPackage, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactPackage{}, err
	}
	return s.repository.GetArtifactPackage(ctx, sourceID, manifestLocator)
}

func (s *Store) ListArtifactPackagesForSource(
	ctx context.Context,
	sourceID spec.SourceID,
) ([]spec.ArtifactPackage, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.repository.ListArtifactPackagesForSource(ctx, sourceID)
}

// PublishArtifactPackage persists app-local discovery metadata for a portable
// package manifest. The caller owns source parsing and never leaks local paths
// into the portable manifest itself.
func (s *Store) PublishArtifactPackage(ctx context.Context, artifactPackage spec.ArtifactPackage) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	return s.repository.UpsertArtifactPackage(ctx, artifactPackage)
}
