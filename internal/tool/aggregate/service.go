package aggregate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/consumerapi"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	toolRuntime "github.com/flexigpt/flexigpt-app/internal/tool/runtime"
)

type Service struct {
	tools   *toolConsumerAPI.API
	runtime *toolRuntime.Service
}

type InvokeRequest struct {
	Target    composition.CapabilityTarget `json:"target"`
	Args      json.RawMessage              `json:"args"`
	TimeoutMS int                          `json:"timeoutMS,omitempty"`
}

func New(
	tools *toolConsumerAPI.API,
	runtimeService *toolRuntime.Service,
) (*Service, error) {
	if tools == nil || runtimeService == nil {
		return nil, fmt.Errorf(
			"%w: Tool Aggregate dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &Service{
		tools:   tools,
		runtime: runtimeService,
	}, nil
}

// ResolveToolTarget resolves a source-backed Tool capability target for the
// Go Tool runtime. Direct Tool capability targets remain owned by their
// application-supplied runtime adapter and are not coerced into Artifacts.
func (s *Service) ResolveToolTarget(
	ctx context.Context,
	target composition.CapabilityTarget,
) (toolConsumerAPI.ResolvedToolView, error) {
	if err := s.ready(ctx); err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}
	if err := target.Validate(); err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}
	if target.Type != declaration.TypeTool {
		return toolConsumerAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: capability target type is %q, expected tool",
			spec.ErrUnsupported,
			target.Type,
		)
	}
	if target.Form != composition.TargetFormArtifact ||
		target.Artifact == nil {
		return toolConsumerAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: direct Tool capability targets require an application runtime adapter",
			spec.ErrUnsupported,
		)
	}

	value, err := s.tools.ResolveEnabledTool(ctx, *target.Artifact)
	if err != nil {
		return toolConsumerAPI.ResolvedToolView{}, err
	}
	if value.Tool.Artifact.Ref() != *target.Artifact ||
		value.Tool.Name != target.Name {
		return toolConsumerAPI.ResolvedToolView{}, fmt.Errorf(
			"%w: Tool capability target no longer matches its Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	return value, nil
}

func (s *Service) Invoke(
	ctx context.Context,
	request InvokeRequest,
) (*toolRuntime.InvokeResponse, error) {
	if err := s.ready(ctx); err != nil {
		return nil, err
	}
	resolved, err := s.ResolveToolTarget(ctx, request.Target)
	if err != nil {
		return nil, err
	}
	if resolved.Tool.Implementation.Kind != toolv1.ImplementationKindGo {
		return nil, fmt.Errorf(
			"%w: SDK Tools execute through provider inference",
			spec.ErrUnsupported,
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
		return spec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: Tool Aggregate context is nil",
			spec.ErrInvalid,
		)
	}
	return ctx.Err()
}
