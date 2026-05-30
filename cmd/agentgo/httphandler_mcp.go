package main

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	mcpHTTPTag        = "MCP"
	mcpHTTPPathPrefix = "/mcp"
)

// InitMCPWrapperHandlers registers HTTP/Huma endpoints for the MCP Wails backend wrapper.
//
// Server-scoped routes always carry both bundleID and serverID in the path.
// Request bodies intentionally do not carry bundleID/serverID for server-scoped
// operations; path identity is authoritative.
func InitMCPWrapperHandlers(api huma.API, w *MCPWrapper) {
	if api == nil || w == nil {
		return
	}

	huma.Register(api, huma.Operation{
		OperationID: "put-mcp-bundle",
		Method:      http.MethodPut,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}",
		Summary:     "Create or replace an MCP bundle",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.PutMCPBundleRequest) (*spec.PutMCPBundleResponse, error) {
		return w.PutMCPBundle(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "patch-mcp-bundle",
		Method:      http.MethodPatch,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}",
		Summary:     "Enable or disable an MCP bundle",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.PatchMCPBundleRequest) (*spec.PatchMCPBundleResponse, error) {
		return w.PatchMCPBundle(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-mcp-bundle",
		Method:      http.MethodDelete,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}",
		Summary:     "Soft-delete an MCP bundle if empty",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.DeleteMCPBundleRequest) (*spec.DeleteMCPBundleResponse, error) {
		return w.DeleteMCPBundle(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-mcp-bundles",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles",
		Summary:     "List MCP bundles",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ListMCPBundlesRequest) (*spec.ListMCPBundlesResponse, error) {
		return w.ListMCPBundles(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "put-mcp-server",
		Method:      http.MethodPut,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}",
		Summary:     "Create or replace an MCP server",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.PutMCPServerRequest) (*spec.PutMCPServerResponse, error) {
		return w.PutMCPServer(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-mcp-server",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}",
		Summary:     "Get an MCP server",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.GetMCPServerRequest) (*spec.GetMCPServerResponse, error) {
		return w.GetMCPServer(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-mcp-servers",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers",
		Summary:     "List MCP servers in a bundle",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ListMCPServersRequest) (*spec.ListMCPServersResponse, error) {
		return w.ListMCPServers(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "patch-mcp-server-enabled",
		Method:      http.MethodPatch,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/enabled",
		Summary:     "Enable or disable an MCP server",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.PatchMCPServerEnabledRequest) (*spec.PatchMCPServerEnabledResponse, error) {
		return w.PatchMCPServerEnabled(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "patch-mcp-server-policy",
		Method:      http.MethodPatch,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/policy",
		Summary:     "Update MCP server policy",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.PatchMCPServerPolicyRequest) (*spec.PatchMCPServerPolicyResponse, error) {
		return w.PatchMCPServerPolicy(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-mcp-server",
		Method:      http.MethodDelete,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}",
		Summary:     "Soft-delete an MCP server",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.DeleteMCPServerRequest) (*spec.DeleteMCPServerResponse, error) {
		return w.DeleteMCPServer(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "connect-mcp-server",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/connect",
		Summary:     "Connect an MCP server",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ConnectMCPServerRequest) (*spec.ConnectMCPServerResponse, error) {
		return w.ConnectMCPServer(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "disconnect-mcp-server",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/disconnect",
		Summary:     "Disconnect an MCP server",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.DisconnectMCPServerRequest) (*spec.DisconnectMCPServerResponse, error) {
		return w.DisconnectMCPServer(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "refresh-mcp-server",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/refresh",
		Summary:     "Refresh MCP server discovery",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.RefreshMCPServerRequest) (*spec.RefreshMCPServerResponse, error) {
		return w.RefreshMCPServer(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-mcp-server-status",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/status",
		Summary:     "Get MCP server runtime status",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.GetMCPServerStatusRequest) (*spec.GetMCPServerStatusResponse, error) {
		return w.GetMCPServerStatus(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-mcp-server-auth-status",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/auth/status",
		Summary:     "Get MCP server auth status",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.GetMCPServerAuthStatusRequest) (*spec.GetMCPServerAuthStatusResponse, error) {
		return w.GetMCPServerAuthStatus(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-mcp-server-auth-health",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/auth/health",
		Summary:     "Get MCP server auth health",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.GetMCPServerAuthHealthRequest) (*spec.GetMCPServerAuthHealthResponse, error) {
		return w.GetMCPServerAuthHealth(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-mcp-server-tools",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/tools",
		Summary:     "List discovered MCP tools",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ListMCPServerToolsRequest) (*spec.ListMCPServerToolsResponse, error) {
		return w.ListMCPServerTools(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-mcp-server-resources",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/resources",
		Summary:     "List discovered MCP resources",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ListMCPServerResourcesRequest) (*spec.ListMCPServerResourcesResponse, error) {
		return w.ListMCPServerResources(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-mcp-server-resource-templates",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/resource-templates",
		Summary:     "List discovered MCP resource templates",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ListMCPServerResourceTemplatesRequest) (*spec.ListMCPServerResourceTemplatesResponse, error) {
		return w.ListMCPServerResourceTemplates(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-mcp-server-prompts",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/prompts",
		Summary:     "List discovered MCP prompts",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ListMCPServerPromptsRequest) (*spec.ListMCPServerPromptsResponse, error) {
		return w.ListMCPServerPrompts(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "read-mcp-resource",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/resources/read",
		Summary:     "Read an MCP resource",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.MCPReadResourceRequest) (*spec.MCPReadResourceResponse, error) {
		return w.ReadMCPResource(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-mcp-prompt",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/prompts/get",
		Summary:     "Get an MCP prompt",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.MCPGetPromptRequest) (*spec.MCPGetPromptResponse, error) {
		return w.GetMCPPrompt(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "complete-mcp-argument",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/complete",
		Summary:     "Complete an MCP prompt/resource argument",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.MCPCompleteArgumentRequest) (*spec.MCPCompletionResult, error) {
		return w.CompleteMCPArgument(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "evaluate-mcp-tool-call",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/tools/evaluate",
		Summary:     "Evaluate MCP tool call policy",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.EvaluateMCPToolCallRequest) (*spec.EvaluateMCPToolCallResponse, error) {
		return w.EvaluateMCPToolCall(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "invoke-mcp-tool",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/tools/invoke",
		Summary:     "Invoke an MCP tool",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.InvokeMCPToolRequest) (*spec.InvokeMCPToolResponse, error) {
		return w.InvokeMCPTool(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "resolve-mcp-approval",
		Method:      http.MethodPost,
		Path:        mcpHTTPPathPrefix + "/approvals/resolve",
		Summary:     "Resolve a pending MCP approval",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ResolveMCPApprovalRequest) (*spec.ResolveMCPApprovalResponse, error) {
		return w.ResolveMCPApproval(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-pending-mcp-oauth-authorizations",
		Method:      http.MethodGet,
		Path:        mcpHTTPPathPrefix + "/oauth/authorizations",
		Summary:     "List pending MCP OAuth authorizations",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.ListPendingMCPOAuthAuthorizationsRequest) (*spec.ListPendingMCPOAuthAuthorizationsResponse, error) {
		return w.ListPendingMCPOAuthAuthorizations(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "cancel-pending-mcp-oauth-authorization",
		Method:      http.MethodDelete,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/oauth/authorization",
		Summary:     "Cancel a pending MCP OAuth authorization",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.CancelPendingMCPOAuthAuthorizationRequest) (*spec.CancelPendingMCPOAuthAuthorizationResponse, error) {
		return w.CancelPendingMCPOAuthAuthorization(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "put-mcp-server-secret",
		Method:      http.MethodPut,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/secrets",
		Summary:     "Create or replace an MCP server secret",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.PutMCPServerSecretRequest) (*spec.PutMCPServerSecretResponse, error) {
		return w.PutMCPServerSecret(req)
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-mcp-server-secret",
		Method:      http.MethodDelete,
		Path:        mcpHTTPPathPrefix + "/bundles/{bundleID}/servers/{serverID}/secrets",
		Summary:     "Delete an MCP server secret",
		Tags:        []string{mcpHTTPTag},
	}, func(ctx context.Context, req *spec.DeleteMCPServerSecretRequest) (*spec.DeleteMCPServerSecretResponse, error) {
		return w.DeleteMCPServerSecret(req)
	})
}
