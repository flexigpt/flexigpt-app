package toolruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/llmtools-go"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	"github.com/flexigpt/flexigpt-app/internal/tool/store"
	"github.com/flexigpt/flexigpt-app/internal/tool/storehelper"

	"github.com/flexigpt/flexigpt-app/internal/toolruntime/httprunner"
	"github.com/flexigpt/flexigpt-app/internal/toolruntime/spec"
)

// ToolRuntime executes tools (HTTP/Go) using tool definitions retrieved from ToolStore.
type ToolRuntime struct {
	store *store.ToolStore
}

func NewToolRuntime(s *store.ToolStore) *ToolRuntime {
	return &ToolRuntime{store: s}
}

// InvokeTool locates a tool version in the ToolStore and executes it according to its type.
// - Validates request, slug/version.
// - Enforces bundle/tool enabled state.
// - Dispatches to HTTP or Go runner with functional options constructed from the request body.
func (rt *ToolRuntime) InvokeTool(
	ctx context.Context,
	req *spec.InvokeToolRequest,
) (*spec.InvokeToolResponse, error) {
	if req == nil || req.Body == nil ||
		req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, toolSlug, version and body required",
			toolSpec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.ToolSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}

	args := json.RawMessage(req.Body.Args)

	// Load bundle and tool definitions from the store.
	bundle, isBuiltIn, err := rt.store.GetAnyToolBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: bundle %s", toolSpec.ErrBundleDisabled, req.BundleID)
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
		return nil, fmt.Errorf("%w: nil tool body", toolSpec.ErrToolNotFound)
	}
	if !tool.IsEnabled {
		return nil, fmt.Errorf(
			"%w: %s/%s@%s",
			toolSpec.ErrToolDisabled,
			req.BundleID,
			req.ToolSlug,
			req.Version,
		)
	}

	// Defensive validation of the tool record.
	if err := storehelper.ValidateTool(tool); err != nil {
		return nil, fmt.Errorf("tool validation failed: %w", err)
	}

	var (
		outputs []llmtoolsgoSpec.ToolOutputUnion
		md      map[string]any
		isError bool
		errMsg  string
	)

	switch tool.Type {
	case toolSpec.ToolTypeHTTP:
		var hopts []httprunner.HTTPOption
		if req.Body.HTTPOptions != nil {
			if req.Body.HTTPOptions.TimeoutMS > 0 {
				hopts = append(hopts, httprunner.WithHTTPTimeoutMS(req.Body.HTTPOptions.TimeoutMS))
			}
			if len(req.Body.HTTPOptions.ExtraHeaders) > 0 {
				hopts = append(hopts, httprunner.WithHTTPExtraHeaders(req.Body.HTTPOptions.ExtraHeaders))
			}
			if len(req.Body.HTTPOptions.Secrets) > 0 {
				hopts = append(hopts, httprunner.WithHTTPSecrets(req.Body.HTTPOptions.Secrets))
			}
		}

		r, configErr := httprunner.NewHTTPToolRunner(*tool.HTTPImpl, hopts...)
		if configErr != nil {
			return nil, configErr
		}
		outputs, md, err = r.Run(ctx, args)

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
		// SDK-backed tools are not invoked through ToolRuntime; they are surfaced to the model as provider server
		// tools.
		return nil, fmt.Errorf("unsupported tool type for InvokeTool: %s", tool.Type)

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
