package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

type Components struct {
	Service   *engine.Service
	Refresher *engine.Refresher
	Query     *engine.QueryService

	ContextAdapter *contextadapter.Adapter
	SkillAdapter   *skilladapter.Adapter
}

func NewComponents(
	artifacts *system.Components,
	config Config,
) (*Components, error) {
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
	profiles := config.normalizedDiscoveryProfiles()
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
		artifacts.SourceRuntime,
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
		artifacts.SourceRuntime,
		artifacts.Records,
		artifacts.Definitions,
	)
	if err != nil {
		return nil, err
	}
	contextAdapter, err := contextadapter.NewAdapter(query)
	if err != nil {
		return nil, err
	}
	skillAdapter, err := skilladapter.NewAdapter(query)
	if err != nil {
		return nil, err
	}
	return &Components{
		Service:        service,
		Refresher:      refresher,
		Query:          query,
		ContextAdapter: contextAdapter,
		SkillAdapter:   skillAdapter,
	}, nil
}
