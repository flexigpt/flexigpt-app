package spec

import (
	"errors"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
)

const (
	MCPHostName    = "FlexiGPT"
	MCPHostVersion = "dev"

	NotificationRefreshDebounce = 1 * time.Second
	MaxMCPServerPageSize        = 256
	DefaultMCPPageSize          = 25
)

var (
	ErrMCPInvalidRequest  = errors.New("invalid mcp request")
	ErrMCPRuntimeNotReady = errors.New("mcp runtime is not ready")
	ErrMCPAuthRequired    = errors.New("mcp authorization required")
	ErrMCPPolicyDenied    = errors.New("mcp policy denied request")
	ErrMCPApprovalNeeded  = errors.New("mcp approval required")
	ErrMCPStaleReference  = errors.New("mcp stale reference")
	ErrMCPServerDisabled  = errors.New("mcp server is disabled")
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

type (
	JSONRawString = string
)

type MCPTransportType string

const (
	MCPTransportStreamableHTTP MCPTransportType = "streamableHttp"
	MCPTransportStdio          MCPTransportType = "stdio"
)

type MCPTrustLevel string

const (
	MCPTrustLevelUntrusted MCPTrustLevel = "untrusted"
	MCPTrustLevelTrusted   MCPTrustLevel = "trusted"
)

type MCPServerStatus string

const (
	MCPServerStatusDisabled     MCPServerStatus = "disabled"
	MCPServerStatusDisconnected MCPServerStatus = "disconnected"
	MCPServerStatusConnecting   MCPServerStatus = "connecting"
	MCPServerStatusReady        MCPServerStatus = "ready"
	MCPServerStatusError        MCPServerStatus = "error"
)

type MCPHTTPAuthMode string

const (
	MCPHTTPAuthNone              MCPHTTPAuthMode = "none"
	MCPHTTPAuthAPIKey            MCPHTTPAuthMode = "apiKey"
	MCPHTTPAuthOAuth             MCPHTTPAuthMode = "oauth"
	MCPHTTPAuthClientCredentials MCPHTTPAuthMode = "clientCredentials"
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

type MCPApprovalRule string

const (
	MCPApprovalRuleAsk   MCPApprovalRule = "ask"
	MCPApprovalRuleAllow MCPApprovalRule = "allow"
	MCPApprovalRuleDeny  MCPApprovalRule = "deny"
)

type MCPExecutionMode string

const (
	MCPExecutionModeManual MCPExecutionMode = "manual"
	MCPExecutionModeAuto   MCPExecutionMode = "auto"
)

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

type MCPInvocationSource string

const (
	MCPInvocationSourceModel MCPInvocationSource = "model"
	MCPInvocationSourceUser  MCPInvocationSource = "user"
	MCPInvocationSourceApp   MCPInvocationSource = "app"
)

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

type MCPContentType string

const (
	MCPContentTypeText         MCPContentType = "text"
	MCPContentTypeImage        MCPContentType = "image"
	MCPContentTypeAudio        MCPContentType = "audio"
	MCPContentTypeResourceLink MCPContentType = "resource_link"
	MCPContentTypeResource     MCPContentType = "resource"
)

type MCPResourceContents struct {
	URI      string         `json:"uri"`
	MIMEType string         `json:"mimeType,omitempty"`
	Text     string         `json:"text,omitempty"`
	Blob     []byte         `json:"blob,omitempty"`
	Meta     map[string]any `json:"_meta,omitempty"`
}

type MCPIcon struct {
	Source   string   `json:"src"`
	MIMEType string   `json:"mimeType,omitempty"`
	Sizes    []string `json:"sizes,omitempty"`
	Theme    string   `json:"theme,omitempty"`
}

type MCPContent struct {
	Type MCPContentType `json:"type"`

	Text     string `json:"text,omitempty"`
	Data     []byte `json:"data,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`

	URI         string `json:"uri,omitempty"`
	Name        string `json:"name,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Size        *int64 `json:"size,omitempty"`

	Resource *MCPResourceContents `json:"resource,omitempty"`

	Annotations map[string]any `json:"annotations,omitempty"`
	Meta        map[string]any `json:"_meta,omitempty"`
	Icons       []MCPIcon      `json:"icons,omitempty"`
}

type MCPPromptMessage struct {
	Role    string     `json:"role"`
	Content MCPContent `json:"content"`
}

type MCPToolAnnotations struct {
	DestructiveHint *bool  `json:"destructiveHint,omitempty"`
	IdempotentHint  bool   `json:"idempotentHint"`
	OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
	ReadOnlyHint    bool   `json:"readOnlyHint"`
	Title           string `json:"title,omitempty"`
}

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

type MCPOAuthAuthorization struct {
	Server           artifact.ArtifactRef `json:"server"`
	AuthorizationURL string               `json:"authorizationURL"`
	ExpiresAt        string               `json:"expiresAt,omitempty"`
}
