package collection

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

// ArtifactMembershipView is one direct external Collection member whose
// declared type and name match a selected Artifact. It remains visible when
// that relationship is unavailable or ambiguous.
type ArtifactMembershipView struct {
	Collection         artifact.ArtifactRef     `json:"collection"`
	CollectionName     basespec.LogicalName     `json:"collectionName"`
	CollectionRevision uint64                   `json:"collectionRevision"`
	MemberIndex        int                      `json:"memberIndex"`
	Member             MemberReference          `json:"member"`
	Status             resolve.ResolutionStatus `json:"status"`
	ResolvedArtifact   *artifact.ArtifactRef    `json:"resolvedArtifact,omitempty"`
	ResolvedToArtifact bool                     `json:"resolvedToArtifact"`
	Code               string                   `json:"code,omitempty"`
	Message            string                   `json:"message,omitempty"`
}

func (a *API) ListMembershipsForArtifact(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]ArtifactMembershipView, error) {
	if a == nil || a.resolver == nil {
		return nil, fmt.Errorf(
			"%w: Collection resolver is unavailable",
			basespec.ErrUnsupported,
		)
	}
	target, err := a.artifacts.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	targetType := declaration.Type(target.Kind)
	if err := targetType.Validate(); err != nil {
		return nil, err
	}
	if a.domain != nil && !a.domain.allows(targetType) {
		return nil, fmt.Errorf(
			"%w: Artifact type %q is not supported by the %s Collection domain",
			basespec.ErrUnsupported,
			targetType,
			a.domain.Name,
		)
	}

	targetTerminal := ref
	if terminal, resolveErr := a.resolver.ResolveDeclarationArtifact(
		ctx,
		ref,
	); resolveErr == nil {
		targetTerminal = terminal
	}

	collections, err := a.ListDomain(ctx, ref.RootID)
	if err != nil {
		return nil, err
	}
	output := make([]ArtifactMembershipView, 0)
	for _, collectionValue := range collections {
		graph, err := a.resolver.ResolveArtifact(
			ctx,
			collectionValue.Artifact.Ref(),
		)
		if err != nil {
			return nil, err
		}
		if graph.Root == nil ||
			graph.Root.Type != declaration.TypeCollection {
			return nil, fmt.Errorf(
				"%w: Collection %q did not resolve as a Collection",
				basespec.ErrReferenceUnresolved,
				collectionValue.Artifact.ID,
			)
		}

		for index, relationship := range graph.Root.MemberResults {
			header := relationship.Declared.Header()
			if header.Type != targetType ||
				header.Name != string(target.LogicalName) {
				continue
			}
			form, err := relationship.Declared.CompositionForm()
			if err != nil {
				return nil, err
			}
			if form != declaration.CompositionEntryReference {
				continue
			}
			member, err := memberReferenceFromEntry(
				relationship.Declared,
			)
			if err != nil {
				return nil, err
			}

			view := ArtifactMembershipView{
				Collection:         collectionValue.Artifact.Ref(),
				CollectionName:     collectionValue.Name,
				CollectionRevision: collectionValue.Artifact.Revision,
				MemberIndex:        index,
				Member:             member,
				Status:             relationship.Status,
			}
			if relationship.Issue != nil {
				view.Code = relationship.Issue.Code
				view.Message = relationship.Issue.Message
			}
			if relationship.Resolved != nil {
				if resolvedRef, found := relationship.Resolved.ArtifactRef(); found {
					terminal := resolvedRef
					if value, resolveErr := a.resolver.ResolveDeclarationArtifact(
						ctx,
						resolvedRef,
					); resolveErr == nil {
						terminal = value
					}
					view.ResolvedArtifact = &terminal
					view.ResolvedToArtifact = terminal == targetTerminal
				}
			}
			output = append(output, view)
		}
	}
	return output, nil
}
