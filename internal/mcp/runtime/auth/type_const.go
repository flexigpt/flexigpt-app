package auth

import (
	"errors"
	"time"

	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

var (
	ErrMCPAuthRequired       = errors.New("mcp authorization required")
	ErrMCPInvalidAuthRequest = errors.New("invalid mcp auth request")
)

type GrantType string

const (
	GrantTypeAuthorizationCode GrantType = "authorization_code"
	GrantTypeRefreshToken      GrantType = "refresh_token"
)

type MCPAuthState string

const (
	MCPAuthStateNotRequired       MCPAuthState = "notRequired"
	MCPAuthStateRequired          MCPAuthState = "required"
	MCPAuthStateAuthorized        MCPAuthState = "authorized"
	MCPAuthStateExpired           MCPAuthState = "expired"
	MCPAuthStateInsufficientScope MCPAuthState = "insufficientScope"
	MCPAuthStateError             MCPAuthState = "error"
)

type MCPAuthHealthState string

const (
	MCPAuthHealthStateNotRequired          MCPAuthHealthState = "notRequired"
	MCPAuthHealthStateNotConfigured        MCPAuthHealthState = "notConfigured"
	MCPAuthHealthStateAuthorizationNeeded  MCPAuthHealthState = "authorizationNeeded"
	MCPAuthHealthStateAuthorizationPending MCPAuthHealthState = "authorizationPending"
	MCPAuthHealthStateAuthorized           MCPAuthHealthState = "authorized"
	MCPAuthHealthStateExpired              MCPAuthHealthState = "expired"
	MCPAuthHealthStateInsufficientScope    MCPAuthHealthState = "insufficientScope"
	MCPAuthHealthStateError                MCPAuthHealthState = "error"
)

type MCPAuthSettings struct {
	// Empty means a random loopback port is used for the current process.
	OAuthLoopbackListenAddr string `json:"oauthLoopbackListenAddr,omitempty"`
}

type MCPAuthStatus struct {
	Server   mcpServer.ServerID        `json:"server"`
	AuthMode mcpServer.MCPHTTPAuthMode `json:"authMode"`
	State    MCPAuthState              `json:"state"`

	Scopes              []string   `json:"scopes,omitempty"`
	ExpiresAt           *time.Time `json:"expiresAt,omitempty"`
	LastError           string     `json:"lastError,omitempty"`
	AuthorizationServer string     `json:"authorizationServer,omitempty"`
	Resource            string     `json:"resource,omitempty"`
}

type MCPAuthHealth struct {
	Server   mcpServer.ServerID        `json:"server"`
	AuthMode mcpServer.MCPHTTPAuthMode `json:"authMode"`
	State    MCPAuthHealthState        `json:"state"`

	Configured bool `json:"configured"`

	Resource  string     `json:"resource,omitempty"`
	Scopes    []string   `json:"scopes,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	AuthorizationPending   bool   `json:"authorizationPending,omitempty"`
	AuthorizationURL       string `json:"authorizationURL,omitempty"`
	AuthorizationExpiresAt string `json:"authorizationExpiresAt,omitempty"`

	OAuthRedirectURL        string `json:"oauthRedirectURL,omitempty"`
	OAuthLoopbackListenAddr string `json:"oauthLoopbackListenAddr,omitempty"`
	OAuthLoopbackReady      *bool  `json:"oauthLoopbackReady,omitempty"`
	OAuthLoopbackError      string `json:"oauthLoopbackError,omitempty"`

	LastError string `json:"lastError,omitempty"`
}

type MCPOAuthAuthorization struct {
	Server           mcpServer.ServerID `json:"server"`
	AuthorizationURL string             `json:"authorizationURL"`
	ExpiresAt        string             `json:"expiresAt,omitempty"`
}
