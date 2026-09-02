package aggregate

import (
	"errors"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

var (
	ErrInvalidRequest = errors.New("invalid Skill runtime request")
	ErrSkillNotFound  = errors.New("runtime Skill not found")
)

const maxSkillToolArgsBytes = 1 << 20

type RuntimeSkillFilter struct {
	Types          []string               `json:"types,omitempty"`
	Inserts        []document.SkillInsert `json:"inserts,omitempty"`
	NamePrefix     string                 `json:"namePrefix,omitempty"`
	LocationPrefix string                 `json:"locationPrefix,omitempty"`
	AllowArtifacts []artifact.ArtifactRef `json:"allowArtifacts,omitempty"`

	SessionID agentskillsRuntimeSpec.SessionID     `json:"sessionID,omitempty"`
	Activity  agentskillsRuntimeSpec.SkillActivity `json:"activity,omitempty"`
}

type GetSkillsPromptRequestBody struct {
	Filter *RuntimeSkillFilter `json:"filter,omitempty"`
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

type CreateSkillSessionRequestBody struct {
	// Optional: close this previous session (best-effort) before creating a new one.
	CloseSessionID agentskillsRuntimeSpec.SessionID `json:"closeSessionID,omitempty"`

	MaxActivePerSession int                    `json:"maxActivePerSession,omitempty"`
	AllowArtifacts      []artifact.ArtifactRef `json:"allowArtifacts,omitempty"`
	ActiveArtifacts     []artifact.ArtifactRef `json:"activeArtifacts,omitempty"`
}

// CreateSkillSessionRequest creates a session using Artifact Store identities.
type CreateSkillSessionRequest struct {
	Body *CreateSkillSessionRequestBody
}

type CreateSkillSessionResponseBody struct {
	SessionID       agentskillsRuntimeSpec.SessionID `json:"sessionID"`
	ActiveArtifacts []artifact.ArtifactRef           `json:"activeArtifacts"`
}

type CreateSkillSessionResponse struct {
	Body *CreateSkillSessionResponseBody
}

type CloseSkillSessionRequest struct {
	SessionID agentskillsRuntimeSpec.SessionID `path:"sessionID" required:"true"`
}
type CloseSkillSessionResponse struct{}

type RenderSkillRequestBody struct {
	Artifact  artifact.ArtifactRef `json:"artifact"            required:"true"`
	Arguments map[string]string    `json:"arguments,omitempty"`
}

type RenderSkillRequest struct {
	Body *RenderSkillRequestBody
}

type RenderSkillResponseBody struct {
	Text string `json:"text"`

	Insert document.SkillInsert `json:"insert"`

	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	DisplayName string `json:"displayName,omitempty"`

	// SourceTags are tags parsed from SKILL.md frontmatter.
	SourceTags []string `json:"sourceTags,omitempty"`

	Resources provider.SkillResourceInfo `json:"resources"`

	Arguments        []document.SkillArgument `json:"arguments,omitempty"`
	AppliedArguments map[string]string        `json:"appliedArguments,omitempty"`
	RawFrontmatter   map[string]any           `json:"rawFrontmatter,omitempty"`
	Warnings         []string                 `json:"warnings,omitempty"`
}

type RenderSkillResponse struct {
	Body *RenderSkillResponseBody
}

// RuntimeSkillListItem is keyed only by durable Artifact Store identity.
// SkillDef is intentionally process-local and never exposed.
type RuntimeSkillListItem struct {
	SkillRef artifact.ArtifactRef `json:"skillRef"`

	// Copy of the runtime-facing identity fields (excluding Location) + runtime-indexed metadata.
	// These are read-only and exist only for display/debug.
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Digest      string `json:"digest,omitempty"`

	Insert    document.SkillInsert     `json:"insert,omitempty"`
	Arguments []document.SkillArgument `json:"arguments,omitempty"`
	// SourceTags are tags parsed from SKILL.md frontmatter.
	SourceTags     []string                   `json:"sourceTags,omitempty"`
	Resources      provider.SkillResourceInfo `json:"resources"`
	RawFrontmatter map[string]any             `json:"rawFrontmatter,omitempty"`
	Warnings       []string                   `json:"warnings,omitempty"`

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
	SessionID agentskillsRuntimeSpec.SessionID `json:"sessionID"      required:"true"`
	ToolName  string                           `json:"toolName"       required:"true"` // "skills-load" | "skills-unload" | "skills-readresource" | "skills-runscript"
	Args      jsonutil.JSONRawString           `json:"args,omitempty"`                 // JSON object string
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

// ArtifactSkillSummary is the runtime-derived summary used by durable
// selection validators. It contains no filesystem path and no process-local
// SkillDef identity.
type ArtifactSkillSummary struct {
	Artifact     artifact.ArtifactRef
	IsEnabled    bool
	Insert       document.SkillInsert
	HasArguments bool
	HasResources bool
}
