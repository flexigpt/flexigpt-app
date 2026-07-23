package engine

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
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
	Record           record.Record
	DefinitionDigest artifactstore.Digest
	SourceID         artifactstore.SourceID
	TrustReference   string
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

// RecordRuntimePolicy is the default local Workspace trust boundary.
//
// Discovery and management remain available, but runtime handoff requires the
// record-local RuntimeAllowed flag to have been set explicitly.
type RecordRuntimePolicy struct{}

func NewRecordRuntimePolicy() *RecordRuntimePolicy {
	return &RecordRuntimePolicy{}
}

func (*RecordRuntimePolicy) Decide(
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
	if !request.Workspace.Root.Enabled {
		return RuntimeDecision{
			Disposition: RuntimeUnavailable,
			Code:        DiagnosticCodeRuntimeUnavailable,
			Message:     "the Workspace is disabled",
		}
	}
	if !request.Record.Enabled ||
		request.Record.State != record.StateAvailable {
		return RuntimeDecision{
			Disposition: RuntimeUnavailable,
			Code:        DiagnosticCodeRuntimeUnavailable,
			Message:     "the Workspace record is not enabled and available",
		}
	}
	allowed, err := RecordRuntimeAllowed(request.Record)
	if err != nil {
		return RuntimeDecision{
			Disposition: RuntimeUnavailable,
			Code:        DiagnosticCodeRuntimeUnavailable,
			Message:     "the Workspace record has invalid local runtime policy data",
		}
	}
	if !allowed {
		return RuntimeDecision{
			Disposition: RuntimeDenied,
			Code:        DiagnosticCodeRuntimeDenied,
			Message:     "runtime use requires explicit approval for this Workspace record",
		}
	}
	return RuntimeDecision{Disposition: RuntimeAllowed}
}

func RuntimeDecisionDiagnostic(
	decision RuntimeDecision,
	value record.Record,
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
			Locator:            value.Occurrence.Locator,
			SubresourceLocator: value.Occurrence.SubresourceLocator,
		},
	}
}
