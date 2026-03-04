package spec

import (
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/attachment"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

const (
	ConversationFileExtension = "json"
	MaxPageSize               = 256
	DefaultPageSize           = 12
	ConversationSchemaVersion = "v1.0.0"
)

// ConversationMessage represents a single *turn* in the conversation.
//
// Examples:
//   - User turn: text + attachments + per-turn tool choices.
//   - Assistant turn: one or more messages, tool calls, tool outputs, reasoning, usage.
type ConversationMessage struct {
	ID        string                 `json:"id"`
	CreatedAt time.Time              `json:"createdAt"`
	Role      inferenceSpec.RoleEnum `json:"role"`
	Status    inferenceSpec.Status   `json:"status,omitzero"`

	// Default model configuration for this turn. This can be empty and would mean that model param have been carried
	// over from previous messages.
	ModelParam *inferenceSpec.ModelParam `json:"modelParam,omitempty"`

	// Canonical, lossless events for this turn, in the order they occurred.
	//
	// For a user turn, you typically have exactly one InputKindInputMessage
	// entry in Inputs, possibly preceded by earlier tool outputs, etc.
	// For an assistant turn, you typically have:
	//   - one or more OutputKindOutputMessage entries,
	//   - zero or more OutputKindReasoningMessage entries,
	//   - zero or more tool call / web-search events, etc.
	Inputs  []inferenceSpec.InputUnion  `json:"inputs,omitempty"`
	Outputs []inferenceSpec.OutputUnion `json:"outputs,omitempty"`

	// Tool choices that were *available* when this turn ran.
	// For the next completion, the app can choose to reuse or override these.
	ToolChoices      []inferenceSpec.ToolChoice `json:"toolChoices,omitempty"`
	ToolStoreChoices []toolSpec.ToolStoreChoice `json:"toolStoreChoices,omitempty"`
	// Attachments that backed this turn's user input (files, URLs, etc).
	// These are ref attachments; ContentBlock may or may not be hydrated.
	Attachments      []attachment.Attachment `json:"attachments,omitempty"`
	EnabledSkillRefs []skillSpec.SkillRef    `json:"enabledSkillRefs,omitempty"`
	ActiveSkillRefs  []skillSpec.SkillRef    `json:"activeSkillRefs,omitempty"`

	// Usage / error info from the model/provider for this turn
	// (usually attached to assistant turns).
	Usage        *inferenceSpec.Usage `json:"usage,omitempty"`
	Error        *inferenceSpec.Error `json:"error,omitempty"`
	DebugDetails any                  `json:"debugDetails,omitempty"`

	// Arbitrary UI/app metadata (tags, pinned, read state, etc.).
	Meta map[string]any `json:"meta,omitempty"`
}

// Conversation is the full chat, stored as a single JSON file.
type Conversation struct {
	SchemaVersion string    `json:"schemaVersion"`
	ID            string    `json:"id"`
	Title         string    `json:"title,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	ModifiedAt    time.Time `json:"modifiedAt"`

	// Ordered list of turns (messages) in the transcript.
	Messages []ConversationMessage `json:"messages"`

	// Extra metadata for your app (folders, tags, project, etc.).
	Meta map[string]any `json:"meta,omitempty"`
}
