package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/providerapi"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/providerapi"
	workspaceProviderAPI "github.com/flexigpt/flexigpt-app/internal/workspace/store/providerapi"
)

func composeArtifactStore(
	ctx context.Context,
	baseDirectory string,
) (*compositionapi.Store, error) {
	if err := artifactbuiltin.ValidateApplicationTopology(); err != nil {
		return nil, err
	}

	workspaceProvider, err := workspaceProviderAPI.NewProvider()
	if err != nil {
		return nil, err
	}

	skillPlugin, err := skillProviderAPI.NewProvider()
	if err != nil {
		return nil, err
	}

	mcpProvider, err := mcpProviderAPI.NewProvider()
	if err != nil {
		return nil, err
	}

	providers := []providerapi.Provider{
		workspaceProvider,
		skillPlugin,
		mcpProvider,
	}

	return compositionapi.Open(
		ctx,
		compositionapi.Config{
			BaseDirectory:    baseDirectory,
			Providers:        providers,
			ProtectedRootIDs: artifactbuiltin.ProtectedRootIDs(),
			RetainedRoots:    artifactbuiltin.RetainedRootDrafts(),
		},
	)
}
