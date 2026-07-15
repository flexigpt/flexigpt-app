package spec

import (
	"context"
	"time"
)

// ArtifactMetadataRepository is the app-local data-access boundary. Business
// logic depends only on this interface, allowing SQLite to be replaced by a
// different metadata backend without changing service behavior.
type ArtifactMetadataRepository interface {
	Close() error

	CreateRoot(ctx context.Context, root ArtifactRoot) error
	GetRoot(ctx context.Context, rootID RootID, includeSoftDeleted bool) (ArtifactRoot, error)
	ListRoots(ctx context.Context, includeSoftDeleted bool) ([]ArtifactRoot, error)
	UpdateRoot(ctx context.Context, root ArtifactRoot, expectedModifiedAt time.Time) error

	CreateSource(ctx context.Context, source ArtifactSource) error
	GetSource(ctx context.Context, sourceID SourceID) (ArtifactSource, error)
	ListSources(ctx context.Context) ([]ArtifactSource, error)
	UpdateSource(ctx context.Context, source ArtifactSource, expectedModifiedAt time.Time) error
	DeleteSource(ctx context.Context, sourceID SourceID, expectedModifiedAt time.Time) error

	CreateRootSourceAttachment(ctx context.Context, attachment RootSourceAttachment) error
	GetRootSourceAttachment(
		ctx context.Context,
		rootID RootID,
		sourceID SourceID,
	) (RootSourceAttachment, error)
	ListRootSourceAttachments(ctx context.Context, rootID RootID) ([]RootSourceAttachment, error)
	UpdateRootSourceAttachment(
		ctx context.Context,
		attachment RootSourceAttachment,
		expectedModifiedAt time.Time,
	) error
	DeleteRootSourceAttachment(
		ctx context.Context,
		rootID RootID,
		sourceID SourceID,
		expectedModifiedAt time.Time,
	) error

	CreateCollection(ctx context.Context, collection ArtifactCollection) error
	GetCollection(
		ctx context.Context,
		collectionID CollectionID,
		includeSoftDeleted bool,
	) (ArtifactCollection, error)
	GetCollectionByRootSlug(
		ctx context.Context,
		rootID RootID,
		slug CollectionSlug,
		includeSoftDeleted bool,
	) (ArtifactCollection, error)
	ListCollections(ctx context.Context, rootID RootID, includeSoftDeleted bool) ([]ArtifactCollection, error)
	UpdateCollection(
		ctx context.Context,
		collection ArtifactCollection,
		expectedModifiedAt time.Time,
	) error
	CountRecordsInCollection(ctx context.Context, collectionID CollectionID) (int64, error)

	GetArtifactPackage(
		ctx context.Context,
		sourceID SourceID,
		manifestLocator SourceLocator,
	) (ArtifactPackage, error)
	ListArtifactPackagesForSource(ctx context.Context, sourceID SourceID) ([]ArtifactPackage, error)
	UpsertArtifactPackage(ctx context.Context, artifactPackage ArtifactPackage) error

	GetCatalogResource(ctx context.Context, key CatalogResourceKey) (CatalogResource, error)
	ListCatalogResourcesForSource(ctx context.Context, sourceID SourceID) ([]CatalogResource, error)
	ListPublishedCatalogResourcesForRoot(ctx context.Context, rootID RootID) ([]CatalogResource, error)
	ListCatalogResourceRevisions(
		ctx context.Context,
		key CatalogResourceKey,
	) ([]CatalogResourceRevision, error)
	PublishRootScan(
		ctx context.Context,
		publication RootScanPublication,
	) (RootCatalogGeneration, error)
	GetRootCatalogGeneration(ctx context.Context, rootID RootID) (RootCatalogGeneration, error)

	CreateRecord(ctx context.Context, record ArtifactRecord) error
	GetRecord(ctx context.Context, recordID RecordID) (ArtifactRecord, error)
	ListRecordsForRoot(ctx context.Context, rootID RootID) ([]ArtifactRecord, error)
	FindRecordBySource(
		ctx context.Context,
		rootID RootID,
		key CatalogResourceKey,
		kind ArtifactKind,
	) (ArtifactRecord, error)
	UpdateRecord(ctx context.Context, record ArtifactRecord, expectedModifiedAt time.Time) error
	DeleteRecord(ctx context.Context, recordID RecordID, expectedModifiedAt time.Time) error

	PublishRecordSynchronization(
		ctx context.Context,
		publication RecordSynchronizationPublication,
	) error
	PublishRecordTransfer(ctx context.Context, publication RecordTransferPublication) error

	ListTransferProvenance(ctx context.Context, recordID RecordID) ([]TransferProvenance, error)

	ReplaceDependencySnapshots(
		ctx context.Context,
		publication DependencySnapshotPublication,
	) error
	ListDependencySnapshots(
		ctx context.Context,
		recordID RecordID,
	) ([]ArtifactDependencySnapshot, error)
}

// PortableContentRepository is the boundary for shareable JSON/files. Its
// implementation must use approved MapStore APIs and never direct filesystem
// calls from Artifact Store business logic.
type PortableContentRepository interface {
	Close() error
	PutDefinition(ctx context.Context, file ArtifactDefinitionFile) (CanonicalDefinition, error)
	GetDefinition(ctx context.Context, digest Digest) (CanonicalDefinition, error)
	PutAsset(ctx context.Context, content []byte) (Digest, int64, error)
	GetAsset(ctx context.Context, digest Digest) ([]byte, error)
	PutPackageManifest(ctx context.Context, key string, manifest PortablePackageManifest) error
	GetPackageManifest(ctx context.Context, key string) (PortablePackageManifest, error)
}
