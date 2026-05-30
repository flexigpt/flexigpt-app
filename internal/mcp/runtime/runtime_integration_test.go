package runtime_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/sdkclient"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store"
)

type integrationState struct {
	mu         sync.Mutex
	echoCalls  int
	laterCalls int
}

func TestRuntimeManagerEndToEndWithStreamableHTTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	state := &integrationState{}

	srv := mcpSDK.NewServer(
		&mcpSDK.Implementation{
			Name:    "integration-server",
			Version: "1.0.0",
		},
		&mcpSDK.ServerOptions{
			Instructions: "integration test server",
			CompletionHandler: func(ctx context.Context, req *mcpSDK.CompleteRequest) (*mcpSDK.CompleteResult, error) {
				return &mcpSDK.CompleteResult{
					Completion: mcpSDK.CompletionResultDetails{
						Values:  []string{"alpha", "beta"},
						Total:   2,
						HasMore: false,
					},
				}, nil
			},
		},
	)

	addTool := func(name, text string, counter *int) {
		srv.AddTool(&mcpSDK.Tool{
			Name:        name,
			Description: "test tool",
			InputSchema: map[string]any{"type": "object"},
		}, func(ctx context.Context, req *mcpSDK.CallToolRequest) (*mcpSDK.CallToolResult, error) {
			state.mu.Lock()
			*counter++
			state.mu.Unlock()

			var args map[string]any
			if req.Params != nil && len(req.Params.Arguments) > 0 {
				_ = json.Unmarshal(req.Params.Arguments, &args)
			}

			msg := fmt.Sprintf("%s:%v", name, args["message"])
			if text != "" {
				msg = text
			}

			return &mcpSDK.CallToolResult{
				Content: []mcpSDK.Content{
					&mcpSDK.TextContent{Text: msg},
				},
			}, nil
		})
	}

	addTool("echo", "", &state.echoCalls)

	srv.AddResource(&mcpSDK.Resource{
		URI:         "file:///demo",
		Name:        "demo",
		Title:       "Demo Resource",
		Description: "demo resource",
		MIMEType:    "text/plain",
	}, func(ctx context.Context, req *mcpSDK.ReadResourceRequest) (*mcpSDK.ReadResourceResult, error) {
		return &mcpSDK.ReadResourceResult{
			Contents: []*mcpSDK.ResourceContents{
				{URI: "file:///demo", MIMEType: "text/plain", Text: "resource-body"},
			},
		}, nil
	})

	srv.AddPrompt(&mcpSDK.Prompt{
		Name:        "greet",
		Title:       "Greet",
		Description: "greet prompt",
	}, func(ctx context.Context, req *mcpSDK.GetPromptRequest) (*mcpSDK.GetPromptResult, error) {
		return &mcpSDK.GetPromptResult{
			Description: "greet prompt",
			Messages: []*mcpSDK.PromptMessage{
				{
					Role: "assistant",
					Content: &mcpSDK.TextContent{
						Text: "prompt-body",
					},
				},
			},
		}, nil
	})

	httph := mcpSDK.NewStreamableHTTPHandler(func(*http.Request) *mcpSDK.Server {
		return srv
	}, &mcpSDK.StreamableHTTPOptions{
		JSONResponse:               true,
		DisableLocalhostProtection: true,
	})

	ht := httptest.NewServer(httph)
	t.Cleanup(ht.Close)

	dir := t.TempDir()
	st, err := store.NewMCPStore(t.Context(), dir)
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	bundleID := bundleitemutils.BundleID("bundle-a")
	if _, err := st.PutMCPBundle(ctx, &spec.PutMCPBundleRequest{
		BundleID: bundleID,
		Body: &spec.PutMCPBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug(bundleID),
			DisplayName: "Integration Bundle",
			IsEnabled:   true,
			Description: "Integration bundle for end-to-end tests",
		},
	}); err != nil {
		t.Fatalf("PutMCPBundle: %v", err)
	}

	policy := spec.MCPServerPolicy{
		DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
		DefaultExecutionMode: spec.MCPExecutionModeManual,
	}
	serverID := spec.MCPServerID("integration-server")

	if _, err := st.PutMCPServer(ctx, &spec.PutMCPServerRequest{
		BundleID: bundleID,
		ServerID: serverID,
		Body: &spec.PutMCPServerPayload{
			DisplayName: "Integration Server",
			Enabled:     true,
			Transport:   spec.MCPTransportStreamableHTTP,
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      ht.URL,
				AuthMode: spec.MCPHTTPAuthNone,
			},
			DefaultPolicy: &policy,
		},
	}); err != nil {
		t.Fatalf("PutMCPServer: %v", err)
	}

	authMgr := auth.NewAuthManager(nil)
	factory := sdkclient.NewFactory()
	rm := runtime.NewRuntimeManager(st, authMgr, factory)
	t.Cleanup(func() { _ = rm.Close(t.Context()) })

	connectResp, err := rm.Connect(ctx, &spec.ConnectMCPServerRequest{BundleID: bundleID, ServerID: serverID})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if connectResp == nil || connectResp.Body == nil {
		t.Fatalf("Connect response body is nil")
	}
	if connectResp.Body.ToolCount != 1 {
		t.Fatalf("ToolCount = %d, want 1", connectResp.Body.ToolCount)
	}
	if connectResp.Body.ResourceCount != 1 {
		t.Fatalf("ResourceCount = %d, want 1", connectResp.Body.ResourceCount)
	}
	if connectResp.Body.PromptCount != 1 {
		t.Fatalf("PromptCount = %d, want 1", connectResp.Body.PromptCount)
	}
	if connectResp.Body.SnapshotDigest == "" {
		t.Fatalf("SnapshotDigest is empty")
	}

	stStatus, err := rm.Status(ctx, &spec.GetMCPServerStatusRequest{BundleID: bundleID, ServerID: serverID})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if stStatus.Body.Status != spec.MCPServerStatusReady {
		t.Fatalf("Status = %q, want ready", stStatus.Body.Status)
	}

	authStatus, ok := authMgr.GetAuthStatus(bundleID, serverID)
	if !ok {
		t.Fatalf("auth status missing")
	}
	if authStatus.State != spec.MCPAuthStateNotRequired {
		t.Fatalf("auth state = %q, want notRequired", authStatus.State)
	}
	if authStatus.Resource != ht.URL {
		t.Fatalf("auth resource = %q, want %q", authStatus.Resource, ht.URL)
	}

	t.Run("call tool", func(t *testing.T) {
		toolsResp, err := rm.ListTools(ctx, &spec.ListMCPServerToolsRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		if len(toolsResp.Body.Tools) != 1 {
			t.Fatalf("ListTools len = %d, want 1", len(toolsResp.Body.Tools))
		}
		tool := toolsResp.Body.Tools[0]

		body, cfg, returnedTool, err := rm.CallTool(ctx, bundleID, serverID, spec.InvokeMCPToolRequestBody{
			Source:           spec.MCPInvocationSourceUser,
			ToolName:         tool.ToolName,
			ProviderToolName: tool.ProviderToolName,
			ToolDigest:       tool.Digest,
			Arguments: map[string]any{
				"message": "hello",
			},
			ToolUseID: "tool-use-1",
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		if cfg.ID != serverID {
			t.Fatalf("cfg.ID = %q, want %q", cfg.ID, serverID)
		}
		if returnedTool.ToolName != "echo" {
			t.Fatalf("tool = %q, want echo", returnedTool.ToolName)
		}
		if len(body.Content) != 1 || body.Content[0].Text != "echo:hello" {
			t.Fatalf("CallTool content = %#v, want echo:hello", body.Content)
		}
		if body.Provenance.ServerDisplayName != "Integration Server" {
			t.Fatalf("Provenance.ServerDisplayName = %q", body.Provenance.ServerDisplayName)
		}
		if body.Provenance.ToolUseID != "tool-use-1" {
			t.Fatalf("Provenance.ToolUseID = %q", body.Provenance.ToolUseID)
		}
		if state.echoCalls != 1 {
			t.Fatalf("echo calls = %d, want 1", state.echoCalls)
		}
	})

	t.Run("resource prompt and completion", func(t *testing.T) {
		readResp, err := rm.ReadResource(ctx, &spec.MCPReadResourceRequest{
			BundleID: bundleID,
			ServerID: serverID,
			Body: &spec.MCPReadResourceRequestBody{
				URI: "file:///demo",
			},
		})
		if err != nil {
			t.Fatalf("ReadResource: %v", err)
		}
		if len(readResp.Body.Contents) != 1 {
			t.Fatalf("ReadResource contents = %#v", readResp.Body.Contents)
		}
		if readResp.Body.Contents[0].Resource == nil || readResp.Body.Contents[0].Resource.Text != "resource-body" {
			t.Fatalf("ReadResource contents = %#v", readResp.Body.Contents[0])
		}
		promptResp, err := rm.GetPrompt(ctx, &spec.MCPGetPromptRequest{
			BundleID: bundleID,
			ServerID: serverID,
			Body: &spec.MCPGetPromptRequestBody{
				PromptName: "greet",
				Arguments: map[string]string{
					"name": "world",
				},
			},
		})
		if err != nil {
			t.Fatalf("GetPrompt: %v", err)
		}
		if len(promptResp.Body.Messages) != 1 {
			t.Fatalf("GetPrompt messages = %d, want 1", len(promptResp.Body.Messages))
		}
		if promptResp.Body.Messages[0].Content.Text != "prompt-body" {
			t.Fatalf("GetPrompt content = %q", promptResp.Body.Messages[0].Content.Text)
		}

		completeResp, err := rm.Complete(ctx, &spec.MCPCompleteArgumentRequest{
			BundleID: bundleID,
			ServerID: serverID,
			Body: &spec.MCPCompleteArgumentRequestBody{
				RefType:       "prompt",
				Name:          "greet",
				ArgumentName:  "name",
				ArgumentValue: "he",
				Context: map[string]string{
					"prefix": "hello",
				},
			},
		})
		if err != nil {
			t.Fatalf("Complete: %v", err)
		}
		if !slicesEqual(completeResp.Values, []string{"alpha", "beta"}) {
			t.Fatalf("Complete values = %#v, want [alpha beta]", completeResp.Values)
		}
	})

	t.Run("refresh after server change", func(t *testing.T) {
		addTool("later", "later:ok", &state.laterCalls)

		updated, err := rm.Refresh(ctx, &spec.RefreshMCPServerRequest{BundleID: bundleID, ServerID: serverID})
		if err != nil {
			t.Fatalf("Refresh: %v", err)
		}
		if updated == nil || updated.Body == nil {
			t.Fatalf("Refresh body is nil")
		}
		if updated.Body.ToolCount != 2 {
			t.Fatalf("Refresh ToolCount = %d, want 2", updated.Body.ToolCount)
		}

		toolsResp, err := rm.ListTools(ctx, &spec.ListMCPServerToolsRequest{BundleID: bundleID, ServerID: serverID})
		if err != nil {
			t.Fatalf("ListTools after refresh: %v", err)
		}
		if len(toolsResp.Body.Tools) != 2 {
			t.Fatalf("ListTools after refresh len = %d, want 2", len(toolsResp.Body.Tools))
		}

		var laterTool spec.MCPToolCapability
		for _, tool := range toolsResp.Body.Tools {
			if tool.ToolName == "later" {
				laterTool = tool
				break
			}
		}
		if laterTool.ToolName != "later" {
			t.Fatalf("later tool not found")
		}

		body, _, _, err := rm.CallTool(ctx, bundleID, serverID, spec.InvokeMCPToolRequestBody{
			Source:           spec.MCPInvocationSourceUser,
			ToolName:         laterTool.ToolName,
			ProviderToolName: laterTool.ProviderToolName,
			ToolDigest:       laterTool.Digest,
			Arguments: map[string]any{
				"message": "ignored",
			},
			ToolUseID: "tool-use-2",
		})
		if err != nil {
			t.Fatalf("CallTool(later): %v", err)
		}
		if len(body.Content) != 1 || body.Content[0].Text != "later:ok" {
			t.Fatalf("later tool content = %#v, want later:ok", body.Content)
		}
		if state.laterCalls != 1 {
			t.Fatalf("later calls = %d, want 1", state.laterCalls)
		}
	})

	t.Run("disconnect", func(t *testing.T) {
		if _, err := rm.Disconnect(
			ctx,
			&spec.DisconnectMCPServerRequest{BundleID: bundleID, ServerID: serverID},
		); err != nil {
			t.Fatalf("Disconnect: %v", err)
		}
		stStatus, err := rm.Status(ctx, &spec.GetMCPServerStatusRequest{BundleID: bundleID, ServerID: serverID})
		if err != nil {
			t.Fatalf("Status after disconnect: %v", err)
		}
		if stStatus.Body.Status != spec.MCPServerStatusDisconnected {
			t.Fatalf("Status after disconnect = %q, want disconnected", stStatus.Body.Status)
		}
		if _, ok := authMgr.GetAuthStatus(bundleID, serverID); ok {
			t.Fatalf("auth status still present after disconnect")
		}
	})
}

func slicesEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
