package workspace

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/workspace/contextadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
	"github.com/flexigpt/flexigpt-app/internal/workspace/skilladapter"
)

type components struct {
	service   *engine.Service
	refresher *engine.Refresher
	query     *engine.QueryService
	policy    *engine.ArtifactPolicy

	contextAdapter *contextadapter.Adapter
	skillAdapter   *skilladapter.Adapter
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
	discoveryPolicyRevision, err := config.discoveryPolicyRevision()
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
		if !dependencies.HasDecoder(support.DecoderID) {
			return nil, fmt.Errorf(
				"%w: workspace decoder %q is not registered with artifact store",
				engine.ErrInvalidWorkspace,
				support.DecoderID,
			)
		}
		decoderIDs = append(decoderIDs, support.DecoderID)
	}

	service, err := engine.NewService(
		dependencies.Collections,
		dependencies.Sources,
		discoveryPolicyRevision,
	)
	if err != nil {
		return nil, err
	}
	planner, err := engine.NewPlanner(
		profiles,
		discoveryPolicyRevision,
		decoderIDs...,
	)
	if err != nil {
		return nil, err
	}
	loader, err := engine.NewDescriptorLoader(
		dependencies.SourceRuntime,
	)
	if err != nil {
		return nil, err
	}
	policy, err := engine.NewArtifactPolicy(supports...)
	if err != nil {
		return nil, err
	}
	refresher, err := engine.NewRefresher(
		service,
		loader,
		planner,
		dependencies.Refresh,
		policy,
	)
	if err != nil {
		return nil, err
	}
	query, err := engine.NewQueryService(
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
	skillAdapter, err := skilladapter.NewAdapter(
		query,
		runtimePolicy,
		dependencies.SourceRuntime,
	)
	if err != nil {
		return nil, err
	}
	return &components{
		service:        service,
		refresher:      refresher,
		query:          query,
		policy:         policy,
		contextAdapter: contextAdapter,
		skillAdapter:   skillAdapter,
	}, nil
}
