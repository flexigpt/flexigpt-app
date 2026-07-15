package auth

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type authStatusKey struct {
	BundleID bundleitemutils.BundleID
	ServerID spec.MCPServerID
}

type oauthAuthorizationPendingLister interface {
	Pending() []spec.MCPOAuthAuthorization
}

type oauthLoopbackInfoProvider interface {
	ListenAddr() string
	RedirectURL() string
}

// BuildAuthHealth returns the user-facing auth health for a server.
//
// Important storage model:
//   - OAuth authorization-code tokens are process-local. After app restart,
//     OAuth should show authorization-needed unless a new authorization is
//     pending in the loopback broker.
//   - PAT/API-key values are explicit user-provided secrets and are represented
//     by streamableHttp.secretHeaderRefs. Those are persistent config secrets.
//
// This function derives "configured" from the current server config and then
// overlays process-local runtime status and pending OAuth broker state.
func (m *AuthManager) BuildAuthHealth(ctx context.Context, cfg spec.MCPServerConfig) spec.MCPAuthHealth {
	def := DefaultMCPAuthStatusFromConfig(cfg)
	st := def

	if m != nil {
		if saved, ok := m.GetAuthStatus(cfg.BundleID, cfg.ID); ok {
			st = MergeMCPAuthStatus(saved, cfg)
		}
	}

	if def.AuthMode == spec.MCPHTTPAuthOAuth && st.State != spec.MCPAuthStateAuthorized && m != nil {
		if stored, ok := m.storedOAuthAuthStatus(ctx, def); ok {
			st = MergeMCPAuthStatus(stored, cfg)
		}
	}

	health := spec.MCPAuthHealth{
		BundleID:  cfg.BundleID,
		ServerID:  cfg.ID,
		AuthMode:  def.AuthMode,
		Resource:  def.Resource,
		Scopes:    slices.Clone(st.Scopes),
		ExpiresAt: cloneTimePtr(st.ExpiresAt),
		LastError: st.LastError,
	}

	if m != nil {
		health.OAuthRedirectURL = m.oauthRedirectURL
		if info, ok := m.oauthBroker.(oauthLoopbackInfoProvider); ok {
			health.OAuthLoopbackListenAddr = info.ListenAddr()
			if health.OAuthRedirectURL == "" {
				health.OAuthRedirectURL = info.RedirectURL()
			}
		}
	}

	configured := authHealthConfigured(cfg, m, def.AuthMode, st)
	state := authHealthStateFromStatus(st)

	if def.AuthMode == spec.MCPHTTPAuthNone {
		configured = true
		state = spec.MCPAuthHealthStateNotRequired
	}

	if pending := m.findPendingOAuthAuthorization(cfg.BundleID, cfg.ID); pending != nil {
		configured = true
		state = spec.MCPAuthHealthStateAuthorizationPending
		health.AuthorizationPending = true
		health.AuthorizationURL = pending.AuthorizationURL
		health.AuthorizationExpiresAt = pending.ExpiresAt
		health.LastError = ""
	}

	if !configured && def.AuthMode != spec.MCPHTTPAuthNone {
		state = spec.MCPAuthHealthStateNotConfigured
	}

	switch def.AuthMode {
	case spec.MCPHTTPAuthAPIKey, spec.MCPHTTPAuthClientCredentials:
		// These modes are non-interactive. "Configured" means the required
		// secret reference exists. A runtime failure may still override this
		// with Error via saved process-local status.
		if configured && state == spec.MCPAuthHealthStateAuthorizationNeeded {
			state = spec.MCPAuthHealthStateAuthorized
		}
	case spec.MCPHTTPAuthOAuth:
		if configured && state == "" {
			state = spec.MCPAuthHealthStateAuthorizationNeeded
		}
	default:
	}

	health.Configured = configured
	health.State = state
	return health
}

func (m *AuthManager) storedOAuthAuthStatus(
	ctx context.Context,
	base spec.MCPAuthStatus,
) (spec.MCPAuthStatus, bool) {
	if m == nil || m.oauthTokenStore == nil {
		return spec.MCPAuthStatus{}, false
	}
	tok, err := m.oauthTokenStore.LoadOAuthToken(ctx, base)
	if err != nil || tok == nil {
		return spec.MCPAuthStatus{}, false
	}
	st := authStatusFromToken(base, tok)
	if st.State == spec.MCPAuthStateAuthorized {
		_ = m.SaveAuthStatus(context.WithoutCancel(ctx), st)
		return st, true
	}

	// Do not leave an expired or malformed token in storage. The next request
	// should perform a fresh authorization flow instead of repeatedly trying a
	// known-bad bearer token.
	_ = m.oauthTokenStore.DeleteOAuthToken(context.WithoutCancel(ctx), base)
	if st.State != spec.MCPAuthStateExpired {
		st.State = spec.MCPAuthStateExpired
	}
	if st.LastError == "" {
		st.LastError = "Persisted OAuth token is expired or invalid"
	}
	_ = m.SaveAuthStatus(context.WithoutCancel(ctx), st)
	return st, true
}

func (m *AuthManager) findPendingOAuthAuthorization(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) *spec.MCPOAuthAuthorization {
	if m == nil || m.oauthBroker == nil || bundleID == "" || serverID == "" {
		return nil
	}
	lister, ok := m.oauthBroker.(oauthAuthorizationPendingLister)
	if !ok {
		return nil
	}
	for _, pending := range lister.Pending() {
		if pending.BundleID == bundleID && pending.ServerID == serverID {
			cp := pending
			return &cp
		}
	}
	return nil
}

func MergeMCPAuthStatus(st spec.MCPAuthStatus, cfg spec.MCPServerConfig) spec.MCPAuthStatus {
	def := DefaultMCPAuthStatusFromConfig(cfg)
	if st.BundleID != "" && st.BundleID != def.BundleID {
		return def
	}
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
	if st.BundleID == "" {
		st.BundleID = def.BundleID
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
	if st.State == spec.MCPAuthStateAuthorized {
		if !authStatusCanBeAuthorized(cfg, def) {
			return def
		}
		if authStatusExpiredByClock(st) {
			st.State = spec.MCPAuthStateExpired
			if st.LastError == "" {
				st.LastError = "OAuth token is expired"
			}
		}
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
		BundleID: cfg.BundleID,
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
	case spec.MCPHTTPAuthAPIKey:
		st.State = spec.MCPAuthStateRequired
	case spec.MCPHTTPAuthNone, "":
		st.State = spec.MCPAuthStateNotRequired
	default:
		st.State = spec.MCPAuthStateError
		st.LastError = errStrUnsupportedAuthMode
	}

	return st
}

func authStatusCanBeAuthorized(cfg spec.MCPServerConfig, def spec.MCPAuthStatus) bool {
	switch def.AuthMode {
	case spec.MCPHTTPAuthAPIKey:
		return cfg.StreamableHTTP != nil && len(cfg.StreamableHTTP.SecretHeaderRefs) > 0
	case spec.MCPHTTPAuthClientCredentials:
		return cfg.StreamableHTTP != nil &&
			strings.TrimSpace(cfg.StreamableHTTP.ClientCredentialRef) != ""
	default:
		return true
	}
}

func authStatusExpiredByClock(st spec.MCPAuthStatus) bool {
	if st.ExpiresAt == nil || st.ExpiresAt.IsZero() {
		return false
	}
	return !time.Now().UTC().Before(st.ExpiresAt.UTC())
}

func authHealthConfigured(
	cfg spec.MCPServerConfig,
	m *AuthManager,
	mode spec.MCPHTTPAuthMode,
	st spec.MCPAuthStatus,
) bool {
	switch mode {
	case spec.MCPHTTPAuthNone:
		return true
	case spec.MCPHTTPAuthAPIKey:
		return cfg.StreamableHTTP != nil && len(cfg.StreamableHTTP.SecretHeaderRefs) > 0
	case spec.MCPHTTPAuthClientCredentials:
		return cfg.StreamableHTTP != nil && strings.TrimSpace(cfg.StreamableHTTP.ClientCredentialRef) != ""
	case spec.MCPHTTPAuthOAuth:
		return st.State == spec.MCPAuthStateAuthorized ||
			(m != nil && m.oauthBroker != nil && strings.TrimSpace(m.oauthRedirectURL) != "")
	default:
		return false
	}
}

func authHealthStateFromStatus(st spec.MCPAuthStatus) spec.MCPAuthHealthState {
	switch st.State {
	case spec.MCPAuthStateNotRequired:
		return spec.MCPAuthHealthStateNotRequired
	case spec.MCPAuthStateAuthorized:
		return spec.MCPAuthHealthStateAuthorized
	case spec.MCPAuthStateExpired:
		return spec.MCPAuthHealthStateExpired
	case spec.MCPAuthStateInsufficientScope:
		return spec.MCPAuthHealthStateInsufficientScope
	case spec.MCPAuthStateError:
		return spec.MCPAuthHealthStateError
	case spec.MCPAuthStateRequired, "":
		return spec.MCPAuthHealthStateAuthorizationNeeded
	default:
		return spec.MCPAuthHealthStateError
	}
}

func (m *AuthManager) SaveAuthStatus(ctx context.Context, st spec.MCPAuthStatus) error {
	if m == nil {
		return nil
	}
	if st.BundleID == "" {
		return fmt.Errorf("%w: bundleID required", spec.ErrMCPInvalidRequest)
	}
	if st.ServerID == "" {
		return fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	cloned := cloneAuthStatus(st)
	key := authStatusKey{BundleID: st.BundleID, ServerID: st.ServerID}

	m.mu.Lock()
	if m.statuses == nil {
		m.statuses = map[authStatusKey]spec.MCPAuthStatus{}
	}
	m.statuses[key] = cloned

	m.mu.Unlock()

	return nil
}

func (m *AuthManager) GetAuthStatus(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
) (spec.MCPAuthStatus, bool) {
	if m == nil || bundleID == "" || serverID == "" {
		return spec.MCPAuthStatus{}, false
	}

	m.mu.RLock()
	st, ok := m.statuses[authStatusKey{BundleID: bundleID, ServerID: serverID}]

	m.mu.RUnlock()

	if !ok {
		return spec.MCPAuthStatus{}, false
	}
	return cloneAuthStatus(st), true
}

func (m *AuthManager) ClearAuthStatus(bundleID bundleitemutils.BundleID, serverID spec.MCPServerID) {
	if m == nil || bundleID == "" || serverID == "" {
		return
	}

	m.mu.Lock()
	delete(m.statuses, authStatusKey{BundleID: bundleID, ServerID: serverID})

	m.mu.Unlock()
}

func (m *AuthManager) ClearAuthStatuses() {
	if m == nil {
		return
	}

	m.mu.Lock()
	m.statuses = map[authStatusKey]spec.MCPAuthStatus{}
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

func cloneTimePtr(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	t := *in
	return &t
}
