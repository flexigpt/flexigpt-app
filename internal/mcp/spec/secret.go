package spec

const SecretRefVersion = "mcpv1"

type MCPSecretKind string

const (
	MCPSecretKindStdioEnv          MCPSecretKind = "stdioEnv" //nolint:gosec // Enum val.
	MCPSecretKindHTTPHeader        MCPSecretKind = "httpHeader"
	MCPSecretKindHTTPToken         MCPSecretKind = "httpToken"
	MCPSecretKindOAuthClientSecret MCPSecretKind = "oauthClientSecret" //nolint:gosec // Enum val.
	// MCPSecretKindOAuthClientCredentials stores a JSON object: {"clientID":"...","clientSecret":"..."}.
	MCPSecretKindOAuthClientCredentials MCPSecretKind = "oauthClientCredentials" //nolint:gosec // Enum val.
)

type MCPSecretRef struct {
	ServerID MCPServerID   `json:"serverID"`
	Kind     MCPSecretKind `json:"kind"`
	Slot     string        `json:"slot,omitempty"`
}
