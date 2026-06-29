package runtime

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/apps"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestRuntimeManagerCurrentSnapshotBranches(t *testing.T) {
	st, bundleID, serverID := newRuntimeStore(t, true)
	mgr := NewMCPRuntimeManager(st, nil, nil)

	t.Run("valid last-known snapshot is returned while disconnected", func(t *testing.T) {
		snap := coverageDiscoverySnapshot(serverID, "alpha")
		snap.Digest = computeDiscoverySnapshotDigest(snap)

		mgr.sessions[serverID] = &sessionState{
			bundleID:          bundleID,
			serverID:          serverID,
			status:            spec.MCPServerStatusDisconnected,
			snapshot:          snap,
			snapshotExpiresAt: time.Now().UTC().Add(time.Minute),
		}

		got, err := mgr.currentSnapshot(t.Context(), bundleID, serverID)
		if err != nil {
			t.Fatalf("currentSnapshot: %v", err)
		}
		if got.Digest != snap.Digest {
			t.Fatalf("Digest = %q, want %q", got.Digest, snap.Digest)
		}
		if len(got.Tools) != 1 || got.Tools[0].ToolName != "alpha" {
			t.Fatalf("Tools = %#v, want alpha", got.Tools)
		}
	})

	t.Run("expired last-known snapshot is cleared", func(t *testing.T) {
		snap := coverageDiscoverySnapshot(serverID, "alpha")
		snap.Digest = computeDiscoverySnapshotDigest(snap)

		mgr.sessions[serverID] = &sessionState{
			bundleID:          bundleID,
			serverID:          serverID,
			status:            spec.MCPServerStatusDisconnected,
			snapshot:          snap,
			snapshotExpiresAt: time.Now().UTC().Add(-time.Minute),
		}

		_, err := mgr.currentSnapshot(t.Context(), bundleID, serverID)
		if err == nil || !errors.Is(err, spec.ErrMCPRuntimeNotReady) {
			t.Fatalf("currentSnapshot expired = %v, want ErrMCPRuntimeNotReady", err)
		}

		state := mgr.sessions[serverID]
		if state == nil {
			t.Fatalf("session state missing after expiry")
		}
		if state.snapshot.Digest != "" || len(state.snapshot.Tools) != 0 {
			t.Fatalf("snapshot was not cleared: %#v", state.snapshot)
		}
		if state.snapshot.BundleID != bundleID || state.snapshot.ServerID != serverID {
			t.Fatalf("cleared snapshot identity = %#v, want bundle/server retained", state.snapshot)
		}
	})
}

func TestRuntimeManagerRequestValidationBranches(t *testing.T) {
	t.Run("connect validation", func(t *testing.T) {
		mgr := NewMCPRuntimeManager(nil, nil, nil)
		cases := []struct {
			name string
			req  *spec.ConnectMCPServerRequest
		}{
			{name: "nil request", req: nil},
			{name: "missing bundleID", req: &spec.ConnectMCPServerRequest{ServerID: "server"}},
			{name: "missing serverID", req: &spec.ConnectMCPServerRequest{BundleID: "bundle"}},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				_, err := mgr.Connect(t.Context(), tt.req)
				if err == nil || !errors.Is(err, spec.ErrMCPInvalidRequest) ||
					!strings.Contains(err.Error(), "bundleID and serverID required") {
					t.Fatalf("Connect error = %v, want invalid request", err)
				}
			})
		}
	})

	t.Run("list and read validation", func(t *testing.T) {
		mgr := NewMCPRuntimeManager(nil, nil, nil)
		tests := []struct {
			name string
			err  func() error
		}{
			{
				name: "list tools",
				err: func() error {
					_, err := mgr.ListTools(t.Context(), nil)
					return err
				},
			},
			{
				name: "list resources",
				err: func() error {
					_, err := mgr.ListResources(t.Context(), nil)
					return err
				},
			},
			{
				name: "list resource templates",
				err: func() error {
					_, err := mgr.ListResourceTemplates(t.Context(), nil)
					return err
				},
			},
			{
				name: "list prompts",
				err: func() error {
					_, err := mgr.ListPrompts(t.Context(), nil)
					return err
				},
			},
			{
				name: "read resource",
				err: func() error {
					_, err := mgr.ReadResource(t.Context(), nil)
					return err
				},
			},
			{
				name: "get prompt",
				err: func() error {
					_, err := mgr.GetPrompt(t.Context(), nil)
					return err
				},
			},
			{
				name: "complete",
				err: func() error {
					_, err := mgr.Complete(t.Context(), nil)
					return err
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.err()
				if err == nil || !errors.Is(err, spec.ErrMCPInvalidRequest) {
					t.Fatalf("error = %v, want ErrMCPInvalidRequest", err)
				}
			})
		}
	})

	t.Run("disconnect bundle mismatch", func(t *testing.T) {
		st, bundleID, serverID := newRuntimeStore(t, true)
		mgr := NewMCPRuntimeManager(st, nil, nil)
		mgr.sessions[serverID] = &sessionState{
			bundleID: bundleID,
			serverID: serverID,
			status:   spec.MCPServerStatusReady,
		}

		_, err := mgr.Disconnect(t.Context(), &spec.DisconnectMCPServerRequest{
			BundleID: "other-bundle",
			ServerID: serverID,
		})
		if err == nil || !errors.Is(err, spec.ErrMCPInvalidRequest) ||
			!strings.Contains(err.Error(), "connected under bundle") {
			t.Fatalf("Disconnect error = %v, want bundle mismatch", err)
		}
	})
}

func TestRuntimeManagerToolBranches(t *testing.T) {
	t.Run("list tools applies overlays and app render info propagates", func(t *testing.T) {
		allow := spec.MCPApprovalRuleAllow
		auto := spec.MCPExecutionModeAuto
		appURI := "ui://echo"

		snap := coverageDiscoverySnapshot(runtimeCoverageServerID, "echo")
		snap.Tools[0].App = &spec.MCPToolAppInfo{
			ResourceURI: appURI,
			Visibility:  []string{apps.VisibilityModel, apps.VisibilityApp},
		}

		mgr, bundleID, serverID := newConnectedRuntimeForTest(t, snap, func(payload *spec.PutMCPServerPayload) {
			payload.AppsPolicy = &spec.MCPAppsPolicy{
				Enabled:                    true,
				AllowAppInitiatedToolCalls: true,
			}
			payload.ToolPolicies = map[string]spec.MCPToolPolicyOverride{
				"echo": {
					ToolName:         "echo",
					ApprovalRule:     &allow,
					ExecutionMode:    &auto,
					ExpectedDigest:   "different-digest",
					AllowStaleDigest: true,
				},
			}
		})

		listResp, err := mgr.ListTools(t.Context(), &spec.ListMCPServerToolsRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		if listResp.Body == nil || len(listResp.Body.Tools) != 1 {
			t.Fatalf("ListTools body = %#v", listResp.Body)
		}
		tool := listResp.Body.Tools[0]
		if tool.App == nil || tool.App.ResourceURI != appURI {
			t.Fatalf("ListTools App = %#v, want %q", tool.App, appURI)
		}
		if tool.ApprovalRule != allow || tool.ExecutionMode != auto {
			t.Fatalf("ListTools overlay = %#v, want allow/auto", tool)
		}

		dryRunBody, _, dryRunTool, err := mgr.CallToolDryRun(
			t.Context(),
			bundleID,
			serverID,
			spec.InvokeMCPToolRequestBody{
				Source:           spec.MCPInvocationSourceUser,
				ToolName:         tool.ToolName,
				ProviderToolName: tool.ProviderToolName,
				ToolDigest:       "stale-digest",
				Arguments: map[string]any{
					"message": "hello",
				},
			},
		)
		if err != nil {
			t.Fatalf("CallToolDryRun: %v", err)
		}
		if dryRunBody == nil || dryRunBody.BundleID != bundleID || dryRunBody.ServerID != serverID {
			t.Fatalf("CallToolDryRun body = %#v", dryRunBody)
		}
		if !dryRunTool.Stale {
			t.Fatalf("CallToolDryRun tool should be stale: %#v", dryRunTool)
		}

		body, cfg, returnedTool, err := mgr.CallTool(t.Context(), bundleID, serverID, spec.InvokeMCPToolRequestBody{
			Source:           spec.MCPInvocationSourceUser,
			ToolName:         tool.ToolName,
			ProviderToolName: tool.ProviderToolName,
			ToolDigest:       "stale-digest",
			Arguments: map[string]any{
				"message": "hello",
			},
			ToolUseID: "tool-use-1",
		})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}
		if cfg.ID != serverID {
			t.Fatalf("CallTool cfg.ID = %q, want %q", cfg.ID, serverID)
		}
		if returnedTool.ToolName != "echo" {
			t.Fatalf("CallTool returned tool = %#v", returnedTool)
		}
		if body == nil || len(body.Content) != 1 || body.Content[0].Text != "called:echo:hello" {
			t.Fatalf("CallTool content = %#v, want called:echo:hello", body)
		}
		if body.App == nil || body.App.ResourceURI != appURI {
			t.Fatalf("CallTool App = %#v, want %q", body.App, appURI)
		}
		if body.Provenance.AppResourceURI != appURI {
			t.Fatalf("CallTool provenance app URI = %q, want %q", body.Provenance.AppResourceURI, appURI)
		}
		if body.Provenance.ToolUseID != "tool-use-1" {
			t.Fatalf("CallTool provenance toolUseID = %q, want tool-use-1", body.Provenance.ToolUseID)
		}
	})

	t.Run("call tool rejects stale digest when not allowed", func(t *testing.T) {
		snap := coverageDiscoverySnapshot(runtimeCoverageServerID, "echo")
		mgr, bundleID, serverID := newConnectedRuntimeForTest(t, snap, nil)

		listResp, err := mgr.ListTools(t.Context(), &spec.ListMCPServerToolsRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		tool := listResp.Body.Tools[0]

		_, _, _, err = mgr.CallTool(t.Context(), bundleID, serverID, spec.InvokeMCPToolRequestBody{
			Source:           spec.MCPInvocationSourceUser,
			ToolName:         tool.ToolName,
			ProviderToolName: tool.ProviderToolName,
			ToolDigest:       tool.Digest + "-stale",
		})
		if err == nil || !errors.Is(err, spec.ErrMCPStaleReference) ||
			!strings.Contains(err.Error(), toolDigestChangedReason) {
			t.Fatalf("CallTool stale digest error = %v, want stale reference", err)
		}
	})

	t.Run("call tool rejects disabled, task-required, and missing tools", func(t *testing.T) {
		cases := []struct {
			name        string
			mutate      func(*spec.MCPDiscoverySnapshot)
			toolName    string
			wantErr     error
			wantMessage string
		}{
			{
				name: "disabled",
				mutate: func(snap *spec.MCPDiscoverySnapshot) {
					snap.Tools[0].Enabled = false
				},
				toolName:    "echo",
				wantErr:     spec.ErrMCPPolicyDenied,
				wantMessage: "disabled or unsupported",
			},
			{
				name: "task required",
				mutate: func(snap *spec.MCPDiscoverySnapshot) {
					snap.Tools[0].TaskSupport = spec.MCPTaskSupportRequired
				},
				toolName:    "echo",
				wantErr:     spec.ErrMCPPolicyDenied,
				wantMessage: "disabled or unsupported",
			},
			{
				name:        "missing",
				toolName:    "missing",
				wantErr:     spec.ErrMCPInvalidRequest,
				wantMessage: "tool missing",
			},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				snap := coverageDiscoverySnapshot(runtimeCoverageServerID, "echo")
				if tt.mutate != nil {
					tt.mutate(&snap)
				}
				mgr, bundleID, serverID := newConnectedRuntimeForTest(t, snap, nil)

				_, _, _, err := mgr.CallTool(t.Context(), bundleID, serverID, spec.InvokeMCPToolRequestBody{
					Source:   spec.MCPInvocationSourceUser,
					ToolName: tt.toolName,
				})
				if err == nil || !errors.Is(err, tt.wantErr) || !strings.Contains(err.Error(), tt.wantMessage) {
					t.Fatalf("CallTool error = %v, want %v containing %q", err, tt.wantErr, tt.wantMessage)
				}
			})
		}
	})
}

func TestToolBridgeValidationBranches(t *testing.T) {
	var nilBridge *ToolBridge
	if _, err := nilBridge.Evaluate(t.Context(), nil); err == nil ||
		!errors.Is(err, spec.ErrMCPRuntimeNotReady) ||
		!strings.Contains(err.Error(), "nil tool bridge") {
		t.Fatalf("nil bridge Evaluate error = %v, want nil tool bridge", err)
	}
	if _, err := nilBridge.Invoke(t.Context(), nil); err == nil ||
		!errors.Is(err, spec.ErrMCPRuntimeNotReady) ||
		!strings.Contains(err.Error(), "nil tool bridge") {
		t.Fatalf("nil bridge Invoke error = %v, want nil tool bridge", err)
	}

	fixture := newRuntimeFixture(t, spec.MCPServerPolicy{
		DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
		DefaultExecutionMode: spec.MCPExecutionModeManual,
	})
	bridge := NewToolBridge(fixture.mgr, NewApprovalManager(time.Minute))

	if _, err := bridge.Evaluate(t.Context(), nil); err == nil ||
		!errors.Is(err, spec.ErrMCPInvalidRequest) ||
		!strings.Contains(err.Error(), "missing request") {
		t.Fatalf("bridge Evaluate nil request error = %v, want missing request", err)
	}
	if _, err := bridge.Invoke(
		t.Context(),
		&spec.InvokeMCPToolRequest{BundleID: fixture.bundleID, ServerID: fixture.serverID},
	); err == nil ||
		!errors.Is(err, spec.ErrMCPInvalidRequest) ||
		!strings.Contains(err.Error(), "missing request") {
		t.Fatalf("bridge Invoke nil body error = %v, want missing request", err)
	}

	listResp, err := fixture.mgr.ListTools(t.Context(), &spec.ListMCPServerToolsRequest{
		BundleID: fixture.bundleID,
		ServerID: fixture.serverID,
	})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	tool := listResp.Body.Tools[0]

	_, err = bridge.Evaluate(t.Context(), &spec.EvaluateMCPToolCallRequest{
		BundleID: fixture.bundleID,
		ServerID: fixture.serverID,
		Body: &spec.InvokeMCPToolRequestBody{
			Source:           spec.MCPInvocationSourceApp,
			ToolName:         tool.ToolName,
			ProviderToolName: tool.ProviderToolName,
			ToolDigest:       tool.Digest,
		},
	})
	if err == nil || !errors.Is(err, spec.ErrMCPInvalidRequest) ||
		!strings.Contains(err.Error(), "appInstanceID is required") {
		t.Fatalf("app source validation error = %v, want appInstanceID required", err)
	}
}

func newConnectedRuntimeForTest(
	t *testing.T,
	snap spec.MCPDiscoverySnapshot,
	configure func(*spec.PutMCPServerPayload),
) (*MCPRuntimeManager, bundleitemutils.BundleID, spec.MCPServerID) {
	t.Helper()

	st, bundleID, serverID := newRuntimeStore(t, true)

	payload := &spec.PutMCPServerPayload{
		DisplayName: "Server A",
		Enabled:     true,
		Transport:   spec.MCPTransportStreamableHTTP,
		StreamableHTTP: &spec.MCPStreamableHTTPConfig{
			URL:      "http://127.0.0.1:1234/mcp",
			AuthMode: spec.MCPHTTPAuthNone,
		},
	}
	if configure != nil {
		configure(payload)
	}
	if _, err := st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		BundleID: bundleID,
		ServerID: serverID,
		Body:     payload,
	}); err != nil {
		t.Fatalf("PutMCPServer: %v", err)
	}

	snap.BundleID = bundleID
	if len(snap.Tools) > 0 {
		for i := range snap.Tools {
			snap.Tools[i].BundleID = bundleID
			snap.Tools[i].ServerID = serverID
		}
	}
	if len(snap.Resources) > 0 {
		for i := range snap.Resources {
			snap.Resources[i].BundleID = bundleID
			snap.Resources[i].ServerID = serverID
		}
	}
	if len(snap.ResourceTemplates) > 0 {
		for i := range snap.ResourceTemplates {
			snap.ResourceTemplates[i].BundleID = bundleID
			snap.ResourceTemplates[i].ServerID = serverID
		}
	}
	if len(snap.Prompts) > 0 {
		for i := range snap.Prompts {
			snap.Prompts[i].BundleID = bundleID
			snap.Prompts[i].ServerID = serverID
		}
	}

	session := &fakeClientSession{
		discoverSnapshots: []spec.MCPDiscoverySnapshot{snap},
	}
	mgr := NewMCPRuntimeManager(st, nil, &fakeClientFactory{session: session})
	if _, err := mgr.Connect(t.Context(), &spec.ConnectMCPServerRequest{
		BundleID: bundleID,
		ServerID: serverID,
	}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(t.Context()) })

	return mgr, bundleID, serverID
}
