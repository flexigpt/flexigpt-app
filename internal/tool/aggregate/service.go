package aggregate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/tool/runtime"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/tool/store/consumerapi"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
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

func (s *Service) MapArtifactTarget(
	ctx context.Context,
	request resolve.ArtifactTargetRequest,
) (resolve.MappedTarget, bool, error) {
	if s == nil || s.tools == nil {
		return resolve.MappedTarget{}, false, basespec.ErrClosed
	}
	if request.Type != declaration.TypeTool {
		return resolve.MappedTarget{}, false, nil
	}

	value, err := s.tools.ResolveEnabledTool(
		ctx,
		request.Artifact.Ref(),
	)
	if err != nil {
		return resolve.MappedTarget{}, true, err
	}
	if request.Definition.Digest != value.Tool.Definition.Digest {
		return resolve.MappedTarget{}, true, fmt.Errorf(
			"%w: Tool Artifact definition changed during target mapping",
			basespec.ErrRefreshRequired,
		)
	}

	target, err := NewMappedTarget(value)
	if err != nil {
		return resolve.MappedTarget{}, true, err
	}
	return target, true, nil
}

func (s *Service) ResolveMappedTool(
	ctx context.Context,
	target resolve.MappedTarget,
) (toolDomain.ResolvedTool, error) {
	if s == nil || s.tools == nil {
		return toolDomain.ResolvedTool{}, basespec.ErrClosed
	}

	value, err := DecodeTarget(target)
	if err != nil {
		return toolDomain.ResolvedTool{}, err
	}

	resolved, err := s.tools.ResolveEnabledTool(
		ctx,
		value.ToolArtifact,
	)
	if err != nil {
		return toolDomain.ResolvedTool{}, err
	}

	document := resolved.Tool.Document
	if resolved.Tool.Definition.Digest != value.DefinitionDigest ||
		resolved.Tool.Artifact.LogicalName != value.Name ||
		string(document.Version) != value.Version ||
		string(document.Implementation.Kind) != value.Implementation {
		return toolDomain.ResolvedTool{}, fmt.Errorf(
			"%w: mapped Tool target no longer matches its Tool Artifact",
			basespec.ErrReferenceUnresolved,
		)
	}
	return resolved, nil
}

func (s *Service) Invoke(
	ctx context.Context,
	request InvokeRequest,
) (*toolRuntime.InvokeResponse, error) {
	if s == nil || s.runtime == nil {
		return nil, basespec.ErrClosed
	}

	resolved, err := s.ResolveMappedTool(ctx, request.Target)
	if err != nil {
		return nil, err
	}
	if resolved.Tool.Document.Implementation.Kind !=
		toolv1.ImplementationKindGo {
		return nil, fmt.Errorf(
			"%w: SDK Tools execute through provider inference",
			basespec.ErrUnsupported,
		)
	}

	return s.runtime.Invoke(ctx, toolRuntime.InvokeRequest{
		Function:  resolved.Tool.Document.Implementation.Function,
		Args:      request.Args,
		TimeoutMS: request.TimeoutMS,
	})
}
