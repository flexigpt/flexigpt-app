package collection

import (
	"context"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Reader interface {
	Get(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) (Collection, error)

	ListByRoot(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]Collection, error)

	GetAttachment(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		sourceID artifactstore.SourceID,
	) (Attachment, error)

	ListAttachments(
		ctx context.Context,
		ref artifactstore.CollectionRef,
	) ([]Attachment, error)
}

type Repository interface {
	Reader

	Create(
		ctx context.Context,
		value Collection,
		attachments []Attachment,
	) error

	Update(
		ctx context.Context,
		value Collection,
		expectedRevision uint64,
	) error

	Retire(
		ctx context.Context,
		value Collection,
		expectedRevision uint64,
	) error

	Purge(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		expectedRevision uint64,
	) error

	Attach(
		ctx context.Context,
		value Attachment,
		expectedCollectionRevision uint64,
	) (Collection, error)

	UpdateAttachment(
		ctx context.Context,
		value Attachment,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
	) (Collection, error)

	Detach(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		sourceID artifactstore.SourceID,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
		modifiedAt time.Time,
	) (Collection, error)

	ReplaceAttachment(
		ctx context.Context,
		ref artifactstore.CollectionRef,
		previousSourceID artifactstore.SourceID,
		expectedPreviousRevision uint64,
		replacement Attachment,
		expectedCollectionRevision uint64,
	) (Collection, error)
}
