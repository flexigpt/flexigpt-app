package basespec

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	MaxKindBytes             = 128
	MaxFingerprintBytes      = 128
	MaxSchemaIDBytes         = 256
	MaxDisplayNameBytes      = 256
	MaxDescriptionBytes      = 16 * 1024
	MaxLogicalNameBytes      = 256
	MaxURIBytes              = 16 * 1024
	MaxVersionBytes          = 256
	MaxSourceGenerationBytes = 1024
	MaxLocatorBytes          = 4096

	MaxLabels                 = 64
	MaxLabelValueBytes        = 256
	MaxConfigBytes            = 1 << 20
	MaxLocalDataBytes         = 1 << 20
	MaxDefinitionBodyBytes    = 4 << 20
	MaxDefinitionBytes        = 16 << 20
	MaxDefinitionDependencies = 4096
	MaxCandidateBytes         = 4 << 20
	MaxScanBytes              = int64(512 << 20)

	DefaultMaxCandidates   = 10_000
	DefaultMaxEntries      = 100_000
	DefaultMaxDepth        = 64
	MaxDiscoveryCandidates = 100_000
	MaxDiscoveryEntries    = 1_000_000
	MaxDiscoveryDepth      = 256
)

var identifierPattern = regexp.MustCompile(
	`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`,
)

type (
	RootID             string
	SourceID           string
	CollectionID       string
	ArtifactID         string
	SourceKind         string
	CollectionKind     string
	ArtifactKind       string
	SchemaID           string
	AttachmentRole     string
	DecoderID          string
	Locator            string
	SubresourceLocator string

	LogicalName    string
	LogicalVersion string
)

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

func ValidateLocator(value Locator, allowRoot bool) error {
	return validateRelativePath("locator", string(value), allowRoot)
}

func ValidateSubresourceLocator(value SubresourceLocator) error {
	if value == "" {
		return nil
	}
	return validateRelativePath("subresource locator", string(value), false)
}

// ValidatePortableLocator applies the platform-independent locator rules used
// for managed content and portable packages. Platform-specific filename
// mapping and collision handling belong to the storage implementation.
//
// Generic Source locators can describe an existing platform-specific Source.
// Portable locators remain bounded slash-separated relative references.
func ValidatePortableLocator(value Locator, allowRoot bool) error {
	return ValidateLocator(value, allowRoot)
}

func ValidatePortableSubresourceLocator(value SubresourceLocator) error {
	if value == "" {
		return nil
	}
	return ValidatePortableLocator(Locator(value), false)
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
