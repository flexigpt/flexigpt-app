package aggregate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/model/inferenceadapter"
	modelConsumerAPI "github.com/flexigpt/flexigpt-app/internal/model/store/consumerapi"
)

type Service struct {
	store   *modelConsumerAPI.ManagementStoreFacade
	runtime *inferenceadapter.RuntimeAdapter
}

func New(
	store *modelConsumerAPI.ManagementStoreFacade,
	runtimeAdapter *inferenceadapter.RuntimeAdapter,
) (*Service, error) {
	if store == nil || runtimeAdapter == nil {
		return nil, fmt.Errorf(
			"%w: Model Aggregate dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		store:   store,
		runtime: runtimeAdapter,
	}, nil
}

func (s *Service) ResolveRuntimeModel(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (inferenceadapter.RuntimeConfiguration, error) {
	return s.ResolveRuntimeModelWithRequestPatch(ctx, ref, nil)
}

func (s *Service) ResolveRuntimeModelWithRequestPatch(
	ctx context.Context,
	ref artifact.ArtifactRef,
	requestPatch json.RawMessage,
) (inferenceadapter.RuntimeConfiguration, error) {
	if err := s.ready(ctx); err != nil {
		return inferenceadapter.RuntimeConfiguration{}, err
	}

	resolved, err := s.store.ResolveModel(ctx, ref)
	if err != nil {
		return inferenceadapter.RuntimeConfiguration{}, err
	}
	return s.runtime.ResolveWithRequestPatch(ctx, resolved, requestPatch)
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
			basespec.ErrRefreshRequired,
		)
	}

	target, err := NewMappedTarget(resolved)
	return target, true, err
}

func (s *Service) ResolveMappedRuntimeModel(
	ctx context.Context,
	target resolve.MappedTarget,
) (inferenceadapter.RuntimeConfiguration, error) {
	if err := s.ready(ctx); err != nil {
		return inferenceadapter.RuntimeConfiguration{}, err
	}

	value, err := DecodeTarget(target)
	if err != nil {
		return inferenceadapter.RuntimeConfiguration{}, err
	}

	resolved, err := s.store.ResolveModel(ctx, value.ModelArtifact)
	if err != nil {
		return inferenceadapter.RuntimeConfiguration{}, err
	}
	if resolved.Model.Artifact.LogicalName != value.Name ||
		resolved.Model.Definition.Digest != value.ModelDefinitionDigest ||
		resolved.Provider.Artifact.Ref() != value.ProviderArtifact ||
		resolved.Provider.Definition.Digest != value.ProviderDefinitionDigest ||
		resolved.Fingerprint != value.ConfigurationFingerprint {
		return inferenceadapter.RuntimeConfiguration{}, fmt.Errorf(
			"%w: mapped Model target no longer matches current source-backed configuration",
			basespec.ErrReferenceUnresolved,
		)
	}

	return s.runtime.Resolve(ctx, resolved)
}

func (s *Service) ready(ctx context.Context) error {
	if s == nil || s.store == nil || s.runtime == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Model Aggregate context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}
