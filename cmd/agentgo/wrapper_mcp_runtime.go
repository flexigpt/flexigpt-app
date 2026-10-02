package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/invocation"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

// MCPRuntimeWrapper exposes only pure runtime operations. Store and aggregate
// operations are intentionally exposed by their own wrappers.
type MCPRuntimeWrapper struct {
	runtime    *mcpConnection.MCPRuntimeManager
	toolBridge *invocation.ToolBridge
	auth       *mcpAuth.AuthManager

	oauthBroker *mcpAuth.OAuthLoopbackBroker
}

func withMCPRuntime[T any](
	w *MCPRuntimeWrapper,
	fn func() (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if err := w.ready(); err != nil {
			return zero, err
		}
		return fn()
	})
}

func withMCPRuntimeError(
	w *MCPRuntimeWrapper,
	fn func() error,
) error {
	return withRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		return fn()
	})
}

// ConnectMCPServer starts the managed asynchronous connection flow. The
// current connection state is returned by aggregate.GetMCPServer.
func (w *MCPRuntimeWrapper) ConnectMCPServer(
	server mcpServer.ServerID,
) (*mcpServer.MCPServerRuntimeSnapshot, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPServerRuntimeSnapshot, error) {
		return w.runtime.StartConnect(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) DisconnectMCPServer(
	server mcpServer.ServerID,
) error {
	return withMCPRuntimeError(w, func() error {
		return w.runtime.Disconnect(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) RefreshMCPServer(
	server mcpServer.ServerID,
) (*mcpServer.MCPServerRuntimeSnapshot, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPServerRuntimeSnapshot, error) {
		return w.runtime.Refresh(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerTools(
	server mcpServer.ServerID,
	pageSize int,
	pageToken string,
) (mcpConnection.MCPToolCapabilityPage, error) {
	return withMCPRuntime(w, func() (mcpConnection.MCPToolCapabilityPage, error) {
		return w.runtime.ListToolsPage(
			context.Background(),
			server,
			pageSize,
			pageToken,
		)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerResources(
	server mcpServer.ServerID,
	pageSize int,
	pageToken string,
) (mcpConnection.MCPResourcePage, error) {
	return withMCPRuntime(w, func() (mcpConnection.MCPResourcePage, error) {
		return w.runtime.ListResourcesPage(
			context.Background(),
			server,
			pageSize,
			pageToken,
		)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerResourceTemplates(
	server mcpServer.ServerID,
	pageSize int,
	pageToken string,
) (mcpConnection.MCPResourceTemplatePage, error) {
	return withMCPRuntime(w, func() (mcpConnection.MCPResourceTemplatePage, error) {
		return w.runtime.ListResourceTemplatesPage(
			context.Background(),
			server,
			pageSize,
			pageToken,
		)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerPrompts(
	server mcpServer.ServerID,
	pageSize int,
	pageToken string,
) (mcpConnection.MCPPromptPage, error) {
	return withMCPRuntime(w, func() (mcpConnection.MCPPromptPage, error) {
		return w.runtime.ListPromptsPage(
			context.Background(),
			server,
			pageSize,
			pageToken,
		)
	})
}

func (w *MCPRuntimeWrapper) ReadMCPResource(
	server mcpServer.ServerID,
	uri string,
) (*mcpServer.MCPReadResourceResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPReadResourceResponseBody, error) {
		return w.runtime.ReadResource(context.Background(), server, uri)
	})
}

func (w *MCPRuntimeWrapper) GetMCPPrompt(
	server mcpServer.ServerID,
	name string,
	arguments map[string]string,
) (*mcpServer.MCPGetPromptResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPGetPromptResponseBody, error) {
		return w.runtime.GetPrompt(context.Background(), server, name, arguments)
	})
}

func (w *MCPRuntimeWrapper) CompleteMCPArgument(
	server mcpServer.ServerID,
	request mcpServer.MCPCompleteArgumentRequestBody,
) (*mcpServer.MCPCompletionResult, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPCompletionResult, error) {
		return w.runtime.Complete(context.Background(), server, request)
	})
}

func (w *MCPRuntimeWrapper) CheckMCPToolCall(
	server mcpServer.ServerID,
	request *mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.MCPApprovalEvaluation, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPApprovalEvaluation, error) {
		if request == nil {
			return nil, fmt.Errorf(
				"%w: MCP tool request is required",
				mcpServer.ErrMCPInvalidRuntimeRequest,
			)
		}
		return w.toolBridge.Evaluate(context.Background(), server, *request)
	})
}

func (w *MCPRuntimeWrapper) ResolveMCPToolApproval(
	approvalID string,
	resolution mcpServer.MCPApprovalResolution,
) (mcpServer.MCPApprovalResolutionResult, error) {
	return withMCPRuntime(w, func() (mcpServer.MCPApprovalResolutionResult, error) {
		return w.toolBridge.ResolveApproval(
			context.Background(),
			approvalID,
			resolution,
		)
	})
}

func (w *MCPRuntimeWrapper) InvokeMCPTool(
	server mcpServer.ServerID,
	request *mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.InvokeMCPToolResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpServer.InvokeMCPToolResponseBody, error) {
		if request == nil {
			return nil, fmt.Errorf(
				"%w: MCP tool request is required",
				mcpServer.ErrMCPInvalidRuntimeRequest,
			)
		}
		return w.toolBridge.Invoke(context.Background(), server, *request)
	})
}

func (w *MCPRuntimeWrapper) CancelMCPServerAuthorization(
	server mcpServer.ServerID,
) (bool, error) {
	return withMCPRuntime(w, func() (bool, error) {
		if err := server.Validate(); err != nil {
			return false, err
		}
		cancelled := w.oauthBroker.Cancel(server)
		if err := w.runtime.Disconnect(context.Background(), server); err != nil {
			slog.Warn(
				"cancel MCP OAuth runtime connection",
				"server",
				server,
				"error",
				err,
			)
		}
		w.auth.ClearAuthStatus(server)
		return cancelled, nil
	})
}

func (w *MCPRuntimeWrapper) ready() error {
	if w == nil ||
		w.runtime == nil ||
		w.toolBridge == nil ||
		w.auth == nil ||
		w.oauthBroker == nil {
		return spec.ErrClosed
	}
	return nil
}

func (w *MCPRuntimeWrapper) close() {
	if w == nil {
		return
	}

	runtimeManager := w.runtime
	broker := w.oauthBroker

	w.runtime = nil
	w.toolBridge = nil
	w.auth = nil
	w.oauthBroker = nil

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if runtimeManager != nil {
		if err := runtimeManager.Close(ctx); err != nil {
			slog.Error("close artifact-backed MCP runtime", "error", err)
		}
	}
	if broker != nil {
		if err := broker.Close(); err != nil {
			slog.Error("close MCP OAuth broker", "error", err)
		}
	}
}
