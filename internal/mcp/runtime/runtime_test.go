package runtime

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store"
)

type fakeClientSession struct {
	mu sync.Mutex

	discoverSnapshots []spec.MCPDiscoverySnapshot
	discoverCalls     int
	closeCalls        int

	lastToolName string
	lastToolArgs map[string]any
	lastURI      string
	lastPrompt   string
	lastComplete spec.MCPCompleteArgumentRequestBody
}

func (f *fakeClientSession) Close(ctx context.Context) error {
	f.mu.Lock()
	f.closeCalls++
	f.mu.Unlock()
	return nil
}

func (f *fakeClientSession) Ping(ctx context.Context) error { return nil }

func (f *fakeClientSession) Discover(
	ctx context.Context,
	serverID spec.MCPServerID,
	policy spec.MCPServerPolicy,
	trustLevel spec.MCPTrustLevel,
) (spec.MCPDiscoverySnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.discoverCalls++
	idx := f.discoverCalls - 1
	if len(f.discoverSnapshots) == 0 {
		return spec.MCPDiscoverySnapshot{ServerID: serverID}, nil
	}
	if idx >= len(f.discoverSnapshots) {
		idx = len(f.discoverSnapshots) - 1
	}
	snap := cloneDiscoverySnapshot(f.discoverSnapshots[idx])
	snap.ServerID = serverID
	for i := range snap.Tools {
		snap.Tools[i].ServerID = serverID
	}
	for i := range snap.Resources {
		snap.Resources[i].ServerID = serverID
	}
	for i := range snap.ResourceTemplates {
		snap.ResourceTemplates[i].ServerID = serverID
	}
	for i := range snap.Prompts {
		snap.Prompts[i].ServerID = serverID
	}
	return snap, nil
}

func (f *fakeClientSession) CallTool(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (*spec.InvokeMCPToolResponseBody, error) {
	f.mu.Lock()
	f.lastToolName = toolName
	f.lastToolArgs = maps.Clone(args)
	f.mu.Unlock()

	msg := "called:" + toolName
	if v, ok := args["message"].(string); ok && v != "" {
		msg = msg + ":" + v
	}

	return &spec.InvokeMCPToolResponseBody{
		Content: []spec.MCPContent{
			{Type: spec.MCPContentTypeText, Text: msg},
		},
		StructuredContent: map[string]any{
			"tool": toolName,
		},
	}, nil
}

func (f *fakeClientSession) ReadResource(
	ctx context.Context,
	uri string,
) (*spec.MCPReadResourceResponseBody, error) {
	f.mu.Lock()
	f.lastURI = uri
	f.mu.Unlock()

	return &spec.MCPReadResourceResponseBody{
		URI: uri,
		Contents: []spec.MCPContent{
			{Type: spec.MCPContentTypeText, Text: "resource:" + uri},
		},
	}, nil
}

func (f *fakeClientSession) GetPrompt(
	ctx context.Context,
	name string,
	args map[string]string,
) (*spec.MCPGetPromptResponseBody, error) {
	f.mu.Lock()
	f.lastPrompt = name
	f.mu.Unlock()

	return &spec.MCPGetPromptResponseBody{
		PromptName:  name,
		Description: "prompt:" + name,
		Messages: []spec.MCPPromptMessage{
			{
				Role: "assistant",
				Content: spec.MCPContent{
					Type: spec.MCPContentTypeText,
					Text: "prompt:" + name,
				},
			},
		},
	}, nil
}

func (f *fakeClientSession) Complete(
	ctx context.Context,
	req spec.MCPCompleteArgumentRequestBody,
) (*spec.MCPCompletionResult, error) {
	f.mu.Lock()
	f.lastComplete = req
	f.mu.Unlock()

	return &spec.MCPCompletionResult{
		Values:  []string{req.ArgumentValue, req.ArgumentValue + "-2"},
		Total:   2,
		HasMore: false,
	}, nil
}

type fakeClientFactory struct {
	session *fakeClientSession

	mu           sync.Mutex
	connectCalls int
	lastResolved auth.ResolvedTransportAuth
}

func (f *fakeClientFactory) Connect(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	resolved auth.ResolvedTransportAuth,
	events ClientNotificationSink,
) (ClientSession, error) {
	f.mu.Lock()
	f.connectCalls++
	f.lastResolved = resolved
	f.mu.Unlock()
	return f.session, nil
}

type runtimeFixture struct {
	mgr      *RuntimeManager
	session  *fakeClientSession
	serverID spec.MCPServerID
}

func newRuntimeFixture(
	t *testing.T,
	policy spec.MCPServerPolicy,
	snapshots ...spec.MCPDiscoverySnapshot,
) *runtimeFixture {
	t.Helper()

	dir := t.TempDir()
	st, err := store.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	serverID := spec.MCPServerID("server-1")

	if len(snapshots) == 0 {
		snapshots = []spec.MCPDiscoverySnapshot{
			makeDiscoverySnapshot(serverID, "echo"),
		}
	}

	if _, err := st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		ServerID: serverID,
		Body: &spec.PutMCPServerPayload{
			DisplayName: "Fixture Server",
			Enabled:     true,
			Transport:   spec.MCPTransportStreamableHTTP,
			StreamableHTTP: &spec.MCPStreamableHTTPConfig{
				URL:      "http://127.0.0.1:1234/mcp",
				AuthMode: spec.MCPHTTPAuthNone,
			},
			DefaultPolicy: &policy,
		},
	}); err != nil {
		t.Fatalf("PutMCPServer: %v", err)
	}

	session := &fakeClientSession{discoverSnapshots: snapshots}
	factory := &fakeClientFactory{session: session}
	mgr := NewRuntimeManager(st, nil, factory)
	if _, err := mgr.Connect(t.Context(), &spec.ConnectMCPServerRequest{ServerID: serverID}); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	t.Cleanup(func() { _ = mgr.Close(t.Context()) })

	return &runtimeFixture{
		mgr:      mgr,
		session:  session,
		serverID: serverID,
	}
}

func makeDiscoverySnapshot(serverID spec.MCPServerID, toolNames ...string) spec.MCPDiscoverySnapshot {
	snap := spec.MCPDiscoverySnapshot{ServerID: serverID}
	for _, name := range toolNames {
		snap.Tools = append(snap.Tools, makeTool(serverID, name))
	}
	return snap
}

func makeTool(serverID spec.MCPServerID, name string) spec.MCPToolCapability {
	return spec.MCPToolCapability{
		ServerID:         serverID,
		ToolName:         name,
		ProviderToolName: ProviderToolName(serverID, name),
		ChoiceID:         ChoiceID(serverID, name),
		DisplayName:      name,
		Digest:           "digest-" + name,
		Enabled:          true,
		TaskSupport:      spec.MCPTaskSupportForbidden,
		InferredRisk:     spec.MCPToolRiskUnknown,
	}
}

func TestSanitizeNameAndIDs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "alpha", want: "alpha"},
		{name: "trim and replace", in: "  hello world  ", want: "hello_world"},
		{name: "leading digits", in: "123abc", want: "s_123abc"},
		{name: "empty", in: "   ", want: "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeName(tt.in); got != tt.want {
				t.Fatalf("sanitizeName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}

	serverID := spec.MCPServerID("srv-1")
	toolName := "tool name"

	gotTool := ProviderToolName(serverID, toolName)
	if gotTool != ProviderToolName(serverID, toolName) {
		t.Fatalf("ProviderToolName is not deterministic")
	}
	if len(gotTool) > maxProviderToolNameLen {
		t.Fatalf("ProviderToolName too long: %d", len(gotTool))
	}

	gotChoice := ChoiceID(serverID, toolName)
	if gotChoice != ChoiceID(serverID, toolName) {
		t.Fatalf("ChoiceID is not deterministic")
	}
	if len(gotChoice) != len("mcp-")+16 {
		t.Fatalf("ChoiceID length = %d, want %d", len(gotChoice), len("mcp-")+16)
	}
}

func TestEvaluateAndExecutionMode(t *testing.T) {
	allow := spec.MCPApprovalRuleAllow
	deny := spec.MCPApprovalRuleDeny
	ask := spec.MCPApprovalRuleAsk
	auto := spec.MCPExecutionModeAuto
	manual := spec.MCPExecutionModeManual

	baseServer := spec.MCPServerConfig{
		ID: "server",
		DefaultPolicy: spec.MCPServerPolicy{
			DefaultApprovalRule:  ask,
			DefaultExecutionMode: manual,
		},
	}
	baseTool := spec.MCPToolCapability{
		ServerID:     baseServer.ID,
		ToolName:     "echo",
		Digest:       "digest",
		Enabled:      true,
		TaskSupport:  spec.MCPTaskSupportForbidden,
		InferredRisk: spec.MCPToolRiskUnknown,
	}

	tests := []struct {
		name         string
		mutate       func(*spec.MCPServerConfig, *spec.MCPToolCapability)
		wantDecision spec.MCPApprovalDecision
		wantReason   string
	}{
		{
			name: "disabled",
			mutate: func(_ *spec.MCPServerConfig, tool *spec.MCPToolCapability) {
				tool.Enabled = false
			},
			wantDecision: spec.MCPApprovalDecisionDenied,
			wantReason:   "tool is disabled or unsupported",
		},
		{
			name: "task required",
			mutate: func(_ *spec.MCPServerConfig, tool *spec.MCPToolCapability) {
				tool.TaskSupport = spec.MCPTaskSupportRequired
			},
			wantDecision: spec.MCPApprovalDecisionDenied,
			wantReason:   "task-required MCP tools are unsupported",
		},
		{
			name: "policy deny",
			mutate: func(server *spec.MCPServerConfig, _ *spec.MCPToolCapability) {
				server.DefaultPolicy.DefaultApprovalRule = deny
			},
			wantDecision: spec.MCPApprovalDecisionDenied,
			wantReason:   toolPolicyDeniesReason,
		},
		{
			name: "digest changed",
			mutate: func(server *spec.MCPServerConfig, _ *spec.MCPToolCapability) {
				server.ToolPolicies = map[string]spec.MCPToolPolicyOverride{
					"echo": {
						ToolName:       "echo",
						ExpectedDigest: "other-digest",
					},
				}
			},
			wantDecision: spec.MCPApprovalDecisionDenied,
			wantReason:   toolDigestChangedReason,
		},
		{
			name: "rule ask",
			mutate: func(server *spec.MCPServerConfig, _ *spec.MCPToolCapability) {
				server.DefaultPolicy.DefaultApprovalRule = ask
			},
			wantDecision: spec.MCPApprovalDecisionApprovalRequired,
			wantReason:   "approval rule is ask",
		},
		{
			name: "unknown risk requires approval",
			mutate: func(server *spec.MCPServerConfig, _ *spec.MCPToolCapability) {
				server.DefaultPolicy.DefaultApprovalRule = allow
				server.DefaultPolicy.RequireApprovalForUnknownRisk = true
			},
			wantDecision: spec.MCPApprovalDecisionApprovalRequired,
			wantReason:   "unknown-risk tool requires approval",
		},
		{
			name: "allowed",
			mutate: func(server *spec.MCPServerConfig, tool *spec.MCPToolCapability) {
				server.DefaultPolicy.DefaultApprovalRule = allow
				server.DefaultPolicy.RequireApprovalForUnknownRisk = false
				tool.InferredRisk = spec.MCPToolRiskRead
			},
			wantDecision: spec.MCPApprovalDecisionAllowed,
			wantReason:   policyAllowedReason,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := baseServer
			tool := baseTool
			tt.mutate(&server, &tool)

			got := Evaluate(EvaluationInput{
				Server: server,
				Tool:   tool,
				Req: spec.InvokeMCPToolRequestBody{
					Source:    spec.MCPInvocationSourceUser,
					ServerID:  server.ID,
					ToolName:  tool.ToolName,
					Arguments: map[string]any{"message": "hi"},
				},
			})

			if got.Decision != tt.wantDecision {
				t.Fatalf("Decision = %q, want %q", got.Decision, tt.wantDecision)
			}
			if got.Reason != tt.wantReason {
				t.Fatalf("Reason = %q, want %q", got.Reason, tt.wantReason)
			}
			if got.Summary == nil {
				t.Fatalf("Summary is nil")
			}
			if got.Summary.ServerID != server.ID || got.Summary.ToolName != tool.ToolName {
				t.Fatalf("Summary = %#v", got.Summary)
			}
		})
	}

	tests2 := []struct {
		name   string
		server spec.MCPServerConfig
		tool   spec.MCPToolCapability
		want   spec.MCPExecutionMode
	}{
		{
			name: "default manual",
			server: spec.MCPServerConfig{
				DefaultPolicy: spec.DefaultMCPServerPolicy(),
			},
			tool: spec.MCPToolCapability{ToolName: "echo"},
			want: manual,
		},
		{
			name: "default auto",
			server: spec.MCPServerConfig{
				DefaultPolicy: spec.MCPServerPolicy{
					DefaultExecutionMode: auto,
				},
			},
			tool: spec.MCPToolCapability{ToolName: "echo"},
			want: auto,
		},
		{
			name: "override",
			server: spec.MCPServerConfig{
				DefaultPolicy: spec.MCPServerPolicy{
					DefaultExecutionMode: manual,
				},
				ToolPolicies: map[string]spec.MCPToolPolicyOverride{
					"echo": {
						ToolName:      "echo",
						ExecutionMode: &auto,
					},
				},
			},
			tool: spec.MCPToolCapability{ToolName: "echo"},
			want: auto,
		},
	}

	for _, tt := range tests2 {
		t.Run("execution/"+tt.name, func(t *testing.T) {
			if got := ExecutionMode(tt.server, tt.tool); got != tt.want {
				t.Fatalf("ExecutionMode = %q, want %q", got, tt.want)
			}
		})
	}

	_ = deny
}

func TestNormalizeRawJSONAndApprovalManager(t *testing.T) {
	t.Run("normalize raw json", func(t *testing.T) {
		tests := []struct {
			name string
			in   spec.JSONRawString
			want spec.JSONRawString
		}{
			{name: "empty", in: "", want: "{}"},
			{name: "whitespace", in: "   ", want: "{}"},
			{name: "canonical object", in: `{"b":2,"a":1}`, want: `{"a":1,"b":2}`},
			{name: "invalid", in: "not-json", want: "not-json"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := normalizeRawJSON(tt.in); got != tt.want {
					t.Fatalf("normalizeRawJSON(%q) = %q, want %q", tt.in, got, tt.want)
				}
			})
		}
	})

	summary := spec.MCPApprovalSummary{
		ServerID:   "server",
		ToolName:   "tool",
		ToolDigest: "digest",
		Risk:       spec.MCPToolRiskWrite,
		Arguments:  spec.JSONRawString(`{"b":2,"a":1}`),
	}

	t.Run("allow once and consume", func(t *testing.T) {
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
			t.Fatalf("token = %#v", token)
		}
		if token.ApprovalID != id {
			t.Fatalf("token.ApprovalID = %q, want %q", token.ApprovalID, id)
		}

		expected := summary
		expected.Arguments = spec.JSONRawString(`{ "a" : 1, "b" : 2 }`)
		gotID, err := mgr.VerifyAndConsumeToken(t.Context(), token.Token, expected)
		if err != nil {
			t.Fatalf("VerifyAndConsumeToken: %v", err)
		}
		if gotID != id {
			t.Fatalf("VerifyAndConsumeToken id = %q, want %q", gotID, id)
		}
		if err := mgr.VerifyAndConsume(t.Context(), id, token.Token); err == nil {
			t.Fatalf("VerifyAndConsume succeeded twice, want error")
		}
	})

	t.Run("allow always caches decision", func(t *testing.T) {
		mgr := NewApprovalManager(time.Minute)

		id, err := mgr.Create(t.Context(), summary)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		token, err := mgr.Resolve(t.Context(), id, spec.MCPApprovalResolutionAllowAlways)
		if err != nil {
			t.Fatalf("Resolve allowAlways: %v", err)
		}
		if token == nil || token.Token == "" {
			t.Fatalf("token = %#v", token)
		}
		if got, ok := mgr.LookupDecision(summary); !ok || got != spec.MCPApprovalResolutionAllowAlways {
			t.Fatalf("LookupDecision = %q, %v; want allowAlways, true", got, ok)
		}
	})

	t.Run("deny always caches decision", func(t *testing.T) {
		mgr := NewApprovalManager(time.Minute)

		id, err := mgr.Create(t.Context(), summary)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		token, err := mgr.Resolve(t.Context(), id, spec.MCPApprovalResolutionDenyAlways)
		if err != nil {
			t.Fatalf("Resolve denyAlways: %v", err)
		}
		if token != nil {
			t.Fatalf("token = %#v, want nil", token)
		}
		if got, ok := mgr.LookupDecision(summary); !ok || got != spec.MCPApprovalResolutionDenyAlways {
			t.Fatalf("LookupDecision = %q, %v; want denyAlways, true", got, ok)
		}
	})

	t.Run("expired approval", func(t *testing.T) {
		mgr := NewApprovalManager(time.Minute)

		id, err := mgr.Create(t.Context(), summary)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		mgr.mu.Lock()
		mgr.pending[id].ExpiresAt = time.Now().UTC().Add(-time.Second)
		mgr.mu.Unlock()

		if _, err := mgr.Resolve(
			t.Context(),
			id,
			spec.MCPApprovalResolutionAllowOnce,
		); err == nil ||
			!strings.Contains(err.Error(), "approval expired") {
			t.Fatalf("Resolve expired error = %v, want approval expired", err)
		}
	})
}

func TestDiscoverySnapshotDigestAndPagination(t *testing.T) {
	snapA := spec.MCPDiscoverySnapshot{
		ServerID: "server",
		ServerCapabilities: &spec.MCPServerCapabilitiesSummary{
			Experimental: map[string]any{
				"a": 1,
				"b": 2,
			},
		},
		Tools: []spec.MCPToolCapability{
			{ServerID: "server", ToolName: "alpha", Digest: "1"},
		},
	}
	snapB := snapA
	snapB.ServerCapabilities = &spec.MCPServerCapabilitiesSummary{
		Experimental: map[string]any{
			"b": 2,
			"a": 1,
		},
	}

	digestA := computeDiscoverySnapshotDigest(snapA)
	digestB := computeDiscoverySnapshotDigest(snapB)
	if digestA == "" || digestA != digestB {
		t.Fatalf("digestA = %q, digestB = %q", digestA, digestB)
	}

	items := []int{1, 2, 3}
	tests := []struct {
		name      string
		token     string
		pageSize  int
		want      []int
		wantNext  bool
		wantError string
	}{
		{
			name:     "first page",
			pageSize: 2,
			want:     []int{1, 2},
			wantNext: true,
		},
		{
			name:     "second page",
			pageSize: 2,
			token: mustDiscoveryPageToken(t, spec.MCPDiscoveryPageToken{
				ServerID:       "server",
				SnapshotDigest: digestA,
				Kind:           discoveryPageKindTools,
				PageSize:       2,
				Index:          2,
			}),
			want:     []int{3},
			wantNext: false,
		},
		{
			name:      "bad token",
			token:     "bad-token",
			pageSize:  2,
			wantError: "bad pageToken",
		},
		{
			name: "stale token",
			token: mustDiscoveryPageToken(t, spec.MCPDiscoveryPageToken{
				ServerID:       "server",
				SnapshotDigest: "stale-digest",
				Kind:           discoveryPageKindTools,
				PageSize:       2,
				Index:          2,
			}),
			pageSize:  2,
			wantError: "stale pageToken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, next, err := paginateDiscoveryItems(
				"server",
				digestA,
				discoveryPageKindTools,
				items,
				tt.pageSize,
				tt.token,
			)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("err = %v, want substring %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("paginateDiscoveryItems: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("page = %#v, want %#v", got, tt.want)
			}
			if (next != nil) != tt.wantNext {
				t.Fatalf("next present = %v, want %v", next != nil, tt.wantNext)
			}
		})
	}
}

func TestToolBridgeApprovalFlow(t *testing.T) {
	allowPolicy := spec.MCPServerPolicy{
		DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
		DefaultExecutionMode: spec.MCPExecutionModeManual,
	}
	askPolicy := spec.MCPServerPolicy{
		DefaultApprovalRule:  spec.MCPApprovalRuleAsk,
		DefaultExecutionMode: spec.MCPExecutionModeManual,
	}

	tests := []struct {
		name         string
		policy       spec.MCPServerPolicy
		wantDecision spec.MCPApprovalDecision
		useToken     bool
		wantCallText string
	}{
		{
			name:         "allowed",
			policy:       allowPolicy,
			wantDecision: spec.MCPApprovalDecisionAllowed,
			useToken:     false,
			wantCallText: "called:echo:hello",
		},
		{
			name:         "approval required",
			policy:       askPolicy,
			wantDecision: spec.MCPApprovalDecisionApprovalRequired,
			useToken:     true,
			wantCallText: "called:echo:hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newRuntimeFixture(t, tt.policy)
			approvals := NewApprovalManager(time.Minute)
			bridge := NewToolBridge(fixture.mgr, approvals)

			listResp, err := fixture.mgr.ListTools(t.Context(), &spec.ListMCPServerToolsRequest{
				ServerID: fixture.serverID,
			})
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}
			if len(listResp.Body.Tools) != 1 {
				t.Fatalf("ListTools len = %d, want 1", len(listResp.Body.Tools))
			}
			tool := listResp.Body.Tools[0]

			req := &spec.InvokeMCPToolRequest{
				Body: &spec.InvokeMCPToolRequestBody{
					Source:           spec.MCPInvocationSourceUser,
					ServerID:         fixture.serverID,
					ToolName:         tool.ToolName,
					ProviderToolName: tool.ProviderToolName,
					ToolDigest:       tool.Digest,
					Arguments: map[string]any{
						"message": "hello",
					},
					ToolUseID: "use-1",
				},
			}

			eval, err := bridge.Evaluate(t.Context(), &spec.EvaluateMCPToolCallRequest{Body: req.Body})
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if eval.Body.Decision != tt.wantDecision {
				t.Fatalf("Evaluate.Decision = %q, want %q", eval.Body.Decision, tt.wantDecision)
			}

			if tt.useToken {
				if eval.Body.ApprovalID == "" {
					t.Fatalf("approval ID is empty")
				}
				token, err := approvals.Resolve(
					t.Context(),
					eval.Body.ApprovalID,
					spec.MCPApprovalResolutionAllowOnce,
				)
				if err != nil {
					t.Fatalf("Resolve: %v", err)
				}
				req.Body.ApprovalToken = token.Token
			} else if eval.Body.ApprovalID != "" {
				t.Fatalf("approval ID = %q, want empty", eval.Body.ApprovalID)
			}

			got, err := bridge.Invoke(t.Context(), req)
			if err != nil {
				t.Fatalf("Invoke: %v", err)
			}
			if got.Body == nil {
				t.Fatalf("Invoke body is nil")
			}
			if len(got.Body.Content) != 1 || got.Body.Content[0].Text != tt.wantCallText {
				t.Fatalf("Invoke content = %#v, want %q", got.Body.Content, tt.wantCallText)
			}
			if got.Body.Provenance.ServerID != fixture.serverID {
				t.Fatalf("Provenance.ServerID = %q, want %q", got.Body.Provenance.ServerID, fixture.serverID)
			}
			if got.Body.Provenance.ToolName != tool.ToolName {
				t.Fatalf("Provenance.ToolName = %q, want %q", got.Body.Provenance.ToolName, tool.ToolName)
			}
			if got.Body.Provenance.ToolDigest != tool.Digest {
				t.Fatalf("Provenance.ToolDigest = %q, want %q", got.Body.Provenance.ToolDigest, tool.Digest)
			}
			if got.Body.Provenance.ToolUseID != "use-1" {
				t.Fatalf("Provenance.ToolUseID = %q, want use-1", got.Body.Provenance.ToolUseID)
			}
			if tt.useToken {
				if got.Body.Provenance.ApprovalID == "" {
					t.Fatalf("Provenance.ApprovalID is empty")
				}
			} else if got.Body.Provenance.ApprovalID != "" {
				t.Fatalf("Provenance.ApprovalID = %q, want empty", got.Body.Provenance.ApprovalID)
			}
		})
	}
}

func TestRuntimeManagerRefreshFromNotification(t *testing.T) {
	allowPolicy := spec.MCPServerPolicy{
		DefaultApprovalRule:  spec.MCPApprovalRuleAllow,
		DefaultExecutionMode: spec.MCPExecutionModeManual,
	}

	tests := []struct {
		name string
		kind ClientNotificationKind
	}{
		{name: "tools", kind: ClientNotificationToolListChanged},
		{name: "resources", kind: ClientNotificationResourceListChanged},
		{name: "prompts", kind: ClientNotificationPromptListChanged},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initial := makeDiscoverySnapshot("server-1", "alpha")
			next := makeDiscoverySnapshot("server-1", "alpha", "beta")
			fixture := newRuntimeFixture(t, allowPolicy, initial, next)

			ctx := t.Context()

			fixture.mgr.OnClientNotification(ctx, ClientNotification{
				ServerID: fixture.serverID,
				Kind:     tt.kind,
			})

			fixture.mgr.mu.RLock()
			timer := fixture.mgr.notificationRefreshTimers[fixture.serverID]
			fixture.mgr.mu.RUnlock()
			if timer == nil {
				t.Fatalf("notification timer is nil")
			}
			timer.Stop()

			fixture.mgr.mu.Lock()
			delete(fixture.mgr.notificationRefreshTimers, fixture.serverID)
			fixture.mgr.mu.Unlock()

			fixture.mgr.refreshFromNotification(ctx, fixture.serverID, string(tt.kind), nil)

			fixture.session.mu.Lock()
			if fixture.session.discoverCalls != 2 {
				fixture.session.mu.Unlock()
				t.Fatalf("Discover calls = %d, want 2", fixture.session.discoverCalls)
			}
			fixture.session.mu.Unlock()

			statusResp, err := fixture.mgr.Status(ctx, &spec.GetMCPServerStatusRequest{ServerID: fixture.serverID})
			if err != nil {
				t.Fatalf("Status: %v", err)
			}
			if statusResp.Body.ToolCount != 2 {
				t.Fatalf("ToolCount = %d, want 2", statusResp.Body.ToolCount)
			}

			listResp, err := fixture.mgr.ListTools(ctx, &spec.ListMCPServerToolsRequest{ServerID: fixture.serverID})
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}
			if len(listResp.Body.Tools) != 2 {
				t.Fatalf("ListTools len = %d, want 2", len(listResp.Body.Tools))
			}
		})
	}
}

func mustDiscoveryPageToken(t *testing.T, tok spec.MCPDiscoveryPageToken) string {
	t.Helper()
	raw, err := encodeDiscoveryPageToken(tok)
	if err != nil {
		t.Fatalf("encodeDiscoveryPageToken: %v", err)
	}
	return raw
}

func TestNormalizeRawJSONStandalone(t *testing.T) {
	tests := []struct {
		name string
		in   spec.JSONRawString
		want spec.JSONRawString
	}{
		{name: "empty", in: "", want: "{}"},
		{name: "canonical", in: `{"b":2,"a":1}`, want: `{"a":1,"b":2}`},
		{name: "invalid", in: "not-json", want: "not-json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRawJSON(tt.in); got != tt.want {
				t.Fatalf("normalizeRawJSON(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestApprovalManagerLookupAndExpiry(t *testing.T) {
	mgr := NewApprovalManager(time.Minute)
	summary := spec.MCPApprovalSummary{
		ServerID:   "server",
		ToolName:   "tool",
		ToolDigest: "digest",
		Risk:       spec.MCPToolRiskWrite,
		Arguments:  spec.JSONRawString(`{"a":1}`),
	}

	id, err := mgr.Create(t.Context(), summary)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, ok := mgr.LookupDecision(summary); ok {
		t.Fatalf("LookupDecision unexpectedly found decision")
	}

	mgr.mu.Lock()
	mgr.pending[id].ExpiresAt = time.Now().UTC().Add(-time.Second)
	mgr.mu.Unlock()

	if _, err := mgr.Resolve(
		t.Context(),
		id,
		spec.MCPApprovalResolutionAllowOnce,
	); err == nil ||
		!errors.Is(err, spec.ErrMCPInvalidRequest) {
		t.Fatalf("Resolve expired error = %v, want invalid request", err)
	}
}
