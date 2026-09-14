package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/format/markdown"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/provider"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/providerapi"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/providerapi"
)

func composeArtifactStore(
	ctx context.Context,
	baseDirectory string,
) (*compositionapi.Store, error) {
	if err := builtin.ValidateApplicationTopology(); err != nil {
		return nil, err
	}

	canonicalProvider, err := provider.New()
	if err != nil {
		return nil, err
	}

	markdownProvider, err := markdown.NewProvider()
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
		canonicalProvider,
		markdownProvider,
		skillPlugin,
		mcpProvider,
	}

	return compositionapi.Open(
		ctx,
		compositionapi.Config{
			BaseDirectory:    baseDirectory,
			Providers:        providers,
			ProtectedRootIDs: builtin.ProtectedRootIDs(),
			RetainedRoots:    builtin.RetainedRootDrafts(),
		},
	)
}
