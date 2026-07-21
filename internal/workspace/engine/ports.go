package engine

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type sourceLookup interface {
	Get(
		ctx context.Context,
		id artifactstore.SourceID,
	) (source.Source, error)
}

type recordLookup interface {
	Get(
		ctx context.Context,
		id artifactstore.RecordID,
	) (record.Record, error)

	ListByRoot(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]record.Record, error)
}

type definitionLookup interface {
	Get(
		ctx context.Context,
		digest artifactstore.Digest,
	) (definition.Definition, error)
}

type workspaceRootStore interface {
	Create(
		ctx context.Context,
		draft root.RootDraft,
		attachments []root.AttachmentDraft,
	) (root.Root, []root.Attachment, error)

	Get(
		ctx context.Context,
		id artifactstore.RootID,
	) (root.Root, error)

	List(
		ctx context.Context,
		includeDeleted bool,
	) ([]root.Root, error)

	Update(
		ctx context.Context,
		id artifactstore.RootID,
		update root.RootUpdate,
	) (root.Root, error)

	Delete(
		ctx context.Context,
		id artifactstore.RootID,
		expectedRevision uint64,
	) (root.Root, error)

	Attach(
		ctx context.Context,
		rootID artifactstore.RootID,
		expectedRootRevision uint64,
		draft root.AttachmentDraft,
	) (root.Root, root.Attachment, error)

	GetAttachment(
		ctx context.Context,
		rootID artifactstore.RootID,
		sourceID artifactstore.SourceID,
	) (root.Attachment, error)

	ListAttachments(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]root.Attachment, error)

	Detach(
		ctx context.Context,
		rootID artifactstore.RootID,
		sourceID artifactstore.SourceID,
		expectedRootRevision uint64,
		expectedAttachmentRevision uint64,
	) (root.Root, error)
}

type catalogSnapshotReader interface {
	GetCurrent(
		ctx context.Context,
		rootID artifactstore.RootID,
	) (catalog.Snapshot, error)
}
