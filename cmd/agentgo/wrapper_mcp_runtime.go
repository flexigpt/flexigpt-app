package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime/invocation"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
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
	runtime    *mcpRuntime.MCPRuntimeManager
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
	server mcpSpec.ServerID,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
		return w.runtime.Connect(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) StartMCPServerConnect(
	server mcpSpec.ServerID,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
		return w.runtime.StartConnect(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) DisconnectMCPServer(
	server mcpSpec.ServerID,
) error {
	return withMCPRuntimeError(w, func() error {
		return w.runtime.Disconnect(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) RefreshMCPServer(
	server mcpSpec.ServerID,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
		return w.runtime.Refresh(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) GetMCPServerStatus(
	server mcpSpec.ServerID,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
		return w.runtime.Status(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerTools(
	server mcpSpec.ServerID,
) ([]mcpRuntime.MCPToolCapability, error) {
	return withMCPRuntime(w, func() ([]mcpRuntime.MCPToolCapability, error) {
		return w.runtime.ListTools(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerToolsPage(
	server mcpSpec.ServerID,
	pageSize int,
	pageToken string,
) ([]mcpRuntime.MCPToolCapability, *string, error) {
	if err := w.ready(); err != nil {
		return nil, nil, err
	}
	return w.runtime.ListToolsPage(context.Background(), server, pageSize, pageToken)
}

func (w *MCPRuntimeWrapper) ListMCPServerResources(
	server mcpSpec.ServerID,
) ([]mcpRuntime.MCPResourceRef, error) {
	return withMCPRuntime(w, func() ([]mcpRuntime.MCPResourceRef, error) {
		return w.runtime.ListResources(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerResourcesPage(
	server mcpSpec.ServerID,
	pageSize int,
	pageToken string,
) ([]mcpRuntime.MCPResourceRef, *string, error) {
	if err := w.ready(); err != nil {
		return nil, nil, err
	}
	return w.runtime.ListResourcesPage(context.Background(), server, pageSize, pageToken)
}

func (w *MCPRuntimeWrapper) ListMCPServerResourceTemplates(
	server mcpSpec.ServerID,
) ([]mcpRuntime.MCPResourceTemplateRef, error) {
	return withMCPRuntime(w, func() ([]mcpRuntime.MCPResourceTemplateRef, error) {
		return w.runtime.ListResourceTemplates(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerResourceTemplatesPage(
	server mcpSpec.ServerID,
	pageSize int,
	pageToken string,
) ([]mcpRuntime.MCPResourceTemplateRef, *string, error) {
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
	server mcpSpec.ServerID,
) ([]mcpRuntime.MCPPromptRef, error) {
	return withMCPRuntime(w, func() ([]mcpRuntime.MCPPromptRef, error) {
		return w.runtime.ListPrompts(context.Background(), server)
	})
}

func (w *MCPRuntimeWrapper) ListMCPServerPromptsPage(
	server mcpSpec.ServerID,
	pageSize int,
	pageToken string,
) ([]mcpRuntime.MCPPromptRef, *string, error) {
	if err := w.ready(); err != nil {
		return nil, nil, err
	}
	return w.runtime.ListPromptsPage(context.Background(), server, pageSize, pageToken)
}

func (w *MCPRuntimeWrapper) ReadMCPResource(
	server mcpSpec.ServerID,
	uri string,
) (*mcpRuntime.MCPReadResourceResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPReadResourceResponseBody, error) {
		return w.runtime.ReadResource(context.Background(), server, uri)
	})
}

func (w *MCPRuntimeWrapper) GetMCPPrompt(
	server mcpSpec.ServerID,
	name string,
	arguments map[string]string,
) (*mcpRuntime.MCPGetPromptResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPGetPromptResponseBody, error) {
		return w.runtime.GetPrompt(context.Background(), server, name, arguments)
	})
}

func (w *MCPRuntimeWrapper) CompleteMCPArgument(
	server mcpSpec.ServerID,
	request mcpRuntime.MCPCompleteArgumentRequestBody,
) (*mcpRuntime.MCPCompletionResult, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPCompletionResult, error) {
		return w.runtime.Complete(context.Background(), server, request)
	})
}

func (w *MCPRuntimeWrapper) EvaluateMCPToolCall(
	server mcpSpec.ServerID,
	request *mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.MCPApprovalEvaluation, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPApprovalEvaluation, error) {
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", mcpRuntime.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.Evaluate(context.Background(), server, *request)
	})
}

func (w *MCPRuntimeWrapper) EvaluateMappedMCPToolCall(
	mapping mcpRuntime.MCPProviderToolMapping,
	request *mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.MCPApprovalEvaluation, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.MCPApprovalEvaluation, error) {
		if request == nil {
			return nil, fmt.Errorf("%w: mapped MCP tool request is required", mcpRuntime.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.EvaluateMapped(context.Background(), mapping, *request)
	})
}

func (w *MCPRuntimeWrapper) ResolveMCPApproval(
	approvalID string,
	resolution mcpRuntime.MCPApprovalResolution,
) (mcpRuntime.MCPApprovalResolutionResult, error) {
	return withMCPRuntime(w, func() (mcpRuntime.MCPApprovalResolutionResult, error) {
		return w.toolBridge.ResolveApproval(context.Background(), approvalID, resolution)
	})
}

func (w *MCPRuntimeWrapper) InvokeMCPTool(
	server mcpSpec.ServerID,
	request *mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.InvokeMCPToolResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.InvokeMCPToolResponseBody, error) {
		if request == nil {
			return nil, fmt.Errorf("%w: MCP tool request is required", mcpRuntime.ErrMCPInvalidRuntimeRequest)
		}
		return w.toolBridge.Invoke(context.Background(), server, *request)
	})
}

func (w *MCPRuntimeWrapper) InvokeMappedMCPTool(
	mapping mcpRuntime.MCPProviderToolMapping,
	request *mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.InvokeMCPToolResponseBody, error) {
	return withMCPRuntime(w, func() (*mcpRuntime.InvokeMCPToolResponseBody, error) {
		if request == nil {
			return nil, fmt.Errorf("%w: mapped MCP tool request is required", mcpRuntime.ErrMCPInvalidRuntimeRequest)
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
	server mcpSpec.ServerID,
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
