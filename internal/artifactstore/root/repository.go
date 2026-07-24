package root

import (
	"context"
	"encoding/json"
	"time"

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
	Enabled                    bool
	Data                       json.RawMessage
}

type Repository interface {
	Create(
		ctx context.Context,
		root Root,
		attachments []Attachment,
	) error

	Get(
		ctx context.Context,
		id artifactstore.RootID,
	) (Root, error)

	List(ctx context.Context) ([]Root, error)

	Update(
		ctx context.Context,
		value Root,
		expectedRevision uint64,
	) error

	// Retire soft-deletes a root and removes all active root-scoped state that
	// would otherwise retain attached sources or expose cataloged resources.
	Retire(
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
		modifiedAt time.Time,
	) (Root, error)
}
