package main

import (
	"context"
	"errors"

	agentBuiltin "github.com/flexigpt/flexigpt-app/internal/agent/store/builtin"
	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type AgentStoreWrapper struct {
	api *agentConsumerAPI.API
}

func NewAgentBuiltInInstaller(
	hydrator topology.CompiledHydrationCoordinator,
) (builtin.HydrationInstaller, error) {
	if hydrator == nil {
		return nil, errors.New(
			"agent built-in installer dependencies are incomplete",
		)
	}

	return agentBuiltin.NewInstaller(
		agentBuiltin.InstallerDependencies{
			Hydrator: hydrator,
		},
	)
}

func InitAgentStoreWrapper(
	wrapper *AgentStoreWrapper,
	roots compositionapi.RootAPI,
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	fallbackProviders map[declaration.Type]resolve.FallbackProvider,
	targetMappers map[declaration.Type]resolve.ArtifactTargetMapper,
	locatorResolvers ...providerapi.LocatorResolverFactory,
) error {
	if wrapper == nil ||
		roots == nil ||
		sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil {
		return errors.New("agent store wrapper dependencies are incomplete")
	}

	api, err := agentConsumerAPI.New(
		sources,
		discovery,
		artifacts,
		resources,
		managedArtifacts,
		protection,
		agentConsumerAPI.WithRoots(roots),
		agentConsumerAPI.WithLocatorResolvers(locatorResolvers),
		agentConsumerAPI.WithFallbackProviders(fallbackProviders),
		agentConsumerAPI.WithTargetMappers(targetMappers),
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
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, basespec.ErrClosed
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
	[]collection.ListItem,
	error,
) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) ([]collection.ListItem, error) {
			return api.ListAgentCollectionsForManagement(
				context.Background(),
			)
		},
	)
}

func (w *AgentStoreWrapper) GetAgent(
	ref artifact.ArtifactRef,
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
	ref artifact.ArtifactRef,
) (agentConsumerAPI.AgentTextMaterialization, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (agentConsumerAPI.AgentTextMaterialization, error) {
			return api.MaterializeAgentText(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ResolveAgent(
	ref artifact.ArtifactRef,
) (agentConsumerAPI.AgentResolution, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (agentConsumerAPI.AgentResolution, error) {
			return api.ResolveAgent(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ResolveAgentCapabilities(
	ref artifact.ArtifactRef,
) (agentConsumerAPI.AgentCapabilityPlan, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (agentConsumerAPI.AgentCapabilityPlan, error) {
			return api.ResolveAgentCapabilities(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) SetAgentEnabled(
	ref artifact.ArtifactRef,
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
	request collection.CreateRequest,
) (collection.CollectionView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (collection.CollectionView, error) {
			return api.CreateAgentCollection(
				context.Background(),
				request,
			)
		},
	)
}

func (w *AgentStoreWrapper) GetAgentCollection(
	ref artifact.ArtifactRef,
) (collection.CollectionView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (collection.CollectionView, error) {
			return api.GetAgentCollection(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentCollections(
	rootID root.RootID,
) ([]collection.ListItem, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) ([]collection.ListItem, error) {
			return api.ListAgentCollections(context.Background(), rootID)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentCollectionMembers(
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (collection.CollectionCapabilityPlan, error) {
			return api.ListAgentCollectionMembers(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *AgentStoreWrapper) UpdateAgentCollection(
	request collection.UpdateRequest,
) (collection.CollectionView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (collection.CollectionView, error) {
			return api.UpdateAgentCollection(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) AddAgentCollectionMember(
	request collection.AddMemberRequest,
) (collection.CollectionView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			collection.CollectionView,
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
	request collection.AddArtifactMemberRequest,
) (collection.CollectionView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			collection.CollectionView,
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
	request collection.RemoveMemberRequest,
) (collection.CollectionView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (
			collection.CollectionView,
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
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (collection.CollectionView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (collection.CollectionView, error) {
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
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
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
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
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
