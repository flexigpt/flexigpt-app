package consumerapi

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
	modelOverlay "github.com/flexigpt/flexigpt-app/internal/model/store/overlay"
)

func (a *API) ResolveModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedModel, error) {
	if err := a.ready(ctx); err != nil {
		return ResolvedModel{}, err
	}

	model, err := a.loadModel(ctx, ref)
	if err != nil {
		return ResolvedModel{}, err
	}
	if !model.Artifact.Enabled {
		return ResolvedModel{}, fmt.Errorf(
			"%w: Model %q is disabled",
			basespec.ErrReferenceUnresolved,
			model.Artifact.LogicalName,
		)
	}

	provider, err := a.resolveProviderReference(
		ctx,
		model.Artifact.RootID,
		model.Document.Provider,
	)
	if err != nil {
		return ResolvedModel{}, err
	}
	if !provider.Artifact.Enabled {
		return ResolvedModel{}, fmt.Errorf(
			"%w: Model Provider %q is disabled",
			basespec.ErrReferenceUnresolved,
			provider.Artifact.LogicalName,
		)
	}

	adapter, found, err := a.adapters.LookupModelAdapter(
		ctx,
		provider.Document.Adapter,
	)
	if err != nil {
		return ResolvedModel{}, err
	}
	if !found {
		return ResolvedModel{}, fmt.Errorf(
			"%w: Model Provider adapter %q is not installed",
			basespec.ErrUnsupported,
			provider.Document.Adapter,
		)
	}
	if err := adapter.Validate(); err != nil {
		return ResolvedModel{}, err
	}
	if adapter.ID != provider.Document.Adapter {
		return ResolvedModel{}, fmt.Errorf(
			"%w: Model adapter registry returned %q for Provider adapter %q",
			basespec.ErrInvalid,
			adapter.ID,
			provider.Document.Adapter,
		)
	}

	providerOverlay, _, err := a.overlays.GetProviderOverlay(
		ctx,
		provider.Artifact.Ref(),
	)
	if err != nil {
		return ResolvedModel{}, err
	}
	modelOverlayValue, _, err := a.overlays.GetModelOverlay(
		ctx,
		model.Artifact.Ref(),
	)
	if err != nil {
		return ResolvedModel{}, err
	}

	fingerprint, err := resolvedFingerprint(
		model,
		provider,
		providerOverlay,
		modelOverlayValue,
		adapter,
	)
	if err != nil {
		return ResolvedModel{}, err
	}

	return ResolvedModel{
		Model:           model,
		Provider:        provider,
		ProviderOverlay: providerOverlay.Clone(),
		ModelOverlay:    modelOverlayValue.Clone(),
		Adapter:         adapter,
		Fingerprint:     fingerprint,
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
				basespec.ErrIdentityConflict,
				reference.Name,
				len(candidates),
				rootID,
			)
		}
	}

	return modelDomain.Provider{}, fmt.Errorf(
		"%w: Model Provider %q is unresolved",
		basespec.ErrReferenceUnresolved,
		reference.Name,
	)
}

func (a *API) availableIdentityCandidates(
	ctx context.Context,
	rootID root.RootID,
	kind artifact.ArtifactKind,
	name basespec.LogicalName,
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
			basespec.ErrReferenceUnresolved,
			provider.Artifact.LogicalName,
		)
	}

	type candidate struct {
		reference declaration.ArtifactNameReference
		source    DefaultModelSource
	}
	candidates := make([]candidate, 0, 3)

	if a.protection.IsProtectedRoot(provider.Artifact.RootID) {
		overlay, found, err := a.overlays.GetProviderOverlay(
			ctx,
			provider.Artifact.Ref(),
		)
		if err != nil {
			return DefaultModelResolution{}, err
		}
		if found && overlay.DefaultModel != nil {
			candidates = append(candidates, candidate{
				reference: overlay.DefaultModel.Clone(),
				source:    DefaultModelSourceProtectedOverlay,
			})
		}
	} else {
		value, found, err := readMutableProviderDefaultModel(
			provider.Artifact.Data,
		)
		if err != nil {
			return DefaultModelResolution{}, err
		}
		if found {
			candidates = append(candidates, candidate{
				reference: value,
				source:    DefaultModelSourceMutableArtifactData,
			})
		}
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
					basespec.ErrReferenceUnresolved,
					reference.Name,
				)
				continue
			}
			return resolved, nil
		default:
			return ResolvedModel{}, fmt.Errorf(
				"%w: default Model %q resolves to %d Artifacts in Root %q",
				basespec.ErrIdentityConflict,
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
		basespec.ErrReferenceUnresolved,
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
		basespec.ErrReferenceUnresolved,
		provider.Artifact.LogicalName,
	)
}

func bestEffortDefaultFailure(err error) bool {
	return errors.Is(err, basespec.ErrReferenceUnresolved) ||
		errors.Is(err, basespec.ErrArtifactNotFound) ||
		errors.Is(err, basespec.ErrDefinitionNotFound) ||
		errors.Is(err, basespec.ErrIdentityConflict) ||
		errors.Is(err, basespec.ErrUnsupported)
}

func resolvedFingerprint(
	model modelDomain.Model,
	provider modelDomain.Provider,
	providerOverlay modelOverlay.ProviderOverlay,
	modelOverlayValue modelOverlay.ModelOverlay,
	adapter AdapterDescriptor,
) (cryptoutil.Digest, error) {
	return cryptoutil.CanonicalDigest(struct {
		Model struct {
			Ref        artifact.ArtifactRef `json:"ref"`
			Revision   uint64               `json:"revision"`
			Definition cryptoutil.Digest    `json:"definition"`
		} `json:"model"`

		Provider struct {
			Ref        artifact.ArtifactRef `json:"ref"`
			Revision   uint64               `json:"revision"`
			Definition cryptoutil.Digest    `json:"definition"`
		} `json:"provider"`

		ProviderOverlay struct {
			Revision      uint64 `json:"revision"`
			CredentialRef string `json:"credentialRef,omitempty"`
		} `json:"providerOverlay"`

		ModelOverlay struct {
			Revision uint64 `json:"revision"`
		} `json:"modelOverlay"`

		Adapter AdapterDescriptor `json:"adapter"`
	}{
		Model: struct {
			Ref        artifact.ArtifactRef `json:"ref"`
			Revision   uint64               `json:"revision"`
			Definition cryptoutil.Digest    `json:"definition"`
		}{
			Ref:        model.Artifact.Ref(),
			Revision:   model.Artifact.Revision,
			Definition: model.Definition.Digest,
		},
		Provider: struct {
			Ref        artifact.ArtifactRef `json:"ref"`
			Revision   uint64               `json:"revision"`
			Definition cryptoutil.Digest    `json:"definition"`
		}{
			Ref:        provider.Artifact.Ref(),
			Revision:   provider.Artifact.Revision,
			Definition: provider.Definition.Digest,
		},
		ProviderOverlay: struct {
			Revision      uint64 `json:"revision"`
			CredentialRef string `json:"credentialRef,omitempty"`
		}{
			Revision:      providerOverlay.Revision,
			CredentialRef: providerOverlay.CredentialRef,
		},
		ModelOverlay: struct {
			Revision uint64 `json:"revision"`
		}{
			Revision: modelOverlayValue.Revision,
		},
		Adapter: adapter,
	})
}
