//nolint:nilnil // Test.
package runtime

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store"
)

const (
	runtimeCoverageBundleID = bundleitemutils.BundleID("bundle-a")
	runtimeCoverageServerID = spec.MCPServerID("server-a")
)

type sequencedClientFactory struct {
	mu sync.Mutex

	sessions []*fakeClientSession
	idx      int
	err      error

	connectCalls int
}

func (f *sequencedClientFactory) Connect(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	resolved auth.ResolvedTransportAuth,
	events ClientNotificationSink,
) (ClientSession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.connectCalls++
	if f.err != nil {
		return nil, f.err
	}
	if f.idx >= len(f.sessions) {
		return nil, errors.New("no more test sessions")
	}
	s := f.sessions[f.idx]
	f.idx++
	return s, nil
}

type closingClientSession struct {
	closeCalls int
}

func (s *closingClientSession) Close(ctx context.Context) error {
	s.closeCalls++
	return errors.New("close failed")
}

func (s *closingClientSession) Ping(ctx context.Context) error { return nil }

func (s *closingClientSession) Discover(
	ctx context.Context,
	serverID spec.MCPServerID,
	policy spec.MCPServerPolicy,
	trustLevel spec.MCPTrustLevel,
) (spec.MCPDiscoverySnapshot, error) {
	return coverageDiscoverySnapshot(serverID, "discover"), nil
}

func (s *closingClientSession) CallTool(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (*spec.InvokeMCPToolResponseBody, error) {
	return nil, nil
}

func (s *closingClientSession) ReadResource(
	ctx context.Context,
	uri string,
) (*spec.MCPReadResourceResponseBody, error) {
	return nil, nil
}

func (s *closingClientSession) GetPrompt(
	ctx context.Context,
	name string,
	args map[string]string,
) (*spec.MCPGetPromptResponseBody, error) {
	return nil, nil
}

func (s *closingClientSession) Complete(
	ctx context.Context,
	req spec.MCPCompleteArgumentRequestBody,
) (*spec.MCPCompletionResult, error) {
	return nil, nil
}

type nilResponseSession struct{}

func (s *nilResponseSession) Close(ctx context.Context) error { return nil }
func (s *nilResponseSession) Ping(ctx context.Context) error  { return nil }

func (s *nilResponseSession) Discover(
	ctx context.Context,
	serverID spec.MCPServerID,
	policy spec.MCPServerPolicy,
	trustLevel spec.MCPTrustLevel,
) (spec.MCPDiscoverySnapshot, error) {
	return coverageDiscoverySnapshot(serverID, "echo"), nil
}

func (s *nilResponseSession) CallTool(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (*spec.InvokeMCPToolResponseBody, error) {
	return nil, nil
}

func (s *nilResponseSession) ReadResource(
	ctx context.Context,
	uri string,
) (*spec.MCPReadResourceResponseBody, error) {
	return nil, nil
}

func (s *nilResponseSession) GetPrompt(
	ctx context.Context,
	name string,
	args map[string]string,
) (*spec.MCPGetPromptResponseBody, error) {
	return nil, nil
}

func (s *nilResponseSession) Complete(
	ctx context.Context,
	req spec.MCPCompleteArgumentRequestBody,
) (*spec.MCPCompletionResult, error) {
	return nil, nil
}

func TestApprovalManagerEdgeBranches(t *testing.T) {
	summary := spec.MCPApprovalSummary{
		BundleID:   runtimeCoverageBundleID,
		ServerID:   runtimeCoverageServerID,
		ToolName:   "tool",
		ToolDigest: "digest",
		Risk:       spec.MCPToolRiskWrite,
		Arguments:  spec.JSONRawString(`{"b":2,"a":1}`),
	}

	t.Run("random token and decision key are deterministic", func(t *testing.T) {
		tok, err := randomToken()
		if err != nil {
			t.Fatalf("randomToken: %v", err)
		}
		if len(tok) != 43 {
			t.Fatalf("randomToken length = %d, want 43", len(tok))
		}
		if _, err := base64.RawURLEncoding.DecodeString(tok); err != nil {
			t.Fatalf("randomToken is not valid base64url: %v", err)
		}

		got1 := getApprovalDecisionKey(summary)
		got2 := getApprovalDecisionKey(summary)
		if got1 == "" || got1 != got2 {
			t.Fatalf("decision key not deterministic: %q vs %q", got1, got2)
		}
	})

	t.Run("resolve branches", func(t *testing.T) {
		mgr := NewApprovalManager(time.Minute)
		id, err := mgr.Create(t.Context(), summary)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		if _, err := mgr.Resolve(t.Context(), id, spec.MCPApprovalResolution("bogus")); err == nil ||
			!errors.Is(err, spec.ErrMCPInvalidRequest) ||
			!strings.Contains(err.Error(), "invalid resolution") {
			t.Fatalf("Resolve(invalid) err = %v, want invalid resolution", err)
		}

		mgr = NewApprovalManager(time.Minute)
		id, err = mgr.Create(t.Context(), summary)
		if err != nil {
			t.Fatalf("Create #2: %v", err)
		}
		if tok, err := mgr.Resolve(t.Context(), id, spec.MCPApprovalResolutionDenyOnce); err != nil {
			t.Fatalf("Resolve denyOnce: %v", err)
		} else if tok != nil {
			t.Fatalf("Resolve denyOnce token = %#v, want nil", tok)
		}
		if got, ok := mgr.LookupDecision(summary); ok || got != "" {
			t.Fatalf("LookupDecision after denyOnce = %q, %v, want none", got, ok)
		}

		mgr = NewApprovalManager(time.Minute)
		id, err = mgr.Create(t.Context(), summary)
		if err != nil {
			t.Fatalf("Create #3: %v", err)
		}
		if tok, err := mgr.Resolve(t.Context(), id, spec.MCPApprovalResolutionDenyAlways); err != nil {
			t.Fatalf("Resolve denyAlways: %v", err)
		} else if tok != nil {
			t.Fatalf("Resolve denyAlways token = %#v, want nil", tok)
		}
		if got, ok := mgr.LookupDecision(summary); !ok || got != spec.MCPApprovalResolutionDenyAlways {
			t.Fatalf("LookupDecision after denyAlways = %q, %v, want denyAlways", got, ok)
		}

		mgr = NewApprovalManager(time.Minute)
		id, err = mgr.Create(t.Context(), summary)
		if err != nil {
			t.Fatalf("Create #4: %v", err)
		}
		token, err := mgr.Resolve(t.Context(), id, spec.MCPApprovalResolutionAllowOnce)
		if err != nil {
			t.Fatalf("Resolve allowOnce: %v", err)
		}
		if token == nil || token.Token == "" {
			t.Fatalf("Resolve allowOnce token = %#v, want token", token)
		}
		if gotID, err := mgr.VerifyAndConsumeToken(t.Context(), token.Token, summary); err != nil {
			t.Fatalf("VerifyAndConsumeToken: %v", err)
		} else if gotID != id {
			t.Fatalf("VerifyAndConsumeToken id = %q, want %q", gotID, id)
		}
	})

	t.Run("verify branches", func(t *testing.T) {
		mgr := NewApprovalManager(time.Minute)
		id, err := mgr.Create(t.Context(), summary)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		token, err := mgr.Resolve(t.Context(), id, spec.MCPApprovalResolutionAllowOnce)
		if err != nil {
			t.Fatalf("Resolve allowOnce: %v", err)
		}
		if token == nil || token.Token == "" {
			t.Fatalf("Resolve allowOnce token = %#v, want token", token)
		}

		pending := mgr.pending[id]
		expected := summary
		expected.ToolName = "other-tool"

		if err := mgr.VerifyAndConsume(t.Context(), "", token.Token); err == nil ||
			!errors.Is(err, spec.ErrMCPApprovalNeeded) ||
			!strings.Contains(err.Error(), "approval token required") {
			t.Fatalf("VerifyAndConsume(empty id) err = %v, want approval token required", err)
		}
		if err := mgr.VerifyAndConsume(t.Context(), "missing", token.Token); err == nil ||
			!errors.Is(err, spec.ErrMCPApprovalNeeded) ||
			!strings.Contains(err.Error(), "approval not found") {
			t.Fatalf("VerifyAndConsume(missing id) err = %v, want approval not found", err)
		}
		if _, err := mgr.VerifyAndConsumeToken(t.Context(), "", summary); err == nil ||
			!errors.Is(err, spec.ErrMCPApprovalNeeded) ||
			!strings.Contains(err.Error(), "approval token required") {
			t.Fatalf("VerifyAndConsumeToken(empty token) err = %v, want approval token required", err)
		}
		if _, err := mgr.VerifyAndConsumeToken(t.Context(), "missing", summary); err == nil ||
			!errors.Is(err, spec.ErrMCPApprovalNeeded) ||
			!strings.Contains(err.Error(), "approval not found") {
			t.Fatalf("VerifyAndConsumeToken(missing token) err = %v, want approval not found", err)
		}
		if err := mgr.VerifyAndConsume(t.Context(), id, "wrong-token"); err == nil ||
			!errors.Is(err, spec.ErrMCPApprovalNeeded) ||
			!strings.Contains(err.Error(), "bad approval token") {
			t.Fatalf("VerifyAndConsume(bad token) err = %v, want bad approval token", err)
		}
		if err := mgr.verifyLocked(pending, token.Token, &expected); err == nil ||
			!errors.Is(err, spec.ErrMCPApprovalNeeded) ||
			!strings.Contains(err.Error(), "does not match requested tool call") {
			t.Fatalf("verifyLocked(summary mismatch) err = %v, want summary mismatch", err)
		}
		if err := mgr.VerifyAndConsume(t.Context(), id, token.Token); err != nil {
			t.Fatalf("VerifyAndConsume(success): %v", err)
		}
		mgr.pending[id] = pending
		if err := mgr.verifyLocked(pending, token.Token, nil); err == nil ||
			!errors.Is(err, spec.ErrMCPApprovalNeeded) ||
			!strings.Contains(err.Error(), "already consumed") {
			t.Fatalf("verifyLocked(consumed) err = %v, want already consumed", err)
		}
	})
}

func TestProviderToolNameTruncationAndSanitization(t *testing.T) {
	serverID := spec.MCPServerID("1234567890-very-long-server-name-!@#")
	toolName := strings.Repeat("tool-name-", 12) + "extra"

	got := ProviderToolName(serverID, toolName)
	if got != ProviderToolName(serverID, toolName) {
		t.Fatalf("ProviderToolName not deterministic: %q", got)
	}
	if len(got) > maxProviderToolNameLen {
		t.Fatalf("ProviderToolName length = %d, want <= %d", len(got), maxProviderToolNameLen)
	}

	parts := strings.Split(got, "__")
	if len(parts) != 4 {
		t.Fatalf("ProviderToolName format = %q, want 4 parts", got)
	}
	if !strings.HasPrefix(parts[1], "s_") {
		t.Fatalf("ProviderToolName server part = %q, want sanitized leading-digit prefix", parts[1])
	}
	if len(parts[1]) > maxServerPartLen {
		t.Fatalf("server part length = %d, want <= %d", len(parts[1]), maxServerPartLen)
	}
	if len(parts[2]) < minToolPartLen {
		t.Fatalf("tool part length = %d, want >= %d", len(parts[2]), minToolPartLen)
	}

	choice := ChoiceID(serverID, toolName)
	if choice != ChoiceID(serverID, toolName) {
		t.Fatalf("ChoiceID not deterministic: %q", choice)
	}
	if len(choice) != len("mcp-")+16 {
		t.Fatalf("ChoiceID length = %d, want %d", len(choice), len("mcp-")+16)
	}
}

func TestRuntimeManagerSnapshotReadyClientAndRefreshBranches(t *testing.T) {
	st, bundleID, serverID := newRuntimeStore(t, true)
	mgr := NewMCPRuntimeManager(st, nil, nil)

	t.Run("status with no session is disconnected", func(t *testing.T) {
		statusResp, err := mgr.Status(t.Context(), &spec.GetMCPServerStatusRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if statusResp.Body == nil {
			t.Fatalf("Status body is nil")
		}
		if statusResp.Body.Status != spec.MCPServerStatusDisconnected {
			t.Fatalf("Status = %q, want disconnected", statusResp.Body.Status)
		}
	})

	t.Run("refresh without a connected session is not ready", func(t *testing.T) {
		refreshMgr := NewMCPRuntimeManager(st, nil, nil)
		_, err := refreshMgr.Refresh(t.Context(), &spec.RefreshMCPServerRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err == nil || !errors.Is(err, spec.ErrMCPRuntimeNotReady) {
			t.Fatalf("Refresh = %v, want ErrMCPRuntimeNotReady", err)
		}
	})

	t.Run("ready client and status with in-memory session", func(t *testing.T) {
		session := &fakeClientSession{
			discoverSnapshots: []spec.MCPDiscoverySnapshot{
				coverageDiscoverySnapshot(serverID, "session-tool"),
			},
		}
		mgr.sessions[serverID] = &sessionState{
			bundleID: bundleID,
			serverID: serverID,
			status:   spec.MCPServerStatusReady,
			client:   session,
			snapshot: coverageDiscoverySnapshot(serverID, "session-tool"),
		}

		statusResp, err := mgr.Status(t.Context(), &spec.GetMCPServerStatusRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("Status(ready): %v", err)
		}
		if statusResp.Body == nil {
			t.Fatalf("Status body is nil")
		}
		if statusResp.Body.Status != spec.MCPServerStatusReady {
			t.Fatalf("Status = %q, want ready", statusResp.Body.Status)
		}
		if statusResp.Body.ToolCount != 1 {
			t.Fatalf("ToolCount = %d, want 1", statusResp.Body.ToolCount)
		}

		got, err := mgr.currentSnapshot(t.Context(), bundleID, serverID)
		if err != nil {
			t.Fatalf("currentSnapshot(session): %v", err)
		}
		if len(got.Tools) != 1 || got.Tools[0].ToolName != "session-tool" {
			t.Fatalf("currentSnapshot(session) = %#v", got.Tools)
		}

		client, cfg, err := mgr.readyClient(t.Context(), bundleID, serverID)
		if err != nil {
			t.Fatalf("readyClient(session): %v", err)
		}
		if client != session {
			t.Fatalf("readyClient returned %#v, want %#v", client, session)
		}
		if cfg.ID != serverID {
			t.Fatalf("readyClient cfg.ID = %q, want %q", cfg.ID, serverID)
		}

		mgr.sessions[serverID].status = spec.MCPServerStatusDisconnected
		if _, _, err := mgr.readyClient(t.Context(), bundleID, serverID); err == nil ||
			!errors.Is(err, spec.ErrMCPRuntimeNotReady) ||
			!strings.Contains(err.Error(), "not connected") {
			t.Fatalf("readyClient(disconnected) err = %v, want not connected", err)
		}
	})

	t.Run("disabled servers are rejected by readyClient", func(t *testing.T) {
		stDisabled, bundleID2, serverID2 := newRuntimeStore(t, false)
		mgrDisabled := NewMCPRuntimeManager(stDisabled, nil, nil)

		if _, _, err := mgrDisabled.readyClient(t.Context(), bundleID2, serverID2); err == nil ||
			!errors.Is(err, spec.ErrMCPServerDisabled) {
			t.Fatalf("readyClient(disabled) err = %v, want ErrMCPServerDisabled", err)
		}
	})
}

func TestRuntimeManagerConnectBranches(t *testing.T) {
	t.Run("factory error publishes status error", func(t *testing.T) {
		st, bundleID, serverID := newRuntimeStore(t, true)
		factory := &sequencedClientFactory{
			err: errors.New("connect failed"),
		}
		mgr := NewMCPRuntimeManager(st, nil, factory)

		_, err := mgr.Connect(t.Context(), &spec.ConnectMCPServerRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err == nil || !strings.Contains(err.Error(), "connect failed") {
			t.Fatalf("Connect(factory error) err = %v, want connect failed", err)
		}
		if factory.connectCalls != 1 {
			t.Fatalf("factory connectCalls = %d, want 1", factory.connectCalls)
		}

		statusResp, err := mgr.Status(t.Context(), &spec.GetMCPServerStatusRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("Status(after factory error): %v", err)
		}
		if statusResp.Body.Status != spec.MCPServerStatusError {
			t.Fatalf("Status = %q, want error", statusResp.Body.Status)
		}
		if !strings.Contains(statusResp.Body.LastError, "connect failed") {
			t.Fatalf("LastError = %q, want connect failed", statusResp.Body.LastError)
		}
	})

	t.Run("disabled server never reaches factory", func(t *testing.T) {
		st, bundleID, serverID := newRuntimeStore(t, false)
		factory := &sequencedClientFactory{}
		mgr := NewMCPRuntimeManager(st, nil, factory)

		_, err := mgr.Connect(t.Context(), &spec.ConnectMCPServerRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err == nil || !errors.Is(err, spec.ErrMCPServerDisabled) {
			t.Fatalf("Connect(disabled) err = %v, want ErrMCPServerDisabled", err)
		}
		if factory.connectCalls != 0 {
			t.Fatalf("factory connectCalls = %d, want 0", factory.connectCalls)
		}

		statusResp, err := mgr.Status(t.Context(), &spec.GetMCPServerStatusRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("Status(after disabled connect): %v", err)
		}
		if statusResp.Body.Status != spec.MCPServerStatusDisabled {
			t.Fatalf("Status = %q, want disabled", statusResp.Body.Status)
		}
	})

	t.Run("reconnect closes old session and replaces snapshot", func(t *testing.T) {
		st, bundleID, serverID := newRuntimeStore(t, true)

		session1 := &fakeClientSession{
			discoverSnapshots: []spec.MCPDiscoverySnapshot{
				coverageDiscoverySnapshot(serverID, "alpha"),
			},
		}
		session2 := &fakeClientSession{
			discoverSnapshots: []spec.MCPDiscoverySnapshot{
				coverageDiscoverySnapshot(serverID, "beta"),
			},
		}
		factory := &sequencedClientFactory{
			sessions: []*fakeClientSession{session1, session2},
		}
		mgr := NewMCPRuntimeManager(st, nil, factory)

		resp1, err := mgr.Connect(t.Context(), &spec.ConnectMCPServerRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("Connect #1: %v", err)
		}
		if resp1 == nil || resp1.Body == nil {
			t.Fatalf("Connect #1 response body is nil")
		}

		tools1, err := mgr.ListTools(t.Context(), &spec.ListMCPServerToolsRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("ListTools after connect #1: %v", err)
		}
		if len(tools1.Body.Tools) != 1 || tools1.Body.Tools[0].ToolName != "alpha" {
			t.Fatalf("ListTools after connect #1 = %#v, want alpha", tools1.Body.Tools)
		}

		resp2, err := mgr.Connect(t.Context(), &spec.ConnectMCPServerRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("Connect #2: %v", err)
		}
		if resp2 == nil || resp2.Body == nil {
			t.Fatalf("Connect #2 response body is nil")
		}
		if session1.closeCalls != 1 {
			t.Fatalf("session1 closeCalls = %d, want 1", session1.closeCalls)
		}
		if session2.closeCalls != 0 {
			t.Fatalf("session2 closeCalls = %d, want 0", session2.closeCalls)
		}
		if factory.connectCalls != 2 {
			t.Fatalf("factory connectCalls = %d, want 2", factory.connectCalls)
		}

		tools2, err := mgr.ListTools(t.Context(), &spec.ListMCPServerToolsRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("ListTools after connect #2: %v", err)
		}
		if len(tools2.Body.Tools) != 1 || tools2.Body.Tools[0].ToolName != "beta" {
			t.Fatalf("ListTools after connect #2 = %#v, want beta", tools2.Body.Tools)
		}

		statusResp, err := mgr.Status(t.Context(), &spec.GetMCPServerStatusRequest{
			BundleID: bundleID,
			ServerID: serverID,
		})
		if err != nil {
			t.Fatalf("Status(after reconnect): %v", err)
		}
		if statusResp.Body.Status != spec.MCPServerStatusReady {
			t.Fatalf("Status = %q, want ready", statusResp.Body.Status)
		}
	})
}

func TestRuntimeManagerCloseBranches(t *testing.T) {
	ctx := t.Context()

	st, bundleID, serverID := newRuntimeStore(t, true)
	authMgr := auth.NewAuthManager(nil)
	if err := authMgr.SaveAuthStatus(t.Context(), spec.MCPAuthStatus{
		BundleID: bundleID,
		ServerID: serverID,
		AuthMode: spec.MCPHTTPAuthNone,
		State:    spec.MCPAuthStateNotRequired,
	}); err != nil {
		t.Fatalf("SaveAuthStatus: %v", err)
	}

	client := &closingClientSession{}
	mgr := NewMCPRuntimeManager(st, authMgr, nil)
	mgr.sessions[serverID] = &sessionState{
		bundleID: bundleID,
		serverID: serverID,
		status:   spec.MCPServerStatusReady,
		client:   client,
	}
	mgr.notificationRefreshTimers[serverID] = time.NewTimer(time.Hour)

	if err := mgr.Close(ctx); err == nil || !strings.Contains(err.Error(), "close failed") {
		t.Fatalf("Close err = %v, want close failed", err)
	}
	if client.closeCalls != 1 {
		t.Fatalf("client closeCalls = %d, want 1", client.closeCalls)
	}
	if _, ok := authMgr.GetAuthStatus(bundleID, serverID); ok {
		t.Fatalf("auth status still present after Close")
	}
	if len(mgr.sessions) != 0 {
		t.Fatalf("sessions len = %d, want 0", len(mgr.sessions))
	}
	if len(mgr.notificationRefreshTimers) != 0 {
		t.Fatalf("notification timers len = %d, want 0", len(mgr.notificationRefreshTimers))
	}
}

func TestRuntimeManagerNilResponseBranches(t *testing.T) {
	st, bundleID, serverID := newRuntimeStore(t, true)
	mgr := NewMCPRuntimeManager(st, nil, nil)

	mgr.sessions[serverID] = &sessionState{
		bundleID: bundleID,
		serverID: serverID,
		status:   spec.MCPServerStatusReady,
		client:   &nilResponseSession{},
		snapshot: coverageDiscoverySnapshot(serverID, "echo"),
	}

	t.Run("call tool nil response", func(t *testing.T) {
		_, _, _, err := mgr.CallTool(t.Context(), bundleID, serverID, spec.InvokeMCPToolRequestBody{
			Source:   spec.MCPInvocationSourceUser,
			ToolName: "echo",
		})
		if err == nil || !errors.Is(err, spec.ErrMCPRuntimeNotReady) ||
			!strings.Contains(err.Error(), "tool call returned nil response") {
			t.Fatalf("CallTool err = %v, want nil response", err)
		}
	})

	t.Run("read resource nil response", func(t *testing.T) {
		_, err := mgr.ReadResource(t.Context(), &spec.MCPReadResourceRequest{
			BundleID: bundleID,
			ServerID: serverID,
			Body: &spec.MCPReadResourceRequestBody{
				URI: "file:///demo",
			},
		})
		if err == nil || !errors.Is(err, spec.ErrMCPRuntimeNotReady) ||
			!strings.Contains(err.Error(), "resource read returned nil response") {
			t.Fatalf("ReadResource err = %v, want nil response", err)
		}
	})

	t.Run("get prompt nil response", func(t *testing.T) {
		_, err := mgr.GetPrompt(t.Context(), &spec.MCPGetPromptRequest{
			BundleID: bundleID,
			ServerID: serverID,
			Body: &spec.MCPGetPromptRequestBody{
				PromptName: "greet",
			},
		})
		if err == nil || !errors.Is(err, spec.ErrMCPRuntimeNotReady) ||
			!strings.Contains(err.Error(), "prompt read returned nil response") {
			t.Fatalf("GetPrompt err = %v, want nil response", err)
		}
	})

	t.Run("complete nil response", func(t *testing.T) {
		_, err := mgr.Complete(t.Context(), &spec.MCPCompleteArgumentRequest{
			BundleID: bundleID,
			ServerID: serverID,
			Body: &spec.MCPCompleteArgumentRequestBody{
				RefType:       "prompt",
				Name:          "greet",
				ArgumentName:  "name",
				ArgumentValue: "he",
			},
		})
		if err == nil || !errors.Is(err, spec.ErrMCPRuntimeNotReady) ||
			!strings.Contains(err.Error(), "completion returned nil response") {
			t.Fatalf("Complete err = %v, want nil response", err)
		}
	})
}

func TestToolBridgeCachedDecisionAndDeniedBranches(t *testing.T) {
	t.Run("cached decisions are applied", func(t *testing.T) {
		summary := spec.MCPApprovalSummary{
			BundleID: runtimeCoverageBundleID,
			ServerID: runtimeCoverageServerID,
			ToolName: "tool",
			Risk:     spec.MCPToolRiskWrite,
		}

		approvals := NewApprovalManager(time.Minute)
		approvals.mu.Lock()
		approvals.decisions[getApprovalDecisionKey(summary)] = spec.MCPApprovalResolutionAllowAlways
		approvals.mu.Unlock()

		bridge := NewToolBridge(nil, approvals)
		got := bridge.applyCachedDecision(spec.MCPApprovalEvaluation{
			Decision: spec.MCPApprovalDecisionApprovalRequired,
			Summary:  &summary,
		})
		if got.Decision != spec.MCPApprovalDecisionAllowed || got.Reason != "cached allow-always decision" {
			t.Fatalf("applyCachedDecision(allowAlways) = %#v", got)
		}

		approvals.mu.Lock()
		approvals.decisions[getApprovalDecisionKey(summary)] = spec.MCPApprovalResolutionDenyAlways
		approvals.mu.Unlock()

		got = bridge.applyCachedDecision(spec.MCPApprovalEvaluation{
			Decision: spec.MCPApprovalDecisionApprovalRequired,
			Summary:  &summary,
		})
		if got.Decision != spec.MCPApprovalDecisionDenied || got.Reason != "cached deny-always decision" {
			t.Fatalf("applyCachedDecision(denyAlways) = %#v", got)
		}
	})

	t.Run("approval required without token", func(t *testing.T) {
		fixture := newRuntimeFixture(t, spec.MCPServerPolicy{
			DefaultApprovalRule:  spec.MCPApprovalRuleAsk,
			DefaultExecutionMode: spec.MCPExecutionModeManual,
		})
		approvals := NewApprovalManager(time.Minute)
		bridge := NewToolBridge(fixture.mgr, approvals)

		listResp, err := fixture.mgr.ListTools(t.Context(), &spec.ListMCPServerToolsRequest{
			BundleID: fixture.bundleID,
			ServerID: fixture.serverID,
		})
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		if len(listResp.Body.Tools) != 1 {
			t.Fatalf("ListTools len = %d, want 1", len(listResp.Body.Tools))
		}
		tool := listResp.Body.Tools[0]

		eval, err := bridge.Evaluate(t.Context(), &spec.EvaluateMCPToolCallRequest{
			BundleID: fixture.bundleID,
			ServerID: fixture.serverID,
			Body: &spec.InvokeMCPToolRequestBody{
				Source:           spec.MCPInvocationSourceUser,
				ToolName:         tool.ToolName,
				ProviderToolName: tool.ProviderToolName,
				ToolDigest:       tool.Digest,
				Arguments: map[string]any{
					"message": "hello",
				},
			},
		})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if eval.Body == nil || eval.Body.Decision != spec.MCPApprovalDecisionApprovalRequired {
			t.Fatalf("Evaluate = %#v, want approvalRequired", eval.Body)
		}
		if eval.Body.ApprovalID == "" {
			t.Fatalf("Evaluate approvalID is empty")
		}

		_, err = bridge.Invoke(t.Context(), &spec.InvokeMCPToolRequest{
			BundleID: fixture.bundleID,
			ServerID: fixture.serverID,
			Body: &spec.InvokeMCPToolRequestBody{
				Source:           spec.MCPInvocationSourceUser,
				ToolName:         tool.ToolName,
				ProviderToolName: tool.ProviderToolName,
				ToolDigest:       tool.Digest,
				Arguments: map[string]any{
					"message": "hello",
				},
			},
		})
		if err == nil || !errors.Is(err, spec.ErrMCPApprovalNeeded) {
			t.Fatalf("Invoke(no token) err = %v, want ErrMCPApprovalNeeded", err)
		}
	})

	t.Run("invoke rejects stale digest", func(t *testing.T) {
		fixture := newRuntimeFixture(t, spec.MCPServerPolicy{
			DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
			DefaultExecutionMode: spec.MCPExecutionModeManual,
		})
		bridge := NewToolBridge(fixture.mgr, NewApprovalManager(time.Minute))

		listResp, err := fixture.mgr.ListTools(t.Context(), &spec.ListMCPServerToolsRequest{
			BundleID: fixture.bundleID,
			ServerID: fixture.serverID,
		})
		if err != nil {
			t.Fatalf("ListTools: %v", err)
		}
		if len(listResp.Body.Tools) != 1 {
			t.Fatalf("ListTools len = %d, want 1", len(listResp.Body.Tools))
		}
		tool := listResp.Body.Tools[0]

		_, err = bridge.Invoke(t.Context(), &spec.InvokeMCPToolRequest{
			BundleID: fixture.bundleID,
			ServerID: fixture.serverID,
			Body: &spec.InvokeMCPToolRequestBody{
				Source:           spec.MCPInvocationSourceUser,
				ToolName:         tool.ToolName,
				ProviderToolName: tool.ProviderToolName,
				ToolDigest:       tool.Digest + "-stale",
				Arguments: map[string]any{
					"message": "hello",
				},
			},
		})
		if err == nil || !errors.Is(err, spec.ErrMCPStaleReference) ||
			!strings.Contains(err.Error(), toolDigestChangedReason) {
			t.Fatalf("Invoke(stale digest) err = %v, want stale reference", err)
		}
	})
}

func newRuntimeStore(t *testing.T, enabled bool) (*store.Store, bundleitemutils.BundleID, spec.MCPServerID) {
	t.Helper()

	st, err := store.NewMCPStore(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}

	bundleID := runtimeCoverageBundleID
	if _, err := st.PutMCPBundle(t.Context(), &spec.PutMCPBundleRequest{
		BundleID: bundleID,
		Body: &spec.PutMCPBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug(bundleID),
			DisplayName: "Bundle A",
			IsEnabled:   true,
			Description: "Bundle A bundle",
		},
	}); err != nil {
		t.Fatalf("PutMCPBundle: %v", err)
	}

	serverID := runtimeCoverageServerID
	if _, err := st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		BundleID: bundleID,
		ServerID: serverID,
		Body: &spec.PutMCPServerPayload{
			DisplayName: "Server A",
			Enabled:     enabled,
			Transport:   spec.MCPTransportStreamableHTTP,
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      "http://127.0.0.1:1234/mcp",
				AuthMode: spec.MCPHTTPAuthNone,
			},
		},
	}); err != nil {
		t.Fatalf("PutMCPServer: %v", err)
	}

	t.Cleanup(func() { _ = st.Close() })
	return st, bundleID, serverID
}

func coverageDiscoverySnapshot(serverID spec.MCPServerID, toolNames ...string) spec.MCPDiscoverySnapshot {
	snap := spec.MCPDiscoverySnapshot{
		BundleID: runtimeCoverageBundleID,
		ServerID: serverID,
	}
	for _, name := range toolNames {
		snap.Tools = append(snap.Tools, coverageTool(serverID, name))
	}
	return snap
}

func coverageTool(serverID spec.MCPServerID, toolName string) spec.MCPToolCapability {
	return spec.MCPToolCapability{
		BundleID:         runtimeCoverageBundleID,
		ServerID:         serverID,
		ToolName:         toolName,
		ProviderToolName: ProviderToolName(serverID, toolName),
		ChoiceID:         ChoiceID(serverID, toolName),
		DisplayName:      toolName,
		Digest:           "digest-" + toolName,
		Enabled:          true,
		TaskSupport:      spec.MCPTaskSupportForbidden,
		InferredRisk:     spec.MCPToolRiskUnknown,
		ApprovalRule:     spec.MCPApprovalRuleAllow,
		ExecutionMode:    spec.MCPExecutionModeManual,
	}
}
