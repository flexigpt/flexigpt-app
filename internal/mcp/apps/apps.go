package apps

import (
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	// AppExtensionID is the MCP extension identifier for MCP Apps.
	AppExtensionID = "io.modelcontextprotocol/ui"

	// AppMIMEType is the required MIME type for an MCP Apps UI resource.
	AppMIMEType = "text/html;profile=mcp-app"

	VisibilityModel = "model"
	VisibilityApp   = "app"
)

// IsAppMIMEType returns true if mime is a valid MCP Apps MIME type,
// tolerating whitespace and additional parameters after the profile.
func IsAppMIMEType(mime string) bool {
	norm := strings.ToLower(strings.TrimSpace(mime))
	norm = strings.ReplaceAll(norm, " ", "")
	return norm == AppMIMEType || strings.HasPrefix(norm, AppMIMEType+";")
}

// ToolVisibleToModel reports whether the tool can be exposed to the LLM.
// A nil/empty visibility list defaults to model+app, so unknown servers
// don't accidentally hide tools.
func ToolVisibleToModel(info *spec.MCPToolAppInfo) bool {
	if info == nil || len(info.Visibility) == 0 {
		return true
	}
	for _, v := range info.Visibility {
		if strings.EqualFold(strings.TrimSpace(v), VisibilityModel) {
			return true
		}
	}
	return false
}

// ValidateAppToolInvocation enforces the cross-server, visibility, and
// policy constraints for tool calls whose source is "app".
//
//   - serverID must match between the request and the app instance (the
//     caller is responsible for passing the app's serverID).
//   - server.appsPolicy.enabled must be true.
//   - server.appsPolicy.allowAppInitiatedToolCalls must be true.
//   - The tool must declare "app" visibility (or omit visibility entirely).
func ValidateAppToolInvocation(
	cfg spec.MCPServerConfig,
	tool spec.MCPToolCapability,
	appServerID spec.MCPServerID,
) error {
	policy := EffectiveAppsPolicy(cfg)
	if !policy.Enabled {
		return fmt.Errorf("%w: MCP Apps is not enabled for server %s", spec.ErrMCPPolicyDenied, cfg.ID)
	}
	if !policy.AllowAppInitiatedToolCalls {
		return fmt.Errorf("%w: app-initiated tool calls are not allowed for server %s", spec.ErrMCPPolicyDenied, cfg.ID)
	}
	if appServerID != "" && appServerID != cfg.ID {
		return fmt.Errorf(
			"%w: app on server %s cannot call tools on server %s",
			spec.ErrMCPPolicyDenied, appServerID, cfg.ID,
		)
	}
	if !ToolVisibleToApp(tool.App) {
		return fmt.Errorf("%w: tool %q is not visible to apps", spec.ErrMCPPolicyDenied, tool.ToolName)
	}
	return nil
}

// ToolVisibleToApp reports whether the tool may be called by an MCP App.
func ToolVisibleToApp(info *spec.MCPToolAppInfo) bool {
	if info == nil || len(info.Visibility) == 0 {
		return true
	}
	for _, v := range info.Visibility {
		if strings.EqualFold(strings.TrimSpace(v), VisibilityApp) {
			return true
		}
	}
	return false
}

// EffectiveAppsPolicy returns the configured policy or a safe default.
func EffectiveAppsPolicy(cfg spec.MCPServerConfig) spec.MCPAppsPolicy {
	if cfg.AppsPolicy != nil {
		return *cfg.AppsPolicy
	}
	return spec.MCPAppsPolicy{
		Enabled:                          false,
		AllowAppInitiatedToolCalls:       false,
		RequireApprovalForOpenLink:       true,
		RequireApprovalForContextUpdates: true,
	}
}

// DefaultSandboxCSP returns a restrictive CSP suitable for srcdoc iframes
// hosting untrusted MCP App HTML. With sandbox="allow-scripts" the iframe
// has a unique opaque origin, so 'self' refers to that opaque origin.
func DefaultSandboxCSP() string {
	return strings.Join([]string{
		"default-src 'none'",
		"script-src 'self' 'unsafe-inline'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"media-src 'self' data:",
		"font-src 'self' data:",
		"connect-src 'none'",
		"frame-src 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'none'",
	}, "; ")
}
