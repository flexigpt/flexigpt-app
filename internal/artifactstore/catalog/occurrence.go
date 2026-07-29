package catalog

import (
	"fmt"
	"sort"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type OccurrenceState string

const (
	OccurrenceValid   OccurrenceState = "valid"
	OccurrenceInvalid OccurrenceState = "invalid"
	OccurrenceMissing OccurrenceState = "missing"
)

type OccurrenceKey struct {
	CollectionID       basespec.CollectionID       `json:"collectionID"`
	SourceID           basespec.SourceID           `json:"sourceID"`
	Locator            basespec.Locator            `json:"locator"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
}

type Occurrence struct {
	RootID              basespec.RootID         `json:"rootID"`
	CollectionID        basespec.CollectionID   `json:"collectionID"`
	Key                 OccurrenceKey           `json:"key"`
	Kind                basespec.ArtifactKind   `json:"kind,omitempty"`
	LogicalName         basespec.LogicalName    `json:"logicalName,omitempty"`
	LogicalVersion      basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DefinitionDigest    *cryptoutil.Digest      `json:"definitionDigest,omitempty"`
	SourceContentDigest *cryptoutil.Digest      `json:"sourceContentDigest,omitempty"`
	DecoderID           basespec.DecoderID      `json:"decoderID,omitempty"`
	State               OccurrenceState         `json:"state"`
	Diagnostics         []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
	ObservedAt          time.Time               `json:"observedAt"`
}

func (o Occurrence) Validate() error {
	if err := basespec.ValidateRootID(o.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionID(o.CollectionID); err != nil {
		return err
	}
	if o.Key.CollectionID != o.CollectionID {
		return fmt.Errorf("%w: occurrence key collection mismatch", basespec.ErrInvalid)
	}
	if err := o.Key.Validate(); err != nil {
		return err
	}
	if o.Kind != "" {
		if err := basespec.ValidateArtifactKind(o.Kind); err != nil {
			return err
		}
	}
	if o.LogicalName != "" {
		if err := basespec.ValidateLogicalName(o.LogicalName); err != nil {
			return err
		}
	}
	if err := basespec.ValidateLogicalVersion(o.LogicalVersion, true); err != nil {
		return err
	}
	if o.DefinitionDigest != nil {
		if err := cryptoutil.ValidateDigest(*o.DefinitionDigest); err != nil {
			return err
		}
	}
	if o.SourceContentDigest != nil {
		if err := cryptoutil.ValidateDigest(*o.SourceContentDigest); err != nil {
			return err
		}
	}
	if o.DecoderID != "" {
		if err := basespec.ValidateDecoderID(o.DecoderID); err != nil {
			return err
		}
	}
	switch o.State {
	case OccurrenceValid:
		if err := basespec.ValidateArtifactKind(o.Kind); err != nil {
			return err
		}
		if err := basespec.ValidateLogicalName(o.LogicalName); err != nil {
			return err
		}
		if err := basespec.ValidateLogicalVersion(o.LogicalVersion, true); err != nil {
			return err
		}
		if o.DefinitionDigest == nil || o.SourceContentDigest == nil {
			return fmt.Errorf(
				"%w: valid occurrence requires definition and source content digests",
				basespec.ErrInvalid,
			)
		}
		if err := cryptoutil.ValidateDigest(*o.DefinitionDigest); err != nil {
			return err
		}
		if err := cryptoutil.ValidateDigest(*o.SourceContentDigest); err != nil {
			return err
		}
		if err := basespec.ValidateDecoderID(o.DecoderID); err != nil {
			return err
		}

	case OccurrenceInvalid, OccurrenceMissing:
	default:
		return fmt.Errorf(
			"%w: invalid occurrence state %q",
			basespec.ErrInvalid,
			o.State,
		)
	}
	if err := diagnostic.ValidateDiagnostics(o.Diagnostics); err != nil {
		return err
	}
	if o.ObservedAt.IsZero() {
		return fmt.Errorf("%w: occurrence observed time is required", basespec.ErrInvalid)
	}
	return nil
}

func (k OccurrenceKey) Validate() error {
	if err := basespec.ValidateCollectionID(k.CollectionID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(k.SourceID); err != nil {
		return err
	}
	if err := basespec.ValidateLocator(k.Locator, false); err != nil {
		return err
	}
	return basespec.ValidateSubresourceLocator(k.SubresourceLocator)
}

func SortOccurrences(values []Occurrence) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Key.CollectionID != values[right].Key.CollectionID {
			return values[left].Key.CollectionID < values[right].Key.CollectionID
		}
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
