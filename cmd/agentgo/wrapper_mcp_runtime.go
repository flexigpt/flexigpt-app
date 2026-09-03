package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	mcpConversation "github.com/flexigpt/flexigpt-app/internal/mcp/conversation"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/invocation"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type MCPGlobalSettingsView struct {
	Settings                mcpAuth.MCPAuthSettings `json:"settings"`
	Revision                uint64                  `json:"revision"`
	OAuthRedirectURL        string                  `json:"oauthRedirectURL,omitempty"`
	OAuthLoopbackListenAddr string                  `json:"oauthLoopbackListenAddr,omitempty"`
	OAuthRestartRequired    bool                    `json:"oauthRestartRequired"`
	OAuthLoopbackReady      bool                    `json:"oauthLoopbackReady"`
	OAuthLoopbackError      string                  `json:"oauthLoopbackError,omitempty"`
}

// MCPRuntimeWrapper exposes pure Runtime operations. Every server argument is
// an opaque runtime ServerID. ArtifactRef translation is deliberately absent.
type MCPRuntimeWrapper struct {
	runtime    *mcpConnection.MCPRuntimeManager
	toolBridge *invocation.ToolBridge
	auth       *mcpAuth.AuthManager

	settings                       *mcpSettingsAdapter
	oauthBroker                    *mcpAuth.OAuthLoopbackBroker
	oauthLoopbackListenAddrAtStart string
}

func withMCPRuntime[T any](
	w *MCPRuntimeWrapper,
	fn func() (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
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
	return middleware.WithRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		return fn()
	})
}

func (w *MCPRuntimeWrapper) ConnectMCPServer(
	server mcpServer.ServerID,
) (*mcpServer.MCPServerRuntimeSnapshot, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPServerRuntimeSnapshot, error) {
		return w.runtime.Connect(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) StartMCPServerConnect(
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

func (w *MCPRuntimeWrapper) GetMCPServerStatus(
	server mcpServer.ServerID,
) (*mcpServer.MCPServerRuntimeSnapshot, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPServerRuntimeSnapshot, error) {
		return w.runtime.Status(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerTools(
	server mcpServer.ServerID,
) ([]mcpServer.MCPToolCapability, error) {
	return withMCPRuntime(w, func() ([]mcpServer.MCPToolCapability, error) {
		return w.runtime.ListTools(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerToolsPage(
	server mcpServer.ServerID,
	pageSize int,
	pageToken string,
) ([]mcpServer.MCPToolCapability, *string, error) {
	if err := w.ready(); err != nil {
		return nil, nil, err
	}
	return w.runtime.ListToolsPage(context.Background(), server, pageSize, pageToken)
}

func (w *MCPRuntimeWrapper) ListMCPServerResources(
	server mcpServer.ServerID,
) ([]mcpServer.MCPResourceRef, error) {
	return withMCPRuntime(w, func() ([]mcpServer.MCPResourceRef, error) {
		return w.runtime.ListResources(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerResourcesPage(
	server mcpServer.ServerID,
	pageSize int,
	pageToken string,
) ([]mcpServer.MCPResourceRef, *string, error) {
	if err := w.ready(); err != nil {
		return nil, nil, err
	}
	return w.runtime.ListResourcesPage(context.Background(), server, pageSize, pageToken)
}

func (w *MCPRuntimeWrapper) ListMCPServerResourceTemplates(
	server mcpServer.ServerID,
) ([]mcpServer.MCPResourceTemplateRef, error) {
	return withMCPRuntime(w, func() ([]mcpServer.MCPResourceTemplateRef, error) {
		return w.runtime.ListResourceTemplates(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerResourceTemplatesPage(
	server mcpServer.ServerID,
	pageSize int,
	pageToken string,
) ([]mcpServer.MCPResourceTemplateRef, *string, error) {
	if err := w.ready(); err != nil {
		return nil, nil, err
	}
	return w.runtime.ListResourceTemplatesPage(
		context.Background(),
		server,
		pageSize,
		pageToken,
	)
}

func (w *MCPRuntimeWrapper) ListMCPServerPrompts(
	server mcpServer.ServerID,
) ([]mcpServer.MCPPromptRef, error) {
	return withMCPRuntime(w, func() ([]mcpServer.MCPPromptRef, error) {
		return w.runtime.ListPrompts(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerPromptsPage(
	server mcpServer.ServerID,
	pageSize int,
	pageToken string,
) ([]mcpServer.MCPPromptRef, *string, error) {
	if err := w.ready(); err != nil {
		return nil, nil, err
	}
	return w.runtime.ListPromptsPage(context.Background(), server, pageSize, pageToken)
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

func (w *MCPRuntimeWrapper) EvaluateMCPToolCall(
	server mcpServer.ServerID,
	request *mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.MCPApprovalEvaluation, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPApprovalEvaluation, error) {
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", mcpServer.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.Evaluate(context.Background(), server, *request)
	})
}

func (w *MCPRuntimeWrapper) EvaluateMappedMCPToolCall(
	mapping mcpConversation.MCPProviderToolMapping,
	request *mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.MCPApprovalEvaluation, error) {
	return withMCPRuntime(w, func() (*mcpServer.MCPApprovalEvaluation, error) {
		if request == nil {
			return nil, fmt.Errorf("%w: mapped MCP tool request is required", mcpServer.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.EvaluateMapped(context.Background(), mapping, *request)
	})
}

func (w *MCPRuntimeWrapper) ResolveMCPApproval(
	approvalID string,
	resolution mcpServer.MCPApprovalResolution,
) (mcpServer.MCPApprovalResolutionResult, error) {
	return withMCPRuntime(w, func() (mcpServer.MCPApprovalResolutionResult, error) {
		return w.toolBridge.ResolveApproval(context.Background(), approvalID, resolution)
	})
}

func (w *MCPRuntimeWrapper) InvokeMCPTool(
	server mcpServer.ServerID,
	request *mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.InvokeMCPToolResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpServer.InvokeMCPToolResponseBody, error) {
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", mcpServer.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.Invoke(context.Background(), server, *request)
	})
}

func (w *MCPRuntimeWrapper) InvokeMappedMCPTool(
	mapping mcpConversation.MCPProviderToolMapping,
	request *mcpServer.InvokeMCPToolRequestBody,
) (*mcpServer.InvokeMCPToolResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpServer.InvokeMCPToolResponseBody, error) {
		if request == nil {
			return nil, fmt.Errorf("%w: mapped MCP tool request is required", mcpServer.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.InvokeMapped(context.Background(), mapping, *request)
	})
}

func (w *MCPRuntimeWrapper) ListPendingMCPOAuthAuthorizations() (
	[]mcpAuth.MCPOAuthAuthorization,
	error,
) {
	return withMCPRuntime(w, func() ([]mcpAuth.MCPOAuthAuthorization, error) {
		values := w.oauthBroker.Pending()
		if values == nil {
			values = []mcpAuth.MCPOAuthAuthorization{}
		}
		return values, nil
	})
}

func (w *MCPRuntimeWrapper) CancelPendingMCPOAuthAuthorization(
	server mcpServer.ServerID,
) (bool, error) {
	return withMCPRuntime(w, func() (bool, error) {
		if err := server.Validate(); err != nil {
			return false, err
		}
		cancelled := w.oauthBroker.Cancel(server)
		if err := w.runtime.Disconnect(context.Background(), server); err != nil {
			slog.Warn("cancel MCP OAuth runtime connection", "server", server, "error", err)
		}
		w.auth.ClearAuthStatus(server)
		return cancelled, nil
	})
}

func (w *MCPRuntimeWrapper) UpdateMCPGlobalSettings(
	expectedRevision uint64,
	settings mcpAuth.MCPAuthSettings,
) (uint64, error) {
	return withMCPRuntime(w, func() (uint64, error) {
		return w.settings.PutMCPGlobalSettings(context.Background(), expectedRevision, settings)
	})
}

func (w *MCPRuntimeWrapper) GetMCPGlobalSettings() (
	MCPGlobalSettingsView,
	error,
) {
	return withMCPRuntime(w, func() (MCPGlobalSettingsView, error) {
		settings, revision, err := w.settings.GetMCPGlobalSettings(context.Background())
		if err != nil {
			return MCPGlobalSettingsView{}, err
		}
		view := MCPGlobalSettingsView{
			Settings: settings,
			Revision: revision,
		}
		view.OAuthRedirectURL = w.oauthBroker.RedirectURL()
		view.OAuthLoopbackListenAddr = w.oauthBroker.ListenAddr()
		view.OAuthLoopbackReady, view.OAuthLoopbackError = w.oauthBroker.Readiness()
		view.OAuthRestartRequired =
			strings.TrimSpace(settings.OAuthLoopbackListenAddr) !=
				w.oauthLoopbackListenAddrAtStart
		return view, nil
	})
}

func (w *MCPRuntimeWrapper) ready() error {
	if w == nil ||
		w.runtime == nil ||
		w.toolBridge == nil ||
		w.auth == nil ||
		w.settings == nil ||
		w.oauthBroker == nil {
		return basespec.ErrClosed
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
	w.settings = nil
	w.oauthBroker = nil
	w.oauthLoopbackListenAddrAtStart = ""

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
