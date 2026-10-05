package aggregate

import (
	"context"
	"errors"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/consumerapi"
)

type Service struct {
	store       *modelConsumerAPI.ManagementStoreFacade
	runtime     RuntimeResolver
	preferences DefaultProviderPreferences
	providers   ProviderRuntimePublisher
}

func New(
	store *modelConsumerAPI.ManagementStoreFacade,
	runtimeResolver RuntimeResolver,
	preferences DefaultProviderPreferences,
) (*Service, error) {
	if store == nil || runtimeResolver == nil || preferences == nil {
		return nil, fmt.Errorf(
			"%w: Model Aggregate dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &Service{
		store:       store,
		runtime:     runtimeResolver,
		preferences: preferences,
	}, nil
}

// InitializeProviderRuntime registers currently available, enabled Providers.
// It resolves each Provider's current settings and credential through Model
// Store and the runtime adapter, without resolving Models.
//
// Call once after catalog hydration and publisher binding, before serving
// requests. Independent Provider failures are joined so others can initialize.
func (s *Service) InitializeProviderRuntime(
	ctx context.Context,
	providers []modelConsumerAPI.ProviderListItem,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if _, err := s.providerRuntimePublisher(); err != nil {
		return err
	}

	// Inference registration is keyed by logical name, not ArtifactRef. Do not
	// silently choose one Root's settings or credentials for a colliding name.
	candidates := make([]modelConsumerAPI.ProviderListItem, 0, len(providers))
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
		if err := s.publishProviderRuntime(ctx, provider.Ref); err != nil {
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

func (s *Service) ResolveRuntimeModel(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (RuntimeConfiguration, error) {
	return s.ResolveRuntimeConfiguration(ctx, RuntimeModelRequest{
		Model: ref,
	})
}

func (s *Service) ResolveRuntimeModelWithRequestPatch(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	requestPatch *RuntimeRequestPatch,
) (RuntimeConfiguration, error) {
	return s.ResolveRuntimeConfiguration(ctx, RuntimeModelRequest{
		Model:        ref,
		RequestPatch: requestPatch,
	})
}

// ResolveRuntimeConfiguration owns completion-time Model runtime resolution.
// The Wails wrapper only supplies a typed request value and handles transport
// concerns such as callbacks, cancellation, and event delivery.
func (s *Service) ResolveRuntimeConfiguration(
	ctx context.Context,
	request RuntimeModelRequest,
) (RuntimeConfiguration, error) {
	if err := s.ready(ctx); err != nil {
		return RuntimeConfiguration{}, err
	}
	if err := request.Validate(); err != nil {
		return RuntimeConfiguration{}, err
	}
	resolved, err := s.store.ResolveModel(ctx, request.Model)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	preparedRequestPatch, err := request.RequestPatch.Prepare()
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	return s.runtime.ResolveRuntime(ctx, resolved, preparedRequestPatch)
}

// ResolveRuntimeModelTarget resolves a source-backed Model capability target
// into runtime configuration. Direct Model capability targets are deliberately
// not decoded here: their runtime preparation remains owned by the
// application-supplied direct capability provider.
func (s *Service) ResolveRuntimeModelTarget(
	ctx context.Context,
	target composition.CapabilityTarget,
) (RuntimeConfiguration, error) {
	if err := s.ready(ctx); err != nil {
		return RuntimeConfiguration{}, err
	}
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
	if target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: direct Model capability targets require an application runtime adapter",
			spec.ErrUnsupported,
		)
	}
	resolved, err := s.store.ResolveModel(ctx, *target.Artifact)
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
	return s.runtime.ResolveRuntime(
		ctx,
		resolved,
		PreparedRuntimeRequestPatch{},
	)
}

func (s *Service) ready(ctx context.Context) error {
	if s == nil || s.store == nil || s.runtime == nil || s.preferences == nil {
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Model Aggregate context is nil",
			spec.ErrInvalid,
		)
	}
	return ctx.Err()
}
