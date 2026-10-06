package main

import (
	"context"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/auth"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type MCPSettingsView struct {
	Settings mcpAuth.MCPAuthSettings `json:"settings"`
	Revision uint64                  `json:"revision"`
}

type MCPStoreWrapper struct {
	api        *mcpAPI.Service
	management *mcpAPI.MCPListService
	roots      root.API
	settings   *mcpSettingsAdapter
}

func withMCPStore[T any](
	w *MCPStoreWrapper,
	fn func(*mcpAPI.Service) (T, error),
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
	fn func(*mcpAPI.MCPListService) (T, error),
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
) ([]mcpAPI.ServerListItem, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) ([]mcpAPI.ServerListItem, error) {
		return api.ListServers(context.Background(), mcpAPI.ListServersRequest{
			RootID: rootID,
		})
	})
}

func (w *MCPStoreWrapper) ListMCPPolicies(
	rootID rootModel.RootID,
) ([]mcpAPI.PolicyListItem, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) ([]mcpAPI.PolicyListItem, error) {
		return api.ListPolicies(context.Background(), mcpAPI.ListPoliciesRequest{
			RootID: rootID,
		})
	})
}

func (w *MCPStoreWrapper) ListMCPPluginsPage(
	pageSize int,
	pageToken string,
) (mcpAPI.PluginPage, error) {
	return withMCPStoreManagement(
		w,
		func(service *mcpAPI.MCPListService) (mcpAPI.PluginPage, error) {
			return service.ListPluginsPage(context.Background(), pageSize, pageToken)
		},
	)
}

func (w *MCPStoreWrapper) ListMCPServersPage(
	pageSize int,
	pageToken string,
) (mcpAPI.ServerPage, error) {
	return withMCPStoreManagement(w, func(service *mcpAPI.MCPListService) (mcpAPI.ServerPage, error) {
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
) (mcpAPI.ServerSecretsView, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) (mcpAPI.ServerSecretsView, error) {
		return api.GetServerSecrets(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) GetMCPPolicy(
	ref artifactModel.ArtifactRef,
) (mcpAPI.PolicyView, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) (mcpAPI.PolicyView, error) {
		return api.GetMCPPolicy(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) CreateMCPPlugin(
	request pluginAPI.CreateRequest,
) (pluginAPI.PluginView, error) {
	return withRecoveryResp(
		func() (pluginAPI.PluginView, error) {
			if w == nil || w.api == nil {
				return pluginAPI.PluginView{}, spec.ErrClosed
			}

			// A blank RootID means "create in the retained user Root". The
			// management page must not be unable to create a custom MCP Bundle
			// merely because its baseline discovery has not completed.
			if request.RootID == "" {
				if w.roots == nil {
					return pluginAPI.PluginView{}, spec.ErrClosed
				}
				if _, err := w.roots.Create(
					context.Background(),
					topology.UserRootDraft(),
				); err != nil {
					return pluginAPI.PluginView{}, err
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
) (pluginAPI.PluginView, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) (pluginAPI.PluginView, error) {
		return api.GetMCPPlugin(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) SetMCPPluginEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (pluginAPI.PluginView, error) {
	return withMCPStore(
		w,
		func(api *mcpAPI.Service) (pluginAPI.PluginView, error) {
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
) ([]pluginAPI.ListItem, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) ([]pluginAPI.ListItem, error) {
		return api.ListMCPPlugins(context.Background(), rootID)
	})
}

func (w *MCPStoreWrapper) ListMCPPluginMemberships(
	ref artifactModel.ArtifactRef,
) ([]pluginAPI.ArtifactMembershipView, error) {
	return withMCPStore(
		w,
		func(api *mcpAPI.Service) ([]pluginAPI.ArtifactMembershipView, error) {
			return api.ListMCPPluginMemberships(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *MCPStoreWrapper) UpdateMCPPlugin(
	request pluginAPI.UpdateRequest,
) (pluginAPI.PluginView, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) (pluginAPI.PluginView, error) {
		return api.UpdateMCPPlugin(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) AddMCPPluginMember(
	request pluginAPI.AddMemberRequest,
) (pluginAPI.PluginView, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) (pluginAPI.PluginView, error) {
		return api.AddMCPPluginMember(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) AddMCPServerToPlugin(
	request pluginAPI.AddArtifactMemberRequest,
) (pluginAPI.PluginView, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) (pluginAPI.PluginView, error) {
		return api.AddMCPServerToPlugin(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) RemoveMCPPluginMember(
	request pluginAPI.RemoveMemberRequest,
) (pluginAPI.PluginView, error) {
	return withMCPStore(w, func(api *mcpAPI.Service) (pluginAPI.PluginView, error) {
		return api.RemoveMCPPluginMember(context.Background(), request)
	})
}

func (w *MCPStoreWrapper) DeleteMCPPlugin(
	request pluginAPI.DeleteRequest,
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
