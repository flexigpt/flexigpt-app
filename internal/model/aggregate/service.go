package aggregate

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
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
			model.ErrInvalid,
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
	seen := make(map[artifact.ArtifactRef]struct{}, len(providers))
	names := make(map[model.LogicalName]int)
	for _, provider := range providers {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !provider.Enabled || provider.State != artifact.StateAvailable {
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
					model.ErrIdentityConflict,
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
	ref artifact.ArtifactRef,
) (RuntimeConfiguration, error) {
	return s.ResolveRuntimeConfiguration(ctx, RuntimeModelRequest{
		Model: ref,
	})
}

func (s *Service) ResolveRuntimeModelWithRequestPatch(
	ctx context.Context,
	ref artifact.ArtifactRef,
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

func (s *Service) MapModelTarget(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (resolve.MappedTarget, error) {
	if err := s.ready(ctx); err != nil {
		return resolve.MappedTarget{}, err
	}

	resolved, err := s.store.ResolveModel(ctx, ref)
	if err != nil {
		return resolve.MappedTarget{}, err
	}
	return NewMappedTarget(resolved)
}

func (s *Service) MapArtifactTarget(
	ctx context.Context,
	request resolve.ArtifactTargetRequest,
) (resolve.MappedTarget, bool, error) {
	if err := s.ready(ctx); err != nil {
		return resolve.MappedTarget{}, false, err
	}
	if request.Type != declaration.TypeModel {
		return resolve.MappedTarget{}, false, nil
	}

	resolved, err := s.store.ResolveModel(ctx, request.Artifact.Ref())
	if err != nil {
		return resolve.MappedTarget{}, true, err
	}
	if request.Definition.Digest != resolved.Model.Definition.Digest {
		return resolve.MappedTarget{}, true, fmt.Errorf(
			"%w: Model Definition changed during target mapping",
			model.ErrRefreshRequired,
		)
	}

	target, err := NewMappedTarget(resolved)
	return target, true, err
}

func (s *Service) ResolveMappedRuntimeModel(
	ctx context.Context,
	target resolve.MappedTarget,
) (RuntimeConfiguration, error) {
	if err := s.ready(ctx); err != nil {
		return RuntimeConfiguration{}, err
	}

	value, err := DecodeTarget(target)
	if err != nil {
		return RuntimeConfiguration{}, err
	}

	resolved, err := s.store.ResolveModel(ctx, value.ModelArtifact)
	if err != nil {
		return RuntimeConfiguration{}, err
	}
	if resolved.Model.Artifact.LogicalName != value.Name ||
		resolved.Model.Definition.Digest != value.ModelDefinitionDigest ||
		resolved.Provider.Artifact.Ref() != value.ProviderArtifact ||
		resolved.Provider.Definition.Digest != value.ProviderDefinitionDigest ||
		resolved.Fingerprint != value.ConfigurationFingerprint {
		return RuntimeConfiguration{}, fmt.Errorf(
			"%w: mapped Model target no longer matches current source-backed configuration",
			model.ErrReferenceUnresolved,
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
		return model.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Model Aggregate context is nil",
			model.ErrInvalid,
		)
	}
	return ctx.Err()
}
