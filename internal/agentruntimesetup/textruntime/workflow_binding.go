package textruntime

import (
	"context"
	"errors"
	"strings"
	"time"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/workflowruntime"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/texttool"
)

const (
	ApplyUnifiedDiffActivityID      = "flexigpt.text.apply-unified-diff"
	ApplyUnifiedDiffActivityVersion = "1"

	applyUnifiedDiffBindingVersion     = "text-apply-unified-diff/v1"
	applyUnifiedDiffBindingFingerprint = "flexigpt.text.apply-unified-diff"
	applyUnifiedDiffReceiptSystem      = "flexigpt.text.apply-unified-diff"
	applyUnifiedDiffUncertainCode      = "apply_unified_diff_uncertain"
	applyUnifiedDiffMissingOutputCode  = "apply_unified_diff_missing_output"
)

type applyUnifiedDiffActivityBinding struct {
	workflowruntime.WorkBinding
}

type boundApplyUnifiedDiff struct {
	Input texttool.ApplyUnifiedDiffArgs
}

// ApplyUnifiedDiffActivityBinding returns the text-domain Workflow binding.
// Workflow composition registers it, but does not own diff behavior.
func ApplyUnifiedDiffActivityBinding() workflowruntime.WorkBinding {
	return applyUnifiedDiffActivityBinding{}
}

func (applyUnifiedDiffActivityBinding) ID() string {
	return ApplyUnifiedDiffActivityID
}

func (applyUnifiedDiffActivityBinding) Version() string {
	return ApplyUnifiedDiffActivityVersion
}

func (applyUnifiedDiffActivityBinding) Effect() sessionv3Runtime.EffectContract {
	return sessionv3Runtime.EffectContract{
		Kind: sessionv3Runtime.EffectKindNonIdempotent,
	}
}

func (applyUnifiedDiffActivityBinding) WorkflowPortCapabilities() workflowruntime.WorkBindingPortCapabilities {
	return workflowruntime.WorkBindingPortCapabilities{
		Context:          false,
		InputPorts:       false,
		FinalOutputPorts: true,
		OutputStream:     false,
	}
}

func (applyUnifiedDiffActivityBinding) Bind(
	_ context.Context,
	request workflowruntime.WorkBindingRequest,
) (sessionv3Runtime.SemanticBinding, any, error) {
	input, err := sessionv3Runtime.DecodeCanonicalValue[texttool.ApplyUnifiedDiffArgs](request.Input)
	if err != nil {
		return sessionv3Runtime.SemanticBinding{}, nil, errors.New(
			"apply unified diff input is invalid",
		)
	}

	return sessionv3Runtime.SemanticBinding{
		Version:           applyUnifiedDiffBindingVersion,
		TargetFingerprint: applyUnifiedDiffBindingFingerprint,
		BoundAt:           time.Now().UTC(),
	}, boundApplyUnifiedDiff{Input: input}, nil
}

func (applyUnifiedDiffActivityBinding) Execute(
	ctx context.Context,
	_ workflowruntime.WorkBindingRequest,
	private any,
) (sessionv3Runtime.WorkExecutionResult, error) {
	bound, found := private.(boundApplyUnifiedDiff)
	if !found {
		return uncertainDiffFailure(
			applyUnifiedDiffUncertainCode,
			errors.New("apply unified diff binding state is unavailable"),
			nil,
		), nil
	}

	result, err := llmtoolsutil.ApplyUnifiedDiff(ctx, bound.Input)
	if err != nil {
		var output *sessionv3Runtime.WorkItemOutput
		if result != nil {
			output = &sessionv3Runtime.WorkItemOutput{
				Value: sessionv3Runtime.MustCanonicalValue(*result),
			}
		}
		return uncertainDiffFailure(
			applyUnifiedDiffUncertainCode,
			err,
			output,
		), nil
	}
	if result == nil {
		return uncertainDiffFailure(
			applyUnifiedDiffMissingOutputCode,
			errors.New("apply unified diff returned no output"),
			nil,
		), nil
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: sessionv3Runtime.MustCanonicalValue(*result),
			Receipt: &sessionv3Runtime.ExternalReceipt{
				System:      applyUnifiedDiffReceiptSystem,
				Reference:   ApplyUnifiedDiffActivityID,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, nil
}

func (applyUnifiedDiffActivityBinding) ResolveWait(
	_ context.Context,
	_ workflowruntime.WorkBindingWaitRequest,
	_ any,
) (sessionv3Runtime.WaitResolutionResult, error) {
	return sessionv3Runtime.WaitResolutionResult{}, errors.New(
		"apply unified diff does not expose waits",
	)
}

func uncertainDiffFailure(
	code string,
	cause error,
	output *sessionv3Runtime.WorkItemOutput,
) sessionv3Runtime.WorkExecutionResult {
	message := "binding execution failed"
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		message = strings.ToLower(cause.Error())
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: output,
		Error: &sessionv3Runtime.SanitizedError{
			Code:     code,
			Category: sessionv3Runtime.ErrorCategoryUncertainty,
			Message:  message,
		},
		Confirmed: false,
	}
}
