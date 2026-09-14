package consumerapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/prompt"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/skill"
)

func (a *StoreAPI) ComposeWorkspacePrompt(
	ctx context.Context,
	workspace WorkspaceRef,
	artifacts []artifact.ArtifactRef,
) (prompt.Plan, error) {
	value, err := a.GetWorkspace(ctx, workspace)
	if err != nil {
		return prompt.Plan{}, err
	}
	return a.contextAdapter.Compose(ctx, value, artifacts)
}

func (a *StoreAPI) LoadWorkspaceSkills(
	ctx context.Context,
	workspace WorkspaceRef,
	artifacts []artifact.ArtifactRef,
) (skill.LoadPlan, error) {
	value, err := a.GetWorkspace(ctx, workspace)
	if err != nil {
		return skill.LoadPlan{}, err
	}
	return a.skillAdapter.Load(ctx, value, artifacts)
}

func (a *StoreAPI) ListWorkspaceSkills(
	ctx context.Context,
	workspace WorkspaceRef,
) ([]skill.WorkspaceSkill, error) {
	value, err := a.GetWorkspace(ctx, workspace)
	if err != nil {
		return nil, err
	}
	return a.skillAdapter.List(ctx, value)
}
