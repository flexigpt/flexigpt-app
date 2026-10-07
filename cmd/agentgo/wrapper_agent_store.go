package main

import (
	"context"
	"errors"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/agentcatalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type AgentStoreWrapper struct {
	api   *agentAPI.Service
	roots root.API
}

func NewAgentBuiltInInstaller(
	hydrator installModel.CompiledHydrationCoordinator,
) (installFlow.HydrationInstaller, error) {
	if hydrator == nil {
		return nil, errors.New(
			"agent built-in installer dependencies are incomplete",
		)
	}

	return agentcatalog.NewInstaller(
		agentcatalog.InstallerDependencies{
			Hydrator: hydrator,
		},
	)
}

func InitAgentStoreWrapper(
	wrapper *AgentStoreWrapper,
	roots root.API,
	cat catalog.API,
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	resources resourceFlow.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	definitions definition.API,
	resolver *composition.Resolver,
	interpretations *coreinterpretation.Registry,
) error {
	if wrapper == nil ||
		roots == nil ||
		cat == nil || definitions == nil ||
		sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil ||
		interpretations == nil {
		return errors.New("agent store wrapper dependencies are incomplete")
	}

	support, err := llmsupport.Agent()
	if err != nil {
		return err
	}

	api, err := agentAPI.New(
		sources,
		discovery,
		artifacts,
		cat,
		resources,
		managedArtifacts,
		protection,
		definitions,
		agentAPI.WithCompositionResolver(resolver),
		agentAPI.WithDeclarationInterpretations(
			interpretations,
		),
		agentAPI.WithSupport(support),
	)
	if err != nil {
		return err
	}

	wrapper.api = api
	wrapper.roots = roots
	return nil
}

func withAgentStore[T any](
	w *AgentStoreWrapper,
	fn func(*agentAPI.Service) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, spec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *AgentStoreWrapper) ListAgents(
	request agentAPI.ListAgentsRequest,
) ([]agentAPI.AgentListItem, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) ([]agentAPI.AgentListItem, error) {
			return api.ListAgents(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentsForManagement() (
	[]agentAPI.AgentListItem,
	error,
) {
	return withRecoveryResp(func() ([]agentAPI.AgentListItem, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, spec.ErrClosed
		}
		ctx := context.Background()
		roots, err := w.roots.List(ctx)
		if err != nil {
			return nil, err
		}
		output := make([]agentAPI.AgentListItem, 0)
		for _, rootValue := range roots {
			values, err := w.api.ListAgents(
				ctx,
				agentAPI.ListAgentsRequest{RootID: rootValue.ID},
			)
			if err != nil {
				return nil, err
			}
			output = append(output, values...)
		}
		sort.Slice(output, func(left, right int) bool {
			if output[left].Ref.RootID != output[right].Ref.RootID {
				return output[left].Ref.RootID < output[right].Ref.RootID
			}
			if output[left].Name != output[right].Name {
				return output[left].Name < output[right].Name
			}
			return output[left].Ref.ArtifactID < output[right].Ref.ArtifactID
		})
		return output, nil
	})
}

// ListAgentPluginsForManagement returns every Agent Plugin across
// every Root. This avoids frontend root discovery through an unrelated global
// Agent list and ensures empty Plugins remain visible.
func (w *AgentStoreWrapper) ListAgentPluginsForManagement() (
	[]pluginAPI.ListItem,
	error,
) {
	return withRecoveryResp(func() ([]pluginAPI.ListItem, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, spec.ErrClosed
		}
		ctx := context.Background()
		roots, err := w.roots.List(ctx)
		if err != nil {
			return nil, err
		}
		output := make([]pluginAPI.ListItem, 0)
		for _, rootValue := range roots {
			values, err := w.api.ListAgentPlugins(ctx, rootValue.ID)
			if err != nil {
				return nil, err
			}
			output = append(output, values...)
		}
		sort.Slice(output, func(left, right int) bool {
			if output[left].Ref.RootID != output[right].Ref.RootID {
				return output[left].Ref.RootID < output[right].Ref.RootID
			}
			if output[left].Name != output[right].Name {
				return output[left].Name < output[right].Name
			}
			return output[left].Ref.ArtifactID < output[right].Ref.ArtifactID
		})
		return output, nil
	})
}

func (w *AgentStoreWrapper) GetAgent(
	ref artifactModel.ArtifactRef,
) (agentAPI.AgentView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (
			agentAPI.AgentView,
			error,
		) {
			return api.GetAgent(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ResolveAgent(
	ref artifactModel.ArtifactRef,
) (agentAPI.AgentResolution, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (agentAPI.AgentResolution, error) {
			return api.ResolveAgent(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ResolveAgentCapabilities(
	ref artifactModel.ArtifactRef,
) (agentAPI.AgentCapabilityPlan, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (agentAPI.AgentCapabilityPlan, error) {
			return api.ResolveAgentCapabilities(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) SetAgentEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (agentAPI.AgentView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (agentAPI.AgentView, error) {
			return api.SetAgentEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *AgentStoreWrapper) CreateAgentPlugin(
	request pluginAPI.CreateRequest,
) (pluginAPI.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (pluginAPI.PluginView, error) {
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
			return api.CreateAgentPlugin(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) GetAgentPlugin(
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (pluginAPI.PluginView, error) {
			return api.GetAgentPlugin(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentPlugins(
	rootID rootModel.RootID,
) ([]pluginAPI.ListItem, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) ([]pluginAPI.ListItem, error) {
			return api.ListAgentPlugins(context.Background(), rootID)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentPluginMembers(
	ref artifactModel.ArtifactRef,
) (pluginAPI.DirectMembership, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (pluginAPI.DirectMembership, error) {
			return api.ListAgentPluginMembers(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *AgentStoreWrapper) ResolveAgentPluginCapabilities(
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginCapabilityPlan, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (
			pluginAPI.PluginCapabilityPlan,
			error,
		) {
			return api.ResolveAgentPluginCapabilities(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *AgentStoreWrapper) UpdateAgentPlugin(
	request pluginAPI.UpdateRequest,
) (pluginAPI.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (pluginAPI.PluginView, error) {
			return api.UpdateAgentPlugin(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) AddAgentPluginMember(
	request pluginAPI.AddMemberRequest,
) (pluginAPI.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (
			pluginAPI.PluginView,
			error,
		) {
			return api.AddAgentPluginMember(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) AddAgentPluginArtifactMember(
	request pluginAPI.AddArtifactMemberRequest,
) (pluginAPI.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (
			pluginAPI.PluginView,
			error,
		) {
			return api.AddAgentPluginArtifactMember(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) RemoveAgentPluginMember(
	request pluginAPI.RemoveMemberRequest,
) (pluginAPI.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (
			pluginAPI.PluginView,
			error,
		) {
			return api.RemoveAgentPluginMember(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) SetAgentPluginEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (pluginAPI.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (pluginAPI.PluginView, error) {
			return api.SetAgentPluginEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *AgentStoreWrapper) DeleteAgentPlugin(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return spec.ErrClosed
		}
		return w.api.DeleteAgentPlugin(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *AgentStoreWrapper) ListAgentImportDestinations() (
	[]agentAPI.AgentImportDestination,
	error,
) {
	return withRecoveryResp(func() ([]agentAPI.AgentImportDestination, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, spec.ErrClosed
		}

		ctx := context.Background()
		roots, err := w.roots.List(ctx)
		if err != nil {
			return nil, err
		}
		sort.Slice(roots, func(left, right int) bool {
			return roots[left].ID < roots[right].ID
		})

		output := make([]agentAPI.AgentImportDestination, 0)
		for _, rootValue := range roots {
			values, err := w.api.ListAgentImportDestinations(
				ctx,
				rootValue.ID,
			)
			if err != nil {
				return nil, err
			}
			for index := range values {
				values[index].RootDisplayName = rootValue.DisplayName
			}
			output = append(output, values...)
		}
		sort.Slice(output, func(left, right int) bool {
			if output[left].RootID != output[right].RootID {
				return output[left].RootID < output[right].RootID
			}
			if output[left].PluginName != output[right].PluginName {
				return output[left].PluginName < output[right].PluginName
			}
			return output[left].Plugin.ArtifactID <
				output[right].Plugin.ArtifactID
		})
		return output, nil
	})
}

func (w *AgentStoreWrapper) PreviewAgentImport(
	request agentAPI.AgentImportPreviewRequest,
) (agentAPI.AgentImportPreview, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (
			agentAPI.AgentImportPreview,
			error,
		) {
			return api.PreviewAgentImport(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) CommitAgentImport(
	request agentAPI.AgentImportCommitRequest,
) (agentAPI.AgentImportCommitResult, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (
			agentAPI.AgentImportCommitResult,
			error,
		) {
			return api.CommitAgentImport(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) ExportAgent(
	request agentAPI.AgentExportRequest,
) (agentAPI.AgentExportResult, error) {
	return withAgentStore(
		w,
		func(api *agentAPI.Service) (
			agentAPI.AgentExportResult,
			error,
		) {
			return api.ExportAgent(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) DeleteManagedAgent(
	request agentAPI.ManagedAgentDeleteRequest,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return spec.ErrClosed
		}
		return w.api.DeleteManagedAgent(context.Background(), request)
	})
}

func (w *AgentStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.roots = nil
}
