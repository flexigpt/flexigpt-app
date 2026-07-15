package artifactstore

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

// GetCatalogResource returns a source-local catalog observation by its natural
// key. It does not access source content.
func (s *Store) GetCatalogResource(
	ctx context.Context,
	key spec.CatalogResourceKey,
) (spec.CatalogResource, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.CatalogResource{}, err
	}
	return s.repository.GetCatalogResource(ctx, key)
}

// ListCatalogResourcesForSource lists all current catalog observations for one
// app-local source registration.
func (s *Store) ListCatalogResourcesForSource(
	ctx context.Context,
	sourceID spec.SourceID,
) ([]spec.CatalogResource, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.repository.ListCatalogResourcesForSource(ctx, sourceID)
}

// ListCatalogResourcesForRoot lists resources from enabled source attachments
// belonging to one active root.
func (s *Store) ListCatalogResourcesForRoot(
	ctx context.Context,
	rootID spec.RootID,
) ([]spec.CatalogResource, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	if _, err := s.repository.GetRoot(ctx, rootID, false); err != nil {
		return nil, err
	}
	return s.repository.ListCatalogResourcesForRoot(ctx, rootID)
}

// GetRootCatalogGeneration returns the most recently published generation for
// one root.
func (s *Store) GetRootCatalogGeneration(
	ctx context.Context,
	rootID spec.RootID,
) (spec.RootCatalogGeneration, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	if _, err := s.repository.GetRoot(ctx, rootID, false); err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	return s.repository.GetRootCatalogGeneration(ctx, rootID)
}

// ListDefinitionHistory returns retained digest revisions for one source-local
// occurrence. Definition bodies are resolved separately through the portable
// content repository.
func (s *Store) ListDefinitionHistory(
	ctx context.Context,
	key spec.CatalogResourceKey,
) ([]spec.CatalogResourceRevision, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.repository.ListCatalogResourceRevisions(ctx, key)
}

// GetDefinitionByDigest resolves portable content through the injected
// MapStore-backed content repository. It never reads a filesystem directly.
func (s *Store) GetDefinitionByDigest(
	ctx context.Context,
	digest spec.Digest,
) (spec.CanonicalDefinition, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	if s.portableContent == nil {
		return spec.CanonicalDefinition{}, fmt.Errorf(
			"%w: portable content repository is not configured",
			spec.ErrUnsupported,
		)
	}
	definition, err := s.portableContent.GetDefinition(ctx, digest)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}
	canonical, err := baseutils.CanonicalizeDefinition(definition)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}
	if canonical.Digest != digest {
		return spec.CanonicalDefinition{}, fmt.Errorf(
			"%w: requested %q, resolved %q",
			spec.ErrDigestMismatch,
			digest,
			canonical.Digest,
		)
	}
	return canonical, nil
}
