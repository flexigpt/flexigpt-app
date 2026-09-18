package consumerapi

import (
	"context"
	"fmt"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

func (a *API) CreateAgentCollection(
	ctx context.Context,
	request CreateAgentCollectionRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return collection.CollectionView{}, err
	}
	if request.SourceID != "" {
		if err := request.SourceID.Validate(); err != nil {
			return collection.CollectionView{}, err
		}
	}
	if err := request.Name.Validate(); err != nil {
		return collection.CollectionView{}, err
	}
	if err := basespec.ValidateRequiredText(
		"Agent Collection display name",
		request.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return collection.CollectionView{}, err
	}
	if err := basespec.ValidateOptionalText(
		"Agent Collection description",
		request.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return collection.CollectionView{}, err
	}

	created, err := a.collections.Create(
		ctx,
		collection.CreateRequest{
			RootID:      request.RootID,
			SourceID:    request.SourceID,
			Name:        request.Name,
			DisplayName: request.DisplayName,
			Description: request.Description,
		},
	)
	if err != nil {
		return collection.CollectionView{}, err
	}
	return a.GetAgentCollection(ctx, created.Artifact.Ref())
}

func (a *API) EnsureAgentBaselineCollection(
	ctx context.Context,
	rootID root.RootID,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}

	value, err := a.collections.EnsureBaseline(ctx, rootID)
	if err != nil {
		return collection.CollectionView{}, err
	}
	return a.GetAgentCollection(ctx, value.Artifact.Ref())
}

func (a *API) GetAgentCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return a.collections.Read(ctx, ref)
}

func (a *API) ListAgentCollections(
	ctx context.Context,
	rootID root.RootID,
) ([]collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return nil, basespec.ErrClosed
	}
	return a.collections.ListDomain(ctx, rootID)
}

func (a *API) UpdateAgentCollection(
	ctx context.Context,
	request UpdateAgentCollectionRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	if request.DisplayName != "" {
		if err := basespec.ValidateRequiredText(
			"Agent Collection display name",
			request.DisplayName,
			basespec.MaxDisplayNameBytes,
		); err != nil {
			return collection.CollectionView{}, err
		}
	}
	if err := basespec.ValidateOptionalText(
		"Agent Collection description",
		request.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return collection.CollectionView{}, err
	}

	updated, err := a.collections.Update(
		ctx,
		collection.UpdateRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedRevision,
			DisplayName:      request.DisplayName,
			Description:      request.Description,
		},
	)
	if err != nil {
		return collection.CollectionView{}, err
	}
	return a.GetAgentCollection(ctx, updated.Artifact.Ref())
}

func (a *API) SetAgentCollectionEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}

	updated, err := a.collections.SetEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
	if err != nil {
		return collection.CollectionView{}, err
	}
	return a.GetAgentCollection(ctx, updated.Artifact.Ref())
}

func (a *API) DeleteAgentCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
) error {
	if a == nil || a.collections == nil {
		return basespec.ErrClosed
	}
	return a.collections.Delete(
		ctx,
		collection.DeleteRequest{
			Collection:       ref,
			ExpectedRevision: expectedRevision,
		},
	)
}

func (a *API) AddAgentCollectionEntry(
	ctx context.Context,
	request AddAgentCollectionEntryRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}

	value, err := a.collections.AddEntry(
		ctx,
		collection.AddEntryRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedRevision,
			Entry:            request.Entry,
		},
	)
	if err != nil {
		return collection.CollectionView{}, err
	}
	return a.GetAgentCollection(ctx, value.Artifact.Ref())
}

func (a *API) EnsureAgentCollectionEntry(
	ctx context.Context,
	request AddAgentCollectionEntryRequest,
) (collection.MemberMutationResult, error) {
	if a == nil || a.collections == nil {
		return collection.MemberMutationResult{}, basespec.ErrClosed
	}

	result, err := a.collections.EnsureEntry(
		ctx,
		collection.AddEntryRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedRevision,
			Entry:            request.Entry,
		},
	)
	if err != nil {
		return collection.MemberMutationResult{}, err
	}

	view, err := a.GetAgentCollection(ctx, result.Collection.Artifact.Ref())
	if err != nil {
		return collection.MemberMutationResult{}, err
	}
	result.Collection = view
	return result, nil
}

func (a *API) AttachAgentToCollection(
	ctx context.Context,
	request AttachAgentToCollectionRequest,
) (collection.MemberMutationResult, error) {
	if a == nil || a.collections == nil {
		return collection.MemberMutationResult{}, basespec.ErrClosed
	}
	if request.ExpectedRevision == 0 {
		return collection.MemberMutationResult{}, fmt.Errorf(
			"%w: expected Agent Collection revision is required",
			basespec.ErrInvalid,
		)
	}
	if err := request.Collection.Validate(); err != nil {
		return collection.MemberMutationResult{}, err
	}
	if err := request.Agent.Validate(); err != nil {
		return collection.MemberMutationResult{}, err
	}

	collectionValue, err := a.collections.Get(ctx, request.Collection)
	if err != nil {
		return collection.MemberMutationResult{}, err
	}
	if collectionValue.Artifact.Revision != request.ExpectedRevision {
		return collection.MemberMutationResult{}, basespec.ErrConflict
	}

	target, err := a.GetAgent(ctx, request.Agent)
	if err != nil {
		return collection.MemberMutationResult{}, err
	}

	member := collection.MemberReference{
		Type: declaration.TypeAgent,
		Name: target.LogicalName,
	}

	switch target.RootID {
	case collectionValue.Artifact.RootID:
		if target.Binding.SourceID != collectionValue.Artifact.Binding.SourceID {
			break
		}
		if target.Binding.SubresourceLocator != "" {
			return collection.MemberMutationResult{}, fmt.Errorf(
				"%w: a contained Agent cannot be attached as an exact managed Collection member",
				basespec.ErrUnsupported,
			)
		}

		member, err = a.collections.MemberForCollectionSource(
			ctx,
			request.Collection,
			declaration.TypeAgent,
			target.LogicalName,
			target.Binding.Locator,
		)
		if err != nil {
			return collection.MemberMutationResult{}, err
		}

	case builtin.BuiltinRootID:
		member.Scope = declaration.LookupScopeBuiltin

	default:
		return collection.MemberMutationResult{}, fmt.Errorf(
			"%w: Agent %q belongs to unsupported Root %q",
			basespec.ErrUnsupported,
			target.ID,
			target.RootID,
		)
	}

	result, err := a.collections.EnsureMember(
		ctx,
		collection.AddMemberRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedRevision,
			Member:           member,
		},
	)
	if err != nil {
		return collection.MemberMutationResult{}, err
	}

	view, err := a.GetAgentCollection(ctx, result.Collection.Artifact.Ref())
	if err != nil {
		return collection.MemberMutationResult{}, err
	}
	result.Collection = view
	return result, nil
}

func (a *API) DetachAgentFromCollection(
	ctx context.Context,
	request DetachAgentFromCollectionRequest,
) (collection.CollectionView, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}

	value, err := a.collections.RemoveMember(
		ctx,
		collection.RemoveMemberRequest{
			Collection:       request.Collection,
			ExpectedRevision: request.ExpectedRevision,
			Index:            request.Index,
		},
	)
	if err != nil {
		return collection.CollectionView{}, err
	}
	return a.GetAgentCollection(ctx, value.Artifact.Ref())
}

func (a *API) ResolveAgentCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	if a == nil || a.collections == nil {
		return collection.CollectionCapabilityPlan{}, basespec.ErrClosed
	}
	return a.collections.ResolveCapabilities(ctx, ref)
}

func (a *API) ListAgentCollectionMembers(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionCapabilityPlan, error) {
	return a.ResolveAgentCollection(ctx, ref)
}

func (a *API) ListDirectAgentMemberships(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]collection.ArtifactMembershipView, error) {
	if a == nil || a.collections == nil {
		return nil, basespec.ErrClosed
	}
	if _, err := a.GetAgent(ctx, ref); err != nil {
		return nil, err
	}
	return a.collections.ListMembershipsForArtifact(ctx, ref)
}

func (a *API) IsManagedAgentCollection(
	value collection.CollectionView,
) bool {
	if value.Artifact.Binding.SubresourceLocator != "" {
		return false
	}
	if value.Artifact.Kind != artifact.ArtifactKind(
		pluginv1.PluginType,
	) {
		return false
	}
	return value.Baseline ||
		value.Editable &&
			value.Artifact.LogicalName != "" &&
			value.Artifact.RootID != builtin.BuiltinRootID &&
			value.Artifact.Binding.Locator != ""
}

func (a *API) IsAgentArtifact(
	value artifact.Artifact,
) bool {
	return agentDomain.IsAgentKind(value.Kind)
}
