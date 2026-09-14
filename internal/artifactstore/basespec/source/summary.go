package source

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

type Summary struct {
	ID             SourceID            `json:"id"`
	RootID         root.RootID         `json:"rootID"`
	RootStorageKey basespec.StorageKey `json:"rootStorageKey"`
	StorageKey     basespec.StorageKey `json:"storageKey"`
	Kind           SourceKind          `json:"kind"`
	DisplayName    string              `json:"displayName"`
	Enabled        bool                `json:"enabled"`
	Discovery      DiscoverySpec       `json:"discovery"`
	Revision       uint64              `json:"revision"`
	CreatedAt      time.Time           `json:"createdAt"`
	ModifiedAt     time.Time           `json:"modifiedAt"`
	RetiredAt      *time.Time          `json:"retiredAt,omitempty"`
}

func (s Summary) Validate() error {
	if err := s.RootID.Validate(); err != nil {
		return err
	}
	if err := s.RootStorageKey.Validate(); err != nil {
		return err
	}
	if err := s.ID.Validate(); err != nil {
		return err
	}
	if err := s.StorageKey.Validate(); err != nil {
		return err
	}

	if err := s.Kind.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"source display name",
		s.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := s.Discovery.Validate(); err != nil {
		return fmt.Errorf("source discovery: %w", err)
	}
	if s.Revision == 0 {
		return fmt.Errorf("%w: source revision must be greater than zero", basespec.ErrInvalid)
	}
	if s.CreatedAt.IsZero() || s.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: source timestamps are required", basespec.ErrInvalid)
	}
	if s.ModifiedAt.Before(s.CreatedAt) {
		return fmt.Errorf("%w: source modified time precedes creation", basespec.ErrInvalid)
	}
	if s.RetiredAt != nil {
		if s.RetiredAt.IsZero() ||
			s.RetiredAt.Before(s.CreatedAt) ||
			s.RetiredAt.Before(s.ModifiedAt) {
			return fmt.Errorf("%w: source retirement time is invalid", basespec.ErrInvalid)
		}
		if s.Enabled {
			return fmt.Errorf("%w: retired source cannot be enabled", basespec.ErrInvalid)
		}
	}
	return nil
}

func (s Summary) Clone() Summary {
	output := s
	output.Discovery = s.Discovery.Clone()
	if s.RetiredAt != nil {
		retiredAt := *s.RetiredAt
		output.RetiredAt = &retiredAt
	}
	return output
}
