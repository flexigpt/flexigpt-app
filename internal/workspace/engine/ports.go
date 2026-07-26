package engine

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type sourceSummaryLookup interface {
	Get(
		ctx context.Context,
		rootID artifactstore.RootID,
		id artifactstore.SourceID,
	) (source.Summary, error)
}

type artifactLookup interface {
	Get(
		ctx context.Context,
		ref artifactstore.ArtifactRef,
	) (artifact.Artifact, error)

	ListByCollection(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) ([]artifact.Artifact, error)
}

type definitionLookup interface {
	Get(
		ctx context.Context,
		rootID artifactstore.RootID,
		digest artifactstore.Digest,
	) (definition.Definition, error)
}

type workspaceCollectionStore interface {
	Create(
		ctx context.Context,
		rootID artifactstore.RootID,
		draft collection.Draft,
		attachments []collection.AttachmentDraft,
	) (collection.Collection, []collection.Attachment, error)

	Get(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) (collection.Collection, error)

	ListByRoot(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]collection.Collection, error)

	Update(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		update collection.Update,
	) (collection.Collection, error)

	Retire(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		expectedRevision uint64,
	) (collection.Collection, error)

	Attach(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		expectedCollectionRevision uint64,
		draft collection.AttachmentDraft,
	) (collection.Collection, collection.Attachment, error)

	GetAttachment(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		sourceID artifactstore.SourceID,
	) (collection.Attachment, error)

	ListAttachments(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) ([]collection.Attachment, error)

	UpdateAttachment(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		sourceID artifactstore.SourceID,
		update collection.AttachmentUpdate,
	) (collection.Collection, collection.Attachment, error)

	Detach(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		sourceID artifactstore.SourceID,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
	) (collection.Collection, error)

	ReplaceAttachment(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		replacement collection.AttachmentReplacement,
	) (collection.Collection, collection.Attachment, error)
}

type catalogSnapshotReader interface {
	GetCurrent(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) (catalog.Snapshot, error)
}
