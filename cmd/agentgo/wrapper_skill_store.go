package main

import (
	"context"
	"errors"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/composition/local/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	skillBuiltin "github.com/flexigpt/flexigpt-app/internal/skill/store/builtin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

type SkillStoreWrapper struct {
	api   *skillConsumerAPI.API
	roots compositionapi.RootAPI
}

func NewSkillBuiltInInstaller(
	hydrator topology.CompiledHydrationCoordinator,
) (builtin.HydrationInstaller, error) {
	if hydrator == nil {
		return nil, errors.New("skill built-in installer dependencies are incomplete")
	}

	return skillBuiltin.NewInstaller(
		skillBuiltin.InstallerDependencies{
			Hydrator: hydrator,
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
		skillConsumerAPI.WithTargetMappers(
			targetMappers,
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
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, model.ErrClosed
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
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return model.ErrClosed
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
) ([]skillConsumerAPI.SkillListItem, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) ([]skillConsumerAPI.SkillListItem, error) {
		return api.ListSkills(context.Background(), skillConsumerAPI.ListSkillsRequest{
			RootID: rootID,
		})
	})
}

func (w *SkillStoreWrapper) ListSkillsForManagement() (
	[]skillConsumerAPI.SkillListItem,
	error,
) {
	return withRecoveryResp(func() ([]skillConsumerAPI.SkillListItem, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, model.ErrClosed
		}
		roots, err := w.roots.List(context.Background())
		if err != nil {
			return nil, err
		}

		output := make([]skillConsumerAPI.SkillListItem, 0)
		for _, rootValue := range roots {
			values, err := w.api.ListSkills(
				context.Background(),
				skillConsumerAPI.ListSkillsRequest{
					RootID: rootValue.ID,
				},
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

			return output[left].Ref.ArtifactID <
				output[right].Ref.ArtifactID
		})
		return output, nil
	})
}

// ListSkillCollectionsForManagement returns every domain-visible Skill
// Collection across every Root. Root remains an internal storage concern;
// callers receive CollectionView values and decide presentation from
// Editable, Deletable, and Baseline.
func (w *SkillStoreWrapper) ListSkillCollectionsForManagement() (
	[]collection.ListItem,
	error,
) {
	return withRecoveryResp(func() ([]collection.ListItem, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, model.ErrClosed
		}

		roots, err := w.roots.List(context.Background())
		if err != nil {
			return nil, err
		}

		output := make([]collection.ListItem, 0)
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
			if output[left].Ref.RootID != output[right].Ref.RootID {
				return output[left].Ref.RootID < output[right].Ref.RootID
			}
			if output[left].Name != output[right].Name {
				return output[left].Name < output[right].Name
			}
			return output[left].Ref.ArtifactID <
				output[right].Ref.ArtifactID
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
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return model.ErrClosed
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
	return withRecoveryResp(
		func() (collection.CollectionView, error) {
			if w == nil || w.api == nil {
				return collection.CollectionView{}, model.ErrClosed
			}

			// A blank RootID is UI request routing to the retained user Root.
			// "compositionapi.Open" owns retained Root creation during startup;
			// an ordinary management request must not mutate topology.
			if request.RootID == "" {
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
) ([]collection.ListItem, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) ([]collection.ListItem, error) {
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
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return model.ErrClosed
		}
		return w.api.DeleteSkillCollection(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) close() {
	if w == nil {
		return
	}

	w.api = nil
	w.roots = nil
}
