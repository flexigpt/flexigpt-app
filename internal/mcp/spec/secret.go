package spec

const SecretRefVersion = "mcpv1"

type MCPSecretKind string

const (
	//nolint:gosec // Enum val.
	MCPSecretKindStdioEnv MCPSecretKind = "stdioEnv"
	// MCPSecretKindOAuthClientCredentials stores a JSON object: {"clientID":"...","clientSecret":"..."}.
	//nolint:gosec // Enum val.
	MCPSecretKindOAuthClientCredentials MCPSecretKind = "oauthClientCredentials"
)

type MCPSecretRef struct {
	ServerID MCPServerID   `json:"serverID"`
	Kind     MCPSecretKind `json:"kind"`
	Slot     string        `json:"slot,omitempty"`
}
