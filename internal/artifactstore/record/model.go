package record

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Mode string

const (
	ModeLinked Mode = "linked"
	ModePinned Mode = "pinned"
)

type State string

const (
	StateAvailable    State = "available"
	StateStale        State = "stale"
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
	Mode               Mode                       `json:"mode"`
	PinnedDefinition   *artifactstore.Digest      `json:"pinnedDefinition,omitempty"`
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
	switch r.Mode {
	case ModeLinked:
		if r.PinnedDefinition != nil {
			return fmt.Errorf(
				"%w: linked record cannot have a pinned definition",
				artifactstore.ErrInvalid,
			)
		}
	case ModePinned:
		if r.PinnedDefinition == nil {
			return fmt.Errorf(
				"%w: pinned record requires a pinned definition",
				artifactstore.ErrInvalid,
			)
		}
		if err := artifactstore.ValidateDigest(*r.PinnedDefinition); err != nil {
			return err
		}
		if r.ResolvedDefinition == nil ||
			*r.ResolvedDefinition != *r.PinnedDefinition {
			return fmt.Errorf(
				"%w: pinned record must resolve to its pinned definition",
				artifactstore.ErrInvalid,
			)
		}
	default:
		return fmt.Errorf("%w: invalid record mode %q", artifactstore.ErrInvalid, r.Mode)
	}
	if r.ResolvedDefinition != nil {
		if err := artifactstore.ValidateDigest(*r.ResolvedDefinition); err != nil {
			return err
		}
	}
	switch r.State {
	case StateAvailable, StateStale, StateIncompatible:
		if r.ResolvedDefinition == nil {
			return fmt.Errorf(
				"%w: record state %q requires a resolved definition",
				artifactstore.ErrInvalid,
				r.State,
			)
		}
	case StateMissing, StateInvalid:
	default:
		return fmt.Errorf("%w: invalid record state %q", artifactstore.ErrInvalid, r.State)
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
	return nil
}

func TypedOccurrenceKey(
	rootID artifactstore.RootID,
	key catalog.OccurrenceKey,
	kind artifactstore.ArtifactKind,
) string {
	return string(rootID) + "\x00" +
		key.String() + "\x00" +
		string(kind)
}
