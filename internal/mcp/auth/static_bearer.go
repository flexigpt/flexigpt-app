package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"golang.org/x/oauth2"
)

func normalizeHTTPAuthMode(mode spec.MCPHTTPAuthMode) spec.MCPHTTPAuthMode {
	mode = spec.MCPHTTPAuthMode(strings.TrimSpace(string(mode)))
	if mode == "" {
		return spec.MCPHTTPAuthNone
	}
	return mode
}

func managedResolvedHTTPHeader(name string) bool {
	switch {
	case strings.EqualFold(name, "accept"):
		return true
	case strings.EqualFold(name, "content-type"):
		return true
	case strings.EqualFold(name, "mcp-protocol-version"):
		return true
	case strings.EqualFold(name, "mcp-session-id"):
		return true
	case strings.EqualFold(name, "mcp-method"):
		return true
	case strings.EqualFold(name, "mcp-name"):
		return true
	case strings.EqualFold(name, "last-event-id"):
		return true
	case strings.EqualFold(name, "content-length"):
		return true
	case strings.EqualFold(name, "host"):
		return true
	case strings.EqualFold(name, "connection"):
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type staticBearerOAuthHandler struct {
	token string
}

func (h *staticBearerOAuthHandler) TokenSource(context.Context) (oauth2.TokenSource, error) {
	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: h.token,
		TokenType:   "Bearer",
	}), nil
}

func (h *staticBearerOAuthHandler) Authorize(
	ctx context.Context,
	req *http.Request,
	resp *http.Response,
) error {
	drainAndClose(resp)
	if resp == nil {
		return fmt.Errorf("%w: bearer token rejected", spec.ErrMCPAuthRequired)
	}
	return fmt.Errorf("%w: bearer token rejected: %s", spec.ErrMCPAuthRequired, resp.Status)
}

func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}
