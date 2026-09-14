package decoder

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/instructionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/loopv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/teamv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workflowv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
)

// DefinitionForEntry projects one named canonical declaration Entry into the
// generic immutable Artifact Store Definition model.
//
// Entry bytes are already canonical. Projection validates the concrete
// declaration type but retains those exact canonical bytes as Definition.Body.
func DefinitionForEntry(
	entry declaration.Entry,
) (definition.Definition, error) {
	if err := ValidateEntryTree(entry); err != nil {
		return definition.Definition{}, err
	}
	return definitionForEntry(entry)
}

// DefinitionForNamedEntry projects one already tree-validated named entry.
// Canonical and source-format decoders use this after validating the complete
// containing declaration and walking its named entries.
func DefinitionForNamedEntry(
	named declaration.NamedEntry,
) (definition.Definition, error) {
	return definitionForNamedEntry(named)
}

// definitionForNamedEntry is used by canonicalDecoder after the complete
// containing document has already passed ValidateEntryTree.
func definitionForNamedEntry(
	named declaration.NamedEntry,
) (definition.Definition, error) {
	if err := named.Validate(); err != nil {
		return definition.Definition{}, err
	}
	return definitionForEntry(named.Entry)
}

func definitionForEntry(
	entry declaration.Entry,
) (definition.Definition, error) {
	if err := entry.Validate(); err != nil {
		return definition.Definition{}, err
	}

	header := entry.Header()
	if header.Name == "" {
		return definition.Definition{}, fmt.Errorf(
			"%w: Definition projection requires a named declaration",
			basespec.ErrInvalid,
		)
	}

	body, err := entry.CanonicalJSON()
	if err != nil {
		return definition.Definition{}, err
	}

	switch header.Type {
	case declaration.TypeInstruction:
		value, err := instructionv1.DecodeInstructionEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			instructionv1.InstructionSchemaKey,
			"",
			body,
		)

	case declaration.TypeContext:
		value, err := contextv1.DecodeContextEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			contextv1.ContextSchemaKey,
			"",
			body,
		)

	case declaration.TypeTool:
		value, err := toolv1.DecodeToolEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			toolv1.ToolSchemaKey,
			"",
			body,
		)

	case declaration.TypeModel:
		value, err := modelv1.DecodeModelEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			modelv1.ModelSchemaKey,
			"",
			body,
		)

	case declaration.TypeSkill:
		value, err := skillv1.DecodeSkillEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			skillv1.SkillSchemaKey,
			"",
			body,
		)

	case declaration.TypeMCP:
		value, err := mcpv1.DecodeMCPEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			mcpv1.MCPSchemaKey,
			"",
			body,
		)

	case declaration.TypeMCPPolicy:
		value, err := mcppolicyv1.DecodeMCPPolicyEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			mcppolicyv1.MCPPolicySchemaKey,
			"",
			body,
		)

	case declaration.TypeCollection:
		value, err := collectionv1.DecodeCollectionEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			collectionv1.CollectionSchemaKey,
			basespec.LogicalVersion(value.Version),
			body,
		)

	case declaration.TypeAgent:
		value, err := agentv1.DecodeAgentEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			agentv1.AgentSchemaKey,
			"",
			body,
		)

	case declaration.TypeTeam:
		value, err := teamv1.DecodeTeamEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			teamv1.TeamSchemaKey,
			"",
			body,
		)

	case declaration.TypeLoop:
		value, err := loopv1.DecodeLoopEntry(
			entry,
			false,
		)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			loopv1.LoopSchemaKey,
			"",
			body,
		)

	case declaration.TypeWorkflow:
		value, err := workflowv1.DecodeWorkflowEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			workflowv1.WorkflowSchemaKey,
			"",
			body,
		)

	case declaration.TypeWorkspace:
		value, err := workspacev1.DecodeWorkspaceEntry(entry)
		if err != nil {
			return definition.Definition{}, err
		}
		return definitionForDocument(
			value.Header,
			workspacev1.WorkspaceSchemaKey,
			"",
			body,
		)

	default:
		return definition.Definition{}, fmt.Errorf(
			"%w: unsupported canonical declaration type %q",
			basespec.ErrUnsupported,
			header.Type,
		)
	}
}

func definitionForDocument(
	header declaration.Header,
	key schema.Key,
	logicalVersion basespec.LogicalVersion,
	body []byte,
) (definition.Definition, error) {
	if err := key.Validate(); err != nil {
		return definition.Definition{}, err
	}
	if err := header.Validate(declaration.HeaderValidation{
		ExpectedType: declaration.Type(key.Kind),
		APIVersion:   key.SchemaVersion,
		RequireName:  true,
	}); err != nil {
		return definition.Definition{}, err
	}

	value := definition.Definition{
		Kind:           artifact.ArtifactKind(key.Kind),
		SchemaID:       key.SchemaID,
		SchemaVersion:  key.SchemaVersion,
		LogicalName:    basespec.LogicalName(header.Name),
		LogicalVersion: logicalVersion,
		DisplayName:    header.Name,
		Description:    header.Description,
		Body:           json.RawMessage(body),
	}
	return definition.Canonicalize(value)
}
