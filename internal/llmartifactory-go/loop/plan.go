package loop

import (
	"bytes"
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	loopv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/loop/contract/v1"
)

type Service struct {
	resolver *composition.Resolver
}

type Plan struct {
	Loop          artifactModel.ArtifactRef  `json:"loop"`
	Name          spec.LogicalName           `json:"name"`
	Body          *Body                      `json:"body,omitempty"`
	MaxIterations int                        `json:"maxIterations,omitempty"`
	Until         *declaration.OutputMatch   `json:"until,omitempty"`
	Capabilities  composition.CapabilityPlan `json:"capabilities"`
	Complete      bool                       `json:"complete"`
}

type Body struct {
	Type   declaration.Type              `json:"type"`
	Name   spec.LogicalName              `json:"name"`
	Form   declaration.MemberForm        `json:"form"`
	Status composition.ResolutionStatus  `json:"status"`
	Target *composition.CapabilityTarget `json:"target,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewService(resolver *composition.Resolver) (*Service, error) {
	if resolver == nil {
		return nil, fmt.Errorf(
			"%w: Loop plan resolver is required",
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

	resolved, err := s.resolver.ResolveLoop(ctx, ref)
	if err != nil {
		return Plan{}, err
	}
	if resolved == nil || resolved.Type != declaration.TypeLoop ||
		resolved.Definition == nil {
		return Plan{}, fmt.Errorf(
			"%w: Artifact did not resolve as a Loop",
			spec.ErrReferenceUnresolved,
		)
	}
	loopRef, found := resolved.ArtifactRef()
	if !found {
		return Plan{}, fmt.Errorf(
			"%w: Loop plan root has no Artifact",
			spec.ErrReferenceUnresolved,
		)
	}

	document, err := loopv1.DecodeAdmittedLoopJSON(resolved.Definition.Body)
	if err != nil {
		return Plan{}, err
	}
	capabilities, err := composition.CapabilityPlanForResolvedEntry(resolved)
	if err != nil {
		return Plan{}, err
	}

	output := Plan{
		Loop:          loopRef,
		Name:          spec.LogicalName(document.Name),
		MaxIterations: document.MaxIterations,
		Capabilities:  capabilities,
		Complete:      capabilities.Complete,
	}
	if document.Until != nil {
		value := document.Until.Clone()
		output.Until = &value
	}
	if document.Body == nil {
		return output, nil
	}

	relationship, found := loopBodyRelationship(
		resolved.Relationships,
		*document.Body,
	)
	if !found {
		return Plan{}, fmt.Errorf(
			"%w: Loop body has no resolved relationship",
			spec.ErrInvalid,
		)
	}
	form, err := document.Body.MemberForm()
	if err != nil {
		return Plan{}, err
	}
	header := document.Body.Header()
	body := Body{
		Type:   header.Type,
		Name:   spec.LogicalName(header.Name),
		Form:   form,
		Status: relationship.Status,
	}
	if relationship.Issue != nil {
		body.Code = relationship.Issue.Code
		body.Message = relationship.Issue.Message
	}
	if relationship.Resolved != nil && relationship.Resolved.Target != nil {
		target := relationship.Resolved.Target.Clone()
		body.Target = &target
	}
	output.Body = &body
	return output, nil
}

func loopBodyRelationship(
	values []composition.ResolvedRelationship,
	member declaration.Entry,
) (composition.ResolvedRelationship, bool) {
	identity, err := declaration.MemberIdentityJSON(member)
	if err != nil {
		return composition.ResolvedRelationship{}, false
	}
	for _, value := range values {
		current, err := declaration.MemberIdentityJSON(value.Declared)
		if err == nil && bytes.Equal(current, identity) {
			return value, true
		}
	}
	return composition.ResolvedRelationship{}, false
}
