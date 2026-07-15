package artifactstore

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

// DiagnosticValidationError reports non-recoverable diagnostics emitted by a
// registered root, collection, or frontend validator.
type DiagnosticValidationError struct {
	Scope       string
	Diagnostics []spec.Diagnostic
}

func (e *DiagnosticValidationError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s validation reported %d error diagnostic(s)", e.Scope, len(e.Diagnostics))
}

func (e *DiagnosticValidationError) Unwrap() error {
	return spec.ErrInvalidRequest
}

func errorDiagnostics(scope string, diagnostics []spec.Diagnostic) error {
	if err := validate.ValidateDiagnostics(diagnostics); err != nil {
		return fmt.Errorf(
			"%w: %s returned invalid diagnostics: %w",
			spec.ErrInvalidRequest,
			scope,
			err,
		)
	}
	errorsOnly := make([]spec.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == spec.DiagnosticSeverityError {
			errorsOnly = append(errorsOnly, diagnostic)
		}
	}
	if len(errorsOnly) == 0 {
		return nil
	}
	return &DiagnosticValidationError{Scope: scope, Diagnostics: errorsOnly}
}

func appendBoundedDiagnostics(
	current []spec.Diagnostic,
	incoming ...spec.Diagnostic,
) []spec.Diagnostic {
	if len(incoming) == 0 {
		return current
	}
	maximum := spec.MaxDiagnosticsPerEntity
	if len(current)+len(incoming) <= maximum {
		return append(current, incoming...)
	}
	truncated := spec.Diagnostic{
		Severity: spec.DiagnosticSeverityWarning,
		Code:     "artifactstore.diagnostics.truncated",
		Message:  "additional diagnostics were omitted because the entity diagnostic limit was reached",
	}
	if len(current) >= maximum {
		current = current[:maximum]
		current[maximum-1] = truncated
		return current
	}
	remaining := maximum - len(current)
	if remaining > 1 {
		current = append(current, incoming[:min(remaining-1, len(incoming))]...)
	}
	current = append(current, truncated)
	return current
}
