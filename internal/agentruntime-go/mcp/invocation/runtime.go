package invocation

import (
	"context"

	mcpConnection "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/connection"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
)

type Runtime interface {
	CallToolDryRun(
		ctx context.Context,
		server mcpServer.ServerID,
		request mcpServer.InvokeMCPToolRequestBody,
	) (mcpServer.RuntimeConfig, mcpServer.MCPToolCapability, error)

	CallTool(
		ctx context.Context,
		server mcpServer.ServerID,
		request mcpServer.InvokeMCPToolRequestBody,
	) (*mcpServer.InvokeMCPToolResponseBody, error)

	SetSessionLifecycleCleaner(
		cleaner mcpConnection.SessionLifecycleCleaner,
	)
}
