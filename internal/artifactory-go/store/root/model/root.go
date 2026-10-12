package model

import (
	"fmt"
	"time"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
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
	ID          RootID               `json:"id"`
	StorageKey  storeSpec.StorageKey `json:"storageKey"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description,omitempty"`
	Revision    uint64               `json:"revision"`
	CreatedAt   time.Time            `json:"createdAt"`
	ModifiedAt  time.Time            `json:"modifiedAt"`
	RetiredAt   *time.Time           `json:"retiredAt,omitempty"`
}

func (r Root) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if err := r.StorageKey.Validate(); err != nil {
		return err
	}
	if err := storeSpec.ValidateRequiredText(
		"root display name",
		r.DisplayName,
		storeSpec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := storeSpec.ValidateOptionalText(
		"root description",
		r.Description,
		storeSpec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if r.Revision == 0 {
		return fmt.Errorf("%w: root revision must be positive", storeSpec.ErrInvalid)
	}
	if r.CreatedAt.IsZero() || r.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: root timestamps are required", storeSpec.ErrInvalid)
	}
	if r.ModifiedAt.Before(r.CreatedAt) {
		return fmt.Errorf("%w: root modified time precedes creation", storeSpec.ErrInvalid)
	}
	if r.RetiredAt != nil {
		if r.RetiredAt.IsZero() ||
			r.RetiredAt.Before(r.CreatedAt) ||
			r.RetiredAt.Before(r.ModifiedAt) {
			return fmt.Errorf(
				"%w: root retirement time is invalid",
				storeSpec.ErrInvalid,
			)
		}
	}
	return nil
}
