package resolve

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

type NamedRelationshipRequest struct {
	RootID root.RootID
	Member declaration.Entry
}

type NamedRelationshipTarget struct {
	Type declaration.Type

	Artifact *artifact.ArtifactRef
	Mapped   *MappedTarget
}

// NamedRelationshipInspection preserves ordinary relationship-resolution
// status without requiring callers to treat missing or ambiguous targets as
// operation failures.
type NamedRelationshipInspection struct {
	Type   declaration.Type
	Status ResolutionStatus

	Artifact *artifact.ArtifactRef
	Mapped   *MappedTarget
	Issue    *ResolutionIssue
}

// InspectNamedRelationship resolves one named external declaration member and
// projects normal relationship failures as unavailable or ambiguous results.
// Infrastructure failures remain returned errors.
func (r *Resolver) InspectNamedRelationship(
	ctx context.Context,
	request NamedRelationshipRequest,
) (NamedRelationshipInspection, error) {
	if r == nil || r.artifacts == nil {
		return NamedRelationshipInspection{}, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return NamedRelationshipInspection{}, err
	}
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
			basespec.ErrInvalid,
		)
	}

	header := request.Member.Header()
	if header.Locator != nil {
		return NamedRelationshipInspection{}, fmt.Errorf(
			"%w: named relationship inspection does not accept a locator",
			basespec.ErrInvalid,
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
		basespec.LogicalName(header.Name),
		expectedVersion,
		relationship.Scope,
		nil,
		0,
	)
	if err != nil {
		status, issue, observed := namedRelationshipInspectionFailure(err)
		if !observed {
			return NamedRelationshipInspection{}, err
		}
		output.Status = status
		output.Issue = &issue
		return output, nil
	}
	if value == nil {
		return NamedRelationshipInspection{}, fmt.Errorf(
			"%w: named relationship has no target",
			basespec.ErrReferenceUnresolved,
		)
	}

	output.Status = ResolutionAvailable
	if ref, found := value.ArtifactRef(); found {
		copyRef := ref
		output.Artifact = &copyRef
	}
	output.Mapped = cloneMappedTarget(value.Mapped)
	if output.Artifact == nil && output.Mapped == nil {
		return NamedRelationshipInspection{}, fmt.Errorf(
			"%w: named relationship resolved without a target",
			basespec.ErrReferenceUnresolved,
		)
	}
	return output, nil
}

func namedRelationshipInspectionFailure(
	err error,
) (ResolutionStatus, ResolutionIssue, bool) {
	status, issue, partial := resolutionFailure(err)
	if partial {
		return status, issue, true
	}
	if errors.Is(err, basespec.ErrInvalid) ||
		errors.Is(err, basespec.ErrDigestMismatch) {
		return ResolutionUnavailable, ResolutionIssue{
			Code:    "artifact.reference-invalid",
			Message: diagnostic.BoundedMessage(err.Error()),
		}, true
	}
	return "", ResolutionIssue{}, false
}

// ResolveNamedRelationship preflights one named external declaration member
// through the normal shared resolver lookup path. It intentionally does not
// expose generic arbitrary Artifact lookup and does not support located,
// contained, or selector relationships.
func (r *Resolver) ResolveNamedRelationship(
	ctx context.Context,
	request NamedRelationshipRequest,
) (NamedRelationshipTarget, error) {
	if r == nil || r.artifacts == nil {
		return NamedRelationshipTarget{}, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return NamedRelationshipTarget{}, err
	}
	if err := request.RootID.Validate(); err != nil {
		return NamedRelationshipTarget{}, err
	}

	form, err := request.Member.MemberForm()
	if err != nil {
		return NamedRelationshipTarget{}, err
	}
	if form != declaration.MemberNamed {
		return NamedRelationshipTarget{}, fmt.Errorf(
			"%w: named relationship preflight requires a named member",
			basespec.ErrInvalid,
		)
	}

	header := request.Member.Header()
	if header.Locator != nil {
		return NamedRelationshipTarget{}, fmt.Errorf(
			"%w: named relationship preflight does not accept a locator",
			basespec.ErrInvalid,
		)
	}

	relationship, err := request.Member.Relationship()
	if err != nil {
		return NamedRelationshipTarget{}, err
	}
	expectedVersion, err := memberTextLogicalVersion(request.Member)
	if err != nil {
		return NamedRelationshipTarget{}, err
	}

	state := newResolutionState()
	value, err := r.resolveNamedMember(
		ctx,
		&state,
		request.RootID,
		header.Type,
		basespec.LogicalName(header.Name),
		expectedVersion,
		relationship.Scope,
		nil,
		0,
	)
	if err != nil {
		return NamedRelationshipTarget{}, err
	}
	if value == nil {
		return NamedRelationshipTarget{}, fmt.Errorf(
			"%w: named relationship has no target",
			basespec.ErrReferenceUnresolved,
		)
	}

	output := NamedRelationshipTarget{
		Type: header.Type,
	}
	if ref, found := value.ArtifactRef(); found {
		copyRef := ref
		output.Artifact = &copyRef
	}
	output.Mapped = cloneMappedTarget(value.Mapped)
	if output.Artifact == nil && output.Mapped == nil {
		return NamedRelationshipTarget{}, fmt.Errorf(
			"%w: named relationship resolved without a target",
			basespec.ErrReferenceUnresolved,
		)
	}
	return output, nil
}
