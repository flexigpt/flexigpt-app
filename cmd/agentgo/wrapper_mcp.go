package main

import (
	"context"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/sdkclient"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store"

	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type MCPWrapper struct {
	store      *store.Store
	auth       *runtime.AuthManager
	runtime    *runtime.RuntimeManager
	approvals  *runtime.ApprovalManager
	toolBridge *runtime.ToolBridge
}

func InitMCPWrapper(w *MCPWrapper, baseDir string, secrets runtime.SecretResolver) error {
	st, err := store.NewStore(baseDir)
	if err != nil {
		return err
	}

	authMgr := runtime.NewAuthManager(secrets)
	rt := runtime.NewRuntimeManager(st, authMgr, sdkclient.NewFactory())
	appr := runtime.NewApprovalManager(5 * time.Minute)
	tb := runtime.NewToolBridge(rt, appr)

	w.store = st
	w.auth = authMgr
	w.runtime = rt
	w.approvals = appr
	w.toolBridge = tb

	return nil
}

func (w *MCPWrapper) PutMCPServer(req *spec.PutMCPServerRequest) (*spec.PutMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PutMCPServerResponse, error) {
		return w.store.PutMCPServer(context.Background(), req)
	})
}

func (w *MCPWrapper) GetMCPServer(req *spec.GetMCPServerRequest) (*spec.GetMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetMCPServerResponse, error) {
		return w.store.GetMCPServer(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServers(req *spec.ListMCPServersRequest) (*spec.ListMCPServersResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServersResponse, error) {
		return w.store.ListMCPServers(context.Background(), req)
	})
}

func (w *MCPWrapper) PatchMCPServerEnabled(
	req *spec.PatchMCPServerEnabledRequest,
) (*spec.PatchMCPServerEnabledResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchMCPServerEnabledResponse, error) {
		if req != nil && req.Body != nil && !req.Body.Enabled {
			_, _ = w.runtime.Disconnect(context.Background(), &spec.DisconnectMCPServerRequest{
				ServerID: req.ServerID,
			})
		}
		return w.store.PatchMCPServerEnabled(context.Background(), req)
	})
}

func (w *MCPWrapper) PatchMCPServerPolicy(
	req *spec.PatchMCPServerPolicyRequest,
) (*spec.PatchMCPServerPolicyResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.PatchMCPServerPolicyResponse, error) {
		return w.store.PatchMCPServerPolicy(context.Background(), req)
	})
}

func (w *MCPWrapper) DeleteMCPServer(req *spec.DeleteMCPServerRequest) (*spec.DeleteMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DeleteMCPServerResponse, error) {
		if req != nil {
			_, _ = w.runtime.Disconnect(context.Background(), &spec.DisconnectMCPServerRequest{
				ServerID: req.ServerID,
			})
		}
		return w.store.DeleteMCPServer(context.Background(), req)
	})
}

func (w *MCPWrapper) ConnectMCPServer(req *spec.ConnectMCPServerRequest) (*spec.ConnectMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ConnectMCPServerResponse, error) {
		return w.runtime.Connect(context.Background(), req)
	})
}

func (w *MCPWrapper) DisconnectMCPServer(
	req *spec.DisconnectMCPServerRequest,
) (*spec.DisconnectMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.DisconnectMCPServerResponse, error) {
		return w.runtime.Disconnect(context.Background(), req)
	})
}

func (w *MCPWrapper) RefreshMCPServer(req *spec.RefreshMCPServerRequest) (*spec.RefreshMCPServerResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.RefreshMCPServerResponse, error) {
		return w.runtime.Refresh(context.Background(), req)
	})
}

func (w *MCPWrapper) GetMCPServerStatus(
	req *spec.GetMCPServerStatusRequest,
) (*spec.GetMCPServerStatusResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.GetMCPServerStatusResponse, error) {
		return w.runtime.Status(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServerTools(
	req *spec.ListMCPServerToolsRequest,
) (*spec.ListMCPServerToolsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServerToolsResponse, error) {
		return w.runtime.ListTools(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServerResources(
	req *spec.ListMCPServerResourcesRequest,
) (*spec.ListMCPServerResourcesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServerResourcesResponse, error) {
		return w.runtime.ListResources(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServerResourceTemplates(
	req *spec.ListMCPServerResourceTemplatesRequest,
) (*spec.ListMCPServerResourceTemplatesResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServerResourceTemplatesResponse, error) {
		return w.runtime.ListResourceTemplates(context.Background(), req)
	})
}

func (w *MCPWrapper) ListMCPServerPrompts(
	req *spec.ListMCPServerPromptsRequest,
) (*spec.ListMCPServerPromptsResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ListMCPServerPromptsResponse, error) {
		return w.runtime.ListPrompts(context.Background(), req)
	})
}

func (w *MCPWrapper) ReadMCPResource(
	req *spec.MCPReadResourceRequest,
) (*spec.MCPReadResourceResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.MCPReadResourceResponse, error) {
		return w.runtime.ReadResource(context.Background(), req)
	})
}

func (w *MCPWrapper) GetMCPPrompt(req *spec.MCPGetPromptRequest) (*spec.MCPGetPromptResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.MCPGetPromptResponse, error) {
		return w.runtime.GetPrompt(context.Background(), req)
	})
}

func (w *MCPWrapper) CompleteMCPArgument(
	req *spec.MCPCompleteArgumentRequest,
) (*spec.MCPCompletionResult, error) {
	return middleware.WithRecoveryResp(func() (*spec.MCPCompletionResult, error) {
		return w.runtime.Complete(context.Background(), req)
	})
}

func (w *MCPWrapper) EvaluateMCPToolCall(
	req *spec.EvaluateMCPToolCallRequest,
) (*spec.EvaluateMCPToolCallResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.EvaluateMCPToolCallResponse, error) {
		return w.toolBridge.Evaluate(context.Background(), req)
	})
}

func (w *MCPWrapper) ResolveMCPApproval(
	req *spec.ResolveMCPApprovalRequest,
) (*spec.ResolveMCPApprovalResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.ResolveMCPApprovalResponse, error) {
		if req == nil || req.Body == nil {
			return nil, spec.ErrMCPInvalidRequest
		}
		token, err := w.approvals.Resolve(context.Background(), req.Body.ApprovalID, req.Body.Resolution)
		if err != nil {
			return nil, err
		}
		return &spec.ResolveMCPApprovalResponse{Body: token}, nil
	})
}

func (w *MCPWrapper) InvokeMCPTool(req *spec.InvokeMCPToolRequest) (*spec.InvokeMCPToolResponse, error) {
	return middleware.WithRecoveryResp(func() (*spec.InvokeMCPToolResponse, error) {
		return w.toolBridge.Invoke(context.Background(), req)
	})
}

func (w *MCPWrapper) close() {
	if w == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if w.runtime != nil {
		_ = w.runtime.Close(ctx)
	}
	if w.store != nil {
		_ = w.store.Close()
	}
}
