package collection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Collection struct {
	ID          artifactstore.CollectionID   `json:"id"`
	RootID      artifactstore.RootID         `json:"rootID"`
	Kind        artifactstore.CollectionKind `json:"kind"`
	DisplayName string                       `json:"displayName"`
	Description string                       `json:"description,omitempty"`
	Enabled     bool                         `json:"enabled"`
	Data        json.RawMessage              `json:"-"`
	Revision    uint64                       `json:"revision"`
	CreatedAt   time.Time                    `json:"createdAt"`
	ModifiedAt  time.Time                    `json:"modifiedAt"`
	RetiredAt   *time.Time                   `json:"retiredAt,omitempty"`
}

func (c Collection) Ref() artifactstore.CollectionRef {
	return artifactstore.CollectionRef{
		RootID:       c.RootID,
		CollectionID: c.ID,
	}
}

func (c Collection) Validate() error {
	if err := artifactstore.ValidateCollectionID(c.ID); err != nil {
		return err
	}
	if err := artifactstore.ValidateRootID(c.RootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateCollectionKind(c.Kind); err != nil {
		return err
	}
	if err := artifactstore.ValidateRequiredText(
		"collection display name",
		c.DisplayName,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := artifactstore.ValidateOptionalText(
		"collection description",
		c.Description,
		artifactstore.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if _, err := jsoncanon.CanonicalizeObject(
		c.Data,
		artifactstore.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: collection data: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	if c.Revision == 0 {
		return fmt.Errorf(
			"%w: collection revision must be positive",
			artifactstore.ErrInvalid,
		)
	}
	if c.CreatedAt.IsZero() || c.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: collection timestamps are required",
			artifactstore.ErrInvalid,
		)
	}
	if c.ModifiedAt.Before(c.CreatedAt) {
		return fmt.Errorf(
			"%w: collection modified time precedes creation",
			artifactstore.ErrInvalid,
		)
	}
	if c.RetiredAt != nil {
		if c.RetiredAt.IsZero() ||
			c.RetiredAt.Before(c.CreatedAt) ||
			c.RetiredAt.Before(c.ModifiedAt) {
			return fmt.Errorf(
				"%w: collection retirement time is invalid",
				artifactstore.ErrInvalid,
			)
		}
		if c.Enabled {
			return fmt.Errorf(
				"%w: retired collection cannot be enabled",
				artifactstore.ErrInvalid,
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
	Kind        artifactstore.CollectionKind `json:"kind"`
	DisplayName string                       `json:"displayName"`
	Description string                       `json:"description,omitempty"`
	Enabled     bool                         `json:"enabled"`
	Data        json.RawMessage              `json:"data"`
}

type Update struct {
	ExpectedRevision uint64          `json:"expectedRevision"`
	DisplayName      string          `json:"displayName"`
	Description      string          `json:"description,omitempty"`
	Enabled          bool            `json:"enabled"`
	Data             json.RawMessage `json:"data"`
}
