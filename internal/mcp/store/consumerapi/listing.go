package consumerapi

import (
	"context"
	"errors"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
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

	if cached, found := a.serverListDocuments.Load(
		entry.Definition.Digest,
	); found {
		value, ok := cached.(mcpDomainServer.ServerDocument)
		if !ok {
			return nil, errors.New("mcp document not found")
		}
		return &value, nil
	}

	value, err := mcpDomainServer.ServerDocumentFromDefinition(
		*entry.Document,
	)
	if err != nil {
		return nil, err
	}
	actual, loaded := a.serverListDocuments.LoadOrStore(
		entry.Definition.Digest,
		value,
	)
	if loaded {
		v, ok := actual.(mcpDomainServer.ServerDocument)
		if !ok {
			return nil, errors.New("mcp document not found")
		}
		value = v
	}
	return &value, nil
}

func (a *API) policyListDocument(
	entry catalog.Entry,
) (*mcppolicyv1.MCPPolicyDocument, error) {
	if entry.Definition == nil || entry.Document == nil {
		//nolint:nilnil // Ok.
		return nil, nil
	}

	if cached, found := a.policyListDocuments.Load(
		entry.Definition.Digest,
	); found {
		value, ok := cached.(mcppolicyv1.MCPPolicyDocument)
		if !ok {
			return nil, errors.New("mcp doc not found")
		}
		return &value, nil
	}

	value, err := mcppolicyv1.DecodeMCPPolicyJSON(
		entry.Document.Body,
	)
	if err != nil {
		return nil, err
	}
	actual, loaded := a.policyListDocuments.LoadOrStore(
		entry.Definition.Digest,
		value,
	)
	if loaded {
		v, ok := actual.(mcppolicyv1.MCPPolicyDocument)
		if !ok {
			return nil, errors.New("mcp document not found")
		}
		value = v
	}
	return &value, nil
}
