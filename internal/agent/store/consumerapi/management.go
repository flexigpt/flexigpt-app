package consumerapi

import (
	"context"
	"fmt"
	"sort"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

func (a *API) ListAgentsForManagement(
	ctx context.Context,
) ([]AgentListItem, error) {
	roots, err := a.managementRoots(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]AgentListItem, 0)
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
		if output[left].Ref.RootID != output[right].Ref.RootID {
			return output[left].Ref.RootID <
				output[right].Ref.RootID
		}
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}

func (a *API) ListAgentCollectionsForManagement(
	ctx context.Context,
) ([]collection.ListItem, error) {
	roots, err := a.managementRoots(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]collection.ListItem, 0)
	for _, rootValue := range roots {
		values, err := a.ListAgentCollections(ctx, rootValue.ID)
		if err != nil {
			return nil, err
		}
		output = append(output, values...)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Ref.RootID != output[right].Ref.RootID {
			return output[left].Ref.RootID <
				output[right].Ref.RootID
		}
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID < output[right].Ref.ArtifactID
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
		return output[left].Collection.ArtifactID <
			output[right].Collection.ArtifactID
	})
	return output, nil
}

func (a *API) ensureDefaultAgentCollectionRoot(
	ctx context.Context,
) (root.RootID, error) {
	if a == nil || a.roots == nil {
		return "", spec.ErrClosed
	}
	if ctx == nil {
		return "", fmt.Errorf(
			"%w: default Agent Collection Root context is nil",
			spec.ErrInvalid,
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
			spec.ErrInvalid,
			value.ID,
		)
	}
	return value.ID, nil
}

func (a *API) managementRoots(
	ctx context.Context,
) ([]root.Root, error) {
	if a == nil || a.roots == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Agent management Root list context is nil",
			spec.ErrInvalid,
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
