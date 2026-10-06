package team

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	teamv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/team/contract/v1"
)

type Service struct {
	resolver *composition.Resolver
}

type Plan struct {
	Team         artifactModel.ArtifactRef  `json:"team"`
	Name         spec.LogicalName           `json:"name"`
	Members      []Member                   `json:"members"`
	Program      *Program                   `json:"program,omitempty"`
	Capabilities composition.CapabilityPlan `json:"capabilities"`
	Complete     bool                       `json:"complete"`
}

type Member struct {
	Type     declaration.Type             `json:"type"`
	Name     spec.LogicalName             `json:"name,omitempty"`
	Form     declaration.MemberForm       `json:"form"`
	Required bool                         `json:"required"`
	Status   composition.ResolutionStatus `json:"status"`

	Target          *composition.CapabilityTarget `json:"target,omitempty"`
	SelectorMatches []SelectorMatch               `json:"selectorMatches,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type SelectorMatch struct {
	Artifact artifactModel.ArtifactRef     `json:"artifact"`
	Status   composition.ResolutionStatus  `json:"status"`
	Target   *composition.CapabilityTarget `json:"target,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type Program struct {
	Type   declaration.Type              `json:"type"`
	Name   spec.LogicalName              `json:"name"`
	Status composition.ResolutionStatus  `json:"status"`
	Target *composition.CapabilityTarget `json:"target,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewService(resolver *composition.Resolver) (*Service, error) {
	if resolver == nil {
		return nil, fmt.Errorf(
			"%w: Team plan resolver is required",
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

	resolved, err := s.resolver.ResolveTeam(ctx, ref)
	if err != nil {
		return Plan{}, err
	}
	if resolved == nil || resolved.Type != declaration.TypeTeam ||
		resolved.Definition == nil {
		return Plan{}, fmt.Errorf(
			"%w: Artifact did not resolve as a Team",
			spec.ErrReferenceUnresolved,
		)
	}

	teamRef, found := resolved.ArtifactRef()
	if !found {
		return Plan{}, fmt.Errorf(
			"%w: Team plan root has no Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	document, err := teamv1.DecodeAdmittedTeamJSON(resolved.Definition.Body)
	if err != nil {
		return Plan{}, err
	}
	capabilities, err := composition.CapabilityPlanForResolvedEntry(resolved)
	if err != nil {
		return Plan{}, err
	}

	relationships, err := teamRelationshipsByIdentity(resolved.Relationships)
	if err != nil {
		return Plan{}, err
	}
	ordered, err := declaration.SortedMembers("Team members", document.Members)
	if err != nil {
		return Plan{}, err
	}

	output := Plan{
		Team:         teamRef,
		Name:         spec.LogicalName(document.Name),
		Members:      make([]Member, 0, len(ordered)),
		Capabilities: capabilities,
		Complete:     capabilities.Complete,
	}
	for _, declared := range ordered {
		relationship, found := relationships[teamRelationshipIdentity(declared)]
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: Team member has no resolved relationship",
				spec.ErrInvalid,
			)
		}
		member, err := projectMember(relationship)
		if err != nil {
			return Plan{}, err
		}
		output.Members = append(output.Members, member)
	}

	for _, declared := range []*declaration.Entry{
		document.Loop,
		document.Workflow,
	} {
		if declared == nil {
			continue
		}
		relationship, found := relationships[teamRelationshipIdentity(*declared)]
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: Team program member has no resolved relationship",
				spec.ErrInvalid,
			)
		}
		value, err := projectProgram(relationship)
		if err != nil {
			return Plan{}, err
		}
		output.Program = &value
	}
	return output, nil
}

func teamRelationshipsByIdentity(
	values []composition.ResolvedRelationship,
) (map[string]composition.ResolvedRelationship, error) {
	output := make(map[string]composition.ResolvedRelationship, len(values))
	for _, value := range values {
		key := teamRelationshipIdentity(value.Declared)
		if _, duplicate := output[key]; duplicate {
			return nil, fmt.Errorf(
				"%w: Team relationship identity is duplicated",
				spec.ErrIdentityConflict,
			)
		}
		output[key] = value
	}
	return output, nil
}

func teamRelationshipIdentity(entry declaration.Entry) string {
	raw, err := declaration.MemberIdentityJSON(entry)
	if err != nil {
		return ""
	}
	return string(raw)
}

func projectMember(
	relationship composition.ResolvedRelationship,
) (Member, error) {
	form, err := relationship.Declared.MemberForm()
	if err != nil {
		return Member{}, err
	}
	header := relationship.Declared.Header()
	output := Member{
		Type:     header.Type,
		Name:     spec.LogicalName(header.Name),
		Form:     form,
		Required: relationship.Required,
		Status:   relationship.Status,
	}
	if relationship.Issue != nil {
		output.Code = relationship.Issue.Code
		output.Message = relationship.Issue.Message
	}
	if relationship.Resolved != nil && relationship.Resolved.Target != nil {
		target := relationship.Resolved.Target.Clone()
		output.Target = &target
	}
	if relationship.Selector == nil {
		return output, nil
	}

	output.SelectorMatches = make(
		[]SelectorMatch,
		0,
		len(relationship.Selector.Matches),
	)
	for _, match := range relationship.Selector.Matches {
		value := SelectorMatch{
			Artifact: match.Artifact,
			Status:   match.Status,
		}
		if match.Issue != nil {
			value.Code = match.Issue.Code
			value.Message = match.Issue.Message
		}
		if match.Resolved != nil && match.Resolved.Target != nil {
			target := match.Resolved.Target.Clone()
			value.Target = &target
		}
		output.SelectorMatches = append(output.SelectorMatches, value)
	}
	return output, nil
}

func projectProgram(
	relationship composition.ResolvedRelationship,
) (Program, error) {
	header := relationship.Declared.Header()
	output := Program{
		Type:   header.Type,
		Name:   spec.LogicalName(header.Name),
		Status: relationship.Status,
	}
	if relationship.Issue != nil {
		output.Code = relationship.Issue.Code
		output.Message = relationship.Issue.Message
	}
	if relationship.Resolved != nil && relationship.Resolved.Target != nil {
		target := relationship.Resolved.Target.Clone()
		output.Target = &target
	}
	return output, nil
}
