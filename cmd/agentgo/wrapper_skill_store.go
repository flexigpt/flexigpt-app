package main

import (
	"context"
	"errors"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillBuiltin "github.com/flexigpt/flexigpt-app/internal/skill/store/builtin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

type SkillStoreWrapper struct {
	api   *skillConsumerAPI.API
	roots compositionapi.RootAPI
}

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

func (w *SkillStoreWrapper) GetSkill(
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (artifact.Artifact, error) {
		return api.GetSkill(context.Background(), ref)
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
	return withSkillStore(w, func(api *skillConsumerAPI.API) (collection.CollectionView, error) {
		return api.CreateSkillCollection(context.Background(), request)
	})
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

func (w *SkillStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.roots = nil
}
