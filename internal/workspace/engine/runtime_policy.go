package engine

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
)

type RuntimeUse string

const (
	RuntimeUseContextPrompt RuntimeUse = "context-prompt"
	RuntimeUseSkill         RuntimeUse = "skill"
)

type RuntimeDisposition string

const (
	RuntimeAllowed     RuntimeDisposition = "allowed"
	RuntimeDenied      RuntimeDisposition = "denied"
	RuntimeUnavailable RuntimeDisposition = "unavailable"
)

type RuntimePolicyRequest struct {
	Use              RuntimeUse
	Workspace        Workspace
	Artifact         artifact.Artifact
	DefinitionDigest artifactstore.Digest
	SourceID         artifactstore.SourceID
}

type RuntimeDecision struct {
	Disposition RuntimeDisposition
	Code        string
	Message     string
}

func (d RuntimeDecision) Validate() error {
	switch d.Disposition {
	case RuntimeAllowed:
		if d.Code != "" || d.Message != "" {
			return fmt.Errorf(
				"%w: allowed runtime decision cannot contain denial details",
				ErrInvalidWorkspace,
			)
		}
		return nil

	case RuntimeDenied, RuntimeUnavailable:
		if err := artifactstore.ValidateIdentifier(
			"runtime policy diagnostic code",
			d.Code,
			artifactstore.MaxDiagnosticCodeBytes,
		); err != nil {
			return err
		}
		return artifactstore.ValidateRequiredText(
			"runtime policy diagnostic message",
			d.Message,
			artifactstore.MaxDiagnosticMessageBytes,
		)

	default:
		return fmt.Errorf(
			"%w: unsupported runtime disposition %q",
			ErrInvalidWorkspace,
			d.Disposition,
		)
	}
}

type SourceUsePolicy interface {
	Decide(
		ctx context.Context,
		request RuntimePolicyRequest,
	) RuntimeDecision
}

// ArtifactRuntimePolicy is the default local Workspace trust boundary.
//
// Discovery and management remain available. Runtime use is enabled by default
// unless the Artifact-local RuntimeDisabled flag explicitly disables it.
type ArtifactRuntimePolicy struct{}

func NewArtifactRuntimePolicy() *ArtifactRuntimePolicy {
	return &ArtifactRuntimePolicy{}
}

func (*ArtifactRuntimePolicy) Decide(
	ctx context.Context,
	request RuntimePolicyRequest,
) RuntimeDecision {
	if err := ctx.Err(); err != nil {
		return RuntimeDecision{
			Disposition: RuntimeUnavailable,
			Code:        DiagnosticCodeRuntimeUnavailable,
			Message:     "runtime policy evaluation was cancelled",
		}
	}
	if !request.Workspace.Collection.Enabled {
		return RuntimeDecision{
			Disposition: RuntimeUnavailable,
			Code:        DiagnosticCodeRuntimeUnavailable,
			Message:     "the Workspace is disabled",
		}
	}
	if !request.Artifact.Enabled ||
		request.Artifact.State != artifact.StateAvailable {
		return RuntimeDecision{
			Disposition: RuntimeUnavailable,
			Code:        DiagnosticCodeRuntimeUnavailable,
			Message:     "the Workspace Artifact is not enabled and available",
		}
	}
	disabled, err := ArtifactRuntimeDisabled(request.Artifact)
	if err != nil {
		return RuntimeDecision{
			Disposition: RuntimeUnavailable,
			Code:        DiagnosticCodeRuntimeUnavailable,
			Message:     "the Workspace Artifact has invalid local runtime policy data",
		}
	}
	if disabled {
		return RuntimeDecision{
			Disposition: RuntimeDenied,
			Code:        DiagnosticCodeRuntimeDenied,
			Message:     "runtime use is disabled for this Workspace Artifact",
		}
	}
	return RuntimeDecision{Disposition: RuntimeAllowed}
}

func RuntimeDecisionDiagnostic(
	decision RuntimeDecision,
	value artifact.Artifact,
) artifactstore.Diagnostic {
	severity := artifactstore.DiagnosticWarning
	if decision.Disposition == RuntimeUnavailable {
		severity = artifactstore.DiagnosticError
	}
	return artifactstore.Diagnostic{
		Severity: severity,
		Code:     decision.Code,
		Message:  decision.Message,
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}
