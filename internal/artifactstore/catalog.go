package artifactstore

import (
	"context"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

// GetCatalogResource returns a source-local catalog observation by its natural
// key. It does not access source content.
func (s *Store) GetCatalogResource(
	ctx context.Context,
	key spec.CatalogResourceKey,
) (spec.CatalogResource, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.CatalogResource{}, err
	}
	defer finish()
	return s.repository.GetCatalogResource(ctx, key)
}

// ListCatalogResourcesForSource lists all current catalog observations for one
// app-local source registration.
func (s *Store) ListCatalogResourcesForSource(
	ctx context.Context,
	sourceID spec.SourceID,
) ([]spec.CatalogResource, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	return s.repository.ListCatalogResourcesForSource(ctx, sourceID)
}

// ListCatalogResourcesForRoot lists resources from enabled source attachments
// belonging to one active root.
func (s *Store) ListCatalogResourcesForRoot(
	ctx context.Context,
	rootID spec.RootID,
) ([]spec.CatalogResource, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	if _, err := s.repository.GetRoot(ctx, rootID, false); err != nil {
		return nil, err
	}
	generation, err := s.repository.GetRootCatalogGeneration(ctx, rootID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureRootCatalogCurrent(ctx, rootID, generation); err != nil {
		return nil, err
	}
	resources, err := s.repository.ListPublishedCatalogResourcesForRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	confirmed, err := s.repository.GetRootCatalogGeneration(ctx, rootID)
	if err != nil {
		return nil, err
	}
	if confirmed.Generation != generation.Generation {
		return nil, fmt.Errorf("%w: root catalog changed while it was being read", spec.ErrConflict)
	}
	if err := s.ensureRootCatalogCurrent(ctx, rootID, confirmed); err != nil {
		return nil, err
	}
	return resources, nil
}

// GetRootCatalogGeneration returns the most recently published generation for
// one root.
func (s *Store) GetRootCatalogGeneration(
	ctx context.Context,
	rootID spec.RootID,
) (spec.RootCatalogGeneration, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.RootCatalogGeneration{}, err
	}
	defer finish()
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
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	return s.repository.ListCatalogResourceRevisions(ctx, key)
}

// GetDefinitionByDigest resolves portable content through the injected
// MapStore-backed content repository. It never reads a filesystem directly.
func (s *Store) GetDefinitionByDigest(
	ctx context.Context,
	digest spec.Digest,
) (spec.CanonicalDefinition, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}
	defer finish()
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

func (s *Store) ensureRootCatalogCurrent(
	ctx context.Context,
	rootID spec.RootID,
	generation spec.RootCatalogGeneration,
) error {
	root, err := s.repository.GetRoot(ctx, rootID, false)
	if err != nil {
		return err
	}
	if !root.Enabled {
		return fmt.Errorf("%w: root %q is disabled", spec.ErrConflict, rootID)
	}
	if root.MountRevision != generation.RootRevision {
		return fmt.Errorf(
			"%w: root %q mount changed after catalog generation %d; rescan the root",
			spec.ErrConflict,
			rootID,
			generation.Generation,
		)
	}
	if generation.RootID != rootID {
		return fmt.Errorf(
			"%w: catalog generation belongs to a different root",
			spec.ErrInvalidRequest,
		)
	}

	attachments, err := s.repository.ListRootSourceAttachments(ctx, rootID)
	if err != nil {
		return err
	}
	current := make(map[spec.SourceID]spec.SourceCatalogVersion)
	for _, attachment := range attachments {
		source, err := s.repository.GetSource(ctx, attachment.SourceID)
		if err != nil {
			return err
		}
		if !attachment.Enabled || !source.Enabled {
			continue
		}
		if source.LastObservedGeneration == nil {
			return fmt.Errorf(
				"%w: root %q has an unobserved active source %q; rescan the root",
				spec.ErrConflict,
				rootID,
				source.SourceID,
			)
		}
		current[source.SourceID] = spec.SourceCatalogVersion{
			Generation:          *source.LastObservedGeneration,
			ObservationRevision: source.ObservationRevision,
		}
	}
	if !maps.Equal(current, generation.SourceVersions) {
		return fmt.Errorf(
			"%w: root %q source catalogs changed after catalog generation %d; rescan the root",
			spec.ErrConflict,
			rootID,
			generation.Generation,
		)
	}
	return nil
}

func (s *Store) publishedCatalogResource(
	ctx context.Context,
	rootID spec.RootID,
	key spec.CatalogResourceKey,
) (spec.CatalogResource, error) {
	resources, err := s.ListCatalogResourcesForRoot(ctx, rootID)
	if err != nil {
		return spec.CatalogResource{}, err
	}
	for _, resource := range resources {
		if resource.SourceID == key.SourceID &&
			resource.Locator == key.Locator &&
			resource.SubresourceLocator == key.SubresourceLocator {
			return resource, nil
		}
	}
	return spec.CatalogResource{}, fmt.Errorf(
		"%w: catalog resource %q/%q/%q is not published for root %q",
		spec.ErrNotFound,
		key.SourceID,
		key.Locator,
		key.SubresourceLocator,
		rootID,
	)
}
