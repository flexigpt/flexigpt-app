package composition

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type NamedRelationshipRequest struct {
	RootID rootModel.RootID
	Member declaration.Entry
}

// NamedRelationshipInspection preserves normal unavailable and ambiguous
// outcomes as data. Infrastructure failures still return an error.
type NamedRelationshipInspection struct {
	Type   declaration.Type
	Status ResolutionStatus
	Target *CapabilityTarget
	Issue  *ResolutionIssue
}

// InspectNamedRelationship resolves exactly one named external relationship.
// It deliberately does not accept selector, contained, or located members.
func (r *Resolver) InspectNamedRelationship(
	ctx context.Context,
	request NamedRelationshipRequest,
) (NamedRelationshipInspection, error) {
	if err := request.RootID.Validate(); err != nil {
		return NamedRelationshipInspection{}, err
	}

	form, err := request.Member.MemberForm()
	if err != nil {
		return NamedRelationshipInspection{}, err
	}
	if form != declaration.MemberNamed {
		return NamedRelationshipInspection{}, fmt.Errorf(
			"%w: named relationship inspection requires a named member",
			spec.ErrInvalid,
		)
	}

	header := request.Member.Header()
	if header.Locator != nil {
		return NamedRelationshipInspection{}, fmt.Errorf(
			"%w: named relationship inspection does not accept a locator",
			spec.ErrInvalid,
		)
	}

	relationship, err := request.Member.Relationship()
	if err != nil {
		return NamedRelationshipInspection{}, err
	}
	expectedVersion, err := memberTextLogicalVersion(request.Member)
	if err != nil {
		return NamedRelationshipInspection{}, err
	}

	output := NamedRelationshipInspection{
		Type: header.Type,
	}
	state := newResolutionState()
	value, err := r.resolveNamedMember(
		ctx,
		&state,
		request.RootID,
		header.Type,
		spec.LogicalName(header.Name),
		expectedVersion,
		relationship.Scope,
		nil,
		0,
	)
	if err != nil {
		status, issue, observed := resolutionFailure(err)
		if !observed {
			return NamedRelationshipInspection{}, err
		}
		output.Status = status
		output.Issue = &issue
		return output, nil
	}
	if value == nil || value.Target == nil {
		return NamedRelationshipInspection{}, fmt.Errorf(
			"%w: named relationship resolved without a target",
			spec.ErrReferenceUnresolved,
		)
	}

	output.Status = ResolutionAvailable
	output.Target = pointerTarget(*value.Target)
	return output, nil
}

// ResolveNamedRelationship is the strict counterpart of
// InspectNamedRelationship. Normal unavailable and ambiguous relationship
// outcomes are returned as errors.
func (r *Resolver) ResolveNamedRelationship(
	ctx context.Context,
	request NamedRelationshipRequest,
) (CapabilityTarget, error) {
	inspection, err := r.InspectNamedRelationship(ctx, request)
	if err != nil {
		return CapabilityTarget{}, err
	}
	if inspection.Status != ResolutionAvailable ||
		inspection.Target == nil {
		message := "named relationship is unavailable"
		if inspection.Issue != nil && inspection.Issue.Message != "" {
			message = inspection.Issue.Message
		}
		return CapabilityTarget{}, fmt.Errorf(
			"%w: %s",
			spec.ErrReferenceUnresolved,
			message,
		)
	}
	return inspection.Target.Clone(), nil
}
