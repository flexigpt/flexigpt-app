package engine

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
)

type Refresher struct {
	workspaces *Service
	loader     *DescriptorLoader
	planner    *Planner
	runner     refresh.Runner
	policy     *ArtifactPolicy
}

func NewRefresher(
	workspaces *Service,
	loader *DescriptorLoader,
	planner *Planner,
	runner refresh.Runner,
	policy *ArtifactPolicy,
) (*Refresher, error) {
	if workspaces == nil ||
		loader == nil ||
		planner == nil ||
		runner == nil ||
		policy == nil {
		return nil, fmt.Errorf(
			"%w: Workspace refresher dependencies are incomplete",
			ErrInvalidWorkspace,
		)
	}
	return &Refresher{
		workspaces: workspaces,
		loader:     loader,
		planner:    planner,
		runner:     runner,
		policy:     policy,
	}, nil
}

func (r *Refresher) Refresh(
	ctx context.Context,
	workspace artifactstore.CollectionRef,
) (refresh.Result, error) {
	if err := workspace.Validate(); err != nil {
		return refresh.Result{}, err
	}
	value, err := r.workspaces.PrepareRefresh(ctx, workspace)
	if err != nil {
		return refresh.Result{}, err
	}
	observation, err := r.loader.Load(ctx, value)
	if err != nil {
		return refresh.Result{}, err
	}
	plan, err := r.planner.Build(value, observation)
	if err != nil {
		return refresh.Result{}, err
	}
	if observation.SourceID != "" {
		found := false
		for index := range plan.Sources {
			if plan.Sources[index].SourceID != observation.SourceID {
				continue
			}
			plan.Sources[index].ExpectedGeneration = observation.Generation
			found = true
			break
		}
		if !found {
			return refresh.Result{}, fmt.Errorf(
				"%w: descriptor source %q is not part of the Workspace discovery plan",
				ErrInvalidWorkspace,
				observation.SourceID,
			)
		}
	}
	return r.runner.Refresh(ctx, workspace, plan, r.policy)
}
