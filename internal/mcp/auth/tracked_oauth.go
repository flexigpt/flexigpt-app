package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

type trackedOAuthHandler struct {
	inner  mcpAuth.OAuthHandler
	sink   AuthStatusSink
	status spec.MCPAuthStatus
}

func (h *trackedOAuthHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	return h.inner.TokenSource(ctx)
}

func (h *trackedOAuthHandler) Authorize(
	ctx context.Context,
	req *http.Request,
	resp *http.Response,
) error {
	err := h.inner.Authorize(ctx, req, resp)
	if err != nil {
		h.publish(ctx, authStatusFromHTTPFailure(h.status, resp, err))
		return err
	}

	st := h.status
	st.State = spec.MCPAuthStateAuthorized
	st.LastError = ""
	h.publish(ctx, st)
	return nil
}

func (h *trackedOAuthHandler) publish(ctx context.Context, st spec.MCPAuthStatus) {
	if h == nil || h.sink == nil {
		return
	}

	_ = h.sink.SaveAuthStatus(context.WithoutCancel(ctx), st)
}

func authStatusFromHTTPFailure(
	base spec.MCPAuthStatus,
	resp *http.Response,
	err error,
) spec.MCPAuthStatus {
	st := base
	st.State = spec.MCPAuthStateError
	if err != nil {
		st.LastError = err.Error()
	}
	if resp == nil {
		return st
	}

	challengeErr, scopes := bearerChallengeValues(resp.Header.Values("WWW-Authenticate"))
	if len(scopes) > 0 {
		st.Scopes = scopes
	}
	switch {
	case resp.StatusCode == http.StatusForbidden && challengeErr == "insufficient_scope":
		st.State = spec.MCPAuthStateInsufficientScope
	case resp.StatusCode == http.StatusUnauthorized:
		st.State = spec.MCPAuthStateRequired
	case resp.StatusCode == http.StatusForbidden:
		st.State = spec.MCPAuthStateError
	}
	return st
}

func bearerChallengeValues(headers []string) (challengeErr string, scopes []string) {
	challenges, err := oauthex.ParseWWWAuthenticate(headers)
	if err != nil {
		return "", nil
	}
	for _, c := range challenges {
		if c.Scheme != "bearer" {
			continue
		}
		if challengeErr == "" {
			challengeErr = c.Params["error"]
		}
		if scope := c.Params["scope"]; scope != "" {
			scopes = strings.Fields(scope)
		}
	}
	return challengeErr, scopes
}
