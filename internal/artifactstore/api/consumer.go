package api

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/managedartifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

// ConsumerAPI is the complete Artifact Store capability granted to
// application-level consumers.
//
// It exposes Store-owned commands and immutable Store results. It excludes
// repositories, source snapshots, source configuration, discovery plans,
// reconciliation, provider registries, protected-topology reset, and Store
// composition.
type ConsumerAPI interface {
	ResourceResolver

	IsProtectedRoot(
		rootID basespec.RootID,
	) bool

	RequirePrivilegedInstaller(
		ctx context.Context,
	) error

	CanonicalizeExpected(
		ctx context.Context,
		expected providerapi.SchemaKey,
		raw []byte,
	) (providerapi.ParsedDocument, error)

	CreateSource(
		ctx context.Context,
		rootID basespec.RootID,
		draft source.Draft,
	) (source.Summary, error)

	CreateSourceWithStatus(
		ctx context.Context,
		rootID basespec.RootID,
		draft source.Draft,
	) (source.Summary, bool, error)

	DiscardSource(
		ctx context.Context,
		rootID basespec.RootID,
		sourceID basespec.SourceID,
		expectedRevision uint64,
	) error

	GetSource(
		ctx context.Context,
		rootID basespec.RootID,
		sourceID basespec.SourceID,
	) (source.Summary, error)

	CreateCollection(
		ctx context.Context,
		rootID basespec.RootID,
		draft collection.Draft,
		attachments []collection.AttachmentDraft,
	) (collection.Collection, []collection.Attachment, error)

	GetCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)

	GetRetiredCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)

	ListCollections(
		ctx context.Context,
		rootID basespec.RootID,
	) ([]collection.Collection, error)

	UpdateCollection(
		ctx context.Context,
		ref collection.CollectionRef,
		update collection.Update,
	) (collection.Collection, error)

	RetireCollection(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) (collection.Collection, error)

	PurgeCollection(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) error

	AttachCollectionSource(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedCollectionRevision uint64,
		draft collection.AttachmentDraft,
	) (collection.Collection, collection.Attachment, error)

	GetCollectionAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
	) (collection.Attachment, error)

	ListCollectionAttachments(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]collection.Attachment, error)

	UpdateCollectionAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
		update collection.AttachmentUpdate,
	) (collection.Collection, collection.Attachment, error)

	DetachCollectionSource(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
	) (collection.Collection, error)

	ReplaceCollectionAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		replacement collection.AttachmentReplacement,
	) (collection.Collection, collection.Attachment, error)

	GetArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	ListCollectionArtifacts(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)

	AdoptArtifact(
		ctx context.Context,
		request artifact.AdoptRequest,
	) (artifact.Artifact, error)

	PinArtifact(
		ctx context.Context,
		request artifact.PinRequest,
	) (artifact.Artifact, error)

	SetArtifactEnabled(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		enabled bool,
	) (artifact.Artifact, error)

	SetArtifactName(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		name string,
	) (artifact.Artifact, error)

	UpdateArtifactData(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		data json.RawMessage,
	) (artifact.Artifact, error)

	UnadoptArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		suppress bool,
	) error

	PurgeArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error

	PurgeAndSuppressArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error

	ListCollectionSuppressions(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Suppression, error)

	SuppressBinding(
		ctx context.Context,
		request artifact.SuppressRequest,
	) (artifact.Suppression, error)

	UnsuppressBinding(
		ctx context.Context,
		ref collection.CollectionRef,
		binding artifact.SourceBinding,
		expectedRevision uint64,
	) error

	RefreshCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) (refresh.Result, error)

	CurrentCollectionCatalog(
		ctx context.Context,
		ref collection.CollectionRef,
	) (catalog.Snapshot, error)

	InspectCollectionCatalog(
		ctx context.Context,
		ref collection.CollectionRef,
	) (refresh.CatalogInspection, error)

	PublishManagedArtifact(
		ctx context.Context,
		request managedartifact.PublishRequest,
	) (managedartifact.PublishResult, error)

	PublishManagedCollection(
		ctx context.Context,
		request managedartifact.PublishCollectionRequest,
	) (managedartifact.PublishCollectionResult, error)

	RemoveManagedArtifact(
		ctx context.Context,
		request managedartifact.RemoveRequest,
	) error
}
