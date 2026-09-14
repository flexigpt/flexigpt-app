package decoder

import (
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
)

// ValidateEntryTree validates every concrete nested declaration in a
// canonical declaration document. Exact type/name references are intentionally
// not decoded as declarations because they are symbolic references.
func ValidateEntryTree(root declaration.Entry) error {
	return validateEntryTree(root, false, 0)
}

func validateEntryTree(
	entry declaration.Entry,
	implicitLoopBody bool,
	depth int,
) error {
	if depth > basespec.MaxDiscoveryDepth {
		return fmt.Errorf(
			"%w: declaration nesting exceeds depth %d",
			basespec.ErrInvalid,
			basespec.MaxDiscoveryDepth,
		)
	}
	if err := entry.Validate(); err != nil {
		return err
	}
	if entry.IsSymbolic() {
		return nil
	}

	switch entry.Header().Type {
	case declaration.TypeInstruction:
		_, err := instructionv1.DecodeInstructionEntry(entry)
		return err

	case declaration.TypeContext:
		_, err := contextv1.DecodeContextEntry(entry)
		return err

	case declaration.TypeTool:
		_, err := toolv1.DecodeToolEntry(entry)
		return err

	case declaration.TypeModel:
		_, err := modelv1.DecodeModelEntry(entry)
		return err

	case declaration.TypeSkill:
		value, err := skillv1.DecodeSkillEntry(entry)
		if err != nil {
			return err
		}
		return validateEntryTreeSlice(
			"skill allowedTools",
			value.AllowedTools,
			depth+1,
		)

	case declaration.TypeMCP:
		_, err := mcpv1.DecodeMCPEntry(entry)
		return err

	case declaration.TypeMCPPolicy:
		_, err := mcppolicyv1.DecodeMCPPolicyEntry(entry)
		return err

	case declaration.TypeCollection:
		value, err := collectionv1.DecodeCollectionEntry(entry)
		if err != nil {
			return err
		}
		return validateEntryTreeSlice(
			"collection members",
			value.Members,
			depth+1,
		)

	case declaration.TypeAgent:
		value, err := agentv1.DecodeAgentEntry(entry)
		if err != nil {
			return err
		}
		if err := validateEntryTreeSlice(
			"agent members",
			value.Members,
			depth+1,
		); err != nil {
			return err
		}
		if value.Program == nil {
			return nil
		}
		return validateEntryTree(
			*value.Program,
			true,
			depth+1,
		)

	case declaration.TypeTeam:
		value, err := teamv1.DecodeTeamEntry(entry)
		if err != nil {
			return err
		}
		if err := validateEntryTreeSlice(
			"team members",
			value.Members,
			depth+1,
		); err != nil {
			return err
		}
		if value.Program == nil {
			return nil
		}
		return validateEntryTree(
			*value.Program,
			true,
			depth+1,
		)

	case declaration.TypeLoop:
		value, err := loopv1.DecodeLoopEntry(
			entry,
			implicitLoopBody,
		)
		if err != nil {
			return err
		}
		if value.Body == nil {
			return nil
		}
		return validateEntryTree(*value.Body, false, depth+1)

	case declaration.TypeWorkflow:
		value, err := workflowv1.DecodeWorkflowEntry(entry)
		if err != nil {
			return err
		}
		for index, node := range value.Nodes {
			if err := validateEntryTree(
				node.Target,
				false,
				depth+1,
			); err != nil {
				return fmt.Errorf(
					"workflow nodes[%d].target: %w",
					index,
					err,
				)
			}
		}
		return nil

	case declaration.TypeWorkspace:
		value, err := workspacev1.DecodeWorkspaceEntry(entry)
		if err != nil {
			return err
		}
		if err := validateEntryTreeSlice(
			"workspace roots",
			value.Roots,
			depth+1,
		); err != nil {
			return err
		}
		for index, source := range value.Declarations {
			nested, found, err := source.AsEntry()
			if err != nil {
				return fmt.Errorf(
					"workspace declarations[%d]: %w",
					index,
					err,
				)
			}
			if !found {
				continue
			}
			if err := validateEntryTree(
				nested,
				false,
				depth+1,
			); err != nil {
				return fmt.Errorf(
					"workspace declarations[%d]: %w",
					index,
					err,
				)
			}
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: unsupported declaration type %q",
			basespec.ErrUnsupported,
			entry.Header().Type,
		)
	}
}

func validateEntryTreeSlice(
	label string,
	entries []declaration.Entry,
	depth int,
) error {
	for index, entry := range entries {
		if err := validateEntryTree(entry, false, depth); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
	}
	return nil
}
