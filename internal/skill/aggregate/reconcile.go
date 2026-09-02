package aggregate

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

func (s *Service) resyncCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if err := ref.Validate(); err != nil {
		return err
	}

	catalogID, err := skillStore.CollectionCatalogID(ref)
	if err != nil {
		return err
	}

	return s.runtime.SyncCatalog(ctx, catalogID)
}
