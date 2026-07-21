package workspace

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
)

type Refresher struct {
	workspaces *Service
	loader     *DefinitionLoader
	planner    *Planner
	runner     refresh.Runner
	policy     *RecordPolicy
}

func NewRefresher(
	workspaces *Service,
	loader *DefinitionLoader,
	planner *Planner,
	runner refresh.Runner,
	policy *RecordPolicy,
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
	rootID artifactstore.RootID,
) (refresh.Result, error) {
	value, err := r.workspaces.Get(ctx, rootID)
	if err != nil {
		return refresh.Result{}, err
	}
	definitionPreferences, err := r.loader.Load(ctx, value)
	if err != nil {
		return refresh.Result{}, err
	}
	plan, err := r.planner.Build(value, definitionPreferences)
	if err != nil {
		return refresh.Result{}, err
	}
	return r.runner.Refresh(ctx, rootID, plan, r.policy)
}
