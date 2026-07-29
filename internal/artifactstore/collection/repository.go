package collection

import (
	"context"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type Reader interface {
	Get(
		ctx context.Context,
		ref basespec.CollectionRef,
	) (Collection, error)

	ListByRoot(
		ctx context.Context,
		rootID basespec.RootID,
	) ([]Collection, error)

	GetAttachment(
		ctx context.Context,
		ref basespec.CollectionRef,
		sourceID basespec.SourceID,
	) (Attachment, error)

	ListAttachments(
		ctx context.Context,
		ref basespec.CollectionRef,
	) ([]Attachment, error)
}

// RetiredReader is intentionally separate from Reader. Ordinary consumers
// must not treat retired Collections as active aggregates, while typed domain
// lifecycle services need to verify Collection kind before destructive purge.
type RetiredReader interface {
	GetRetired(
		ctx context.Context,
		ref basespec.CollectionRef,
	) (Collection, error)
}

type Repository interface {
	Reader
	RetiredReader

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
		ref basespec.CollectionRef,
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
		ref basespec.CollectionRef,
		sourceID basespec.SourceID,
		expectedCollectionRevision uint64,
		expectedAttachmentRevision uint64,
		modifiedAt time.Time,
	) (Collection, error)

	ReplaceAttachment(
		ctx context.Context,
		ref basespec.CollectionRef,
		previousSourceID basespec.SourceID,
		expectedPreviousRevision uint64,
		replacement Attachment,
		expectedCollectionRevision uint64,
	) (Collection, error)
}
