package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/pluginv1"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/catalog"
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
) ([]collection.ListItem, error) {
	if err := a.ready(ctx); err != nil {
		return nil, err
	}
	return a.collections.ListDomain(ctx, collection.ListRequest{
		RootID: a.builtinRoot,
	})
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
	name model.LogicalName,
) (collection.CollectionView, error) {
	collectionName, found := a.collectionByTool[name]
	if !found {
		return collection.CollectionView{}, fmt.Errorf(
			"%w: Tool %q has no generated Tool Collection",
			model.ErrReferenceUnresolved,
			name,
		)
	}

	entries, err := a.artifacts.FindByIdentity(
		ctx,
		a.builtinRoot,
		artifact.ArtifactKind(pluginv1.PluginType),
		collectionName,
		catalog.ListOptions{},
	)
	if err != nil {
		return collection.CollectionView{}, err
	}

	matches := make([]artifact.ArtifactRef, 0, 1)
	for _, entry := range entries {
		if entry.State != artifact.StateAvailable ||
			entry.Binding.SourceID != a.builtinSource ||
			entry.Binding.SubresourceLocator != "" {
			continue
		}
		matches = append(matches, entry.Ref())
	}
	switch len(matches) {
	case 1:
		return a.GetToolCollection(ctx, matches[0])
	case 0:
		return collection.CollectionView{}, fmt.Errorf(
			"%w: Tool %q has no Tool Collection",
			model.ErrReferenceUnresolved,
			name,
		)
	default:
		return collection.CollectionView{}, fmt.Errorf(
			"%w: Tool %q belongs to %d Tool Collections",
			model.ErrIdentityConflict,
			name,
			len(matches),
		)
	}
}
