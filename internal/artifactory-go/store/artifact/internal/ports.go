package internal

import (
	"context"

	catalog "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definition "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Reader interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	ListByRoot(
		ctx context.Context,
		rootID root.RootID,
	) ([]artifact.Artifact, error)

	ListBySource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) ([]artifact.Artifact, error)

	FindByIdentity(
		ctx context.Context,
		rootID root.RootID,
		kind artifact.ArtifactKind,
		logicalName spec.LogicalName,
	) ([]artifact.Artifact, error)

	FindByOrigin(
		ctx context.Context,
		rootID root.RootID,
		binding artifact.SourceBinding,
		kind artifact.ArtifactKind,
	) (artifact.Artifact, error)
}

// CatalogReader returns committed lightweight catalog rows. It deliberately
// does not return Artifact.Data, diagnostics, or Definition.Body.
type CatalogReader interface {
	ListCatalogByRoot(
		ctx context.Context,
		rootID root.RootID,
		options catalog.ListOptions,
	) ([]catalog.Entry, error)

	ListCatalogBySource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		options catalog.ListOptions,
	) ([]catalog.Entry, error)

	FindCatalogByIdentity(
		ctx context.Context,
		rootID root.RootID,
		kind artifact.ArtifactKind,
		logicalName spec.LogicalName,
		options catalog.ListOptions,
	) ([]catalog.Entry, error)

	GetMany(
		ctx context.Context,
		refs []artifact.ArtifactRef,
	) ([]artifact.Artifact, error)
}

type Repository interface {
	Reader
	CatalogReader

	Create(
		ctx context.Context,
		value artifact.Artifact,
	) error

	UpdateLocal(
		ctx context.Context,
		value artifact.Artifact,
		expectedRevision uint64,
	) error

	UpdateSourceState(
		ctx context.Context,
		value SourceStateUpdate,
	) error

	Purge(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error
}

type DefinitionReader interface {
	GetDefinitions(
		ctx context.Context,
		keys []definition.Key,
	) ([]definition.Definition, error)

	GetDefinition(
		ctx context.Context,
		rootID root.RootID,
		digest cryptoutil.Digest,
	) (definition.Definition, error)
}
