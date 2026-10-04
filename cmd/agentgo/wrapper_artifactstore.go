package main

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/locator"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providercanonical"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/providermarkdown"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/keyringmapstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/providerapi"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/workspace/defaultpolicy"
)

const artifactStoreSecretValuesFileName = "secrets.json"

func composeArtifactStore(
	ctx context.Context,
	baseDirectory string,
) (*compose.Store, []locator.Factory, error) {
	if err := documentTopology.ValidateApplicationTopology(); err != nil {
		return nil, nil, err
	}

	canonicalRegistration, err := providercanonical.NewRegistration()
	if err != nil {
		return nil, nil, err
	}

	markdownRegistration, err := providermarkdown.NewRegistration()
	if err != nil {
		return nil, nil, err
	}

	skillRegistration, err := skillProviderAPI.NewRegistration()
	if err != nil {
		return nil, nil, err
	}

	mcpRegistration, err := mcpProviderAPI.NewRegistration()
	if err != nil {
		return nil, nil, err
	}

	locatorRegistry, err := locator.NewRegistry(
		canonicalRegistration.LocatorFactories()...,
	)
	if err != nil {
		return nil, nil, err
	}

	schemaCodecs := canonicalRegistration.SchemaCodecs()

	decoders := canonicalRegistration.Decoders()
	decoders = append(
		decoders,
		markdownRegistration.Decoders()...,
	)
	decoders = append(
		decoders,
		skillRegistration.Decoders()...,
	)
	decoders = append(
		decoders,
		mcpRegistration.Decoders()...,
	)

	workspaceFS, err := builtin.EmbeddedWorkspacePackages()
	if err != nil {
		return nil, nil, err
	}

	secretValues, err := keyringmapstore.New(
		filepath.Join(
			baseDirectory,
			artifactStoreSecretValuesFileName,
		),
		keyringmapstore.Config{},
	)
	if err != nil {
		return nil, nil, err
	}

	protectedOverlayNamespaces := append(
		modelOverlay.Namespaces(),
		mcpOverlay.Namespaces()...,
	)
	storeOverlayNamespaces := append(
		modelOverlay.StoreNamespaces(),
		mcpOverlay.StoreNamespaces()...,
	)

	store, err := local.Open(
		ctx,
		local.Config{
			BaseDirectory: baseDirectory,
			EmbeddedProviders: map[string]fs.FS{
				defaultpolicy.ProviderKey: workspaceFS,
			},
			SchemaCodecs: schemaCodecs,
			Decoders:     decoders,

			ProtectedRootIDs:           documentTopology.ProtectedRootIDs(),
			RetainedRoots:              documentTopology.RetainedRootDrafts(),
			ProtectedOverlayNamespaces: protectedOverlayNamespaces,
			StoreOverlayNamespaces:     storeOverlayNamespaces,
			SecretValues:               secretValues,
		},
	)
	if err != nil {
		// "local.Open" retains caller ownership of a supplied backend on a
		// failed open. This composition owns it until successful assembly.
		return nil, nil, errors.Join(err, secretValues.Close())
	}

	return store, locatorRegistry.Factories(), nil
}
