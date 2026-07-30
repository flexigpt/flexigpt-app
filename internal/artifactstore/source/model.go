package source

import (
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Source struct {
	ID          basespec.SourceID   `json:"id"`
	RootID      basespec.RootID     `json:"rootID"`
	Kind        basespec.SourceKind `json:"kind"`
	DisplayName string              `json:"displayName"`
	Enabled     bool                `json:"enabled"`
	Config      json.RawMessage     `json:"-"`

	Revision   uint64     `json:"revision"`
	CreatedAt  time.Time  `json:"createdAt"`
	ModifiedAt time.Time  `json:"modifiedAt"`
	RetiredAt  *time.Time `json:"retiredAt,omitempty"`
}

type Summary struct {
	ID          basespec.SourceID   `json:"id"`
	RootID      basespec.RootID     `json:"rootID"`
	Kind        basespec.SourceKind `json:"kind"`
	DisplayName string              `json:"displayName"`
	Enabled     bool                `json:"enabled"`
	Revision    uint64              `json:"revision"`
	CreatedAt   time.Time           `json:"createdAt"`
	ModifiedAt  time.Time           `json:"modifiedAt"`
	RetiredAt   *time.Time          `json:"retiredAt,omitempty"`
}

func (s Source) Clone() Source {
	output := s
	output.Config = append(json.RawMessage(nil), s.Config...)
	output.RetiredAt = cloneTime(s.RetiredAt)
	return output
}

func (s Summary) Validate() error {
	if err := basespec.ValidateRootID(s.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(s.ID); err != nil {
		return err
	}

	if err := basespec.ValidateSourceKind(s.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"source display name",
		s.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
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

func (s Source) Validate() error {
	if err := s.Summary().Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateSourceKind(s.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"source display name",
		s.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if _, err := jsonutil.CanonicalizeObject(
		s.Config,
		basespec.MaxConfigBytes,
	); err != nil {
		return fmt.Errorf("%w: source config: %w", basespec.ErrInvalid, err)
	}

	return nil
}

func (s Source) Summary() Summary {
	return Summary{
		ID:          s.ID,
		RootID:      s.RootID,
		Kind:        s.Kind,
		DisplayName: s.DisplayName,
		Enabled:     s.Enabled,
		Revision:    s.Revision,
		CreatedAt:   s.CreatedAt,
		ModifiedAt:  s.ModifiedAt,
		RetiredAt:   cloneTime(s.RetiredAt),
	}
}

type Draft struct {
	Kind        basespec.SourceKind
	DisplayName string
	Enabled     bool
	Config      json.RawMessage
}

type Update struct {
	ExpectedRevision uint64
	DisplayName      string
	Enabled          bool

	// Config is write-only replacement configuration. A nil value preserves
	// the current normalized configuration so public callers can update Source
	// metadata without reading or resending private Source configuration.
	Config json.RawMessage
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

type Entry struct {
	Locator    basespec.Locator
	Name       string
	SizeBytes  int64
	Mode       uint32
	ModifiedAt time.Time

	IsDirectory bool
	IsRegular   bool
}

func (e Entry) Validate() error {
	if err := basespec.ValidateLocator(e.Locator, true); err != nil {
		return err
	}
	if e.Name == "" {
		return fmt.Errorf("%w: source entry name is empty", basespec.ErrInvalid)
	}
	if e.Locator != "." &&
		e.Name != path.Base(string(e.Locator)) {
		return fmt.Errorf(
			"%w: source entry name does not match locator",
			basespec.ErrInvalid,
		)
	}
	if e.SizeBytes < 0 {
		return fmt.Errorf("%w: source entry size is negative", basespec.ErrInvalid)
	}
	if e.IsDirectory == e.IsRegular {
		return fmt.Errorf(
			"%w: source entry must identify exactly one entry type",
			basespec.ErrInvalid,
		)
	}
	return nil
}
