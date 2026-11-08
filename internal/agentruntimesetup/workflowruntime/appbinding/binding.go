package appbinding

import (
	"context"
	"errors"
	"strings"
	"time"

	attachmentRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/attachment"
	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/workflowruntime"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/texttool"
)

const (
	NormalizeURLServiceID      = "flexigpt.url.normalize"
	NormalizeURLServiceVersion = "1"

	ApplyUnifiedDiffActivityID      = "flexigpt.text.apply-unified-diff"
	ApplyUnifiedDiffActivityVersion = "1"

	applicationServiceBindingVersion  = "session-v3-application-service/v1"
	applicationActivityBindingVersion = "session-v3-application-activity/v1"

	normalizeURLBindingFingerprint      = "flexigpt.url.normalize"
	applyUnifiedDiffBindingFingerprint  = "flexigpt.text.apply-unified-diff"
	normalizeURLReceiptSystem           = "flexigpt.url.normalize"
	applyUnifiedDiffReceiptSystem       = "flexigpt.text.apply-unified-diff"
	normalizeURLUnavailableCode         = "url_normalization_failed"
	applyUnifiedDiffUncertainCode       = "apply_unified_diff_uncertain"
	applyUnifiedDiffMissingOutputCode   = "apply_unified_diff_missing_output"
	unsupportedBindingContextMessage    = "binding context is unavailable"
	unsupportedBindingInputPortMessage  = "binding input ports are unavailable"
	normalizeURLInputValidationMessage  = "url normalization input is invalid"
	normalizeURLResultValidationMessage = "url normalization returned no url reference"
)

type normalizeURLInput struct {
	URL string `json:"url"`
}

type normalizeURLOutput struct {
	URL            string   `json:"url"`
	Normalized     string   `json:"normalized"`
	Label          string   `json:"label,omitempty"`
	Mode           string   `json:"mode"`
	AvailableModes []string `json:"availableModes,omitempty"`
}

type normalizeURLService struct{}

type applyUnifiedDiffActivity struct{}

func Register(
	registry *workflowruntime.BindingRegistry,
) error {
	if registry == nil {
		return errors.New("application workflow binding registry is unavailable")
	}
	if err := registry.RegisterService(normalizeURLService{}); err != nil {
		return err
	}
	return registry.RegisterActivity(applyUnifiedDiffActivity{})
}

func (normalizeURLService) ID() string {
	return NormalizeURLServiceID
}

func (normalizeURLService) Version() string {
	return NormalizeURLServiceVersion
}

func (normalizeURLService) Effect() sessionv3Runtime.EffectContract {
	return sessionv3Runtime.EffectContract{
		Kind: sessionv3Runtime.EffectKindRead,
	}
}

func (normalizeURLService) Bind(
	_ context.Context,
	request workflowruntime.WorkBindingRequest,
) (sessionv3Runtime.SemanticBinding, any, error) {
	if err := requireBindingWithoutInputPorts(request); err != nil {
		return sessionv3Runtime.SemanticBinding{}, nil, err
	}
	if _, err := decodeNormalizeURLInput(request.Input); err != nil {
		return sessionv3Runtime.SemanticBinding{}, nil, err
	}
	return applicationBinding(
		applicationServiceBindingVersion,
		normalizeURLBindingFingerprint,
	), nil, nil
}

func (normalizeURLService) Execute(
	ctx context.Context,
	request workflowruntime.WorkBindingRequest,
	_ any,
) (sessionv3Runtime.WorkExecutionResult, error) {
	if err := requireBindingWithoutInputPorts(request); err != nil {
		return confirmedBindingFailure(
			normalizeURLUnavailableCode,
			err,
		), nil
	}
	input, err := decodeNormalizeURLInput(request.Input)
	if err != nil {
		return confirmedBindingFailure(
			normalizeURLUnavailableCode,
			err,
		), nil
	}

	attachment, err := attachmentRuntime.BuildAttachmentForURLWithContext(
		ctx,
		input.URL,
	)
	if err != nil {
		return confirmedBindingFailure(
			normalizeURLUnavailableCode,
			err,
		), nil
	}
	if attachment == nil || attachment.URLRef == nil {
		return confirmedBindingFailure(
			normalizeURLUnavailableCode,
			errors.New(normalizeURLResultValidationMessage),
		), nil
	}

	modes := make(
		[]string,
		0,
		len(attachment.AvailableContentBlockModes),
	)
	for _, mode := range attachment.AvailableContentBlockModes {
		modes = append(modes, string(mode))
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: sessionv3Runtime.MustCanonicalValue(normalizeURLOutput{
				URL:            attachment.URLRef.URL,
				Normalized:     attachment.URLRef.Normalized,
				Label:          attachment.Label,
				Mode:           string(attachment.Mode),
				AvailableModes: modes,
			}),
			Receipt: &sessionv3Runtime.ExternalReceipt{
				System:      normalizeURLReceiptSystem,
				Reference:   attachment.URLRef.Normalized,
				ConfirmedAt: time.Now().UTC(),
			},
		},
		Confirmed: true,
	}, nil
}

func (normalizeURLService) ResolveWait(
	_ context.Context,
	_ workflowruntime.WorkBindingWaitRequest,
	_ any,
) (sessionv3Runtime.WaitResolutionResult, error) {
	return sessionv3Runtime.WaitResolutionResult{}, errors.New(
		"url normalization does not expose waits",
	)
}

func (applyUnifiedDiffActivity) ID() string {
	return ApplyUnifiedDiffActivityID
}

func (applyUnifiedDiffActivity) Version() string {
	return ApplyUnifiedDiffActivityVersion
}

func (applyUnifiedDiffActivity) Effect() sessionv3Runtime.EffectContract {
	return sessionv3Runtime.EffectContract{
		Kind: sessionv3Runtime.EffectKindNonIdempotent,
	}
}

func (applyUnifiedDiffActivity) Bind(
	_ context.Context,
	request workflowruntime.WorkBindingRequest,
) (sessionv3Runtime.SemanticBinding, any, error) {
	if err := requireBindingWithoutInputPorts(request); err != nil {
		return sessionv3Runtime.SemanticBinding{}, nil, err
	}
	if _, err := sessionv3Runtime.DecodeCanonicalValue[texttool.ApplyUnifiedDiffArgs](request.Input); err != nil {
		return sessionv3Runtime.SemanticBinding{}, nil, errors.New(
			"apply unified diff input is invalid",
		)
	}
	return applicationBinding(
		applicationActivityBindingVersion,
		applyUnifiedDiffBindingFingerprint,
	), nil, nil
}

func (applyUnifiedDiffActivity) Execute(
	ctx context.Context,
	request workflowruntime.WorkBindingRequest,
	_ any,
) (sessionv3Runtime.WorkExecutionResult, error) {
	if err := requireBindingWithoutInputPorts(request); err != nil {
		return uncertainBindingFailure(
			applyUnifiedDiffUncertainCode,
			err,
			nil,
		), nil
	}
	input, err := sessionv3Runtime.DecodeCanonicalValue[texttool.ApplyUnifiedDiffArgs](request.Input)
	if err != nil {
		return uncertainBindingFailure(
			applyUnifiedDiffUncertainCode,
			errors.New("apply unified diff input is invalid"),
			nil,
		), nil
	}

	result, applyErr := llmtoolsutil.ApplyUnifiedDiff(ctx, input)
	if applyErr != nil {
		var output *sessionv3Runtime.WorkItemOutput
		if result != nil {
			output = &sessionv3Runtime.WorkItemOutput{
				Value: sessionv3Runtime.MustCanonicalValue(*result),
			}
		}
		return uncertainBindingFailure(
			applyUnifiedDiffUncertainCode,
			applyErr,
			output,
		), nil
	}
	if result == nil {
		return uncertainBindingFailure(
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

func (applyUnifiedDiffActivity) ResolveWait(
	_ context.Context,
	_ workflowruntime.WorkBindingWaitRequest,
	_ any,
) (sessionv3Runtime.WaitResolutionResult, error) {
	return sessionv3Runtime.WaitResolutionResult{}, errors.New(
		"apply unified diff does not expose waits",
	)
}

func applicationBinding(
	version string,
	fingerprint string,
) sessionv3Runtime.SemanticBinding {
	return sessionv3Runtime.SemanticBinding{
		Version:           version,
		TargetFingerprint: fingerprint,
		BoundAt:           time.Now().UTC(),
	}
}

func requireBindingWithoutInputPorts(
	request workflowruntime.WorkBindingRequest,
) error {
	if len(request.Context) != 0 {
		return errors.New(unsupportedBindingContextMessage)
	}
	if len(request.Ports) != 0 {
		return errors.New(unsupportedBindingInputPortMessage)
	}
	return nil
}

func decodeNormalizeURLInput(
	value sessionv3Runtime.CanonicalValue,
) (normalizeURLInput, error) {
	input, err := sessionv3Runtime.DecodeCanonicalValue[normalizeURLInput](
		value,
	)
	if err != nil || strings.TrimSpace(input.URL) == "" {
		return normalizeURLInput{}, errors.New(
			normalizeURLInputValidationMessage,
		)
	}
	return input, nil
}

func confirmedBindingFailure(
	code string,
	cause error,
) sessionv3Runtime.WorkExecutionResult {
	return sessionv3Runtime.WorkExecutionResult{
		Error: &sessionv3Runtime.SanitizedError{
			Code:     code,
			Category: sessionv3Runtime.ErrorCategoryExecution,
			Message:  lowerError(cause),
		},
		Confirmed: true,
	}
}

func uncertainBindingFailure(
	code string,
	cause error,
	output *sessionv3Runtime.WorkItemOutput,
) sessionv3Runtime.WorkExecutionResult {
	return sessionv3Runtime.WorkExecutionResult{
		Output: output,
		Error: &sessionv3Runtime.SanitizedError{
			Code:     code,
			Category: sessionv3Runtime.ErrorCategoryUncertainty,
			Message:  lowerError(cause),
		},
		Confirmed: false,
	}
}

func lowerError(
	cause error,
) string {
	if cause == nil || strings.TrimSpace(cause.Error()) == "" {
		return "binding execution failed"
	}
	return strings.ToLower(cause.Error())
}
