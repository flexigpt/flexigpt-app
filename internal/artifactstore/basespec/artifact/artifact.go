package artifact

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

type (
	ArtifactID   string
	ArtifactKind string
)

func (v ArtifactID) Validate() error {
	if err := uuidutil.ValidateUUIDv7(string(v)); err != nil {
		return fmt.Errorf("Artifact ID: %w", err)
	}
	return nil
}

// Validate - ArtifactKind remains an opaque provider and contract-owned identifier.
// Artifact Store does not enumerate, interpret, or otherwise know typed
// declaration vocabulary.
func (v ArtifactKind) Validate() error {
	return basespec.ValidateIdentifier(
		"Artifact kind",
		string(v),
		basespec.MaxKindBytes,
	)
}

type ArtifactRef struct {
	RootID     root.RootID `json:"rootID"`
	ArtifactID ArtifactID  `json:"artifactID"`
}

func (r ArtifactRef) Validate() error {
	if err := r.RootID.Validate(); err != nil {
		return err
	}
	return r.ArtifactID.Validate()
}

type ArtifactAddress struct {
	RootID      root.RootID          `json:"rootID"`
	ArtifactID  ArtifactID           `json:"artifactID"`
	Kind        ArtifactKind         `json:"kind"`
	LogicalName basespec.LogicalName `json:"logicalName"`
}

func (a ArtifactAddress) Validate() error {
	if err := a.RootID.Validate(); err != nil {
		return err
	}
	if err := a.ArtifactID.Validate(); err != nil {
		return err
	}
	if err := a.Kind.Validate(); err != nil {
		return err
	}
	return a.LogicalName.Validate()
}

// Artifact is the Root-local source-backed record for one named declaration.
//
// Binding, kind, logical identity, Definition state, source content state,
// and diagnostics are synchronized from Source refresh.
//
// DisplayName, Enabled, and Data are local Store state retained across source
// refreshes. Enabled is mutable for Artifacts in both mutable and protected
// Roots. Protected Root policy continues to protect source, package, display,
// generic data, and purge mutation.
type Artifact struct {
	ID      ArtifactID    `json:"id"`
	RootID  root.RootID   `json:"rootID"`
	Binding SourceBinding `json:"binding"`

	Kind           ArtifactKind            `json:"kind"`
	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`

	ResolvedDefinition  *cryptoutil.Digest `json:"resolvedDefinition,omitempty"`
	SourceContentDigest *cryptoutil.Digest `json:"sourceContentDigest,omitempty"`

	State       State                   `json:"state"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics,omitempty"`

	DisplayName string `json:"displayName"`
	// Enabled is universal local Artifact metadata. It is not source-owned and
	// does not alter source refresh, Definition resolution, package lifecycle,
	// or Artifact lifecycle behavior.
	Enabled bool            `json:"enabled"`
	Data    json.RawMessage `json:"-"`

	Revision   uint64    `json:"revision"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

func (a Artifact) Ref() ArtifactRef {
	return ArtifactRef{
		RootID:     a.RootID,
		ArtifactID: a.ID,
	}
}

func (a Artifact) Address() ArtifactAddress {
	return ArtifactAddress{
		RootID:      a.RootID,
		ArtifactID:  a.ID,
		Kind:        a.Kind,
		LogicalName: a.LogicalName,
	}
}

func (a Artifact) Validate() error {
	if err := a.ID.Validate(); err != nil {
		return err
	}
	if err := a.RootID.Validate(); err != nil {
		return err
	}
	if err := a.Binding.Validate(); err != nil {
		return err
	}
	if err := a.Kind.Validate(); err != nil {
		return err
	}
	if err := a.LogicalName.Validate(); err != nil {
		return err
	}
	if err := a.LogicalVersion.Validate(true); err != nil {
		return err
	}
	if err := a.State.Validate(
		a.ResolvedDefinition,
		a.SourceContentDigest,
	); err != nil {
		return err
	}
	if err := diagnostic.Validate(a.Diagnostics); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"Artifact display name",
		a.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if _, err := jsonutil.CanonicalizeObject(
		a.Data,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: Artifact local data: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if a.Revision == 0 {
		return fmt.Errorf(
			"%w: Artifact revision must be positive",
			basespec.ErrInvalid,
		)
	}
	if a.CreatedAt.IsZero() || a.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: Artifact timestamps are required",
			basespec.ErrInvalid,
		)
	}
	if a.ModifiedAt.Before(a.CreatedAt) {
		return fmt.Errorf(
			"%w: Artifact modified time precedes creation",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (a Artifact) Clone() Artifact {
	output := a
	output.ResolvedDefinition = cryptoutil.CloneDigest(
		a.ResolvedDefinition,
	)
	output.SourceContentDigest = cryptoutil.CloneDigest(
		a.SourceContentDigest,
	)
	output.Diagnostics = diagnostic.Clone(a.Diagnostics)
	output.Data = append(json.RawMessage(nil), a.Data...)
	return output
}
