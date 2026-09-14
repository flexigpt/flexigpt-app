package domain

import "errors"

var (
	ErrInvalidWorkspace    = errors.New("workspace: invalid")
	ErrNotWorkspace        = errors.New("workspace: artifact is not a workspace")
	ErrReferenceUnresolved = errors.New("workspace: reference unresolved")
)

const (
	DiagnosticCodeArtifactUnavailable = "workspace.artifact.unavailable"
	DiagnosticCodeArtifactUnresolved  = "workspace.artifact.unresolved"
	DiagnosticCodeRuntimeDisabled     = "workspace.runtime.disabled"
	DiagnosticCodeProjectionInvalid   = "workspace.projection.invalid"
)
