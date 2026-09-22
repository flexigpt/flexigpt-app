package main

import (
	"context"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providercanonical"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providermarkdown"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/providerapi"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/workspace/defaultpolicy"
)

func composeArtifactStore(
	ctx context.Context,
	baseDirectory string,
) (*compositionapi.Store, error) {
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

	providers := []providerapi.Provider{
		canonicalProvider,
		markdownProvider,
		skillProvider,
		mcpProvider,
	}

	return compositionapi.Open(
		ctx,
		compositionapi.Config{
			BaseDirectory: baseDirectory,
			EmbeddedProviders: map[string]fs.FS{
				defaultpolicy.ProviderKey: workspaceFS,
			},
			Providers:        providers,
			ProtectedRootIDs: documentTopology.ProtectedRootIDs(),
			RetainedRoots:    documentTopology.RetainedRootDrafts(),
		},
	)
}
