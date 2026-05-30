package auth

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func MergeMCPAuthStatus(st spec.MCPAuthStatus, cfg spec.MCPServerConfig) spec.MCPAuthStatus {
	def := DefaultMCPAuthStatusFromConfig(cfg)
	if st.ServerID != "" && st.ServerID != def.ServerID {
		return def
	}
	if st.AuthMode != "" && st.AuthMode != def.AuthMode {
		return def
	}
	if st.Resource != "" && def.Resource != "" && st.Resource != def.Resource {
		return def
	}
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
	if def.AuthMode == spec.MCPHTTPAuthNone {
		st.State = def.State
		st.Scopes = nil
		st.ExpiresAt = nil
		st.LastError = ""
		st.AuthorizationServer = ""
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
		st.AuthMode = normalizeHTTPAuthMode(cfg.StreamableHTTP.AuthMode)
		st.Resource = strings.TrimSpace(cfg.StreamableHTTP.URL)
	}

	switch st.AuthMode {
	case spec.MCPHTTPAuthOAuth:
		st.State = spec.MCPAuthStateRequired
	case spec.MCPHTTPAuthClientCredentials:
		st.State = spec.MCPAuthStateRequired
	case spec.MCPHTTPAuthNone, "":
		st.State = spec.MCPAuthStateNotRequired
	default:
		st.State = spec.MCPAuthStateNotRequired
	}

	return st
}

func (m *AuthManager) SaveAuthStatus(ctx context.Context, st spec.MCPAuthStatus) error {
	if m == nil {
		return nil
	}
	if st.ServerID == "" {
		return fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	cloned := cloneAuthStatus(st)

	m.mu.Lock()
	if m.statuses == nil {
		m.statuses = map[spec.MCPServerID]spec.MCPAuthStatus{}
	}
	m.statuses[st.ServerID] = cloned
	m.mu.Unlock()

	return nil
}

func (m *AuthManager) GetAuthStatus(serverID spec.MCPServerID) (spec.MCPAuthStatus, bool) {
	if m == nil || serverID == "" {
		return spec.MCPAuthStatus{}, false
	}

	m.mu.RLock()
	st, ok := m.statuses[serverID]
	m.mu.RUnlock()

	if !ok {
		return spec.MCPAuthStatus{}, false
	}
	return cloneAuthStatus(st), true
}

func (m *AuthManager) ClearAuthStatus(serverID spec.MCPServerID) {
	if m == nil || serverID == "" {
		return
	}

	m.mu.Lock()
	delete(m.statuses, serverID)
	m.mu.Unlock()
}

func (m *AuthManager) ClearAuthStatuses() {
	if m == nil {
		return
	}

	m.mu.Lock()
	m.statuses = map[spec.MCPServerID]spec.MCPAuthStatus{}
	m.mu.Unlock()
}

func cloneAuthStatus(in spec.MCPAuthStatus) spec.MCPAuthStatus {
	out := in
	out.Scopes = slices.Clone(in.Scopes)
	if in.ExpiresAt != nil {
		t := *in.ExpiresAt
		out.ExpiresAt = &t
	}
	return out
}
