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
	if err := validateRelativePath(
		"portable locator",
		string(v),
		allowRoot,
	); err != nil {
		return err
	}
	if v == "." {
		return nil
	}

	for segment := range strings.SplitSeq(string(v), "/") {
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
