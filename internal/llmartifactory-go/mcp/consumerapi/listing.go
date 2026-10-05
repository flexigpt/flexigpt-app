package consumerapi

import (
	"context"
	"sort"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
)

func (a *API) listServers(
	ctx context.Context,
	request ListServersRequest,
) ([]ServerListItem, error) {
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	entries, err := a.cat.ListByRoot(
		ctx,
		request.RootID,
		catalogModel.ListOptions{
			Kind:    mcpDomain.MCPArtifactKind,
			Enabled: request.Enabled,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]ServerListItem, 0)
	for _, entry := range entries {
		if entry.Kind != mcpDomain.MCPArtifactKind {
			continue
		}
		if request.Enabled != nil && entry.Enabled != *request.Enabled {
			continue
		}

		item := ServerListItem{
			Ref:         entry.Ref(),
			Name:        entry.LogicalName,
			DisplayName: entry.DisplayName,
			State:       entry.State,
			Enabled:     entry.Enabled,
			Revision:    entry.Revision,
			BuiltIn:     a.protection.IsProtectedRoot(entry.RootID),
		}
		if entry.Definition != nil {
			item.DefinitionDigest = entry.Definition.Digest
			item.Description = entry.Definition.Description
		}
		output = append(output, item)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}

func (a *API) listPolicies(
	ctx context.Context,
	request ListPoliciesRequest,
) ([]PolicyListItem, error) {
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	entries, err := a.cat.ListByRoot(
		ctx,
		request.RootID,
		catalogModel.ListOptions{
			Kind:    mcpDomain.MCPPolicyArtifactKind,
			Enabled: request.Enabled,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]PolicyListItem, 0)
	for _, entry := range entries {
		if entry.Kind != mcpDomain.MCPPolicyArtifactKind {
			continue
		}
		if request.Enabled != nil && entry.Enabled != *request.Enabled {
			continue
		}

		item := PolicyListItem{
			Ref:         entry.Ref(),
			Name:        entry.LogicalName,
			DisplayName: entry.DisplayName,
			State:       entry.State,
			Enabled:     entry.Enabled,
			Revision:    entry.Revision,
			BuiltIn:     a.protection.IsProtectedRoot(entry.RootID),
		}
		if entry.Definition != nil {
			item.DefinitionDigest = entry.Definition.Digest
			item.Description = entry.Definition.Description
		}
		output = append(output, item)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}
