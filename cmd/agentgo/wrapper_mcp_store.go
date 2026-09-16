package main

import (
	"context"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type MCPStoreWrapper struct {
	api   *mcpConsumerAPI.API
	roots compositionapi.RootAPI
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

func (w *MCPStoreWrapper) ListMCPServersForManagement() (
	[]artifact.Artifact,
	error,
) {
	return middleware.WithRecoveryResp(func() ([]artifact.Artifact, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, basespec.ErrClosed
		}
		roots, err := w.roots.List(context.Background())
		if err != nil {
			return nil, err
		}
		output := make([]artifact.Artifact, 0)
		for _, rootValue := range roots {
			values, err := w.api.ListServers(
				context.Background(),
				rootValue.ID,
			)
			if err != nil {
				return nil, err
			}
			output = append(output, values...)
		}
		sort.Slice(output, func(left, right int) bool {
			if output[left].RootID != output[right].RootID {
				return output[left].RootID < output[right].RootID
			}
			if output[left].LogicalName != output[right].LogicalName {
				return output[left].LogicalName <
					output[right].LogicalName
			}
			return output[left].ID < output[right].ID
		})
		return output, nil
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

func (w *MCPStoreWrapper) UpsertManagedMCPPolicy(
	request mcpConsumerAPI.ManagedMCPPolicyUpsertRequest,
) (mcpConsumerAPI.ManagedMCPPolicyUpsertResult, error) {
	return withMCPStore(
		w,
		func(api *mcpConsumerAPI.API) (mcpConsumerAPI.ManagedMCPPolicyUpsertResult, error) {
			return api.UpsertManagedMCPPolicy(
				context.Background(),
				request,
			)
		},
	)
}

func (w *MCPStoreWrapper) CreateManagedMCP(
	request mcpConsumerAPI.ManagedMCPCreateRequest,
) (mcpConsumerAPI.ManagedMCPCreateResult, error) {
	return withMCPStore(
		w,
		func(api *mcpConsumerAPI.API) (mcpConsumerAPI.ManagedMCPCreateResult, error) {
			return api.CreateManagedMCP(context.Background(), request)
		},
	)
}

func (w *MCPStoreWrapper) PurgeManagedMCP(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
		}
		return w.api.PurgeManagedMCP(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *MCPStoreWrapper) PurgeManagedMCPPolicy(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
		}
		return w.api.PurgeManagedMCPPolicy(
			context.Background(),
			ref,
			expectedRevision,
		)
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
	w.roots = nil
}
