package llmtoolsutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

var errInvalid = errors.New("invalid request")

type GoToolCaller interface {
	CallGoTool(
		ctx context.Context,
		function string,
		args json.RawMessage,
		timeout time.Duration,
	) ([]llmtoolsSpec.ToolOutputUnion, error)
}

type InvokeRequest struct {
	Function  string                 `json:"function"`
	Args      jsonutil.JSONRawString `json:"args,omitempty"`
	TimeoutMS int                    `json:"timeoutMS,omitempty"`
}

type InvokeResponse struct {
	Outputs      []llmtoolsSpec.ToolOutputUnion `json:"outputs,omitempty"`
	Meta         map[string]any                 `json:"meta,omitempty"`
	IsError      bool                           `json:"isError"`
	ErrorMessage string                         `json:"errorMessage,omitempty"`
}

func Invoke(
	ctx context.Context,
	caller GoToolCaller,
	request InvokeRequest,
) (*InvokeResponse, error) {
	if caller == nil {
		return nil, fmt.Errorf("%w: go caller is nil", errInvalid)
	}
	function := strings.TrimSpace(request.Function)
	if function == "" {
		return nil, fmt.Errorf("%w: go tool function is required", errInvalid)
	}

	var args json.RawMessage
	var err error
	argsRawStr := request.Args
	if strings.TrimSpace(string(argsRawStr)) == "" {
		args = json.RawMessage(`{}`)
	} else {
		args, err = jsonutil.DecodeJSONStringRawInto[json.RawMessage](argsRawStr)
	}
	if err != nil {
		return nil, err
	}

	if !json.Valid(args) {
		return nil, fmt.Errorf(
			"%w: Tool invocation arguments are invalid JSON",
			errInvalid,
		)
	}

	if request.TimeoutMS < 0 ||
		int64(request.TimeoutMS) > int64((1<<63-1)/time.Millisecond) {
		return nil, fmt.Errorf(
			"%w: Tool timeout is outside the supported range",
			errInvalid,
		)
	}

	var timeout time.Duration
	if request.TimeoutMS > 0 {
		timeout = time.Duration(request.TimeoutMS) * time.Millisecond
	}

	outputs, callErr := caller.CallGoTool(
		ctx,
		function,
		args,
		timeout,
	)
	response := &InvokeResponse{
		Outputs: outputs,
		Meta: map[string]any{
			"implementation": "go",
			"function":       function,
		},
	}
	if callErr != nil {
		response.IsError = true
		response.ErrorMessage = callErr.Error()
	}
	return response, nil
}
