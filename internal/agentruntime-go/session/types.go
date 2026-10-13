package session

import (
	"encoding/json"
	"time"
)

type (
	SessionID     string
	RunID         string
	CommandID     string
	OperationID   string
	ToolCallID    string
	ToolAttemptID string
	ToolResultID  string
	WaitID        string
	Revision      uint64
)

type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleDeveloper MessageRole = "developer"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

type RunState string

const (
	RunStateQueued     RunState = "queued"
	RunStateRunning    RunState = "running"
	RunStateWaiting    RunState = "waiting"
	RunStatePaused     RunState = "paused"
	RunStateCancelling RunState = "cancelling"
	RunStateCompleted  RunState = "completed"
	RunStateFailed     RunState = "failed"
	RunStateCancelled  RunState = "cancelled"
)

func (s RunState) Terminal() bool {
	switch s {
	case RunStateCompleted, RunStateFailed, RunStateCancelled:
		return true
	default:
		return false
	}
}

type OperationState string

const (
	OperationStateQueued      OperationState = "queued"
	OperationStatePreparing   OperationState = "preparing"
	OperationStateRunning     OperationState = "running"
	OperationStateCompleted   OperationState = "completed"
	OperationStateFailed      OperationState = "failed"
	OperationStateInterrupted OperationState = "interrupted"
	OperationStateCancelled   OperationState = "cancelled"
)

func (s OperationState) Terminal() bool {
	switch s {
	case OperationStateCompleted,
		OperationStateFailed,
		OperationStateInterrupted,
		OperationStateCancelled:
		return true
	default:
		return false
	}
}

type ToolCallState string

const (
	ToolCallStateProposed         ToolCallState = "proposed"
	ToolCallStatePending          ToolCallState = "pending"
	ToolCallStateEvaluationQueued ToolCallState = "evaluationQueued"
	ToolCallStateEvaluating       ToolCallState = "evaluating"
	ToolCallStateReady            ToolCallState = "ready"
	ToolCallStateExecutionQueued  ToolCallState = "executionQueued"
	ToolCallStateWaiting          ToolCallState = "waiting"
	ToolCallStateRunning          ToolCallState = "running"
	ToolCallStateResolved         ToolCallState = "resolved"
	ToolCallStateFailed           ToolCallState = "failed"
	ToolCallStateCancelled        ToolCallState = "cancelled"
	ToolCallStateObserved         ToolCallState = "observed"
)

type ToolAttemptState string

const (
	ToolAttemptStateQueued    ToolAttemptState = "queued"
	ToolAttemptStateRunning   ToolAttemptState = "running"
	ToolAttemptStateCompleted ToolAttemptState = "completed"
	ToolAttemptStateFailed    ToolAttemptState = "failed"
	ToolAttemptStateCancelled ToolAttemptState = "cancelled"
	ToolAttemptStateUncertain ToolAttemptState = "uncertain"
)

type ToolResultKind string

const (
	ToolResultKindSuccess   ToolResultKind = "success"
	ToolResultKindError     ToolResultKind = "error"
	ToolResultKindSkipped   ToolResultKind = "skipped"
	ToolResultKindUncertain ToolResultKind = "uncertain"
)

type ToolBindingKind string

const (
	ToolBindingKindGo         ToolBindingKind = "go"
	ToolBindingKindMCP        ToolBindingKind = "mcp"
	ToolBindingKindSkill      ToolBindingKind = "skill"
	ToolBindingKindProvider   ToolBindingKind = "provider"
	ToolBindingKindUnresolved ToolBindingKind = "unresolved"
)

type WaitKind string

const (
	WaitKindApproval            WaitKind = "approval"
	WaitKindManualToolExecution WaitKind = "manualToolExecution"
	WaitKindUserInput           WaitKind = "userInput"
	WaitKindConfiguration       WaitKind = "configuration"
	WaitKindAuthentication      WaitKind = "authentication"
	WaitKindContinuationReview  WaitKind = "continuationReview"
	WaitKindRecovery            WaitKind = "recovery"
)

type WaitStatus string

const (
	WaitStatusPending   WaitStatus = "pending"
	WaitStatusResolving WaitStatus = "resolving"
	WaitStatusResolved  WaitStatus = "resolved"
	WaitStatusExpired   WaitStatus = "expired"
	WaitStatusCancelled WaitStatus = "cancelled"
	WaitStatusInvalid   WaitStatus = "invalid"
)

// WaitResolution is a Session-owned, generic human-intervention decision.
//
// Concrete adapters translate these values to provider or policy-specific
// decisions. The Session core never accepts external approval-token or
// MCP-resolution strings directly.
type WaitResolution string

const (
	WaitResolutionApproveOnce   WaitResolution = "approveOnce"
	WaitResolutionApproveAlways WaitResolution = "approveAlways"
	WaitResolutionDenyOnce      WaitResolution = "denyOnce"
	WaitResolutionDenyAlways    WaitResolution = "denyAlways"
	WaitResolutionContinue      WaitResolution = "continue"
	WaitResolutionCancel        WaitResolution = "cancel"
)

type AutomationHoldScope string

const (
	AutomationHoldOneBatch   AutomationHoldScope = "oneBatch"
	AutomationHoldIndefinite AutomationHoldScope = "indefinite"
)

type AttachmentReferenceKind string

const (
	AttachmentReferenceFile      AttachmentReferenceKind = "file"
	AttachmentReferenceDirectory AttachmentReferenceKind = "directory"
	AttachmentReferenceImage     AttachmentReferenceKind = "image"
	AttachmentReferenceURL       AttachmentReferenceKind = "url"
	AttachmentReferenceGeneric   AttachmentReferenceKind = "generic"
)

type ArtifactReference struct {
	RootID     string `json:"rootID"`
	ArtifactID string `json:"artifactID"`
}

type CapabilityTarget struct {
	Form             string             `json:"form"`
	Type             string             `json:"type"`
	Name             string             `json:"name"`
	Provenance       string             `json:"provenance"`
	Artifact         *ArtifactReference `json:"artifact,omitempty"`
	ProviderIdentity string             `json:"providerIdentity,omitempty"`
	ProviderLocalID  string             `json:"providerLocalID,omitempty"`
	Evidence         string             `json:"evidence,omitempty"`
}

type LocalAttachmentReference struct {
	Path         string `json:"path,omitempty"`
	OriginalPath string `json:"originalPath,omitempty"`
	Name         string `json:"name,omitempty"`
	Exists       bool   `json:"exists"`
	IsDirectory  bool   `json:"isDirectory,omitempty"`
	Size         int64  `json:"size,omitempty"`
	OriginalSize int64  `json:"originalSize,omitempty"`
	ModifiedAt   string `json:"modifiedAt,omitempty"`
	OriginalMod  string `json:"originalModifiedAt,omitempty"`
	MIMEType     string `json:"mimeType,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	Format       string `json:"format,omitempty"`
}

type URLAttachmentReference struct {
	URL        string `json:"url"`
	Normalized string `json:"normalized,omitempty"`
	Original   string `json:"original,omitempty"`
}

type GenericAttachmentReference struct {
	Handle         string `json:"handle"`
	OriginalHandle string `json:"originalHandle,omitempty"`
}

type DirectoryAttachmentReference struct {
	Path             string `json:"path"`
	OriginalPath     string `json:"originalPath,omitempty"`
	Name             string `json:"name,omitempty"`
	ScannedFileCount int    `json:"scannedFileCount,omitempty"`
	HasMore          bool   `json:"hasMore,omitempty"`

	// Expanded is populated exactly once during submission preparation.
	// Historical replay uses these immutable child references rather than
	// walking the current directory contents again.
	Expanded []AttachmentReference `json:"expanded,omitempty"`
}

type AttachmentReference struct {
	Kind           AttachmentReferenceKind       `json:"kind"`
	Label          string                        `json:"label,omitempty"`
	Mode           string                        `json:"mode,omitempty"`
	AvailableModes []string                      `json:"availableModes,omitempty"`
	File           *LocalAttachmentReference     `json:"file,omitempty"`
	Image          *LocalAttachmentReference     `json:"image,omitempty"`
	URL            *URLAttachmentReference       `json:"url,omitempty"`
	Generic        *GenericAttachmentReference   `json:"generic,omitempty"`
	Directory      *DirectoryAttachmentReference `json:"directory,omitempty"`
}

type HistorySelection struct {
	All                      bool `json:"all,omitempty"`
	IncludePreviousUserTurns int  `json:"includePreviousUserTurns,omitempty"`
}

type ModelToolSelection struct {
	ChoiceID           string           `json:"choiceID"`
	Target             CapabilityTarget `json:"target"`
	AutoExecute        bool             `json:"autoExecute,omitempty"`
	UserArguments      json.RawMessage  `json:"userArguments,omitempty"`
	ImplementationKind string           `json:"implementationKind,omitempty"`
	ToolType           string           `json:"toolType,omitempty"`
	ProviderManaged    bool             `json:"providerManaged,omitempty"`
}

type SessionConfiguration struct {
	Model          ArtifactReference    `json:"model"`
	RequestPatch   json.RawMessage      `json:"requestPatch,omitempty"`
	SystemPrompt   *string              `json:"systemPrompt,omitempty"`
	History        HistorySelection     `json:"history,omitzero"`
	ToolSelections []ModelToolSelection `json:"toolSelections,omitempty"`
	MCPContext     json.RawMessage      `json:"mcpContext,omitempty"`

	// InstalledSkills is the complete effective Skill allow-list for the Run.
	// The frontend resolves Agent and Workspace state, then supplies the final
	// references here. Session does not distinguish their origin.
	InstalledSkills []ArtifactReference `json:"installedSkills,omitempty"`

	// Metadata is frontend-owned opaque presentation/provenance data. Session
	// persists it but never interprets Agent or Workspace data within it.
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type UserInput struct {
	Text          string                `json:"text,omitempty"`
	Attachments   []AttachmentReference `json:"attachments,omitempty"`
	MCPAppContext json.RawMessage       `json:"mcpAppContext,omitempty"`
	Metadata      json.RawMessage       `json:"metadata,omitempty"`
}

type ContinuationInput struct {
	Revision         Revision              `json:"revision"`
	ConsumedRevision Revision              `json:"consumedRevision,omitempty"`
	Text             string                `json:"text,omitempty"`
	Attachments      []AttachmentReference `json:"attachments,omitempty"`
	MCPAppContext    json.RawMessage       `json:"mcpAppContext,omitempty"`
	Configuration    *SessionConfiguration `json:"configuration,omitempty"`
	Metadata         json.RawMessage       `json:"metadata,omitempty"`
	UpdatedAt        time.Time             `json:"updatedAt,omitzero"`
}

type TranscriptMessage struct {
	ID                  string                `json:"id"`
	CreatedAt           time.Time             `json:"createdAt"`
	Role                MessageRole           `json:"role"`
	Status              string                `json:"status,omitempty"`
	Text                string                `json:"text,omitempty"`
	Attachments         []AttachmentReference `json:"attachments,omitempty"`
	AttachmentsHydrated bool                  `json:"attachmentsHydrated,omitempty"`
	ProtocolInputs      []json.RawMessage     `json:"protocolInputs,omitempty"`
	ProtocolOutputs     []json.RawMessage     `json:"protocolOutputs,omitempty"`
	ToolResults         []ToolResult          `json:"toolResults,omitempty"`
	MCPAppContext       json.RawMessage       `json:"mcpAppContext,omitempty"`
	OperationID         OperationID           `json:"operationID,omitempty"`
	Error               *OperationError       `json:"error,omitempty"`
	Metadata            json.RawMessage       `json:"metadata,omitempty"`
	RuntimeMetadata     json.RawMessage       `json:"runtimeMetadata,omitempty"`
}

type OperationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ToolBinding struct {
	Kind ToolBindingKind `json:"kind"`
	Data json.RawMessage `json:"data,omitempty"`
}

type ToolResult struct {
	ID             ToolResultID    `json:"id"`
	AttemptID      ToolAttemptID   `json:"attemptID,omitempty"`
	CallID         ToolCallID      `json:"callID"`
	ProviderCallID string          `json:"providerCallID,omitempty"`
	Name           string          `json:"name"`
	ChoiceID       string          `json:"choiceID,omitempty"`
	Type           string          `json:"type,omitempty"`
	Kind           ToolResultKind  `json:"kind"`
	Text           string          `json:"text,omitempty"`
	Structured     json.RawMessage `json:"structured,omitempty"`
	App            json.RawMessage `json:"app,omitempty"`
	Error          string          `json:"error,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
}

type ToolAttempt struct {
	ID         ToolAttemptID    `json:"id"`
	CallID     ToolCallID       `json:"callID"`
	State      ToolAttemptState `json:"state"`
	Automatic  bool             `json:"automatic"`
	StartedAt  time.Time        `json:"startedAt,omitzero"`
	FinishedAt time.Time        `json:"finishedAt,omitzero"`
	Error      string           `json:"error,omitempty"`
	Uncertain  bool             `json:"uncertain,omitempty"`
}

type ToolCall struct {
	ID                    ToolCallID      `json:"id"`
	OriginOperationID     OperationID     `json:"originOperationID"`
	ProviderCallID        string          `json:"providerCallID,omitempty"`
	Name                  string          `json:"name"`
	ChoiceID              string          `json:"choiceID,omitempty"`
	Type                  string          `json:"type,omitempty"`
	Arguments             json.RawMessage `json:"arguments,omitempty"`
	Binding               ToolBinding     `json:"binding"`
	Required              bool            `json:"required"`
	ProviderManaged       bool            `json:"providerManaged,omitempty"`
	Automatic             bool            `json:"automatic,omitempty"`
	State                 ToolCallState   `json:"state"`
	Attempts              []ToolAttempt   `json:"attempts,omitempty"`
	Results               []ToolResult    `json:"results,omitempty"`
	SelectedResultID      ToolResultID    `json:"selectedResultID,omitempty"`
	WaitID                WaitID          `json:"waitID,omitempty"`
	LastFailure           string          `json:"lastFailure,omitempty"`
	ConsumedByOperationID OperationID     `json:"consumedByOperationID,omitempty"`

	// Authorization is executor-private. It is deliberately excluded from
	// public snapshots and semantic checkpoint serialization.
	Authorization json.RawMessage `json:"-"`
}

type ToolCallProposal struct {
	ID              ToolCallID      `json:"id,omitempty"`
	ProviderCallID  string          `json:"providerCallID,omitempty"`
	Name            string          `json:"name"`
	ChoiceID        string          `json:"choiceID,omitempty"`
	Type            string          `json:"type,omitempty"`
	Arguments       json.RawMessage `json:"arguments,omitempty"`
	Binding         ToolBinding     `json:"binding"`
	Required        bool            `json:"required"`
	ProviderManaged bool            `json:"providerManaged,omitempty"`
	Automatic       bool            `json:"automatic,omitempty"`
	Error           string          `json:"error,omitempty"`
}

type WaitingRequest struct {
	ID                 WaitID           `json:"id"`
	Kind               WaitKind         `json:"kind"`
	Status             WaitStatus       `json:"status"`
	OperationID        OperationID      `json:"operationID,omitempty"`
	ToolCallID         ToolCallID       `json:"toolCallID,omitempty"`
	Summary            json.RawMessage  `json:"summary,omitempty"`
	InputSchema        json.RawMessage  `json:"inputSchema,omitempty"`
	AllowedResolutions []WaitResolution `json:"allowedResolutions,omitempty"`
	CreatedAt          time.Time        `json:"createdAt"`
	UpdatedAt          time.Time        `json:"updatedAt"`
	ExpiresAt          *time.Time       `json:"expiresAt,omitempty"`
	Error              string           `json:"error,omitempty"`
}

type AutomationHold struct {
	Enabled           bool                `json:"enabled"`
	Scope             AutomationHoldScope `json:"scope,omitempty"`
	TargetOperationID OperationID         `json:"targetOperationID,omitempty"`
	SetByCommandID    CommandID           `json:"setByCommandID,omitempty"`
}

type AutomationControls struct {
	AutomaticTools        AutomationHold `json:"automaticTools"`
	AutomaticContinuation AutomationHold `json:"automaticContinuation"`
	Paused                bool           `json:"paused"`
	CancellationRequested bool           `json:"cancellationRequested"`
}

type SkillSessionState struct {
	ID                    string              `json:"id,omitempty"`
	Generation            Revision            `json:"generation"`
	EstablishedGeneration Revision            `json:"establishedGeneration,omitempty"`
	Enabled               []ArtifactReference `json:"enabled,omitempty"`
	Active                []ArtifactReference `json:"active,omitempty"`
}

type RunLimits struct {
	MaxModelOperations int `json:"maxModelOperations,omitempty"`
	MaxToolAttempts    int `json:"maxToolAttempts,omitempty"`
}

type RunUsage struct {
	ModelOperations int `json:"modelOperations"`
	ToolAttempts    int `json:"toolAttempts"`
}

type RunOutcomeKind string

const (
	RunOutcomeCompleted RunOutcomeKind = "completed"
	RunOutcomeFailed    RunOutcomeKind = "failed"
	RunOutcomeCancelled RunOutcomeKind = "cancelled"
)

type RunOutcome struct {
	Kind    RunOutcomeKind `json:"kind"`
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message,omitempty"`
}

type DriverState struct {
	Kind    string          `json:"kind"`
	Version uint32          `json:"version"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type ModelOperationInput struct {
	Configuration        SessionConfiguration `json:"configuration"`
	History              []TranscriptMessage  `json:"history"`
	Current              TranscriptMessage    `json:"current"`
	ContinuationRevision Revision             `json:"continuationRevision,omitempty"`
}

type ModelOperation struct {
	ID               OperationID         `json:"id"`
	State            OperationState      `json:"state"`
	StartedAt        time.Time           `json:"startedAt,omitzero"`
	FinishedAt       time.Time           `json:"finishedAt,omitzero"`
	Request          ModelOperationInput `json:"request"`
	StreamedText     string              `json:"streamedText,omitempty"`
	StreamedThinking string              `json:"streamedThinking,omitempty"`
	ProtocolOutputs  []json.RawMessage   `json:"protocolOutputs,omitempty"`
	ToolCallIDs      []ToolCallID        `json:"toolCallIDs,omitempty"`
	Error            *OperationError     `json:"error,omitempty"`
	Usable           bool                `json:"usable"`
	Metadata         json.RawMessage     `json:"metadata,omitempty"`
	RuntimeMetadata  json.RawMessage     `json:"runtimeMetadata,omitempty"`
}

type Run struct {
	ID                     RunID                `json:"id"`
	SessionID              SessionID            `json:"sessionID"`
	ParentRunID            RunID                `json:"parentRunID,omitempty"`
	RootRunID              RunID                `json:"rootRunID,omitempty"`
	Revision               Revision             `json:"revision"`
	State                  RunState             `json:"state"`
	CreatedAt              time.Time            `json:"createdAt"`
	UpdatedAt              time.Time            `json:"updatedAt"`
	Driver                 DriverState          `json:"driver"`
	Configuration          SessionConfiguration `json:"configuration"`
	SkillSession           SkillSessionState    `json:"skillSession"`
	Controls               AutomationControls   `json:"controls"`
	Limits                 RunLimits            `json:"limits"`
	Usage                  RunUsage             `json:"usage"`
	Continuation           ContinuationInput    `json:"continuation"`
	ModelOperations        []ModelOperation     `json:"modelOperations,omitempty"`
	ToolCalls              []ToolCall           `json:"toolCalls,omitempty"`
	WaitingRequests        []WaitingRequest     `json:"waitingRequests,omitempty"`
	RequestedCallIDs       []ToolCallID         `json:"requestedCallIDs,omitempty"`
	ContinuationRequested  bool                 `json:"continuationRequested,omitempty"`
	ActiveModelOperationID OperationID          `json:"activeModelOperationID,omitempty"`
	Outcome                *RunOutcome          `json:"outcome,omitempty"`
	EventSequence          uint64               `json:"eventSequence"`
}

type Session struct {
	ID            SessionID            `json:"id"`
	Revision      Revision             `json:"revision"`
	Title         string               `json:"title,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
	UpdatedAt     time.Time            `json:"updatedAt"`
	Configuration SessionConfiguration `json:"configuration"`
	Transcript    []TranscriptMessage  `json:"transcript,omitempty"`
	RunIDs        []RunID              `json:"runIDs,omitempty"`
	EventSequence uint64               `json:"eventSequence"`
}

type SessionSnapshot struct {
	Session  Session `json:"session"`
	Runs     []Run   `json:"runs"`
	Sequence uint64  `json:"sequence"`
}

type RunSnapshot struct {
	SessionID       SessionID `json:"sessionID"`
	SessionRevision Revision  `json:"sessionRevision"`
	Run             Run       `json:"run"`
}

type SessionListItem struct {
	ID         SessionID `json:"id"`
	Revision   Revision  `json:"revision"`
	Title      string    `json:"title,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt"`
	ActiveRuns []RunID   `json:"activeRuns,omitempty"`
}

type CommandMeta struct {
	ID                      CommandID `json:"id"`
	ExpectedSessionRevision *Revision `json:"expectedSessionRevision,omitempty"`
	ExpectedRunRevision     *Revision `json:"expectedRunRevision,omitempty"`
}

type CommandReceipt struct {
	CommandID                CommandID     `json:"commandID"`
	Accepted                 bool          `json:"accepted"`
	Duplicate                bool          `json:"duplicate,omitempty"`
	SessionID                SessionID     `json:"sessionID,omitempty"`
	RunID                    RunID         `json:"runID,omitempty"`
	SessionRevision          Revision      `json:"sessionRevision,omitempty"`
	RunRevision              Revision      `json:"runRevision,omitempty"`
	OperationIDs             []OperationID `json:"operationIDs,omitempty"`
	ToolCallIDs              []ToolCallID  `json:"toolCallIDs,omitempty"`
	WaitIDs                  []WaitID      `json:"waitIDs,omitempty"`
	AlreadyActiveToolCallIDs []ToolCallID  `json:"alreadyActiveToolCallIDs,omitempty"`
	StarterText              string        `json:"starterText,omitempty"`
	Message                  string        `json:"message,omitempty"`
}

type CreateSessionCommand struct {
	Meta          CommandMeta          `json:"meta"`
	SessionID     SessionID            `json:"sessionID,omitempty"`
	Title         string               `json:"title,omitempty"`
	Configuration SessionConfiguration `json:"configuration,omitzero"`
}

type UpdateSessionConfigurationCommand struct {
	Meta          CommandMeta          `json:"meta"`
	SessionID     SessionID            `json:"sessionID"`
	Configuration SessionConfiguration `json:"configuration"`
}

type RenameSessionCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	Title     string      `json:"title"`
}

type DeleteSessionCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
}

type StartRunCommand struct {
	Meta          CommandMeta           `json:"meta"`
	SessionID     SessionID             `json:"sessionID"`
	RunID         RunID                 `json:"runID,omitempty"`
	Input         UserInput             `json:"input"`
	Configuration *SessionConfiguration `json:"configuration,omitempty"`
	Limits        *RunLimits            `json:"limits,omitempty"`
}

type UpdateContinuationInputCommand struct {
	Meta                  CommandMeta       `json:"meta"`
	SessionID             SessionID         `json:"sessionID"`
	RunID                 RunID             `json:"runID"`
	ExpectedInputRevision *Revision         `json:"expectedInputRevision,omitempty"`
	Input                 ContinuationInput `json:"input"`
}

type ContinueRunCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	RunID     RunID       `json:"runID"`
}

type ExecuteCallsCommand struct {
	Meta      CommandMeta  `json:"meta"`
	SessionID SessionID    `json:"sessionID"`
	RunID     RunID        `json:"runID"`
	CallIDs   []ToolCallID `json:"callIDs,omitempty"`
	Continue  bool         `json:"continue"`
}

type RetryCallCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	RunID     RunID       `json:"runID"`
	CallID    ToolCallID  `json:"callID"`
}

type SkipCallCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	RunID     RunID       `json:"runID"`
	CallID    ToolCallID  `json:"callID"`
}

type CompleteCallAsErrorCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	RunID     RunID       `json:"runID"`
	CallID    ToolCallID  `json:"callID"`
	Error     string      `json:"error"`
}

type ResolveWaitCommand struct {
	Meta       CommandMeta     `json:"meta"`
	SessionID  SessionID       `json:"sessionID"`
	RunID      RunID           `json:"runID"`
	WaitID     WaitID          `json:"waitID"`
	Resolution WaitResolution  `json:"resolution"`
	Input      json.RawMessage `json:"input,omitempty"`
}

type SetAutomationHoldCommand struct {
	Meta         CommandMeta         `json:"meta"`
	SessionID    SessionID           `json:"sessionID"`
	RunID        RunID               `json:"runID"`
	Tools        bool                `json:"tools"`
	Continuation bool                `json:"continuation"`
	Enabled      bool                `json:"enabled"`
	Scope        AutomationHoldScope `json:"scope,omitempty"`
}

type PauseRunCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	RunID     RunID       `json:"runID"`
}

type ResumeRunCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	RunID     RunID       `json:"runID"`
}

type CancelRunCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	RunID     RunID       `json:"runID"`
	Reason    string      `json:"reason,omitempty"`
}

type EditAndResendCommand struct {
	Meta             CommandMeta           `json:"meta"`
	SessionID        SessionID             `json:"sessionID"`
	ReplaceMessageID string                `json:"replaceMessageID"`
	RunID            RunID                 `json:"runID,omitempty"`
	Input            UserInput             `json:"input"`
	Configuration    *SessionConfiguration `json:"configuration,omitempty"`
	Limits           *RunLimits            `json:"limits,omitempty"`
}
