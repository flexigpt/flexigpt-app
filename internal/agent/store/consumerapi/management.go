package consumerapi

import (
	"context"
	"fmt"
	"sort"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

func (a *API) ListAgentsForManagement(
	ctx context.Context,
) ([]AgentView, error) {
	roots, err := a.managementRoots(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]AgentView, 0)
	for _, rootValue := range roots {
		values, err := a.ListAgents(ctx, ListAgentsRequest{
			RootID: rootValue.ID,
		})
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Artifact.RootID != output[right].Artifact.RootID {
			return output[left].Artifact.RootID <
				output[right].Artifact.RootID
		}
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Artifact.ID < output[right].Artifact.ID
	})
	return output, nil
}

func (a *API) ListAgentCollectionsForManagement(
	ctx context.Context,
) ([]collection.CollectionView, error) {
	roots, err := a.managementRoots(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]collection.CollectionView, 0)
	for _, rootValue := range roots {
		values, err := a.ListAgentCollections(ctx, rootValue.ID)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Artifact.RootID != output[right].Artifact.RootID {
			return output[left].Artifact.RootID <
				output[right].Artifact.RootID
		}
		if output[left].Artifact.LogicalName !=
			output[right].Artifact.LogicalName {
			return output[left].Artifact.LogicalName <
				output[right].Artifact.LogicalName
		}
		return output[left].Artifact.ID < output[right].Artifact.ID
	})
	return output, nil
}

func (a *API) ListAgentImportDestinationsForManagement(
	ctx context.Context,
) ([]AgentImportDestination, error) {
	roots, err := a.managementRoots(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]AgentImportDestination, 0)
	for _, rootValue := range roots {
		values, err := a.ListAgentImportDestinations(ctx, rootValue.ID)
		if err != nil {
			return nil, err
		}
		for index := range values {
			values[index].RootDisplayName = rootValue.DisplayName
		}
		output = append(output, values...)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		if output[left].CollectionName != output[right].CollectionName {
			return output[left].CollectionName <
				output[right].CollectionName
		}
		return output[left].Collection.Artifact.ID <
			output[right].Collection.Artifact.ID
	})
	return output, nil
}

func (a *API) ensureDefaultAgentCollectionRoot(
	ctx context.Context,
) (root.RootID, error) {
	if a == nil || a.roots == nil {
		return "", basespec.ErrClosed
	}
	if ctx == nil {
		return "", fmt.Errorf(
			"%w: default Agent Collection Root context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	value, err := a.roots.Create(
		ctx,
		documentTopology.UserRootDraft(),
	)
	if err != nil {
		return "", err
	}
	if value.ID != documentTopology.UserRootID() {
		return "", fmt.Errorf(
			"%w: default Agent Collection Root has unexpected ID %q",
			basespec.ErrInvalid,
			value.ID,
		)
	}
	return value.ID, nil
}

func (a *API) managementRoots(
	ctx context.Context,
) ([]root.Root, error) {
	if a == nil || a.roots == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Agent management Root list context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	values, err := a.roots.List(ctx)
	if err != nil {
		return nil, err
	}

	output := append([]root.Root(nil), values...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].ID < output[right].ID
	})
	return output, nil
}
