package decoder

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/loopv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/skillv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/teamv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workflowv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// ValidateEntryTree validates every concrete nested declaration in a
// canonical declaration document. External composition references are
// validated as edges and intentionally are not decoded as concrete
// declarations.
func ValidateEntryTree(root declaration.Entry) error {
	return validateEntryTree(root, 0)
}

func validateEntryTree(
	entry declaration.Entry,
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

	switch entry.Header().Type {
	case declaration.TypeText:
		_, err := textv1.DecodeTextEntry(entry)
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
		return validateMemberTreeSlice(
			"skill allowedTools",
			value.AllowedTools,
			depth+1,
			true,
		)

	case declaration.TypeMCP:
		_, err := mcpv1.DecodeMCPEntry(entry)
		return err

	case declaration.TypeMCPPolicy:
		_, err := mcppolicyv1.DecodeMCPPolicyEntry(entry)
		return err

	case declaration.TypePlugin:
		value, err := pluginv1.DecodePluginEntry(entry)
		if err != nil {
			return err
		}
		return validateMemberTreeSlice(
			"plugin members",
			value.Members,
			depth+1,
			true,
		)

	case declaration.TypeAgent:
		value, err := agentv1.DecodeAgentEntry(entry)
		if err != nil {
			return err
		}
		if err := validateMemberTreeSlice(
			"agent members",
			value.Members,
			depth+1,
			true,
		); err != nil {
			return err
		}
		for label, program := range map[string]*declaration.Entry{
			"agent loop":     value.Loop,
			"agent workflow": value.Workflow,
		} {
			if program == nil {
				continue
			}
			if err := validateMemberTree(label, *program, depth+1, false); err != nil {
				return err
			}
		}
		return nil

	case declaration.TypeTeam:
		value, err := teamv1.DecodeTeamEntry(entry)
		if err != nil {
			return err
		}
		if err := validateMemberTreeSlice(
			"team members",
			value.Members,
			depth+1,
			true,
		); err != nil {
			return err
		}
		for label, program := range map[string]*declaration.Entry{
			"team loop":     value.Loop,
			"team workflow": value.Workflow,
		} {
			if program == nil {
				continue
			}
			if err := validateMemberTree(label, *program, depth+1, false); err != nil {
				return err
			}
		}
		return nil

	case declaration.TypeLoop:
		value, err := loopv1.DecodeLoopEntry(entry)
		if err != nil {
			return err
		}
		if value.Body == nil {
			return nil
		}
		return validateMemberTree("loop body", *value.Body, depth+1, false)

	case declaration.TypeWorkflow:
		value, err := workflowv1.DecodeWorkflowEntry(entry)
		if err != nil {
			return err
		}
		for index, node := range value.Nodes {
			if err := validateMemberTree(
				"workflow node",
				node.Member,
				depth+1,
				false,
			); err != nil {
				return fmt.Errorf(
					"workflow nodes[%d]: %w",
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
		return validateMemberTreeSlice(
			"workspace members",
			value.Members,
			depth+1,
			true,
		)

	default:
		return fmt.Errorf(
			"%w: unsupported declaration type %q",
			basespec.ErrUnsupported,
			entry.Header().Type,
		)
	}
}

func validateMemberTreeSlice(
	label string,
	entries []declaration.Entry,
	depth int,
	allowSelectors bool,
) error {
	if err := declaration.ValidateMemberUniqueness(
		label,
		entries,
	); err != nil {
		return err
	}
	for index, entry := range entries {
		if err := validateMemberTree(
			label,
			entry,
			depth,
			allowSelectors,
		); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
	}
	return nil
}

func validateMemberTree(
	label string,
	member declaration.Entry,
	depth int,
	allowSelector bool,
) error {
	form, err := member.MemberForm()
	if err != nil {
		return err
	}
	switch form {
	case declaration.MemberNamed:
		return nil
	case declaration.MemberSelector:
		if allowSelector {
			return nil
		}
		return fmt.Errorf(
			"%w: %s does not allow member selectors",
			basespec.ErrInvalid,
			label,
		)
	case declaration.MemberContained:
		target, err := member.ContainedDeclaration()
		if err != nil {
			return err
		}
		return validateEntryTree(target, depth)
	default:
		return fmt.Errorf("%w: unknown member form", basespec.ErrInvalid)
	}
}
