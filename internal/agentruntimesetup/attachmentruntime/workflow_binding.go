package attachmentruntime

import (
	"context"
	"errors"
	"strings"
	"time"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/workbinding"
	"github.com/flexigpt/flexigpt-app/internal/agentruntimesetup/workflowruntime"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	NormalizeURLServiceID      = "flexigpt.url.normalize"
	NormalizeURLServiceVersion = "1"

	normalizeURLBindingVersion     = "attachment-url-normalize/v1"
	normalizeURLBindingFingerprint = "flexigpt.url.normalize"
	normalizeURLReceiptSystem      = "flexigpt.url.normalize"
	normalizeURLUnavailableCode    = "url_normalization_failed"
)

type normalizeURLInput struct {
	URL string `json:"url"`
}

type normalizeURLServiceBinding struct{}

type boundNormalizeURL struct {
	Input normalizeURLInput
}

// NormalizeURLServiceBinding returns the attachment-domain Workflow binding.
// Workflow composition registers it, but does not implement URL parsing.
func NormalizeURLServiceBinding() workbinding.Binding {
	return normalizeURLServiceBinding{}
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

func (normalizeURLServiceBinding) WorkflowPortCapabilities() workflowruntime.WorkBindingPortCapabilities {
	return workbinding.PortCapabilities{
		Context:          false,
		InputPorts:       false,
		FinalOutputPorts: true,
		OutputStream:     false,
	}
}

func (normalizeURLServiceBinding) Bind(
	_ context.Context,
	request workbinding.Request,
) (workbinding.BindResult, error) {
	input, err := decodeNormalizeURLInput(request.Input)
	if err != nil {
		return workbinding.BindResult{}, err
	}

	return workbinding.BindResult{
		SemanticBinding: sessionv3Runtime.SemanticBinding{
			Version:           normalizeURLBindingVersion,
			TargetFingerprint: normalizeURLBindingFingerprint,
			BoundAt:           time.Now().UTC(),
		},
		Private: boundNormalizeURL{Input: input},
	}, nil
}

func (normalizeURLServiceBinding) Execute(
	ctx context.Context,
	_ workbinding.Request,
	private any,
) (sessionv3Runtime.WorkExecutionResult, error) {
	bound, found := private.(boundNormalizeURL)
	if !found {
		return normalizeURLFailure(
			errors.New("url normalization binding state is unavailable"),
		), nil
	}

	result, err := NormalizeURL(ctx, bound.Input.URL)
	if err != nil {
		return normalizeURLFailure(err), nil
	}

	return sessionv3Runtime.WorkExecutionResult{
		Output: &sessionv3Runtime.WorkItemOutput{
			Value: jsonutil.MustCanonicalValue(result),
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
	_ workbinding.WaitRequest,
	_ any,
) (sessionv3Runtime.WaitResolutionResult, error) {
	return sessionv3Runtime.WaitResolutionResult{}, errors.New(
		"url normalization does not expose waits",
	)
}

func decodeNormalizeURLInput(
	value jsonutil.CanonicalValue,
) (normalizeURLInput, error) {
	input, err := jsonutil.DecodeCanonicalValue[normalizeURLInput](
		value,
	)
	if err != nil || strings.TrimSpace(input.URL) == "" {
		return normalizeURLInput{}, errors.New(
			"url normalization input is invalid",
		)
	}
	return input, nil
}

func normalizeURLFailure(
	cause error,
) sessionv3Runtime.WorkExecutionResult {
	message := "binding execution failed"
	if cause != nil && strings.TrimSpace(cause.Error()) != "" {
		message = strings.ToLower(cause.Error())
	}

	return sessionv3Runtime.WorkExecutionResult{
		Error: &sessionv3Runtime.SanitizedError{
			Code:     normalizeURLUnavailableCode,
			Category: sessionv3Runtime.ErrorCategoryExecution,
			Message:  message,
		},
		Confirmed: true,
	}
}
