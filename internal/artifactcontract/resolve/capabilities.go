package resolve

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

// CapabilityOccurrence is a cycle-safe projection of one declared
// composition relationship.
type CapabilityOccurrence struct {
	Path     string                `json:"path"`
	Type     declaration.Type      `json:"type"`
	Name     basespec.LogicalName  `json:"name"`
	Status   ResolutionStatus      `json:"status"`
	Artifact *artifact.ArtifactRef `json:"artifact,omitempty"`
	Code     string                `json:"code,omitempty"`
	Message  string                `json:"message,omitempty"`
}

// CapabilityPlan is the frontend-safe flattened relationship projection of a
// Collection, Agent, Team, Skill, Loop, Workflow, Workspace, or leaf Artifact.
type CapabilityPlan struct {
	RootArtifact *artifact.ArtifactRef  `json:"rootArtifact,omitempty"`
	RootType     declaration.Type       `json:"rootType"`
	RootName     basespec.LogicalName   `json:"rootName"`
	Occurrences  []CapabilityOccurrence `json:"occurrences"`
	Complete     bool                   `json:"complete"`
}

func (r *Resolver) ResolveCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	graph, err := r.ResolveArtifact(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return CapabilityPlanForGraph(graph)
}

func CapabilityPlanForGraph(
	graph Graph,
) (CapabilityPlan, error) {
	if graph.Root == nil {
		return CapabilityPlan{}, fmt.Errorf(
			"%w: capability graph has no root",
			basespec.ErrReferenceUnresolved,
		)
	}

	name, err := resolvedEntryName(graph.Root)
	if err != nil {
		return CapabilityPlan{}, err
	}
	plan := CapabilityPlan{
		RootType:    graph.Root.Type,
		RootName:    name,
		Occurrences: make([]CapabilityOccurrence, 0),
	}
	if ref, found := graph.Root.ArtifactRef(); found {
		value := ref
		plan.RootArtifact = &value
	}

	collector := capabilityCollector{plan: &plan}
	collector.collectEntry("", graph.Root)
	plan.Complete = RequireComplete(plan.Occurrences) == nil
	return plan, nil
}

// RequireComplete rejects the first unavailable or ambiguous relationship in
// deterministic plan order.
func RequireComplete(
	occurrences []CapabilityOccurrence,
) error {
	for _, occurrence := range occurrences {
		if occurrence.Status == ResolutionAvailable {
			continue
		}
		return fmt.Errorf(
			"%w: capability %q (%s/%s) is %s: %s",
			basespec.ErrReferenceUnresolved,
			occurrence.Path,
			occurrence.Type,
			occurrence.Name,
			occurrence.Status,
			occurrence.Message,
		)
	}
	return nil
}

type capabilityCollector struct {
	plan *CapabilityPlan
}

func (c capabilityCollector) collectRelationship(
	path string,
	relationship ResolvedRelationship,
) {
	header := relationship.Declared.Header()
	occurrence := CapabilityOccurrence{
		Path:   path,
		Type:   header.Type,
		Name:   basespec.LogicalName(header.Name),
		Status: relationship.Status,
	}
	if relationship.Issue != nil {
		occurrence.Code = relationship.Issue.Code
		occurrence.Message = relationship.Issue.Message
	}
	if relationship.Resolved != nil {
		if ref, found := relationship.Resolved.ArtifactRef(); found {
			value := ref
			occurrence.Artifact = &value
		}
	}
	c.plan.Occurrences = append(c.plan.Occurrences, occurrence)

	if relationship.IsAvailable() {
		c.collectEntry(path, relationship.Resolved)
	}
}

func (c capabilityCollector) collectEntry(
	path string,
	entry *ResolvedEntry,
) {
	if entry == nil {
		return
	}

	switch entry.Type {
	case declaration.TypeCollection,
		declaration.TypeAgent,
		declaration.TypeTeam:
		for index, relationship := range entry.MemberResults {
			c.collectRelationship(
				childPath(path, membersStr, strconv.Itoa(index)),
				relationship,
			)
		}
		if entry.ProgramResult != nil {
			c.collectRelationship(
				childPath(path, "program"),
				*entry.ProgramResult,
			)
		}

	case declaration.TypeSkill:
		for index, relationship := range entry.AllowedToolResults {
			c.collectRelationship(
				childPath(path, "allowedTools", strconv.Itoa(index)),
				relationship,
			)
		}

	case declaration.TypeLoop:
		if entry.Loop != nil && entry.Loop.BodyResult != nil {
			c.collectRelationship(
				childPath(path, "body"),
				*entry.Loop.BodyResult,
			)
		}

	case declaration.TypeWorkflow:
		if entry.Workflow == nil {
			return
		}
		for _, node := range entry.Workflow.Nodes {
			if node.TargetResult == nil {
				continue
			}
			c.collectRelationship(
				childPath(
					path,
					"nodes",
					declaration.StableWorkflowNodeSegment(node.ID),
					"target",
				),
				*node.TargetResult,
			)
		}

	case declaration.TypeWorkspace:
		if entry.Workspace == nil {
			return
		}
		for index, relationship := range entry.Workspace.RootResults {
			c.collectRelationship(
				childPath(path, "roots", strconv.Itoa(index)),
				relationship,
			)
		}
	default:
	}
}

func childPath(parent string, segments ...string) string {
	output := parent
	for _, segment := range segments {
		if output != "" {
			output += "/"
		}
		output += segment
	}
	return output
}

func resolvedEntryName(
	entry *ResolvedEntry,
) (basespec.LogicalName, error) {
	if entry.Artifact != nil {
		return entry.Artifact.LogicalName, nil
	}
	if entry.Inline != nil {
		return basespec.LogicalName(entry.Inline.Header().Name), nil
	}
	return "", fmt.Errorf(
		"%w: resolved capability root has no declaration identity",
		basespec.ErrReferenceUnresolved,
	)
}
