package spec

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
)

type MCPSecretRef struct {
	ServerID MCPServerID   `json:"serverID"`
	Kind     MCPSecretKind `json:"kind"`
	Slot     string        `json:"slot,omitempty"`
}
