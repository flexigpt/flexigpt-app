package agent

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type BaselineEnsurer interface {
	EnsureAgentBaselinePlugin(
		ctx context.Context,
		rootID rootModel.RootID,
	) (pluginAPI.PluginView, error)
}

type baselineEnsurer struct {
	api *Service
}

func NewBaselineEnsurer(
	api *Service,
) (BaselineEnsurer, error) {
	if api == nil || api.plugins == nil {
		return nil, fmt.Errorf(
			"%w: Agent baseline ensurer requires plugins",
			spec.ErrInvalid,
		)
	}
	return &baselineEnsurer{api: api}, nil
}

func (s *baselineEnsurer) EnsureAgentBaselinePlugin(
	ctx context.Context,
	rootID rootModel.RootID,
) (pluginAPI.PluginView, error) {
	if s == nil || s.api == nil {
		return pluginAPI.PluginView{}, spec.ErrClosed
	}
	return s.api.ensureAgentBaselinePlugin(ctx, rootID)
}
