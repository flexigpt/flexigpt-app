package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
)

type Components struct {
	Service   *Service
	Refresher *Refresher
	Query     *QueryService
	Policy    *RecordPolicy
	Planner   *Planner
	Loader    *DefinitionLoader
	Context   *ContextProvider
	Skills    *SkillFacade
}

func NewComponents(
	artifacts *system.Components,
	config Config,
) (*Components, error) {
	if artifacts == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store components are nil",
			ErrInvalidWorkspace,
		)
	}
	if artifacts.DecoderRegistry == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store decoder registry is nil",
			ErrInvalidWorkspace,
		)
	}

	supports, err := config.normalizedSupports()
	if err != nil {
		return nil, err
	}
	decoderIDs := make([]artifactstore.DecoderID, 0, len(supports))
	for _, support := range supports {
		if _, exists := artifacts.DecoderRegistry.Get(support.DecoderID); !exists {
			return nil, fmt.Errorf(
				"%w: Workspace decoder %q was not registered with Artifact Store",
				ErrInvalidWorkspace,
				support.DecoderID,
			)
		}
		decoderIDs = append(decoderIDs, support.DecoderID)
	}

	service, err := NewService(
		artifacts.Catalog,
		artifacts.Sources,
	)
	if err != nil {
		return nil, err
	}
	planner, err := NewPlanner(decoderIDs...)
	if err != nil {
		return nil, err
	}
	loader, err := NewDefinitionLoader(
		artifacts.SourceRepository,
		artifacts.SourceRegistry,
	)
	if err != nil {
		return nil, err
	}
	policy, err := NewRecordPolicy(supports...)
	if err != nil {
		return nil, err
	}
	refresher, err := NewRefresher(
		service,
		loader,
		planner,
		artifacts.Refresh,
		policy,
	)
	if err != nil {
		return nil, err
	}
	query, err := NewQueryService(
		service,
		artifacts.Catalog,
		artifacts.Sources,
		artifacts.Records,
		artifacts.Definitions,
	)
	if err != nil {
		return nil, err
	}
	contextProvider, err := NewContextProvider(query)
	if err != nil {
		return nil, err
	}
	skillFacade, err := NewSkillFacade(query)
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
