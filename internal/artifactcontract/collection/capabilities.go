package collection

import (
	"context"
	"fmt"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

type CollectionCapabilityOccurrence struct {
	Path     string                   `json:"path"`
	Type     declaration.Type         `json:"type"`
	Name     basespec.LogicalName     `json:"name"`
	Status   resolve.ResolutionStatus `json:"status"`
	Artifact *artifact.ArtifactRef    `json:"artifact,omitempty"`
	Code     string                   `json:"code,omitempty"`
	Message  string                   `json:"message,omitempty"`
}

type CollectionCapabilityPlan struct {
	Collection  CollectionView                   `json:"collection"`
	Occurrences []CollectionCapabilityOccurrence `json:"occurrences"`
}

func (a *API) ResolveCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CollectionCapabilityPlan, error) {
	if a == nil || a.resolver == nil {
		return CollectionCapabilityPlan{}, fmt.Errorf(
			"%w: Collection resolver is unavailable",
			basespec.ErrUnsupported,
		)
	}

	graph, err := a.resolver.ResolveArtifact(ctx, ref)
	if err != nil {
		return CollectionCapabilityPlan{}, err
	}
	if graph.Root == nil || graph.Root.Type != declaration.TypeCollection {
		return CollectionCapabilityPlan{}, fmt.Errorf(
			"%w: Artifact %q is not a Collection",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}
	terminal, found := graph.Root.ArtifactRef()
	if !found {
		return CollectionCapabilityPlan{}, fmt.Errorf(
			"%w: Collection has no source-backed Artifact",
			basespec.ErrReferenceUnresolved,
		)
	}
	view, err := a.Read(ctx, terminal)
	if err != nil {
		return CollectionCapabilityPlan{}, err
	}

	plan := CollectionCapabilityPlan{
		Collection:  view,
		Occurrences: make([]CollectionCapabilityOccurrence, 0),
	}
	collector := collectionCapabilityCollector{plan: &plan}
	for index, relationship := range graph.Root.MemberResults {
		collector.collectRelationship(
			"members/"+strconv.Itoa(index),
			relationship,
		)
	}
	return plan, nil
}

type collectionCapabilityCollector struct {
	plan *CollectionCapabilityPlan
}

func (c collectionCapabilityCollector) collectRelationship(
	path string,
	relationship resolve.ResolvedRelationship,
) {
	header := relationship.Declared.Header()
	occurrence := CollectionCapabilityOccurrence{
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

func (c collectionCapabilityCollector) collectEntry(
	path string,
	entry *resolve.ResolvedEntry,
) {
	if entry == nil {
		return
	}

	switch entry.Type {
	case declaration.TypeCollection, declaration.TypeAgent, declaration.TypeTeam:
		for index, relationship := range entry.MemberResults {
			c.collectRelationship(
				path+"/members/"+strconv.Itoa(index),
				relationship,
			)
		}
		if entry.ProgramResult != nil {
			c.collectRelationship(path+"/program", *entry.ProgramResult)
		}

	case declaration.TypeSkill:
		for index, relationship := range entry.AllowedToolResults {
			c.collectRelationship(
				path+"/allowedTools/"+strconv.Itoa(index),
				relationship,
			)
		}

	case declaration.TypeLoop:
		if entry.Loop != nil && entry.Loop.BodyResult != nil {
			c.collectRelationship(path+"/body", *entry.Loop.BodyResult)
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
				path+"/nodes/"+node.ID+"/target",
				*node.TargetResult,
			)
		}

	case declaration.TypeWorkspace:
		if entry.Workspace == nil {
			return
		}
		for index, relationship := range entry.Workspace.RootResults {
			c.collectRelationship(
				path+"/roots/"+strconv.Itoa(index),
				relationship,
			)
		}
	default:
	}
}
