package runtime

import (
	"fmt"
	"strings"

	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
)

const (
	// AppExtensionID is the MCP extension identifier for MCP Apps.
	AppExtensionID = "io.modelcontextprotocol/ui"

	// AppMIMEType is the required MIME type for an MCP Apps UI resource.
	AppMIMEType = "text/html;profile=mcp-app"

	VisibilityModel = "model"
	VisibilityApp   = "app"
)

type MCPAppModelContextUpdate struct {
	InstanceID string           `json:"instanceID,omitempty"`
	Server     mcpSpec.ServerID `json:"server"`

	ResourceURI string `json:"resourceUri,omitempty"`

	Content           []MCPContent `json:"content,omitempty"`
	StructuredContent any          `json:"structuredContent,omitempty"`

	UpdatedAt string `json:"updatedAt,omitempty"`
}

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
func ToolVisibleToModel(info *MCPToolAppInfo) bool {
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

// ValidateAppToolInvocation is the Artifact-backed MCP App
// authorization check. It is the target API used by the Artifact Store MCP
// runtime.
func ValidateAppToolInvocation(
	p mcpSpec.MCPAppsPolicy,
	tool MCPToolCapability,
	appServer mcpSpec.ServerID,
) error {
	if !p.Enabled {
		return fmt.Errorf(
			"%w: MCP Apps is not enabled for server %q",
			ErrMCPPolicyDenied,
			tool.Server,
		)
	}
	if !p.AllowAppInitiatedToolCalls {
		return fmt.Errorf(
			"%w: app-initiated MCP tool calls are not allowed",
			ErrMCPPolicyDenied,
		)
	}
	if err := tool.Server.Validate(); err != nil {
		return err
	}
	if appServer != tool.Server {
		return fmt.Errorf(
			"%w: MCP App cannot call a tool owned by another Server Artifact",
			ErrMCPPolicyDenied,
		)
	}
	if !ToolVisibleToApp(tool.App) {
		return fmt.Errorf(
			"%w: MCP tool %q is not visible to apps",
			ErrMCPPolicyDenied,
			tool.ToolName,
		)
	}
	return nil
}

// ToolVisibleToApp reports whether the tool may be called by an MCP App.
func ToolVisibleToApp(info *MCPToolAppInfo) bool {
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
