package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

func toolCollectionPolicy() collection.DomainPolicy {
	return collection.DomainPolicy{
		Name:        "tool",
		ReadOnly:    true,
		PackageKind: toolDomain.ToolCollectionPackageKind,
		DocumentUse: documentTopology.DocumentUseToolCollection,
		AllowedMemberTypes: []declaration.Type{
			declaration.TypeTool,
		},
		AllowedMemberForms: []declaration.MemberForm{
			declaration.MemberNamed,
		},
		ValidateDocument: func(document pluginv1.PluginDocument) error {
			_, err := toolDomain.ValidateToolCollectionDocument(document)
			return err
		},
	}
}

func (a *API) ListToolCollections(
	ctx context.Context,
) ([]collection.CollectionView, error) {
	if err := a.ready(ctx); err != nil {
		return nil, err
	}

	values, err := a.collections.ListDomain(ctx, a.builtinRoot)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		if err := a.requireBuiltinArtifact(value.Artifact); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func (a *API) GetToolCollection(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (collection.CollectionView, error) {
	if err := a.ready(ctx); err != nil {
		return collection.CollectionView{}, err
	}
	if err := a.requireBuiltinRef(ref); err != nil {
		return collection.CollectionView{}, err
	}

	view, err := a.collections.Read(ctx, ref)
	if err != nil {
		return collection.CollectionView{}, err
	}
	if err := a.requireBuiltinArtifact(view.Artifact); err != nil {
		return collection.CollectionView{}, err
	}
	return view, nil
}

func (a *API) SetToolCollectionEnabled(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (collection.CollectionView, error) {
	if _, err := a.GetToolCollection(ctx, ref); err != nil {
		return collection.CollectionView{}, err
	}
	return a.collections.SetEnabled(ctx, ref, expectedRevision, enabled)
}

func (a *API) collectionForTool(
	ctx context.Context,
	name basespec.LogicalName,
) (collection.CollectionView, error) {
	values, err := a.ListToolCollections(ctx)
	if err != nil {
		return collection.CollectionView{}, err
	}

	matches := make([]collection.CollectionView, 0, 1)
	for _, value := range values {
		for _, member := range value.Members {
			if member.Type == declaration.TypeTool && member.Name == name {
				matches = append(matches, value)
				break
			}
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return collection.CollectionView{}, fmt.Errorf(
			"%w: Tool %q has no Tool Collection",
			basespec.ErrReferenceUnresolved,
			name,
		)
	default:
		return collection.CollectionView{}, fmt.Errorf(
			"%w: Tool %q belongs to %d Tool Collections",
			basespec.ErrIdentityConflict,
			name,
			len(matches),
		)
	}
}
