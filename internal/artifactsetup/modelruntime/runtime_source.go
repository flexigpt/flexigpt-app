package modelruntime

import (
	"context"
	"errors"
	"fmt"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inferencewrapper/spec"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
			spec.ErrInvalid,
		)
	}
	if a.store != nil {
		return fmt.Errorf(
			"%w: Model runtime store is already bound",
			spec.ErrConflict,
		)
	}
	a.store = store
	return nil
}

func (a *RuntimeAdapter) ResolveRuntimeModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (RuntimeConfiguration, error) {
	return a.ResolveRuntimeConfiguration(
		ctx,
		inferencewrapperSpec.RuntimeModelRequest{Model: ref},
	)
}

func (a *RuntimeAdapter) ResolveRuntimeModelWithRequestPatch(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	requestPatch *inferencewrapperSpec.RuntimeRequestPatch,
) (RuntimeConfiguration, error) {
	return a.ResolveRuntimeConfiguration(
		ctx,
		inferencewrapperSpec.RuntimeModelRequest{
			Model:        ref,
			RequestPatch: requestPatch,
		},
	)
}

// ResolveRuntimeConfiguration implements the inference consumer capability.
// Completion supplies the typed request and owns only execution/transport.
func (a *RuntimeAdapter) ResolveRuntimeConfiguration(
	ctx context.Context,
	request inferencewrapperSpec.RuntimeModelRequest,
) (RuntimeConfiguration, error) {
	prepared, err := prepareRuntimeRequestPatch(request.RequestPatch)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	resolved, err := a.store.Models.Capabilities.Resolve(ctx, request.Model)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	return a.resolveRuntime(ctx, resolved, prepared)
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
			spec.ErrUnsupported,
			target.Type,
		)
	}
	if target.Form != composition.TargetFormArtifact || target.Artifact == nil {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: direct Model capability targets require an application runtime adapter",
			spec.ErrUnsupported,
		)
	}

	resolved, err := a.store.Models.Capabilities.Resolve(ctx, *target.Artifact)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	if resolved.Model.Artifact.Ref() != *target.Artifact ||
		resolved.Model.Artifact.LogicalName != target.Name {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: Model capability target no longer matches current source-backed configuration",
			spec.ErrReferenceUnresolved,
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
	resolved, err := a.store.Providers.Capabilities.Resolve(ctx, ref)
	if err != nil {
		return inferenceSpec.ProviderParam{}, err
	}
	return a.ResolveProviderRuntime(ctx, resolved)
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
			spec.ErrInvalid,
		)
	}

	candidates := make([]modelAPI.ProviderListItem, 0, len(providers))
	seen := make(map[artifactModel.ArtifactRef]struct{}, len(providers))
	names := make(map[spec.LogicalName]int)
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
					spec.ErrIdentityConflict,
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
