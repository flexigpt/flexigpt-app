package engine

import (
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

const (
	DiagnosticCodeArtifactInvalid         = "workspace.artifact.invalid"
	DiagnosticCodeContextInvalidContent   = "workspace.context.invalid-content"
	DiagnosticCodeContextInvalidUTF8      = "workspace.context.invalid-utf8"
	DiagnosticCodeDefinitionInvalid       = "workspace.definition.invalid"
	DiagnosticCodeRecordSchemaUnsupported = "workspace.record.schema-unsupported"
	DiagnosticCodeSkillInvalid            = "workspace.skill.invalid"
)

func WorkspaceArtifactErrorDiagnostics(
	locator artifactstore.Locator,
	err error,
) []artifactstore.Diagnostic {
	return WorkspaceArtifactDiagnostics(
		locator,
		DiagnosticCodeArtifactInvalid,
		err.Error(),
	)
}

func WorkspaceArtifactDiagnostics(
	locator artifactstore.Locator,
	code string,
	message string,
) []artifactstore.Diagnostic {
	for len(message) > artifactstore.MaxDiagnosticMessageBytes {
		_, size := utf8.DecodeLastRuneInString(message)
		message = message[:len(message)-size]
	}
	return []artifactstore.Diagnostic{{
		Severity: artifactstore.DiagnosticError,
		Code:     code,
		Message:  message,
		Location: &artifactstore.DiagnosticLocation{
			Locator: locator,
		},
	}}
}
