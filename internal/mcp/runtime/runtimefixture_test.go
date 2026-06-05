package runtime

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store"
)

type runtimeFixture struct {
	mgr      *MCPRuntimeManager
	session  *fakeClientSession
	bundleID bundleitemutils.BundleID
	serverID spec.MCPServerID
}

func newRuntimeFixture(
	t *testing.T,
	policy spec.MCPServerPolicy,
	snapshots ...spec.MCPDiscoverySnapshot,
) *runtimeFixture {
	t.Helper()

	dir := t.TempDir()
	st, err := store.NewMCPStore(t.Context(), dir)
	if err != nil {
		t.Fatalf("NewMCPStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	bundleID := bundleitemutils.BundleID("bundle-a")
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

	serverID := spec.MCPServerID("server-1")

	if len(snapshots) == 0 {
		snapshots = []spec.MCPDiscoverySnapshot{
			makeDiscoverySnapshot(serverID, "echo"),
		}
	}

	if _, err := st.PutMCPServer(t.Context(), &spec.PutMCPServerRequest{
		BundleID: bundleID,
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
	mgr := NewMCPRuntimeManager(st, nil, factory)
	if _, err := mgr.Connect(
		t.Context(),
		&spec.ConnectMCPServerRequest{BundleID: bundleID, ServerID: serverID},
	); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	t.Cleanup(func() { _ = mgr.Close(t.Context()) })

	return &runtimeFixture{
		mgr:      mgr,
		session:  session,
		bundleID: bundleID,
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
