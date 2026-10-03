package model

import (
	"fmt"
	"time"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// State records one successfully published Source refresh.
type State struct {
	RootID               rootModel.RootID        `json:"rootID"`
	SourceID             sourceModel.SourceID    `json:"sourceID"`
	SourceRevision       uint64                  `json:"sourceRevision"`
	SourceGeneration     string                  `json:"sourceGeneration"`
	DiscoveryFingerprint cryptoutil.Digest       `json:"discoveryFingerprint"`
	DecoderFingerprint   cryptoutil.Digest       `json:"decoderFingerprint"`
	Revision             uint64                  `json:"revision"`
	RefreshedAt          time.Time               `json:"refreshedAt"`
	Diagnostics          []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
}

func (s State) Validate() error {
	if err := s.RootID.Validate(); err != nil {
		return err
	}
	if err := s.SourceID.Validate(); err != nil {
		return err
	}
	if s.SourceRevision == 0 || s.Revision == 0 {
		return fmt.Errorf(
			"%w: source refresh revisions must be positive",
			spec.ErrInvalid,
		)
	}
	if err := spec.ValidateSourceGeneration(s.SourceGeneration); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(s.DiscoveryFingerprint); err != nil {
		return fmt.Errorf(
			"source discovery fingerprint: %w",
			err,
		)
	}
	if err := cryptoutil.ValidateDigest(s.DecoderFingerprint); err != nil {
		return fmt.Errorf(
			"source decoder fingerprint: %w",
			err,
		)
	}
	if s.RefreshedAt.IsZero() {
		return fmt.Errorf(
			"%w: Source refresh time is required",
			spec.ErrInvalid,
		)
	}
	return diagnostic.Validate(s.Diagnostics)
}

func (s State) Clone() State {
	output := s
	output.Diagnostics = diagnostic.Clone(s.Diagnostics)
	return output
}
