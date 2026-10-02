package diagnostic

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const (
	MaxDiagnosticCodeBytes    = 128
	MaxDiagnosticMessageBytes = 4096
	MaxDiagnostics            = 128
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Location struct {
	Locator            spec.Locator            `json:"locator,omitempty"`
	SubresourceLocator spec.SubresourceLocator `json:"subresourceLocator,omitempty"`
	Line               int                     `json:"line,omitempty"`
	Column             int                     `json:"column,omitempty"`
}

type Diagnostic struct {
	Severity Severity  `json:"severity"`
	Code     string    `json:"code"`
	Message  string    `json:"message"`
	Location *Location `json:"location,omitempty"`
}

func Validate(values []Diagnostic) error {
	if len(values) > MaxDiagnostics {
		return fmt.Errorf(
			"%w: diagnostics exceed %d entries",
			spec.ErrInvalid,
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

func (d Diagnostic) Validate() error {
	switch d.Severity {
	case SeverityError, SeverityWarning, SeverityInfo:
	default:
		return fmt.Errorf("%w: invalid diagnostic severity %q", spec.ErrInvalid, d.Severity)
	}
	if err := spec.ValidateIdentifier("diagnostic code", d.Code, MaxDiagnosticCodeBytes); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
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
		if err := d.Location.Locator.Validate(true); err != nil {
			return fmt.Errorf("diagnostic location: %w", err)
		}
	}
	if d.Location.SubresourceLocator != "" {
		if d.Location.Locator == "" {
			return fmt.Errorf(
				"%w: diagnostic subresource location requires a locator",
				spec.ErrInvalid,
			)
		}
		if err := d.Location.SubresourceLocator.Validate(); err != nil {
			return fmt.Errorf("diagnostic subresource location: %w", err)
		}
	}
	if d.Location.Line < 0 || d.Location.Column < 0 {
		return fmt.Errorf("%w: diagnostic line and column cannot be negative", spec.ErrInvalid)
	}
	return nil
}

func ContainsError(values []Diagnostic) bool {
	for _, value := range values {
		if value.Severity == SeverityError {
			return true
		}
	}
	return false
}

// BoundedMessage converts dynamically generated text into a value
// accepted by Diagnostic.Validate. It is intended for internal errors whose
// text can contain untrusted JSON keys, paths, or decoder output.
func BoundedMessage(value string) string {
	value = strings.ToValidUTF8(value, "\uFFFD")
	value = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, value)
	value = strings.TrimSpace(value)
	if value == "" {
		return "unspecified diagnostic"
	}
	if len(value) <= MaxDiagnosticMessageBytes {
		return value
	}

	const suffix = "..."
	value = value[:MaxDiagnosticMessageBytes-len(suffix)]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	value = strings.TrimRightFunc(value, unicode.IsSpace)
	if value == "" {
		return "diagnostic"
	}
	return value + suffix
}

func Append(
	current []Diagnostic,
	incoming ...Diagnostic,
) []Diagnostic {
	output := append(Clone(current), Clone(incoming)...)
	if len(output) <= MaxDiagnostics {
		return output
	}

	keep := make([]bool, len(output))
	for index := range keep {
		keep[index] = true
	}
	excess := len(output) - MaxDiagnostics
	for index := len(output) - 1; index >= 0 && excess > 0; index-- {
		if output[index].Severity == SeverityError {
			continue
		}
		keep[index] = false
		excess--
	}
	for index := len(output) - 1; index >= 0 && excess > 0; index-- {
		keep[index] = false
		excess--
	}

	trimmed := make([]Diagnostic, 0, MaxDiagnostics)
	for index, value := range output {
		if keep[index] {
			trimmed = append(trimmed, value)
		}
	}
	return trimmed
}

func Clone(values []Diagnostic) []Diagnostic {
	if values == nil {
		return nil
	}
	output := make([]Diagnostic, len(values))
	copy(output, values)
	for index := range output {
		if output[index].Location == nil {
			continue
		}
		location := *output[index].Location
		output[index].Location = &location
	}
	return output
}

func Equal(left, right []Diagnostic) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Severity != right[index].Severity ||
			left[index].Code != right[index].Code ||
			left[index].Message != right[index].Message {
			return false
		}
		if left[index].Location == nil || right[index].Location == nil {
			if left[index].Location != nil || right[index].Location != nil {
				return false
			}
			continue
		}
		if *left[index].Location != *right[index].Location {
			return false
		}
	}
	return true
}
