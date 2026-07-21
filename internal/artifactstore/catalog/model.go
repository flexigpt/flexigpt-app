package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
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

type Attachment struct {
	RootID     artifactstore.RootID         `json:"rootID"`
	SourceID   artifactstore.SourceID       `json:"sourceID"`
	Role       artifactstore.AttachmentRole `json:"role"`
	Priority   int                          `json:"priority"`
	Enabled    bool                         `json:"enabled"`
	Data       json.RawMessage              `json:"data"`
	Revision   uint64                       `json:"revision"`
	CreatedAt  time.Time                    `json:"createdAt"`
	ModifiedAt time.Time                    `json:"modifiedAt"`
}

func (a Attachment) Validate() error {
	if err := artifactstore.ValidateRootID(a.RootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateSourceID(a.SourceID); err != nil {
		return err
	}
	if err := artifactstore.ValidateAttachmentRole(a.Role); err != nil {
		return err
	}
	if _, err := jsoncanon.CanonicalizeObject(
		a.Data,
		artifactstore.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf("%w: attachment data: %w", artifactstore.ErrInvalid, err)
	}
	if a.Revision == 0 {
		return fmt.Errorf("%w: attachment revision must be positive", artifactstore.ErrInvalid)
	}
	if a.CreatedAt.IsZero() || a.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: attachment timestamps are required", artifactstore.ErrInvalid)
	}
	return nil
}

type OccurrenceState string

const (
	OccurrenceValid   OccurrenceState = "valid"
	OccurrenceInvalid OccurrenceState = "invalid"
	OccurrenceMissing OccurrenceState = "missing"
)

type OccurrenceKey struct {
	SourceID           artifactstore.SourceID           `json:"sourceID"`
	Locator            artifactstore.Locator            `json:"locator"`
	SubresourceLocator artifactstore.SubresourceLocator `json:"subresourceLocator,omitempty"`
}

func (k OccurrenceKey) Validate() error {
	if err := artifactstore.ValidateSourceID(k.SourceID); err != nil {
		return err
	}
	if err := artifactstore.ValidateLocator(k.Locator, false); err != nil {
		return err
	}
	return artifactstore.ValidateSubresourceLocator(k.SubresourceLocator)
}

func (k OccurrenceKey) String() string {
	return string(k.SourceID) + "\x00" +
		string(k.Locator) + "\x00" +
		string(k.SubresourceLocator)
}

type Occurrence struct {
	RootID              artifactstore.RootID         `json:"rootID"`
	Key                 OccurrenceKey                `json:"key"`
	Kind                artifactstore.ArtifactKind   `json:"kind,omitempty"`
	LogicalName         artifactstore.LogicalName    `json:"logicalName,omitempty"`
	LogicalVersion      artifactstore.LogicalVersion `json:"logicalVersion,omitempty"`
	DefinitionDigest    *artifactstore.Digest        `json:"definitionDigest,omitempty"`
	SourceContentDigest *artifactstore.Digest        `json:"sourceContentDigest,omitempty"`
	DecoderID           artifactstore.DecoderID      `json:"decoderID,omitempty"`
	State               OccurrenceState              `json:"state"`
	Diagnostics         []artifactstore.Diagnostic   `json:"diagnostics,omitempty"`
	ObservedAt          time.Time                    `json:"observedAt"`
}

func (o Occurrence) Validate() error {
	if err := artifactstore.ValidateRootID(o.RootID); err != nil {
		return err
	}
	if err := o.Key.Validate(); err != nil {
		return err
	}
	switch o.State {
	case OccurrenceValid:
		if err := artifactstore.ValidateArtifactKind(o.Kind); err != nil {
			return err
		}
		if err := artifactstore.ValidateLogicalName(o.LogicalName); err != nil {
			return err
		}
		if err := artifactstore.ValidateLogicalVersion(o.LogicalVersion, true); err != nil {
			return err
		}
		if o.DefinitionDigest == nil || o.SourceContentDigest == nil {
			return fmt.Errorf(
				"%w: valid occurrence requires definition and source content digests",
				artifactstore.ErrInvalid,
			)
		}
		if err := artifactstore.ValidateDigest(*o.DefinitionDigest); err != nil {
			return err
		}
		if err := artifactstore.ValidateDigest(*o.SourceContentDigest); err != nil {
			return err
		}
		if err := artifactstore.ValidateDecoderID(o.DecoderID); err != nil {
			return err
		}

	case OccurrenceInvalid, OccurrenceMissing:
		if o.DefinitionDigest != nil {
			if err := artifactstore.ValidateDigest(*o.DefinitionDigest); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf(
			"%w: invalid occurrence state %q",
			artifactstore.ErrInvalid,
			o.State,
		)
	}
	if err := artifactstore.ValidateDiagnostics(o.Diagnostics); err != nil {
		return err
	}
	if o.ObservedAt.IsZero() {
		return fmt.Errorf("%w: occurrence observed time is required", artifactstore.ErrInvalid)
	}
	return nil
}

type Snapshot struct {
	RootID            artifactstore.RootID              `json:"rootID"`
	Revision          uint64                            `json:"revision"`
	RootRevision      uint64                            `json:"rootRevision"`
	SourceRevisions   map[artifactstore.SourceID]uint64 `json:"sourceRevisions"`
	SourceGenerations map[artifactstore.SourceID]string `json:"sourceGenerations"`
	PublishedAt       time.Time                         `json:"publishedAt"`
	Diagnostics       []artifactstore.Diagnostic        `json:"diagnostics,omitempty"`
	Occurrences       []Occurrence                      `json:"occurrences"`
}

func (s Snapshot) Validate() error {
	if err := artifactstore.ValidateRootID(s.RootID); err != nil {
		return err
	}
	if s.Revision == 0 || s.RootRevision == 0 {
		return fmt.Errorf("%w: catalog revisions must be positive", artifactstore.ErrInvalid)
	}
	for sourceID, revision := range s.SourceRevisions {
		if err := artifactstore.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if revision == 0 {
			return fmt.Errorf("%w: source revision must be positive", artifactstore.ErrInvalid)
		}
	}
	for sourceID, generation := range s.SourceGenerations {
		if err := artifactstore.ValidateSourceID(sourceID); err != nil {
			return err
		}
		if generation == "" {
			return fmt.Errorf("%w: source generation is empty", artifactstore.ErrInvalid)
		}
	}
	if s.PublishedAt.IsZero() {
		return fmt.Errorf("%w: catalog publication time is required", artifactstore.ErrInvalid)
	}
	if err := artifactstore.ValidateDiagnostics(s.Diagnostics); err != nil {
		return err
	}
	for index, occurrence := range s.Occurrences {
		if occurrence.RootID != s.RootID {
			return fmt.Errorf(
				"%w: occurrence %d belongs to another root",
				artifactstore.ErrInvalid,
				index,
			)
		}
		if err := occurrence.Validate(); err != nil {
			return fmt.Errorf("occurrence %d: %w", index, err)
		}
	}
	return nil
}

func SortOccurrences(values []Occurrence) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Key.SourceID != values[right].Key.SourceID {
			return values[left].Key.SourceID < values[right].Key.SourceID
		}
		if values[left].Key.Locator != values[right].Key.Locator {
			return values[left].Key.Locator < values[right].Key.Locator
		}
		return values[left].Key.SubresourceLocator <
			values[right].Key.SubresourceLocator
	})
}
