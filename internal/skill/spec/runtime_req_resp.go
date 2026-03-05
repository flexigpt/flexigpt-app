package spec

import (
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

// JSONRawString mirrors the ToolRuntime API style; it's a raw JSON string.
type JSONRawString = string

type GetSkillsPromptXMLRequestBody struct {
	Filter *RuntimeSkillFilter `json:"filter,omitempty"`
}

type GetSkillsPromptXMLRequest struct {
	Body *GetSkillsPromptXMLRequestBody
}

type GetSkillsPromptXMLResponseBody struct {
	XML string `json:"xml"`
}

type GetSkillsPromptXMLResponse struct {
	Body *GetSkillsPromptXMLResponseBody
}

type CreateSkillSessionRequestBody struct {
	// Optional: close this previous session (best-effort) before creating a new one.
	CloseSessionID agentskillsSpec.SessionID `json:"closeSessionID,omitempty"`

	MaxActivePerSession int        `json:"maxActivePerSession,omitempty"`
	AllowSkillRefs      []SkillRef `json:"allowSkillRefs,omitempty"`  // enabled allowlist (store ids)
	ActiveSkillRefs     []SkillRef `json:"activeSkillRefs,omitempty"` // desired initial active (subset of allowlist)
}

// CreateSkillSessionRequest creates a runtime session using store identities (SkillRef),
// so the backend can translate refs -> runtime SkillDef (including embeddedfs hydration mapping).
type CreateSkillSessionRequest struct {
	Body *CreateSkillSessionRequestBody
}

type CreateSkillSessionResponseBody struct {
	SessionID       agentskillsSpec.SessionID `json:"sessionID"`
	ActiveSkillRefs []SkillRef                `json:"activeSkillRefs"`
}

type CreateSkillSessionResponse struct {
	Body *CreateSkillSessionResponseBody
}

type RuntimeSkillFilter struct {
	Types          []string   `json:"types,omitempty"`
	LocationPrefix string     `json:"locationPrefix,omitempty"`
	AllowSkillRefs []SkillRef `json:"allowSkillRefs,omitempty"`

	SessionID agentskillsSpec.SessionID `json:"sessionID,omitempty"`
	Activity  string                    `json:"activity,omitempty"` // any|active|inactive
}

type CloseSkillSessionRequest struct {
	SessionID agentskillsSpec.SessionID `path:"sessionID" required:"true"`
}
type CloseSkillSessionResponse struct{}

// RuntimeSkillListItem is the public runtime listing shape keyed by store identity (SkillRef).
// SkillDef is intentionally NOT exposed.
type RuntimeSkillListItem struct {
	SkillRef SkillRef `json:"skillRef"`

	// Copy of the runtime-facing identity fields (excluding Location) + runtime-indexed metadata.
	// These are read-only and exist only for display/debug.
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Digest      string `json:"digest,omitempty"`

	// Session-scoped.
	IsActive bool `json:"isActive,omitempty"`

	// Runtime/provider error (if any) for this skill record.
	ErrorMessage string `json:"errorMessage,omitempty"`
}

type ListRuntimeSkillsRequestBody struct {
	Filter *RuntimeSkillFilter `json:"filter,omitempty"`
}
type ListRuntimeSkillsRequest struct {
	Body *ListRuntimeSkillsRequestBody
}

type ListRuntimeSkillsResponseBody struct {
	Skills []RuntimeSkillListItem `json:"skills"`
}

type ListRuntimeSkillsResponse struct {
	Body *ListRuntimeSkillsResponseBody
}

type InvokeSkillToolRequestBody struct {
	SessionID agentskillsSpec.SessionID `json:"sessionID"      required:"true"`
	ToolName  string                    `json:"toolName"       required:"true"` // "skills.load" | "skills.unload" | "skills.readresource" | "skills.runscript"
	Args      JSONRawString             `json:"args,omitempty"`                 // JSON object string
}

type InvokeSkillToolRequest struct {
	Body *InvokeSkillToolRequestBody
}

type InvokeSkillToolResponseBody struct {
	Outputs      []llmtoolsSpec.ToolOutputUnion `json:"outputs,omitempty"`
	Meta         map[string]any                 `json:"meta,omitempty"`
	IsBuiltIn    bool                           `json:"isBuiltIn"`
	IsError      bool                           `json:"isError,omitempty"`
	ErrorMessage string                         `json:"errorMessage,omitempty"`
}

type InvokeSkillToolResponse struct {
	Body *InvokeSkillToolResponseBody
}
