package plugin

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type PluginCapabilityPlan struct {
	Plugin      PluginView                         `json:"plugin"`
	Occurrences []composition.CapabilityOccurrence `json:"occurrences"`
	Complete    bool                               `json:"complete"`
}

func (a *API) ResolveCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (PluginCapabilityPlan, error) {
	if a == nil || a.resolver == nil {
		return PluginCapabilityPlan{}, fmt.Errorf(
			"%w: Plugin resolver is unavailable",
			spec.ErrUnsupported,
		)
	}
	plan, err := a.resolver.ResolvePluginCapabilities(ctx, ref)
	if err != nil {
		return PluginCapabilityPlan{}, err
	}
	if plan.RootType != declaration.TypePlugin || plan.RootArtifact == nil {
		return PluginCapabilityPlan{}, fmt.Errorf(
			"%w: Plugin did not resolve to a source-backed Plugin",
			spec.ErrReferenceUnresolved,
		)
	}
	view, err := a.Read(ctx, *plan.RootArtifact)
	if err != nil {
		return PluginCapabilityPlan{}, err
	}
	return PluginCapabilityPlan{
		Plugin:      view,
		Occurrences: plan.Occurrences,
		Complete:    plan.Complete,
	}, nil
}
