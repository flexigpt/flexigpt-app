package basespec

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

var portableNamePattern = regexp.MustCompile(
	`^[A-Za-z0-9][A-Za-z0-9._-]*$`,
)

func ValidatePortableMetadata(
	logicalName LogicalName,
	logicalVersion LogicalVersion,
	displayName string,
	description string,
	labels map[string]string,
	digest *string,
) error {
	if err := ValidatePortableName(
		"logical name",
		string(logicalName),
	); err != nil {
		return err
	}
	if logicalVersion != "" {
		if err := ValidatePortableName(
			"logical version",
			string(logicalVersion),
		); err != nil {
			return err
		}
	}
	if err := ValidateOptionalText(
		"display name",
		displayName,
		MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := ValidateOptionalText(
		"description",
		description,
		MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := ValidateLabels("", labels); err != nil {
		return err
	}
	if digest != nil {
		if err := cryptoutil.ValidateDigest(cryptoutil.Digest(*digest)); err != nil {
			return fmt.Errorf("document digest: %w", err)
		}
	}
	return nil
}

func ValidateContentReference(
	locator string,
	uri string,
	digest *string,
) error {
	switch {
	case locator != "" && uri != "":
		return errors.New("invalid content reference: cannot contain both locator and URI")

	case locator != "":
		if err := Locator(locator).ValidatePortable(false); err != nil {
			return err
		}

	case uri != "":
		if err := validateContentReferenceURI(uri); err != nil {
			return err
		}

	default:
		return errors.New("invalid content reference:: content reference requires a locator or URI")
	}

	if digest != nil {
		if err := cryptoutil.ValidateDigest(cryptoutil.Digest(*digest)); err != nil {
			return fmt.Errorf("content reference digest: %w", err)
		}
	}
	return nil
}

func validateContentReferenceURI(value string) error {
	if err := ValidateRequiredText(
		"content reference URI",
		value,
		MaxURIBytes,
	); err != nil {
		return err
	}
	if strings.Contains(value, "#") {
		return errors.New("invalid content reference: URI cannot contain a fragment")
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf("invalid content reference: URI: %w", err)
	}
	if !parsed.IsAbs() ||
		parsed.Scheme == "" ||
		parsed.User != nil ||
		parsed.Fragment != "" ||
		strings.EqualFold(parsed.Scheme, "file") {
		return errors.New("invalid content reference: URI")
	}
	return nil
}

func ValidatePortableName(label, value string) error {
	if value == "" ||
		len(value) > MaxLogicalNameBytes ||
		!utf8.ValidString(value) ||
		!portableNamePattern.MatchString(value) {
		return fmt.Errorf(
			"%w: %s %q is not a portable name",
			ErrInvalid,
			label,
			value,
		)
	}
	return nil
}

func ValidateLabels(
	inSubject string,
	values map[string]string,
) error {
	subject := ""
	if inSubject != "" {
		subject = inSubject + " "
	}
	if len(values) > MaxLabels {
		return fmt.Errorf(
			"%w: %s labels exceed %d entries",
			ErrInvalid,
			subject,
			MaxLabels,
		)
	}
	for key, value := range values {
		if err := ValidateIdentifier(
			subject+"label key",
			key,
			MaxKindBytes,
		); err != nil {
			return err
		}
		if err := ValidateRequiredText(
			subject+"label value",
			value,
			MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	return nil
}

// ValidateIdentifier validates an ASCII lower-camel identifier.
//
// Dots and hyphens may separate segments. Every segment must start with a
// lowercase ASCII letter, while subsequent characters may be ASCII letters or
// digits. This accepts documentSets, managedMCP, mcp.policyV1, and
// agent-managedMCP while rejecting PascalCase, underscores, empty segments,
// and leading digits.
func ValidateIdentifier(label, value string, maximum int) error {
	if value == "" ||
		len(value) > maximum ||
		!identifierPattern.MatchString(value) {
		return fmt.Errorf(
			"%w: %s must start lowercase and use lower-camel segments separated by dots or hyphens",
			ErrInvalid,
			label,
		)
	}
	return nil
}

func ValidateOptionalText(label, value string, maximum int) error {
	if value == "" {
		return nil
	}
	return ValidateRequiredText(label, value, maximum)
}

// ValidateIncludePattern is retained as the common compatibility entrypoint.
// New code should use ValidatePathPattern when the pattern is not specifically
// an inclusion pattern.
func ValidateIncludePattern(pattern string) error {
	return ValidatePathPattern(pattern)
}

func ValidateRequiredText(label, value string, maximum int) error {
	if value == "" ||
		!utf8.ValidString(value) ||
		strings.TrimSpace(value) != value {
		return fmt.Errorf(
			"%w: %s - value %q must be non-empty, valid UTF-8, and trimmed",
			ErrInvalid,
			label,
			value,
		)
	}
	if len(value) > maximum {
		return fmt.Errorf(
			"%w: %s exceeds %d bytes",
			ErrInvalid,
			label,
			maximum,
		)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf(
				"%w: %s contains a control character",
				ErrInvalid,
				label,
			)
		}
	}
	return nil
}

func ValidateSourceGeneration(value string) error {
	return ValidateRequiredText(
		"source generation",
		value,
		MaxSourceGenerationBytes,
	)
}
