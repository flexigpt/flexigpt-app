package consumerapi

import (
	"context"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

func (a *API) listServers(
	ctx context.Context,
	request ListServersRequest,
) ([]ServerListItem, error) {
	if a == nil || a.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	entries, err := a.artifacts.ListByRoot(
		ctx,
		request.RootID,
		catalog.ListOptions{
			IncludeDocument: request.IncludeDocument,
			Kind:            mcpDomain.MCPArtifactKind,
			Enabled:         request.Enabled,
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
		if request.IncludeDocument && entry.Document != nil {
			document, err := a.serverListDocument(entry)
			if err != nil {
				return nil, err
			}
			item.Document = document
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
	if a == nil || a.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	entries, err := a.artifacts.ListByRoot(
		ctx,
		request.RootID,
		catalog.ListOptions{
			IncludeDocument: request.IncludeDocument,
			Kind:            mcpDomain.MCPPolicyArtifactKind,
			Enabled:         request.Enabled,
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
		if request.IncludeDocument && entry.Document != nil {
			document, err := a.policyListDocument(entry)
			if err != nil {
				return nil, err
			}
			item.Document = document
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

func (a *API) serverListDocument(
	entry catalog.Entry,
) (*mcpDomainServer.ServerDocument, error) {
	if entry.Definition == nil || entry.Document == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	value, err := a.serverListDocuments.GetOrLoad(
		definition.Key{RootID: entry.RootID, Digest: entry.Definition.Digest},
		len(entry.Document.Body),
		func() (mcpDomainServer.ServerDocument, error) {
			return mcpDomainServer.ServerDocumentFromDefinition(*entry.Document)
		},
	)
	if err != nil {
		return nil, err
	}
	owned := value.Clone()
	return &owned, nil
}

func (a *API) policyListDocument(
	entry catalog.Entry,
) (*mcppolicyv1.MCPPolicyDocument, error) {
	if entry.Definition == nil || entry.Document == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	value, err := a.policyListDocuments.GetOrLoad(
		definition.Key{RootID: entry.RootID, Digest: entry.Definition.Digest},
		len(entry.Document.Body),
		func() (mcppolicyv1.MCPPolicyDocument, error) {
			return mcppolicyv1.DecodeMCPPolicyJSON(entry.Document.Body)
		},
	)
	if err != nil {
		return nil, err
	}
	owned, err := value.Clone()
	if err != nil {
		return nil, err
	}
	return &owned, nil
}
