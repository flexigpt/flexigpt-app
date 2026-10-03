package model

import (
	"encoding/json"
	"fmt"
	"time"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type (
	SourceID   string
	SourceKind string
)

func (v SourceID) Validate() error {
	err := uuidutil.ValidateUUIDv7(string(v))
	if err != nil {
		return fmt.Errorf("source ID: %w", err)
	}
	return nil
}

func (v SourceKind) Validate() error {
	return spec.ValidateIdentifier("source kind", string(v), spec.MaxKindBytes)
}

type Source struct {
	ID             SourceID         `json:"id"`
	RootID         rootModel.RootID `json:"rootID"`
	RootStorageKey spec.StorageKey  `json:"rootStorageKey"`
	StorageKey     spec.StorageKey  `json:"storageKey"`
	Kind           SourceKind       `json:"kind"`
	DisplayName    string           `json:"displayName"`
	Enabled        bool             `json:"enabled"`
	Config         json.RawMessage  `json:"-"`
	Discovery      DiscoverySpec    `json:"discovery"`

	Revision   uint64     `json:"revision"`
	CreatedAt  time.Time  `json:"createdAt"`
	ModifiedAt time.Time  `json:"modifiedAt"`
	RetiredAt  *time.Time `json:"retiredAt,omitempty"`
}

func (s Source) Clone() Source {
	output := s
	output.Config = append(json.RawMessage(nil), s.Config...)
	output.Discovery = s.Discovery.Clone()
	output.RetiredAt = cloneTime(s.RetiredAt)
	return output
}

func (s Source) Validate() error {
	if err := s.Summary().Validate(); err != nil {
		return err
	}
	if _, err := jsonutil.CanonicalizeObject(
		s.Config,
		spec.MaxConfigBytes,
	); err != nil {
		return fmt.Errorf("%w: source config: %w", spec.ErrInvalid, err)
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
		Discovery:      s.Discovery.Clone(),
		Revision:       s.Revision,
		CreatedAt:      s.CreatedAt,
		ModifiedAt:     s.ModifiedAt,
		RetiredAt:      cloneTime(s.RetiredAt),
	}
}
