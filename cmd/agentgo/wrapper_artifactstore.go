package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	mcpSchemaadapter "github.com/flexigpt/flexigpt-app/internal/mcp/store/schemaadapter"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
	skillBundle "github.com/flexigpt/flexigpt-app/internal/skill/store/bundle"
	"github.com/flexigpt/flexigpt-app/internal/workspace"
)

type ArtifactStoreWrapper struct {
	api             *artifactstore.API
	componentsClose func() error
}

func InitArtifactStoreWrapper(
	wrapper *ArtifactStoreWrapper,
	baseDirectory string,
) error {
	if wrapper == nil {
		return errors.New("artifact store wrapper is required")
	}
	if err := artifactbuiltin.ValidateApplicationTopology(); err != nil {
		return err
	}

	workspaceConfig := workspace.DefaultConfig()
	workspaceProvider, err := workspace.NewProvider(workspaceConfig.ProviderConfig())
	if err != nil {
		return err
	}

	skillProvider, err := skillBundle.NewProvider()
	if err != nil {
		return err
	}

	mcpProvider, err := mcpSchemaadapter.NewProvider()
	if err != nil {
		return err
	}

	rootPolicy, err := protection.NewSetRootPolicy(
		artifactbuiltin.ProtectedRootIDs(),
		artifactbuiltin.RetainedRootIDs(),
	)
	if err != nil {
		return err
	}

	components, err := system.Open(
		context.Background(),
		system.Config{
			BaseDirectory: baseDirectory,
			ArtifactProviders: []providerapi.Provider{
				workspaceProvider,
				skillProvider,
				mcpProvider,
			},
			RootMutationPolicy: rootPolicy,
		},
	)
	if err != nil {
		return err
	}

	api, err := artifactstore.New(components)
	if err != nil {
		_ = components.Close()
		return err
	}

	for _, draft := range artifactbuiltin.RetainedRootDrafts() {
		if _, err := api.CreateArtifactRoot(
			context.Background(),
			&artifactstore.CreateArtifactRootRequest{
				Body: &draft,
			},
		); err != nil {
			api.Close()
			_ = components.Close()
			return fmt.Errorf("ensure retained application Root %q: %w", draft.ID, err)
		}
	}

	wrapper.api = api
	wrapper.componentsClose = components.Close

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

func (w *ArtifactStoreWrapper) Store() *artifactstore.API {
	if w == nil {
		return nil
	}
	return w.api
}

func (w *ArtifactStoreWrapper) close() {
	if w == nil {
		return
	}

	api := w.api
	closeStore := w.componentsClose
	w.api = nil
	w.componentsClose = nil

	if api != nil {
		api.Close()
	}
	if closeStore == nil {
		return
	}
	if err := closeStore(); err != nil {
		slog.Error("close artifact store", "error", err)
	}
}
