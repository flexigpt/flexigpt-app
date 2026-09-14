package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/consumerapi"
)

type WorkspaceAggregateWrapper struct {
	api *workspaceConsumerAPI.StoreAPI
}

func (w *WorkspaceAggregateWrapper) SetWorkspaceArtifactRuntimeDisabled(
	workspace workspaceConsumerAPI.WorkspaceRef,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	runtimeDisabled bool,
) (workspaceConsumerAPI.WorkspaceArtifactView, error) {
	return middleware.WithRecoveryResp(
		func() (workspaceConsumerAPI.WorkspaceArtifactView, error) {
			if w == nil || w.api == nil {
				return workspaceConsumerAPI.WorkspaceArtifactView{},
					basespec.ErrClosed
			}
			return w.api.SetArtifactRuntimeDisabled(
				context.Background(),
				workspace,
				ref,
				expectedRevision,
				runtimeDisabled,
			)
		},
	)
}

func (w *WorkspaceAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
}
