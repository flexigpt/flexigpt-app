package agent

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
)

func (a *Service) GetAgent(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (AgentView, error) {
	record, err := a.getAgentRecord(ctx, ref)
	if err != nil {
		return AgentView{}, err
	}
	return a.agentView(ctx, record)
}

func (a *Service) ListAgents(
	ctx context.Context,
	request ListAgentsRequest,
) ([]AgentListItem, error) {
	if a == nil || a.artifacts == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Agent list context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	names := make(map[spec.LogicalName]struct{}, len(request.LogicalNames))
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

	allowedRefs := map[artifactModel.ArtifactRef]struct{}(nil)
	if request.Plugin != nil {
		if err := request.Plugin.Validate(); err != nil {
			return nil, err
		}
		if request.Plugin.RootID != request.RootID {
			return nil, fmt.Errorf(
				"%w: Agent Plugin belongs to another Root",
				spec.ErrInvalid,
			)
		}
		if request.IncludeBuiltin {
			return nil, fmt.Errorf(
				"%w: plugin-filtered Agent lists cannot include another Root",
				spec.ErrInvalid,
			)
		}

		refs, err := a.listPluginAgentRefs(ctx, *request.Plugin)
		if err != nil {
			return nil, err
		}
		allowedRefs = make(map[artifactModel.ArtifactRef]struct{}, len(refs))
		for _, ref := range refs {
			allowedRefs[ref] = struct{}{}
		}
	}

	rootIDs := []rootModel.RootID{request.RootID}
	if request.IncludeBuiltin &&
		request.RootID != topology.BuiltinRootID() {
		rootIDs = append(rootIDs, topology.BuiltinRootID())
	}

	entries := make([]catalogModel.Entry, 0)
	for _, rootID := range rootIDs {
		values, err := a.cat.ListByRoot(
			ctx,
			rootID,
			catalogModel.ListOptions{
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

	seen := make(map[artifactModel.ArtifactRef]struct{}, len(entries))
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
	entry catalogModel.Entry,
) AgentView {
	digest := cryptoutil.Digest("")
	description := ""
	if entry.Definition != nil {
		digest = entry.Definition.Digest
		description = entry.Definition.Description
	}

	managed := false
	if entry.State == artifactModel.StateAvailable &&
		entry.Source.Kind == managedfs.Kind &&
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
