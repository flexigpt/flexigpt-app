package artifactadapter

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

const (
	DiagnosticCodeArtifactInvalid           = spec.DiagnosticCodeArtifactInvalid
	DiagnosticCodeContextInvalidContent     = spec.DiagnosticCodeContextInvalidContent
	DiagnosticCodeContextInvalidUTF8        = spec.DiagnosticCodeContextInvalidUTF8
	DiagnosticCodeArtifactSchemaUnsupported = spec.DiagnosticCodeArtifactSchemaUnsupported
	DiagnosticCodeArtifactKindMismatch      = spec.DiagnosticCodeArtifactKindMismatch
	DiagnosticCodeProjectionInvalid         = spec.DiagnosticCodeProjectionInvalid
	DiagnosticCodeArtifactUnavailable       = spec.DiagnosticCodeArtifactUnavailable
	DiagnosticCodeArtifactUnresolved        = spec.DiagnosticCodeArtifactUnresolved
	DiagnosticCodeRuntimeDenied             = spec.DiagnosticCodeRuntimeDenied
	DiagnosticCodeRuntimeUnavailable        = spec.DiagnosticCodeRuntimeUnavailable
	DiagnosticCodeCatalogStale              = spec.DiagnosticCodeCatalogStale
	DiagnosticCodeCatalogDecoderStale       = spec.DiagnosticCodeCatalogDecoderStale
	DiagnosticCodeCatalogPlanStale          = spec.DiagnosticCodeCatalogPlanStale
)

func WorkspaceArtifactErrorDiagnostics(
	locator basespec.Locator,
	err error,
) []diagnostic.Diagnostic {
	return WorkspaceArtifactDiagnostics(
		locator,
		DiagnosticCodeArtifactInvalid,
		err.Error(),
	)
}

func WorkspaceArtifactDiagnostics(
	locator basespec.Locator,
	code string,
	message string,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.DiagnosticError,
		Code:     code,
		Message:  diagnostic.BoundedDiagnosticMessage(message),
		Location: &diagnostic.DiagnosticLocation{
			Locator: locator,
		},
	}}
}
