package main

import (
	"context"
	"errors"
	"sort"

	agentBuiltin "github.com/flexigpt/flexigpt-app/internal/agent/store/builtin"
	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type AgentStoreWrapper struct {
	api   *agentConsumerAPI.API
	roots compositionapi.RootAPI
}

func NewAgentBuiltInInstaller(
	agents agentConsumerAPI.BuiltinStore,
) (builtin.HydrationInstaller, error) {
	if agents == nil {
		return nil, errors.New(
			"agent built-in installer dependencies are incomplete",
		)
	}

	packages, err := builtin.EmbeddedAgentPackages()
	if err != nil {
		return nil, err
	}
	return agentBuiltin.NewInstaller(
		agentBuiltin.InstallerDependencies{
			Agents:   agents,
			Packages: packages,
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
		agentConsumerAPI.WithLocatorResolvers(locatorResolvers),
		agentConsumerAPI.WithFallbackProviders(fallbackProviders),
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
) ([]agentConsumerAPI.AgentView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) ([]agentConsumerAPI.AgentView, error) {
			return api.ListAgents(context.Background(), request)
		},
	)
}

func (w *AgentStoreWrapper) ListAgentsForManagement() (
	[]agentConsumerAPI.AgentView,
	error,
) {
	return middleware.WithRecoveryResp(
		func() ([]agentConsumerAPI.AgentView, error) {
			if w == nil || w.api == nil || w.roots == nil {
				return nil, basespec.ErrClosed
			}

			roots, err := w.roots.List(context.Background())
			if err != nil {
				return nil, err
			}

			output := make([]agentConsumerAPI.AgentView, 0)
			for _, rootValue := range roots {
				values, err := w.api.ListAgents(
					context.Background(),
					agentConsumerAPI.ListAgentsRequest{
						RootID: rootValue.ID,
					},
				)
				if err != nil {
					return nil, err
				}
				output = append(output, values...)
			}

			sort.Slice(output, func(left, right int) bool {
				if output[left].Artifact.RootID != output[right].Artifact.RootID {
					return output[left].Artifact.RootID <
						output[right].Artifact.RootID
				}
				if output[left].Name != output[right].Name {
					return output[left].Name < output[right].Name
				}
				return output[left].Artifact.ID < output[right].Artifact.ID
			})
			return output, nil
		},
	)
}

// ListAgentCollectionsForManagement returns every Agent Collection across
// every Root. This avoids frontend root discovery through an unrelated global
// Agent list and ensures empty Collections remain visible.
func (w *AgentStoreWrapper) ListAgentCollectionsForManagement() (
	[]collection.CollectionView,
	error,
) {
	return middleware.WithRecoveryResp(
		func() ([]collection.CollectionView, error) {
			if w == nil || w.api == nil || w.roots == nil {
				return nil, basespec.ErrClosed
			}

			roots, err := w.roots.List(context.Background())
			if err != nil {
				return nil, err
			}

			output := make([]collection.CollectionView, 0)
			for _, rootValue := range roots {
				values, err := w.api.ListAgentCollections(
					context.Background(),
					rootValue.ID,
				)
				if err != nil {
					return nil, err
				}
				output = append(output, values...)
			}

			sort.Slice(output, func(left, right int) bool {
				if output[left].Artifact.RootID != output[right].Artifact.RootID {
					return output[left].Artifact.RootID < output[right].Artifact.RootID
				}
				if output[left].Name != output[right].Name {
					return output[left].Name < output[right].Name
				}
				return output[left].Artifact.ID < output[right].Artifact.ID
			})
			return output, nil
		},
	)
}

func (w *AgentStoreWrapper) GetAgent(
	ref artifact.ArtifactRef,
) (agentConsumerAPI.AgentView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (agentConsumerAPI.AgentView, error) {
			return api.GetAgentView(context.Background(), ref)
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
) (resolve.CapabilityPlan, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (resolve.CapabilityPlan, error) {
			return api.ResolveAgentCapabilities(context.Background(), ref)
		},
	)
}

func (w *AgentStoreWrapper) SetAgentEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) (artifact.Artifact, error) {
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
	return middleware.WithRecoveryResp(
		func() (collection.CollectionView, error) {
			if w == nil || w.api == nil {
				return collection.CollectionView{}, basespec.ErrClosed
			}

			// The global management page may be opened before its baseline
			// discovery request finishes. An empty RootID means "use the
			// retained application user Root", not "reject collection
			// creation because a frontend cache is incomplete".
			if request.RootID == "" {
				if w.roots == nil {
					return collection.CollectionView{}, basespec.ErrClosed
				}
				if _, err := w.roots.Create(
					context.Background(),
					documentTopology.UserRootDraft(),
				); err != nil {
					return collection.CollectionView{}, err
				}
				request.RootID = documentTopology.UserRootID()
			}

			return w.api.CreateAgentCollection(
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
) ([]collection.CollectionView, error) {
	return withAgentStore(
		w,
		func(api *agentConsumerAPI.API) ([]collection.CollectionView, error) {
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
	return middleware.WithRecoveryResp(
		func() ([]agentConsumerAPI.AgentImportDestination, error) {
			if w == nil || w.api == nil || w.roots == nil {
				return nil, basespec.ErrClosed
			}

			roots, err := w.roots.List(context.Background())
			if err != nil {
				return nil, err
			}

			output := make([]agentConsumerAPI.AgentImportDestination, 0)
			for _, rootValue := range roots {
				values, err := w.api.ListAgentImportDestinations(
					context.Background(),
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
				return output[left].CollectionName <
					output[right].CollectionName
			})
			return output, nil
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
	w.roots = nil
}
