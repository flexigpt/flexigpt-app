package artifactbuiltin

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type ContentRef struct {
	Locator            string  `json:"locator,omitempty"`
	URI                string  `json:"uri,omitempty"`
	SubresourceLocator string  `json:"subresourceLocator,omitempty"`
	Digest             *string `json:"digest,omitempty"`
	MediaType          string  `json:"mediaType,omitempty"`
	Role               string  `json:"role,omitempty"`
}

func (v ContentRef) Clone() ContentRef {
	output := v
	if v.Digest != nil {
		digest := *v.Digest
		output.Digest = &digest
	}
	return output
}

// ValidateShareableCollectionMetadata validates metadata shared by portable
// collection schemas and local collection records.
//
// It intentionally does not require a document digest, members, or package
// bytes. Those belong to document canonicalization and package hydration.
func ValidateShareableCollectionMetadata(
	logicalName string,
	logicalVersion string,
	displayName string,
	description string,
	labels map[string]string,
) error {
	if err := basespec.ValidatePortableName(
		"logical name",
		logicalName,
	); err != nil {
		return err
	}
	if logicalVersion != "" {
		if err := basespec.ValidatePortableName(
			"logical version",
			logicalVersion,
		); err != nil {
			return err
		}
	}
	if err := basespec.ValidateOptionalText(
		"skill collection display name",
		displayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"skill collection description",
		description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	return validateShareableCollectionLabels(labels)
}

func validateShareableCollectionLabels(
	values map[string]string,
) error {
	if len(values) > basespec.MaxLabels {
		return fmt.Errorf(
			"%w: shareable collection labels exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxLabels,
		)
	}
	for key, value := range values {
		if err := basespec.ValidateIdentifier(
			"shareable collection label key",
			key,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := basespec.ValidateRequiredText(
			"shareable collection label value",
			value,
			basespec.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	return nil
}
