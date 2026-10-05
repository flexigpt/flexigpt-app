package main

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
)

type MCPSettingsView struct {
	Settings mcpAuth.MCPAuthSettings `json:"settings"`
	Revision uint64                  `json:"revision"`
}

type MCPStoreWrapper struct {
	api        *mcpConsumerAPI.API
	management *mcpConsumerAPI.MCPListService
	roots      root.API
	settings   *mcpSettingsAdapter
}

func withMCPStore[T any](
	w *MCPStoreWrapper,
	fn func(*mcpConsumerAPI.API) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, spec.ErrClosed
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
			return zero, spec.ErrClosed
		}
		return fn(w.management)
	})
}

func (w *MCPStoreWrapper) ListMCPServers(
	rootID rootModel.RootID,
) ([]mcpConsumerAPI.ServerListItem, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]mcpConsumerAPI.ServerListItem, error) {
		return api.ListServers(context.Background(), mcpConsumerAPI.ListServersRequest{
			RootID: rootID,
		})
	})
}

func (w *MCPStoreWrapper) ListMCPPolicies(
	rootID rootModel.RootID,
) ([]mcpConsumerAPI.PolicyListItem, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]mcpConsumerAPI.PolicyListItem, error) {
		return api.ListPolicies(context.Background(), mcpConsumerAPI.ListPoliciesRequest{
			RootID: rootID,
		})
	})
}

func (w *MCPStoreWrapper) ListMCPPluginsPage(
	pageSize int,
	pageToken string,
) (mcpConsumerAPI.PluginPage, error) {
	return withMCPStoreManagement(
		w,
		func(service *mcpConsumerAPI.MCPListService) (mcpConsumerAPI.PluginPage, error) {
			return service.ListPluginsPage(context.Background(), pageSize, pageToken)
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
			return MCPSettingsView{}, spec.ErrClosed
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
			return MCPSettingsView{}, spec.ErrClosed
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
	ref artifactModel.ArtifactRef,
) (mcpConsumerAPI.ServerSecretsView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.ServerSecretsView, error) {
		return api.GetServerSecrets(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) GetMCPPolicy(
	ref artifactModel.ArtifactRef,
) (mcpConsumerAPI.PolicyView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.PolicyView, error) {
		return api.GetMCPPolicy(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) CreateMCPPlugin(
	request plugin.CreateRequest,
) (plugin.PluginView, error) {
	return withRecoveryResp(
		func() (plugin.PluginView, error) {
			if w == nil || w.api == nil {
				return plugin.PluginView{}, spec.ErrClosed
			}

			// A blank RootID means "create in the retained user Root". The
			// management page must not be unable to create a custom MCP Bundle
			// merely because its baseline discovery has not completed.
			if request.RootID == "" {
				if w.roots == nil {
					return plugin.PluginView{}, spec.ErrClosed
				}
				if _, err := w.roots.Create(
					context.Background(),
					topology.UserRootDraft(),
				); err != nil {
					return plugin.PluginView{}, err
				}
				request.RootID = topology.UserRootID()
			}

			return w.api.CreateMCPPlugin(
				context.Background(),
				request,
			)
		},
	)
}

func (w *MCPStoreWrapper) GetMCPPlugin(
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (plugin.PluginView, error) {
		return api.GetMCPPlugin(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) SetMCPPluginEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (plugin.PluginView, error) {
	return withMCPStore(
		w,
		func(api *mcpConsumerAPI.API) (plugin.PluginView, error) {
			return api.SetMCPPluginEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *MCPStoreWrapper) ListMCPPlugins(
	rootID rootModel.RootID,
) ([]plugin.ListItem, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]plugin.ListItem, error) {
		return api.ListMCPPlugins(context.Background(), rootID)
	})
}

func (w *MCPStoreWrapper) ListMCPPluginMemberships(
	ref artifactModel.ArtifactRef,
) ([]plugin.ArtifactMembershipView, error) {
	return withMCPStore(
		w,
		func(api *mcpConsumerAPI.API) ([]plugin.ArtifactMembershipView, error) {
			return api.ListMCPPluginMemberships(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *MCPStoreWrapper) UpdateMCPPlugin(
	request plugin.UpdateRequest,
) (plugin.PluginView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (plugin.PluginView, error) {
		return api.UpdateMCPPlugin(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) AddMCPPluginMember(
	request plugin.AddMemberRequest,
) (plugin.PluginView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (plugin.PluginView, error) {
		return api.AddMCPPluginMember(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) AddMCPServerToPlugin(
	request plugin.AddArtifactMemberRequest,
) (plugin.PluginView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (plugin.PluginView, error) {
		return api.AddMCPServerToPlugin(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) RemoveMCPPluginMember(
	request plugin.RemoveMemberRequest,
) (plugin.PluginView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (plugin.PluginView, error) {
		return api.RemoveMCPPluginMember(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) DeleteMCPPlugin(
	request plugin.DeleteRequest,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return spec.ErrClosed
		}
		return w.api.DeleteMCPPlugin(context.Background(), request)
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
