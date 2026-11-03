package modelruntime

import (
	"context"
	"errors"
	"fmt"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	modelAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model"
)

// BindModelStore completes application assembly after Model Store has received
// this adapter's lookup capability. Call exactly once, before serving requests.
// Direct conversion of already-resolved values does not require this binding.
func (a *RuntimeAdapter) BindModelStore(store *modelAPI.Service) error {
	if store == nil {
		return fmt.Errorf(
			"%w: Model runtime store is required",
			storeSpec.ErrInvalid,
		)
	}
	if a == nil {
		return storeSpec.ErrClosed
	}
	if a.store != nil {
		return fmt.Errorf(
			"%w: Model runtime store is already bound",
			storeSpec.ErrConflict,
		)
	}
	a.store = store
	return nil
}

func (a *RuntimeAdapter) ResolveRuntimeModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (RuntimeConfiguration, error) {
	value, _, err := a.ResolveRuntimeModelWithRequestPatch(ctx, ref, nil)
	return value, err
}

func (a *RuntimeAdapter) ResolveRuntimeModelWithRequestPatch(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	requestPatch *RuntimeRequestPatch,
) (RuntimeConfiguration, *RuntimeModelUse, error) {
	prepared, err := prepareRuntimeRequestPatch(requestPatch)
	if err != nil {
		return RuntimeConfiguration{}, nil, err
	}
	store, err := a.modelStore()
	if err != nil {
		return RuntimeConfiguration{}, nil, err
	}
	resolved, err := store.ResolveModel(ctx, ref)
	if err != nil {
		return RuntimeConfiguration{}, nil, err
	}
	value, err := a.resolveRuntime(ctx, resolved, prepared)
	if err != nil {
		return RuntimeConfiguration{}, nil, err
	}
	return value, &RuntimeModelUse{
		RootID:     string(resolved.Model.Artifact.RootID),
		ArtifactID: string(resolved.Model.Artifact.ID),
		Revision:   resolved.Model.Artifact.Revision,
		Digest:     string(resolved.Model.Definition.Digest),
	}, nil
}

// ResolveRuntimeModelTarget handles source-backed Model capability targets.
// Direct targets remain the application direct-capability provider's concern.
func (a *RuntimeAdapter) ResolveRuntimeModelTarget(
	ctx context.Context,
	target composition.CapabilityTarget,
) (RuntimeConfiguration, error) {
	if err := target.Validate(); err != nil {
		return RuntimeConfiguration{}, err
	}
	if target.Type != declaration.TypeModel {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: capability target type is %q, expected model",
			storeSpec.ErrUnsupported,
			target.Type,
		)
	}
	if target.Form != composition.TargetFormArtifact || target.Artifact == nil {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: direct Model capability targets require an application runtime adapter",
			storeSpec.ErrUnsupported,
		)
	}

	store, err := a.modelStore()
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	resolved, err := store.ResolveModel(ctx, *target.Artifact)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	if resolved.Model.Artifact.Ref() != *target.Artifact ||
		resolved.Model.Artifact.LogicalName != target.Name {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: Model capability target no longer matches current source-backed configuration",
			storeSpec.ErrReferenceUnresolved,
		)
	}
	return a.resolveRuntime(ctx, resolved, preparedRuntimeRequestPatch{})
}

// ResolveProvider resolves the current persisted Provider configuration.
// Unlike Model completion resolution, it permits an unconfigured API key.
func (a *RuntimeAdapter) ResolveProvider(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (inferenceSpec.ProviderParam, error) {
	store, err := a.modelStore()
	if err != nil {
		return inferenceSpec.ProviderParam{}, err
	}
	resolved, err := store.ResolveProvider(ctx, ref)
	if err != nil {
		return inferenceSpec.ProviderParam{}, err
	}
	return a.ResolveProviderRuntime(ctx, resolved)
}

func (a *RuntimeAdapter) modelStore() (*modelAPI.Service, error) {
	if a == nil || a.store == nil {
		return nil, storeSpec.ErrClosed
	}
	return a.store, nil
}

// InitializeProviderRuntime registers available enabled Providers after
// hydration. Application setup supplies the selected Provider list and the
// inference registry's publication capability.
//
// Registration is keyed by logical name, so colliding names are rejected
// rather than selecting one Root's credentials arbitrarily.
func (a *RuntimeAdapter) InitializeProviderRuntime(
	ctx context.Context,
	providers []modelAPI.ProviderListItem,
	publish func(context.Context, inferenceSpec.ProviderParam) error,
) error {
	if publish == nil {
		return fmt.Errorf(
			"%w: Model Provider runtime publication capability is required",
			storeSpec.ErrInvalid,
		)
	}

	candidates := make([]modelAPI.ProviderListItem, 0, len(providers))
	seen := make(map[artifactModel.ArtifactRef]struct{}, len(providers))
	names := make(map[storeSpec.LogicalName]int)
	for _, provider := range providers {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !provider.Enabled || provider.State != artifactModel.StateAvailable {
			continue
		}
		if _, duplicate := seen[provider.Ref]; duplicate {
			continue
		}
		seen[provider.Ref] = struct{}{}
		names[provider.Name]++
		candidates = append(candidates, provider)
	}

	var result error
	for _, provider := range candidates {
		if err := ctx.Err(); err != nil {
			return errors.Join(result, err)
		}
		if names[provider.Name] > 1 {
			result = errors.Join(
				result,
				fmt.Errorf(
					"%w: runtime Provider %q (%s/%s) has a duplicate logical name",
					storeSpec.ErrIdentityConflict,
					provider.Name,
					provider.Ref.RootID,
					provider.Ref.ArtifactID,
				),
			)
			continue
		}

		value, err := a.ResolveProvider(ctx, provider.Ref)
		if err == nil {
			err = publish(ctx, value)
		}
		if err != nil {
			result = errors.Join(
				result,
				fmt.Errorf(
					"initialize runtime Provider %q (%s/%s): %w",
					provider.Name,
					provider.Ref.RootID,
					provider.Ref.ArtifactID,
					err,
				),
			)
		}
	}
	return errors.Join(result, ctx.Err())
}
