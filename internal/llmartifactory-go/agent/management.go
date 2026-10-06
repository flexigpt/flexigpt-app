package agent

import (
	"context"
	"fmt"
	"sort"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func (a *Service) ListAgentsForManagement(
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

func (a *Service) ListAgentPluginsForManagement(
	ctx context.Context,
) ([]pluginAPI.ListItem, error) {
	roots, err := a.managementRoots(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]pluginAPI.ListItem, 0)
	for _, rootValue := range roots {
		values, err := a.ListAgentPlugins(ctx, rootValue.ID)
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

func (a *Service) ListAgentImportDestinationsForManagement(
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
		if output[left].PluginName != output[right].PluginName {
			return output[left].PluginName <
				output[right].PluginName
		}
		return output[left].Plugin.ArtifactID <
			output[right].Plugin.ArtifactID
	})
	return output, nil
}

func (a *Service) ensureDefaultAgentPluginRoot(
	ctx context.Context,
) (rootModel.RootID, error) {
	if a == nil || a.roots == nil {
		return "", spec.ErrClosed
	}
	if ctx == nil {
		return "", fmt.Errorf(
			"%w: default Agent Plugin Root context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	value, err := a.roots.Create(
		ctx,
		topology.UserRootDraft(),
	)
	if err != nil {
		return "", err
	}
	if value.ID != topology.UserRootID() {
		return "", fmt.Errorf(
			"%w: default Agent Plugin Root has unexpected ID %q",
			spec.ErrInvalid,
			value.ID,
		)
	}
	return value.ID, nil
}

func (a *Service) managementRoots(
	ctx context.Context,
) ([]rootModel.Root, error) {
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

	output := append([]rootModel.Root(nil), values...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].ID < output[right].ID
	})
	return output, nil
}
