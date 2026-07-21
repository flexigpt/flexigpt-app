package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
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

type Config struct {
	Descriptors []Descriptor
	DecoderIDs  []artifactstore.DecoderID
}

func DefaultConfig() Config {
	return Config{
		Descriptors: []Descriptor{
			{Kind: ContextKind, SchemaID: ContextSchemaID},
			{Kind: SkillKind, SchemaID: SkillSchemaID},
		},
		DecoderIDs: []artifactstore.DecoderID{
			ContextDecoderID,
			SkillDecoderID,
		},
	}
}

func BuiltinDecoders() []discovery.Decoder {
	return []discovery.Decoder{
		NewDefinitionDecoder(),
		NewContextDecoder(),
		NewSkillDecoder(),
	}
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
	if _, exists := artifacts.DecoderRegistry.Get(DefinitionDecoderID); !exists {
		return nil, fmt.Errorf(
			"%w: Workspace definition decoder was not registered with Artifact Store",
			ErrInvalidWorkspace,
		)
	}
	for _, decoderID := range config.DecoderIDs {
		if _, exists := artifacts.DecoderRegistry.Get(decoderID); !exists {
			return nil, fmt.Errorf(
				"%w: Workspace decoder %q was not registered with Artifact Store",
				ErrInvalidWorkspace,
				decoderID,
			)
		}
	}

	service, err := NewService(
		artifacts.Catalog,
		artifacts.Sources,
	)
	if err != nil {
		return nil, err
	}
	planner, err := NewPlanner(config.DecoderIDs...)
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
	policy, err := NewRecordPolicy(config.Descriptors...)
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
