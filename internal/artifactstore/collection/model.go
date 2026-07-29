package collection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type CollectionRef struct {
	RootID       basespec.RootID       `json:"rootID"`
	CollectionID basespec.CollectionID `json:"collectionID"`
}

func (r CollectionRef) Validate() error {
	if err := basespec.ValidateRootID(r.RootID); err != nil {
		return err
	}
	return basespec.ValidateCollectionID(r.CollectionID)
}

type Collection struct {
	ID          basespec.CollectionID   `json:"id"`
	RootID      basespec.RootID         `json:"rootID"`
	Kind        basespec.CollectionKind `json:"kind"`
	DisplayName string                  `json:"displayName"`
	Description string                  `json:"description,omitempty"`
	Enabled     bool                    `json:"enabled"`
	Data        json.RawMessage         `json:"-"`
	Revision    uint64                  `json:"revision"`
	CreatedAt   time.Time               `json:"createdAt"`
	ModifiedAt  time.Time               `json:"modifiedAt"`
	RetiredAt   *time.Time              `json:"retiredAt,omitempty"`
}

func (c Collection) Ref() CollectionRef {
	return CollectionRef{
		RootID:       c.RootID,
		CollectionID: c.ID,
	}
}

func (c Collection) Validate() error {
	if err := basespec.ValidateCollectionID(c.ID); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(c.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionKind(c.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"collection display name",
		c.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"collection description",
		c.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if _, err := jsonutil.CanonicalizeObject(
		c.Data,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: collection data: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if c.Revision == 0 {
		return fmt.Errorf(
			"%w: collection revision must be positive",
			basespec.ErrInvalid,
		)
	}
	if c.CreatedAt.IsZero() || c.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: collection timestamps are required",
			basespec.ErrInvalid,
		)
	}
	if c.ModifiedAt.Before(c.CreatedAt) {
		return fmt.Errorf(
			"%w: collection modified time precedes creation",
			basespec.ErrInvalid,
		)
	}
	if c.RetiredAt != nil {
		if c.RetiredAt.IsZero() ||
			c.RetiredAt.Before(c.CreatedAt) ||
			c.RetiredAt.Before(c.ModifiedAt) {
			return fmt.Errorf(
				"%w: collection retirement time is invalid",
				basespec.ErrInvalid,
			)
		}
		if c.Enabled {
			return fmt.Errorf(
				"%w: retired collection cannot be enabled",
				basespec.ErrInvalid,
			)
		}
	}
	return nil
}

func (c Collection) Clone() Collection {
	output := c
	output.Data = append(json.RawMessage(nil), c.Data...)
	if c.RetiredAt != nil {
		value := *c.RetiredAt
		output.RetiredAt = &value
	}
	return output
}

type Draft struct {
	Kind        basespec.CollectionKind `json:"kind"`
	DisplayName string                  `json:"displayName"`
	Description string                  `json:"description,omitempty"`
	Enabled     bool                    `json:"enabled"`
	Data        json.RawMessage         `json:"data"`
}

type Update struct {
	ExpectedRevision uint64          `json:"expectedRevision"`
	DisplayName      string          `json:"displayName"`
	Description      string          `json:"description,omitempty"`
	Enabled          bool            `json:"enabled"`
	Data             json.RawMessage `json:"data"`
}
