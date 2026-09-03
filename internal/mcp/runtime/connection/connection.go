package connection

import (
	"context"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

// PreparedConnection is the runtime-owned result of authorization
// preparation. OAuthHandler is intentionally opaque to the core runtime.
// A concrete transport adapter, such as sdkclient, may interpret it.
type PreparedConnection struct {
	Env     map[string]string
	Headers map[string]string

	SensitiveValues []string
	OAuthHandler    any
}

// ConnectionAuthorizer is a narrow runtime port. The runtime session manager
// knows nothing about settings, secret persistence, artifact storage, OAuth
// loopback listeners, or concrete SDK types.
type ConnectionAuthorizer interface {
	PrepareConnection(
		ctx context.Context,
		config mcpServer.RuntimeConfig,
	) (PreparedConnection, error)

	ConnectionSucceeded(
		ctx context.Context,
		config mcpServer.RuntimeConfig,
	)

	ConnectionFailed(
		ctx context.Context,
		config mcpServer.RuntimeConfig,
		err error,
	)
}

type (
	ClientNotification             = mcpServer.ClientNotification
	ClientNotificationKind         = mcpServer.ClientNotificationKind
	ClientNotificationSink         = mcpServer.ClientNotificationSink
	MCPCompleteArgumentRequestBody = mcpServer.MCPCompleteArgumentRequestBody
	MCPCompletionResult            = mcpServer.MCPCompletionResult
	MCPDiscoveryPageToken          = mcpServer.MCPDiscoveryPageToken
	MCPDiscoverySnapshot           = mcpServer.MCPDiscoverySnapshot
	MCPGetPromptResponseBody       = mcpServer.MCPGetPromptResponseBody
	MCPImplementationInfo          = mcpServer.MCPImplementationInfo
	MCPPromptRef                   = mcpServer.MCPPromptRef
	MCPReadResourceResponseBody    = mcpServer.MCPReadResourceResponseBody
	MCPResourceRef                 = mcpServer.MCPResourceRef
	MCPResourceTemplateRef         = mcpServer.MCPResourceTemplateRef
	MCPServerCapabilitiesSummary   = mcpServer.MCPServerCapabilitiesSummary
	MCPServerRuntimeSnapshot       = mcpServer.MCPServerRuntimeSnapshot
	MCPServerStatus                = mcpServer.MCPServerStatus
	MCPTaskSupport                 = mcpServer.MCPTaskSupport
	MCPToolAppRenderInfo           = mcpServer.MCPToolAppRenderInfo
	MCPToolCapability              = mcpServer.MCPToolCapability
	InvokeMCPToolRequestBody       = mcpServer.InvokeMCPToolRequestBody
	InvokeMCPToolResponseBody      = mcpServer.InvokeMCPToolResponseBody
)

const (
	DefaultMCPPageSize          = mcpServer.DefaultMCPPageSize
	MaxMCPServerPageSize        = mcpServer.MaxMCPServerPageSize
	NotificationRefreshDebounce = mcpServer.NotificationRefreshDebounce

	MCPServerStatusDisabled     = mcpServer.MCPServerStatusDisabled
	MCPServerStatusDisconnected = mcpServer.MCPServerStatusDisconnected
	MCPServerStatusConnecting   = mcpServer.MCPServerStatusConnecting
	MCPServerStatusReady        = mcpServer.MCPServerStatusReady
	MCPServerStatusError        = mcpServer.MCPServerStatusError

	MCPTaskSupportForbidden = mcpServer.MCPTaskSupportForbidden
	MCPTaskSupportOptional  = mcpServer.MCPTaskSupportOptional
	MCPTaskSupportRequired  = mcpServer.MCPTaskSupportRequired
)

var (
	ErrMCPInvalidRuntimeRequest = mcpServer.ErrMCPInvalidRuntimeRequest
	ErrMCPRuntimeNotReady       = mcpServer.ErrMCPRuntimeNotReady
	ErrMCPPolicyDenied          = mcpServer.ErrMCPPolicyDenied
	ErrMCPApprovalNeeded        = mcpServer.ErrMCPApprovalNeeded
	ErrMCPStaleReference        = mcpServer.ErrMCPStaleReference
)
