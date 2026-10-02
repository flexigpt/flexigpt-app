package root

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type RootID string

func (v RootID) Validate() error {
	err := uuidutil.ValidateUUIDv7(string(v))
	if err != nil {
		return fmt.Errorf("root ID: %w", err)
	}
	return nil
}

type Root struct {
	ID          RootID              `json:"id"`
	StorageKey  basespec.StorageKey `json:"storageKey"`
	DisplayName string              `json:"displayName"`
	Description string              `json:"description,omitempty"`
	Revision    uint64              `json:"revision"`
	CreatedAt   time.Time           `json:"createdAt"`
	ModifiedAt  time.Time           `json:"modifiedAt"`
	RetiredAt   *time.Time          `json:"retiredAt,omitempty"`
}

func (r Root) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if err := r.StorageKey.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"root display name",
		r.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"root description",
		r.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if r.Revision == 0 {
		return fmt.Errorf("%w: root revision must be positive", basespec.ErrInvalid)
	}
	if r.CreatedAt.IsZero() || r.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: root timestamps are required", basespec.ErrInvalid)
	}
	if r.ModifiedAt.Before(r.CreatedAt) {
		return fmt.Errorf("%w: root modified time precedes creation", basespec.ErrInvalid)
	}
	if r.RetiredAt != nil {
		if r.RetiredAt.IsZero() ||
			r.RetiredAt.Before(r.CreatedAt) ||
			r.RetiredAt.Before(r.ModifiedAt) {
			return fmt.Errorf(
				"%w: root retirement time is invalid",
				basespec.ErrInvalid,
			)
		}
	}
	return nil
}
