package artifactstore

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
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
