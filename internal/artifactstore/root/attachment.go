package root

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Attachment struct {
	RootID     artifactstore.RootID         `json:"rootID"`
	SourceID   artifactstore.SourceID       `json:"sourceID"`
	Role       artifactstore.AttachmentRole `json:"role"`
	Enabled    bool                         `json:"enabled"`
	Data       json.RawMessage              `json:"data"`
	Revision   uint64                       `json:"revision"`
	CreatedAt  time.Time                    `json:"createdAt"`
	ModifiedAt time.Time                    `json:"modifiedAt"`
}

func (a Attachment) Validate() error {
	if err := artifactstore.ValidateRootID(a.RootID); err != nil {
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
		return fmt.Errorf("%w: attachment data: %w", artifactstore.ErrInvalid, err)
	}
	if a.Revision == 0 {
		return fmt.Errorf("%w: attachment revision must be positive", artifactstore.ErrInvalid)
	}
	if a.CreatedAt.IsZero() || a.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: attachment timestamps are required", artifactstore.ErrInvalid)
	}
	if a.ModifiedAt.Before(a.CreatedAt) {
		return fmt.Errorf(
			"%w: attachment modified time precedes creation",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}
