package artifactsetup

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/keyringmapstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/registration/canonical"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/registration/markdown"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/overlay"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/providerapi"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/overlay"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/providerapi"
)

const secretValuesFileName = "secrets.json"

// Handle owns the local generic Store deployment and the LLM registration
// attachment. The generic Store remains the only aggregate exposing generic
// entity and flow APIs.
type Handle struct {
	Store *compose.Store
	LLM   *llmartifactory.Artifactory

	locatorFactories []locator.Factory

	closeOnce sync.Once
	closeErr  error
}

// LocatorFactories returns independently owned locator factory registrations
// for family and runtime composition.
func (h *Handle) LocatorFactories() []locator.Factory {
	if h == nil {
		return nil
	}
	return append([]locator.Factory(nil), h.locatorFactories...)
}

// Close closes the LLM registration attachment before closing the local
// generic Store deployment and its owned provider resources.
func (h *Handle) Close() error {
	if h == nil {
		return nil
	}
	h.closeOnce.Do(func() {
		if h.LLM != nil {
			h.closeErr = errors.Join(h.closeErr, h.LLM.Close())
			h.LLM = nil
		}
		if h.Store != nil {
			h.closeErr = errors.Join(h.closeErr, h.Store.Close())
			h.Store = nil
		}
		h.locatorFactories = nil
	})
	return h.closeErr
}

// OpenArtifactStore opens FlexiGPT's local generic Artifact Store deployment,
// then attaches the application-selected LLM declaration registrations.
// Product content, topology, protected namespaces, and local keyring identity
// remain application setup concerns rather than LLM library concerns.
func OpenArtifactStore(
	ctx context.Context,
	baseDirectory string,
) (*Handle, error) {
	if err := topology.ValidateApplicationTopology(); err != nil {
		return nil, err
	}

	canon, err := canonical.NewRegistration()
	if err != nil {
		return nil, err
	}
	md, err := markdown.NewRegistration()
	if err != nil {
		return nil, err
	}
	skill, err := skillProviderAPI.NewRegistration()
	if err != nil {
		return nil, err
	}
	mcp, err := mcpProviderAPI.NewRegistration()
	if err != nil {
		return nil, err
	}

	locators, err := locator.NewRegistry(
		canon.LocatorFactories()...,
	)
	if err != nil {
		return nil, err
	}

	codecs := canon.SchemaCodecs()
	decoders := canon.Decoders()
	decoders = append(decoders, md.Decoders()...)
	decoders = append(decoders, skill.Decoders()...)
	decoders = append(decoders, mcp.Decoders()...)

	workspaceFS, err := artifactbuiltin.EmbeddedWorkspacePackages()
	if err != nil {
		return nil, err
	}

	values, err := keyringmapstore.New(
		filepath.Join(baseDirectory, secretValuesFileName),
		keyringmapstore.Config{},
	)
	if err != nil {
		return nil, err
	}

	protectedNamespaces := append(
		modelOverlay.Namespaces(),
		overlay.Namespaces()...,
	)
	storeNamespaces := append(
		modelOverlay.StoreNamespaces(),
		overlay.StoreNamespaces()...,
	)

	store, err := local.Open(ctx, local.Config{
		BaseDirectory: baseDirectory,
		EmbeddedProviders: map[string]fs.FS{
			"workspace-default-policy": workspaceFS,
		},
		SchemaCodecs: codecs,
		Decoders:     decoders,

		ProtectedRootIDs:           topology.ProtectedRootIDs(),
		RetainedRoots:              topology.RetainedRootDrafts(),
		ProtectedOverlayNamespaces: protectedNamespaces,
		StoreOverlayNamespaces:     storeNamespaces,
		SecretValues:               values,
	})
	if err != nil {
		return nil, errors.Join(err, values.Close())
	}

	llm, err := llmartifactory.Open(ctx, llmartifactory.Config{
		Store:            store,
		SchemaCodecs:     codecs,
		Decoders:         decoders,
		LocatorFactories: locators.Factories(),
	})
	if err != nil {
		return nil, errors.Join(err, store.Close())
	}

	return &Handle{
		Store:            store,
		LLM:              llm,
		locatorFactories: locators.Factories(),
	}, nil
}
