package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/llmtools-go"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	"github.com/flexigpt/flexigpt-app/internal/tool/store"

	"github.com/flexigpt/flexigpt-app/internal/tool/runtime/spec"
)

// ToolRuntime executes built-in Go tools retrieved from ToolStore.
type ToolRuntime struct {
	store *store.ToolStore
}

func NewToolRuntime(s *store.ToolStore) *ToolRuntime {
	return &ToolRuntime{store: s}
}

// InvokeTool locates a tool version in the ToolStore and executes it according to its type.
// - Validates request, slug/version.
// - Enforces built-in bundle/tool enabled state.
// - Dispatches only built-in Go tools.
func (rt *ToolRuntime) InvokeTool(
	ctx context.Context,
	req *spec.InvokeToolRequest,
) (*spec.InvokeToolResponse, error) {
	if rt == nil || rt.store == nil {
		return nil, errors.New("tool runtime is not initialized")
	}
	if req == nil || req.Body == nil ||
		req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, errors.New(
			"invalid request: bundleID, toolSlug, version and body required",
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.ToolSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}

	args, err := jsonutil.DecodeJSONStringRaw(req.Body.Args)
	if err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}

	// Load bundle and tool definitions from the store.
	bundle, isBuiltIn, err := rt.store.GetAnyToolBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if !isBuiltIn || !bundle.IsBuiltIn {
		return nil, fmt.Errorf(
			"only built-in tools can be invoked: %s",
			req.BundleID,
		)
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("bundle is disabled: %s", req.BundleID)
	}

	gtResp, err := rt.store.GetTool(ctx, &toolSpec.GetToolRequest{
		BundleID: req.BundleID,
		ToolSlug: req.ToolSlug,
		Version:  req.Version,
	})
	if err != nil {
		return nil, err
	}
	tool := gtResp.Body
	if tool == nil {
		return nil, errors.New("tool not found: nil tool body")
	}
	if !tool.IsBuiltIn {
		return nil, errors.New("only built-in tools can be invoked")
	}
	if !tool.IsEnabled {
		return nil, fmt.Errorf(
			"tool disabled: %s/%s@%s",
			req.BundleID,
			req.ToolSlug,
			req.Version,
		)
	}

	// Defensive validation of the tool record.
	if err := tool.Validate(); err != nil {
		return nil, fmt.Errorf("tool validation failed: %w", err)
	}

	var (
		outputs []llmtoolsSpec.ToolOutputUnion
		md      map[string]any
		isError bool
		errMsg  string
	)

	switch tool.Type {
	case toolSpec.ToolTypeGo:
		var gopts []llmtools.CallOption
		if req.Body.GoOptions != nil && req.Body.GoOptions.TimeoutMS > 0 {
			gopts = append(
				gopts,
				llmtools.WithCallTimeout(time.Duration(req.Body.GoOptions.TimeoutMS)*time.Millisecond),
			)
		}

		outputs, err = llmtoolsutil.CallUsingDefaultGoRegistry(
			ctx,
			strings.TrimSpace(tool.GoImpl.Func),
			args,
			gopts...,
		)
		md = map[string]any{
			"type":     "go",
			"funcName": tool.GoImpl.Func,
		}

	case toolSpec.ToolTypeSDK:
		return nil, errors.New(
			"API-backed tools are invoked by provider inference, not InvokeTool",
		)

	default:
		return nil, fmt.Errorf("unsupported tool type: %s", tool.Type)
	}

	if err != nil {
		// Tool execution errors are surfaced as tool-level errors in the response.
		isError = true
		errMsg = err.Error()
	}

	return &spec.InvokeToolResponse{
		Body: &spec.InvokeToolResponseBody{
			Outputs:      outputs,
			Meta:         md,
			IsBuiltIn:    isBuiltIn,
			IsError:      isError,
			ErrorMessage: errMsg,
		},
	}, nil
}
