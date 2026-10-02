package main

import (
	"context"
	"io/fs"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providercanonical"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providermarkdown"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/providers/secretstore/keyringmapstore"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/providerapi"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/workspace/defaultpolicy"
)

func composeArtifactStore(
	ctx context.Context,
	baseDirectory string,
) (*local.Store, error) {
	if err := documentTopology.ValidateApplicationTopology(); err != nil {
		return nil, err
	}

	canonicalProvider, err := providercanonical.New()
	if err != nil {
		return nil, err
	}

	markdownProvider, err := providermarkdown.NewProvider()
	if err != nil {
		return nil, err
	}

	skillProvider, err := skillProviderAPI.NewProvider()
	if err != nil {
		return nil, err
	}

	mcpProvider, err := mcpProviderAPI.NewProvider()
	if err != nil {
		return nil, err
	}

	workspaceFS, err := builtin.EmbeddedWorkspacePackages()
	if err != nil {
		return nil, err
	}

	providers := []provider.Provider{
		canonicalProvider,
		markdownProvider,
		skillProvider,
		mcpProvider,
	}

	secretValues, err := keyringmapstore.New(
		filepath.Join(
			baseDirectory,
			model.ArtifactStoreSecretValuesFileName,
		),
		keyringmapstore.Config{},
	)
	if err != nil {
		return nil, err
	}

	protectedOverlayNamespaces := append(
		modelOverlay.Namespaces(),
		mcpOverlay.Namespaces()...,
	)
	storeOverlayNamespaces := append(
		modelOverlay.StoreNamespaces(),
		mcpOverlay.StoreNamespaces()...,
	)

	return local.Open(
		ctx,
		local.Config{
			BaseDirectory: baseDirectory,
			EmbeddedProviders: map[string]fs.FS{
				defaultpolicy.ProviderKey: workspaceFS,
			},
			Providers:                  providers,
			ProtectedRootIDs:           documentTopology.ProtectedRootIDs(),
			RetainedRoots:              documentTopology.RetainedRootDrafts(),
			ProtectedOverlayNamespaces: protectedOverlayNamespaces,
			StoreOverlayNamespaces:     storeOverlayNamespaces,
			SecretValues:               secretValues,
		},
	)
}
