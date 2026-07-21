package engine

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
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
	CreateRoot(
		ctx context.Context,
		draft catalog.RootDraft,
		attachments []catalog.AttachmentDraft,
	) (catalog.Root, []catalog.Attachment, error)

	GetRoot(
		ctx context.Context,
		id artifactstore.RootID,
	) (catalog.Root, error)

	ListRoots(
		ctx context.Context,
		includeDeleted bool,
	) ([]catalog.Root, error)

	UpdateRoot(
		ctx context.Context,
		id artifactstore.RootID,
		update catalog.RootUpdate,
	) (catalog.Root, error)

	DeleteRoot(
		ctx context.Context,
		id artifactstore.RootID,
		expectedRevision uint64,
	) (catalog.Root, error)

	Attach(
		ctx context.Context,
		rootID artifactstore.RootID,
		expectedRootRevision uint64,
		draft catalog.AttachmentDraft,
	) (catalog.Root, catalog.Attachment, error)

	GetAttachment(
		ctx context.Context,
		rootID artifactstore.RootID,
		sourceID artifactstore.SourceID,
	) (catalog.Attachment, error)

	ListAttachments(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]catalog.Attachment, error)

	Detach(
		ctx context.Context,
		rootID artifactstore.RootID,
		sourceID artifactstore.SourceID,
		expectedRootRevision uint64,
		expectedAttachmentRevision uint64,
	) (catalog.Root, error)
}

type catalogSnapshotReader interface {
	Current(
		ctx context.Context,
		rootID artifactstore.RootID,
	) (catalog.Snapshot, error)
}
