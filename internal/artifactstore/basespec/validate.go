package basespec

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

var portableNamePattern = regexp.MustCompile(
	`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`,
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
	if !portableNamePattern.MatchString(value) {
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

func ValidateIdentifier(label, value string, maximum int) error {
	if value == "" ||
		len(value) > maximum ||
		!identifierPattern.MatchString(value) {
		return fmt.Errorf(
			"%w: %s must be a lowercase dotted or hyphenated identifier",
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

// ValidateIncludePattern validates a source-relative glob. It deliberately
// rejects path traversal and host-path syntax before passing the pattern to
// path.Match.
func ValidateIncludePattern(pattern string) error {
	if err := ValidateRequiredText(
		"discovery pattern",
		pattern,
		MaxLocatorBytes,
	); err != nil {
		return err
	}
	if strings.HasPrefix(pattern, "/") ||
		strings.ContainsAny(pattern, `\:`) {
		return fmt.Errorf(
			"%w: discovery pattern contains a disallowed path character",
			ErrInvalid,
		)
	}
	for segment := range strings.SplitSeq(pattern, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf(
				"%w: discovery pattern contains an invalid path segment",
				ErrInvalid,
			)
		}
	}
	if _, err := path.Match(pattern, "candidate"); err != nil {
		return fmt.Errorf(
			"%w: invalid discovery pattern %q: %w",
			ErrInvalid,
			pattern,
			err,
		)
	}
	return nil
}

func ValidateRequiredText(label, value string, maximum int) error {
	if value == "" ||
		!utf8.ValidString(value) ||
		strings.TrimSpace(value) != value {
		return fmt.Errorf(
			"%w: %s must be non-empty, valid UTF-8, and trimmed",
			ErrInvalid,
			label,
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
