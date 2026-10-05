package consumerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
)

func agentBuiltinRootID() rootModel.RootID {
	return topology.BuiltinRootID()
}

type agentSourceCacheKey struct {
	rootID   rootModel.RootID
	sourceID sourceModel.SourceID
}

func (a *API) getAgentRecord(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	if a == nil || a.artifacts == nil {
		return artifactModel.Artifact{}, spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return artifactModel.Artifact{}, err
	}

	value, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if !agentDomain.IsAgentKind(value.Kind) {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: Artifact %q is not an Agent",
			spec.ErrNotFound,
			ref.ArtifactID,
		)
	}
	return value.Clone(), nil
}

func (a *API) SetAgentEnabled(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (AgentView, error) {
	if expectedRevision == 0 {
		return AgentView{}, fmt.Errorf(
			"%w: expected Agent Artifact revision is required",
			spec.ErrInvalid,
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
	ref artifactModel.ArtifactRef,
) (AgentResolution, error) {
	if a == nil || a.declarationResolver == nil {
		return AgentResolution{}, spec.ErrClosed
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
			spec.ErrReferenceUnresolved,
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
	ref artifactModel.ArtifactRef,
) (AgentCapabilityPlan, error) {
	if a == nil || a.declarationResolver == nil {
		return AgentCapabilityPlan{}, spec.ErrClosed
	}
	value, err := a.declarationResolver.ResolveAgentCapabilities(ctx, ref)
	if err != nil {
		return AgentCapabilityPlan{}, err
	}

	return projectAgentCapabilityPlan(value)
}

func projectAgentCapabilityPlan(
	value composition.CapabilityPlan,
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
		if occurrence.Target != nil {
			target := occurrence.Target.Clone()
			projected.Target = &target
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
					spec.ErrInvalid,
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
			spec.ErrInvalid,
			name,
		)
	}
	return &value, nil
}

func (a *API) listPluginAgentRefs(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]artifactModel.ArtifactRef, error) {
	if a == nil || a.plugins == nil || a.declarationResolver == nil {
		return nil, spec.ErrClosed
	}
	if _, err := a.plugins.Read(ctx, ref); err != nil {
		return nil, err
	}

	plugin, err := a.declarationResolver.ResolvePluginMembers(ctx, ref)
	if err != nil {
		return nil, err
	}

	refs := make(map[artifactModel.ArtifactRef]struct{})
	for _, relationship := range plugin.Relationships {
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

	output := make([]artifactModel.ArtifactRef, 0, len(refs))
	for target := range refs {
		output = append(output, target)
	}
	return output, nil
}

func (a *API) agentView(
	ctx context.Context,
	record artifactModel.Artifact,
) (AgentView, error) {
	return a.agentViewWithSourceCache(ctx, record, nil)
}

func (a *API) agentViewWithSourceCache(
	ctx context.Context,
	record artifactModel.Artifact,
	sourceCache map[agentSourceCacheKey]sourceModel.Summary,
) (AgentView, error) {
	if !agentDomain.IsAgentKind(record.Kind) {
		return AgentView{}, fmt.Errorf(
			"%w: Artifact %q is not an Agent",
			spec.ErrInvalid,
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

	if record.State != artifactModel.StateAvailable {
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
	record artifactModel.Artifact,
	sourceCache map[agentSourceCacheKey]sourceModel.Summary,
) (bool, error) {
	if record.RootID == agentBuiltinRootID() ||
		record.State != artifactModel.StateAvailable ||
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
	if sourceValue.Kind != managedfs.Kind ||
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
