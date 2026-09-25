package aggregate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/tool/runtime"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/tool/store/consumerapi"
)

type Service struct {
	tools   *toolConsumerAPI.API
	runtime *toolRuntime.Service
}

type InvokeRequest struct {
	Target    resolve.MappedTarget `json:"target"`
	Args      json.RawMessage      `json:"args"`
	TimeoutMS int                  `json:"timeoutMS,omitempty"`
}

func New(
	tools *toolConsumerAPI.API,
	runtimeService *toolRuntime.Service,
) (*Service, error) {
	if tools == nil || runtimeService == nil {
		return nil, fmt.Errorf(
			"%w: Tool Aggregate dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		tools:   tools,
		runtime: runtimeService,
	}, nil
}

func (s *Service) MapToolTarget(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (resolve.MappedTarget, error) {
	if err := s.ready(ctx); err != nil {
		return resolve.MappedTarget{}, err
	}
	value, err := s.tools.ResolveEnabledTool(ctx, ref)
	if err != nil {
		return resolve.MappedTarget{}, err
	}
	return NewMappedTarget(value)
}

func (s *Service) MapArtifactTarget(
	ctx context.Context,
	request resolve.ArtifactTargetRequest,
) (resolve.MappedTarget, bool, error) {
	if err := s.ready(ctx); err != nil {
		return resolve.MappedTarget{}, false, err
	}
	if request.Type != declaration.TypeTool {
		return resolve.MappedTarget{}, false, nil
	}

	value, err := s.tools.ResolveEnabledTool(ctx, request.Artifact.Ref())
	if err != nil {
		return resolve.MappedTarget{}, true, err
	}
	if request.Definition.Digest != value.Tool.DefinitionDigest {
		return resolve.MappedTarget{}, true, fmt.Errorf(
			"%w: Tool definition changed during target mapping",
			basespec.ErrRefreshRequired,
		)
	}

	target, err := NewMappedTarget(value)
	return target, true, err
}

func (s *Service) ResolveMappedTool(
	ctx context.Context,
	target resolve.MappedTarget,
) (toolConsumerAPI.ResolvedToolView, error) {
	if err := s.ready(ctx); err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}
	value, err := DecodeTarget(target)
	if err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}

	resolved, err := s.tools.ResolveEnabledTool(ctx, value.ToolArtifact)
	if err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}
	tool := resolved.Tool
	if tool.DefinitionDigest != value.DefinitionDigest ||
		tool.Name != value.Name ||
		tool.Version != value.Version ||
		tool.Implementation.Kind != value.Implementation {
		return toolConsumerAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: mapped Tool target no longer matches its Artifact",
			basespec.ErrReferenceUnresolved,
		)
	}
	return resolved, nil
}

func (s *Service) Invoke(
	ctx context.Context,
	request InvokeRequest,
) (*toolRuntime.InvokeResponse, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	resolved, err := s.ResolveMappedTool(ctx, request.Target)
	if err != nil {
		return nil, err
	}
	if resolved.Tool.Implementation.Kind != toolv1.ImplementationKindGo {
		return nil, fmt.Errorf(
			"%w: SDK Tools execute through provider inference",
			basespec.ErrUnsupported,
		)
	}

	return s.runtime.Invoke(ctx, toolRuntime.InvokeRequest{
		Function:  resolved.Tool.Implementation.Function,
		Args:      request.Args,
		TimeoutMS: request.TimeoutMS,
	})
}

func (s *Service) ready(ctx context.Context) error {
	if s == nil || s.tools == nil || s.runtime == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Tool Aggregate context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}
