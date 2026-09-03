package invocation

import (
	"context"

	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
)

type Runtime interface {
	CallToolDryRun(
		ctx context.Context,
		server mcpSpec.ServerID,
		request mcpRuntime.InvokeMCPToolRequestBody,
	) (mcpSpec.RuntimeConfig, mcpRuntime.MCPToolCapability, error)

	CallTool(
		ctx context.Context,
		server mcpSpec.ServerID,
		request mcpRuntime.InvokeMCPToolRequestBody,
	) (*mcpRuntime.InvokeMCPToolResponseBody, error)

	SetSessionLifecycleCleaner(
		cleaner mcpRuntime.SessionLifecycleCleaner,
	)
}
