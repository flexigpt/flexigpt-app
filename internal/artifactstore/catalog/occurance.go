package catalog

import (
	"fmt"
	"sort"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

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
