package definition

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Definition struct {
	Digest         cryptoutil.Digest       `json:"digest"`
	Kind           basespec.ArtifactKind   `json:"kind"`
	SchemaID       basespec.SchemaID       `json:"schemaID"`
	SchemaVersion  string                  `json:"schemaVersion"`
	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`
	Body           json.RawMessage         `json:"body"`
	Dependencies   []Selector              `json:"dependencies,omitempty"`
}

func (d Definition) Validate() error {
	if err := cryptoutil.ValidateDigest(d.Digest); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := basespec.ValidateArtifactKind(d.Kind); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := basespec.ValidateSchemaID(d.SchemaID); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := basespec.ValidateRequiredText(
		"definition schema version",
		d.SchemaVersion,
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateLogicalName(d.LogicalName); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := basespec.ValidateLogicalVersion(d.LogicalVersion, true); err != nil {
		return fmt.Errorf("definition: %w", err)
	}
	if err := basespec.ValidateOptionalText(
		"definition display name",
		d.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"definition description",
		d.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if len(d.Labels) > basespec.MaxLabels {
		return fmt.Errorf(
			"%w: definition labels exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxLabels,
		)
	}
	for key, value := range d.Labels {
		if err := basespec.ValidateIdentifier(
			"definition label key",
			key,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := basespec.ValidateRequiredText(
			"definition label value",
			value,
			basespec.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	if len(d.Dependencies) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: definition dependencies exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}
	if _, err := jsonutil.CanonicalizeObject(
		d.Body,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return fmt.Errorf("%w: definition body: %w", basespec.ErrInvalid, err)
	}
	for index, selector := range d.Dependencies {
		if err := selector.Validate(); err != nil {
			return fmt.Errorf("definition dependencies[%d]: %w", index, err)
		}
	}
	return nil
}
