package collectionimpl

import (
	"context"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
)

type Reader interface {
	Get(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)

	ListByRoot(
		ctx context.Context,
		rootID basespec.RootID,
	) ([]collection.Collection, error)

	GetAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
	) (collection.Attachment, error)

	ListAttachments(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]collection.Attachment, error)
}

// RetiredReader is intentionally separate from Reader. Ordinary consumers
// must not treat retired Collections as active aggregates, while typed domain
// lifecycle services need to verify Collection kind before destructive purge.
type RetiredReader interface {
	GetRetired(
		ctx context.Context,
		ref collection.CollectionRef,
	) (collection.Collection, error)
}

type Repository interface {
	Reader
	RetiredReader

	Create(
		ctx context.Context,
		value collection.Collection,
		attachments []collection.Attachment,
	) error

	Update(
		ctx context.Context,
		value collection.Collection,
		expectedRevision uint64,
	) error

	Retire(
		ctx context.Context,
		value collection.Collection,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		ref collection.CollectionRef,
		expectedRevision uint64,
	) error

	Attach(
		ctx context.Context,
		value collection.Attachment,
		expectedCollectionRevision uint64,
	) (collection.Collection, error)

	UpdateAttachment(
		ctx context.Context,
		value collection.Attachment,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
	) (collection.Collection, error)

	Detach(
		ctx context.Context,
		ref collection.CollectionRef,
		sourceID basespec.SourceID,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
		modifiedAt time.Time,
	) (collection.Collection, error)

	ReplaceAttachment(
		ctx context.Context,
		ref collection.CollectionRef,
		previousSourceID basespec.SourceID,
		expectedPreviousRevision uint64,
		replacement collection.Attachment,
		expectedCollectionRevision uint64,
	) (collection.Collection, error)
}
