package main

import (
	"context"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"

	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
)

type MCPSettingsView struct {
	Settings mcpAuth.MCPAuthSettings `json:"settings"`
	Revision uint64                  `json:"revision"`
}

type MCPStoreWrapper struct {
	api        *mcpConsumerAPI.API
	management *mcpConsumerAPI.MCPListService
	roots      compositionapi.RootAPI
	settings   *mcpSettingsAdapter
}

func withMCPStore[T any](
	w *MCPStoreWrapper,
	fn func(*mcpConsumerAPI.API) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, model.ErrClosed
		}
		return fn(w.api)
	})
}

func withMCPStoreManagement[T any](
	w *MCPStoreWrapper,
	fn func(*mcpConsumerAPI.MCPListService) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.management == nil {
			return zero, model.ErrClosed
		}
		return fn(w.management)
	})
}

func (w *MCPStoreWrapper) ListMCPServers(
	rootID root.RootID,
) ([]mcpConsumerAPI.ServerListItem, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]mcpConsumerAPI.ServerListItem, error) {
		return api.ListServers(context.Background(), mcpConsumerAPI.ListServersRequest{
			RootID: rootID,
		})
	})
}

func (w *MCPStoreWrapper) ListMCPPolicies(
	rootID root.RootID,
) ([]mcpConsumerAPI.PolicyListItem, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]mcpConsumerAPI.PolicyListItem, error) {
		return api.ListPolicies(context.Background(), mcpConsumerAPI.ListPoliciesRequest{
			RootID: rootID,
		})
	})
}

func (w *MCPStoreWrapper) ListMCPCollectionsPage(
	pageSize int,
	pageToken string,
) (mcpConsumerAPI.CollectionPage, error) {
	return withMCPStoreManagement(
		w,
		func(service *mcpConsumerAPI.MCPListService) (mcpConsumerAPI.CollectionPage, error) {
			return service.ListCollectionsPage(context.Background(), pageSize, pageToken)
		},
	)
}

func (w *MCPStoreWrapper) ListMCPServersPage(
	pageSize int,
	pageToken string,
) (mcpConsumerAPI.ServerPage, error) {
	return withMCPStoreManagement(w, func(service *mcpConsumerAPI.MCPListService) (mcpConsumerAPI.ServerPage, error) {
		return service.ListServersPage(context.Background(), pageSize, pageToken)
	})
}

func (w *MCPStoreWrapper) GetMCPSettings() (
	MCPSettingsView,
	error,
) {
	return withRecoveryResp(func() (MCPSettingsView, error) {
		if w == nil || w.settings == nil {
			return MCPSettingsView{}, model.ErrClosed
		}
		settings, revision, err := w.settings.getMCPSettings(
			context.Background(),
		)
		if err != nil {
			return MCPSettingsView{}, err
		}
		return MCPSettingsView{
			Settings: settings,
			Revision: revision,
		}, nil
	})
}

func (w *MCPStoreWrapper) SaveMCPSettings(
	expectedRevision uint64,
	settings mcpAuth.MCPAuthSettings,
) (MCPSettingsView, error) {
	return withRecoveryResp(func() (MCPSettingsView, error) {
		if w == nil || w.settings == nil {
			return MCPSettingsView{}, model.ErrClosed
		}
		if _, err := w.settings.putMCPSettings(
			context.Background(),
			expectedRevision,
			settings,
		); err != nil {
			return MCPSettingsView{}, err
		}
		value, revision, err := w.settings.getMCPSettings(context.Background())
		if err != nil {
			return MCPSettingsView{}, err
		}
		return MCPSettingsView{Settings: value, Revision: revision}, nil
	})
}

func (w *MCPStoreWrapper) GetMCPServerSecrets(
	ref artifact.ArtifactRef,
) (mcpConsumerAPI.ServerSecretsView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.ServerSecretsView, error) {
		return api.GetServerSecrets(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) GetMCPPolicy(
	ref artifact.ArtifactRef,
) (mcpConsumerAPI.PolicyView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.PolicyView, error) {
		return api.GetMCPPolicy(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) CreateMCPCollection(
	request collection.CreateRequest,
) (collection.CollectionView, error) {
	return withRecoveryResp(
		func() (collection.CollectionView, error) {
			if w == nil || w.api == nil {
				return collection.CollectionView{}, model.ErrClosed
			}

			// A blank RootID means "create in the retained user Root". The
			// management page must not be unable to create a custom MCP Bundle
			// merely because its baseline discovery has not completed.
			if request.RootID == "" {
				if w.roots == nil {
					return collection.CollectionView{}, model.ErrClosed
				}
				if _, err := w.roots.Create(
					context.Background(),
					documentTopology.UserRootDraft(),
				); err != nil {
					return collection.CollectionView{}, err
				}
				request.RootID = documentTopology.UserRootID()
			}

			return w.api.CreateMCPCollection(
				context.Background(),
				request,
			)
		},
	)
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

func (w *MCPStoreWrapper) ListMCPCollections(
	rootID root.RootID,
) ([]collection.ListItem, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]collection.ListItem, error) {
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

func (w *MCPStoreWrapper) AddMCPServerToCollection(
	request collection.AddArtifactMemberRequest,
) (collection.CollectionView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (collection.CollectionView, error) {
		return api.AddMCPServerToCollection(context.Background(), request)
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
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return model.ErrClosed
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
	w.settings = nil
	w.roots = nil
}
