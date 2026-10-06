package workspace

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/domain"
)

// ConversationSource is the narrow Workspace consumer port used by
// conversation inference hydration. It deliberately exposes only
// consumer-safe Workspace views and plans.
type ConversationSource struct {
	api *Service
}

func NewConversationSource(
	api *Service,
) (*ConversationSource, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Workspace conversation source requires a Store API",
			spec.ErrInvalid,
		)
	}
	return &ConversationSource{api: api}, nil
}

func (s *ConversationSource) ResolveWorkspace(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (workspaceDomain.WorkspaceView, error) {
	value, err := s.api.resolveWorkspace(ctx, ref)
	if err != nil {
		return workspaceDomain.WorkspaceView{}, err
	}
	return value.View(), nil
}

func (s *ConversationSource) ComposeWorkspacePrompt(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (WorkspacePromptPlan, error) {
	return s.api.ComposeWorkspacePrompt(ctx, workspace, artifacts)
}

func (s *ConversationSource) LoadWorkspaceSkills(
	ctx context.Context,
	workspace artifactModel.ArtifactRef,
	artifacts []artifactModel.ArtifactRef,
) (WorkspaceSkillLoadPlan, error) {
	return s.api.LoadWorkspaceSkills(ctx, workspace, artifacts)
}
