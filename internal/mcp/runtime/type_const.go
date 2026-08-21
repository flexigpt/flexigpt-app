package runtime

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
)

var (
	ErrMCPInvalidRuntimeRequest = errors.New("invalid mcp runtime request")
	ErrMCPRuntimeNotReady       = errors.New("mcp runtime is not ready")
	ErrMCPPolicyDenied          = errors.New("mcp policy denied request")
	ErrMCPApprovalNeeded        = errors.New("mcp approval required")
	ErrMCPStaleReference        = errors.New("mcp stale reference")
)

type ClientNotificationKind string

const (
	ClientNotificationToolListChanged     ClientNotificationKind = "toolsListChanged"
	ClientNotificationResourceListChanged ClientNotificationKind = "resourcesListChanged"
	ClientNotificationPromptListChanged   ClientNotificationKind = "promptsListChanged"
	ClientNotificationResourceUpdated     ClientNotificationKind = "resourceUpdated"
	ClientNotificationProgress            ClientNotificationKind = "progress"
)

type ClientNotification struct {
	Server artifact.ArtifactRef
	Kind   ClientNotificationKind

	ResourceURI string

	LoggerName   string
	LoggingLevel string
	LogData      any

	Progress float64
	Total    float64
	Message  string
}

type ClientNotificationSink interface {
	OnClientNotification(ctx context.Context, event ClientNotification)
}

type MCPServerStatus string

const (
	MCPServerStatusDisabled     MCPServerStatus = "disabled"
	MCPServerStatusDisconnected MCPServerStatus = "disconnected"
	MCPServerStatusConnecting   MCPServerStatus = "connecting"
	MCPServerStatusReady        MCPServerStatus = "ready"
	MCPServerStatusError        MCPServerStatus = "error"
)

type MCPImplementationInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type MCPServerCapabilitiesSummary struct {
	Tools                bool           `json:"tools,omitempty"`
	ToolsListChanged     bool           `json:"toolsListChanged,omitempty"`
	Resources            bool           `json:"resources,omitempty"`
	ResourcesSubscribe   bool           `json:"resourcesSubscribe,omitempty"`
	ResourcesListChanged bool           `json:"resourcesListChanged,omitempty"`
	Prompts              bool           `json:"prompts,omitempty"`
	PromptsListChanged   bool           `json:"promptsListChanged,omitempty"`
	Completions          bool           `json:"completions,omitempty"`
	Experimental         map[string]any `json:"experimental,omitempty"`
	Extensions           map[string]any `json:"extensions,omitempty"`
}

type MCPServerRuntimeSnapshot struct {
	Server     artifact.ArtifactRef     `json:"server"`
	Collection collection.CollectionRef `json:"collection"`
	Status     MCPServerStatus          `json:"status"`

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
	Server         artifact.ArtifactRef `json:"server"`
	SnapshotDigest string               `json:"dig"`
	Kind           string               `json:"k"`
	PageSize       int                  `json:"ps"`
	Index          int                  `json:"i"`
}

type MCPToolRisk string

const (
	MCPToolRiskUnknown     MCPToolRisk = "unknown"
	MCPToolRiskRead        MCPToolRisk = "read"
	MCPToolRiskWrite       MCPToolRisk = "write"
	MCPToolRiskDestructive MCPToolRisk = "destructive"
	MCPToolRiskOpenWorld   MCPToolRisk = "openWorld"
)

type MCPTaskSupport string

const (
	MCPTaskSupportForbidden MCPTaskSupport = "forbidden"
	MCPTaskSupportOptional  MCPTaskSupport = "optional"
	MCPTaskSupportRequired  MCPTaskSupport = "required"
)

type MCPToolAnnotations struct {
	DestructiveHint *bool  `json:"destructiveHint,omitempty"`
	IdempotentHint  bool   `json:"idempotentHint"`
	OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
	ReadOnlyHint    bool   `json:"readOnlyHint"`
	Title           string `json:"title,omitempty"`
}

type MCPToolAppInfo struct {
	ResourceURI string   `json:"resourceUri,omitempty"`
	Visibility  []string `json:"visibility,omitempty"`
}

type MCPToolCapability struct {
	Server           artifact.ArtifactRef `json:"server"`
	ToolName         string               `json:"toolName"`
	ProviderToolName string               `json:"providerToolName"`
	ChoiceID         string               `json:"choiceID"`

	Title       string `json:"title,omitempty"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`

	InputSchema  map[string]any `json:"inputSchema,omitempty"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`

	Annotations  *MCPToolAnnotations `json:"annotations,omitempty"`
	InferredRisk MCPToolRisk         `json:"inferredRisk"`

	ApprovalRule  policy.MCPApprovalRule  `json:"approvalRule"`
	ExecutionMode policy.MCPExecutionMode `json:"executionMode"`

	TaskSupport MCPTaskSupport `json:"taskSupport"`

	App *MCPToolAppInfo `json:"app,omitempty"`

	Digest  string `json:"digest"`
	Enabled bool   `json:"enabled"`
	Stale   bool   `json:"stale,omitempty"`
}

type MCPArgumentDefinition struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type MCPResourceRef struct {
	Server      artifact.ArtifactRef `json:"server"`
	URI         string               `json:"uri"`
	Name        string               `json:"name,omitempty"`
	Title       string               `json:"title,omitempty"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description,omitempty"`
	MimeType    string               `json:"mimeType,omitempty"`
	Size        int64                `json:"size,omitempty"`
	Annotations map[string]any       `json:"annotations,omitempty"`
	Digest      string               `json:"digest,omitempty"`
}

type MCPResourceTemplateRef struct {
	Server      artifact.ArtifactRef             `json:"server"`
	URITemplate string                           `json:"uriTemplate"`
	Name        string                           `json:"name,omitempty"`
	Title       string                           `json:"title,omitempty"`
	DisplayName string                           `json:"displayName"`
	Description string                           `json:"description,omitempty"`
	MimeType    string                           `json:"mimeType,omitempty"`
	Arguments   map[string]MCPArgumentDefinition `json:"arguments,omitempty"`
	Annotations map[string]any                   `json:"annotations,omitempty"`
	Digest      string                           `json:"digest,omitempty"`
}

type MCPPromptRef struct {
	Server      artifact.ArtifactRef             `json:"server"`
	PromptName  string                           `json:"promptName"`
	Title       string                           `json:"title,omitempty"`
	DisplayName string                           `json:"displayName"`
	Description string                           `json:"description,omitempty"`
	Arguments   map[string]MCPArgumentDefinition `json:"arguments,omitempty"`
	Digest      string                           `json:"digest,omitempty"`
}

type MCPDiscoverySnapshot struct {
	Server artifact.ArtifactRef `json:"server"`

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

type MCPApprovalDecision string

const (
	MCPApprovalDecisionAllowed          MCPApprovalDecision = "allowed"
	MCPApprovalDecisionDenied           MCPApprovalDecision = "denied"
	MCPApprovalDecisionApprovalRequired MCPApprovalDecision = "approvalRequired"
)

type MCPApprovalResolution string

const (
	MCPApprovalResolutionAllowOnce   MCPApprovalResolution = "allowOnce"
	MCPApprovalResolutionAllowAlways MCPApprovalResolution = "allowAlways"
	MCPApprovalResolutionDenyOnce    MCPApprovalResolution = "denyOnce"
	MCPApprovalResolutionDenyAlways  MCPApprovalResolution = "denyAlways"
)

type MCPApprovalSummary struct {
	Server            artifact.ArtifactRef   `json:"server"`
	ServerDisplayName string                 `json:"serverDisplayName,omitempty"`
	Source            MCPInvocationSource    `json:"source"`
	AppInstanceID     string                 `json:"appInstanceID,omitempty"`
	ToolName          string                 `json:"toolName"`
	ToolDigest        string                 `json:"toolDigest,omitempty"`
	Risk              MCPToolRisk            `json:"risk"`
	Arguments         jsonutil.JSONRawString `json:"arguments,omitempty"`
}

type MCPApprovalEvaluation struct {
	Decision   MCPApprovalDecision `json:"decision"`
	Reason     string              `json:"reason,omitempty"`
	ApprovalID string              `json:"approvalID,omitempty"`
	Summary    *MCPApprovalSummary `json:"summary,omitempty"`
}

// MCPApprovalResolutionResult is returned for every successful resolution.
// Token and ExpiresAt are populated only for allowOnce. Always resolutions
// are remembered in process memory until the associated MCP session ends.
type MCPApprovalResolutionResult struct {
	ApprovalID string                `json:"approvalID"`
	Resolution MCPApprovalResolution `json:"resolution"`
	Decision   MCPApprovalDecision   `json:"decision"`

	RememberedForSession bool   `json:"rememberedForSession,omitempty"`
	Token                string `json:"token,omitempty"`
	ExpiresAt            string `json:"expiresAt,omitempty"`
}
