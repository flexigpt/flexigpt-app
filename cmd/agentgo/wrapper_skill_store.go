package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/skill/aggregatecatalog"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
	skillBundle "github.com/flexigpt/flexigpt-app/internal/skill/store/bundle"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/schemaadapter"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/workspaceadapter"
)

type SkillStoreWrapper struct {
	api    *skillBundle.API
	router *skillStore.ArtifactRouter

	builtInInstaller artifactbuiltin.HydrationInstaller
}

func InitSkillStoreWrapper(
	wrapper *SkillStoreWrapper,
	components *system.Components,
	workspaceSkills *workspaceadapter.Adapter,
) error {
	if wrapper == nil || components == nil || workspaceSkills == nil {
		return errors.New("skill store wrapper dependencies are incomplete")
	}
	autoAdoptionIDProvider := artifact.UUIDArtifactIDProvider()
	api, err := skillBundle.New(skillBundle.Dependencies{
		Sources:                components.Sources,
		Collections:            components.Collections,
		Artifacts:              components.Artifacts,
		Refresh:                components.Refresh,
		Catalogs:               components.Catalogs,
		ManagedArtifacts:       components.ManagedArtifacts,
		SourceRuntime:          components.SourceRuntime,
		HasDecoder:             components.HasDecoder,
		DecoderFingerprint:     components.DecoderFingerprint,
		RootMutationPolicy:     components.RootMutationPolicy(),
		AutoAdoptionIDProvider: autoAdoptionIDProvider,
	})
	if err != nil {
		return err
	}

	router, err := skillStore.NewArtifactRouter(
		components.ArtifactReader,
		components.CollectionReader,
	)
	if err != nil {
		return err
	}

	workspaceResolver, err := workspaceadapter.NewStoreLoader(workspaceSkills)
	if err != nil {
		return err
	}
	bundleResolver, err := skillBundle.NewStoreLoader(api)
	if err != nil {
		return err
	}
	if err := router.Register(
		artifactbuiltin.WorkspaceCollectionV1Kind,
		workspaceResolver,
	); err != nil {
		return err
	}
	if err := router.Register(
		artifactbuiltin.SkillCollectionV1Kind,
		bundleResolver,
	); err != nil {
		return err
	}

	wrapper.api = api
	wrapper.router = router

	skillRegistry, err := schemaadapter.LoadRegistry()
	if err != nil {
		wrapper.close()
		return err
	}

	packages, err := artifactbuiltin.EmbeddedSkillPackages()
	if err != nil {
		wrapper.close()
		return err
	}
	builtIns, err := schemaadapter.NewInstaller(
		schemaadapter.InstallerDependencies{
			Skills:                 api,
			SkillRegistry:          skillRegistry,
			Packages:               packages,
			ShareableCanonicalizer: components.ShareableSchemas,
		},
	)
	if err != nil {
		wrapper.close()
		return err
	}
	wrapper.builtInInstaller = builtIns

	return nil
}

func (w *SkillStoreWrapper) CreateSkillBundle(
	request *skillBundle.CreateBundleRequest,
) (skillBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (skillBundle.Bundle, error) {
		if request == nil {
			return skillBundle.Bundle{}, errors.New("skill bundle request is required")
		}
		value, err := w.api.CreateBundle(context.Background(), *request)

		return value, err
	})
}

// AttachSkillSource attaches a Source already created through Artifact Store
// administration. It intentionally accepts only Source identity and typed
// Skill Bundle role data, never a filesystem path or Source configuration.
func (w *SkillStoreWrapper) AttachSkillSource(
	bundle collection.CollectionRef,
	expectedCollectionRevision uint64,
	draft skillBundle.AttachmentDraft,
) (skillBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (skillBundle.Bundle, error) {
		value, err := w.api.AttachSource(
			context.Background(),
			bundle,
			expectedCollectionRevision,
			draft,
		)
		if err != nil {
			return value, err
		}
		if !value.Collection.Enabled {
			return value, nil
		}
		if _, err := w.api.RefreshBundle(
			context.Background(),
			value.Collection.Ref(),
		); err != nil {
			return value, err
		}
		return value, nil
	})
}

func (w *SkillStoreWrapper) GetSkillBundle(
	ref collection.CollectionRef,
) (skillBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (skillBundle.Bundle, error) {
		return w.api.GetBundle(context.Background(), ref)
	})
}

func (w *SkillStoreWrapper) ListSkillBundles(
	rootID basespec.RootID,
) ([]skillBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() ([]skillBundle.Bundle, error) {
		return w.api.ListBundles(context.Background(), rootID)
	})
}

func (w *SkillStoreWrapper) UpdateSkillBundle(
	request *skillBundle.UpdateBundleRequest,
) (skillBundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (skillBundle.Bundle, error) {
		if request == nil {
			return skillBundle.Bundle{}, errors.New("skill bundle update is required")
		}
		value, err := w.api.UpdateBundle(context.Background(), *request)
		if err != nil {
			return value, err
		}

		ref := value.Collection.Ref()
		if !value.Collection.Enabled {
			return value, nil
		}
		if _, err := w.api.RefreshBundle(
			context.Background(),
			ref,
		); err != nil {
			return value, err
		}
		return value, nil
	})
}

func (w *SkillStoreWrapper) RetireSkillBundle(
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	return middleware.WithRecoveryResp(func() (collection.Collection, error) {
		value, err := w.api.RetireBundle(
			context.Background(),
			ref,
			expectedRevision,
		)
		if err != nil {
			return value, err
		}

		return value, nil
	})
}

func (w *SkillStoreWrapper) PurgeSkillBundle(
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		if err := w.api.PurgeBundle(
			context.Background(),
			ref,
			expectedRevision,
		); err != nil {
			return err
		}
		return nil
	})
}

func (w *SkillStoreWrapper) RefreshSkillBundle(
	ref collection.CollectionRef,
) error {
	return middleware.WithRecovery(func() error {
		_, err := w.api.RefreshBundle(context.Background(), ref)
		if err != nil {
			return err
		}
		return nil
	})
}

func (w *SkillStoreWrapper) CreateManagedSkill(
	request *skillBundle.CreateManagedSkillRequest,
) (skillBundle.CreateManagedSkillResponse, error) {
	return middleware.WithRecoveryResp(
		func() (skillBundle.CreateManagedSkillResponse, error) {
			if request == nil {
				return skillBundle.CreateManagedSkillResponse{},
					errors.New("managed skill request is required")
			}
			value, err := w.api.CreateManagedSkill(context.Background(), *request)
			if err != nil {
				return value, err
			}

			return value, nil
		},
	)
}

func (w *SkillStoreWrapper) GetManagedSkillDocument(
	ref artifact.ArtifactRef,
) (skillBundle.ManagedSkillDocument, error) {
	return middleware.WithRecoveryResp(
		func() (skillBundle.ManagedSkillDocument, error) {
			return w.api.GetManagedSkillDocument(context.Background(), ref)
		},
	)
}

func (w *SkillStoreWrapper) AdoptSkill(
	request *skillBundle.AdoptSkillRequest,
) (artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() (artifact.Artifact, error) {
		if request == nil {
			return artifact.Artifact{}, errors.New("skill adoption request is required")
		}
		value, err := w.api.AdoptSkill(context.Background(), *request)
		if err != nil {
			return value, err
		}

		return value, nil
	})
}

func (w *SkillStoreWrapper) PinSkill(
	request *skillBundle.PinSkillRequest,
) (artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() (artifact.Artifact, error) {
		if request == nil {
			return artifact.Artifact{}, errors.New("skill pin request is required")
		}
		value, err := w.api.PinSkill(context.Background(), *request)
		if err != nil {
			return value, err
		}

		return value, nil
	})
}

func (w *SkillStoreWrapper) ListBundleSkills(
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() ([]artifact.Artifact, error) {
		return w.api.ListSkills(context.Background(), ref)
	})
}

// RuntimeCatalogIDForCollection maps durable Collection identity to the
// runtime-owned opaque catalog identity. It does not read Skill content.
func (w *SkillStoreWrapper) RuntimeCatalogIDForCollection(
	ref collection.CollectionRef,
) (skillRuntime.CatalogID, error) {
	return middleware.WithRecoveryResp(
		func() (skillRuntime.CatalogID, error) {
			return aggregatecatalog.CollectionCatalogID(ref)
		},
	)
}

// ResolveArtifactSkill performs the Store-owned ArtifactRef to Agent Skills
// translation. The returned Collection identifies the catalog that must be
// synchronized before Definition is used in a runtime session.
func (w *SkillStoreWrapper) ResolveArtifactSkill(
	ref artifact.ArtifactRef,
) (skillStore.ResolvedArtifactSkill, error) {
	return middleware.WithRecoveryResp(
		func() (skillStore.ResolvedArtifactSkill, error) {
			return w.router.ResolveArtifactSkill(
				context.Background(),
				ref,
			)
		},
	)
}

func (w *SkillStoreWrapper) SetSkillEnabled(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() (artifact.Artifact, error) {
		value, err := w.api.SetSkillEnabled(
			context.Background(),
			ref,
			expectedRevision,
			enabled,
		)
		if err != nil {
			return value, err
		}
		return value, nil
	})
}

func (w *SkillStoreWrapper) UnadoptSkill(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	suppress bool,
) error {
	return middleware.WithRecovery(func() error {
		return w.api.UnadoptSkill(
			context.Background(),
			ref,
			expectedRevision,
			suppress,
		)
	})
}

func (w *SkillStoreWrapper) PurgeSkill(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		return w.api.PurgeSkill(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *SkillStoreWrapper) close() {
	if w == nil {
		return
	}

	api := w.api
	w.builtInInstaller = nil

	w.router = nil
	w.api = nil

	if api != nil {
		_ = api.Close()
	}
}
