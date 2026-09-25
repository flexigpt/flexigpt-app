package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type GoToolCaller interface {
	CallGoTool(
		ctx context.Context,
		function string,
		args json.RawMessage,
		timeout time.Duration,
	) ([]llmtoolsSpec.ToolOutputUnion, error)
}

type Service struct {
	caller GoToolCaller
}

func New(
	caller GoToolCaller,
) (*Service, error) {
	if caller == nil {
		return nil, fmt.Errorf(
			"%w: Tool Runtime Go Tool caller is nil",
			basespec.ErrInvalid,
		)
	}
	return &Service{caller: caller}, nil
}

func (s *Service) Invoke(
	ctx context.Context,
	request InvokeRequest,
) (*InvokeResponse, error) {
	if s == nil || s.caller == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Tool Runtime context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	function := strings.TrimSpace(request.Function)
	if function == "" {
		return nil, fmt.Errorf(
			"%w: Go Tool function is required",
			basespec.ErrInvalid,
		)
	}

	args := request.Args
	if len(args) == 0 {
		args = json.RawMessage(`{}`)
	}
	if !json.Valid(args) {
		return nil, fmt.Errorf(
			"%w: Tool invocation arguments are invalid JSON",
			basespec.ErrInvalid,
		)
	}

	if request.TimeoutMS < 0 ||
		int64(request.TimeoutMS) > int64((1<<63-1)/time.Millisecond) {
		return nil, fmt.Errorf(
			"%w: Tool timeout is outside the supported range",
			basespec.ErrInvalid,
		)
	}

	var timeout time.Duration
	if request.TimeoutMS > 0 {
		timeout = time.Duration(request.TimeoutMS) * time.Millisecond
	}

	outputs, callErr := s.caller.CallGoTool(
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
