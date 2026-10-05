package domain

import (
	"fmt"
	"slices"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/toolv1"
)

// Tool is internal decoded Store material. Consumer and Wails APIs expose
// explicit views instead of serializing this value.
type Tool struct {
	Artifact   artifactModel.Artifact
	Definition definitionModel.Definition
	Document   toolv1.ToolDocument
}

func DecodeTool(
	record artifactModel.Artifact,
	value definitionModel.Definition,
) (Tool, error) {
	if record.Kind != ToolArtifactKind {
		return Tool{}, fmt.Errorf(
			"%w: Artifact %q is not a Tool",
			spec.ErrUnsupported,
			record.ID,
		)
	}
	if record.State != artifactModel.StateAvailable {
		return Tool{}, fmt.Errorf(
			"%w: Tool Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
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
			spec.ErrUnsupported,
			record.ID,
		)
	}
	if record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != value.Digest {
		return Tool{}, fmt.Errorf(
			"%w: Tool definition changed during read",
			spec.ErrRefreshRequired,
		)
	}

	document, err := toolv1.DecodeToolJSON(value.Body)
	if err != nil {
		return Tool{}, err
	}
	if record.LogicalName != spec.LogicalName(document.Name) {
		return Tool{}, fmt.Errorf(
			"%w: tool identity differs from its declaration",
			spec.ErrDigestMismatch,
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
) ([]spec.LogicalName, error) {
	if err := document.Validate(); err != nil {
		return nil, err
	}
	if document.Locator != nil {
		return nil, fmt.Errorf(
			"%w: Tool Collections cannot be located aliases",
			spec.ErrUnsupported,
		)
	}

	ordered, err := declaration.SortedMembers(
		"Tool Collection members",
		document.Members,
	)
	if err != nil {
		return nil, err
	}

	seen := make(map[spec.LogicalName]struct{}, len(ordered))
	names := make([]spec.LogicalName, 0, len(ordered))
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
				spec.ErrUnsupported,
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
				spec.ErrUnsupported,
				header.Name,
			)
		}

		name := spec.LogicalName(header.Name)
		if err := name.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: Tool Collection repeats Tool %q",
				spec.ErrIdentityConflict,
				name,
			)
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}

	slices.Sort(names)
	return names, nil
}
