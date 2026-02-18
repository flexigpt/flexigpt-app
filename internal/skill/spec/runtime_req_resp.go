package spec

import agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

// RuntimeSkillFilter mirrors the runtime prompt/list filters we expose over HTTP.
//
// IMPORTANT CONTRACT:
//   - allowSkills and all lifecycle-facing selectors are SkillDef (type/name/location)
//     which are the exact user-provided inputs registered into the runtime catalog.
//   - LLM-facing handles (SkillHandle) are NOT used for lifecycle.
type RuntimeSkillFilter struct {
	Types          []string                   `json:"types,omitempty"`
	NamePrefix     string                     `json:"namePrefix,omitempty"`
	LocationPrefix string                     `json:"locationPrefix,omitempty"`
	AllowSkills    []agentskillsSpec.SkillDef `json:"allowSkills,omitempty"`

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
type (
	GetSkillsPromptXMLRequest struct {
		Body *GetSkillsPromptXMLRequestBody
	}
	GetSkillsPromptXMLResponseBody struct {
		XML string `json:"xml"`
	}
)

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
