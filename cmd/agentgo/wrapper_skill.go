package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/builtin/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/builtin/metadata"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/skillbundle"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
	workspaceSpec "github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type SkillBundleWrapper struct {
	api     *skillbundle.API
	runtime *skillruntime.SkillRuntime
}

func InitSkillBundleWrapper(
	wrapper *SkillBundleWrapper,
	components *system.Components,
	workspaceSkills *skilladapter.Adapter,
) error {
	if wrapper == nil || components == nil || workspaceSkills == nil {
		return errors.New("skill bundle wrapper dependencies are incomplete")
	}

	idGenerator := uuidutil.UUIDv7Generator{}
	autoAdoptionIDProvider := skillbundle.ArtifactIDProviderFunc(
		func(ctx context.Context) (basespec.ArtifactID, error) {
			id, err := idGenerator.NewID(ctx)
			if err != nil {
				return "", err
			}
			return basespec.ArtifactID(id), nil
		},
	)

	api, err := skillbundle.New(skillbundle.Dependencies{
		Roots:                  components.Roots,
		Sources:                components.Sources,
		Collections:            components.Collections,
		Artifacts:              components.Artifacts,
		Refresh:                components.Refresh,
		Catalogs:               components.Catalogs,
		Definitions:            components.Definitions,
		SourceRuntime:          components.SourceRuntime,
		HasDecoder:             components.HasDecoder,
		DecoderFingerprint:     components.DecoderFingerprint,
		RootMutationPolicy:     components.RootMutationPolicy(),
		AutoAdoptionIDProvider: autoAdoptionIDProvider,

		GetManagedSourceState: func(
			ctx context.Context,
			rootID basespec.RootID,
			sourceID basespec.SourceID,
		) (sourceSummary source.Summary, generation string, err error) {
			result, err := components.GetManagedSourceState(ctx, rootID, sourceID)
			if err != nil {
				return source.Summary{}, "", err
			}
			return result.Source, result.Generation, nil
		},
		PublishManagedPackage: func(
			ctx context.Context,
			rootID basespec.RootID,
			sourceID basespec.SourceID,
			expectedSourceRevision uint64,
			publication source.ManagedPackagePublication,
		) (sourceSummary source.Summary, generation string, err error) {
			result, err := components.PublishManagedPackage(
				ctx,
				rootID,
				sourceID,
				expectedSourceRevision,
				publication,
			)
			if err != nil {
				return source.Summary{}, "", err
			}
			return result.Source, result.Generation, nil
		},
		PublishProtectedManagedPackage: func(
			ctx context.Context,
			rootID basespec.RootID,
			sourceID basespec.SourceID,
			expectedSourceRevision uint64,
			publication source.ManagedPackagePublication,
		) (sourceSummary source.Summary, generation string, err error) {
			result, err := components.PublishProtectedManagedPackage(
				ctx,
				rootID,
				sourceID,
				expectedSourceRevision,
				publication,
			)
			if err != nil {
				return source.Summary{}, "", err
			}
			return result.Source, result.Generation, nil
		},
		RemoveManagedPackage: func(
			ctx context.Context,
			rootID basespec.RootID,
			sourceID basespec.SourceID,
			expectedSourceRevision uint64,
			directory basespec.Locator,
			expectedGeneration string,
		) (sourceSummary source.Summary, generation string, err error) {
			result, err := components.RemoveManagedPackage(
				ctx,
				rootID,
				sourceID,
				expectedSourceRevision,
				directory,
				expectedGeneration,
			)
			if err != nil {
				return source.Summary{}, "", err
			}
			return result.Source, result.Generation, nil
		},
	})
	if err != nil {
		return err
	}

	router, err := skillruntime.NewArtifactRouter(
		components.ArtifactReader,
		components.CollectionReader,
	)
	if err != nil {
		return err
	}
	workspaceResolver, err := skilladapter.NewRuntimeResolver(workspaceSkills)
	if err != nil {
		return err
	}
	bundleResolver, err := skillbundle.NewRuntimeResolver(api)
	if err != nil {
		return err
	}
	if err := router.Register(
		workspaceSpec.CollectionKind,
		workspaceResolver,
	); err != nil {
		return err
	}
	if err := router.Register(
		skillbundle.CollectionKind,
		bundleResolver,
	); err != nil {
		return err
	}

	runtime, err := skillruntime.NewSkillRuntime(
		skillruntime.WithArtifactResolver(router),
	)
	if err != nil {
		return err
	}

	wrapper.api = api
	wrapper.runtime = runtime

	registry, err := metadata.LoadRegistry()
	if err != nil {
		wrapper.close()
		return err
	}
	packages, err := fs.Sub(
		builtin.BuiltInSkillBundlesFS,
		builtin.BuiltInSkillBundlesRootDir,
	)
	if err != nil {
		wrapper.close()
		return err
	}
	builtIns, err := artifactbuiltin.NewInstaller(
		artifactbuiltin.InstallerDependencies{
			Topology: components,
			Skills:   api,
			Registry: registry,
			Packages: packages,
		},
	)
	if err != nil {
		wrapper.close()
		return err
	}
	if err := builtIns.Ensure(context.Background()); err != nil {
		wrapper.close()
		return err
	}

	refs, err := api.SkillBundleRefs(context.Background())
	if err != nil {
		wrapper.close()
		return err
	}
	for _, ref := range refs {
		if err := runtime.ResyncCollection(context.Background(), ref); err != nil {
			if components.RootMutationPolicy().IsProtectedRoot(ref.RootID) {
				wrapper.close()
				return err
			}
			slog.Warn(
				"could not load user Skill Bundle into runtime",
				"rootID", ref.RootID,
				"collectionID", ref.CollectionID,
				"error", err,
			)
		}
	}

	return nil
}

func (w *SkillBundleWrapper) CreateSkillBundle(
	request *skillbundle.CreateBundleRequest,
) (skillbundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (skillbundle.Bundle, error) {
		if request == nil {
			return skillbundle.Bundle{}, errors.New("skill bundle request is required")
		}
		value, err := w.api.CreateBundle(context.Background(), *request)

		return value, err
	})
}

// AttachSkillSource attaches a Source already created through Artifact Store
// administration. It intentionally accepts only Source identity and typed
// Skill Bundle role data, never a filesystem path or Source configuration.
func (w *SkillBundleWrapper) AttachSkillSource(
	bundle collection.CollectionRef,
	expectedCollectionRevision uint64,
	draft skillbundle.AttachmentDraft,
) (skillbundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (skillbundle.Bundle, error) {
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
			return value, w.runtime.RemoveCollection(
				context.Background(),
				value.Collection.Ref(),
			)
		}
		if _, err := w.api.RefreshBundle(
			context.Background(),
			value.Collection.Ref(),
		); err != nil {
			return value, err
		}
		return value, w.runtime.ResyncCollection(
			context.Background(),
			value.Collection.Ref(),
		)
	})
}

func (w *SkillBundleWrapper) GetSkillBundle(
	ref collection.CollectionRef,
) (skillbundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (skillbundle.Bundle, error) {
		return w.api.GetBundle(context.Background(), ref)
	})
}

func (w *SkillBundleWrapper) ListSkillBundles(
	rootID basespec.RootID,
) ([]skillbundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() ([]skillbundle.Bundle, error) {
		return w.api.ListBundles(context.Background(), rootID)
	})
}

func (w *SkillBundleWrapper) UpdateSkillBundle(
	request *skillbundle.UpdateBundleRequest,
) (skillbundle.Bundle, error) {
	return middleware.WithRecoveryResp(func() (skillbundle.Bundle, error) {
		if request == nil {
			return skillbundle.Bundle{}, errors.New("skill bundle update is required")
		}
		value, err := w.api.UpdateBundle(context.Background(), *request)
		if err != nil {
			return value, err
		}

		ref := value.Collection.Ref()
		if !value.Collection.Enabled {
			return value, w.runtime.RemoveCollection(
				context.Background(),
				ref,
			)
		}
		if _, err := w.api.RefreshBundle(
			context.Background(),
			ref,
		); err != nil {
			return value, err
		}
		return value, w.runtime.ResyncCollection(
			context.Background(),
			ref,
		)
	})
}

func (w *SkillBundleWrapper) RetireSkillBundle(
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

		return value, w.runtime.RemoveCollection(
			context.Background(),
			ref,
		)
	})
}

func (w *SkillBundleWrapper) PurgeSkillBundle(
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
		return w.runtime.RemoveCollection(context.Background(), ref)
	})
}

func (w *SkillBundleWrapper) RefreshSkillBundle(
	ref collection.CollectionRef,
) error {
	return middleware.WithRecovery(func() error {
		_, err := w.api.RefreshBundle(context.Background(), ref)
		if err != nil {
			return err
		}
		return w.runtime.ResyncCollection(context.Background(), ref)
	})
}

func (w *SkillBundleWrapper) GetLinkedPortableSkillBundleJSON(
	ref collection.CollectionRef,
) (string, error) {
	return middleware.WithRecoveryResp(func() (string, error) {
		value, err := w.api.BuildLinkedPortableBundleJSON(
			context.Background(),
			ref,
		)
		if err != nil {
			return "", err
		}
		return string(value), nil
	})
}

func (w *SkillBundleWrapper) CreateManagedSkill(
	request *skillbundle.CreateManagedSkillRequest,
) (skillbundle.CreateManagedSkillResponse, error) {
	return middleware.WithRecoveryResp(
		func() (skillbundle.CreateManagedSkillResponse, error) {
			if request == nil {
				return skillbundle.CreateManagedSkillResponse{},
					errors.New("managed skill request is required")
			}
			value, err := w.api.CreateManagedSkill(context.Background(), *request)
			if err != nil {
				return value, err
			}
			ref := collection.CollectionRef{
				RootID:       value.Artifact.RootID,
				CollectionID: value.Artifact.CollectionID,
			}
			return value, w.runtime.ResyncCollection(context.Background(), ref)
		},
	)
}

func (w *SkillBundleWrapper) GetManagedSkillDocument(
	ref artifact.ArtifactRef,
) (skillbundle.ManagedSkillDocument, error) {
	return middleware.WithRecoveryResp(
		func() (skillbundle.ManagedSkillDocument, error) {
			return w.api.GetManagedSkillDocument(context.Background(), ref)
		},
	)
}

func (w *SkillBundleWrapper) AdoptSkill(
	request *skillbundle.AdoptSkillRequest,
) (artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() (artifact.Artifact, error) {
		if request == nil {
			return artifact.Artifact{}, errors.New("skill adoption request is required")
		}
		value, err := w.api.AdoptSkill(context.Background(), *request)
		if err != nil {
			return value, err
		}

		return value, w.runtime.ResyncCollection(
			context.Background(),
			collection.CollectionRef{
				RootID:       value.RootID,
				CollectionID: value.CollectionID,
			},
		)
	})
}

func (w *SkillBundleWrapper) PinSkill(
	request *skillbundle.PinSkillRequest,
) (artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() (artifact.Artifact, error) {
		if request == nil {
			return artifact.Artifact{}, errors.New("skill pin request is required")
		}
		value, err := w.api.PinSkill(context.Background(), *request)
		if err != nil {
			return value, err
		}

		return value, w.runtime.ResyncCollection(
			context.Background(),
			collection.CollectionRef{
				RootID:       value.RootID,
				CollectionID: value.CollectionID,
			},
		)
	})
}

func (w *SkillBundleWrapper) ListBundleSkills(
	ref collection.CollectionRef,
) ([]artifact.Artifact, error) {
	return middleware.WithRecoveryResp(func() ([]artifact.Artifact, error) {
		return w.api.ListSkills(context.Background(), ref)
	})
}

func (w *SkillBundleWrapper) SetSkillEnabled(
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
		collectionRef := collection.CollectionRef{
			RootID:       value.RootID,
			CollectionID: value.CollectionID,
		}
		return value, w.runtime.ResyncCollection(
			context.Background(),
			collectionRef,
		)
	})
}

func (w *SkillBundleWrapper) UnadoptSkill(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	suppress bool,
) error {
	return middleware.WithRecovery(func() error {
		value, err := w.api.GetSkill(context.Background(), ref)
		if err != nil {
			return err
		}
		if err := w.api.UnadoptSkill(
			context.Background(),
			ref,
			expectedRevision,
			suppress,
		); err != nil {
			return err
		}
		return w.runtime.ResyncCollection(
			context.Background(),
			collection.CollectionRef{
				RootID:       value.RootID,
				CollectionID: value.CollectionID,
			},
		)
	})
}

func (w *SkillBundleWrapper) PurgeSkill(
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		value, err := w.api.GetSkill(context.Background(), ref)
		if err != nil {
			return err
		}
		if err := w.api.PurgeSkill(
			context.Background(),
			ref,
			expectedRevision,
		); err != nil {
			return err
		}
		return w.runtime.ResyncCollection(
			context.Background(),
			collection.CollectionRef{
				RootID:       value.RootID,
				CollectionID: value.CollectionID,
			},
		)
	})
}

func (w *SkillBundleWrapper) CreateSkillSession(
	request *skillruntime.CreateSkillSessionRequest,
) (*skillruntime.CreateSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntime.CreateSkillSessionResponse, error) {
		return w.runtime.CreateSkillSession(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) CloseSkillSession(
	request *skillruntime.CloseSkillSessionRequest,
) (*skillruntime.CloseSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntime.CloseSkillSessionResponse, error) {
		return w.runtime.CloseSkillSession(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) GetSkillsPrompt(
	request *skillruntime.GetSkillsPromptRequest,
) (*skillruntime.GetSkillsPromptResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntime.GetSkillsPromptResponse, error) {
		return w.runtime.GetSkillsPrompt(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) ListRuntimeSkills(
	request *skillruntime.ListRuntimeSkillsRequest,
) (*skillruntime.ListRuntimeSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntime.ListRuntimeSkillsResponse, error) {
		return w.runtime.ListRuntimeSkills(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) RenderSkill(
	request *skillruntime.RenderSkillRequest,
) (*skillruntime.RenderSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntime.RenderSkillResponse, error) {
		return w.runtime.RenderSkill(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) InvokeSkillTool(
	request *skillruntime.InvokeSkillToolRequest,
) (*skillruntime.InvokeSkillToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntime.InvokeSkillToolResponse, error) {
		return w.runtime.InvokeSkillTool(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) close() {
	if w == nil {
		return
	}

	runtime := w.runtime
	api := w.api
	w.runtime = nil
	w.api = nil

	if runtime != nil {
		runtime.Close()
	}
	if api != nil {
		_ = api.Close()
	}
}
