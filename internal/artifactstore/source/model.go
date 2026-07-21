package source

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Source struct {
	ID          artifactstore.SourceID   `json:"id"`
	Kind        artifactstore.SourceKind `json:"kind"`
	DisplayName string                   `json:"displayName"`
	Enabled     bool                     `json:"enabled"`
	Config      json.RawMessage          `json:"config"`
	Revision    uint64                   `json:"revision"`
	CreatedAt   time.Time                `json:"createdAt"`
	ModifiedAt  time.Time                `json:"modifiedAt"`
}

type Summary struct {
	ID          artifactstore.SourceID   `json:"id"`
	Kind        artifactstore.SourceKind `json:"kind"`
	DisplayName string                   `json:"displayName"`
	Enabled     bool                     `json:"enabled"`
	Revision    uint64                   `json:"revision"`
	CreatedAt   time.Time                `json:"createdAt"`
	ModifiedAt  time.Time                `json:"modifiedAt"`
}

func (s Source) Validate() error {
	if err := artifactstore.ValidateSourceID(s.ID); err != nil {
		return err
	}
	if err := artifactstore.ValidateSourceKind(s.Kind); err != nil {
		return err
	}
	if err := artifactstore.ValidateRequiredText(
		"source display name",
		s.DisplayName,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if _, err := jsoncanon.CanonicalizeObject(
		s.Config,
		artifactstore.MaxConfigBytes,
	); err != nil {
		return fmt.Errorf("%w: source config: %w", artifactstore.ErrInvalid, err)
	}
	if s.Revision == 0 {
		return fmt.Errorf("%w: source revision must be greater than zero", artifactstore.ErrInvalid)
	}
	if s.CreatedAt.IsZero() || s.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: source timestamps are required", artifactstore.ErrInvalid)
	}
	if s.ModifiedAt.Before(s.CreatedAt) {
		return fmt.Errorf("%w: source modified time precedes creation", artifactstore.ErrInvalid)
	}
	return nil
}

func (s Source) Summary() Summary {
	return Summary{
		ID:          s.ID,
		Kind:        s.Kind,
		DisplayName: s.DisplayName,
		Enabled:     s.Enabled,
		Revision:    s.Revision,
		CreatedAt:   s.CreatedAt,
		ModifiedAt:  s.ModifiedAt,
	}
}

type Draft struct {
	Kind        artifactstore.SourceKind
	DisplayName string
	Enabled     bool
	Config      json.RawMessage
}

type Update struct {
	ExpectedRevision uint64
	DisplayName      string
	Enabled          bool
	Config           json.RawMessage
}

type Entry struct {
	Locator     artifactstore.Locator
	Name        string
	SizeBytes   int64
	Mode        uint32
	ModifiedAt  time.Time
	IsDirectory bool
	IsRegular   bool
	IsSymlink   bool
}

func (e Entry) Validate() error {
	if err := artifactstore.ValidateLocator(e.Locator, true); err != nil {
		return err
	}
	if e.Name == "" {
		return fmt.Errorf("%w: source entry name is empty", artifactstore.ErrInvalid)
	}
	if e.SizeBytes < 0 {
		return fmt.Errorf("%w: source entry size is negative", artifactstore.ErrInvalid)
	}
	modes := 0
	if e.IsDirectory {
		modes++
	}
	if e.IsRegular {
		modes++
	}
	if e.IsSymlink {
		modes++
	}
	if modes != 1 {
		return fmt.Errorf(
			"%w: source entry must identify exactly one entry type",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}
