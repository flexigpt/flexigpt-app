package root

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Root struct {
	ID          artifactstore.RootID `json:"id"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description,omitempty"`
	Revision    uint64               `json:"revision"`
	CreatedAt   time.Time            `json:"createdAt"`
	ModifiedAt  time.Time            `json:"modifiedAt"`
	RetiredAt   *time.Time           `json:"retiredAt,omitempty"`
}

func (r Root) Validate() error {
	if err := artifactstore.ValidateRootID(r.ID); err != nil {
		return err
	}
	if err := artifactstore.ValidateRequiredText(
		"root display name",
		r.DisplayName,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := artifactstore.ValidateOptionalText(
		"root description",
		r.Description,
		artifactstore.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if r.Revision == 0 {
		return fmt.Errorf("%w: root revision must be positive", artifactstore.ErrInvalid)
	}
	if r.CreatedAt.IsZero() || r.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: root timestamps are required", artifactstore.ErrInvalid)
	}
	if r.ModifiedAt.Before(r.CreatedAt) {
		return fmt.Errorf("%w: root modified time precedes creation", artifactstore.ErrInvalid)
	}
	if r.RetiredAt != nil {
		if r.RetiredAt.IsZero() ||
			r.RetiredAt.Before(r.CreatedAt) ||
			r.RetiredAt.Before(r.ModifiedAt) {
			return fmt.Errorf(
				"%w: root retirement time is invalid",
				artifactstore.ErrInvalid,
			)
		}
	}
	return nil
}
