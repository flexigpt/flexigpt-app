package artifactimpl

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// SourceStateUpdate carries only refresh-owned Artifact fields.
//
// Local DisplayName, Enabled, and Data are intentionally absent so refresh
// cannot overwrite consumer-controlled local state.
type SourceStateUpdate struct {
	ArtifactID artifact.ArtifactID `json:"artifactID"`
	RootID     root.RootID         `json:"rootID"`

	Binding        artifact.SourceBinding  `json:"binding"`
	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`

	ResolvedDefinition  *cryptoutil.Digest      `json:"resolvedDefinition,omitempty"`
	SourceContentDigest *cryptoutil.Digest      `json:"sourceContentDigest,omitempty"`
	State               artifact.State          `json:"state"`
	Diagnostics         []diagnostic.Diagnostic `json:"diagnostics,omitempty"`

	Revision         uint64    `json:"revision"`
	ModifiedAt       time.Time `json:"modifiedAt"`
	ExpectedRevision uint64    `json:"expectedRevision"`
}

func (u SourceStateUpdate) Validate() error {
	if err := u.ArtifactID.Validate(); err != nil {
		return err
	}
	if err := u.RootID.Validate(); err != nil {
		return err
	}
	if err := u.Binding.Validate(); err != nil {
		return err
	}
	if err := u.LogicalName.Validate(); err != nil {
		return err
	}
	if err := u.LogicalVersion.Validate(true); err != nil {
		return err
	}
	if err := u.State.Validate(
		u.ResolvedDefinition,
		u.SourceContentDigest,
	); err != nil {
		return err
	}
	if err := diagnostic.Validate(u.Diagnostics); err != nil {
		return err
	}
	if u.ExpectedRevision == 0 ||
		u.Revision != u.ExpectedRevision+1 ||
		u.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: invalid source-derived Artifact update",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func (u SourceStateUpdate) Clone() SourceStateUpdate {
	output := u
	output.ResolvedDefinition = cryptoutil.CloneDigest(
		u.ResolvedDefinition,
	)
	output.SourceContentDigest = cryptoutil.CloneDigest(
		u.SourceContentDigest,
	)
	output.Diagnostics = diagnostic.Clone(u.Diagnostics)
	return output
}

type Synchronization struct {
	Creates     []artifact.Artifact
	Updates     []SourceStateUpdate
	Diagnostics []diagnostic.Diagnostic
}

func (s Synchronization) Clone() Synchronization {
	output := s
	output.Creates = make([]artifact.Artifact, len(s.Creates))
	for index, value := range s.Creates {
		output.Creates[index] = value.Clone()
	}
	output.Updates = make([]SourceStateUpdate, len(s.Updates))
	for index, value := range s.Updates {
		output.Updates[index] = value.Clone()
	}
	output.Diagnostics = diagnostic.Clone(s.Diagnostics)
	return output
}
