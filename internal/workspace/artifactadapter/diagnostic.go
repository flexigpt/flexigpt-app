package artifactadapter

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
)

const (
	DiagnosticCodeArtifactInvalid           = "workspace.artifact.invalid"
	DiagnosticCodeContextInvalidContent     = "workspace.context.invalid-content"
	DiagnosticCodeContextInvalidUTF8        = "workspace.context.invalid-utf8"
	DiagnosticCodeDefinitionInvalid         = "workspace.definition.invalid"
	DiagnosticCodeArtifactSchemaUnsupported = "workspace.artifact.schema-unsupported"
	DiagnosticCodeArtifactKindMismatch      = "workspace.artifact.kind-mismatch"
	DiagnosticCodeProjectionInvalid         = "workspace.projection.invalid"
	DiagnosticCodeArtifactUnavailable       = "workspace.artifact.unavailable"
	DiagnosticCodeArtifactUnresolved        = "workspace.artifact.unresolved"
	DiagnosticCodeRuntimeDenied             = "workspace.runtime.denied"
	DiagnosticCodeRuntimeUnavailable        = "workspace.runtime.unavailable"
	DiagnosticCodeCatalogStale              = "workspace.catalog.stale"
	DiagnosticCodeCatalogDecoderStale       = "workspace.catalog.decoder-stale"
	DiagnosticCodeCatalogPolicyStale        = "workspace.catalog.policy-stale"
	DiagnosticCodeSkillInvalid              = "workspace.skill.invalid"
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
