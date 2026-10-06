package workflow

import (
	"context"
	"fmt"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

type Service struct {
	resolver *composition.Resolver
}

type Plan struct {
	Workflow     artifactModel.ArtifactRef  `json:"workflow"`
	Name         spec.LogicalName           `json:"name"`
	Start        []string                   `json:"start,omitempty"`
	Nodes        []Node                     `json:"nodes"`
	Edges        []Edge                     `json:"edges"`
	Capabilities composition.CapabilityPlan `json:"capabilities"`
	Complete     bool                       `json:"complete"`
}

type Node struct {
	ID     string                        `json:"id"`
	Join   workflowv1.Join               `json:"join,omitempty"`
	Type   declaration.Type              `json:"type"`
	Name   spec.LogicalName              `json:"name"`
	Form   declaration.MemberForm        `json:"form"`
	Status composition.ResolutionStatus  `json:"status"`
	Target *composition.CapabilityTarget `json:"target,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type Edge struct {
	From  string                   `json:"from"`
	To    string                   `json:"to"`
	Match *declaration.OutputMatch `json:"match,omitempty"`
}

func NewService(resolver *composition.Resolver) (*Service, error) {
	if resolver == nil {
		return nil, fmt.Errorf(
			"%w: Workflow plan resolver is required",
			spec.ErrInvalid,
		)
	}
	return &Service{resolver: resolver}, nil
}

func (s *Service) ResolvePlan(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (Plan, error) {
	if s == nil || s.resolver == nil {
		return Plan{}, spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return Plan{}, err
	}

	resolved, err := s.resolver.ResolveWorkflow(ctx, ref)
	if err != nil {
		return Plan{}, err
	}
	if resolved == nil || resolved.Type != declaration.TypeWorkflow ||
		resolved.Definition == nil {
		return Plan{}, fmt.Errorf(
			"%w: Artifact did not resolve as a Workflow",
			spec.ErrReferenceUnresolved,
		)
	}

	workflowRef, found := resolved.ArtifactRef()
	if !found {
		return Plan{}, fmt.Errorf(
			"%w: Workflow plan root has no Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	document, err := workflowv1.DecodeAdmittedWorkflowJSON(
		resolved.Definition.Body,
	)
	if err != nil {
		return Plan{}, err
	}
	capabilities, err := composition.CapabilityPlanForResolvedEntry(resolved)
	if err != nil {
		return Plan{}, err
	}

	nodes := append([]workflowv1.Node(nil), document.Nodes...)
	sort.Slice(nodes, func(left, right int) bool {
		return nodes[left].ID < nodes[right].ID
	})

	output := Plan{
		Workflow:     workflowRef,
		Name:         spec.LogicalName(document.Name),
		Start:        append([]string(nil), document.Start...),
		Nodes:        make([]Node, 0, len(nodes)),
		Edges:        make([]Edge, 0, len(document.Edges)),
		Capabilities: capabilities,
		Complete:     capabilities.Complete,
	}
	for _, value := range nodes {
		relationship, found, err := workflowNodeRelationship(
			resolved.Relationships,
			value.ID,
		)
		if err != nil {
			return Plan{}, err
		}
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: Workflow node %q has no resolved relationship",
				spec.ErrInvalid,
				value.ID,
			)
		}

		form, err := value.Member.MemberForm()
		if err != nil {
			return Plan{}, err
		}
		header := value.Member.Header()
		node := Node{
			ID:     value.ID,
			Join:   value.Join,
			Type:   header.Type,
			Name:   spec.LogicalName(header.Name),
			Form:   form,
			Status: relationship.Status,
		}
		if relationship.Issue != nil {
			node.Code = relationship.Issue.Code
			node.Message = relationship.Issue.Message
		}
		if relationship.Resolved != nil && relationship.Resolved.Target != nil {
			target := relationship.Resolved.Target.Clone()
			node.Target = &target
		}
		output.Nodes = append(output.Nodes, node)
	}
	for _, value := range document.Edges {
		edge := Edge{
			From: value.From,
			To:   value.To,
		}
		if value.Match != nil {
			match := value.Match.Clone()
			edge.Match = &match
		}
		output.Edges = append(output.Edges, edge)
	}
	return output, nil
}

func workflowNodeRelationship(
	values []composition.ResolvedRelationship,
	nodeID string,
) (composition.ResolvedRelationship, bool, error) {
	segment, err := workflowv1.NodeRelationshipSegment(nodeID)
	if err != nil {
		return composition.ResolvedRelationship{}, false, err
	}
	for _, value := range values {
		if len(value.Path) < 2 ||
			value.Path[0] != "nodes" ||
			value.Path[1] != segment {
			continue
		}
		return value, true, nil
	}
	return composition.ResolvedRelationship{}, false, nil
}
