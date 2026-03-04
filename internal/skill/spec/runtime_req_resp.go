package spec

import (
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

// JSONRawString mirrors the ToolRuntime API style; it's a raw JSON string.
type JSONRawString = string

// RuntimeSkillFilter mirrors the runtime prompt/list filters we expose over HTTP.
//
// IMPORTANT CONTRACT:
//   - allowSkills and all lifecycle-facing selectors are SkillDef (type/name/location)
//     which are the exact user-provided inputs registered into the runtime catalog.
//   - LLM-facing handles (SkillHandle) are NOT used for lifecycle.
type RuntimeSkillFilter struct {
	Types          []string   `json:"types,omitempty"`
	NamePrefix     string     `json:"namePrefix,omitempty"`
	LocationPrefix string     `json:"locationPrefix,omitempty"`
	AllowSkillRefs []SkillRef `json:"allowSkillRefs,omitempty"`

	SessionID agentskillsSpec.SessionID `json:"sessionID,omitempty"`
	Activity  string                    `json:"activity,omitempty"` // any|active|inactive
}

type CreateSkillSessionRequestBody struct {
	MaxActivePerSession int                        `json:"maxActivePerSession,omitempty"`
	ActiveSkills        []agentskillsSpec.SkillDef `json:"activeSkills,omitempty"`
}
type CreateSkillSessionRequest struct {
	Body *CreateSkillSessionRequestBody
}

type CreateSkillSessionResponseBody struct {
	SessionID    agentskillsSpec.SessionID  `json:"sessionID"`
	ActiveSkills []agentskillsSpec.SkillDef `json:"activeSkills"`
}
type CreateSkillSessionResponse struct {
	Body *CreateSkillSessionResponseBody
}

type CloseSkillSessionRequest struct {
	SessionID agentskillsSpec.SessionID `path:"sessionID" required:"true"`
}
type CloseSkillSessionResponse struct{}

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

type ListRuntimeSkillsRequestBody struct {
	Filter *RuntimeSkillFilter `json:"filter,omitempty"`
}
type (
	ListRuntimeSkillsRequest struct {
		Body *ListRuntimeSkillsRequestBody
	}
	ListRuntimeSkillsResponseBody struct {
		Skills []agentskillsSpec.SkillRecord `json:"skills"`
	}
)

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
	IsError      bool                           `json:"isError,omitzero"`
	ErrorMessage string                         `json:"errorMessage,omitzero"`
}

type InvokeSkillToolResponse struct {
	Body *InvokeSkillToolResponseBody
}
