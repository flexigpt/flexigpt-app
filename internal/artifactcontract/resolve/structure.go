package resolve

import (
	"context"
	"fmt"
	"sort"

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
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

func (r *Resolver) resolveStructure(
	ctx context.Context,
	state *resolutionState,
	node *ResolvedEntry,
	entry declaration.Entry,
	depth int,
) error {
	rootID, found := node.RootID()
	if !found {
		return fmt.Errorf(
			"%w: resolved declaration has no Root scope",
			basespec.ErrInvalid,
		)
	}
	if node.Artifact == nil {
		return fmt.Errorf(
			"%w: mapped fallback target cannot declare child relationships",
			basespec.ErrInvalid,
		)
	}
	from := node.Artifact

	switch node.Type {
	case declaration.TypeText:
		_, err := textv1.DecodeTextEntry(entry)
		return err

	case declaration.TypeModel:
		_, err := modelv1.DecodeModelEntry(entry)
		return err

	case declaration.TypeTool:
		_, err := toolv1.DecodeToolEntry(entry)
		return err

	case declaration.TypeSkill:
		value, err := skillv1.DecodeSkillEntry(entry)
		if err != nil {
			return err
		}
		node.AllowedTools, node.AllowedToolResults, err = r.resolveEntries(
			ctx,
			state,
			rootID,
			value.AllowedTools,
			from,
			depth+1,
			[]string{"allowedTools"},
		)
		return err

	case declaration.TypeMCP:
		value, err := mcpv1.DecodeMCPEntry(entry)
		if err != nil {
			return err
		}
		if value.Policy == nil {
			return nil
		}
		policyMember, err := declaration.NewSymbolicEntry(
			declaration.TypeMCPPolicy,
			value.Policy.Name,
		)
		if err != nil {
			return err
		}
		relationship, err := r.resolveMember(
			ctx,
			state,
			rootID,
			policyMember,
			from,
			depth+1,
			[]string{"policy"},
		)
		if err != nil {
			return err
		}
		required := true
		if value.Policy.Required != nil {
			required = *value.Policy.Required
		}
		relationship.Required = required
		node.MCP = &ResolvedMCP{
			Policy:         relationship.Resolved,
			PolicyResult:   pointerRelationship(relationship),
			PolicyRequired: required,
		}
		return nil

	case declaration.TypeMCPPolicy:
		_, err := mcppolicyv1.DecodeMCPPolicyEntry(entry)
		return err

	case declaration.TypePlugin:
		value, err := pluginv1.DecodePluginEntry(entry)
		if err != nil {
			return err
		}
		node.Members, node.MemberResults, err = r.resolveEntries(
			ctx,
			state,
			rootID,
			value.Members,
			from,
			depth+1,
			[]string{membersStr},
		)
		return err

	case declaration.TypeAgent:
		value, err := agentv1.DecodeAgentEntry(entry)
		if err != nil {
			return err
		}
		node.Members, node.MemberResults, err = r.resolveEntries(
			ctx,
			state,
			rootID,
			value.Members,
			from,
			depth+1,
			[]string{membersStr},
		)
		if err != nil {
			return err
		}
		if value.Loop != nil {
			relationship, err := r.resolveSingleMember(
				ctx,
				state,
				rootID,
				*value.Loop,
				from,
				depth+1,
				[]string{loopStr},
			)
			if err != nil {
				return err
			}
			node.DirectLoop = relationship.Resolved
			node.DirectLoopResult = pointerRelationship(relationship)
		}
		if value.Workflow != nil {
			relationship, err := r.resolveSingleMember(
				ctx,
				state,
				rootID,
				*value.Workflow,
				from,
				depth+1,
				[]string{workflowStr},
			)
			if err != nil {
				return err
			}
			node.DirectWorkflow = relationship.Resolved
			node.DirectWorkflowResult = pointerRelationship(relationship)
		}
		return nil

	case declaration.TypeTeam:
		value, err := teamv1.DecodeTeamEntry(entry)
		if err != nil {
			return err
		}
		node.Members, node.MemberResults, err = r.resolveEntries(
			ctx,
			state,
			rootID,
			value.Members,
			from,
			depth+1,
			[]string{membersStr},
		)
		if err != nil {
			return err
		}
		if value.Loop != nil {
			relationship, err := r.resolveSingleMember(
				ctx,
				state,
				rootID,
				*value.Loop,
				from,
				depth+1,
				[]string{loopStr},
			)
			if err != nil {
				return err
			}
			node.DirectLoop = relationship.Resolved
			node.DirectLoopResult = pointerRelationship(relationship)
		}
		if value.Workflow != nil {
			relationship, err := r.resolveSingleMember(
				ctx,
				state,
				rootID,
				*value.Workflow,
				from,
				depth+1,
				[]string{workflowStr},
			)
			if err != nil {
				return err
			}
			node.DirectWorkflow = relationship.Resolved
			node.DirectWorkflowResult = pointerRelationship(relationship)
		}
		return nil

	case declaration.TypeLoop:
		value, err := loopv1.DecodeLoopEntry(entry)
		if err != nil {
			return err
		}
		loop := &ResolvedLoop{
			MaxIterations: value.MaxIterations,
			Until:         cloneOutputMatch(value.Until),
		}
		node.Loop = loop
		if value.Body == nil {
			return nil
		}
		relationship, err := r.resolveSingleMember(
			ctx,
			state,
			rootID,
			*value.Body,
			from,
			depth+1,
			[]string{"body"},
		)
		if err != nil {
			return err
		}
		loop.Body = relationship.Resolved
		loop.BodyResult = pointerRelationship(relationship)
		return nil

	case declaration.TypeWorkflow:
		value, err := workflowv1.DecodeWorkflowEntry(entry)
		if err != nil {
			return err
		}
		nodes := append([]workflowv1.Node(nil), value.Nodes...)
		sort.SliceStable(nodes, func(left, right int) bool {
			return nodes[left].ID < nodes[right].ID
		})
		edges := append([]workflowv1.Edge(nil), value.Edges...)
		sort.SliceStable(edges, func(left, right int) bool {
			return workflowEdgeKey(edges[left]) < workflowEdgeKey(edges[right])
		})
		start := append([]string(nil), value.Start...)
		sort.Strings(start)

		workflow := &ResolvedWorkflow{
			Start: start,
			Nodes: make([]ResolvedWorkflowNode, 0, len(nodes)),
			Edges: make([]ResolvedWorkflowEdge, 0, len(edges)),
		}
		node.Workflow = workflow
		for _, workflowNode := range nodes {
			relationship, err := r.resolveSingleMember(
				ctx,
				state,
				rootID,
				workflowNode.Member,
				from,
				depth+1,
				[]string{
					"nodes",
					declaration.StableWorkflowNodeSegment(workflowNode.ID),
				},
			)
			if err != nil {
				return err
			}
			join := workflowNode.Join
			if join == "" {
				join = workflowv1.JoinAll
			}
			workflow.Nodes = append(workflow.Nodes, ResolvedWorkflowNode{
				ID:           workflowNode.ID,
				Join:         string(join),
				Member:       relationship.Resolved,
				MemberResult: pointerRelationship(relationship),
			})
		}
		for _, edge := range edges {
			workflow.Edges = append(workflow.Edges, ResolvedWorkflowEdge{
				From:  edge.From,
				To:    edge.To,
				Match: cloneOutputMatch(edge.Match),
			})
		}
		return nil

	case declaration.TypeWorkspace:
		value, err := workspacev1.DecodeWorkspaceEntry(entry)
		if err != nil {
			return err
		}
		members, results, err := r.resolveEntries(
			ctx,
			state,
			rootID,
			value.Members,
			from,
			depth+1,
			[]string{membersStr},
		)
		if err != nil {
			return err
		}
		node.Workspace = &ResolvedWorkspace{
			Members:       members,
			MemberResults: results,
		}
		return nil

	default:
		return fmt.Errorf(
			"%w: unsupported declaration type %q",
			basespec.ErrUnsupported,
			node.Type,
		)
	}
}

func (r *Resolver) resolveSingleMember(
	ctx context.Context,
	state *resolutionState,
	rootID root.RootID,
	member declaration.Entry,
	from *artifact.Artifact,
	depth int,
	relationshipPath []string,
) (ResolvedRelationship, error) {
	form, err := member.MemberForm()
	if err != nil {
		return ResolvedRelationship{}, err
	}
	if form == declaration.MemberSelector {
		return ResolvedRelationship{}, fmt.Errorf(
			"%w: singular relationship does not allow member selectors",
			basespec.ErrInvalid,
		)
	}
	return r.resolveMember(
		ctx,
		state,
		rootID,
		member,
		from,
		depth,
		relationshipPath,
	)
}

func workflowEdgeKey(value workflowv1.Edge) string {
	match := ""
	if value.Match != nil {
		match = value.Match.Pointer + "\x00" + string(value.Match.Schema)
	}
	return value.From + "\x00" + value.To + "\x00" + match
}
