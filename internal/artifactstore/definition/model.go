package definition

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

const FileFormatV1 = "artifact-definition/v1"

type Selector struct {
	Kind              artifactstore.ArtifactKind `json:"kind"`
	LogicalName       artifactstore.LogicalName  `json:"logicalName,omitempty"`
	VersionConstraint string                     `json:"versionConstraint,omitempty"`
	Labels            map[string]string          `json:"labels,omitempty"`
}

func (s Selector) Validate() error {
	if err := artifactstore.ValidateArtifactKind(s.Kind); err != nil {
		return fmt.Errorf("selector: %w", err)
	}
	if s.LogicalName != "" {
		if err := artifactstore.ValidateLogicalName(s.LogicalName); err != nil {
			return fmt.Errorf("selector: %w", err)
		}
	}
	if len(s.VersionConstraint) > artifactstore.MaxVersionBytes {
		return fmt.Errorf(
			"%w: selector version constraint exceeds %d bytes",
			artifactstore.ErrInvalid,
			artifactstore.MaxVersionBytes,
		)
	}
	if len(s.Labels) > artifactstore.MaxLabels {
		return fmt.Errorf(
			"%w: selector labels exceed %d entries",
			artifactstore.ErrInvalid,
			artifactstore.MaxLabels,
		)
	}
	for key, value := range s.Labels {
		if err := artifactstore.ValidateIdentifier(
			"selector label key",
			key,
			artifactstore.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := artifactstore.ValidateRequiredText(
			"selector label value",
			value,
			artifactstore.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	return nil
}

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

type File struct {
	Format     string     `json:"format"`
	Definition Definition `json:"definition"`
}

func (f File) Validate() error {
	if f.Format != FileFormatV1 {
		return fmt.Errorf(
			"%w: unsupported definition file format %q",
			artifactstore.ErrInvalid,
			f.Format,
		)
	}
	return f.Definition.Validate()
}

func cloneLabels(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	maps.Copy(output, input)
	return output
}

func cloneSelectors(input []Selector) []Selector {
	if input == nil {
		return nil
	}
	output := make([]Selector, len(input))
	for index, value := range input {
		output[index] = value
		output[index].Labels = cloneLabels(value.Labels)
	}
	return output
}

func sortedLabels(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	output := make(map[string]string, len(input))
	for _, key := range keys {
		output[key] = input[key]
	}
	return output
}
