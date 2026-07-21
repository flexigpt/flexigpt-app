package artifactstore

import "fmt"

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
		if d.Location.Locator == "" {
			return fmt.Errorf(
				"%w: diagnostic subresource location requires a locator",
				ErrInvalid,
			)
		}
		if err := ValidateSubresourceLocator(d.Location.SubresourceLocator); err != nil {
			return fmt.Errorf("diagnostic subresource location: %w", err)
		}
	}
	if d.Location.Line < 0 || d.Location.Column < 0 {
		return fmt.Errorf("%w: diagnostic line and column cannot be negative", ErrInvalid)
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

func CloneDiagnostics(values []Diagnostic) []Diagnostic {
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

func EqualDiagnostics(left, right []Diagnostic) bool {
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

func AppendDiagnostics(
	current []Diagnostic,
	incoming ...Diagnostic,
) []Diagnostic {
	if len(incoming) == 0 {
		return CloneDiagnostics(current)
	}
	if len(current) >= MaxDiagnostics {
		return CloneDiagnostics(current[:MaxDiagnostics])
	}
	remaining := MaxDiagnostics - len(current)
	if len(incoming) > remaining {
		incoming = incoming[:remaining]
	}
	return append(CloneDiagnostics(current), CloneDiagnostics(incoming)...)
}
