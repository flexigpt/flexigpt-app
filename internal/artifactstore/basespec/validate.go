package basespec

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
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
) error {
	if err := ValidatePortableName(
		"logical name",
		string(logicalName),
	); err != nil {
		return err
	}
	if err := ValidateLogicalVersion(
		logicalVersion,
		true,
	); err != nil {
		return err
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
	return ValidateLabels("", labels)
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

func ValidateRootID(value RootID) error {
	return uuidutil.ValidateUUIDv7("root ID", string(value))
}

func ValidateSourceID(value SourceID) error {
	return uuidutil.ValidateUUIDv7("source ID", string(value))
}

func ValidateCollectionID(value CollectionID) error {
	return uuidutil.ValidateUUIDv7("collection ID", string(value))
}

func ValidateArtifactID(value ArtifactID) error {
	return uuidutil.ValidateUUIDv7("artifact ID", string(value))
}

func ValidateSourceKind(value SourceKind) error {
	return ValidateIdentifier("source kind", string(value), MaxKindBytes)
}

func ValidateCollectionKind(value CollectionKind) error {
	return ValidateIdentifier("collection kind", string(value), MaxKindBytes)
}

func ValidateArtifactKind(value ArtifactKind) error {
	return ValidateIdentifier("artifact kind", string(value), MaxKindBytes)
}

func ValidateSchemaID(value SchemaID) error {
	return ValidateIdentifier("schema ID", string(value), MaxSchemaIDBytes)
}

func ValidateAttachmentRole(value AttachmentRole) error {
	return ValidateIdentifier("attachment role", string(value), MaxKindBytes)
}

func ValidateDecoderID(value DecoderID) error {
	return ValidateIdentifier("decoder ID", string(value), MaxKindBytes)
}

func ValidateSourceGeneration(value string) error {
	return ValidateRequiredText(
		"source generation",
		value,
		MaxSourceGenerationBytes,
	)
}

func ValidateLogicalName(value LogicalName) error {
	return ValidateRequiredText(
		"logical name",
		string(value),
		MaxLogicalNameBytes,
	)
}

func ValidateLogicalVersion(value LogicalVersion, optional bool) error {
	if value == "" && optional {
		return nil
	}
	return ValidateRequiredText(
		"logical version",
		string(value),
		MaxVersionBytes,
	)
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

func ValidateLocator(value Locator, allowRoot bool) error {
	return validateRelativePath("locator", string(value), allowRoot)
}

func ValidateSubresourceLocator(value SubresourceLocator) error {
	if value == "" {
		return nil
	}
	return validateRelativePath("subresource locator", string(value), false)
}

// PortableLocatorIdentity returns the case-insensitive portable identity used
// to reject package entries that would collide on case-insensitive filesystems.
func PortableLocatorIdentity(
	value Locator,
	allowRoot bool,
) (string, error) {
	if err := ValidatePortableLocator(value, allowRoot); err != nil {
		return "", err
	}
	if value == "." {
		return ".", nil
	}
	return strings.ToLower(string(value)), nil
}

func ValidatePortableSubresourceLocator(value SubresourceLocator) error {
	if value == "" {
		return nil
	}
	return ValidatePortableLocator(Locator(value), false)
}

// ValidatePortableLocator applies the platform-independent locator rules used
// for managed content and portable packages. Platform-specific filename
// mapping and collision handling belong to the storage implementation.
//
// Generic Source locators can describe an existing platform-specific Source.
// Portable locators remain bounded slash-separated relative references.
func ValidatePortableLocator(value Locator, allowRoot bool) error {
	if err := validateRelativePath(
		"portable locator",
		string(value),
		allowRoot,
	); err != nil {
		return err
	}
	if value == "." {
		return nil
	}

	for segment := range strings.SplitSeq(string(value), "/") {
		if strings.HasSuffix(segment, ".") ||
			strings.HasSuffix(segment, " ") {
			return fmt.Errorf(
				"%w: portable locator contains a trailing dot or space",
				ErrInvalid,
			)
		}
		if strings.ContainsAny(segment, `<>"|?*`) {
			return fmt.Errorf(
				"%w: portable locator contains a platform-reserved character",
				ErrInvalid,
			)
		}

		baseName, _, _ := strings.Cut(segment, ".")
		if _, reserved := portableReservedBaseNames[strings.ToUpper(baseName)]; reserved {
			return fmt.Errorf(
				"%w: portable locator contains reserved basename %q",
				ErrInvalid,
				segment,
			)
		}
	}
	return nil
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

func validateRelativePath(label, value string, allowRoot bool) error {
	if value == "." && allowRoot {
		return nil
	}
	if value == "" ||
		len(value) > MaxLocatorBytes ||
		!utf8.ValidString(value) {
		return fmt.Errorf(
			"%w: %s must be a bounded relative path",
			ErrInvalid,
			label,
		)
	}
	if strings.ContainsRune(value, 0) ||
		strings.Contains(value, "\\") ||
		strings.Contains(value, ":") ||
		strings.HasPrefix(value, "/") {
		return fmt.Errorf(
			"%w: %s contains a disallowed path character",
			ErrInvalid,
			label,
		)
	}
	parts := strings.SplitSeq(value, "/")
	for part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf(
				"%w: %s contains an invalid path segment",
				ErrInvalid,
				label,
			)
		}
		for _, character := range part {
			if unicode.IsControl(character) {
				return fmt.Errorf(
					"%w: %s contains a control character",
					ErrInvalid,
					label,
				)
			}
		}
	}
	return nil
}
