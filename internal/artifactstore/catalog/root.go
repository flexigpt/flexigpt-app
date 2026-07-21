package catalog

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Root struct {
	ID          artifactstore.RootID   `json:"id"`
	Kind        artifactstore.RootKind `json:"kind"`
	DisplayName string                 `json:"displayName"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Data        json.RawMessage        `json:"data"`
	Revision    uint64                 `json:"revision"`
	CreatedAt   time.Time              `json:"createdAt"`
	ModifiedAt  time.Time              `json:"modifiedAt"`
	DeletedAt   *time.Time             `json:"deletedAt,omitempty"`
}

func (r Root) Validate() error {
	if err := artifactstore.ValidateRootID(r.ID); err != nil {
		return err
	}
	if err := artifactstore.ValidateRootKind(r.Kind); err != nil {
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
	if _, err := jsoncanon.CanonicalizeObject(
		r.Data,
		artifactstore.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf("%w: root data: %w", artifactstore.ErrInvalid, err)
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
	if r.DeletedAt != nil && r.Enabled {
		return fmt.Errorf("%w: deleted root cannot be enabled", artifactstore.ErrInvalid)
	}
	return nil
}
