package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	defaultOAuthLoopbackTTL  = 5 * time.Minute
	defaultOAuthCallbackPath = "/mcp/oauth/callback"
)

type oauthLoopbackResult struct {
	Result *OAuthAuthorizationResult
	Err    error
}
type pendingOAuthAuthorization struct {
	ID               string
	ServerID         spec.MCPServerID
	AuthorizationURL string
	State            string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	ResultCh         chan oauthLoopbackResult
}

type OAuthLoopbackBrokerOptions struct {
	TTL          time.Duration
	CallbackPath string
	Logger       *slog.Logger
}

type OAuthLoopbackBroker struct {
	mu sync.Mutex

	ttl          time.Duration
	callbackPath string
	redirectURL  string
	logger       *slog.Logger

	server   *http.Server
	listener net.Listener

	pendingByServer map[spec.MCPServerID]*pendingOAuthAuthorization
	pendingByState  map[string]*pendingOAuthAuthorization
}

func NewOAuthLoopbackBroker(ctx context.Context, opts *OAuthLoopbackBrokerOptions) (*OAuthLoopbackBroker, error) {
	var options OAuthLoopbackBrokerOptions
	if opts != nil {
		options = *opts
	}

	ttl := options.TTL
	if ttl <= 0 {
		ttl = defaultOAuthLoopbackTTL
	}

	callbackPath := strings.TrimSpace(options.CallbackPath)
	if callbackPath == "" {
		callbackPath = defaultOAuthCallbackPath
	}
	if !strings.HasPrefix(callbackPath, "/") {
		callbackPath = "/" + callbackPath
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	lc := &net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start OAuth loopback listener: %w", err)
	}

	b := &OAuthLoopbackBroker{
		ttl:             ttl,
		callbackPath:    callbackPath,
		logger:          logger,
		listener:        ln,
		pendingByServer: map[spec.MCPServerID]*pendingOAuthAuthorization{},
		pendingByState:  map[string]*pendingOAuthAuthorization{},
	}

	redirect := url.URL{
		Scheme: "http",
		Host:   ln.Addr().String(),
		Path:   callbackPath,
	}
	b.redirectURL = redirect.String()

	b.server = &http.Server{
		Handler:           b,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := b.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Warn("mcp oauth loopback server stopped unexpectedly", "err", err)
		}
	}()

	return b, nil
}

func (b *OAuthLoopbackBroker) RedirectURL() string {
	if b == nil {
		return ""
	}
	return b.redirectURL
}

func (b *OAuthLoopbackBroker) FetchAuthorizationCode(
	ctx context.Context,
	req OAuthAuthorizationRequest,
) (*OAuthAuthorizationResult, error) {
	if b == nil {
		return nil, fmt.Errorf("%w: OAuth loopback broker is not configured", spec.ErrMCPAuthRequired)
	}
	if req.ServerID == "" {
		return nil, fmt.Errorf("%w: OAuth serverID required", spec.ErrMCPInvalidRequest)
	}
	if strings.TrimSpace(req.AuthorizationURL) == "" {
		return nil, fmt.Errorf("%w: OAuth authorization URL required", spec.ErrMCPAuthRequired)
	}

	state, err := authorizationState(req.AuthorizationURL)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	id := rand.Text()

	p := &pendingOAuthAuthorization{
		ID:               id,
		ServerID:         req.ServerID,
		AuthorizationURL: req.AuthorizationURL,
		State:            state,
		CreatedAt:        now,
		ExpiresAt:        now.Add(b.ttl),
		ResultCh:         make(chan oauthLoopbackResult, 1),
	}

	b.mu.Lock()
	b.purgeExpiredLocked(now)

	if old := b.pendingByServer[req.ServerID]; old != nil {
		b.completeLocked(old, nil, fmt.Errorf("%w: OAuth authorization superseded", spec.ErrMCPAuthRequired))
	}

	b.pendingByServer[p.ServerID] = p
	b.pendingByState[p.State] = p
	b.mu.Unlock()

	timer := time.NewTimer(time.Until(p.ExpiresAt))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		b.removeIfCurrent(p)
		return nil, ctx.Err()

	case <-timer.C:
		b.removeIfCurrent(p)
		return nil, fmt.Errorf("%w: OAuth authorization expired", spec.ErrMCPAuthRequired)

	case result := <-p.ResultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Result, nil
	}
}

func (b *OAuthLoopbackBroker) Pending() []spec.MCPOAuthAuthorization {
	if b == nil {
		return nil
	}

	now := time.Now().UTC()

	b.mu.Lock()
	defer b.mu.Unlock()

	b.purgeExpiredLocked(now)

	out := make([]spec.MCPOAuthAuthorization, 0, len(b.pendingByServer))
	for _, p := range b.pendingByServer {
		out = append(out, spec.MCPOAuthAuthorization{
			ServerID:         p.ServerID,
			AuthorizationURL: p.AuthorizationURL,
			ExpiresAt:        p.ExpiresAt.Format(time.RFC3339Nano),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ServerID == out[j].ServerID {
			return out[i].ExpiresAt < out[j].ExpiresAt
		}
		return out[i].ServerID < out[j].ServerID
	})

	return out
}

func (b *OAuthLoopbackBroker) Cancel(serverID spec.MCPServerID) bool {
	if b == nil || serverID == "" {
		return false
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	p := b.pendingByServer[serverID]
	if p == nil {
		return false
	}

	b.completeLocked(p, nil, fmt.Errorf("%w: OAuth authorization cancelled", spec.ErrMCPAuthRequired))
	return true
}

func (b *OAuthLoopbackBroker) Close() error {
	if b == nil {
		return nil
	}

	b.mu.Lock()
	for _, p := range b.pendingByServer {
		b.completeLocked(p, nil, fmt.Errorf("%w: OAuth broker closed", spec.ErrMCPAuthRequired))
	}
	b.mu.Unlock()

	if b.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return b.server.Shutdown(ctx)
}

func (b *OAuthLoopbackBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != b.callbackPath {
		http.NotFound(w, r)
		return
	}

	q := r.URL.Query()
	state := q.Get("state")
	if state == "" {
		http.Error(w, "missing OAuth state", http.StatusBadRequest)
		return
	}

	b.mu.Lock()
	p := b.pendingByState[state]
	if p == nil {
		b.mu.Unlock()
		http.Error(w, "unknown or expired OAuth state", http.StatusBadRequest)
		return
	}

	if authErr := q.Get("error"); authErr != "" {
		desc := q.Get("error_description")
		err := fmt.Errorf("%w: authorization server returned %s%s",
			spec.ErrMCPAuthRequired,
			authErr,
			formatOAuthErrorDescription(desc),
		)
		b.completeLocked(p, nil, err)
		b.mu.Unlock()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("FlexiGPT authorization was not completed. You may close this window."))
		return
	}

	code := q.Get("code")
	if code == "" {
		b.mu.Unlock()
		http.Error(w, "missing OAuth authorization code", http.StatusBadRequest)
		return
	}

	result := &OAuthAuthorizationResult{
		Code:  code,
		State: state,
		Iss:   q.Get("iss"),
	}

	b.completeLocked(p, result, nil)
	b.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("FlexiGPT received the authorization response. You may close this window."))
}

func (b *OAuthLoopbackBroker) removeIfCurrent(p *pendingOAuthAuthorization) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if current := b.pendingByServer[p.ServerID]; current == p {
		b.removeLocked(p)
	}
}

func (b *OAuthLoopbackBroker) purgeExpiredLocked(now time.Time) {
	for _, p := range b.pendingByServer {
		if now.After(p.ExpiresAt) {
			b.completeLocked(p, nil, fmt.Errorf("%w: OAuth authorization expired", spec.ErrMCPAuthRequired))
		}
	}
}

func (b *OAuthLoopbackBroker) completeLocked(
	p *pendingOAuthAuthorization,
	result *OAuthAuthorizationResult,
	err error,
) {
	if p == nil {
		return
	}

	b.removeLocked(p)

	select {
	case p.ResultCh <- oauthLoopbackResult{Result: result, Err: err}:
	default:
	}
}

func (b *OAuthLoopbackBroker) removeLocked(p *pendingOAuthAuthorization) {
	delete(b.pendingByServer, p.ServerID)
	delete(b.pendingByState, p.State)
}

func authorizationState(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: invalid OAuth authorization URL", spec.ErrMCPAuthRequired)
	}

	state := u.Query().Get("state")
	if state == "" {
		return "", fmt.Errorf("%w: OAuth authorization URL missing state", spec.ErrMCPAuthRequired)
	}
	return state, nil
}

func formatOAuthErrorDescription(desc string) string {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return ""
	}
	return ": " + desc
}
