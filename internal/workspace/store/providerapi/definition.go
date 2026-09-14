package providerapi

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

// DefinitionForEntry projects one fully named canonical declaration into the
// generic Store Definition model. The Store remains unaware of this mapping.
func DefinitionForEntry(
	entry declaration.Entry,
) (definition.Definition, error) {
	if err := entry.Validate(); err != nil {
		return definition.Definition{}, err
	}
	if entry.Header().Name == "" {
		return definition.Definition{}, fmt.Errorf(
			"%w: standalone Definition projection requires a named declaration",
			basespec.ErrInvalid,
		)
	}

	raw, err := entry.CanonicalJSON()
	if err != nil {
		return definition.Definition{}, err
	}

	switch entry.Header().Type {
	case declaration.TypeInstruction:
		value, err := instructionv1.DecodeInstructionJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := contextv1.DecodeContextJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := toolv1.DecodeToolJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := modelv1.DecodeModelJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := skillv1.DecodeSkillJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := mcpv1.DecodeMCPJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		return mcpv1.DefinitionForDeclaration(value)

	case declaration.TypeMCPPolicy:
		value, err := mcppolicyv1.DecodeMCPPolicyJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := collectionv1.DecodeCollectionJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := agentv1.DecodeAgentJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := teamv1.DecodeTeamJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := loopv1.DecodeLoopJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := workflowv1.DecodeWorkflowJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
		value, err := workspacev1.DecodeWorkspaceJSON(raw)
		if err != nil {
			return definition.Definition{}, err
		}
		body, err := value.CanonicalJSON()
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
			entry.Header().Type,
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
