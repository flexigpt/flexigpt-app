package domain

import (
	"errors"
	"fmt"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
)

var ErrNotToolCollection = errors.New("not a Tool Collection")

type Tool struct {
	Artifact   artifact.Artifact
	Definition definition.Definition
	Document   toolv1.ToolDocument
}

type ToolCollection struct {
	Artifact   artifact.Artifact
	Definition definition.Definition
	Document   pluginv1.PluginDocument
	ToolNames  []basespec.LogicalName
}

type ResolvedTool struct {
	Tool       Tool
	Collection ToolCollection
}

func (v ResolvedTool) Enabled() bool {
	return v.Tool.Artifact.Enabled &&
		v.Collection.Artifact.Enabled
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
	if value.Kind != ToolArtifactKind ||
		value.SchemaID != toolv1.ToolSchemaKey.SchemaID ||
		value.SchemaVersion != toolv1.ToolSchemaKey.SchemaVersion {
		return Tool{}, fmt.Errorf(
			"%w: Tool Artifact %q has an unsupported Tool schema",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	document, err := toolv1.DecodeToolJSON(value.Body)
	if err != nil {
		return Tool{}, err
	}
	if record.LogicalName != basespec.LogicalName(document.Name) {
		return Tool{}, fmt.Errorf(
			"%w: Tool Artifact identity differs from Tool declaration",
			basespec.ErrDigestMismatch,
		)
	}
	return Tool{
		Artifact:   record.Clone(),
		Definition: value.Clone(),
		Document:   document,
	}, nil
}

func DecodeToolCollection(
	record artifact.Artifact,
	value definition.Definition,
) (ToolCollection, error) {
	if record.Kind != artifact.ArtifactKind(pluginv1.PluginType) ||
		record.Binding.SubresourceLocator != "" {
		return ToolCollection{}, fmt.Errorf(
			"%w: Artifact %q is not a Tool Collection",
			ErrNotToolCollection,
			record.ID,
		)
	}
	if record.State != artifact.StateAvailable {
		return ToolCollection{}, fmt.Errorf(
			"%w: Tool Collection Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	document, err := pluginv1.DecodePluginJSON(value.Body)
	if err != nil {
		return ToolCollection{}, err
	}
	if document.Locator != nil {
		return ToolCollection{}, fmt.Errorf(
			"%w: located Plugin aliases are not Tool Collections",
			ErrNotToolCollection,
		)
	}
	if record.LogicalName != basespec.LogicalName(document.Name) {
		return ToolCollection{}, fmt.Errorf(
			"%w: Tool Collection identity differs from Plugin declaration",
			basespec.ErrDigestMismatch,
		)
	}

	tools, err := ValidateToolCollectionDocument(document)
	if err != nil {
		return ToolCollection{}, err
	}

	return ToolCollection{
		Artifact:   record.Clone(),
		Definition: value.Clone(),
		Document:   document,
		ToolNames:  tools,
	}, nil
}

func ValidateToolCollectionDocument(
	document pluginv1.PluginDocument,
) ([]basespec.LogicalName, error) {
	if document.Locator != nil {
		return nil, fmt.Errorf(
			"%w: located Plugin aliases are not Tool Collections",
			ErrNotToolCollection,
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
	tools := make([]basespec.LogicalName, 0, len(ordered))
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
				"%w: Tool Collection members must be named built-in Tool references",
				ErrNotToolCollection,
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
				ErrNotToolCollection,
				header.Name,
			)
		}

		name := basespec.LogicalName(header.Name)
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Tool Collection repeats Tool %q",
				basespec.ErrIdentityConflict,
				name,
			)
		}
		seen[name] = struct{}{}
		tools = append(tools, name)
	}

	slices.Sort(tools)
	return tools, nil
}

func (v ToolCollection) HasTool(
	name basespec.LogicalName,
) bool {
	return slices.Contains(v.ToolNames, name)
}
