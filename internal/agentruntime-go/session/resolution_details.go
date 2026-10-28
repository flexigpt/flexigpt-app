package session

// WorkspaceContextOutcome records the exact send-time outcome of one selected
// Workspace Context item. It is a domain result, not a frontend-derived label.
type WorkspaceContextOutcome string

const (
	WorkspaceContextOutcomeIncluded          WorkspaceContextOutcome = "included"
	WorkspaceContextOutcomePartiallyIncluded WorkspaceContextOutcome = "partiallyIncluded"
	WorkspaceContextOutcomeNotIncluded       WorkspaceContextOutcome = "notIncluded"
	WorkspaceContextOutcomeNotAllowed        WorkspaceContextOutcome = "notAllowed"
	WorkspaceContextOutcomeUnavailable       WorkspaceContextOutcome = "unavailable"
)

// MCPSelectionAvailability records whether one selected MCP capability was
// actually available at operation preparation time.
type MCPSelectionAvailability string

const (
	MCPSelectionAvailabilityAvailable   MCPSelectionAvailability = "available"
	MCPSelectionAvailabilityUnavailable MCPSelectionAvailability = "unavailable"
)

// SkillResolutionState belongs to a Skill resolution item for one operation.
// It must not be inferred from current Skill catalog state.
type SkillResolutionState struct {
	Enabled          bool `json:"enabled"`
	Active           bool `json:"active"`
	SessionAvailable bool `json:"sessionAvailable"`
}

// WorkspaceContextResolution belongs only to a workspaceContext resolution
// item. DiagnosticSummary is deliberately user-safe and never mirrors the
// internal Reason field.
type WorkspaceContextResolution struct {
	Outcome           WorkspaceContextOutcome `json:"outcome"`
	DiagnosticSummary string                  `json:"diagnosticSummary,omitempty"`
}

// MCPResolutionDetails belongs only to an MCP server resolution item.
type MCPResolutionDetails struct {
	InstructionsIncluded bool                            `json:"instructionsIncluded"`
	Tools                []MCPToolResolution             `json:"tools,omitempty"`
	Resources            []MCPResourceResolution         `json:"resources,omitempty"`
	ResourceTemplates    []MCPResourceTemplateResolution `json:"resourceTemplates,omitempty"`
	Prompts              []MCPPromptResolution           `json:"prompts,omitempty"`
}

type MCPToolResolution struct {
	DisplayName  string                   `json:"displayName"`
	Description  string                   `json:"description,omitempty"`
	Availability MCPSelectionAvailability `json:"availability"`
}

type MCPResourceResolution struct {
	DisplayName  string                   `json:"displayName"`
	Description  string                   `json:"description,omitempty"`
	Availability MCPSelectionAvailability `json:"availability"`
}

type MCPResourceTemplateResolution struct {
	DisplayName  string                   `json:"displayName"`
	Description  string                   `json:"description,omitempty"`
	Availability MCPSelectionAvailability `json:"availability"`
}

type MCPPromptResolution struct {
	DisplayName  string                   `json:"displayName"`
	Description  string                   `json:"description,omitempty"`
	Availability MCPSelectionAvailability `json:"availability"`
}
