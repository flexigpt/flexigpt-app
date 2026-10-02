package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
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
		return artifact.Artifact{}, model.ErrClosed
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
			model.ErrNotFound,
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
			model.ErrInvalid,
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
		return AgentResolution{}, model.ErrClosed
	}
	plan, err := a.declarationResolver.ResolveAgentCapabilities(ctx, ref)
	if err != nil {
		return AgentResolution{}, err
	}
	projectedPlan, err := projectAgentCapabilityPlan(plan)
	if err != nil {
		return AgentResolution{}, err
	}
	if plan.RootArtifact == nil {
		return AgentResolution{}, fmt.Errorf(
			"%w: Agent resolution has no Artifact root",
			model.ErrReferenceUnresolved,
		)
	}
	agent, err := a.GetAgent(ctx, *plan.RootArtifact)
	if err != nil {
		return AgentResolution{}, err
	}
	return AgentResolution{
		Agent:        agent,
		Capabilities: projectedPlan,
	}, nil
}

func (a *API) ResolveAgentCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (AgentCapabilityPlan, error) {
	if a == nil || a.declarationResolver == nil {
		return AgentCapabilityPlan{}, model.ErrClosed
	}
	value, err := a.declarationResolver.ResolveAgentCapabilities(ctx, ref)
	if err != nil {
		return AgentCapabilityPlan{}, err
	}

	return projectAgentCapabilityPlan(value)
}

func projectAgentCapabilityPlan(
	value resolve.CapabilityPlan,
) (AgentCapabilityPlan, error) {
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
		if occurrence.Mapped != nil {
			target := *occurrence.Mapped
			projected.Mapped = &target
		}

		if raw, found := occurrence.Overrides["autoExecute"]; found {
			value, err := projectAgentBoolean(raw, "autoExecute")
			if err != nil {
				return AgentCapabilityPlan{}, fmt.Errorf(
					"agent capability %q autoExecute override: %w",
					occurrence.Path,
					err,
				)
			}
			projected.AutoExecute = value
		}

		if raw, found := occurrence.Overrides["includeSystemPrompt"]; found {
			value, err := projectAgentBoolean(raw, "includeSystemPrompt")
			if err != nil {
				return AgentCapabilityPlan{}, fmt.Errorf(
					"agent capability %q includeSystemPrompt override: %w",
					occurrence.Path,
					err,
				)
			}
			projected.IncludeSystemPrompt = value
		}

		if raw, found := occurrence.Use["mode"]; found {
			var mode AgentSkillUseMode
			if err := json.Unmarshal(raw, &mode); err != nil {
				return AgentCapabilityPlan{}, fmt.Errorf(
					"agent capability %q Skill use mode: %w",
					occurrence.Path,
					err,
				)
			}

			switch mode {
			case AgentSkillUseModeAvailable,
				AgentSkillUseModeActive,
				AgentSkillUseModeInstructions:
				projected.SkillUseMode = mode
			default:
				return AgentCapabilityPlan{}, fmt.Errorf(
					"%w: Agent capability %q has unsupported Skill use mode %q",
					model.ErrInvalid,
					occurrence.Path,
					mode,
				)
			}
		}

		output.Occurrences = append(output.Occurrences, projected)
	}
	return output, nil
}

func projectAgentBoolean(
	raw json.RawMessage,
	name string,
) (*bool, error) {
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf(
			"%w: Agent relationship field %q must be boolean",
			model.ErrInvalid,
			name,
		)
	}
	return &value, nil
}

func (a *API) listCollectionAgentRefs(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]artifact.ArtifactRef, error) {
	if a == nil || a.collections == nil || a.declarationResolver == nil {
		return nil, model.ErrClosed
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
			model.ErrInvalid,
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
