package consumerapi

import (
	"context"
	"fmt"
	"sort"

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

func (a *API) GetAgent(
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

func (a *API) GetAgentView(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (AgentView, error) {
	value, err := a.GetAgent(ctx, ref)
	if err != nil {
		return AgentView{}, err
	}
	return a.agentView(ctx, value)
}

func (a *API) ListAgents(
	ctx context.Context,
	request ListAgentsRequest,
) ([]AgentView, error) {
	if a == nil || a.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Agent list context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	names := make(
		map[basespec.LogicalName]struct{},
		len(request.LogicalNames),
	)
	for index, name := range request.LogicalNames {
		if err := name.Validate(); err != nil {
			return nil, fmt.Errorf(
				"agent list logicalNames[%d]: %w",
				index,
				err,
			)
		}
		names[name] = struct{}{}
	}

	var records []artifact.Artifact
	if request.Collection != nil {
		if err := request.Collection.Validate(); err != nil {
			return nil, err
		}
		if request.Collection.RootID != request.RootID {
			return nil, fmt.Errorf(
				"%w: Agent Collection belongs to another Root",
				basespec.ErrInvalid,
			)
		}
		if request.IncludeBuiltin {
			return nil, fmt.Errorf(
				"%w: collection-filtered Agent lists cannot include another Root",
				basespec.ErrInvalid,
			)
		}

		values, err := a.listCollectionAgents(ctx, *request.Collection)
		if err != nil {
			return nil, err
		}
		records = values
	} else {
		rootIDs := []root.RootID{request.RootID}
		if request.IncludeBuiltin && request.RootID != agentBuiltinRootID() {
			rootIDs = append(rootIDs, agentBuiltinRootID())
		}

		for _, rootID := range rootIDs {
			values, err := a.artifacts.ListByRoot(ctx, rootID)
			if err != nil {
				return nil, err
			}
			records = append(records, values...)
		}
	}

	seen := make(map[artifact.ArtifactRef]struct{}, len(records))
	output := make([]AgentView, 0, len(records))
	for _, record := range records {
		if !agentDomain.IsAgentKind(record.Kind) {
			continue
		}
		if len(names) != 0 {
			if _, found := names[record.LogicalName]; !found {
				continue
			}
		}
		if request.Enabled != nil && record.Enabled != *request.Enabled {
			continue
		}
		if _, duplicate := seen[record.Ref()]; duplicate {
			continue
		}
		seen[record.Ref()] = struct{}{}

		view, err := a.agentView(ctx, record)
		if err != nil {
			return nil, err
		}
		output = append(output, view)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		if output[left].Artifact.RootID != output[right].Artifact.RootID {
			return output[left].Artifact.RootID <
				output[right].Artifact.RootID
		}
		return output[left].Artifact.ID < output[right].Artifact.ID
	})
	return output, nil
}

func (a *API) SetAgentEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (artifact.Artifact, error) {
	if expectedRevision == 0 {
		return artifact.Artifact{}, fmt.Errorf(
			"%w: expected Agent Artifact revision is required",
			basespec.ErrInvalid,
		)
	}
	if _, err := a.GetAgent(ctx, ref); err != nil {
		return artifact.Artifact{}, err
	}
	return a.artifacts.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
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
	agent, err := a.GetAgentView(ctx, *plan.RootArtifact)
	if err != nil {
		return AgentResolution{}, err
	}
	return AgentResolution{
		Agent:        agent,
		Capabilities: plan,
	}, nil
}

func (a *API) ResolveAgentCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (resolve.CapabilityPlan, error) {
	if a == nil || a.declarationResolver == nil {
		return resolve.CapabilityPlan{}, basespec.ErrClosed
	}
	return a.declarationResolver.ResolveAgentCapabilities(ctx, ref)
}

func (a *API) listCollectionAgents(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]artifact.Artifact, error) {
	if a == nil || a.collections == nil || a.declarationResolver == nil {
		return nil, basespec.ErrClosed
	}
	if _, err := a.collections.Read(ctx, ref); err != nil {
		return nil, err
	}

	plugin, err := a.declarationResolver.ResolvePlugin(ctx, ref)
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

	output := make([]artifact.Artifact, 0, len(refs))
	for target := range refs {
		record, err := a.GetAgent(ctx, target)
		if err != nil {
			return nil, err
		}
		output = append(output, record)
	}
	return output, nil
}

func (a *API) agentView(
	ctx context.Context,
	record artifact.Artifact,
) (AgentView, error) {
	if !agentDomain.IsAgentKind(record.Kind) {
		return AgentView{}, fmt.Errorf(
			"%w: Artifact %q is not an Agent",
			basespec.ErrInvalid,
			record.ID,
		)
	}

	view := AgentView{
		Artifact:    record.Clone(),
		Name:        record.LogicalName,
		DisplayName: record.DisplayName,
		BuiltIn:     record.RootID == agentBuiltinRootID(),
	}

	if record.State != artifact.StateAvailable {
		return view, nil
	}

	definitionValue, err := a.artifacts.GetDefinition(ctx, record.Ref())
	if err != nil {
		return AgentView{}, err
	}
	if definitionValue.Kind != agentDomain.AgentArtifactKind {
		return AgentView{}, fmt.Errorf(
			"%w: Agent Artifact Definition has another kind",
			basespec.ErrDigestMismatch,
		)
	}

	document, err := agentv1.DecodeAgentJSON(definitionValue.Body)
	if err != nil {
		return AgentView{}, err
	}
	if document.Name != string(record.LogicalName) {
		return AgentView{}, fmt.Errorf(
			"%w: Agent declaration name differs from Artifact identity",
			basespec.ErrDigestMismatch,
		)
	}

	view.Description = document.Description
	sourceValue, err := a.sources.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return AgentView{}, err
	}
	if sourceValue.Kind != source.SourceKindManagedDirectory ||
		sourceValue.StorageKey != agentDomain.AgentManagedSourceStorageKey ||
		record.Binding.SubresourceLocator != "" {
		return view, nil
	}
	if _, err := agentDomain.ManagedPackageAddressFromAgentLocator(
		record.Binding.Locator,
	); err == nil {
		view.Managed = true
	}
	return view, nil
}
