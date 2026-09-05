package artifactstore

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
)

var (
	_ topology.Ensurer              = (*API)(nil)
	_ topology.HydrationCoordinator = (*API)(nil)
)

func (a *API) EnsureProtectedTopology(
	ctx context.Context,
	declaration topology.Declaration,
) (topology.Installed, error) {
	if err := a.check(ctx); err != nil {
		return topology.Installed{}, err
	}
	return a.components.EnsureProtectedTopology(ctx, declaration)
}

func (a *API) PrepareTopologyHydrations(
	ctx context.Context,
	desired []topology.Hydration,
) (map[string]bool, error) {
	if err := a.check(ctx); err != nil {
		return nil, err
	}
	return a.components.PrepareTopologyHydrations(ctx, desired)
}

func (a *API) CommitTopologyHydration(
	ctx context.Context,
	desired topology.Hydration,
) error {
	if err := a.check(ctx); err != nil {
		return err
	}
	return a.components.CommitTopologyHydration(ctx, desired)
}
