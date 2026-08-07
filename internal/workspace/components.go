package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/skill/workspaceadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/discovery"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type components struct {
	workspaceRootID basespec.RootID
	service         *artifactadapter.Service
	refresher       *discovery.Refresher
	query           *artifactadapter.QueryService
	policy          *artifactadapter.ArtifactPolicy

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
	if config.AutoAdoptionIDProvider == nil {
		return nil, fmt.Errorf(
			"%w: Workspace automatic-adoption Artifact ID provider is required at application composition",
			spec.ErrInvalidWorkspace,
		)
	}

	supports, err := config.normalizedSupports()
	if err != nil {
		return nil, err
	}
	discoveryPolicyRevision, err := config.discoveryPolicyRevision()
	if err != nil {
		return nil, err
	}
	skillConventions, err := config.skillConventions()
	if err != nil {
		return nil, err
	}
	profiles := config.normalizedDiscoveryProfiles(skillConventions)
	decoderIDs := make([]basespec.DecoderID, 0, len(supports))
	for _, support := range supports {
		if !dependencies.HasDecoder(support.DecoderID) {
			return nil, fmt.Errorf(
				"%w: workspace decoder %q is not registered with artifact store",
				spec.ErrInvalidWorkspace,
				support.DecoderID,
			)
		}
		decoderIDs = append(decoderIDs, support.DecoderID)
	}

	service, err := artifactadapter.NewService(
		dependencies.Collections,
		dependencies.Sources,
		discoveryPolicyRevision,
		config.WorkspaceRootID,
		dependencies.RootMutationPolicy,
	)
	if err != nil {
		return nil, err
	}
	planner, err := discovery.NewPlanner(
		profiles,
		discoveryPolicyRevision,
		decoderIDs...,
	)
	if err != nil {
		return nil, err
	}
	loader, err := discovery.NewDescriptorLoader(
		dependencies.SourceRuntime,
	)
	if err != nil {
		return nil, err
	}
	policy, err := artifactadapter.NewArtifactPolicy(
		config.AutoAdoptionIDProvider,
		supports...,
	)
	if err != nil {
		return nil, err
	}
	refresher, err := discovery.NewRefresher(
		service,
		loader,
		planner,
		dependencies.Refresh,
		policy,
	)
	if err != nil {
		return nil, err
	}
	query, err := artifactadapter.NewQueryService(
		service,
		dependencies.Catalogs,
		dependencies.Artifacts,
		dependencies.Definitions,
		dependencies.DecoderFingerprint,
		discoveryPolicyRevision,
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
		dependencies.SourceRuntime,
	)
	if err != nil {
		return nil, err
	}
	return &components{
		workspaceRootID: config.WorkspaceRootID,
		service:         service,
		refresher:       refresher,
		query:           query,
		policy:          policy,
		contextAdapter:  contextAdapter,
		skillAdapter:    skillAdapter,
	}, nil
}
