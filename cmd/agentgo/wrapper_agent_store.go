package main

import (
	"context"
	"errors"

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
	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type AgentStoreWrapper struct {
	api *agentConsumerAPI.API
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
	interpretations *interpretation.Registry,
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

	api, err := agentConsumerAPI.New(
		sources,
		discovery,
		artifacts,
		cat,
		resources,
		managedArtifacts,
		protection,
		definitions,
		agentConsumerAPI.WithRoots(roots),
		agentConsumerAPI.WithCompositionResolver(resolver),
		agentConsumerAPI.WithDeclarationInterpretations(
			interpretations,
		),
	)
	if err != nil {
		return err
	}

	wrapper.api = api
	return nil
}

func withAgentStore[T any](
	w *AgentStoreWrapper,
	fn func(*agentConsumerAPI.API) (T, error),
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
	request agentConsumerAPI.ListAgentsRequest,
) ([]agentConsumerAPI.AgentListItem, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) ([]agentConsumerAPI.AgentListItem, error) {
			return api.ListAgents(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentsForManagement() (
	[]agentConsumerAPI.AgentListItem,
	error,
) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) ([]agentConsumerAPI.AgentListItem, error) {
			return api.ListAgentsForManagement(context.Background())
		},
	)
}

// ListAgentCollectionsForManagement returns every Agent Collection across
// every Root. This avoids frontend root discovery through an unrelated global
// Agent list and ensures empty Collections remain visible.
func (w *AgentStoreWrapper) ListAgentCollectionsForManagement() (
	[]plugin.ListItem,
	error,
) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) ([]plugin.ListItem, error) {
			return api.ListAgentCollectionsForManagement(
				context.Background(),
			)
		},
	)
}

func (w *AgentStoreWrapper) GetAgent(
	ref artifactModel.ArtifactRef,
) (agentConsumerAPI.AgentView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			agentConsumerAPI.AgentView,
			error,
		) {
			return api.GetAgent(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) MaterializeAgentText(
	ref artifactModel.ArtifactRef,
) (agentConsumerAPI.AgentTextMaterialization, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (agentConsumerAPI.AgentTextMaterialization, error) {
			return api.MaterializeAgentText(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ResolveAgent(
	ref artifactModel.ArtifactRef,
) (agentConsumerAPI.AgentResolution, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (agentConsumerAPI.AgentResolution, error) {
			return api.ResolveAgent(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ResolveAgentCapabilities(
	ref artifactModel.ArtifactRef,
) (agentConsumerAPI.AgentCapabilityPlan, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (agentConsumerAPI.AgentCapabilityPlan, error) {
			return api.ResolveAgentCapabilities(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) SetAgentEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (agentConsumerAPI.AgentView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (agentConsumerAPI.AgentView, error) {
			return api.SetAgentEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *AgentStoreWrapper) CreateAgentCollection(
	request plugin.CreateRequest,
) (plugin.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (plugin.PluginView, error) {
			return api.CreateAgentCollection(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) GetAgentCollection(
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (plugin.PluginView, error) {
			return api.GetAgentCollection(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentCollections(
	rootID rootModel.RootID,
) ([]plugin.ListItem, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) ([]plugin.ListItem, error) {
			return api.ListAgentCollections(context.Background(), rootID)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentCollectionMembers(
	ref artifactModel.ArtifactRef,
) (plugin.PluginCapabilityPlan, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (plugin.PluginCapabilityPlan, error) {
			return api.ListAgentCollectionMembers(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *AgentStoreWrapper) UpdateAgentCollection(
	request plugin.UpdateRequest,
) (plugin.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (plugin.PluginView, error) {
			return api.UpdateAgentCollection(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) AddAgentCollectionMember(
	request plugin.AddMemberRequest,
) (plugin.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			plugin.PluginView,
			error,
		) {
			return api.AddAgentCollectionMember(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) AddAgentCollectionArtifactMember(
	request plugin.AddArtifactMemberRequest,
) (plugin.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			plugin.PluginView,
			error,
		) {
			return api.AddAgentCollectionArtifactMember(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) RemoveAgentCollectionMember(
	request plugin.RemoveMemberRequest,
) (plugin.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			plugin.PluginView,
			error,
		) {
			return api.RemoveAgentCollectionMember(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) SetAgentCollectionEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (plugin.PluginView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (plugin.PluginView, error) {
			return api.SetAgentCollectionEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *AgentStoreWrapper) DeleteAgentCollection(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return spec.ErrClosed
		}
		return w.api.DeleteAgentCollection(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *AgentStoreWrapper) ListAgentImportDestinations() (
	[]agentConsumerAPI.AgentImportDestination,
	error,
) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			[]agentConsumerAPI.AgentImportDestination,
			error,
		) {
			return api.ListAgentImportDestinationsForManagement(
				context.Background(),
			)
		},
	)
}

func (w *AgentStoreWrapper) PreviewAgentImport(
	request agentConsumerAPI.AgentImportPreviewRequest,
) (agentConsumerAPI.AgentImportPreview, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			agentConsumerAPI.AgentImportPreview,
			error,
		) {
			return api.PreviewAgentImport(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) CommitAgentImport(
	request agentConsumerAPI.AgentImportCommitRequest,
) (agentConsumerAPI.AgentImportCommitResult, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			agentConsumerAPI.AgentImportCommitResult,
			error,
		) {
			return api.CommitAgentImport(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) ExportAgent(
	request agentConsumerAPI.AgentExportRequest,
) (agentConsumerAPI.AgentExportResult, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			agentConsumerAPI.AgentExportResult,
			error,
		) {
			return api.ExportAgent(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) DeleteManagedAgent(
	request agentConsumerAPI.ManagedAgentDeleteRequest,
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
}
