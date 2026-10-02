package definition

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// Key identifies one immutable Definition in one Root.
//
// It is a storage identity, not a consumer request or transport projection.
type Key struct {
	RootID root.RootID
	Digest cryptoutil.Digest
}

func (k Key) Validate() error {
	if err := k.RootID.Validate(); err != nil {
		return err
	}
	return cryptoutil.ValidateDigest(k.Digest)
}

// Definition is a canonical source-decoded Artifact definition.
//
// Digest is a canonical content fingerprint used for integrity, comparison,
// source-state reconciliation, and immutable Root-local persistence. It is
// not a semantic name or an Artifact address.
type Definition struct {
	Digest         cryptoutil.Digest     `json:"digest"`
	Kind           artifact.ArtifactKind `json:"kind"`
	SchemaID       schema.SchemaID       `json:"schemaID"`
	SchemaVersion  string                `json:"schemaVersion"`
	LogicalName    model.LogicalName     `json:"logicalName"`
	LogicalVersion model.LogicalVersion  `json:"logicalVersion,omitempty"`
	DisplayName    string                `json:"displayName,omitempty"`
	Description    string                `json:"description,omitempty"`
	Labels         map[string]string     `json:"labels,omitempty"`
	Body           json.RawMessage       `json:"body"`
	Dependencies   []Selector            `json:"dependencies,omitempty"`
}

func (d Definition) Validate() error {
	canonical, err := Canonicalize(d)
	if err != nil {
		return err
	}
	if canonical.Digest != d.Digest ||
		!bytes.Equal(canonical.Body, d.Body) {
		return fmt.Errorf(
			"%w: Definition digest or Body is not canonical",
			model.ErrDigestMismatch,
		)
	}
	return nil
}

func validateDefinitionFields(d Definition) error {
	if err := cryptoutil.ValidateDigest(d.Digest); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := d.Kind.Validate(); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := d.SchemaID.Validate(); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := model.ValidateRequiredText(
		"definition schema version",
		d.SchemaVersion,
		model.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := d.LogicalName.Validate(); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := d.LogicalVersion.Validate(true); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := model.ValidateOptionalText(
		"definition display name",
		d.DisplayName,
		model.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := model.ValidateOptionalText(
		"definition description",
		d.Description,
		model.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := model.ValidateLabels("definition", d.Labels); err != nil {
		return err
	}
	if len(d.Dependencies) > model.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: definition dependencies exceed %d entries",
			model.ErrInvalid,
			model.MaxDefinitionDependencies,
		)
	}
	for index, selector := range d.Dependencies {
		if err := selector.Validate(); err != nil {
			return fmt.Errorf("definition dependencies[%d]: %w", index, err)
		}
	}
	return nil
}
