package secret

import (
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
)

var ErrNotFound = errors.New("MCP secret not found")

const SecretRefVersion = "mcpv1"

type MCPSecretKind string

const (
	//nolint:gosec // Enum val.
	MCPSecretKindStdioEnv MCPSecretKind = "stdioEnv"
	// MCPSecretKindOAuthClientCredentials stores a JSON object with OAuth client
	// credentials: {"clientID":"...","clientSecret":"..."}.
	// clientSecret is optional for authorization-code public clients using PKCE
	// and required for the client_credentials grant.
	//nolint:gosec // Enum val.
	MCPSecretKindOAuthClientCredentials MCPSecretKind = "oauthClientCredentials"

	// MCPSecretKindOAuthToken stores the app-managed OAuth authorization-code
	// token JSON for one MCP server. It is internal and is not user-editable via
	// the MCP server secret UI.
	MCPSecretKindOAuthToken MCPSecretKind = "oauthToken"

	MCPSecretKindHTTPHeader MCPSecretKind = "httpHeader"
)

func (kind MCPSecretKind) Validate() error {
	switch kind.normalized() {
	case MCPSecretKindStdioEnv,
		MCPSecretKindHTTPHeader,
		MCPSecretKindOAuthClientCredentials,
		MCPSecretKindOAuthToken:
		return nil
	default:
		return fmt.Errorf("secret ref kind %q is invalid", kind)
	}
}

type MCPSecretRef struct {
	Server artifact.ArtifactRef `json:"server"`
	Kind   MCPSecretKind        `json:"kind"`
	Slot   string               `json:"slot,omitempty"`
}

func (ref MCPSecretRef) Validate() error {
	return validateSecret(ref)
}
