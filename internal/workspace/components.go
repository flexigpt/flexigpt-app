package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/skill/provider"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

type components struct {
	service   *engine.Service
	refresher *engine.Refresher
	query     *engine.QueryService

	contextAdapter *contextadapter.Adapter
	skillAdapter   *skilladapter.Adapter
	skillProvider  provider.Provider
}

func newComponents(
	artifacts *system.Components,
	config Config,
) (*components, error) {
	if artifacts == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store components are nil",
			engine.ErrInvalidWorkspace,
		)
	}
	if artifacts.SourceRuntime == nil {
		return nil, fmt.Errorf(
			"%w: artifact store source runtime is nil",
			engine.ErrInvalidWorkspace,
		)
	}

	supports, err := config.normalizedSupports()
	if err != nil {
		return nil, err
	}
	skillConventions, err := config.skillConventions()
	if err != nil {
		return nil, err
	}
	profiles := config.normalizedDiscoveryProfiles(skillConventions)
	decoderIDs := make([]artifactstore.DecoderID, 0, len(supports))
	for _, support := range supports {
		if !artifacts.HasDecoder(support.DecoderID) {
			return nil, fmt.Errorf(
				"%w: workspace decoder %q is not registered with artifact store",
				engine.ErrInvalidWorkspace,
				support.DecoderID,
			)
		}
		decoderIDs = append(decoderIDs, support.DecoderID)
	}

	service, err := engine.NewService(
		artifacts.Roots,
		artifacts.Sources,
	)
	if err != nil {
		return nil, err
	}
	planner, err := engine.NewPlanner(
		profiles,
		decoderIDs...,
	)
	if err != nil {
		return nil, err
	}
	loader, err := engine.NewDefinitionLoader(
		artifacts.SourceRuntime,
	)
	if err != nil {
		return nil, err
	}
	policy, err := engine.NewRecordPolicy(supports...)
	if err != nil {
		return nil, err
	}
	refresher, err := engine.NewRefresher(
		service,
		loader,
		planner,
		artifacts.Refresh,
		policy,
	)
	if err != nil {
		return nil, err
	}
	query, err := engine.NewQueryService(
		service,
		artifacts.Catalogs,
		artifacts.Records,
		artifacts.Definitions,
		supports...,
	)
	if err != nil {
		return nil, err
	}
	runtimePolicy := config.runtimePolicy()
	contextAdapter, err := contextadapter.NewAdapter(
		query,
		runtimePolicy,
		config.contextCompositionPolicy(),
	)
	if err != nil {
		return nil, err
	}
	skillAdapter, err := skilladapter.NewAdapter(
		query,
		runtimePolicy,
	)
	if err != nil {
		return nil, err
	}
	workspaceSkillProvider, err := provider.NewWorkspace(skillAdapter)
	if err != nil {
		return nil, err
	}
	return &components{
		service:        service,
		refresher:      refresher,
		query:          query,
		contextAdapter: contextAdapter,
		skillAdapter:   skillAdapter,
		skillProvider:  workspaceSkillProvider,
	}, nil
}
