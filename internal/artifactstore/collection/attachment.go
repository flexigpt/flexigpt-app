package collection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Attachment struct {
	RootID       basespec.RootID         `json:"rootID"`
	CollectionID basespec.CollectionID   `json:"collectionID"`
	SourceID     basespec.SourceID       `json:"sourceID"`
	Role         basespec.AttachmentRole `json:"role"`
	Enabled      bool                    `json:"enabled"`
	Revision     uint64                  `json:"revision"`
	CreatedAt    time.Time               `json:"createdAt"`
	ModifiedAt   time.Time               `json:"modifiedAt"`

	Data json.RawMessage `json:"-"`
}

func (a Attachment) Validate() error {
	if err := basespec.ValidateRootID(a.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionID(a.CollectionID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(a.SourceID); err != nil {
		return err
	}
	if err := basespec.ValidateAttachmentRole(a.Role); err != nil {
		return err
	}
	if _, err := jsonutil.CanonicalizeObject(
		a.Data,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: attachment data: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if a.Revision == 0 {
		return fmt.Errorf(
			"%w: attachment revision must be positive",
			basespec.ErrInvalid,
		)
	}
	if a.CreatedAt.IsZero() || a.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: attachment timestamps are required",
			basespec.ErrInvalid,
		)
	}
	if a.ModifiedAt.Before(a.CreatedAt) {
		return fmt.Errorf(
			"%w: attachment modified time precedes creation",
			basespec.ErrInvalid,
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
	SourceID basespec.SourceID       `json:"sourceID"`
	Role     basespec.AttachmentRole `json:"role"`
	Enabled  bool                    `json:"enabled"`
	Data     json.RawMessage         `json:"data"`
}

type AttachmentUpdate struct {
	ExpectedCollectionRevision uint64                  `json:"expectedCollectionRevision"`
	ExpectedAttachmentRevision uint64                  `json:"expectedAttachmentRevision"`
	Role                       basespec.AttachmentRole `json:"role"`
	Enabled                    bool                    `json:"enabled"`
	Data                       json.RawMessage         `json:"data"`
}

type AttachmentReplacement struct {
	ExpectedCollectionRevision uint64            `json:"expectedCollectionRevision"`
	PreviousSourceID           basespec.SourceID `json:"previousSourceID,omitempty"`
	PreviousAttachmentRevision uint64            `json:"previousAttachmentRevision,omitempty"`
	Replacement                AttachmentDraft   `json:"replacement"`
}
