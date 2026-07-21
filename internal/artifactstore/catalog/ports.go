package catalog

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type RootDraft struct {
	Kind        artifactstore.RootKind
	DisplayName string
	Description string
	Enabled     bool
	Data        json.RawMessage
}

type AttachmentDraft struct {
	SourceID artifactstore.SourceID
	Role     artifactstore.AttachmentRole
	Priority int
	Enabled  bool
	Data     json.RawMessage
}

type RootUpdate struct {
	ExpectedRevision uint64
	DisplayName      string
	Description      string
	Enabled          bool
	Data             json.RawMessage
}

type AttachmentUpdate struct {
	ExpectedRootRevision       uint64
	ExpectedAttachmentRevision uint64
	Role                       artifactstore.AttachmentRole
	Priority                   int
	Enabled                    bool
	Data                       json.RawMessage
}

type Repository interface {
	CreateRoot(
		ctx context.Context,
		root Root,
		attachments []Attachment,
	) error

	GetRoot(
		ctx context.Context,
		id artifactstore.RootID,
		includeDeleted bool,
	) (Root, error)

	ListRoots(
		ctx context.Context,
		includeDeleted bool,
	) ([]Root, error)

	UpdateRoot(
		ctx context.Context,
		value Root,
		expectedRevision uint64,
	) error

	Attach(
		ctx context.Context,
		attachment Attachment,
		expectedRootRevision uint64,
	) (Root, error)

	GetAttachment(
		ctx context.Context,
		rootID artifactstore.RootID,
		sourceID artifactstore.SourceID,
	) (Attachment, error)

	ListAttachments(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]Attachment, error)

	UpdateAttachment(
		ctx context.Context,
		value Attachment,
		expectedRootRevision uint64,
		expectedAttachmentRevision uint64,
	) (Root, error)

	Detach(
		ctx context.Context,
		rootID artifactstore.RootID,
		sourceID artifactstore.SourceID,
		expectedRootRevision uint64,
		expectedAttachmentRevision uint64,
	) (Root, error)

	GetCurrentCatalog(
		ctx context.Context,
		rootID artifactstore.RootID,
	) (Snapshot, error)
}
