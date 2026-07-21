package artifactstore

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	DigestSHA256Prefix = "sha256:"

	MaxKindBytes              = 128
	MaxSchemaIDBytes          = 256
	MaxDisplayNameBytes       = 256
	MaxDescriptionBytes       = 16 * 1024
	MaxLogicalNameBytes       = 256
	MaxVersionBytes           = 256
	MaxLocatorBytes           = 4096
	MaxDiagnosticCodeBytes    = 128
	MaxDiagnosticMessageBytes = 4096
	MaxDiagnostics            = 128
	MaxLabels                 = 64
	MaxLabelValueBytes        = 256
	MaxConfigBytes            = 1 << 20
	MaxLocalDataBytes         = 1 << 20
	MaxDefinitionBodyBytes    = 4 << 20
	MaxCandidateBytes         = 4 << 20
	MaxScanBytes              = int64(512 << 20)
	DefaultMaxCandidates      = 10_000
	DefaultMaxEntries         = 100_000
	DefaultMaxDepth           = 64
)

var (
	uuidV7Pattern = regexp.MustCompile(
		`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
	)
	identifierPattern = regexp.MustCompile(
		`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`,
	)
	digestPattern = regexp.MustCompile(`^` + DigestSHA256Prefix + `[0-9a-f]{64}$`)
)

type (
	RootID             string
	SourceID           string
	RecordID           string
	RootKind           string
	SourceKind         string
	ArtifactKind       string
	SchemaID           string
	AttachmentRole     string
	DecoderID          string
	Locator            string
	SubresourceLocator string
	Digest             string
	LogicalName        string
	LogicalVersion     string
)

type DiagnosticSeverity string

const (
	DiagnosticError   DiagnosticSeverity = "error"
	DiagnosticWarning DiagnosticSeverity = "warning"
	DiagnosticInfo    DiagnosticSeverity = "info"
)

type DiagnosticLocation struct {
	Locator            Locator            `json:"locator,omitempty"`
	SubresourceLocator SubresourceLocator `json:"subresourceLocator,omitempty"`
	Line               int                `json:"line,omitempty"`
	Column             int                `json:"column,omitempty"`
}

type Diagnostic struct {
	Severity DiagnosticSeverity  `json:"severity"`
	Code     string              `json:"code"`
	Message  string              `json:"message"`
	Location *DiagnosticLocation `json:"location,omitempty"`
}

func (d Diagnostic) Validate() error {
	switch d.Severity {
	case DiagnosticError, DiagnosticWarning, DiagnosticInfo:
	default:
		return fmt.Errorf("%w: invalid diagnostic severity %q", ErrInvalid, d.Severity)
	}
	if err := ValidateIdentifier("diagnostic code", d.Code, MaxDiagnosticCodeBytes); err != nil {
		return err
	}
	if err := ValidateRequiredText(
		"diagnostic message",
		d.Message,
		MaxDiagnosticMessageBytes,
	); err != nil {
		return err
	}
	if d.Location == nil {
		return nil
	}
	if d.Location.Locator != "" {
		if err := ValidateLocator(d.Location.Locator, true); err != nil {
			return fmt.Errorf("diagnostic location: %w", err)
		}
	}
	if d.Location.SubresourceLocator != "" {
		if err := ValidateSubresourceLocator(d.Location.SubresourceLocator); err != nil {
			return fmt.Errorf("diagnostic subresource location: %w", err)
		}
	}
	if d.Location.Line < 0 || d.Location.Column < 0 {
		return fmt.Errorf("%w: diagnostic line and column cannot be negative", ErrInvalid)
	}
	return nil
}

func ValidateDiagnostics(values []Diagnostic) error {
	if len(values) > MaxDiagnostics {
		return fmt.Errorf(
			"%w: diagnostics exceed %d entries",
			ErrInvalid,
			MaxDiagnostics,
		)
	}
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("diagnostics[%d]: %w", index, err)
		}
	}
	return nil
}

func ContainsErrorDiagnostic(values []Diagnostic) bool {
	for _, value := range values {
		if value.Severity == DiagnosticError {
			return true
		}
	}
	return false
}

func AppendDiagnostics(
	current []Diagnostic,
	incoming ...Diagnostic,
) []Diagnostic {
	if len(incoming) == 0 {
		return current
	}
	if len(current) >= MaxDiagnostics {
		return current[:MaxDiagnostics]
	}
	remaining := MaxDiagnostics - len(current)
	if len(incoming) > remaining {
		incoming = incoming[:remaining]
	}
	return append(current, incoming...)
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type UUIDv7Generator struct{}

func (UUIDv7Generator) NewID(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	value, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate UUIDv7: %w", err)
	}
	return value.String(), nil
}

func ValidateRootID(value RootID) error {
	return ValidateUUIDv7("root ID", string(value))
}

func ValidateSourceID(value SourceID) error {
	return ValidateUUIDv7("source ID", string(value))
}

func ValidateRecordID(value RecordID) error {
	return ValidateUUIDv7("record ID", string(value))
}

func ValidateUUIDv7(label, value string) error {
	if !uuidV7Pattern.MatchString(value) {
		return fmt.Errorf("%w: %s must be a canonical UUIDv7", ErrInvalid, label)
	}
	return nil
}

func ValidateRootKind(value RootKind) error {
	return ValidateIdentifier("root kind", string(value), MaxKindBytes)
}

func ValidateSourceKind(value SourceKind) error {
	return ValidateIdentifier("source kind", string(value), MaxKindBytes)
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

func ValidateDigest(value Digest) error {
	if !digestPattern.MatchString(string(value)) {
		return fmt.Errorf(
			"%w: digest must be sha256:<64 lowercase hexadecimal characters>",
			ErrInvalid,
		)
	}
	return nil
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
