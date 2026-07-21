package definition

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

type Definition struct {
	Digest         artifactstore.Digest         `json:"digest"`
	Kind           artifactstore.ArtifactKind   `json:"kind"`
	SchemaID       artifactstore.SchemaID       `json:"schemaID"`
	SchemaVersion  string                       `json:"schemaVersion"`
	LogicalName    artifactstore.LogicalName    `json:"logicalName"`
	LogicalVersion artifactstore.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                       `json:"displayName,omitempty"`
	Description    string                       `json:"description,omitempty"`
	Labels         map[string]string            `json:"labels,omitempty"`
	Body           json.RawMessage              `json:"body"`
	Dependencies   []Selector                   `json:"dependencies,omitempty"`
}

func (d Definition) Validate() error {
	if err := artifactstore.ValidateDigest(d.Digest); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := artifactstore.ValidateArtifactKind(d.Kind); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := artifactstore.ValidateSchemaID(d.SchemaID); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := artifactstore.ValidateRequiredText(
		"definition schema version",
		d.SchemaVersion,
		artifactstore.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := artifactstore.ValidateLogicalName(d.LogicalName); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := artifactstore.ValidateLogicalVersion(d.LogicalVersion, true); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := artifactstore.ValidateOptionalText(
		"definition display name",
		d.DisplayName,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := artifactstore.ValidateOptionalText(
		"definition description",
		d.Description,
		artifactstore.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if len(d.Labels) > artifactstore.MaxLabels {
		return fmt.Errorf(
			"%w: definition labels exceed %d entries",
			artifactstore.ErrInvalid,
			artifactstore.MaxLabels,
		)
	}
	for key, value := range d.Labels {
		if err := artifactstore.ValidateIdentifier(
			"definition label key",
			key,
			artifactstore.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := artifactstore.ValidateRequiredText(
			"definition label value",
			value,
			artifactstore.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	if _, err := jsoncanon.CanonicalizeObject(
		d.Body,
		artifactstore.MaxDefinitionBodyBytes,
	); err != nil {
		return fmt.Errorf("%w: definition body: %w", artifactstore.ErrInvalid, err)
	}
	for index, selector := range d.Dependencies {
		if err := selector.Validate(); err != nil {
			return fmt.Errorf("definition dependencies[%d]: %w", index, err)
		}
	}
	return nil
}
