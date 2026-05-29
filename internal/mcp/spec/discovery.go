package spec

type MCPServerRuntimeSnapshot struct {
	ServerID MCPServerID     `json:"serverID"`
	Status   MCPServerStatus `json:"status"`

	NegotiatedProtocolVersion string                        `json:"negotiatedProtocolVersion,omitempty"`
	ServerInfo                *MCPImplementationInfo        `json:"serverInfo,omitempty"`
	ServerCapabilities        *MCPServerCapabilitiesSummary `json:"serverCapabilities,omitempty"`
	Instructions              string                        `json:"instructions,omitempty"`

	LastError       string `json:"lastError,omitempty"`
	LastConnectedAt string `json:"lastConnectedAt,omitempty"`
	LastSyncedAt    string `json:"lastSyncedAt,omitempty"`

	ToolCount             int `json:"toolCount"`
	ResourceCount         int `json:"resourceCount"`
	ResourceTemplateCount int `json:"resourceTemplateCount"`
	PromptCount           int `json:"promptCount"`

	SnapshotDigest string `json:"snapshotDigest,omitempty"`
}

// MCPDiscoveryPageToken is an opaque cursor for paginating cached discovery
// snapshots. It is encoded as base64(JSON) and should not be interpreted by
// callers.
type MCPDiscoveryPageToken struct {
	ServerID       MCPServerID `json:"sid"`
	SnapshotDigest string      `json:"dig"`
	Kind           string      `json:"k"`
	PageSize       int         `json:"ps"`
	Index          int         `json:"i"`
}

type MCPToolAppInfo struct {
	ResourceURI string   `json:"resourceUri,omitempty"`
	Visibility  []string `json:"visibility,omitempty"`
}

type MCPToolCapability struct {
	ServerID         MCPServerID `json:"serverID"`
	ToolName         string      `json:"toolName"`
	ProviderToolName string      `json:"providerToolName"`
	ChoiceID         string      `json:"choiceID"`

	Title       string `json:"title,omitempty"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`

	InputSchema  map[string]any `json:"inputSchema,omitempty"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`

	Annotations  *MCPToolAnnotations `json:"annotations,omitempty"`
	InferredRisk MCPToolRisk         `json:"inferredRisk"`

	ApprovalRule  MCPApprovalRule  `json:"approvalRule"`
	ExecutionMode MCPExecutionMode `json:"executionMode"`

	TaskSupport MCPTaskSupport `json:"taskSupport"`

	App *MCPToolAppInfo `json:"app,omitempty"`

	Digest  string `json:"digest"`
	Enabled bool   `json:"enabled"`
	Stale   bool   `json:"stale,omitempty"`
}

type MCPResourceRef struct {
	ServerID    MCPServerID    `json:"serverID"`
	URI         string         `json:"uri"`
	Name        string         `json:"name,omitempty"`
	Title       string         `json:"title,omitempty"`
	DisplayName string         `json:"displayName"`
	Description string         `json:"description,omitempty"`
	MimeType    string         `json:"mimeType,omitempty"`
	Size        int64          `json:"size,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
	Digest      string         `json:"digest,omitempty"`
}

type MCPResourceTemplateRef struct {
	ServerID    MCPServerID       `json:"serverID"`
	URITemplate string            `json:"uriTemplate"`
	Name        string            `json:"name,omitempty"`
	Title       string            `json:"title,omitempty"`
	DisplayName string            `json:"displayName"`
	Description string            `json:"description,omitempty"`
	MimeType    string            `json:"mimeType,omitempty"`
	Arguments   map[string]string `json:"arguments,omitempty"`
	Annotations map[string]any    `json:"annotations,omitempty"`
	Digest      string            `json:"digest,omitempty"`
}

type MCPPromptRef struct {
	ServerID    MCPServerID       `json:"serverID"`
	PromptName  string            `json:"promptName"`
	Title       string            `json:"title,omitempty"`
	DisplayName string            `json:"displayName"`
	Description string            `json:"description,omitempty"`
	Arguments   map[string]string `json:"arguments,omitempty"`
	Digest      string            `json:"digest,omitempty"`
}

type MCPDiscoverySnapshot struct {
	ServerID MCPServerID `json:"serverID"`

	NegotiatedProtocolVersion string                        `json:"negotiatedProtocolVersion,omitempty"`
	ServerInfo                *MCPImplementationInfo        `json:"serverInfo,omitempty"`
	ServerCapabilities        *MCPServerCapabilitiesSummary `json:"serverCapabilities,omitempty"`
	Instructions              string                        `json:"instructions,omitempty"`

	Tools             []MCPToolCapability      `json:"tools,omitempty"`
	Resources         []MCPResourceRef         `json:"resources,omitempty"`
	ResourceTemplates []MCPResourceTemplateRef `json:"resourceTemplates,omitempty"`
	Prompts           []MCPPromptRef           `json:"prompts,omitempty"`

	Digest   string `json:"digest,omitempty"`
	SyncedAt string `json:"syncedAt,omitempty"`
}
