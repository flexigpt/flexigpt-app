package plugin

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

// ArtifactMembershipView is one direct external Collection member whose
// declared type and name match a selected Artifact. It remains visible when
// that relationship is unavailable or ambiguous.
type ArtifactMembershipView struct {
	Collection         artifactModel.ArtifactRef    `json:"collection"`
	CollectionName     spec.LogicalName             `json:"collectionName"`
	CollectionRevision uint64                       `json:"collectionRevision"`
	MemberIndex        int                          `json:"memberIndex"`
	Member             MemberReference              `json:"member"`
	Status             composition.ResolutionStatus `json:"status"`
	ResolvedArtifact   *artifactModel.ArtifactRef   `json:"resolvedArtifact,omitempty"`
	ResolvedToArtifact bool                         `json:"resolvedToArtifact"`
	Code               string                       `json:"code,omitempty"`
	Message            string                       `json:"message,omitempty"`
}

func (a *API) ListMembershipsForArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) ([]ArtifactMembershipView, error) {
	if a == nil || a.resolver == nil {
		return nil, fmt.Errorf(
			"%w: Collection resolver is unavailable",
			spec.ErrUnsupported,
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
			spec.ErrUnsupported,
			targetType,
			a.domain.Name,
		)
	}

	targetTerminal := ref
	if terminal, resolveErr := a.resolver.ResolveTerminalArtifact(
		ctx,
		ref,
	); resolveErr == nil {
		targetTerminal = terminal
	}

	plugins, err := a.ListDomain(ctx, ListRequest{
		RootID: ref.RootID,
	})
	if err != nil {
		return nil, err
	}
	output := make([]ArtifactMembershipView, 0)
	for _, collectionValue := range plugins {
		plugin, err := a.resolver.ResolvePluginMembers(
			ctx,
			collectionValue.Ref,
		)
		if err != nil {
			return nil, err
		}
		if plugin == nil || plugin.Type != declaration.TypePlugin {
			return nil, fmt.Errorf(
				"%w: Plugin %q did not resolve as a Plugin",
				spec.ErrReferenceUnresolved,
				collectionValue.Ref.ArtifactID,
			)
		}

		for index, relationship := range plugin.MemberResults {
			header := relationship.Declared.Header()
			if header.Type != targetType ||
				header.Name != string(target.LogicalName) {
				continue
			}
			form, err := relationship.Declared.MemberForm()
			if err != nil {
				return nil, err
			}
			if form != declaration.MemberNamed {
				continue
			}
			member, err := memberReferenceFromEntry(
				relationship.Declared,
			)
			if err != nil {
				return nil, err
			}

			view := ArtifactMembershipView{
				Collection:         collectionValue.Ref,
				CollectionName:     collectionValue.Name,
				CollectionRevision: collectionValue.Revision,
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
					if value, resolveErr := a.resolver.ResolveTerminalArtifact(
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
