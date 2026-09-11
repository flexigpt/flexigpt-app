package catalog

import (
	"fmt"
	"sort"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type OccurrenceState string

const (
	OccurrenceValid   OccurrenceState = "valid"
	OccurrenceInvalid OccurrenceState = "invalid"
	OccurrenceMissing OccurrenceState = "missing"
)

type OccurrenceKey struct {
	CollectionID       collection.CollectionID     `json:"collectionID"`
	SourceID           source.SourceID             `json:"sourceID"`
	Locator            basespec.Locator            `json:"locator"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
}

type Occurrence struct {
	RootID              root.RootID             `json:"rootID"`
	CollectionID        collection.CollectionID `json:"collectionID"`
	Key                 OccurrenceKey           `json:"key"`
	Kind                artifact.ArtifactKind   `json:"kind,omitempty"`
	LogicalName         basespec.LogicalName    `json:"logicalName,omitempty"`
	LogicalVersion      basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DefinitionDigest    *cryptoutil.Digest      `json:"definitionDigest,omitempty"`
	SourceContentDigest *cryptoutil.Digest      `json:"sourceContentDigest,omitempty"`
	DecoderID           basespec.DecoderID      `json:"decoderID,omitempty"`
	State               OccurrenceState         `json:"state"`
	Diagnostics         []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
	ObservedAt          time.Time               `json:"observedAt"`

	// Definition is the current parsed definition cache. The portable source
	// package remains authoritative. SQLite persists this field with the
	// current catalog occurrence, but it is intentionally not part of the
	// public catalog JSON projection.
	Definition *definition.Definition `json:"-"`
}

func (o Occurrence) Validate() error {
	if err := o.RootID.Validate(); err != nil {
		return err
	}
	if err := o.CollectionID.Validate(); err != nil {
		return err
	}
	if o.Key.CollectionID != o.CollectionID {
		return fmt.Errorf("%w: occurrence key collection mismatch", basespec.ErrInvalid)
	}
	if err := o.Key.Validate(); err != nil {
		return err
	}
	if o.Kind != "" {
		if err := o.Kind.Validate(); err != nil {
			return err
		}
	}
	if o.LogicalName != "" {
		if err := o.LogicalName.Validate(); err != nil {
			return err
		}
	}
	if err := o.LogicalVersion.Validate(true); err != nil {
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
		if err := o.DecoderID.Validate(); err != nil {
			return err
		}
	}
	if o.Definition != nil {
		canonical, err := definition.Canonicalize(*o.Definition)
		if err != nil {
			return fmt.Errorf("occurrence definition: %w", err)
		}
		if canonical.Digest != o.Definition.Digest {
			return fmt.Errorf(
				"%w: occurrence definition is not canonical",
				basespec.ErrInvalid,
			)
		}
		if o.DefinitionDigest == nil ||
			canonical.Digest != *o.DefinitionDigest {
			return fmt.Errorf(
				"%w: occurrence definition does not match fingerprint",
				basespec.ErrDigestMismatch,
			)
		}
	}

	switch o.State {
	case OccurrenceValid:
		if o.Kind == "" {
			return o.Kind.Validate()
		}
		if o.LogicalName == "" {
			return o.LogicalName.Validate()
		}
		if o.DefinitionDigest == nil ||
			o.SourceContentDigest == nil ||
			o.Definition == nil {
			return fmt.Errorf(
				"%w: valid occurrence requires definition, definition fingerprint, and source content fingerprint",
				basespec.ErrInvalid,
			)
		}
		if err := o.DecoderID.Validate(); err != nil {
			return err
		}

	case OccurrenceInvalid, OccurrenceMissing:
		if o.Definition != nil {
			return fmt.Errorf(
				"%w: non-valid occurrence cannot retain a parsed definition",
				basespec.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid occurrence state %q",
			basespec.ErrInvalid,
			o.State,
		)
	}
	if err := diagnostic.Validate(o.Diagnostics); err != nil {
		return err
	}
	if o.ObservedAt.IsZero() {
		return fmt.Errorf("%w: occurrence observed time is required", basespec.ErrInvalid)
	}
	return nil
}

func (o Occurrence) Clone() Occurrence {
	output := o
	output.DefinitionDigest = cryptoutil.CloneDigest(o.DefinitionDigest)
	output.SourceContentDigest = cryptoutil.CloneDigest(o.SourceContentDigest)
	if o.Definition != nil {
		value := o.Definition.Clone()
		output.Definition = &value
	}
	output.Diagnostics = diagnostic.Clone(o.Diagnostics)
	return output
}

func (k OccurrenceKey) Validate() error {
	if err := k.CollectionID.Validate(); err != nil {
		return err
	}
	if err := k.SourceID.Validate(); err != nil {
		return err
	}
	if err := k.Locator.Validate(false); err != nil {
		return err
	}
	return k.SubresourceLocator.Validate()
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

// EqualOccurrences compares occurrence values independently of ordering.
func EqualOccurrences(left, right []Occurrence) bool {
	if len(left) != len(right) {
		return false
	}

	byKey := make(map[OccurrenceKey]Occurrence, len(left))
	for _, value := range left {
		if _, duplicate := byKey[value.Key]; duplicate {
			return false
		}
		byKey[value.Key] = value
	}

	for _, value := range right {
		leftValue, found := byKey[value.Key]
		if !found || !equalOccurrence(leftValue, value) {
			return false
		}
	}
	return true
}

func equalOccurrence(left, right Occurrence) bool {
	if left.Definition == nil || right.Definition == nil {
		if left.Definition != nil || right.Definition != nil {
			return false
		}
	} else if left.Definition.Digest != right.Definition.Digest {
		return false
	}

	return left.RootID == right.RootID &&
		left.CollectionID == right.CollectionID &&
		left.Key == right.Key &&
		left.Kind == right.Kind &&
		left.LogicalName == right.LogicalName &&
		left.LogicalVersion == right.LogicalVersion &&
		cryptoutil.IsDigestEqual(left.DefinitionDigest, right.DefinitionDigest) &&
		cryptoutil.IsDigestEqual(left.SourceContentDigest, right.SourceContentDigest) &&
		left.DecoderID == right.DecoderID &&
		left.State == right.State &&
		left.ObservedAt.Equal(right.ObservedAt) &&
		diagnostic.Equal(left.Diagnostics, right.Diagnostics)
}
