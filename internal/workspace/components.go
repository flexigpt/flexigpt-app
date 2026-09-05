package workspace

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/workspaceadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
)

type components struct {
	workspaceRootID basespec.RootID
	service         *artifactadapter.Service
	query           *artifactadapter.QueryService
	supportedKinds  map[basespec.ArtifactKind]struct{}

	contextAdapter *contextadapter.Adapter
	skillAdapter   *workspaceadapter.Adapter
}

func newComponents(
	dependencies Dependencies,
	config Config,
) (*components, error) {
	if err := dependencies.Validate(); err != nil {
		return nil, err
	}
	supports, err := config.normalizedSupports()
	if err != nil {
		return nil, err
	}

	service, err := artifactadapter.NewService(
		dependencies.Store,
		config.WorkspaceRootID,
	)
	if err != nil {
		return nil, err
	}
	query, err := artifactadapter.NewQueryService(
		service,
		dependencies.Store,
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
	skillAdapter, err := workspaceadapter.NewAdapter(
		query,
		runtimePolicy,
		dependencies.Store,
	)
	if err != nil {
		return nil, err
	}

	supportedKinds := make(
		map[basespec.ArtifactKind]struct{},
		len(supports),
	)
	for _, support := range supports {
		supportedKinds[support.Kind] = struct{}{}
	}

	return &components{
		workspaceRootID: config.WorkspaceRootID,
		service:         service,
		query:           query,
		supportedKinds:  supportedKinds,
		contextAdapter:  contextAdapter,
		skillAdapter:    skillAdapter,
	}, nil
}
