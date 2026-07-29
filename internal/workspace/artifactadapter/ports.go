package artifactadapter

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type sourceSummaryLookup interface {
	Get(
		ctx context.Context,
		rootID basespec.RootID,
		id basespec.SourceID,
	) (source.Summary, error)
}

type artifactLookup interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	ListByCollection(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)
}

type definitionLookup interface {
	Get(
		ctx context.Context,
		rootID basespec.RootID,
		digest cryptoutil.Digest,
	) (definition.Definition, error)
}

type workspaceCollectionStore interface {
	Create(
		ctx context.Context,
		rootID basespec.RootID,
		draft collection.Draft,
		attachments []collection.AttachmentDraft,
	) (collection.Collection, []collection.Attachment, error)

	Get(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)

	GetRetired(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)

	ListByRoot(
		ctx context.Context,
		rootID basespec.RootID,
	) ([]collection.Collection, error)

	Update(
		ctx context.Context,
		ref collection.CollectionRef,
		update collection.Update,
	) (collection.Collection, error)

	Retire(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) (collection.Collection, error)

	Purge(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) error

	Attach(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedCollectionRevision uint64,
		draft collection.AttachmentDraft,
	) (collection.Collection, collection.Attachment, error)

	GetAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
	) (collection.Attachment, error)

	ListAttachments(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]collection.Attachment, error)

	UpdateAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
		update collection.AttachmentUpdate,
	) (collection.Collection, collection.Attachment, error)

	Detach(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
	) (collection.Collection, error)

	ReplaceAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		replacement collection.AttachmentReplacement,
	) (collection.Collection, collection.Attachment, error)
}

type catalogSnapshotReader interface {
	GetCurrent(
		ctx context.Context,
		ref collection.CollectionRef,
	) (catalog.Snapshot, error)
}
