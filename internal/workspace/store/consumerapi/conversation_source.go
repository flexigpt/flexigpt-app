package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

// ConversationSource is the narrow Workspace consumer port used by
// conversation inference hydration. It deliberately exposes only
// consumer-safe Workspace views and plans.
type ConversationSource struct {
	api *StoreAPI
}

func NewConversationSource(
	api *StoreAPI,
) (*ConversationSource, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Workspace conversation source requires a Store API",
			model.ErrInvalid,
		)
	}
	return &ConversationSource{api: api}, nil
}

func (s *ConversationSource) ResolveWorkspace(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (workspaceDomain.WorkspaceView, error) {
	if s == nil || s.api == nil {
		return workspaceDomain.WorkspaceView{}, model.ErrClosed
	}
	value, err := s.api.resolveWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.WorkspaceView{}, err
	}
	return value.View(), nil
}

func (s *ConversationSource) ComposeWorkspacePrompt(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (WorkspacePromptPlan, error) {
	if s == nil || s.api == nil {
		return WorkspacePromptPlan{}, model.ErrClosed
	}
	return s.api.ComposeWorkspacePrompt(ctx, workspace, artifacts)
}

func (s *ConversationSource) LoadWorkspaceSkills(
	ctx context.Context,
	workspace artifact.ArtifactRef,
	artifacts []artifact.ArtifactRef,
) (WorkspaceSkillLoadPlan, error) {
	if s == nil || s.api == nil {
		return WorkspaceSkillLoadPlan{}, model.ErrClosed
	}
	return s.api.LoadWorkspaceSkills(ctx, workspace, artifacts)
}
