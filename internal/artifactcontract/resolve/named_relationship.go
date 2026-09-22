package resolve

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
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

type FallbackCatalogRequest struct {
	RootID root.RootID
	Type   declaration.Type
	Scope  declaration.LookupScope
}

// FallbackCatalogProvider is optional. Exact-name fallback resolution does not
// require catalog support. This interface exists only for bounded management
// reference catalogs.
type FallbackCatalogProvider interface {
	FallbackProvider

	ListFallbackCatalog(
		ctx context.Context,
		request FallbackCatalogRequest,
	) ([]MappedTarget, error)
}
