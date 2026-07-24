package record

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type State string

const (
	StateAvailable    State = "available"
	StateMissing      State = "missing"
	StateInvalid      State = "invalid"
	StateIncompatible State = "incompatible"
)

type Record struct {
	ID                 artifactstore.RecordID     `json:"id"`
	RootID             artifactstore.RootID       `json:"rootID"`
	Occurrence         catalog.OccurrenceKey      `json:"occurrence"`
	Kind               artifactstore.ArtifactKind `json:"kind"`
	Name               string                     `json:"name"`
	Enabled            bool                       `json:"enabled"`
	ResolvedDefinition *artifactstore.Digest      `json:"resolvedDefinition,omitempty"`
	Data               json.RawMessage            `json:"data"`
	State              State                      `json:"state"`
	Diagnostics        []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
	Revision           uint64                     `json:"revision"`
	CreatedAt          time.Time                  `json:"createdAt"`
	ModifiedAt         time.Time                  `json:"modifiedAt"`
}

func (r Record) Validate() error {
	if err := artifactstore.ValidateRecordID(r.ID); err != nil {
		return err
	}
	if err := artifactstore.ValidateRootID(r.RootID); err != nil {
		return err
	}
	if err := r.Occurrence.Validate(); err != nil {
		return err
	}
	if err := artifactstore.ValidateArtifactKind(r.Kind); err != nil {
		return err
	}
	if err := artifactstore.ValidateRequiredText(
		"record name",
		r.Name,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if r.ResolvedDefinition != nil {
		if err := artifactstore.ValidateDigest(*r.ResolvedDefinition); err != nil {
			return err
		}
	}
	if err := validateState(r.State, r.ResolvedDefinition); err != nil {
		return err
	}
	if _, err := jsoncanon.CanonicalizeObject(
		r.Data,
		artifactstore.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf("%w: record data: %w", artifactstore.ErrInvalid, err)
	}
	if err := artifactstore.ValidateDiagnostics(r.Diagnostics); err != nil {
		return err
	}
	if r.Revision == 0 {
		return fmt.Errorf("%w: record revision must be positive", artifactstore.ErrInvalid)
	}
	if r.CreatedAt.IsZero() || r.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: record timestamps are required", artifactstore.ErrInvalid)
	}
	if r.ModifiedAt.Before(r.CreatedAt) {
		return fmt.Errorf(
			"%w: record modified time precedes creation",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}

func validateState(
	state State,
	resolvedDefinition *artifactstore.Digest,
) error {
	switch state {
	case StateAvailable, StateIncompatible:
		if resolvedDefinition == nil {
			return fmt.Errorf(
				"%w: record state %q requires a resolved definition",
				artifactstore.ErrInvalid,
				state,
			)
		}
	case StateMissing, StateInvalid:
	default:
		return fmt.Errorf("%w: invalid record state %q", artifactstore.ErrInvalid, state)
	}
	return nil
}
