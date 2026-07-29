package diagnostic

import (
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

func TestDiagnosticsAreBoundedOwnedAndErrorPreserving(t *testing.T) {
	t.Parallel()

	message := BoundedDiagnosticMessage("\x00  hello\t" + strings.Repeat("é", MaxDiagnosticMessageBytes))
	if err := basespec.ValidateRequiredText("diagnostic message", message, MaxDiagnosticMessageBytes); err != nil {
		t.Fatalf("bounded message is invalid: %v; message=%q", err, message)
	}
	if len(message) > MaxDiagnosticMessageBytes {
		t.Fatalf("bounded message length=%d, maximum=%d", len(message), MaxDiagnosticMessageBytes)
	}
	if !strings.HasSuffix(message, "...") {
		t.Fatalf("bounded long message=%q, want ellipsis", message)
	}

	location := &DiagnosticLocation{Locator: "file.json", Line: 3, Column: 7}
	original := []Diagnostic{{
		Severity: DiagnosticError,
		Code:     "test.error",
		Message:  "failure",
		Location: location,
	}}
	cloned := CloneDiagnostics(original)
	location.Line = 99
	if cloned[0].Location == nil || cloned[0].Location.Line != 3 {
		t.Fatalf("CloneDiagnostics did not clone location: %#v", cloned)
	}

	errorsOnly := make([]Diagnostic, 0, MaxDiagnostics)
	for range MaxDiagnostics {
		errorsOnly = append(errorsOnly, Diagnostic{
			Severity: DiagnosticError,
			Code:     "test.error",
			Message:  "must remain",
		})
	}
	trimmed := AppendDiagnostics(errorsOnly,
		Diagnostic{Severity: DiagnosticWarning, Code: "test.warning", Message: "drop first"},
		Diagnostic{Severity: DiagnosticInfo, Code: "test.info", Message: "drop second"},
	)
	if len(trimmed) != MaxDiagnostics {
		t.Fatalf("trimmed diagnostics=%d, want %d", len(trimmed), MaxDiagnostics)
	}
	for index, diagnostic := range trimmed {
		if diagnostic.Severity != DiagnosticError {
			t.Fatalf("diagnostic %d=%#v, errors must be preserved", index, diagnostic)
		}
	}
}
