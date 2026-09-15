package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/sourceformat"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

func (a *API) serverDocumentForResolvedArtifact(
	ctx context.Context,
	resolved resource.ResolvedArtifact,
) (mcpDomainServer.ServerDocument, error) {
	outer, err := mcpv1.DecodeMCPJSON(resolved.Definition.Body)
	if err != nil {
		return mcpDomainServer.ServerDocument{}, err
	}

	// A locator remains implementation data when the declaration already has
	// complete connection details. Otherwise it identifies the source that
	// supplies the missing command or URL.
	if outer.Locator == nil ||
		outer.Locator.Kind == declaration.LocatorKindCommand ||
		hasInlineMCPConnection(outer) {
		return mcpDomainServer.ServerDocumentFromDefinition(
			resolved.Definition,
		)
	}

	target, err := declaration.ResolveSourceRelativePathLocator(
		*outer.Locator,
		resolved.Artifact.Binding.Locator,
	)
	if err != nil {
		return mcpDomainServer.ServerDocument{}, err
	}
	entry, err := a.resources.ReadSourceEntry(
		ctx,
		resolved.Artifact.RootID,
		resolved.Artifact.Binding.SourceID,
		target,
		basespec.MaxCandidateBytes,
	)
	if err != nil {
		return mcpDomainServer.ServerDocument{}, err
	}
	if entry.SourceRevision !=
		resolved.RefreshState.SourceRevision ||
		entry.SourceGeneration !=
			resolved.RefreshState.SourceGeneration {
		return mcpDomainServer.ServerDocument{}, fmt.Errorf(
			"%w: located MCP source changed during resolution",
			basespec.ErrRefreshRequired,
		)
	}

	definitions, err := decodeLocatedMCPDefinitions(entry.Content)
	if err != nil {
		return mcpDomainServer.ServerDocument{}, err
	}
	selected, err := selectLocatedMCPDefinition(
		definitions,
		outer.Server,
		resolved.Definition.LogicalName,
	)
	if err != nil {
		return mcpDomainServer.ServerDocument{}, err
	}
	if err := requireTerminalLocatedMCPDefinition(selected); err != nil {
		return mcpDomainServer.ServerDocument{}, err
	}
	document, err := mcpDomainServer.ServerDocumentFromDefinition(selected)
	if err != nil {
		return mcpDomainServer.ServerDocument{}, err
	}
	return mcpDomainServer.RebindLocatedDocument(
		document,
		resolved.Definition,
		outer,
	)
}

func requireTerminalLocatedMCPDefinition(
	value definition.Definition,
) error {
	document, err := mcpv1.DecodeMCPJSON(value.Body)
	if err != nil {
		return err
	}
	if document.Locator != nil &&
		document.Locator.Kind != declaration.LocatorKindCommand {
		return fmt.Errorf(
			"%w: source-selected MCP target %q is another source-selected declaration; chained MCP source selection is not supported",
			basespec.ErrLocatorUnresolved,
			value.LogicalName,
		)
	}
	return nil
}

func hasInlineMCPConnection(
	value mcpv1.MCPDocument,
) bool {
	switch value.Transport {
	case mcpv1.TransportStdio:
		return value.Command != ""
	case mcpv1.TransportStreamableHTTP, mcpv1.TransportSSE:
		return value.URL != ""
	default:
		return false
	}
}

func decodeLocatedMCPDefinitions(
	content []byte,
) ([]definition.Definition, error) {
	if sourceformat.IsRetiredMCPCollection(content) {
		return nil, fmt.Errorf(
			"%w: proprietary MCP collection manifests are retired; use a canonical Collection Artifact",
			basespec.ErrUnsupported,
		)
	}
	if sourceformat.IsMCPConfig(content) {
		values, err := sourceformat.DecodeMCPConfig(content)
		if err != nil {
			return nil, err
		}
		return decodedDefinitions(values), nil
	}

	raw, err := yamlutil.CanonicalObjectJSON(
		content,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: located MCP source is not canonical JSON or YAML",
			basespec.ErrLocatorUnresolved,
		)
	}
	document, err := mcpv1.DecodeMCPJSON(raw)
	if err != nil {
		return nil, err
	}
	value, err := mcpv1.DefinitionForDeclaration(document)
	if err != nil {
		return nil, err
	}
	return []definition.Definition{value}, nil
}

func decodedDefinitions(
	values []sourceformat.Decoded,
) []definition.Definition {
	output := make([]definition.Definition, 0, len(values))
	for _, value := range values {
		output = append(output, value.Definition.Clone())
	}
	return output
}

func selectLocatedMCPDefinition(
	values []definition.Definition,
	server string,
	fallbackName basespec.LogicalName,
) (definition.Definition, error) {
	candidates := make([]definition.Definition, 0)
	for _, value := range values {
		if value.Kind != domain.MCPArtifactKind {
			continue
		}
		if server != "" &&
			value.LogicalName != basespec.LogicalName(server) {
			continue
		}
		candidates = append(candidates, value.Clone())
	}

	if server == "" && len(candidates) > 1 {
		byOuterName := make([]definition.Definition, 0, 1)
		for _, candidate := range candidates {
			if candidate.LogicalName == fallbackName {
				byOuterName = append(byOuterName, candidate)
			}
		}
		if len(byOuterName) == 1 {
			candidates = byOuterName
		}
	}

	switch len(candidates) {
	case 0:
		return definition.Definition{}, fmt.Errorf(
			"%w: located MCP server %q was not found",
			basespec.ErrReferenceUnresolved,
			server,
		)
	case 1:
		return candidates[0], nil
	default:
		return definition.Definition{}, fmt.Errorf(
			"%w: located MCP source has %d matching servers",
			basespec.ErrIdentityConflict,
			len(candidates),
		)
	}
}
