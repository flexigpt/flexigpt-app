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
	Policy    *engine.RecordPolicy
	Planner   *engine.Planner
	Loader    *engine.DefinitionLoader

	Context *contextadapter.ContextProvider
	Skills  *skilladapter.SkillFacade
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
	if artifacts.DecoderRegistry == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store decoder registry is nil",
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
		if _, exists := artifacts.DecoderRegistry.Get(support.DecoderID); !exists {
			return nil, fmt.Errorf(
				"%w: Workspace decoder %q was not registered with Artifact Store",
				engine.ErrInvalidWorkspace,
				support.DecoderID,
			)
		}
		decoderIDs = append(decoderIDs, support.DecoderID)
	}

	service, err := engine.NewService(
		artifacts.Catalog,
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
		artifacts.SourceRepository,
		artifacts.SourceRegistry,
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
		artifacts.Catalog,
		artifacts.Sources,
		artifacts.Records,
		artifacts.Definitions,
	)
	if err != nil {
		return nil, err
	}
	contextProvider, err := contextadapter.NewContextProvider(query)
	if err != nil {
		return nil, err
	}
	skillFacade, err := skilladapter.NewSkillFacade(query)
	if err != nil {
		return nil, err
	}
	return &Components{
		Service:   service,
		Refresher: refresher,
		Query:     query,
		Policy:    policy,
		Planner:   planner,
		Loader:    loader,
		Context:   contextProvider,
		Skills:    skillFacade,
	}, nil
}
