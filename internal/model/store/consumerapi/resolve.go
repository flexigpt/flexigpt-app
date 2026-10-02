package consumerapi

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	catalog "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	secret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

func (a *API) ResolveProvider(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedProvider, error) {
	if err := a.ready(ctx); err != nil {
		return ResolvedProvider{}, err
	}

	provider, err := a.loadProvider(ctx, ref)
	if err != nil {
		return ResolvedProvider{}, err
	}
	return a.resolveProvider(ctx, provider)
}

func (a *API) ResolveModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedModel, error) {
	if err := a.ready(ctx); err != nil {
		return ResolvedModel{}, err
	}

	m, err := a.loadModel(ctx, ref)
	if err != nil {
		return ResolvedModel{}, err
	}
	if !m.Artifact.Enabled {
		return ResolvedModel{}, fmt.Errorf(
			"%w: Model %q is disabled",
			spec.ErrReferenceUnresolved,
			m.Artifact.LogicalName,
		)
	}

	provider, err := a.resolveProviderReference(
		ctx,
		m.Artifact.RootID,
		m.Document.Provider,
	)
	if err != nil {
		return ResolvedModel{}, err
	}

	resolvedProvider, err := a.resolveProvider(ctx, provider)
	if err != nil {
		return ResolvedModel{}, err
	}
	modelOverlayValue, _, err := a.overlays.GetModelOverlay(
		ctx,
		m.Artifact.Ref(),
	)
	if err != nil {
		return ResolvedModel{}, err
	}

	fingerprint, err := resolvedFingerprint(
		m,
		resolvedProvider.Provider,
		resolvedProvider.ProviderOverlay,
		resolvedProvider.ProviderCredential,
		modelOverlayValue,
		resolvedProvider.Adapter,
	)
	if err != nil {
		return ResolvedModel{}, err
	}

	return ResolvedModel{
		Model:              m,
		Provider:           resolvedProvider.Provider,
		ProviderOverlay:    resolvedProvider.ProviderOverlay.Clone(),
		ProviderCredential: resolvedProvider.ProviderCredential,
		ModelOverlay:       modelOverlayValue.Clone(),
		Adapter:            resolvedProvider.Adapter,
		Fingerprint:        fingerprint,
	}, nil
}

func (a *API) resolveProvider(
	ctx context.Context,
	provider modelDomain.Provider,
) (ResolvedProvider, error) {
	if !provider.Artifact.Enabled {
		return ResolvedProvider{}, fmt.Errorf(
			"%w: Model Provider %q is disabled",
			spec.ErrReferenceUnresolved,
			provider.Artifact.LogicalName,
		)
	}

	adapter, found, err := a.adapters.LookupModelAdapter(
		ctx,
		provider.Document.Adapter,
	)
	if err != nil {
		return ResolvedProvider{}, err
	}
	if !found {
		return ResolvedProvider{}, fmt.Errorf(
			"%w: Model Provider adapter %q is not installed",
			spec.ErrUnsupported,
			provider.Document.Adapter,
		)
	}
	if err := adapter.Validate(); err != nil {
		return ResolvedProvider{}, err
	}
	if adapter.ID != provider.Document.Adapter {
		return ResolvedProvider{}, fmt.Errorf(
			"%w: Model adapter registry returned %q for Provider adapter %q",
			spec.ErrInvalid,
			adapter.ID,
			provider.Document.Adapter,
		)
	}

	providerOverlay, _, err := a.overlays.GetProviderOverlay(
		ctx,
		provider.Artifact.Ref(),
	)
	if err != nil {
		return ResolvedProvider{}, err
	}

	credential, found, err := a.overlays.GetProviderCredential(
		ctx,
		provider.Artifact.Ref(),
	)
	if err != nil {
		return ResolvedProvider{}, err
	}

	var activeCredential *secret.Binding
	if found && credential.Active() {
		value := credential.Clone()
		activeCredential = &value
	}

	return ResolvedProvider{
		Provider:           provider,
		ProviderOverlay:    providerOverlay.Clone(),
		ProviderCredential: activeCredential,
		Adapter:            adapter,
	}, nil
}

func (a *API) resolveProviderReference(
	ctx context.Context,
	modelRoot root.RootID,
	reference declaration.ArtifactNameReference,
) (modelDomain.Provider, error) {
	roots, err := modelDomain.ArtifactNameReferenceLookupRoots(
		reference,
		modelRoot,
		a.builtinRoot,
	)
	if err != nil {
		return modelDomain.Provider{}, err
	}

	for _, rootID := range roots {
		candidates, err := a.availableIdentityCandidates(
			ctx,
			rootID,
			modelDomain.ModelProviderArtifactKind,
			reference.Name,
		)
		if err != nil {
			return modelDomain.Provider{}, err
		}

		switch len(candidates) {
		case 0:
			continue
		case 1:
			return a.loadProvider(ctx, candidates[0])
		default:
			return modelDomain.Provider{}, fmt.Errorf(
				"%w: Model Provider %q resolves to %d Artifacts in Root %q",
				spec.ErrIdentityConflict,
				reference.Name,
				len(candidates),
				rootID,
			)
		}
	}

	return modelDomain.Provider{}, fmt.Errorf(
		"%w: Model Provider %q is unresolved",
		spec.ErrReferenceUnresolved,
		reference.Name,
	)
}

func (a *API) availableIdentityCandidates(
	ctx context.Context,
	rootID root.RootID,
	kind artifact.ArtifactKind,
	name spec.LogicalName,
) ([]artifact.ArtifactRef, error) {
	entries, err := a.artifacts.FindByIdentity(
		ctx,
		rootID,
		kind,
		name,
		catalog.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	seen := make(map[artifact.ArtifactRef]struct{}, len(entries))
	output := make([]artifact.ArtifactRef, 0, len(entries))
	for _, entry := range entries {
		if entry.State != artifact.StateAvailable {
			continue
		}
		if _, duplicate := seen[entry.Ref()]; duplicate {
			continue
		}
		seen[entry.Ref()] = struct{}{}
		output = append(output, entry.Ref())
	}

	sort.Slice(output, func(left, right int) bool {
		return output[left].ArtifactID < output[right].ArtifactID
	})
	return output, nil
}

// ResolveProviderDefaultModel resolves the Provider's best-effort default
// Model. A stale or unavailable explicit default does not invalidate the
// Provider. The deterministic first enabled linked Model fallback is used when
// possible.
func (a *API) ResolveProviderDefaultModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (DefaultModelResolution, error) {
	if err := a.ready(ctx); err != nil {
		return DefaultModelResolution{}, err
	}

	provider, err := a.loadProvider(ctx, ref)
	if err != nil {
		return DefaultModelResolution{}, err
	}
	if !provider.Artifact.Enabled {
		return DefaultModelResolution{}, fmt.Errorf(
			"%w: Model Provider %q is disabled",
			spec.ErrReferenceUnresolved,
			provider.Artifact.LogicalName,
		)
	}

	type candidate struct {
		reference declaration.ArtifactNameReference
		source    DefaultModelSource
	}
	candidates := make([]candidate, 0, 3)

	overlay, found, err := a.overlays.GetProviderOverlay(
		ctx,
		provider.Artifact.Ref(),
	)
	if err != nil {
		return DefaultModelResolution{}, err
	}
	if found && overlay.DefaultModel != nil {
		source := DefaultModelSourceMutableArtifactData
		if a.protection.IsProtectedRoot(provider.Artifact.RootID) {
			source = DefaultModelSourceProtectedOverlay
		}
		candidates = append(candidates, candidate{
			reference: overlay.DefaultModel.Clone(),
			source:    source,
		})
	}

	if provider.Document.DefaultModel != nil {
		candidates = append(candidates, candidate{
			reference: provider.Document.DefaultModel.Clone(),
			source:    DefaultModelSourceProviderDeclaration,
		})
	}

	for _, candidate := range candidates {
		resolved, err := a.resolveNamedModelForProvider(
			ctx,
			provider,
			candidate.reference,
		)
		if err == nil {
			return DefaultModelResolution{
				Resolved: resolved,
				Source:   candidate.source,
			}, nil
		}
		if !bestEffortDefaultFailure(err) {
			return DefaultModelResolution{}, err
		}
	}

	resolved, err := a.firstEnabledModelForProvider(ctx, provider)
	if err != nil {
		return DefaultModelResolution{}, err
	}
	return DefaultModelResolution{
		Resolved: resolved,
		Source:   DefaultModelSourceFirstEnabledModel,
	}, nil
}

func (a *API) resolveNamedModelForProvider(
	ctx context.Context,
	provider modelDomain.Provider,
	reference declaration.ArtifactNameReference,
) (ResolvedModel, error) {
	roots, err := modelDomain.ArtifactNameReferenceLookupRoots(
		reference,
		provider.Artifact.RootID,
		a.builtinRoot,
	)
	if err != nil {
		return ResolvedModel{}, err
	}

	var lastErr error
	for _, rootID := range roots {
		candidates, err := a.availableIdentityCandidates(
			ctx,
			rootID,
			modelDomain.ModelArtifactKind,
			reference.Name,
		)
		if err != nil {
			return ResolvedModel{}, err
		}

		switch len(candidates) {
		case 0:
			continue
		case 1:
			resolved, err := a.ResolveModel(ctx, candidates[0])
			if err != nil {
				lastErr = err
				continue
			}
			if resolved.Provider.Artifact.Ref() != provider.Artifact.Ref() {
				lastErr = fmt.Errorf(
					"%w: Model %q resolves to another Provider",
					spec.ErrReferenceUnresolved,
					reference.Name,
				)
				continue
			}
			return resolved, nil
		default:
			return ResolvedModel{}, fmt.Errorf(
				"%w: default Model %q resolves to %d Artifacts in Root %q",
				spec.ErrIdentityConflict,
				reference.Name,
				len(candidates),
				rootID,
			)
		}
	}

	if lastErr != nil {
		return ResolvedModel{}, lastErr
	}
	return ResolvedModel{}, fmt.Errorf(
		"%w: default Model %q is unresolved",
		spec.ErrReferenceUnresolved,
		reference.Name,
	)
}

func (a *API) firstEnabledModelForProvider(
	ctx context.Context,
	provider modelDomain.Provider,
) (ResolvedModel, error) {
	items, err := a.ListModels(ctx, ListModelsRequest{
		RootID: provider.Artifact.RootID,
	})
	if err != nil {
		return ResolvedModel{}, err
	}

	for _, item := range items {
		if item.State != artifact.StateAvailable || !item.Enabled {
			continue
		}
		resolved, err := a.ResolveModel(ctx, item.Ref)
		if err != nil {
			if bestEffortDefaultFailure(err) {
				continue
			}
			return ResolvedModel{}, err
		}
		if resolved.Provider.Artifact.Ref() != provider.Artifact.Ref() {
			continue
		}
		return resolved, nil
	}

	return ResolvedModel{}, fmt.Errorf(
		"%w: Model Provider %q has no enabled resolved Models",
		spec.ErrReferenceUnresolved,
		provider.Artifact.LogicalName,
	)
}

func bestEffortDefaultFailure(err error) bool {
	return errors.Is(err, spec.ErrReferenceUnresolved) ||
		errors.Is(err, spec.ErrArtifactNotFound) ||
		errors.Is(err, spec.ErrDefinitionNotFound) ||
		errors.Is(err, spec.ErrIdentityConflict) ||
		errors.Is(err, spec.ErrUnsupported)
}

type resolveModel struct {
	Ref        artifact.ArtifactRef `json:"ref"`
	Revision   uint64               `json:"revision"`
	Definition cryptoutil.Digest    `json:"definition"`
}

type resolveProvider struct {
	Ref        artifact.ArtifactRef `json:"ref"`
	Revision   uint64               `json:"revision"`
	Definition cryptoutil.Digest    `json:"definition"`
}

type resolveProviderOverlay struct {
	Revision uint64 `json:"revision"`
}

type resolveProviderCredential struct {
	Ref      string `json:"ref,omitempty"`
	Revision uint64 `json:"revision,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
}

type resolveModelOverlay struct {
	Revision uint64 `json:"revision"`
}

func resolvedFingerprint(
	m modelDomain.Model,
	provider modelDomain.Provider,
	providerOverlay modelOverlay.ProviderOverlay,
	providerCredential *secret.Binding,
	modelOverlayValue modelOverlay.ModelOverlay,
	adapter AdapterDescriptor,
) (cryptoutil.Digest, error) {
	return cryptoutil.CanonicalDigest(struct {
		Model              resolveModel              `json:"model"`
		Provider           resolveProvider           `json:"provider"`
		ProviderOverlay    resolveProviderOverlay    `json:"providerOverlay"`
		ProviderCredential resolveProviderCredential `json:"providerCredential"`
		ModelOverlay       resolveModelOverlay       `json:"modelOverlay"`
		Adapter            AdapterDescriptor         `json:"adapter"`
	}{
		Model: resolveModel{
			Ref:        m.Artifact.Ref(),
			Revision:   m.Artifact.Revision,
			Definition: m.Definition.Digest,
		},

		Provider: resolveProvider{
			Ref:        provider.Artifact.Ref(),
			Revision:   provider.Artifact.Revision,
			Definition: provider.Definition.Digest,
		},
		ProviderOverlay: resolveProviderOverlay{
			Revision: providerOverlay.Revision,
		},
		ProviderCredential: func() resolveProviderCredential {
			if providerCredential == nil ||
				!providerCredential.Active() {
				return resolveProviderCredential{}
			}
			return resolveProviderCredential{
				Ref:      string(*providerCredential.Ref),
				Revision: providerCredential.Revision,
				SHA256:   providerCredential.SHA256,
			}
		}(),
		ModelOverlay: resolveModelOverlay{
			Revision: modelOverlayValue.Revision,
		},
		Adapter: adapter,
	})
}
