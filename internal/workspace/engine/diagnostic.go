package engine

import "github.com/flexigpt/flexigpt-app/internal/artifactstore"

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
	return []artifactstore.Diagnostic{{
		Severity: artifactstore.DiagnosticError,
		Code:     code,
		Message:  artifactstore.BoundedDiagnosticMessage(message),
		Location: &artifactstore.DiagnosticLocation{
			Locator: locator,
		},
	}}
}
