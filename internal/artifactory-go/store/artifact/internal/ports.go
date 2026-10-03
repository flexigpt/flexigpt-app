package internal

import (
	"context"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Reader interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (artifactModel.Artifact, error)

	ListByRoot(
		ctx context.Context,
		rootID rootModel.RootID,
	) ([]artifactModel.Artifact, error)

	ListBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) ([]artifactModel.Artifact, error)

	FindByIdentity(
		ctx context.Context,
		rootID rootModel.RootID,
		kind artifactModel.ArtifactKind,
		logicalName spec.LogicalName,
	) ([]artifactModel.Artifact, error)

	FindByOrigin(
		ctx context.Context,
		rootID rootModel.RootID,
		binding artifactModel.SourceBinding,
		kind artifactModel.ArtifactKind,
	) (artifactModel.Artifact, error)
}

// CatalogReader returns committed lightweight catalog rows. It deliberately
// does not return Artifact.Data, diagnostics, or Definition.Body.
type CatalogReader interface {
	ListCatalogByRoot(
		ctx context.Context,
		rootID rootModel.RootID,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)

	ListCatalogBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)

	FindCatalogByIdentity(
		ctx context.Context,
		rootID rootModel.RootID,
		kind artifactModel.ArtifactKind,
		logicalName spec.LogicalName,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)

	GetMany(
		ctx context.Context,
		refs []artifactModel.ArtifactRef,
	) ([]artifactModel.Artifact, error)
}

type Repository interface {
	Reader
	CatalogReader

	Create(
		ctx context.Context,
		value artifactModel.Artifact,
	) error

	UpdateLocal(
		ctx context.Context,
		value artifactModel.Artifact,
		expectedRevision uint64,
	) error

	UpdateSourceState(
		ctx context.Context,
		value SourceStateUpdate,
	) error

	Purge(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedRevision uint64,
	) error
}

type DefinitionReader interface {
	GetDefinitions(
		ctx context.Context,
		keys []definitionModel.Key,
	) ([]definitionModel.Definition, error)

	GetDefinition(
		ctx context.Context,
		rootID rootModel.RootID,
		digest cryptoutil.Digest,
	) (definitionModel.Definition, error)
}
