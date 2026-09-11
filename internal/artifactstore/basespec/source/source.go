package source

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type (
	SourceID   string
	SourceKind string
)

const (
	SourceKindFilesystemDirectory SourceKind = "fs-directory"
	SourceKindEmbeddedDirectory   SourceKind = "embedded-directory"
	SourceKindManagedDirectory    SourceKind = "managed-directory"
)

func (v SourceID) Validate() error {
	err := uuidutil.ValidateUUIDv7(string(v))
	if err != nil {
		return fmt.Errorf("source ID: %w", err)
	}
	return nil
}

func (v SourceKind) Validate() error {
	return basespec.ValidateIdentifier("source kind", string(v), basespec.MaxKindBytes)
}

type Source struct {
	ID             SourceID            `json:"id"`
	RootID         root.RootID         `json:"rootID"`
	RootStorageKey basespec.StorageKey `json:"rootStorageKey"`
	StorageKey     basespec.StorageKey `json:"storageKey"`
	Kind           SourceKind          `json:"kind"`
	DisplayName    string              `json:"displayName"`
	Enabled        bool                `json:"enabled"`
	Config         json.RawMessage     `json:"-"`

	Revision   uint64     `json:"revision"`
	CreatedAt  time.Time  `json:"createdAt"`
	ModifiedAt time.Time  `json:"modifiedAt"`
	RetiredAt  *time.Time `json:"retiredAt,omitempty"`
}

func (s Source) Clone() Source {
	output := s
	output.Config = append(json.RawMessage(nil), s.Config...)
	output.RetiredAt = cloneTime(s.RetiredAt)
	return output
}

func (s Source) Validate() error {
	if err := s.Summary().Validate(); err != nil {
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
		ID:             s.ID,
		RootID:         s.RootID,
		RootStorageKey: s.RootStorageKey,
		StorageKey:     s.StorageKey,
		Kind:           s.Kind,
		DisplayName:    s.DisplayName,
		Enabled:        s.Enabled,
		Revision:       s.Revision,
		CreatedAt:      s.CreatedAt,
		ModifiedAt:     s.ModifiedAt,
		RetiredAt:      cloneTime(s.RetiredAt),
	}
}
