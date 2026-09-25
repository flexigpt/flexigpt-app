package domain

import (
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
)

// Tool is internal decoded Store material. Consumer and Wails APIs expose
// explicit views instead of serializing this value.
type Tool struct {
	Artifact   artifact.Artifact
	Definition definition.Definition
	Document   toolv1.ToolDocument
}

func DecodeTool(
	record artifact.Artifact,
	value definition.Definition,
) (Tool, error) {
	if record.Kind != ToolArtifactKind {
		return Tool{}, fmt.Errorf(
			"%w: Artifact %q is not a Tool",
			basespec.ErrUnsupported,
			record.ID,
		)
	}
	if record.State != artifact.StateAvailable {
		return Tool{}, fmt.Errorf(
			"%w: Tool Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if err := value.Validate(); err != nil {
		return Tool{}, err
	}
	if value.Kind != ToolArtifactKind ||
		value.SchemaID != toolv1.ToolSchemaKey.SchemaID ||
		value.SchemaVersion != toolv1.ToolSchemaKey.SchemaVersion {
		return Tool{}, fmt.Errorf(
			"%w: Tool Artifact %q has an unsupported schema",
			basespec.ErrUnsupported,
			record.ID,
		)
	}
	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != value.Digest {
		return Tool{}, fmt.Errorf(
			"%w: Tool definition changed during read",
			basespec.ErrRefreshRequired,
		)
	}

	document, err := toolv1.DecodeToolJSON(value.Body)
	if err != nil {
		return Tool{}, err
	}
	if record.LogicalName != basespec.LogicalName(document.Name) {
		return Tool{}, fmt.Errorf(
			"%w: tool identity differs from its declaration",
			basespec.ErrDigestMismatch,
		)
	}

	return Tool{
		Artifact:   record.Clone(),
		Definition: value.Clone(),
		Document:   document,
	}, nil
}

// ValidateToolCollectionDocument applies the restrictions of the built-in
// Tool catalog. Generic Collection decoding and projection remain owned by
// the collection package.
func ValidateToolCollectionDocument(
	document pluginv1.PluginDocument,
) ([]basespec.LogicalName, error) {
	if err := document.Validate(); err != nil {
		return nil, err
	}
	if document.Locator != nil {
		return nil, fmt.Errorf(
			"%w: Tool Collections cannot be located aliases",
			basespec.ErrUnsupported,
		)
	}

	ordered, err := declaration.SortedMembers(
		"Tool Collection members",
		document.Members,
	)
	if err != nil {
		return nil, err
	}

	seen := make(map[basespec.LogicalName]struct{}, len(ordered))
	names := make([]basespec.LogicalName, 0, len(ordered))
	for index, member := range ordered {
		form, err := member.MemberForm()
		if err != nil {
			return nil, fmt.Errorf(
				"Tool Collection members[%d]: %w",
				index,
				err,
			)
		}
		header := member.Header()
		if form != declaration.MemberNamed ||
			header.Type != declaration.TypeTool ||
			header.Locator != nil {
			return nil, fmt.Errorf(
				"%w: Tool Collections require named built-in Tool references",
				basespec.ErrUnsupported,
			)
		}

		relationship, err := member.Relationship()
		if err != nil {
			return nil, err
		}
		if relationship.Scope != declaration.LookupScopeBuiltin ||
			len(relationship.Overrides) != 0 ||
			len(relationship.Use) != 0 {
			return nil, fmt.Errorf(
				"%w: Tool Collection member %q has unsupported relationship behavior",
				basespec.ErrUnsupported,
				header.Name,
			)
		}

		name := basespec.LogicalName(header.Name)
		if err := name.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Tool Collection repeats Tool %q",
				basespec.ErrIdentityConflict,
				name,
			)
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}

	slices.Sort(names)
	return names, nil
}
