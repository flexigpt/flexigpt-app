package artifactsetup

import (
	"context"
	"errors"
	"path/filepath"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/keyringmapstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/registration"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/workspace"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/overlay"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/overlay"
)

const secretValuesFileName = "secrets.json"

// Handle owns the local generic Store deployment and the LLM registration
// attachment. The generic Store remains the only aggregate exposing generic
// entity and flow APIs.
type Handle struct {
	Store *compose.Store
	LLM   *llmartifactory.Artifactory

	closeOnce sync.Once
	closeErr  error
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

	registrations, err := registration.New()
	if err != nil {
		return nil, err
	}

	workspacePolicySource, err := workspace.DefaultPolicySource()
	if err != nil {
		return nil, err
	}
	workspaceFS, err := artifactbuiltin.EmbeddedWorkspacePackages()
	if err != nil {
		return nil, err
	}
	workspaceContentDigest, err := iofs.ContentDigest(
		ctx,
		workspaceFS,
	)
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
		mcpOverlay.Namespaces()...,
	)
	storeNamespaces := append(
		modelOverlay.StoreNamespaces(),
		mcpOverlay.StoreNamespaces()...,
	)

	store, err := local.Open(ctx, local.Config{
		BaseDirectory: baseDirectory,
		EmbeddedProviders: map[string]iofs.ProviderRegistration{
			workspacePolicySource.ProviderKey: {
				Filesystem: workspaceFS,
				Immutable: &iofs.ImmutableContentEvidence{
					Revision:      workspacePolicySource.Policy.Version,
					ContentDigest: workspaceContentDigest,
				},
			},
		},
		SchemaCodecs: registrations.SchemaCodecs(),
		Decoders:     registrations.Decoders(),

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
		Interpretations:  registrations.Interpretations(),
		LocatorFactories: registrations.LocatorFactories(),
		Scope: composition.ScopeBinding{
			BuiltinRoot: topology.BuiltinRootID(),
		},
	})
	if err != nil {
		return nil, errors.Join(err, store.Close())
	}

	return &Handle{
		Store: store,
		LLM:   llm,
	}, nil
}
