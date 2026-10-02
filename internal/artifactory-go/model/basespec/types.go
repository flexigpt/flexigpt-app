package basespec

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type StorageKey string

func (v StorageKey) Validate() error {
	return ValidateIdentifier(
		"storage key",
		string(v),
		MaxStorageKeyBytes,
	)
}

type LogicalName string

func (v LogicalName) Validate() error {
	return ValidatePortableName(
		"logical name",
		string(v),
	)
}

type LogicalVersion string

func (v LogicalVersion) Validate(optional bool) error {
	if v == "" && optional {
		return nil
	}
	return ValidateRequiredText(
		"logical version",
		string(v),
		MaxVersionBytes,
	)
}

type DecoderID string

func (v DecoderID) Validate() error {
	return ValidateIdentifier("decoder ID", string(v), MaxKindBytes)
}

type SubresourceLocator string

func (v SubresourceLocator) Validate() error {
	if v == "" {
		return nil
	}
	return validateRelativePath("subresource locator", string(v), false)
}

type Locator string

func (v Locator) Validate(allowRoot bool) error {
	return validateRelativePath("locator", string(v), allowRoot)
}

// ValidatePortable applies the platform-independent locator rules used
// for managed content and portable packages. Platform-specific filename
// mapping and collision handling belong to the storage implementation.
//
// Generic Source locators can describe an existing platform-specific Source.
// Portable locators remain bounded slash-separated relative references.
func (v Locator) ValidatePortable(allowRoot bool) error {
	return validatePortablePath(
		"portable locator",
		string(v),
		allowRoot,
		false,
	)
}

// ValidatePortableRelativeReference validates a declaration-relative portable
// path. Dot segments are accepted here because the final resolved
// basespec.Locator is validated separately and cannot escape its Source.
func ValidatePortableRelativeReference(
	label string,
	value string,
	allowRoot bool,
) error {
	return validatePortablePath(label, value, allowRoot, true)
}

func validatePortablePath(
	label string,
	value string,
	allowRoot bool,
	allowDotSegments bool,
) error {
	if value == "." {
		if allowRoot {
			return nil
		}
		return fmt.Errorf(
			"%w: %s must be a bounded relative path",
			ErrInvalid,
			label,
		)
	}
	if value == "" ||
		len(value) > MaxLocatorBytes ||
		!utf8.ValidString(value) ||
		strings.ContainsRune(value, 0) ||
		strings.Contains(value, "\\") ||
		strings.Contains(value, ":") ||
		strings.HasPrefix(value, "/") {
		return fmt.Errorf(
			"%w: %s must be a bounded relative path",
			ErrInvalid,
			label,
		)
	}
	for segment := range strings.SplitSeq(value, "/") {
		if segment == "" {
			return fmt.Errorf(
				"%w: %s contains an invalid path segment",
				ErrInvalid,
				label,
			)
		}
		if segment == "." || segment == ".." {
			if allowDotSegments {
				continue
			}
			return fmt.Errorf(
				"%w: %s contains an invalid path segment",
				ErrInvalid,
				label,
			)
		}
		for _, character := range segment {
			if unicode.IsControl(character) {
				return fmt.Errorf(
					"%w: %s contains a control character",
					ErrInvalid,
					label,
				)
			}
		}
		if strings.HasSuffix(segment, ".") ||
			strings.HasSuffix(segment, " ") {
			return fmt.Errorf(
				"%w: %s contains a trailing dot or space",
				ErrInvalid,
				label,
			)
		}
		if strings.ContainsAny(segment, `<>"|?*`) {
			return fmt.Errorf(
				"%w: %s contains a platform-reserved character",
				ErrInvalid,
				label,
			)
		}

		baseName, _, _ := strings.Cut(segment, ".")
		if _, reserved := portableReservedBaseNames[strings.ToUpper(baseName)]; reserved {
			return fmt.Errorf(
				"%w: %s contains reserved basename %q",
				ErrInvalid,
				label,
				segment,
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
