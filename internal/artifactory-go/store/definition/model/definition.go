package model

import (
	"bytes"
	"encoding/json"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// Key identifies one immutable Definition in one Root.
//
// It is a storage identity, not a consumer request or transport projection.
type Key struct {
	RootID rootModel.RootID
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
	Digest         cryptoutil.Digest          `json:"digest"`
	Kind           artifactModel.ArtifactKind `json:"kind"`
	SchemaID       schemaModel.SchemaID       `json:"schemaID"`
	SchemaVersion  string                     `json:"schemaVersion"`
	LogicalName    storeSpec.LogicalName      `json:"logicalName"`
	LogicalVersion storeSpec.LogicalVersion   `json:"logicalVersion,omitempty"`
	DisplayName    string                     `json:"displayName,omitempty"`
	Description    string                     `json:"description,omitempty"`
	Labels         map[string]string          `json:"labels,omitempty"`
	Body           json.RawMessage            `json:"body"`
	Dependencies   []Selector                 `json:"dependencies,omitempty"`
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
			storeSpec.ErrDigestMismatch,
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
	if err := storeSpec.ValidateRequiredText(
		"definition schema version",
		d.SchemaVersion,
		storeSpec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := d.LogicalName.Validate(); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := d.LogicalVersion.Validate(true); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := storeSpec.ValidateOptionalText(
		"definition display name",
		d.DisplayName,
		storeSpec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := storeSpec.ValidateOptionalText(
		"definition description",
		d.Description,
		storeSpec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := storeSpec.ValidateLabels("definition", d.Labels); err != nil {
		return err
	}
	if len(d.Dependencies) > storeSpec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: definition dependencies exceed %d entries",
			storeSpec.ErrInvalid,
			storeSpec.MaxDefinitionDependencies,
		)
	}
	for index, selector := range d.Dependencies {
		if err := selector.Validate(); err != nil {
			return fmt.Errorf("definition dependencies[%d]: %w", index, err)
		}
	}
	return nil
}
