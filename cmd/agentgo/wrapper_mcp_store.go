package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/mcp/management"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type MCPStoreWrapper struct {
	api        *mcpConsumerAPI.API
	management *management.Service
}

func withMCPStore[T any](
	w *MCPStoreWrapper,
	fn func(*mcpConsumerAPI.API) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.api)
	})
}

func withMCPStoreManagement[T any](
	w *MCPStoreWrapper,
	fn func(*management.Service) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.management == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.management)
	})
}

func (w *MCPStoreWrapper) ListMCPServers(
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]artifact.Artifact, error) {
		return api.ListServers(context.Background(), rootID)
	})
}

func (w *MCPStoreWrapper) ListMCPPolicies(
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]artifact.Artifact, error) {
		return api.ListPolicies(context.Background(), rootID)
	})
}

func (w *MCPStoreWrapper) SetMCPServerEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	return withMCPStore(
		w,
		func(api *mcpConsumerAPI.API) (artifact.Artifact, error) {
			return api.SetServerEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *MCPStoreWrapper) SetMCPPolicyEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	return withMCPStore(
		w,
		func(api *mcpConsumerAPI.API) (artifact.Artifact, error) {
			return api.SetPolicyEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *MCPStoreWrapper) ListMCPCollectionsPage(
	pageSize int,
	pageToken string,
) (management.CollectionPage, error) {
	return withMCPStoreManagement(w, func(service *management.Service) (management.CollectionPage, error) {
		return service.ListCollectionsPage(context.Background(), pageSize, pageToken)
	})
}

func (w *MCPStoreWrapper) ListMCPServersPage(
	pageSize int,
	pageToken string,
) (management.ServerPage, error) {
	return withMCPStoreManagement(w, func(service *management.Service) (management.ServerPage, error) {
		return service.ListServersPage(context.Background(), pageSize, pageToken)
	})
}

func (w *MCPStoreWrapper) GetMCPServerInstallation(
	ref artifact.ArtifactRef,
) (mcpConsumerAPI.ServerInstallationView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.ServerInstallationView, error) {
		return api.GetServerInstallation(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) GetMCPPolicy(
	ref artifact.ArtifactRef,
) (mcpConsumerAPI.PolicyView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.PolicyView, error) {
		return api.GetMCPPolicy(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) ResolveMCPArtifactCapabilities(
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (resolve.CapabilityPlan, error) {
		return api.ResolveArtifactCapabilities(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) CreateMCPCollection(
	request collection.CreateRequest,
) (collection.CollectionView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (collection.CollectionView, error) {
		return api.CreateMCPCollection(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) GetMCPCollection(
	ref artifact.ArtifactRef,
) (collection.CollectionView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (collection.CollectionView, error) {
		return api.GetMCPCollection(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) SetMCPCollectionEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (collection.CollectionView, error) {
	return withMCPStore(
		w,
		func(api *mcpConsumerAPI.API) (collection.CollectionView, error) {
			return api.SetMCPCollectionEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *MCPStoreWrapper) ResolveMCPCollection(
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (collection.CollectionCapabilityPlan, error) {
		return api.ResolveMCPCollection(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) ListMCPCollections(
	rootID root.RootID,
) ([]collection.CollectionView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]collection.CollectionView, error) {
		return api.ListMCPCollections(context.Background(), rootID)
	})
}

func (w *MCPStoreWrapper) ListMCPCollectionMemberships(
	ref artifact.ArtifactRef,
) ([]collection.ArtifactMembershipView, error) {
	return withMCPStore(
		w,
		func(api *mcpConsumerAPI.API) ([]collection.ArtifactMembershipView, error) {
			return api.ListMCPCollectionMemberships(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *MCPStoreWrapper) UpdateMCPCollection(
	request collection.UpdateRequest,
) (collection.CollectionView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (collection.CollectionView, error) {
		return api.UpdateMCPCollection(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) AddMCPCollectionMember(
	request collection.AddMemberRequest,
) (collection.CollectionView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (collection.CollectionView, error) {
		return api.AddMCPCollectionMember(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) AttachMCPArtifactToCollection(
	request collection.AddArtifactMemberRequest,
) (collection.CollectionView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (collection.CollectionView, error) {
		return api.AttachMCPArtifactToCollection(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) RemoveMCPCollectionMember(
	request collection.RemoveMemberRequest,
) (collection.CollectionView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (collection.CollectionView, error) {
		return api.RemoveMCPCollectionMember(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) DeleteMCPCollection(
	request collection.DeleteRequest,
) error {
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
		}
		return w.api.DeleteMCPCollection(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.management = nil
}
