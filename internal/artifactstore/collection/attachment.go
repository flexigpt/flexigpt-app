package collection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Attachment struct {
	RootID       artifactstore.RootID         `json:"rootID"`
	CollectionID artifactstore.CollectionID   `json:"collectionID"`
	SourceID     artifactstore.SourceID       `json:"sourceID"`
	Role         artifactstore.AttachmentRole `json:"role"`
	Enabled      bool                         `json:"enabled"`
	Data         json.RawMessage              `json:"-"`
	Revision     uint64                       `json:"revision"`
	CreatedAt    time.Time                    `json:"createdAt"`
	ModifiedAt   time.Time                    `json:"modifiedAt"`
}

func (a Attachment) Validate() error {
	if err := artifactstore.ValidateRootID(a.RootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateCollectionID(a.CollectionID); err != nil {
		return err
	}
	if err := artifactstore.ValidateSourceID(a.SourceID); err != nil {
		return err
	}
	if err := artifactstore.ValidateAttachmentRole(a.Role); err != nil {
		return err
	}
	if _, err := jsoncanon.CanonicalizeObject(
		a.Data,
		artifactstore.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: attachment data: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	if a.Revision == 0 {
		return fmt.Errorf(
			"%w: attachment revision must be positive",
			artifactstore.ErrInvalid,
		)
	}
	if a.CreatedAt.IsZero() || a.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: attachment timestamps are required",
			artifactstore.ErrInvalid,
		)
	}
	if a.ModifiedAt.Before(a.CreatedAt) {
		return fmt.Errorf(
			"%w: attachment modified time precedes creation",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}

func (a Attachment) Clone() Attachment {
	output := a
	output.Data = append(json.RawMessage(nil), a.Data...)
	return output
}

type AttachmentDraft struct {
	SourceID artifactstore.SourceID       `json:"sourceID"`
	Role     artifactstore.AttachmentRole `json:"role"`
	Enabled  bool                         `json:"enabled"`
	Data     json.RawMessage              `json:"data"`
}

type AttachmentUpdate struct {
	ExpectedCollectionRevision uint64                       `json:"expectedCollectionRevision"`
	ExpectedAttachmentRevision uint64                       `json:"expectedAttachmentRevision"`
	Role                       artifactstore.AttachmentRole `json:"role"`
	Enabled                    bool                         `json:"enabled"`
	Data                       json.RawMessage              `json:"data"`
}

type AttachmentReplacement struct {
	ExpectedCollectionRevision uint64                 `json:"expectedCollectionRevision"`
	PreviousSourceID           artifactstore.SourceID `json:"previousSourceID,omitempty"`
	PreviousAttachmentRevision uint64                 `json:"previousAttachmentRevision,omitempty"`
	Replacement                AttachmentDraft        `json:"replacement"`
}
