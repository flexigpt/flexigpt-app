package apps

import (
	"errors"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func TestToolVisibilityDefaultsAndFilters(t *testing.T) {
	if !ToolVisibleToModel(nil) {
		t.Fatalf("nil app info should default visible to model")
	}
	if !ToolVisibleToApp(nil) {
		t.Fatalf("nil app info should default visible to app")
	}

	appOnly := &spec.MCPToolAppInfo{Visibility: []string{VisibilityApp}}
	if ToolVisibleToModel(appOnly) {
		t.Fatalf("app-only tool should not be model-visible")
	}
	if !ToolVisibleToApp(appOnly) {
		t.Fatalf("app-only tool should be app-visible")
	}

	modelOnly := &spec.MCPToolAppInfo{Visibility: []string{VisibilityModel}}
	if !ToolVisibleToModel(modelOnly) {
		t.Fatalf("model-only tool should be model-visible")
	}
	if ToolVisibleToApp(modelOnly) {
		t.Fatalf("model-only tool should not be app-visible")
	}
}

func TestEffectiveAppsPolicyDefaultIsSafe(t *testing.T) {
	got := EffectiveAppsPolicy(spec.MCPServerConfig{})

	if got.Enabled {
		t.Fatalf("default Apps policy should not enable apps")
	}
	if got.AllowAppInitiatedToolCalls {
		t.Fatalf("default Apps policy should not allow app-initiated tools")
	}
	if !got.RequireApprovalForOpenLink {
		t.Fatalf("default Apps policy should require open-link approval")
	}
	if !got.RequireApprovalForContextUpdates {
		t.Fatalf("default Apps policy should require context-update approval")
	}
}

func TestValidateAppToolInvocation(t *testing.T) {
	cfg := spec.MCPServerConfig{
		ID: "server-a",
		AppsPolicy: &spec.MCPAppsPolicy{
			Enabled:                    true,
			AllowAppInitiatedToolCalls: true,
		},
	}
	tool := spec.MCPToolCapability{
		ToolName: "refresh",
		App: &spec.MCPToolAppInfo{
			Visibility: []string{VisibilityApp},
		},
	}

	if err := ValidateAppToolInvocation(cfg, tool, "server-a"); err != nil {
		t.Fatalf("ValidateAppToolInvocation allowed case: %v", err)
	}

	if err := ValidateAppToolInvocation(cfg, tool, "server-b"); err == nil ||
		!errors.Is(err, spec.ErrMCPPolicyDenied) {
		t.Fatalf("cross-server validation err = %v, want policy denied", err)
	}

	cfg.AppsPolicy.AllowAppInitiatedToolCalls = false
	if err := ValidateAppToolInvocation(cfg, tool, "server-a"); err == nil ||
		!strings.Contains(err.Error(), "app-initiated") {
		t.Fatalf("tool-call-disabled err = %v, want app-initiated denial", err)
	}

	cfg.AppsPolicy.AllowAppInitiatedToolCalls = true
	tool.App.Visibility = []string{VisibilityModel}
	if err := ValidateAppToolInvocation(cfg, tool, "server-a"); err == nil ||
		!strings.Contains(err.Error(), "not visible to apps") {
		t.Fatalf("model-only err = %v, want app visibility denial", err)
	}
}

func TestDefaultSandboxCSP(t *testing.T) {
	csp := DefaultSandboxCSP()

	required := []string{
		"default-src 'none'",
		"connect-src 'none'",
		"frame-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'none'",
	}

	for _, want := range required {
		if !strings.Contains(csp, want) {
			t.Fatalf("DefaultSandboxCSP missing %q in %q", want, csp)
		}
	}
}
