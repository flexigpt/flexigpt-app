package artifact

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

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

type Artifact struct {
	ID                 artifactstore.ArtifactID    `json:"id"`
	RootID             artifactstore.RootID        `json:"rootID"`
	CollectionID       artifactstore.CollectionID  `json:"collectionID"`
	Binding            artifactstore.SourceBinding `json:"binding"`
	Kind               artifactstore.ArtifactKind  `json:"kind"`
	Name               string                      `json:"name"`
	Enabled            bool                        `json:"enabled"`
	Adoption           AdoptionMode                `json:"adoption"`
	ResolvedDefinition *artifactstore.Digest       `json:"resolvedDefinition,omitempty"`
	Data               json.RawMessage             `json:"-"`
	State              State                       `json:"state"`
	Diagnostics        []artifactstore.Diagnostic  `json:"diagnostics,omitempty"`
	Revision           uint64                      `json:"revision"`
	CreatedAt          time.Time                   `json:"createdAt"`
	ModifiedAt         time.Time                   `json:"modifiedAt"`
}

func (a Artifact) Ref() artifactstore.ArtifactRef {
	return artifactstore.ArtifactRef{
		RootID:     a.RootID,
		ArtifactID: a.ID,
	}
}

func (a Artifact) Address() artifactstore.ArtifactAddress {
	return artifactstore.ArtifactAddress{
		RootID:       a.RootID,
		CollectionID: a.CollectionID,
		ArtifactID:   a.ID,
		Kind:         a.Kind,
	}
}

func (a Artifact) Validate() error {
	if err := artifactstore.ValidateArtifactID(a.ID); err != nil {
		return err
	}
	if err := artifactstore.ValidateRootID(a.RootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateCollectionID(a.CollectionID); err != nil {
		return err
	}
	if err := a.Binding.Validate(); err != nil {
		return err
	}
	if err := artifactstore.ValidateArtifactKind(a.Kind); err != nil {
		return err
	}
	if a.Binding.ExpectedKind != a.Kind {
		return fmt.Errorf(
			"%w: artifact binding expected kind does not match artifact kind",
			artifactstore.ErrInvalid,
		)
	}
	if err := artifactstore.ValidateRequiredText(
		"artifact name",
		a.Name,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	switch a.Adoption {
	case AdoptionObserved, AdoptionPinned:
	default:
		return fmt.Errorf(
			"%w: invalid artifact adoption mode %q",
			artifactstore.ErrInvalid,
			a.Adoption,
		)
	}
	if a.ResolvedDefinition != nil {
		if err := artifactstore.ValidateDigest(*a.ResolvedDefinition); err != nil {
			return err
		}
	}
	switch a.State {
	case StateAvailable, StateIncompatible:
		if a.ResolvedDefinition == nil {
			return fmt.Errorf(
				"%w: artifact state %q requires a resolved definition",
				artifactstore.ErrInvalid,
				a.State,
			)
		}

	case StateMissing, StateInvalid:
		if a.ResolvedDefinition != nil {
			return fmt.Errorf(
				"%w: artifact state %q cannot retain a resolved definition",
				artifactstore.ErrInvalid,
				a.State,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid artifact state %q",
			artifactstore.ErrInvalid,
			a.State,
		)
	}
	if _, err := jsoncanon.CanonicalizeObject(
		a.Data,
		artifactstore.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf("%w: artifact data: %w", artifactstore.ErrInvalid, err)
	}
	if err := artifactstore.ValidateDiagnostics(a.Diagnostics); err != nil {
		return err
	}
	if a.Revision == 0 {
		return fmt.Errorf(
			"%w: artifact revision must be positive",
			artifactstore.ErrInvalid,
		)
	}
	if a.CreatedAt.IsZero() || a.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: artifact timestamps are required",
			artifactstore.ErrInvalid,
		)
	}
	if a.ModifiedAt.Before(a.CreatedAt) {
		return fmt.Errorf(
			"%w: artifact modified time precedes creation",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}

func (a Artifact) Clone() Artifact {
	output := a
	output.Data = append(json.RawMessage(nil), a.Data...)
	output.Diagnostics = artifactstore.CloneDiagnostics(a.Diagnostics)
	if a.ResolvedDefinition != nil {
		value := *a.ResolvedDefinition
		output.ResolvedDefinition = &value
	}
	return output
}

type Suppression struct {
	RootID       artifactstore.RootID        `json:"rootID"`
	CollectionID artifactstore.CollectionID  `json:"collectionID"`
	Binding      artifactstore.SourceBinding `json:"binding"`
	Revision     uint64                      `json:"revision"`
	CreatedAt    time.Time                   `json:"createdAt"`
	ModifiedAt   time.Time                   `json:"modifiedAt"`
}

func (s Suppression) Validate() error {
	if err := artifactstore.ValidateRootID(s.RootID); err != nil {
		return err
	}
	if err := artifactstore.ValidateCollectionID(s.CollectionID); err != nil {
		return err
	}
	if err := s.Binding.Validate(); err != nil {
		return err
	}
	if s.Revision == 0 {
		return fmt.Errorf(
			"%w: suppression revision must be positive",
			artifactstore.ErrInvalid,
		)
	}
	if s.CreatedAt.IsZero() || s.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: suppression timestamps are required",
			artifactstore.ErrInvalid,
		)
	}

	if s.ModifiedAt.Before(s.CreatedAt) {
		return fmt.Errorf(
			"%w: suppression modified time precedes creation",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}
