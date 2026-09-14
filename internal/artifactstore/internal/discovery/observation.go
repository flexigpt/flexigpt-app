package discovery

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type ObservationState string

const (
	ObservationValid   ObservationState = "valid"
	ObservationInvalid ObservationState = "invalid"
)

// Observation is transient Source refresh staging data. It is deliberately
// not a public Catalog entity and is never persisted as a current occurrence.
type Observation struct {
	RootID  root.RootID            `json:"rootID"`
	Binding artifact.SourceBinding `json:"binding"`

	Kind           artifact.ArtifactKind   `json:"kind,omitempty"`
	LogicalName    basespec.LogicalName    `json:"logicalName,omitempty"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	Definition     *definition.Definition  `json:"-"`

	SourceContentDigest *cryptoutil.Digest      `json:"sourceContentDigest,omitempty"`
	DecoderID           basespec.DecoderID      `json:"decoderID,omitempty"`
	State               ObservationState        `json:"state"`
	Diagnostics         []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
}

func (o Observation) Validate() error {
	if err := o.RootID.Validate(); err != nil {
		return err
	}
	if err := o.Binding.Validate(); err != nil {
		return err
	}
	if o.SourceContentDigest != nil {
		if err := cryptoutil.ValidateDigest(
			*o.SourceContentDigest,
		); err != nil {
			return err
		}
	}
	if o.DecoderID != "" {
		if err := o.DecoderID.Validate(); err != nil {
			return err
		}
	}
	if err := diagnostic.Validate(o.Diagnostics); err != nil {
		return err
	}

	switch o.State {
	case ObservationValid:
		if err := o.Kind.Validate(); err != nil {
			return err
		}
		if err := o.LogicalName.Validate(); err != nil {
			return err
		}
		if err := o.LogicalVersion.Validate(true); err != nil {
			return err
		}
		if o.Definition == nil ||
			o.SourceContentDigest == nil ||
			o.DecoderID == "" {
			return fmt.Errorf(
				"%w: valid Source observation requires Definition, source digest, and decoder",
				basespec.ErrInvalid,
			)
		}
		canonical, err := definition.Canonicalize(*o.Definition)
		if err != nil {
			return err
		}
		if canonical.Digest != o.Definition.Digest ||
			canonical.Kind != o.Kind ||
			canonical.LogicalName != o.LogicalName ||
			canonical.LogicalVersion != o.LogicalVersion {
			return fmt.Errorf(
				"%w: valid Source observation Definition does not match observation identity",
				basespec.ErrInvalid,
			)
		}

	case ObservationInvalid:
		if o.Kind != "" ||
			o.LogicalName != "" ||
			o.LogicalVersion != "" ||
			o.Definition != nil {
			return fmt.Errorf(
				"%w: invalid Source observation cannot retain declaration identity",
				basespec.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid Source observation state %q",
			basespec.ErrInvalid,
			o.State,
		)
	}
	return nil
}

func (o Observation) Clone() Observation {
	output := o
	output.SourceContentDigest = cryptoutil.CloneDigest(
		o.SourceContentDigest,
	)
	output.Diagnostics = diagnostic.Clone(o.Diagnostics)
	if o.Definition != nil {
		value := o.Definition.Clone()
		output.Definition = &value
	}
	return output
}

type typedOrigin struct {
	Binding artifact.SourceBinding
	Kind    artifact.ArtifactKind
}

func (o Observation) TypedOrigin() typedOrigin {
	return typedOrigin{
		Binding: o.Binding,
		Kind:    o.Kind,
	}
}
