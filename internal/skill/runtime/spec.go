package runtime

import (
	"context"
	"errors"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
)

var (
	ErrInvalidRequest = errors.New("invalid Skill runtime request")
	ErrClosed         = errors.New("skill runtime is closed")
)

const maxSkillToolArgsBytes = 1 << 20

// CatalogID identifies one independently replaceable runtime catalog.
// It has no Artifact Store semantics.
type CatalogID string

// SkillRegistration is the runtime-owned registration input for one Skill.
type SkillRegistration struct {
	Definition provider.SkillDef `json:"definition"`
	Revision   string            `json:"revision,omitempty"`
}

// CatalogSource is owned by the runtime package and implemented by a data
// layer. CatalogID is opaque to runtime. Runtime only receives the Skill
// registrations needed to reconcile Agent Skills state.
type CatalogSource interface {
	Skills(
		ctx context.Context,
		catalogID CatalogID,
	) ([]SkillRegistration, error)
}

type SkillPromptFilter struct {
	Types          []string            `json:"types,omitempty"`
	NamePrefix     string              `json:"namePrefix,omitempty"`
	LocationPrefix string              `json:"locationPrefix,omitempty"`
	AllowSkills    []provider.SkillDef `json:"allowSkills,omitempty"`

	SessionID agentskillsRuntimeSpec.SessionID     `json:"sessionID,omitempty"`
	Activity  agentskillsRuntimeSpec.SkillActivity `json:"activity,omitempty"`
}

type SkillListFilter struct {
	Types          []string               `json:"types,omitempty"`
	NamePrefix     string                 `json:"namePrefix,omitempty"`
	LocationPrefix string                 `json:"locationPrefix,omitempty"`
	AllowSkills    []provider.SkillDef    `json:"allowSkills,omitempty"`
	Inserts        []document.SkillInsert `json:"inserts,omitempty"`

	SessionID agentskillsRuntimeSpec.SessionID     `json:"sessionID,omitempty"`
	Activity  agentskillsRuntimeSpec.SkillActivity `json:"activity,omitempty"`
}

type CreateSkillSessionRequestBody struct {
	CloseSessionID agentskillsRuntimeSpec.SessionID `json:"closeSessionID,omitempty"`

	MaxActivePerSession int `json:"maxActivePerSession,omitempty"`

	// Nil means unrestricted and causes no allowed-skills option to be
	// installed. A non-nil empty slice means no skill is allowed.
	AllowedSkills []provider.SkillDef `json:"allowedSkills,omitempty"`
	ActiveSkills  []provider.SkillDef `json:"activeSkills,omitempty"`
}

type CreateSkillSessionRequest struct {
	Body *CreateSkillSessionRequestBody
}

type CreateSkillSessionResponseBody struct {
	SessionID    agentskillsRuntimeSpec.SessionID `json:"sessionID"`
	ActiveSkills []provider.SkillDef              `json:"activeSkills"`
}

type CreateSkillSessionResponse struct {
	Body *CreateSkillSessionResponseBody
}

type CloseSkillSessionRequest struct {
	SessionID agentskillsRuntimeSpec.SessionID `path:"sessionID" required:"true"`
}

type CloseSkillSessionResponse struct{}

type GetSkillsPromptRequestBody struct {
	Filter *SkillPromptFilter `json:"filter,omitempty"`
}

type GetSkillsPromptRequest struct {
	Body *GetSkillsPromptRequestBody
}

type GetSkillsPromptResponseBody struct {
	Prompt string `json:"prompt"`
}

type GetSkillsPromptResponse struct {
	Body *GetSkillsPromptResponseBody
}

type ListSkillsRequestBody struct {
	Filter *SkillListFilter `json:"filter,omitempty"`
}

type ListSkillsRequest struct {
	Body *ListSkillsRequestBody
}

type ListSkillsResponseBody struct {
	Skills []agentskillsRuntimeSpec.SkillRecord `json:"skills"`
}

type ListSkillsResponse struct {
	Body *ListSkillsResponseBody
}

type RenderSkillRequestBody struct {
	Definition provider.SkillDef `json:"definition"          required:"true"`
	Arguments  map[string]string `json:"arguments,omitempty"`
}

type RenderSkillRequest struct {
	Body *RenderSkillRequestBody
}

type RenderSkillResponse struct {
	Body *agentskillsRuntime.RenderSkillOut
}

type InvokeSkillToolRequestBody struct {
	SessionID agentskillsRuntimeSpec.SessionID `json:"sessionID"      required:"true"`
	ToolName  string                           `json:"toolName"       required:"true"`
	Args      string                           `json:"args,omitempty"`
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

type SyncCatalogRequest struct {
	CatalogID CatalogID `json:"catalogID"`
}

type SyncCatalogResponse struct{}

type RemoveCatalogRequest struct {
	CatalogID CatalogID `json:"catalogID"`
}

type RemoveCatalogResponse struct{}
