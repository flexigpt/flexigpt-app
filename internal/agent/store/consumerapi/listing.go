package consumerapi

import (
	"context"
	"fmt"
	"sort"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func (a *API) GetAgent(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (AgentView, error) {
	record, err := a.getAgentRecord(ctx, ref)
	if err != nil {
		return AgentView{}, err
	}
	return a.agentView(ctx, record)
}

func (a *API) ListAgents(
	ctx context.Context,
	request ListAgentsRequest,
) ([]AgentListItem, error) {
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

	names := make(map[basespec.LogicalName]struct{}, len(request.LogicalNames))
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

	allowedRefs := map[artifact.ArtifactRef]struct{}(nil)
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

		refs, err := a.listCollectionAgentRefs(ctx, *request.Collection)
		if err != nil {
			return nil, err
		}
		allowedRefs = make(map[artifact.ArtifactRef]struct{}, len(refs))
		for _, ref := range refs {
			allowedRefs[ref] = struct{}{}
		}
	}

	rootIDs := []root.RootID{request.RootID}
	if request.IncludeBuiltin &&
		request.RootID != documentTopology.BuiltinRootID() {
		rootIDs = append(rootIDs, documentTopology.BuiltinRootID())
	}

	entries := make([]catalog.Entry, 0)
	for _, rootID := range rootIDs {
		values, err := a.artifacts.ListByRoot(
			ctx,
			rootID,
			catalog.ListOptions{
				Kind:         agentDomain.AgentArtifactKind,
				Enabled:      request.Enabled,
				LogicalNames: request.LogicalNames,
			},
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, values...)
	}

	seen := make(map[artifact.ArtifactRef]struct{}, len(entries))
	output := make([]AgentListItem, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind != agentDomain.AgentArtifactKind {
			continue
		}
		if len(names) != 0 {
			if _, found := names[entry.LogicalName]; !found {
				continue
			}
		}
		if request.Enabled != nil && entry.Enabled != *request.Enabled {
			continue
		}
		if allowedRefs != nil {
			if _, found := allowedRefs[entry.Ref()]; !found {
				continue
			}
		}
		if _, duplicate := seen[entry.Ref()]; duplicate {
			continue
		}
		seen[entry.Ref()] = struct{}{}

		item := AgentListItem{
			AgentView: agentViewFromCatalog(entry),
		}
		output = append(output, item)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		if output[left].Ref.RootID != output[right].Ref.RootID {
			return output[left].Ref.RootID < output[right].Ref.RootID
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}

func agentViewFromCatalog(
	entry catalog.Entry,
) AgentView {
	digest := cryptoutil.Digest("")
	description := ""
	if entry.Definition != nil {
		digest = entry.Definition.Digest
		description = entry.Definition.Description
	}

	managed := false
	if entry.State == artifact.StateAvailable &&
		entry.Source.Kind == source.SourceKindManagedDirectory &&
		entry.Source.StorageKey == agentDomain.AgentManagedSourceStorageKey &&
		entry.Binding.SubresourceLocator == "" {
		_, err := agentDomain.ManagedPackageAddressFromAgentLocator(
			entry.Binding.Locator,
		)
		managed = err == nil
	}

	return AgentView{
		Ref:              entry.Ref(),
		Name:             entry.LogicalName,
		DisplayName:      entry.DisplayName,
		Description:      description,
		State:            entry.State,
		Enabled:          entry.Enabled,
		Revision:         entry.Revision,
		DefinitionDigest: digest,
		BuiltIn:          entry.RootID == agentBuiltinRootID(),
		Managed:          managed,
	}
}
