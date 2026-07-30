package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	"github.com/flexigpt/flexigpt-app/internal/skillbundle"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime"
	skillruntimeSpec "github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

type SkillBundleWrapper struct {
	api     *skillbundle.API
	runtime *skillruntime.SkillRuntime

	mu      sync.Mutex
	managed map[collection.CollectionRef]struct{}

	bootstrapContext context.Context
	bootstrapCancel  context.CancelFunc
	bootstrapWG      sync.WaitGroup
}

func InitSkillBundleWrapper(
	wrapper *SkillBundleWrapper,
	components *system.Components,
	workspaceSkills *skilladapter.Adapter,
) error {
	if wrapper == nil || components == nil || workspaceSkills == nil {
		return errors.New("skill bundle wrapper dependencies are incomplete")
	}

	api, err := skillbundle.New(skillbundle.Dependencies{
		Roots:              components.Roots,
		Sources:            components.Sources,
		Collections:        components.Collections,
		Artifacts:          components.Artifacts,
		Refresh:            components.Refresh,
		Catalogs:           components.Catalogs,
		Definitions:        components.Definitions,
		SourceRuntime:      components.SourceRuntime,
		HasDecoder:         components.HasDecoder,
		DecoderFingerprint: components.DecoderFingerprint,
		IDGenerator:        uuidutil.UUIDv7Generator{},
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
		"workspace.collection",
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

	parent, cancel := context.WithCancel(context.Background())
	wrapper.api = api
	wrapper.runtime = runtime
	wrapper.managed = map[collection.CollectionRef]struct{}{}
	wrapper.bootstrapContext = parent
	wrapper.bootstrapCancel = cancel

	if err := wrapper.bootstrapEmbeddedBuiltIns(context.Background()); err != nil {
		wrapper.close()
		return err
	}
	wrapper.syncKnownSkillBundles()

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
		if err == nil {
			w.syncBundle(value.Collection.Ref())
		}
		return value, err
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
		if err == nil {
			w.syncBundle(request.Bundle)
		}
		return value, err
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
		if err == nil {
			w.removeBundle(ref)
		}
		return value, err
	})
}

func (w *SkillBundleWrapper) PurgeSkillBundle(
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	return middleware.WithRecovery(func() error {
		err := w.api.PurgeBundle(context.Background(), ref, expectedRevision)
		if err == nil {
			w.removeBundle(ref)
		}
		return err
	})
}

func (w *SkillBundleWrapper) RefreshSkillBundle(
	ref collection.CollectionRef,
) error {
	return middleware.WithRecovery(func() error {
		_, err := w.api.RefreshBundle(context.Background(), ref)
		if err == nil {
			w.syncBundle(ref)
		}
		return err
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
			if err == nil {
				w.syncBundle(request.Bundle)
			}
			return value, err
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
		if err == nil {
			w.syncBundle(request.Bundle)
		}
		return value, err
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
		if err == nil {
			w.syncBundle(request.Bundle)
		}
		return value, err
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
		if err == nil {
			w.syncBundle(collection.CollectionRef{
				RootID:       value.RootID,
				CollectionID: value.CollectionID,
			})
		}
		return value, err
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
		err = w.api.UnadoptSkill(
			context.Background(),
			ref,
			expectedRevision,
			suppress,
		)
		if err == nil {
			w.syncBundle(collection.CollectionRef{
				RootID:       value.RootID,
				CollectionID: value.CollectionID,
			})
		}
		return err
	})
}

func (w *SkillBundleWrapper) CreateSkillSession(
	request *skillruntimeSpec.CreateSkillSessionRequest,
) (*skillruntimeSpec.CreateSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.CreateSkillSessionResponse, error) {
		return w.runtime.CreateSkillSession(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) CloseSkillSession(
	request *skillruntimeSpec.CloseSkillSessionRequest,
) (*skillruntimeSpec.CloseSkillSessionResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.CloseSkillSessionResponse, error) {
		return w.runtime.CloseSkillSession(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) GetSkillsPrompt(
	request *skillruntimeSpec.GetSkillsPromptRequest,
) (*skillruntimeSpec.GetSkillsPromptResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.GetSkillsPromptResponse, error) {
		return w.runtime.GetSkillsPrompt(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) ListRuntimeSkills(
	request *skillruntimeSpec.ListRuntimeSkillsRequest,
) (*skillruntimeSpec.ListRuntimeSkillsResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.ListRuntimeSkillsResponse, error) {
		return w.runtime.ListRuntimeSkills(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) RenderSkill(
	request *skillruntimeSpec.RenderSkillRequest,
) (*skillruntimeSpec.RenderSkillResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.RenderSkillResponse, error) {
		return w.runtime.RenderSkill(context.Background(), request)
	})
}

func (w *SkillBundleWrapper) InvokeSkillTool(
	request *skillruntimeSpec.InvokeSkillToolRequest,
) (*skillruntimeSpec.InvokeSkillToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*skillruntimeSpec.InvokeSkillToolResponse, error) {
		return w.runtime.InvokeSkillTool(context.Background(), request)
	})
}

func BindArtifactStoreSkillBundleSynchronization(
	artifacts *ArtifactStoreWrapper,
	bundles *SkillBundleWrapper,
) error {
	if artifacts == nil || bundles == nil {
		return errors.New("artifact and skill bundle wrappers are required")
	}
	artifacts.subscribeRootMutation(bundles.syncBundlesForRoot)
	return nil
}

func (w *SkillBundleWrapper) syncBundle(ref collection.CollectionRef) {
	if w == nil || w.runtime == nil {
		return
	}
	w.mu.Lock()
	w.managed[ref] = struct{}{}
	w.mu.Unlock()
	w.runtime.RequestCollectionResync(ref)
}

func (w *SkillBundleWrapper) removeBundle(ref collection.CollectionRef) {
	if w == nil || w.runtime == nil {
		return
	}
	w.mu.Lock()
	delete(w.managed, ref)
	w.mu.Unlock()
	w.runtime.RequestCollectionRemoval(ref)
}

func (w *SkillBundleWrapper) syncKnownSkillBundles() {
	w.scheduleBundleSynchronization("", false)
}

func (w *SkillBundleWrapper) syncBundlesForRoot(rootID basespec.RootID) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return
	}
	w.scheduleBundleSynchronization(rootID, true)
}

func (w *SkillBundleWrapper) scheduleBundleSynchronization(
	rootID basespec.RootID,
	filterRoot bool,
) {
	if w == nil || w.api == nil || w.runtime == nil ||
		w.bootstrapContext == nil {
		return
	}
	w.bootstrapWG.Go(func() {
		ctx, cancel := context.WithTimeout(w.bootstrapContext, 30*time.Second)
		defer cancel()

		if err := w.bootstrapEmbeddedBuiltIns(ctx); err != nil {
			return
		}

		refs, err := w.api.SkillBundleRefs(ctx)
		if err != nil || ctx.Err() != nil {
			return
		}
		active := make(map[collection.CollectionRef]struct{}, len(refs))
		for _, ref := range refs {
			if filterRoot && ref.RootID != rootID {
				continue
			}
			active[ref] = struct{}{}
			w.syncBundle(ref)
		}

		w.mu.Lock()
		managed := make([]collection.CollectionRef, 0, len(w.managed))
		for ref := range w.managed {
			if filterRoot && ref.RootID != rootID {
				continue
			}
			managed = append(managed, ref)
		}
		w.mu.Unlock()

		for _, ref := range managed {
			if _, exists := active[ref]; !exists {
				w.removeBundle(ref)
			}
		}
	})
}

func (w *SkillBundleWrapper) bootstrapEmbeddedBuiltIns(
	ctx context.Context,
) error {
	if w == nil || w.api == nil {
		return errors.New("skill bundle API is not initialized")
	}
	values, err := w.api.BootstrapEmbeddedBuiltIns(ctx)
	if err != nil {
		return err
	}
	for _, value := range values {
		//nolint:contextcheck // Sync op.
		w.syncBundle(value.Collection.Ref())
	}
	return nil
}

func (w *SkillBundleWrapper) close() {
	if w == nil {
		return
	}
	if w.bootstrapCancel != nil {
		w.bootstrapCancel()
	}
	w.bootstrapWG.Wait()
	if w.runtime != nil {
		w.runtime.Close()
	}
	if w.api != nil {
		_ = w.api.Close()
	}
	w.runtime = nil
	w.api = nil
}
