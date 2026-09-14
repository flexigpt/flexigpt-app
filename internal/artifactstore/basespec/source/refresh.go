package source

import (
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// RefreshState records one successfully published Source scan.
//
// It is Source freshness bookkeeping only. It does not own Artifacts,
// Definitions, typed relationships, or consumer graph state.
type RefreshState struct {
	RootID               root.RootID             `json:"rootID"`
	SourceID             SourceID                `json:"sourceID"`
	SourceRevision       uint64                  `json:"sourceRevision"`
	SourceGeneration     string                  `json:"sourceGeneration"`
	DiscoveryFingerprint cryptoutil.Digest       `json:"discoveryFingerprint"`
	DecoderFingerprint   cryptoutil.Digest       `json:"decoderFingerprint"`
	Revision             uint64                  `json:"revision"`
	RefreshedAt          time.Time               `json:"refreshedAt"`
	Diagnostics          []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
}

func (s RefreshState) Validate() error {
	if err := s.RootID.Validate(); err != nil {
		return err
	}
	if err := s.SourceID.Validate(); err != nil {
		return err
	}
	if s.SourceRevision == 0 || s.Revision == 0 {
		return fmt.Errorf(
			"%w: Source refresh revisions must be positive",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateSourceGeneration(
		s.SourceGeneration,
	); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(
		s.DiscoveryFingerprint,
	); err != nil {
		return fmt.Errorf(
			"Source discovery fingerprint: %w",
			err,
		)
	}
	if err := cryptoutil.ValidateDigest(
		s.DecoderFingerprint,
	); err != nil {
		return fmt.Errorf(
			"Source decoder fingerprint: %w",
			err,
		)
	}
	if s.RefreshedAt.IsZero() {
		return fmt.Errorf(
			"%w: Source refresh time is required",
			basespec.ErrInvalid,
		)
	}
	return diagnostic.Validate(s.Diagnostics)
}

func (s RefreshState) Clone() RefreshState {
	output := s
	output.Diagnostics = diagnostic.Clone(s.Diagnostics)
	return output
}

type RefreshInspection struct {
	State                   RefreshState `json:"state"`
	SourceRevisionChanged   bool         `json:"sourceRevisionChanged"`
	DiscoveryChanged        bool         `json:"discoveryChanged"`
	DecoderChanged          bool         `json:"decoderChanged"`
	SourceGenerationChanged bool         `json:"sourceGenerationChanged"`
}

func (i RefreshInspection) Validate() error {
	return i.State.Validate()
}

func (i RefreshInspection) Clone() RefreshInspection {
	output := i
	output.State = i.State.Clone()
	return output
}

func (i RefreshInspection) IsCurrent() bool {
	return !i.SourceRevisionChanged &&
		!i.DiscoveryChanged &&
		!i.DecoderChanged &&
		!i.SourceGenerationChanged
}
