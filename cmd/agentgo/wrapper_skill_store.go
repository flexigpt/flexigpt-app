package main

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillBuiltin "github.com/flexigpt/flexigpt-app/internal/skill/store/builtin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

type SkillStoreWrapper struct {
	api   *skillConsumerAPI.API
	roots compositionapi.RootAPI

	catalogWarmupMu     sync.Mutex
	catalogWarmupCancel context.CancelFunc
	catalogWarmupDone   chan struct{}
}

const skillCatalogWarmupStopTimeout = 10 * time.Second

func NewSkillBuiltInInstaller(
	skills skillConsumerAPI.BuiltinStore,
) (builtin.HydrationInstaller, error) {
	if skills == nil {
		return nil, errors.New("skill built-in installer dependencies are incomplete")
	}

	packages, err := builtin.EmbeddedSkillPackages()
	if err != nil {
		return nil, err
	}
	return skillBuiltin.NewInstaller(
		skillBuiltin.InstallerDependencies{
			Skills:   skills,
			Packages: packages,
		},
	)
}

func InitSkillStoreWrapper(
	wrapper *SkillStoreWrapper,
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
		return errors.New("skill Store wrapper dependencies are incomplete")
	}

	api, err := skillConsumerAPI.New(
		sources,
		discovery,
		artifacts,
		resources,
		managedArtifacts,
		protection,
		skillConsumerAPI.WithLocatorResolvers(
			locatorResolvers,
		),
		skillConsumerAPI.WithFallbackProviders(
			fallbackProviders,
		),
	)
	if err != nil {
		return err
	}
	wrapper.api = api
	wrapper.roots = roots
	return nil
}

func withSkillStore[T any](
	w *SkillStoreWrapper,
	fn func(*skillConsumerAPI.API) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *SkillStoreWrapper) RegisterSkillDirectory(
	request skillConsumerAPI.SkillDirectoryRegistration,
) (source.Summary, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (source.Summary, error) {
		return api.RegisterSkillDirectory(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) AddSkillPath(
	request skillConsumerAPI.SkillPathRegistration,
) (skillConsumerAPI.SkillPathRegistrationResult, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (skillConsumerAPI.SkillPathRegistrationResult, error) {
		return api.AddSkillPath(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) RefreshSkillSource(
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
		}
		return w.api.RefreshSkillSource(
			context.Background(),
			rootID,
			sourceID,
		)
	})
}

func (w *SkillStoreWrapper) ListSkills(
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) ([]artifact.Artifact, error) {
		return api.ListSkills(context.Background(), rootID)
	})
}

func (w *SkillStoreWrapper) ListSkillsForManagement() (
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
			values, err := w.api.ListSkills(
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

// ListSkillCollectionsForManagement returns every domain-visible Skill
// Collection across every Root. Root remains an internal storage concern;
// callers receive CollectionView values and decide presentation from
// Editable, Deletable, and Baseline.
func (w *SkillStoreWrapper) ListSkillCollectionsForManagement() (
	[]collection.CollectionView,
	error,
) {
	return middleware.WithRecoveryResp(func() ([]collection.CollectionView, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, basespec.ErrClosed
		}

		roots, err := w.roots.List(context.Background())
		if err != nil {
			return nil, err
		}

		output := make([]collection.CollectionView, 0)
		for _, rootValue := range roots {
			values, err := w.api.ListSkillCollections(
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
	})
}

func (w *SkillStoreWrapper) GetSkill(
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (artifact.Artifact, error) {
		return api.GetSkill(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) ResolveSkillCapabilities(
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (resolve.CapabilityPlan, error) {
		return api.ResolveSkillCapabilities(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) SetSkillEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (artifact.Artifact, error) {
		return api.SetSkillEnabled(
			context.Background(),
			ref,
			expectedRevision,
			enabled,
		)
	})
}

func (w *SkillStoreWrapper) CreateManagedSkill(
	request skillConsumerAPI.ManagedSkillCreateRequest,
) (skillConsumerAPI.ManagedSkillCreateResult, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (skillConsumerAPI.ManagedSkillCreateResult, error) {
		return api.CreateManagedSkill(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) ReplaceManagedSkill(
	request skillConsumerAPI.ManagedSkillReplaceRequest,
) (skillConsumerAPI.ManagedSkillReplaceResult, error) {
	return withSkillStore(
		w,
		func(api *skillConsumerAPI.API) (skillConsumerAPI.ManagedSkillReplaceResult, error) {
			return api.ReplaceManagedSkill(
				context.Background(),
				request,
			)
		},
	)
}

func (w *SkillStoreWrapper) GetManagedSkillDocument(
	ref artifact.ArtifactRef,
) (skillDomain.ManagedSkillDocument, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (skillDomain.ManagedSkillDocument, error) {
		return api.GetManagedSkillDocument(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) PurgeSkill(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
		}
		return w.api.PurgeSkill(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *SkillStoreWrapper) CreateSkillCollection(
	request collection.CreateRequest,
) (collection.CollectionView, error) {
	return middleware.WithRecoveryResp(
		func() (collection.CollectionView, error) {
			if w == nil || w.api == nil {
				return collection.CollectionView{}, basespec.ErrClosed
			}

			// A blank RootID means the retained user Artifact Root. The Skill
			// management page must not disable Collection creation merely
			// because asynchronous baseline discovery is incomplete.
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

			return w.api.CreateSkillCollection(context.Background(), request)
		},
	)
}

func (w *SkillStoreWrapper) ResolveSkillCollection(
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (collection.CollectionCapabilityPlan, error) {
		return api.ResolveSkillCollection(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) GetSkillCollection(
	ref artifact.ArtifactRef,
) (collection.CollectionView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (collection.CollectionView, error) {
		return api.GetSkillCollection(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) SetSkillCollectionEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (collection.CollectionView, error) {
	return withSkillStore(
		w,
		func(api *skillConsumerAPI.API) (collection.CollectionView, error) {
			return api.SetSkillCollectionEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *SkillStoreWrapper) ListSkillCollections(
	rootID root.RootID,
) ([]collection.CollectionView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) ([]collection.CollectionView, error) {
		return api.ListSkillCollections(context.Background(), rootID)
	})
}

func (w *SkillStoreWrapper) ListSkillCollectionMemberships(
	ref artifact.ArtifactRef,
) ([]collection.ArtifactMembershipView, error) {
	return withSkillStore(
		w,
		func(api *skillConsumerAPI.API) ([]collection.ArtifactMembershipView, error) {
			return api.ListSkillCollectionMemberships(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *SkillStoreWrapper) UpdateSkillCollection(
	request collection.UpdateRequest,
) (collection.CollectionView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (collection.CollectionView, error) {
		return api.UpdateSkillCollection(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) AddSkillCollectionMember(
	request collection.AddMemberRequest,
) (collection.CollectionView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (collection.CollectionView, error) {
		return api.AddSkillCollectionMember(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) AttachSkillArtifactToCollection(
	request collection.AddArtifactMemberRequest,
) (collection.CollectionView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (collection.CollectionView, error) {
		return api.AttachSkillArtifactToCollection(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) RemoveSkillCollectionMember(
	request collection.RemoveMemberRequest,
) (collection.CollectionView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (collection.CollectionView, error) {
		return api.RemoveSkillCollectionMember(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) DeleteSkillCollection(
	request collection.DeleteRequest,
) error {
	return middleware.WithRecovery(func() error {
		if w == nil || w.api == nil {
			return basespec.ErrClosed
		}
		return w.api.DeleteSkillCollection(context.Background(), request)
	})
}

// startBuiltinCatalogWarmup starts runtime catalog preparation for the global
// built-in and retained user management Roots. Protected topology hydration
// remains synchronous and must complete before this method
// is called.
func (w *SkillStoreWrapper) startBuiltinCatalogWarmup(
	syncRoot func(context.Context, root.RootID) error,
) {
	if w == nil || syncRoot == nil {
		return
	}

	w.catalogWarmupMu.Lock()
	if w.catalogWarmupCancel != nil {
		w.catalogWarmupMu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	w.catalogWarmupCancel = cancel
	w.catalogWarmupDone = done
	w.catalogWarmupMu.Unlock()

	go func() {
		defer close(done)

		for _, rootID := range documentTopology.ManagementRootIDs() {
			if err := ctx.Err(); err != nil {
				return
			}

			err := syncRoot(ctx, rootID)
			if err == nil || ctx.Err() != nil {
				continue
			}
			slog.Warn(
				"warm Skill runtime catalog",
				"rootID",
				rootID,
				"error",
				err,
			)
		}
	}()
}

func (w *SkillStoreWrapper) stopBuiltinCatalogWarmup() {
	if w == nil {
		return
	}

	w.catalogWarmupMu.Lock()
	cancel := w.catalogWarmupCancel
	done := w.catalogWarmupDone
	w.catalogWarmupCancel = nil
	w.catalogWarmupDone = nil
	w.catalogWarmupMu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	if done == nil {
		return
	}

	select {
	case <-done:
	case <-time.After(skillCatalogWarmupStopTimeout):
		slog.Warn("skill runtime catalog warmup did not stop before shutdown")
	}
}

func (w *SkillStoreWrapper) close() {
	if w == nil {
		return
	}
	w.stopBuiltinCatalogWarmup()
	w.api = nil
	w.roots = nil
}
