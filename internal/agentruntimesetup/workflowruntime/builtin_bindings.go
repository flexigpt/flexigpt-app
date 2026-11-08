package workflowruntime

import (
	"context"
	"errors"
	"strings"
	"time"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/attachmentruntime"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/texttool"
)

const (
	NormalizeURLServiceID      = "flexigpt.url.normalize"
	NormalizeURLServiceVersion = "1"

	ApplyUnifiedDiffActivityID      = "flexigpt.text.apply-unified-diff"
	ApplyUnifiedDiffActivityVersion = "1"

	builtinServiceBindingVersion  = "session-v3-builtin-service/v1"
	builtinActivityBindingVersion = "session-v3-builtin-activity/v1"

	normalizeURLBindingFingerprint     = "flexigpt.url.normalize"
	applyUnifiedDiffBindingFingerprint = "flexigpt.text.apply-unified-diff"

	normalizeURLReceiptSystem     = "flexigpt.url.normalize"
	applyUnifiedDiffReceiptSystem = "flexigpt.text.apply-unified-diff"

	normalizeURLUnavailableCode       = "url_normalization_failed"
	applyUnifiedDiffUncertainCode     = "apply_unified_diff_uncertain"
	applyUnifiedDiffMissingOutputCode = "apply_unified_diff_missing_output"
)

type normalizeURLInput struct {
	URL string `json:"url"`
}

type normalizeURLServiceBinding struct{}

type applyUnifiedDiffActivityBinding struct{}

type boundNormalizeURL struct {
	Input normalizeURLInput
}

type boundApplyUnifiedDiff struct {
	Input texttool.ApplyUnifiedDiffArgs
}

func NormalizeURLServiceBinding() WorkBinding {
	return normalizeURLServiceBinding{}
}

func ApplyUnifiedDiffActivityBinding() WorkBinding {
	return applyUnifiedDiffActivityBinding{}
}

func (normalizeURLServiceBinding) ID() string {
	return NormalizeURLServiceID
}

func (normalizeURLServiceBinding) Version() string {
	return NormalizeURLServiceVersion
}

func (normalizeURLServiceBinding) Effect() sessionv3Runtime.EffectContract {
	return sessionv3Runtime.EffectContract{
		Kind: sessionv3Runtime.EffectKindRead,
	}
}

func (normalizeURLServiceBinding) Bind(
	_ context.Context,
	request WorkBindingRequest,
) (sessionv3Runtime.SemanticBinding, any, error) {
	input, err := decodeNormalizeURLInput(request.Input)
	if err != nil {
		return sessionv3Runtime.SemanticBinding{}, nil, err
	}

	return builtinBinding(
		builtinServiceBindingVersion,
		normalizeURLBindingFingerprint,
	), boundNormalizeURL{
		Input: input,
	}, nil
}

func (normalizeURLServiceBinding) Execute(
	ctx context.Context,
	_ WorkBindingRequest,
	private any,
) (sessionv3Runtime.WorkExecutionResult, error) {
	bound, ok := private.(boundNormalizeURL)
	if !ok {
		return confirmedBuiltinBindingFailure(
			normalizeURLUnavailableCode,
			errors.New("url normalization binding state is unavailable"),
		), nil
	}

	result, err := attachmentruntime.NormalizeURL(ctx, bound.Input.URL)
	if err != nil {
		return confirmedBuiltinBindingFailure(
			normalizeURLUnavailableCode,
			err,
		), nil
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: sessionv3Runtime.MustCanonicalValue(result),
			Receipt: &sessionv3Runtime.ExternalReceipt{
				System:      normalizeURLReceiptSystem,
				Reference:   result.Normalized,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, nil
}

func (normalizeURLServiceBinding) ResolveWait(
	_ context.Context,
	_ WorkBindingWaitRequest,
	_ any,
) (sessionv3Runtime.WaitResolutionResult, error) {
	return sessionv3Runtime.WaitResolutionResult{}, errors.New(
		"url normalization does not expose waits",
	)
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

func (applyUnifiedDiffActivityBinding) Bind(
	_ context.Context,
	request WorkBindingRequest,
) (sessionv3Runtime.SemanticBinding, any, error) {
	input, err := sessionv3Runtime.DecodeCanonicalValue[texttool.ApplyUnifiedDiffArgs](request.Input)
	if err != nil {
		return sessionv3Runtime.SemanticBinding{}, nil, errors.New(
			"apply unified diff input is invalid",
		)
	}

	return builtinBinding(
		builtinActivityBindingVersion,
		applyUnifiedDiffBindingFingerprint,
	), boundApplyUnifiedDiff{
		Input: input,
	}, nil
}

func (applyUnifiedDiffActivityBinding) Execute(
	ctx context.Context,
	_ WorkBindingRequest,
	private any,
) (sessionv3Runtime.WorkExecutionResult, error) {
	bound, ok := private.(boundApplyUnifiedDiff)
	if !ok {
		return uncertainBuiltinBindingFailure(
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
		return uncertainBuiltinBindingFailure(
			applyUnifiedDiffUncertainCode,
			err,
			output,
		), nil
	}
	if result == nil {
		return uncertainBuiltinBindingFailure(
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
	_ WorkBindingWaitRequest,
	_ any,
) (sessionv3Runtime.WaitResolutionResult, error) {
	return sessionv3Runtime.WaitResolutionResult{}, errors.New(
		"apply unified diff does not expose waits",
	)
}

func builtinBinding(
	version string,
	fingerprint string,
) sessionv3Runtime.SemanticBinding {
	return sessionv3Runtime.SemanticBinding{
		Version:           version,
		TargetFingerprint: fingerprint,
		BoundAt:           time.Now().UTC(),
	}
}

func decodeNormalizeURLInput(
	value sessionv3Runtime.CanonicalValue,
) (normalizeURLInput, error) {
	input, err := sessionv3Runtime.DecodeCanonicalValue[normalizeURLInput](
		value,
	)
	if err != nil || strings.TrimSpace(input.URL) == "" {
		return normalizeURLInput{}, errors.New(
			"url normalization input is invalid",
		)
	}
	return input, nil
}

func confirmedBuiltinBindingFailure(
	code string,
	cause error,
) sessionv3Runtime.WorkExecutionResult {
	return sessionv3Runtime.WorkExecutionResult{
		Error: &sessionv3Runtime.SanitizedError{
			Code:     code,
			Category: sessionv3Runtime.ErrorCategoryExecution,
			Message:  normalizedBindingError(cause),
		},
		Confirmed: true,
	}
}

func uncertainBuiltinBindingFailure(
	code string,
	cause error,
	output *sessionv3Runtime.WorkItemOutput,
) sessionv3Runtime.WorkExecutionResult {
	return sessionv3Runtime.WorkExecutionResult{
		Output: output,
		Error: &sessionv3Runtime.SanitizedError{
			Code:     code,
			Category: sessionv3Runtime.ErrorCategoryUncertainty,
			Message:  normalizedBindingError(cause),
		},
		Confirmed: false,
	}
}

func normalizedBindingError(cause error) string {
	if cause == nil || strings.TrimSpace(cause.Error()) == "" {
		return "binding execution failed"
	}
	return strings.ToLower(cause.Error())
}
