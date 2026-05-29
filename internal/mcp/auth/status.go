package auth

import (
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func MergeMCPAuthStatus(st spec.MCPAuthStatus, cfg spec.MCPServerConfig) spec.MCPAuthStatus {
	def := DefaultMCPAuthStatusFromConfig(cfg)
	if st.ServerID == "" {
		st.ServerID = def.ServerID
	}
	if st.AuthMode == "" {
		st.AuthMode = def.AuthMode
	}
	if st.Resource == "" {
		st.Resource = def.Resource
	}
	if st.State == "" {
		st.State = def.State
	}
	return st
}

func DefaultMCPAuthStatusFromConfig(cfg spec.MCPServerConfig) spec.MCPAuthStatus {
	st := spec.MCPAuthStatus{
		ServerID: cfg.ID,
		AuthMode: spec.MCPHTTPAuthNone,
		State:    spec.MCPAuthStateNotRequired,
	}

	if cfg.StreamableHTTP != nil {
		st.AuthMode = cfg.StreamableHTTP.AuthMode
		st.Resource = strings.TrimSpace(cfg.StreamableHTTP.URL)
	}

	switch st.AuthMode {
	case spec.MCPHTTPAuthOAuth:
		st.State = spec.MCPAuthStateRequired
	case spec.MCPHTTPAuthClientCredentials,
		spec.MCPHTTPAuthCustomBearer,
		spec.MCPHTTPAuthCustomHeaders:
		st.State = spec.MCPAuthStateAuthorized
	case spec.MCPHTTPAuthNone, "":
		st.State = spec.MCPAuthStateNotRequired
	default:
		st.State = spec.MCPAuthStateNotRequired
	}

	return st
}
