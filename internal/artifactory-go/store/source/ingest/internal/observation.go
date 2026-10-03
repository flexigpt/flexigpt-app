package internal

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
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
	RootID  rootModel.RootID            `json:"rootID"`
	Binding artifactModel.SourceBinding `json:"binding"`

	Kind           artifactModel.ArtifactKind  `json:"kind,omitempty"`
	LogicalName    spec.LogicalName            `json:"logicalName,omitempty"`
	LogicalVersion spec.LogicalVersion         `json:"logicalVersion,omitempty"`
	Definition     *definitionModel.Definition `json:"-"`

	SourceContentDigest *cryptoutil.Digest      `json:"sourceContentDigest,omitempty"`
	DecoderID           spec.DecoderID          `json:"decoderID,omitempty"`
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
				spec.ErrInvalid,
			)
		}
		// Definition admission belongs to the decoder/generated boundary and
		// the persistence writer. Staging validation checks linkage only.
		if err := cryptoutil.ValidateDigest(o.Definition.Digest); err != nil {
			return err
		}
		if o.Definition.Kind != o.Kind ||
			o.Definition.LogicalName != o.LogicalName ||
			o.Definition.LogicalVersion != o.LogicalVersion {
			return fmt.Errorf(
				"%w: valid Source observation Definition does not match observation identity",
				spec.ErrInvalid,
			)
		}

	case ObservationInvalid:
		if o.Kind != "" ||
			o.LogicalName != "" ||
			o.LogicalVersion != "" ||
			o.Definition != nil {
			return fmt.Errorf(
				"%w: invalid Source observation cannot retain declaration identity",
				spec.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid Source observation state %q",
			spec.ErrInvalid,
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
	Binding artifactModel.SourceBinding
	Kind    artifactModel.ArtifactKind
}

func (o Observation) TypedOrigin() typedOrigin {
	return typedOrigin{
		Binding: o.Binding,
		Kind:    o.Kind,
	}
}
