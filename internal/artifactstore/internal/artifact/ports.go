package artifactimpl

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

type Reader interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	ListByCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)

	ListSuppressions(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Suppression, error)
}

type Repository interface {
	Reader

	CreateAdopted(
		ctx context.Context,
		value artifact.Artifact,
		expectedCollectionRevision uint64,
		expectedCatalogRevision uint64,
	) error

	CreatePinned(
		ctx context.Context,
		value artifact.Artifact,
		expectedCollectionRevision uint64,
		expectedCatalogRevision uint64,
	) error

	Update(
		ctx context.Context,
		value artifact.Artifact,
		expectedRevision uint64,
	) error

	Unadopt(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		suppression *artifact.Suppression,
	) error

	Suppress(
		ctx context.Context,
		value artifact.Suppression,
		expectedCollectionRevision uint64,
	) error

	Unsuppress(
		ctx context.Context,
		ref collection.CollectionRef,
		binding artifact.SourceBinding,
		expectedRevision uint64,
	) error

	PurgeAndSuppress(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		suppression artifact.Suppression,
	) error

	Purge(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error
}

type Draft struct {
	ID      basespec.ArtifactID
	Name    string
	Enabled bool
	Data    json.RawMessage
}

type Policy interface {
	Derive(
		ctx context.Context,
		value collection.Collection,
		occurrence catalog.Occurrence,
		def providerapi.Definition,
	) (Draft, bool, []providerapi.Diagnostic, error)
}

type Reconciliation struct {
	Creates     []artifact.Artifact
	Updates     []SourceStateUpdate
	Diagnostics []providerapi.Diagnostic
}
