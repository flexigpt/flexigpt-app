package aggregate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	toolnewRuntime "github.com/flexigpt/flexigpt-app/internal/toolnew/runtime"
	toolnewConsumerAPI "github.com/flexigpt/flexigpt-app/internal/toolnew/store/consumerapi"
	toolnewDomain "github.com/flexigpt/flexigpt-app/internal/toolnew/store/domain"
)

type Service struct {
	tools   *toolnewConsumerAPI.API
	runtime *toolnewRuntime.Service
}

type InvokeRequest struct {
	Target    resolve.MappedTarget `json:"target"`
	Args      json.RawMessage      `json:"args"`
	TimeoutMS int                  `json:"timeoutMS,omitempty"`
}

func New(
	tools *toolnewConsumerAPI.API,
	runtimeService *toolnewRuntime.Service,
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
) (toolnewDomain.ResolvedTool, error) {
	if s == nil || s.tools == nil {
		return toolnewDomain.ResolvedTool{}, basespec.ErrClosed
	}

	value, err := DecodeTarget(target)
	if err != nil {
		return toolnewDomain.ResolvedTool{}, err
	}

	resolved, err := s.tools.ResolveEnabledTool(
		ctx,
		value.ToolArtifact,
	)
	if err != nil {
		return toolnewDomain.ResolvedTool{}, err
	}

	document := resolved.Tool.Document
	if resolved.Tool.Definition.Digest != value.DefinitionDigest ||
		resolved.Tool.Artifact.LogicalName != value.Name ||
		string(document.Version) != value.Version ||
		string(document.Implementation.Kind) != value.Implementation {
		return toolnewDomain.ResolvedTool{}, fmt.Errorf(
			"%w: mapped Tool target no longer matches its Tool Artifact",
			basespec.ErrReferenceUnresolved,
		)
	}
	return resolved, nil
}

func (s *Service) Invoke(
	ctx context.Context,
	request InvokeRequest,
) (*toolnewRuntime.InvokeResponse, error) {
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

	return s.runtime.Invoke(ctx, toolnewRuntime.InvokeRequest{
		Function:  resolved.Tool.Document.Implementation.Function,
		Args:      request.Args,
		TimeoutMS: request.TimeoutMS,
	})
}
