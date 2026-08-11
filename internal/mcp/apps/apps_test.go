package apps

import (
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
