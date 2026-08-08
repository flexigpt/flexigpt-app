package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/artifact"
	skillBundle "github.com/flexigpt/flexigpt-app/internal/skill/bundle"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

type ArtifactStoreWrapper struct {
	api             *artifactstore.API
	components      *system.Components
	builtInTopology builtin.Registry
}

const defaultWorkspaceRootID basespec.RootID = "0198f097-0d5b-7000-8000-000000000001"

const (
	defaultWorkspaceRootDisplayName = "FlexiGPT Workspaces"
	defaultWorkspaceRootDescription = "Local namespace for user Workspace collections."
)

func defaultWorkspaceRootDraft() root.RootDraft {
	return root.RootDraft{
		ID:          defaultWorkspaceRootID,
		DisplayName: defaultWorkspaceRootDisplayName,
		Description: defaultWorkspaceRootDescription,
	}
}

func InitArtifactStoreWrapper(
	wrapper *ArtifactStoreWrapper,
	baseDirectory string,
) error {
	if wrapper == nil {
		return errors.New("artifact store wrapper is required")
	}
	registry, err := builtin.LoadRegistry()
	if err != nil {
		return err
	}
	if registry.Root.ID == defaultWorkspaceRootID {
		return fmt.Errorf(
			"%w: protected built-in Root %q and retained Workspace Root must use distinct IDs",
			basespec.ErrInvalid,
			registry.Root.ID,
		)
	}

	skillDecoder, err := skillArtifact.NewDecoder()
	if err != nil {
		return err
	}
	decoders := append(
		workspace.DefaultDecoders(),
		skillDecoder,
	)
	shareableCodecs := []shareable.Codec{
		skillBundle.NewShareableCodec(),
		workspace.NewShareableCodec(),
	}

	components, err := system.Open(
		context.Background(),
		system.Config{
			BaseDirectory: baseDirectory,
			Decoders:      decoders,
			RootMutationPolicy: protection.StaticRootPolicy{
				RootID:         registry.Root.ID,
				RetainedRootID: defaultWorkspaceRootID,
			},
			ShareableCodecs: shareableCodecs,
		},
	)
	if err != nil {
		return err
	}

	if _, err := components.Roots.Create(
		context.Background(),
		defaultWorkspaceRootDraft(),
	); err != nil {
		_ = components.Close()
		return fmt.Errorf("ensure default Workspace Root: %w", err)
	}

	api, err := artifactstore.New(components)
	if err != nil {
		_ = components.Close()
		return err
	}

	wrapper.components = components
	wrapper.api = api
	wrapper.builtInTopology = registry

	return nil
}

func (w *ArtifactStoreWrapper) CreateArtifactRoot(
	request *artifactstore.CreateArtifactRootRequest,
) (*artifactstore.CreateArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.CreateArtifactRootResponse, error) {
			return w.api.CreateArtifactRoot(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) GetArtifactRoot(
	request *artifactstore.GetArtifactRootRequest,
) (*artifactstore.GetArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.GetArtifactRootResponse, error) {
			return w.api.GetArtifactRoot(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) ListArtifactRoots(
	request *artifactstore.ListArtifactRootsRequest,
) (*artifactstore.ListArtifactRootsResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.ListArtifactRootsResponse, error) {
			return w.api.ListArtifactRoots(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) UpdateArtifactRoot(
	request *artifactstore.UpdateArtifactRootRequest,
) (*artifactstore.UpdateArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.UpdateArtifactRootResponse, error) {
			return w.api.UpdateArtifactRoot(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) RetireArtifactRoot(
	request *artifactstore.RetireArtifactRootRequest,
) (*artifactstore.RetireArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.RetireArtifactRootResponse, error) {
			return w.api.RetireArtifactRoot(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) PurgeArtifactRoot(
	request *artifactstore.PurgeArtifactRootRequest,
) (*artifactstore.PurgeArtifactRootResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.PurgeArtifactRootResponse, error) {
			return w.api.PurgeArtifactRoot(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) GetShareableCollectionDocument(
	request *artifactstore.GetShareableCollectionDocumentRequest,
) (*artifactstore.GetShareableCollectionDocumentResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.GetShareableCollectionDocumentResponse, error) {
			return w.api.GetShareableCollectionDocument(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) StoreShareableCollectionDocument(
	request *artifactstore.StoreShareableCollectionDocumentRequest,
) (*artifactstore.StoreShareableCollectionDocumentResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.StoreShareableCollectionDocumentResponse, error) {
			return w.api.StoreShareableCollectionDocument(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) CreateArtifactSource(
	request *artifactstore.CreateArtifactSourceRequest,
) (*artifactstore.CreateArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.CreateArtifactSourceResponse, error) {
			return w.api.CreateArtifactSource(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) GetArtifactSource(
	request *artifactstore.GetArtifactSourceRequest,
) (*artifactstore.GetArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.GetArtifactSourceResponse, error) {
			return w.api.GetArtifactSource(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) ListArtifactSources(
	request *artifactstore.ListArtifactSourcesRequest,
) (*artifactstore.ListArtifactSourcesResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.ListArtifactSourcesResponse, error) {
			return w.api.ListArtifactSources(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) UpdateArtifactSource(
	request *artifactstore.UpdateArtifactSourceRequest,
) (*artifactstore.UpdateArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.UpdateArtifactSourceResponse, error) {
			return w.api.UpdateArtifactSource(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) RetireArtifactSource(
	request *artifactstore.RetireArtifactSourceRequest,
) (*artifactstore.RetireArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.RetireArtifactSourceResponse, error) {
			return w.api.RetireArtifactSource(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) PurgeArtifactSource(
	request *artifactstore.PurgeArtifactSourceRequest,
) (*artifactstore.PurgeArtifactSourceResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.PurgeArtifactSourceResponse, error) {
			return w.api.PurgeArtifactSource(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) ListArtifactSourceKinds(
	request *artifactstore.ListArtifactSourceKindsRequest,
) (*artifactstore.ListArtifactSourceKindsResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.ListArtifactSourceKindsResponse, error) {
			return w.api.ListArtifactSourceKinds(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) GetManagedSourceState(
	request *artifactstore.GetManagedSourceStateRequest,
) (*artifactstore.GetManagedSourceStateResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.GetManagedSourceStateResponse, error) {
			return w.api.GetManagedSourceState(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) PublishManagedSourcePackage(
	request *artifactstore.PublishManagedSourcePackageRequest,
) (*artifactstore.PublishManagedSourcePackageResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.PublishManagedSourcePackageResponse, error) {
			return w.api.PublishManagedSourcePackage(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) RemoveManagedSourcePackage(
	request *artifactstore.RemoveManagedSourcePackageRequest,
) (*artifactstore.RemoveManagedSourcePackageResponse, error) {
	return middleware.WithRecoveryResp(
		func() (*artifactstore.RemoveManagedSourcePackageResponse, error) {
			return w.api.RemoveManagedSourcePackage(
				context.Background(),
				request,
			)
		},
	)
}

func (w *ArtifactStoreWrapper) close() {
	if w == nil {
		return
	}

	api := w.api
	components := w.components
	w.builtInTopology = builtin.Registry{}
	w.api = nil
	w.components = nil

	if api != nil {
		api.Close()
	}
	if components == nil {
		return
	}
	if err := components.Close(); err != nil {
		slog.Error("close artifact store", "error", err)
	}
}
