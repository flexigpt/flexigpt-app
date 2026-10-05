package main

import (
	"context"
	"errors"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/skillcatalog"
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
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/consumerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/domain"
)

type SkillStoreWrapper struct {
	api   *skillConsumerAPI.API
	roots root.API
}

func NewSkillBuiltInInstaller(
	hydrator installModel.CompiledHydrationCoordinator,
) (installFlow.HydrationInstaller, error) {
	if hydrator == nil {
		return nil, errors.New("skill built-in installer dependencies are incomplete")
	}

	return skillcatalog.NewInstaller(
		skillcatalog.InstallerDependencies{
			Hydrator: hydrator,
		},
	)
}

func InitSkillStoreWrapper(
	wrapper *SkillStoreWrapper,
	roots root.API,
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	cat catalog.API,
	resources resourceFlow.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	definitions definition.API,
	resolver *composition.Resolver,
) error {
	if wrapper == nil ||
		roots == nil ||
		sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil ||
		cat == nil || definitions == nil {
		return errors.New("skill Store wrapper dependencies are incomplete")
	}

	api, err := skillConsumerAPI.New(
		sources,
		discovery,
		artifacts,
		resources,
		managedArtifacts,
		protection,
		cat,
		definitions,
		skillConsumerAPI.WithCompositionResolver(
			resolver,
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
			return zero, spec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *SkillStoreWrapper) RegisterSkillDirectory(
	request skillConsumerAPI.SkillDirectoryRegistration,
) (sourceModel.Summary, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (sourceModel.Summary, error) {
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
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return spec.ErrClosed
		}
		return w.api.RefreshSkillSource(
			context.Background(),
			rootID,
			sourceID,
		)
	})
}

func (w *SkillStoreWrapper) ListSkills(
	rootID rootModel.RootID,
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
			return nil, spec.ErrClosed
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

// ListSkillPluginsForManagement returns every domain-visible Skill
// Plugin across every Root. Root remains an internal storage concern;
// callers receive PluginView values and decide presentation from
// Editable, Deletable, and Baseline.
func (w *SkillStoreWrapper) ListSkillPluginsForManagement() (
	[]plugin.ListItem,
	error,
) {
	return withRecoveryResp(func() ([]plugin.ListItem, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, spec.ErrClosed
		}

		roots, err := w.roots.List(context.Background())
		if err != nil {
			return nil, err
		}

		output := make([]plugin.ListItem, 0)
		for _, rootValue := range roots {
			values, err := w.api.ListSkillPlugins(
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
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (artifactModel.Artifact, error) {
		return api.GetSkill(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) ResolveSkillCapabilities(
	ref artifactModel.ArtifactRef,
) (composition.CapabilityPlan, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (composition.CapabilityPlan, error) {
		return api.ResolveSkillCapabilities(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) SetSkillEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifactModel.Artifact, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (artifactModel.Artifact, error) {
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
	ref artifactModel.ArtifactRef,
) (skillDomain.ManagedSkillDocument, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (skillDomain.ManagedSkillDocument, error) {
		return api.GetManagedSkillDocument(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) PurgeSkill(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return spec.ErrClosed
		}
		return w.api.PurgeSkill(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *SkillStoreWrapper) CreateSkillPlugin(
	request plugin.CreateRequest,
) (plugin.PluginView, error) {
	return withRecoveryResp(
		func() (plugin.PluginView, error) {
			if w == nil || w.api == nil {
				return plugin.PluginView{}, spec.ErrClosed
			}

			// A blank RootID is UI request routing to the retained user Root.
			if request.RootID == "" {
				request.RootID = topology.UserRootID()
			}

			return w.api.CreateSkillPlugin(context.Background(), request)
		},
	)
}

func (w *SkillStoreWrapper) ResolveSkillPlugin(
	ref artifactModel.ArtifactRef,
) (plugin.PluginCapabilityPlan, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (plugin.PluginCapabilityPlan, error) {
		return api.ResolveSkillPlugin(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) GetSkillPlugin(
	ref artifactModel.ArtifactRef,
) (plugin.PluginView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (plugin.PluginView, error) {
		return api.GetSkillPlugin(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) SetSkillPluginEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (plugin.PluginView, error) {
	return withSkillStore(
		w,
		func(api *skillConsumerAPI.API) (plugin.PluginView, error) {
			return api.SetSkillPluginEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *SkillStoreWrapper) ListSkillPlugins(
	rootID rootModel.RootID,
) ([]plugin.ListItem, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) ([]plugin.ListItem, error) {
		return api.ListSkillPlugins(context.Background(), rootID)
	})
}

func (w *SkillStoreWrapper) ListSkillPluginMemberships(
	ref artifactModel.ArtifactRef,
) ([]plugin.ArtifactMembershipView, error) {
	return withSkillStore(
		w,
		func(api *skillConsumerAPI.API) ([]plugin.ArtifactMembershipView, error) {
			return api.ListSkillPluginMemberships(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *SkillStoreWrapper) UpdateSkillPlugin(
	request plugin.UpdateRequest,
) (plugin.PluginView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (plugin.PluginView, error) {
		return api.UpdateSkillPlugin(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) AddSkillPluginMember(
	request plugin.AddMemberRequest,
) (plugin.PluginView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (plugin.PluginView, error) {
		return api.AddSkillPluginMember(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) AttachSkillArtifactToPlugin(
	request plugin.AddArtifactMemberRequest,
) (plugin.PluginView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (plugin.PluginView, error) {
		return api.AttachSkillArtifactToPlugin(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) RemoveSkillPluginMember(
	request plugin.RemoveMemberRequest,
) (plugin.PluginView, error) {
	return withSkillStore(w, func(api *skillConsumerAPI.API) (plugin.PluginView, error) {
		return api.RemoveSkillPluginMember(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) DeleteSkillPlugin(
	request plugin.DeleteRequest,
) error {
	return withRecovery(func() error {
		if w == nil || w.api == nil {
			return spec.ErrClosed
		}
		return w.api.DeleteSkillPlugin(context.Background(), request)
	})
}

func (w *SkillStoreWrapper) close() {
	if w == nil {
		return
	}

	w.api = nil
	w.roots = nil
}
