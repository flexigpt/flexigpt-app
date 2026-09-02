package aggregate

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

func (s *Service) ResyncCollection(
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

func (s *Service) RemoveCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}

	id, err := skillStore.CollectionCatalogID(ref)
	if err != nil {
		return err
	}
	return s.runtime.RemoveCatalog(ctx, id)
}

func (s *Service) WarmCollections(
	ctx context.Context,
	refs []collection.CollectionRef,
) error {
	if err := s.ensureConfigured(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Skill runtime warmup context is nil",
			basespec.ErrInvalid,
		)
	}

	normalized, err := normalizeWarmCollectionRefs(refs)
	if err != nil {
		return err
	}

	var result error
	for _, ref := range normalized {
		if err := ctx.Err(); err != nil {
			return errors.Join(result, err)
		}
		if err := s.ResyncCollection(ctx, ref); err != nil {
			result = errors.Join(
				result,
				fmt.Errorf(
					"sync Skill Collection %q: %w",
					ref.CollectionID,
					err,
				),
			)
		}
	}
	return result
}

func normalizeWarmCollectionRefs(
	refs []collection.CollectionRef,
) ([]collection.CollectionRef, error) {
	seen := make(map[collection.CollectionRef]struct{}, len(refs))
	output := make([]collection.CollectionRef, 0, len(refs))
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[ref]; duplicate {
			continue
		}
		seen[ref] = struct{}{}
		output = append(output, ref)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].CollectionID < output[right].CollectionID
	})
	return output, nil
}
