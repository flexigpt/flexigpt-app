package consumerapi

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

func agentBuiltinRootID() root.RootID {
	return documentTopology.BuiltinRootID()
}

type agentSourceCacheKey struct {
	rootID   root.RootID
	sourceID source.SourceID
}

func (a *API) getAgentRecord(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (artifact.Artifact, error) {
	if a == nil || a.artifacts == nil {
		return artifact.Artifact{}, basespec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return artifact.Artifact{}, err
	}

	value, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifact.Artifact{}, err
	}
	if !agentDomain.IsAgentKind(value.Kind) {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is not an Agent",
			basespec.ErrNotFound,
			ref.ArtifactID,
		)
	}
	return value.Clone(), nil
}

func (a *API) SetAgentEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (AgentView, error) {
	if expectedRevision == 0 {
		return AgentView{}, fmt.Errorf(
			"%w: expected Agent Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	if _, err := a.getAgentRecord(ctx, ref); err != nil {
		return AgentView{}, err
	}
	updated, err := a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
	if err != nil {
		return AgentView{}, err
	}
	return a.agentView(ctx, updated)
}

func (a *API) ResolveAgent(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (AgentResolution, error) {
	if a == nil || a.declarationResolver == nil {
		return AgentResolution{}, basespec.ErrClosed
	}
	plan, err := a.declarationResolver.ResolveAgentCapabilities(ctx, ref)
	if err != nil {
		return AgentResolution{}, err
	}
	if plan.RootArtifact == nil {
		return AgentResolution{}, fmt.Errorf(
			"%w: Agent resolution has no Artifact root",
			basespec.ErrReferenceUnresolved,
		)
	}
	agent, err := a.GetAgent(ctx, *plan.RootArtifact)
	if err != nil {
		return AgentResolution{}, err
	}
	return AgentResolution{
		Agent:        agent,
		Capabilities: projectAgentCapabilityPlan(plan),
	}, nil
}

func (a *API) ResolveAgentCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (AgentCapabilityPlan, error) {
	if a == nil || a.declarationResolver == nil {
		return AgentCapabilityPlan{}, basespec.ErrClosed
	}
	value, err := a.declarationResolver.ResolveAgentCapabilities(ctx, ref)
	if err != nil {
		return AgentCapabilityPlan{}, err
	}
	return projectAgentCapabilityPlan(value), nil
}

func projectAgentCapabilityPlan(
	value resolve.CapabilityPlan,
) AgentCapabilityPlan {
	output := AgentCapabilityPlan{
		Occurrences: make(
			[]AgentCapabilityOccurrence,
			0,
			len(value.Occurrences),
		),
		Complete: value.Complete,
	}
	for _, occurrence := range value.Occurrences {
		projected := AgentCapabilityOccurrence{
			Path:     occurrence.Path,
			Type:     occurrence.Type,
			Name:     occurrence.Name,
			Status:   occurrence.Status,
			Required: occurrence.Required,
			Code:     occurrence.Code,
			Message:  occurrence.Message,
		}
		if occurrence.Artifact != nil {
			ref := *occurrence.Artifact
			projected.Artifact = &ref
		}
		output.Occurrences = append(output.Occurrences, projected)
	}
	return output
}

func (a *API) listCollectionAgentRefs(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]artifact.ArtifactRef, error) {
	if a == nil || a.collections == nil || a.declarationResolver == nil {
		return nil, basespec.ErrClosed
	}
	if _, err := a.collections.Read(ctx, ref); err != nil {
		return nil, err
	}

	plugin, err := a.declarationResolver.ResolvePluginMembers(ctx, ref)
	if err != nil {
		return nil, err
	}

	refs := make(map[artifact.ArtifactRef]struct{})
	for _, relationship := range plugin.MemberResults {
		if relationship.Declared.Header().Type != agentv1.AgentType {
			continue
		}
		if relationship.Resolved != nil {
			if target, found := relationship.Resolved.ArtifactRef(); found {
				refs[target] = struct{}{}
			}
		}
		if relationship.Selector == nil {
			continue
		}
		for _, match := range relationship.Selector.Matches {
			if match.Resolved == nil {
				continue
			}
			if target, found := match.Resolved.ArtifactRef(); found {
				refs[target] = struct{}{}
			}
		}
	}

	output := make([]artifact.ArtifactRef, 0, len(refs))
	for target := range refs {
		output = append(output, target)
	}
	return output, nil
}

func (a *API) agentView(
	ctx context.Context,
	record artifact.Artifact,
) (AgentView, error) {
	return a.agentViewWithSourceCache(ctx, record, nil)
}

func (a *API) agentViewWithSourceCache(
	ctx context.Context,
	record artifact.Artifact,
	sourceCache map[agentSourceCacheKey]source.Summary,
) (AgentView, error) {
	if !agentDomain.IsAgentKind(record.Kind) {
		return AgentView{}, fmt.Errorf(
			"%w: Artifact %q is not an Agent",
			basespec.ErrInvalid,
			record.ID,
		)
	}

	view := AgentView{
		Ref:         record.Ref(),
		Name:        record.LogicalName,
		DisplayName: record.DisplayName,
		State:       record.State,
		Enabled:     record.Enabled,
		Revision:    record.Revision,
		BuiltIn:     record.RootID == agentBuiltinRootID(),
	}
	if record.ResolvedDefinition != nil {
		view.DefinitionDigest = *record.ResolvedDefinition
	}

	if record.State != artifact.StateAvailable {
		return view, nil
	}

	definitionValue, err := a.artifacts.GetDefinition(ctx, record.Ref())
	if err != nil {
		return AgentView{}, err
	}
	view.Description = definitionValue.Description
	managed, err := a.agentManaged(ctx, record, sourceCache)
	if err != nil {
		return AgentView{}, err
	}
	view.Managed = managed
	return view, nil
}

func (a *API) agentManaged(
	ctx context.Context,
	record artifact.Artifact,
	sourceCache map[agentSourceCacheKey]source.Summary,
) (bool, error) {
	if record.RootID == agentBuiltinRootID() ||
		record.State != artifact.StateAvailable ||
		record.Binding.SubresourceLocator != "" {
		return false, nil
	}
	key := agentSourceCacheKey{
		rootID:   record.RootID,
		sourceID: record.Binding.SourceID,
	}
	sourceValue, found := sourceCache[key]
	if !found {
		sourceValue, err := a.sources.Get(
			ctx,
			record.RootID,
			record.Binding.SourceID,
		)
		if err != nil {
			return false, err
		}
		if sourceCache != nil {
			sourceCache[key] = sourceValue
		}
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory ||
		sourceValue.StorageKey != agentDomain.AgentManagedSourceStorageKey {
		return false, nil
	}
	if _, err := agentDomain.ManagedPackageAddressFromAgentLocator(
		record.Binding.Locator,
	); err == nil {
		return true, nil
	}
	return false, nil
}
