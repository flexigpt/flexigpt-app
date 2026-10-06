package spec

import (
	"context"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

// MCPRuntime is the MCP capability consumed by inference preparation and
// hydration. It deliberately excludes Artifact persistence, credentials,
// connection lifecycle management, runtime invalidation, and tool execution.
//
// The application binds an implementation before constructing inference.
// A nil capability means MCP is unavailable for that inference deployment.
type MCPRuntime interface {
	Status(
		ctx context.Context,
		server mcpServer.ServerID,
	) (*mcpServer.MCPServerRuntimeSnapshot, error)

	ListTools(
		ctx context.Context,
		server mcpServer.ServerID,
	) ([]mcpServer.MCPToolCapability, error)

	ListResources(
		ctx context.Context,
		server mcpServer.ServerID,
	) ([]mcpServer.MCPResourceRef, error)

	ListResourceTemplates(
		ctx context.Context,
		server mcpServer.ServerID,
	) ([]mcpServer.MCPResourceTemplateRef, error)

	ListPrompts(
		ctx context.Context,
		server mcpServer.ServerID,
	) ([]mcpServer.MCPPromptRef, error)

	ReadResource(
		ctx context.Context,
		server mcpServer.ServerID,
		uri string,
	) (*mcpServer.MCPReadResourceResponseBody, error)

	GetPrompt(
		ctx context.Context,
		server mcpServer.ServerID,
		name string,
		arguments map[string]string,
	) (*mcpServer.MCPGetPromptResponseBody, error)
}
