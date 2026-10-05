package model

import (
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// PhysicalPackageState records the observed source-side publication state.
// It does not claim that metadata publication succeeded.
type PhysicalPackageState string

const (
	PhysicalPackageUnknown   PhysicalPackageState = ""
	PhysicalPackageUnchanged PhysicalPackageState = "unchanged"
	PhysicalPackageChanged   PhysicalPackageState = "changed"
)

func (s PhysicalPackageState) Validate() error {
	switch s {
	case PhysicalPackageUnknown,
		PhysicalPackageUnchanged,
		PhysicalPackageChanged:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid physical package state %q",
			spec.ErrInvalid,
			s,
		)
	}
}

// PublicationOutcome records each observable boundary of managed package
// publication. RecoveryRequired is true when a physical source mutation is
// known or observed but the catalog outcome was not fully verified.
type PublicationOutcome struct {
	PhysicalPackage    PhysicalPackageState `json:"physicalPackage"`
	SourceGeneration   string               `json:"sourceGeneration,omitempty"`
	SourceAcknowledged bool                 `json:"sourceAcknowledged"`
	RefreshCompleted   bool                 `json:"refreshCompleted"`
	ArtifactVerified   bool                 `json:"artifactVerified"`
	RecoveryRequired   bool                 `json:"recoveryRequired"`
}

func (o PublicationOutcome) Validate() error {
	if err := o.PhysicalPackage.Validate(); err != nil {
		return err
	}
	if o.SourceGeneration != "" {
		if err := spec.ValidateSourceGeneration(o.SourceGeneration); err != nil {
			return err
		}
	}
	if o.ArtifactVerified && !o.SourceAcknowledged {
		return fmt.Errorf(
			"%w: verified package publication requires Source acknowledgment",
			spec.ErrInvalid,
		)
	}
	return nil
}

// RemovalOutcome records the equivalent source-side and catalog-side removal
// boundaries.
type RemovalOutcome struct {
	PhysicalPackage         PhysicalPackageState `json:"physicalPackage"`
	SourceGeneration        string               `json:"sourceGeneration,omitempty"`
	SourceAcknowledged      bool                 `json:"sourceAcknowledged"`
	DiscoveryPruned         bool                 `json:"discoveryPruned"`
	RefreshCompleted        bool                 `json:"refreshCompleted"`
	ExpectedArtifactMissing bool                 `json:"expectedArtifactMissing"`
	RecoveryRequired        bool                 `json:"recoveryRequired"`
}

func (o RemovalOutcome) Validate() error {
	if err := o.PhysicalPackage.Validate(); err != nil {
		return err
	}
	if o.SourceGeneration != "" {
		return spec.ValidateSourceGeneration(o.SourceGeneration)
	}
	return nil
}

// PublicationError preserves observable physical-write information after a
// later Source, refresh, or Artifact verification failure.
type PublicationError struct {
	Outcome PublicationOutcome
	Cause   error
}

func (e *PublicationError) Error() string {
	if e == nil || e.Cause == nil {
		return "managed package publication failed"
	}
	return "managed package publication failed after source-side processing: " +
		e.Cause.Error()
}

func (e *PublicationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// RemovalError preserves observable physical-removal information after a
// later Source, discovery, refresh, or missing-state verification failure.
type RemovalError struct {
	Outcome RemovalOutcome
	Cause   error
}

func (e *RemovalError) Error() string {
	if e == nil || e.Cause == nil {
		return "managed package removal failed"
	}
	return "managed package removal failed after source-side processing: " +
		e.Cause.Error()
}

func (e *RemovalError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func PublicationOutcomeFromError(err error) (PublicationOutcome, bool) {
	var value *PublicationError
	if !errors.As(err, &value) || value == nil {
		return PublicationOutcome{}, false
	}
	return value.Outcome, true
}

func RemovalOutcomeFromError(err error) (RemovalOutcome, bool) {
	var value *RemovalError
	if !errors.As(err, &value) || value == nil {
		return RemovalOutcome{}, false
	}
	return value.Outcome, true
}
