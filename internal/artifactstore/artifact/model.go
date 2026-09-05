package artifact

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// AdoptRequest identifies one currently catalogued valid occurrence that a
// caller explicitly wants to represent as an observed Artifact.
type AdoptRequest struct {
	ArtifactID              basespec.ArtifactID
	Collection              collection.CollectionRef
	Occurrence              catalog.OccurrenceKey
	ExpectedCatalogRevision uint64
	Name                    string
	Enabled                 bool
	Data                    json.RawMessage
}

// PinRequest creates an Artifact binding that remains represented even when
// its source occurrence is missing or temporarily invalid.
type PinRequest struct {
	ArtifactID                 basespec.ArtifactID
	Collection                 collection.CollectionRef
	ExpectedCollectionRevision uint64
	Binding                    SourceBinding
	Name                       string
	Enabled                    bool
	Data                       json.RawMessage
}

// SuppressRequest records that automatic reconciliation must not create an
// observed Artifact for one attached source binding.
type SuppressRequest struct {
	Collection                 collection.CollectionRef
	ExpectedCollectionRevision uint64
	Binding                    SourceBinding
}

type State string

const (
	StateAvailable    State = "available"
	StateMissing      State = "missing"
	StateInvalid      State = "invalid"
	StateIncompatible State = "incompatible"
)

type AdoptionMode string

const (
	AdoptionObserved AdoptionMode = "observed"
	AdoptionPinned   AdoptionMode = "pinned"
)

type ArtifactRef struct {
	RootID     basespec.RootID     `json:"rootID"`
	ArtifactID basespec.ArtifactID `json:"artifactID"`
}

func (r ArtifactRef) Validate() error {
	if err := basespec.ValidateRootID(r.RootID); err != nil {
		return err
	}
	return basespec.ValidateArtifactID(r.ArtifactID)
}

type ArtifactAddress struct {
	RootID       basespec.RootID       `json:"rootID"`
	CollectionID basespec.CollectionID `json:"collectionID"`
	ArtifactID   basespec.ArtifactID   `json:"artifactID"`
	Kind         basespec.ArtifactKind `json:"kind"`
}

func (a ArtifactAddress) Validate() error {
	if err := basespec.ValidateRootID(a.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionID(a.CollectionID); err != nil {
		return err
	}
	if err := basespec.ValidateArtifactID(a.ArtifactID); err != nil {
		return err
	}
	return basespec.ValidateArtifactKind(a.Kind)
}

type SourceBinding struct {
	SourceID           basespec.SourceID           `json:"sourceID"`
	Locator            basespec.Locator            `json:"locator"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
	ExpectedKind       basespec.ArtifactKind       `json:"expectedKind"`
}

type Artifact struct {
	ID                 basespec.ArtifactID   `json:"id"`
	RootID             basespec.RootID       `json:"rootID"`
	CollectionID       basespec.CollectionID `json:"collectionID"`
	Binding            SourceBinding         `json:"binding"`
	Kind               basespec.ArtifactKind `json:"kind"`
	Name               string                `json:"name"`
	Enabled            bool                  `json:"enabled"`
	Adoption           AdoptionMode          `json:"adoption"`
	ResolvedDefinition *cryptoutil.Digest    `json:"resolvedDefinition,omitempty"`
	Data               json.RawMessage       `json:"-"`

	State       State                    `json:"state"`
	Diagnostics []providerapi.Diagnostic `json:"diagnostics,omitempty"`
	Revision    uint64                   `json:"revision"`
	CreatedAt   time.Time                `json:"createdAt"`
	ModifiedAt  time.Time                `json:"modifiedAt"`
}

func (a Artifact) Ref() ArtifactRef {
	return ArtifactRef{
		RootID:     a.RootID,
		ArtifactID: a.ID,
	}
}

func (a Artifact) Address() ArtifactAddress {
	return ArtifactAddress{
		RootID:       a.RootID,
		CollectionID: a.CollectionID,
		ArtifactID:   a.ID,
		Kind:         a.Kind,
	}
}

func (a ArtifactAddress) CollectionRef() collection.CollectionRef {
	return collection.CollectionRef{
		RootID:       a.RootID,
		CollectionID: a.CollectionID,
	}
}

func (a Artifact) Validate() error {
	if err := basespec.ValidateArtifactID(a.ID); err != nil {
		return err
	}
	if err := basespec.ValidateRootID(a.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionID(a.CollectionID); err != nil {
		return err
	}
	if err := a.Binding.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateArtifactKind(a.Kind); err != nil {
		return err
	}
	if a.Binding.ExpectedKind != a.Kind {
		return fmt.Errorf(
			"%w: artifact binding expected kind does not match artifact kind",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateRequiredText(
		"artifact name",
		a.Name,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	switch a.Adoption {
	case AdoptionObserved, AdoptionPinned:
	default:
		return fmt.Errorf(
			"%w: invalid artifact adoption mode %q",
			basespec.ErrInvalid,
			a.Adoption,
		)
	}
	if err := ValidateSourceState(a.State, a.ResolvedDefinition); err != nil {
		return err
	}
	if _, err := jsonutil.CanonicalizeObject(
		a.Data,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf("%w: artifact data: %w", basespec.ErrInvalid, err)
	}
	if err := providerapi.ValidateDiagnostics(a.Diagnostics); err != nil {
		return err
	}
	if a.Revision == 0 {
		return fmt.Errorf(
			"%w: artifact revision must be positive",
			basespec.ErrInvalid,
		)
	}
	if a.CreatedAt.IsZero() || a.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: artifact timestamps are required",
			basespec.ErrInvalid,
		)
	}
	if a.ModifiedAt.Before(a.CreatedAt) {
		return fmt.Errorf(
			"%w: artifact modified time precedes creation",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (a Artifact) Clone() Artifact {
	output := a
	output.Data = append(json.RawMessage(nil), a.Data...)
	output.Diagnostics = providerapi.CloneDiagnostics(a.Diagnostics)
	if a.ResolvedDefinition != nil {
		value := *a.ResolvedDefinition
		output.ResolvedDefinition = &value
	}
	return output
}

type Suppression struct {
	RootID       basespec.RootID       `json:"rootID"`
	CollectionID basespec.CollectionID `json:"collectionID"`
	Binding      SourceBinding         `json:"binding"`
	Revision     uint64                `json:"revision"`
	CreatedAt    time.Time             `json:"createdAt"`
	ModifiedAt   time.Time             `json:"modifiedAt"`
}

func (s Suppression) Validate() error {
	if err := basespec.ValidateRootID(s.RootID); err != nil {
		return err
	}
	if err := basespec.ValidateCollectionID(s.CollectionID); err != nil {
		return err
	}
	if err := s.Binding.Validate(); err != nil {
		return err
	}
	if s.Revision == 0 {
		return fmt.Errorf(
			"%w: suppression revision must be positive",
			basespec.ErrInvalid,
		)
	}
	if s.CreatedAt.IsZero() || s.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: suppression timestamps are required",
			basespec.ErrInvalid,
		)
	}

	if s.ModifiedAt.Before(s.CreatedAt) {
		return fmt.Errorf(
			"%w: suppression modified time precedes creation",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (b SourceBinding) Validate() error {
	if err := basespec.ValidateSourceID(b.SourceID); err != nil {
		return err
	}
	if err := basespec.ValidateLocator(b.Locator, true); err != nil {
		return err
	}
	if err := basespec.ValidateSubresourceLocator(b.SubresourceLocator); err != nil {
		return err
	}
	return basespec.ValidateArtifactKind(b.ExpectedKind)
}

func ValidateSourceState(
	state State,
	resolvedDefinition *cryptoutil.Digest,
) error {
	if resolvedDefinition != nil {
		if err := cryptoutil.ValidateDigest(*resolvedDefinition); err != nil {
			return err
		}
	}

	switch state {
	case StateAvailable, StateIncompatible:
		if resolvedDefinition == nil {
			return fmt.Errorf(
				"%w: artifact state %q requires a resolved definition",
				basespec.ErrInvalid,
				state,
			)
		}

	case StateMissing, StateInvalid:
		if resolvedDefinition != nil {
			return fmt.Errorf(
				"%w: artifact state %q cannot retain a resolved definition",
				basespec.ErrInvalid,
				state,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid artifact state %q",
			basespec.ErrInvalid,
			state,
		)
	}
	return nil
}
