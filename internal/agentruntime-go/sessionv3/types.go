package sessionv3

import (
	"context"
	"errors"
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const SchemaVersion = "session-runtime/v3"

type (
	SessionID           string
	ExecutionID         string
	ActivityID          string
	ProgramFrameID      string
	WorkItemID          string
	AttemptID           string
	WaitID              string
	TriggerID           string
	TriggerOccurrenceID string
	CommandID           string
	AppInstanceID       string
	AppViewID           string
	Revision            uint64
)

// InferenceUsage is the provider-neutral usage record for one LLM WorkItem.
type InferenceUsage = inferenceSpec.Usage

type Session struct {
	ID                    SessionID                    `json:"id"`
	Revision              Revision                     `json:"revision"`
	Title                 string                       `json:"title,omitempty"`
	CreatedAt             time.Time                    `json:"createdAt"`
	UpdatedAt             time.Time                    `json:"updatedAt"`
	Defaults              SessionDefaults              `json:"defaults"`
	Metadata              CanonicalValue               `json:"metadata,omitempty"`
	Timeline              []Activity                   `json:"timeline"`
	Executions            []Execution                  `json:"executions"`
	ConversationOwnership []ConversationOwnershipState `json:"conversationOwnership,omitempty"`
	Triggers              []TriggerRegistration        `json:"triggers,omitempty"`
	EventSequence         uint64                       `json:"eventSequence"`
	Commands              []CommandRecord              `json:"commands,omitempty"`
}

type SessionDefaults struct {
	Model              *ArtifactSelection   `json:"model,omitempty"`
	Agent              *ArtifactSelection   `json:"agent,omitempty"`
	Workspace          *ArtifactSelection   `json:"workspace,omitempty"`
	SystemInstructions []Instruction        `json:"systemInstructions,omitempty"`
	Temperature        *float64             `json:"temperature,omitempty"`
	Metadata           CanonicalValue       `json:"metadata,omitempty"`
	History            HistorySelection     `json:"history"`
	Tools              []ArtifactSelection  `json:"tools,omitempty"`
	Skills             []ArtifactSelection  `json:"skills,omitempty"`
	ActiveSkills       []ArtifactSelection  `json:"activeSkills,omitempty"`
	MCP                []MCPServerSelection `json:"mcp,omitempty"`
	ConversationBudget *ExecutionBudget     `json:"conversationBudget,omitempty"`
}

// SessionDefaultsOverride changes only the existing V3 normal-session
// configuration surface for one submitted message or continuation.
//
// Set values apply first. Clear values apply last, so clear semantics are
// explicit and stable when an older client sends both representations.
type SessionDefaultsOverride struct {
	Model              *ArtifactSelection    `json:"model,omitempty"`
	Agent              *ArtifactSelection    `json:"agent,omitempty"`
	Workspace          *ArtifactSelection    `json:"workspace,omitempty"`
	SystemInstructions *[]Instruction        `json:"systemInstructions,omitempty"`
	Temperature        *float64              `json:"temperature,omitempty"`
	Metadata           *CanonicalValue       `json:"metadata,omitempty"`
	History            *HistorySelection     `json:"history,omitempty"`
	Tools              *[]ArtifactSelection  `json:"tools,omitempty"`
	Skills             *[]ArtifactSelection  `json:"skills,omitempty"`
	ActiveSkills       *[]ArtifactSelection  `json:"activeSkills,omitempty"`
	MCP                *[]MCPServerSelection `json:"mcp,omitempty"`
	Clear              []SessionDefaultField `json:"clear,omitempty"`
}

type SessionDefaultField string

const (
	SessionDefaultFieldModel              SessionDefaultField = "model"
	SessionDefaultFieldAgent              SessionDefaultField = "agent"
	SessionDefaultFieldWorkspace          SessionDefaultField = "workspace"
	SessionDefaultFieldSystemInstructions SessionDefaultField = "systemInstructions"
	SessionDefaultFieldTemperature        SessionDefaultField = "temperature"
	SessionDefaultFieldMetadata           SessionDefaultField = "metadata"
	SessionDefaultFieldHistory            SessionDefaultField = "history"
	SessionDefaultFieldTools              SessionDefaultField = "tools"
	SessionDefaultFieldSkills             SessionDefaultField = "skills"
	SessionDefaultFieldActiveSkills       SessionDefaultField = "activeSkills"
	SessionDefaultFieldMCP                SessionDefaultField = "mcp"
)

type HistorySelection struct {
	All                      bool `json:"all,omitempty"`
	IncludePreviousUserTurns int  `json:"includePreviousUserTurns,omitempty"`
}

type MCPToolExposure string

const (
	MCPToolExposureNone     MCPToolExposure = "none"
	MCPToolExposureAll      MCPToolExposure = "all"
	MCPToolExposureSelected MCPToolExposure = "selected"
)

type MCPToolSelection struct {
	ToolName       string  `json:"toolName"`
	ExpectedDigest string  `json:"expectedDigest,omitempty"`
	ApprovalRule   *string `json:"approvalRule,omitempty"`
	ExecutionMode  *string `json:"executionMode,omitempty"`
}

type MCPResourceSelection struct {
	URI            string `json:"uri"`
	ExpectedDigest string `json:"expectedDigest,omitempty"`
}

type MCPResourceTemplateSelection struct {
	URITemplate    string            `json:"uriTemplate"`
	ExpectedDigest string            `json:"expectedDigest,omitempty"`
	Arguments      map[string]string `json:"arguments,omitempty"`
}

type MCPPromptSelection struct {
	PromptName     string            `json:"promptName"`
	ExpectedDigest string            `json:"expectedDigest,omitempty"`
	Arguments      map[string]string `json:"arguments,omitempty"`
}

type MCPServerSelection struct {
	Server                    ArtifactSelection              `json:"server"`
	IncludeServerInstructions bool                           `json:"includeServerInstructions,omitempty"`
	ToolExposure              MCPToolExposure                `json:"toolExposure"`
	SelectedTools             []MCPToolSelection             `json:"selectedTools,omitempty"`
	Resources                 []MCPResourceSelection         `json:"resources,omitempty"`
	ResourceTemplates         []MCPResourceTemplateSelection `json:"resourceTemplates,omitempty"`
	Prompts                   []MCPPromptSelection           `json:"prompts,omitempty"`
}

type Instruction struct {
	ID      string `json:"id"`
	Label   string `json:"label,omitempty"`
	Content string `json:"content"`
}

type ArtifactSelection struct {
	RootID           string `json:"rootID"`
	ArtifactID       string `json:"artifactID"`
	ExpectedRevision uint64 `json:"expectedRevision,omitempty"`
	ExpectedDigest   string `json:"expectedDigest,omitempty"`
}

type ArtifactPin struct {
	RootID     string `json:"rootID"`
	ArtifactID string `json:"artifactID"`
	Revision   uint64 `json:"revision"`
	Digest     string `json:"digest"`
}

type ActivityKind string

const (
	ActivityKindMessage  ActivityKind = "message"
	ActivityKindAction   ActivityKind = "action"
	ActivityKindWorkflow ActivityKind = "workflow"
	ActivityKindWait     ActivityKind = "wait"
)

type ActivityVisibility string

const (
	ActivityVisibilityUser  ActivityVisibility = "user"
	ActivityVisibilityAudit ActivityVisibility = "audit"
	ActivityVisibilityBoth  ActivityVisibility = "both"
)

type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleDeveloper MessageRole = "developer"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

type Activity struct {
	ID          ActivityID         `json:"id"`
	Kind        ActivityKind       `json:"kind"`
	Visibility  ActivityVisibility `json:"visibility"`
	CreatedAt   time.Time          `json:"createdAt"`
	ExecutionID ExecutionID        `json:"executionID,omitempty"`
	FrameID     ProgramFrameID     `json:"frameID,omitempty"`
	WorkItemID  WorkItemID         `json:"workItemID,omitempty"`
	ParentID    ActivityID         `json:"parentID,omitempty"`
	Message     *MessageActivity   `json:"message,omitempty"`
	Action      *ActionActivity    `json:"action,omitempty"`
	Workflow    *WorkflowActivity  `json:"workflow,omitempty"`
	Wait        *WaitActivity      `json:"wait,omitempty"`
}

type MessageActivity struct {
	Role               MessageRole          `json:"role"`
	Status             string               `json:"status,omitempty"`
	Text               string               `json:"text,omitempty"`
	Attachments        []Attachment         `json:"attachments,omitempty"`
	ToolResults        []ToolResultActivity `json:"toolResults,omitempty"`
	AppContext         []AppContextUpdate   `json:"appContext,omitempty"`
	Metadata           CanonicalValue       `json:"metadata,omitempty"`
	Error              *SanitizedError      `json:"error,omitempty"`
	InvocationDefaults *SessionDefaults     `json:"invocationDefaults,omitempty"`
	InvocationBudget   *ExecutionBudget     `json:"invocationBudget,omitempty"`
}

type AttachmentKind string

const (
	AttachmentKindText  AttachmentKind = "text"
	AttachmentKindImage AttachmentKind = "image"
	AttachmentKindFile  AttachmentKind = "file"
	AttachmentKindURL   AttachmentKind = "url"
)

type AttachmentSourceKind string

const (
	AttachmentSourceKindFile      AttachmentSourceKind = "file"
	AttachmentSourceKindDirectory AttachmentSourceKind = "directory"
	AttachmentSourceKindURL       AttachmentSourceKind = "url"
)

type AttachmentSourceMode string

const (
	AttachmentSourceModeAuto  AttachmentSourceMode = "auto"
	AttachmentSourceModeText  AttachmentSourceMode = "text"
	AttachmentSourceModeFile  AttachmentSourceMode = "file"
	AttachmentSourceModeImage AttachmentSourceMode = "image"
	AttachmentSourceModePage  AttachmentSourceMode = "page"
	AttachmentSourceModeLink  AttachmentSourceMode = "link"
)

// Attachment is a semantic snapshot. It deliberately has no native path,
// file handle, browser object, or process-local reference.
type Attachment struct {
	Kind       AttachmentKind `json:"kind"`
	Label      string         `json:"label,omitempty"`
	MIMEType   string         `json:"mimeType,omitempty"`
	Digest     string         `json:"digest,omitempty"`
	Text       string         `json:"text,omitempty"`
	Base64Data string         `json:"base64Data,omitempty"`
	URL        string         `json:"url,omitempty"`
	SourceURL  string         `json:"sourceURL,omitempty"`
}

// AttachmentSource is command-only admission input. It is never stored in a
// Session document. The setup-owned AttachmentAdmission boundary converts it
// to immutable Attachment values before a message Activity is persisted.
type AttachmentSource struct {
	Kind  AttachmentSourceKind `json:"kind"`
	Path  string               `json:"path,omitempty"`
	URL   string               `json:"url,omitempty"`
	Label string               `json:"label,omitempty"`
	Mode  AttachmentSourceMode `json:"mode,omitempty"`
}

// AttachmentAdmissionRequest contains no durable Attachment payload. Native
// paths and URLs can exist only while the setup-owned admission domain creates
// immutable semantic snapshots.
type AttachmentAdmissionRequest struct {
	SessionID SessionID          `json:"sessionID"`
	Sources   []AttachmentSource `json:"sources"`
}

// AttachmentAdmission is the narrow external-domain boundary for V3 message
// attachment admission. Session Runtime does not inspect native paths, walk
// directories, read files, fetch URLs, or configure provider attachments.
type AttachmentAdmission interface {
	AdmitAttachments(
		ctx context.Context,
		request AttachmentAdmissionRequest,
	) ([]Attachment, error)
}

// ToolResultActivity is an explicit carried Tool result. It is used only by
// normal Session edit-and-resend and continuation behavior. It never fabricates
// Tool success because it is built only from a durable completed WorkItem.
type ToolResultActivity struct {
	WorkItemID     WorkItemID     `json:"workItemID"`
	CallID         string         `json:"callID"`
	ProviderCallID string         `json:"providerCallID,omitempty"`
	ChoiceID       string         `json:"choiceID,omitempty"`
	Name           string         `json:"name"`
	Kind           ToolOutputKind `json:"kind"`
	Text           string         `json:"text,omitempty"`
	Structured     CanonicalValue `json:"structured,omitempty"`
	Error          string         `json:"error,omitempty"`
}

type AppContextContent struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	URI         string `json:"uri,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
	Data        []byte `json:"data,omitempty"`

	DisplayName string `json:"displayName,omitempty"`
}

// AppContextUpdate is Session-owned approved context from an MCP App.
// It contains semantic values only. It does not contain a browser handle,
// MCP connection, approval token, or callback.
type AppContextUpdate struct {
	AppInstanceID     AppInstanceID       `json:"appInstanceID"`
	ViewID            AppViewID           `json:"viewID"`
	ServerID          string              `json:"serverID"`
	ResourceURI       string              `json:"resourceURI,omitempty"`
	Content           []AppContextContent `json:"content,omitempty"`
	StructuredContent CanonicalValue      `json:"structuredContent,omitempty"`
	UpdatedAt         time.Time           `json:"updatedAt"`

	ServerDisplayName      string `json:"serverDisplayName,omitempty"`
	AppDisplayName         string `json:"appDisplayName,omitempty"`
	ContextLabel           string `json:"contextLabel,omitempty"`
	StructuredContentLabel string `json:"structuredContentLabel,omitempty"`
}

type ActionActivity struct {
	Title      string           `json:"title"`
	Summary    string           `json:"summary,omitempty"`
	ActionKind string           `json:"actionKind,omitempty"`
	Outcome    string           `json:"outcome,omitempty"`
	Output     CanonicalValue   `json:"output,omitempty"`
	Error      *SanitizedError  `json:"error,omitempty"`
	Receipt    *ExternalReceipt `json:"receipt,omitempty"`
}

type WorkflowActivity struct {
	Phase   string          `json:"phase"`
	Summary string          `json:"summary,omitempty"`
	Path    string          `json:"path,omitempty"`
	Output  CanonicalValue  `json:"output,omitempty"`
	Error   *SanitizedError `json:"error,omitempty"`
}

type WaitActivity struct {
	WaitID     WaitID          `json:"waitID"`
	Status     WaitStatus      `json:"status"`
	Summary    string          `json:"summary,omitempty"`
	Resolution WaitResolution  `json:"resolution,omitempty"`
	Error      *SanitizedError `json:"error,omitempty"`
}

type ExecutionState string

const (
	ExecutionStateQueued     ExecutionState = "queued"
	ExecutionStateRunning    ExecutionState = "running"
	ExecutionStateWaiting    ExecutionState = "waiting"
	ExecutionStatePaused     ExecutionState = "paused"
	ExecutionStateCancelling ExecutionState = "cancelling"
	ExecutionStateCompleted  ExecutionState = "completed"
	ExecutionStateFailed     ExecutionState = "failed"
	ExecutionStateCancelled  ExecutionState = "cancelled"
)

func (s ExecutionState) Terminal() bool {
	switch s {
	case ExecutionStateCompleted, ExecutionStateFailed, ExecutionStateCancelled:
		return true
	default:
		return false
	}
}

// ExecutionChildLifetime describes an Execution that was created by another
// Execution. Attached child work remains in the parent Execution as a frame.
// Independent child work has its own durable Execution in the same Session.
type ExecutionChildLifetime string

const (
	ExecutionChildLifetimeRoot        ExecutionChildLifetime = "root"
	ExecutionChildLifetimeIndependent ExecutionChildLifetime = "independent"
)

// ExecutionBudget is the durable child budget inherited at independent child
// admission. It contains only typed declarative limits and no process-local
// scheduler or runtime handles.
type ExecutionBudget struct {
	Duration        WorkflowDuration     `json:"duration,omitempty"`
	StepExecutions  int                  `json:"stepExecutions,omitempty"`
	CallAttempts    int                  `json:"callAttempts,omitempty"`
	WorkflowDepth   int                  `json:"workflowDepth,omitempty"`
	Concurrency     int                  `json:"concurrency,omitempty"`
	ModelTokens     int                  `json:"modelTokens,omitempty"`
	MessageCount    int                  `json:"messageCount,omitempty"`
	MessageBytes    int                  `json:"messageBytes,omitempty"`
	EventCount      int                  `json:"eventCount,omitempty"`
	EventBytes      int                  `json:"eventBytes,omitempty"`
	CleanupDuration WorkflowDuration     `json:"cleanupDuration,omitempty"`
	Cost            *ExecutionBudgetCost `json:"cost,omitempty"`
}

type ExecutionBudgetCost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// ChildExecutionOwnership is the evaluated ownership contract captured before
// an independent child Execution starts.
type ChildExecutionOwnership struct {
	Owner            string          `json:"owner,omitempty"`
	CancelWithParent bool            `json:"cancelWithParent"`
	Budget           ExecutionBudget `json:"budget"`
}

// ConversationParticipant identifies the durable owner of a logical
// conversation. The owner is semantic state in the existing Session document.
type ConversationParticipant struct {
	Owner       string         `json:"owner,omitempty"`
	ExecutionID ExecutionID    `json:"executionID,omitempty"`
	FrameID     ProgramFrameID `json:"frameID,omitempty"`
	Agent       *ArtifactPin   `json:"agent,omitempty"`
}

// ConversationOwnershipState is a typed handoff record. It contains no model
// client, callback, browser state, or provider conversation handle.
type ConversationOwnershipState struct {
	ConversationID      string                   `json:"conversationID"`
	Owner               ConversationParticipant  `json:"owner"`
	PreviousOwner       *ConversationParticipant `json:"previousOwner,omitempty"`
	ActiveHandoffID     string                   `json:"activeHandoffID,omitempty"`
	RestoreOnCompletion bool                     `json:"restoreOnCompletion"`
	UpdatedAt           time.Time                `json:"updatedAt"`
	CompletedAt         *time.Time               `json:"completedAt,omitempty"`
}

// ConversationHandoffIntent is emitted only by the Workflow interpreter. The
// runtime persists the resulting ownership state before child dispatch.
type ConversationHandoffIntent struct {
	ID                  string      `json:"id"`
	ConversationID      string      `json:"conversationID"`
	Owner               string      `json:"owner,omitempty"`
	TargetAgent         ArtifactPin `json:"targetAgent"`
	RestoreOnCompletion bool        `json:"restoreOnCompletion"`
}

type Execution struct {
	ID                ExecutionID                  `json:"id"`
	SessionID         SessionID                    `json:"sessionID"`
	Revision          Revision                     `json:"revision"`
	State             ExecutionState               `json:"state"`
	CreatedAt         time.Time                    `json:"createdAt"`
	UpdatedAt         time.Time                    `json:"updatedAt"`
	ParentExecutionID ExecutionID                  `json:"parentExecutionID,omitempty"`
	RootExecutionID   ExecutionID                  `json:"rootExecutionID,omitempty"`
	ParentFrameID     ProgramFrameID               `json:"parentFrameID,omitempty"`
	Lifetime          ExecutionChildLifetime       `json:"lifetime,omitempty"`
	Owner             string                       `json:"owner,omitempty"`
	CancelWithParent  bool                         `json:"cancelWithParent,omitempty"`
	ChildBudget       ExecutionBudget              `json:"childBudget"`
	ChildExecutionIDs []ExecutionID                `json:"childExecutionIDs,omitempty"`
	HandoffID         string                       `json:"handoffID,omitempty"`
	Deadline          *time.Time                   `json:"deadline,omitempty"`
	Budget            ExecutionBudget              `json:"budget"`
	Usage             ExecutionBudgetUsage         `json:"usage"`
	Cleanup           CleanupState                 `json:"cleanup"`
	ProgramFrames     []ProgramFrame               `json:"programFrames"`
	WorkItems         []WorkItem                   `json:"workItems"`
	Attempts          []Attempt                    `json:"attempts"`
	Continuation      ContinuationInput            `json:"continuation"`
	Apps              []AppInstance                `json:"apps,omitempty"`
	Effects           []EffectRecord               `json:"effects,omitempty"`
	Trigger           *TriggerExecution            `json:"trigger,omitempty"`
	Waits             []Wait                       `json:"waits"`
	Signals           []ExecutionSignal            `json:"signals,omitempty"`
	Channels          []ChannelLog                 `json:"channels,omitempty"`
	Resources         []WorkflowResourceDefinition `json:"resources,omitempty"`
	ResourceLeases    []ResourceLease              `json:"resourceLeases,omitempty"`
	ResourceRates     []ResourceRateWindow         `json:"resourceRates,omitempty"`
	EmitReceipts      []EmitReceipt                `json:"emitReceipts,omitempty"`
	Controls          ExecutionControls            `json:"controls"`
	Outcome           *ExecutionOutcome            `json:"outcome,omitempty"`
	EventSequence     uint64                       `json:"eventSequence"`
}

type ExecutionControls struct {
	Paused                bool           `json:"paused"`
	CancellationRequested bool           `json:"cancellationRequested"`
	ContinueRequested     bool           `json:"continueRequested"`
	AutomaticTools        AutomationHold `json:"automaticTools"`
	AutomaticContinuation AutomationHold `json:"automaticContinuation"`
}

type AutomationHoldScope string

const (
	AutomationHoldOneBatch   AutomationHoldScope = "oneBatch"
	AutomationHoldIndefinite AutomationHoldScope = "indefinite"
)

type AutomationHold struct {
	Enabled          bool                `json:"enabled"`
	Scope            AutomationHoldScope `json:"scope,omitempty"`
	TargetWorkItemID WorkItemID          `json:"targetWorkItemID,omitempty"`
	SetByCommandID   CommandID           `json:"setByCommandID,omitempty"`
}

type ExecutionOutcomeKind string

const (
	ExecutionOutcomeCompleted ExecutionOutcomeKind = "completed"
	ExecutionOutcomeFailed    ExecutionOutcomeKind = "failed"
	ExecutionOutcomeCancelled ExecutionOutcomeKind = "cancelled"
)

type ExecutionOutcome struct {
	Kind        ExecutionOutcomeKind `json:"kind"`
	Code        string               `json:"code,omitempty"`
	Message     string               `json:"message,omitempty"`
	Error       *SanitizedError      `json:"error,omitempty"`
	CompletedAt time.Time            `json:"completedAt"`
}

type ProgramFrameState string

const (
	ProgramFrameStateQueued    ProgramFrameState = "queued"
	ProgramFrameStateRunning   ProgramFrameState = "running"
	ProgramFrameStateWaiting   ProgramFrameState = "waiting"
	ProgramFrameStateCompleted ProgramFrameState = "completed"
	ProgramFrameStateFailed    ProgramFrameState = "failed"
	ProgramFrameStateCancelled ProgramFrameState = "cancelled"
)

func (s ProgramFrameState) Terminal() bool {
	switch s {
	case ProgramFrameStateCompleted, ProgramFrameStateFailed, ProgramFrameStateCancelled:
		return true
	default:
		return false
	}
}

// ProgramSnapshot contains only durable orchestration meaning. Payload is
// bounded canonical JSON owned and validated by the named interpreter.
type ProgramSnapshot struct {
	InterpreterID      string         `json:"interpreterID"`
	InterpreterVersion uint32         `json:"interpreterVersion"`
	Artifact           *ArtifactPin   `json:"artifact,omitempty"`
	DefinitionDigest   string         `json:"definitionDigest,omitempty"`
	Payload            CanonicalValue `json:"payload,omitempty"`
}

type ProgramState struct {
	Type    string         `json:"type"`
	Version uint32         `json:"version"`
	Data    CanonicalValue `json:"data,omitempty"`
}

type ChildFrameLifetime string

const (
	ChildFrameLifetimeAttached    ChildFrameLifetime = "attached"
	ChildFrameLifetimeIndependent ChildFrameLifetime = "independent"
)

type ProgramFrame struct {
	ID                ProgramFrameID     `json:"id"`
	ParentFrameID     ProgramFrameID     `json:"parentFrameID,omitempty"`
	Lifetime          ChildFrameLifetime `json:"lifetime,omitempty"`
	Depth             int                `json:"depth,omitempty"`
	Program           ProgramSnapshot    `json:"program"`
	State             ProgramState       `json:"state"`
	Status            ProgramFrameState  `json:"status"`
	Path              string             `json:"path,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
	Deadline          *time.Time         `json:"deadline,omitempty"`
	Budget            *ExecutionBudget   `json:"budget,omitempty"`
	ChildFrameIDs     []ProgramFrameID   `json:"childFrameIDs,omitempty"`
	ChildExecutionIDs []ExecutionID      `json:"childExecutionIDs,omitempty"`
	WorkItemIDs       []WorkItemID       `json:"workItemIDs,omitempty"`
	Ports             []PortPlan         `json:"ports,omitempty"`
	PortInputs        []PortInput        `json:"portInputs,omitempty"`
	Output            CanonicalValue     `json:"output,omitempty"`
	Receipt           *ExternalReceipt   `json:"receipt,omitempty"`
	Error             *SanitizedError    `json:"error,omitempty"`
	HandoffID         string             `json:"handoffID,omitempty"`
	Fork              *ForkBoundary      `json:"fork,omitempty"`
	Joins             []JoinBoundary     `json:"joins,omitempty"`
}

// ForkBoundary is a durable child-scheduling boundary. The parent frame
// persists every child ProgramFrame intent before its Fork WorkItem dispatches.
// No child starts before that WorkItem has a confirmed result.
type ForkBoundary struct {
	WorkItemID     WorkItemID           `json:"workItemID"`
	Frames         []ProgramFrameIntent `json:"frames"`
	ContinueParent bool                 `json:"continueParent,omitempty"`
}

// JoinBoundary persists a pending child completion aggregation. A parent
// interpreter receives ProgramEventChildSettled only after the typed Join
// WorkItem confirms that the named child is terminal.
type JoinBoundary struct {
	WorkItemID   WorkItemID     `json:"workItemID"`
	ChildFrameID ProgramFrameID `json:"childFrameID"`
}

type WorkItemType string

const (
	WorkItemTypeLLM            WorkItemType = "llm"
	WorkItemTypeTool           WorkItemType = "tool"
	WorkItemTypeService        WorkItemType = "service"
	WorkItemTypeActivity       WorkItemType = "activity"
	WorkItemTypeTransform      WorkItemType = "transform"
	WorkItemTypeCondition      WorkItemType = "condition"
	WorkItemTypeStateUpdate    WorkItemType = "stateUpdate"
	WorkItemTypeFork           WorkItemType = "fork"
	WorkItemTypeJoin           WorkItemType = "join"
	WorkItemTypeTargetResolver WorkItemType = "targetResolver"
	WorkItemTypeWait           WorkItemType = "wait"
	WorkItemTypeEmit           WorkItemType = "emit"
	WorkItemTypeCompensation   WorkItemType = "compensation"
	WorkItemTypeReconciliation WorkItemType = "reconciliation"
)

type WorkItemState string

const (
	WorkItemStateReady           WorkItemState = "ready"
	WorkItemStateBinding         WorkItemState = "binding"
	WorkItemStateDispatched      WorkItemState = "dispatched"
	WorkItemStateRetryWaiting    WorkItemState = "retryWaiting"
	WorkItemStateResourceWaiting WorkItemState = "resourceWaiting"
	WorkItemStateWaiting         WorkItemState = "waiting"
	WorkItemStateReconciling     WorkItemState = "reconciling"
	WorkItemStateCompleted       WorkItemState = "completed"
	WorkItemStateFailed          WorkItemState = "failed"
	WorkItemStateUncertain       WorkItemState = "uncertain"
	WorkItemStateCancelled       WorkItemState = "cancelled"
	WorkItemStateBlocked         WorkItemState = "blocked"
)

func (s WorkItemState) Terminal() bool {
	switch s {
	case WorkItemStateCompleted,
		WorkItemStateFailed,
		WorkItemStateUncertain,
		WorkItemStateCancelled,
		WorkItemStateBlocked:
		return true
	default:
		return false
	}
}

// EffectState records the strongest durable fact known about a business
// effect. It is distinct from WorkItem state. A WorkItem can fail while an
// external effect is confirmed, uncertain, or compensated.
type EffectState string

const (
	EffectStateNotApplicable       EffectState = "notApplicable"
	EffectStateNotStarted          EffectState = "notStarted"
	EffectStateDispatched          EffectState = "dispatched"
	EffectStateConfirmedApplied    EffectState = "confirmedApplied"
	EffectStateConfirmedNotApplied EffectState = "confirmedNotApplied"
	EffectStateUncertain           EffectState = "uncertain"
	EffectStateCompensating        EffectState = "compensating"
	EffectStateCompensated         EffectState = "compensated"
	EffectStateCompensationFailed  EffectState = "compensationFailed"
)

type EffectKind string

const (
	EffectKindRead          EffectKind = "read"
	EffectKindIdempotent    EffectKind = "idempotent"
	EffectKindKeyed         EffectKind = "keyed"
	EffectKindNonIdempotent EffectKind = "nonIdempotent"
)

type EffectContract struct {
	Kind           EffectKind `json:"kind"`
	IdempotencyKey string     `json:"idempotencyKey,omitempty"`
	Reconcile      bool       `json:"reconcile,omitempty"`
	Compensate     bool       `json:"compensate,omitempty"`
}

type ReconciliationDecision string

const (
	ReconciliationDecisionApplied    ReconciliationDecision = "applied"
	ReconciliationDecisionNotApplied ReconciliationDecision = "notApplied"
	ReconciliationDecisionUnknown    ReconciliationDecision = "unknown"
)

// WorkflowOperationTargetFacts preserve the semantic target evidence visible
// to a reconciliation block. Private clients and setup-domain handles remain
// outside this durable value.
type WorkflowOperationTargetFacts struct {
	Name         string           `json:"name"`
	WorkItemType WorkItemType     `json:"workItemType"`
	Binding      *SemanticBinding `json:"binding,omitempty"`
}

// WorkflowOperationFacts are the stable facts provided to a reconciliation
// block through the Workflow `operation` CEL variable.
type WorkflowOperationFacts struct {
	WorkItemID     WorkItemID                   `json:"workItemID"`
	AttemptID      AttemptID                    `json:"attemptID,omitempty"`
	Input          CanonicalValue               `json:"input"`
	Target         WorkflowOperationTargetFacts `json:"target"`
	IdempotencyKey string                       `json:"idempotencyKey,omitempty"`
	Receipt        *ExternalReceipt             `json:"receipt,omitempty"`
	Uncertainty    *SanitizedError              `json:"uncertainty"`
}

// ReconciliationWork is a durable intent boundary. It records the original
// operation facts before a reconciliation child ProgramFrame starts.
type ReconciliationWork struct {
	OriginalWorkItemID WorkItemID             `json:"originalWorkItemID"`
	Operation          WorkflowOperationFacts `json:"operation"`
}

// ReconciliationSettlement is emitted by the Workflow interpreter only after
// a reconciliation child block returns a validated typed decision. Runtime
// applies it atomically to the original WorkItem.
type ReconciliationSettlement struct {
	OriginalWorkItemID WorkItemID             `json:"originalWorkItemID"`
	Decision           ReconciliationDecision `json:"decision"`
	Output             CanonicalValue         `json:"output,omitempty"`
	Receipt            *ExternalReceipt       `json:"receipt,omitempty"`
	Error              *SanitizedError        `json:"error,omitempty"`
}

// WorkflowStepEffect is the evaluated declaration-level effect constraint for
// one Workflow call. Its Key is a typed value template. The interpreter
// evaluates it once when the logical WorkItem is prepared and persists the
// concrete key in EffectContract.IdempotencyKey.
type WorkflowStepEffect struct {
	Kind       EffectKind     `json:"kind"`
	Key        *WorkflowValue `json:"key,omitempty"`
	Reconcile  *WorkflowBlock `json:"reconcile,omitempty"`
	Compensate *WorkflowBlock `json:"compensate,omitempty"`
}

// WorkflowEffectPlan is persisted in an effect-bearing WorkItem. It contains
// only the reconciliation block required to recover an uncertain original
// operation. Compensation remains a separate later durable boundary.
type WorkflowEffectPlan struct {
	Reconcile  *WorkflowBlock `json:"reconcile,omitempty"`
	Compensate *WorkflowBlock `json:"compensate,omitempty"`
}

func (p WorkflowEffectPlan) Validate() error {
	if p.Reconcile == nil && p.Compensate == nil {
		return errors.New("workflow effect plan has no recovery block")
	}
	if p.Reconcile != nil {
		if err := p.Reconcile.Validate(); err != nil {
			return err
		}
	}
	if p.Compensate != nil {
		if err := p.Compensate.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type RetryMatcher struct {
	Categories   []ErrorCategory `json:"categories,omitempty"`
	Codes        []string        `json:"codes,omitempty"`
	Retryable    *bool           `json:"retryable,omitempty"`
	EffectStates []EffectState   `json:"effectStates,omitempty"`
	Schema       CanonicalValue  `json:"schema,omitempty"`
}

// RetryBackoff is durable retry timing policy. Durations remain typed values
// rather than generic JSON. Jitter is deterministic per WorkItem attempt.
type RetryBackoff struct {
	Initial    WorkflowDuration `json:"initial"`
	Maximum    WorkflowDuration `json:"maximum,omitempty"`
	Multiplier float64          `json:"multiplier,omitempty"`
	Jitter     float64          `json:"jitter,omitempty"`
}

// RetryPolicy includes the first dispatch attempt in MaxAttempts.
type RetryPolicy struct {
	MaxAttempts       int              `json:"maxAttempts"`
	Match             RetryMatcher     `json:"match"`
	Backoff           RetryBackoff     `json:"backoff"`
	PerAttemptTimeout WorkflowDuration `json:"perAttemptTimeout"`
}

// RetryState is durable scheduler state. It never contains a timer or
// cancellation callback.
type RetryState struct {
	ScheduledAt   time.Time       `json:"scheduledAt"`
	NextAttemptAt time.Time       `json:"nextAttemptAt"`
	LastError     *SanitizedError `json:"lastError,omitempty"`
}

// WorkflowResourceRate is the durable declaration-level rate capacity. Units
// are consumed in fixed windows of Per. Burst is additional capacity available
// within one window.
type WorkflowResourceRate struct {
	Units int              `json:"units"`
	Per   WorkflowDuration `json:"per"`
	Burst int              `json:"burst,omitempty"`
}

// WorkflowResourceDefinition is a named resource available to all attached
// ProgramFrames inside one Execution. It contains no process-local lock,
// ticker, channel, callback, or client.
type WorkflowResourceDefinition struct {
	Name        string                `json:"name"`
	Concurrency int                   `json:"concurrency,omitempty"`
	Rate        *WorkflowResourceRate `json:"rate,omitempty"`
	Keyed       bool                  `json:"keyed,omitempty"`
}

// ResourceClaim is an evaluated WorkItem claim. Key is a concrete durable
// value evaluated once before the WorkItem intent is persisted.
type ResourceClaim struct {
	Resource string `json:"resource"`
	Units    int    `json:"units"`
	Key      string `json:"key,omitempty"`
}

// ResourceLease records a held claim. It is part of the existing Execution
// document and never contains a mutex, cancellation function, or timer.
type ResourceLease struct {
	WorkItemID WorkItemID     `json:"workItemID,omitempty"`
	FrameID    ProgramFrameID `json:"frameID,omitempty"`
	Resource   string         `json:"resource"`
	Units      int            `json:"units"`
	Key        string         `json:"key,omitempty"`
	AcquiredAt time.Time      `json:"acquiredAt"`
}

// ResourceRateWindow is durable rate-accounting state for one declared
// resource. The Runtime reconstructs timing behavior from these facts after
// restore.
type ResourceRateWindow struct {
	Resource      string    `json:"resource"`
	WindowStarted time.Time `json:"windowStarted"`
	UnitsConsumed int       `json:"unitsConsumed"`
}

type ResourceWaitReason string

const (
	ResourceWaitReasonConcurrency ResourceWaitReason = "concurrency"
	ResourceWaitReasonKeyed       ResourceWaitReason = "keyed"
	ResourceWaitReasonRate        ResourceWaitReason = "rate"
)

// ResourceWaitState is durable scheduler state. WakeAt is absent for keyed
// and concurrency waits because the owning Runtime wakes them after lease
// release. A rate wait always has a WakeAt.
type ResourceWaitState struct {
	Reason     ResourceWaitReason `json:"reason"`
	WakeAt     *time.Time         `json:"wakeAt,omitempty"`
	Generation uint64             `json:"generation"`
}

type WorkItem struct {
	ID                    WorkItemID              `json:"id"`
	FrameID               ProgramFrameID          `json:"frameID"`
	Type                  WorkItemType            `json:"type"`
	OriginWorkItemID      WorkItemID              `json:"originWorkItemID,omitempty"`
	State                 WorkItemState           `json:"state"`
	ProgramPath           string                  `json:"programPath,omitempty"`
	Dependencies          []WorkItemID            `json:"dependencies,omitempty"`
	AttemptIDs            []AttemptID             `json:"attemptIDs,omitempty"`
	Deadline              *time.Time              `json:"deadline,omitempty"`
	Effect                EffectContract          `json:"effect"`
	EffectState           EffectState             `json:"effectState,omitempty"`
	EffectPlan            *WorkflowEffectPlan     `json:"effectPlan,omitempty"`
	Automatic             bool                    `json:"automatic"`
	Retry                 *RetryPolicy            `json:"retry,omitempty"`
	RetryState            *RetryState             `json:"retryState,omitempty"`
	Budget                *ExecutionBudget        `json:"budget,omitempty"`
	Cleanup               bool                    `json:"cleanup,omitempty"`
	SagaID                string                  `json:"sagaID,omitempty"`
	CompensationPolicy    CompensationErrorPolicy `json:"compensationPolicy,omitempty"`
	Requested             bool                    `json:"requested"`
	RetainClaimsForFrame  bool                    `json:"retainClaimsForFrame,omitempty"`
	StepExecutionReserved bool                    `json:"stepExecutionReserved,omitempty"`
	Claims                []ResourceClaim         `json:"claims,omitempty"`
	ResourceWait          *ResourceWaitState      `json:"resourceWait,omitempty"`
	Ports                 []PortPlan              `json:"ports,omitempty"`
	PortInputs            []PortInput             `json:"portInputs,omitempty"`
	PortStreams           []PortStream            `json:"portStreams,omitempty"`
	Input                 WorkItemPayload         `json:"input"`
	Binding               *SemanticBinding        `json:"binding,omitempty"`
	Output                *WorkItemOutput         `json:"output,omitempty"`
	Error                 *SanitizedError         `json:"error,omitempty"`
	LLMStream             *LLMStreamState         `json:"llmStream,omitempty"`
	CreatedAt             time.Time               `json:"createdAt"`
	UpdatedAt             time.Time               `json:"updatedAt"`
	DispatchedAt          *time.Time              `json:"dispatchedAt,omitempty"`
	CompletedAt           *time.Time              `json:"completedAt,omitempty"`
	WaitID                WaitID                  `json:"waitID,omitempty"`
}

// WorkItemPayload is a closed tagged union. It must contain exactly one
// non-nil member that matches WorkItem.Type.
type WorkItemPayload struct {
	LLM            *LLMWork            `json:"llm,omitempty"`
	Tool           *ToolWork           `json:"tool,omitempty"`
	Service        *ServiceWork        `json:"service,omitempty"`
	Activity       *ActivityWork       `json:"activity,omitempty"`
	Transform      *TransformWork      `json:"transform,omitempty"`
	Condition      *ConditionWork      `json:"condition,omitempty"`
	StateUpdate    *StateUpdateWork    `json:"stateUpdate,omitempty"`
	Fork           *ForkWork           `json:"fork,omitempty"`
	Join           *JoinWork           `json:"join,omitempty"`
	TargetResolver *TargetResolverWork `json:"targetResolver,omitempty"`
	Wait           *WaitWork           `json:"wait,omitempty"`
	Emit           *EmitWork           `json:"emit,omitempty"`
	Compensation   *CompensationWork   `json:"compensation,omitempty"`
	Reconciliation *ReconciliationWork `json:"reconciliation,omitempty"`
}

type LLMWork struct {
	Model                   *ArtifactSelection `json:"model,omitempty"`
	Agent                   *ArtifactSelection `json:"agent,omitempty"`
	InputActivityID         ActivityID         `json:"inputActivityID"`
	ContinuationWorkItemIDs []WorkItemID       `json:"continuationWorkItemIDs,omitempty"`
	SystemInstructions      []Instruction      `json:"systemInstructions,omitempty"`
	Temperature             *float64           `json:"temperature,omitempty"`

	// InvocationDefaults are immutable per-operation facts. Normal
	// conversation work leaves this unset and uses Session defaults.
	InvocationDefaults *SessionDefaults `json:"invocationDefaults,omitempty"`

	// IsolatedInput prevents an LLM operation from selecting ambient Session
	// history. HistoryActivityIDs are the exact ordered prior Activities for
	// this invocation. InputActivityID remains the final current Activity.
	//
	// Workflow uses this general ProgramFrame capability for explicit target
	// input. The Model and Agent domains do not receive Workflow values.
	IsolatedInput      bool         `json:"isolatedInput,omitempty"`
	HistoryActivityIDs []ActivityID `json:"historyActivityIDs,omitempty"`
}

type ToolWork struct {
	Call   ToolCall       `json:"call"`
	Target *ArtifactPin   `json:"target,omitempty"`
	MCP    *MCPToolTarget `json:"mcp,omitempty"`
}

type ToolCallSource string

const (
	ToolCallSourceModel ToolCallSource = "model"
	ToolCallSourceApp   ToolCallSource = "app"
)

// MCPToolTarget is the semantic Workflow target for an MCP Tool. MCP Runtime
// discovers the current provider Tool and creates the executable MCP binding
// immediately before dispatch.
type MCPToolTarget struct {
	Server   ArtifactPin `json:"server"`
	ToolName string      `json:"toolName"`
}

type ToolCall struct {
	ID             string         `json:"id"`
	ProviderCallID string         `json:"providerCallID,omitempty"`
	Name           string         `json:"name"`
	ChoiceID       string         `json:"choiceID,omitempty"`
	DisplayName    string         `json:"displayName,omitempty"`
	Description    string         `json:"description,omitempty"`
	Source         ToolCallSource `json:"source,omitempty"`
	AppInstanceID  AppInstanceID  `json:"appInstanceID,omitempty"`
	Arguments      CanonicalValue `json:"arguments,omitempty"`
	Binding        ToolBinding    `json:"binding"`
	Automatic      bool           `json:"automatic"`
}

type ToolBindingKind string

const (
	ToolBindingKindGo         ToolBindingKind = "go"
	ToolBindingKindMCP        ToolBindingKind = "mcp"
	ToolBindingKindSkill      ToolBindingKind = "skill"
	ToolBindingKindProvider   ToolBindingKind = "provider"
	ToolBindingKindUnresolved ToolBindingKind = "unresolved"
)

type ToolBinding struct {
	Kind       ToolBindingKind        `json:"kind"`
	Go         *GoToolBinding         `json:"go,omitempty"`
	MCP        *MCPToolBinding        `json:"mcp,omitempty"`
	Skill      *SkillToolBinding      `json:"skill,omitempty"`
	Provider   *ProviderToolBinding   `json:"provider,omitempty"`
	Unresolved *UnresolvedToolBinding `json:"unresolved,omitempty"`
}

type GoToolBinding struct {
	Target      ArtifactPin `json:"target"`
	LogicalName string      `json:"logicalName,omitempty"`
	Function    string      `json:"function,omitempty"`
	TimeoutMS   int         `json:"timeoutMS,omitempty"`
}

type MCPToolBinding struct {
	ServerArtifact   *ArtifactPin `json:"serverArtifact,omitempty"`
	ServerID         string       `json:"serverID"`
	ProviderToolName string       `json:"providerToolName"`
	ChoiceID         string       `json:"choiceID"`
	ToolName         string       `json:"toolName"`
	ToolDigest       string       `json:"toolDigest"`
	ApprovalRule     string       `json:"approvalRule"`
	ExecutionMode    string       `json:"executionMode"`
}

type SkillToolBinding struct {
	Name string `json:"name"`
}

type ProviderToolBinding struct {
	Artifact *ArtifactPin `json:"artifact,omitempty"`
	Type     string       `json:"type"`
	Name     string       `json:"name"`
}

type UnresolvedToolBinding struct {
	Reason string `json:"reason"`
}

type ServiceWork struct {
	ServiceID      string         `json:"serviceID"`
	BindingVersion string         `json:"bindingVersion"`
	Input          CanonicalValue `json:"input"`
	Context        CanonicalValue `json:"context,omitempty"`
}

type ActivityWork struct {
	ActivityID     string         `json:"activityID"`
	BindingVersion string         `json:"bindingVersion"`
	Input          CanonicalValue `json:"input"`
	Context        CanonicalValue `json:"context,omitempty"`
}

type TransformWork struct {
	Input        CanonicalValue `json:"input"`
	OutputSchema CanonicalValue `json:"outputSchema,omitempty"`
}

type ConditionWork struct {
	Input CanonicalValue `json:"input"`
}

type StateAssignment struct {
	Path  string         `json:"path"`
	Value CanonicalValue `json:"value"`
}

type StateUpdateWork struct {
	// StateOwnerID identifies the nearest state-owning scope that supplied
	// Base and Schema. It prevents a child frame from applying its result to
	// a different state boundary after restore or structured control flow.
	StateOwnerID string `json:"stateOwnerID"`

	// Base is the immutable state snapshot observed when the Workflow step was
	// prepared. All assignments apply atomically to this value.
	Base CanonicalValue `json:"base"`

	// Schema is the state-owning block boundary. It is generic JSON Schema
	// data, not a compiled schema.
	Schema CanonicalValue `json:"schema,omitempty"`

	Assignments []StateAssignment `json:"assignments,omitempty"`
	RemovePaths []string          `json:"removePaths,omitempty"`
}

type ForkWork struct {
	ChildFrames []ProgramFrameID `json:"childFrames"`
}

type JoinWork struct {
	ChildFrames []ProgramFrameID `json:"childFrames"`
	Policy      string           `json:"policy"`
	Quorum      int              `json:"quorum,omitempty"`
}

// TargetResolverWork is a durable resolver result boundary. The interpreter
// evaluates declaration templates once, stores normalized candidate members in
// this intent, then persists a resolver WorkItem before accepting membership.
// This prevents selection from changing during retries or recovery.
type TargetResolverWork struct {
	TargetSet string   `json:"targetSet"`
	Members   []string `json:"members"`
}

type TargetResolverOutput struct {
	TargetSet string   `json:"targetSet"`
	Members   []string `json:"members"`
}

type WaitWork struct {
	Request WaitRequest `json:"request"`
}

type EmitDestination struct {
	Channel string `json:"channel,omitempty"`
	Event   string `json:"event,omitempty"`
}

// CompletionHandle identifies one durable WorkItem or ProgramFrame. A caller
// can wait for only one target per wait.
type CompletionHandle struct {
	WorkItemID  WorkItemID     `json:"workItemID,omitempty"`
	FrameID     ProgramFrameID `json:"frameID,omitempty"`
	ExecutionID ExecutionID    `json:"executionID,omitempty"`
}

// ExecutionSignal is the durable event envelope accepted by SignalExecution.
// ConsumedBy is scoped to one durable wait, not globally destructive.
type ExecutionSignal struct {
	ID                       string                    `json:"id"`
	Name                     string                    `json:"name"`
	Source                   string                    `json:"source,omitempty"`
	Data                     CanonicalValue            `json:"data"`
	Correlation              CanonicalValue            `json:"correlation,omitempty"`
	CreatedAt                time.Time                 `json:"createdAt"`
	ConsumedBy               []WaitID                  `json:"consumedBy,omitempty"`
	ConsumedSubscriptions    []string                  `json:"consumedSubscriptions,omitempty"`
	SubscriptionConsumptions []SubscriptionConsumption `json:"subscriptionConsumptions,omitempty"`
}

// SubscriptionConsumption identifies one subscription-scoped event
// consumption fact. FrameID is required because inherited subscriptions can
// have the same declaration name in more than one active ProgramFrame.
type SubscriptionConsumption struct {
	FrameID      ProgramFrameID `json:"frameID"`
	Subscription string         `json:"subscription"`
}

// ChannelMessage is one retained local channel record.
type ChannelMessage struct {
	ID               string         `json:"id"`
	Sequence         uint64         `json:"sequence"`
	Channel          string         `json:"channel"`
	Kind             string         `json:"kind,omitempty"`
	Data             CanonicalValue `json:"data"`
	Correlation      CanonicalValue `json:"correlation,omitempty"`
	DeduplicationKey string         `json:"deduplicationKey,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	ConsumedBy       []WaitID       `json:"consumedBy,omitempty"`
}

// ChannelLog is an invocation-local append-only channel log.
type ChannelLog struct {
	Name         string           `json:"name"`
	NextSequence uint64           `json:"nextSequence"`
	MaxMessages  int              `json:"maxMessages,omitempty"`
	MaxBytes     int              `json:"maxBytes,omitempty"`
	Bytes        int              `json:"bytes,omitempty"`
	Messages     []ChannelMessage `json:"messages"`
}

// EmitReceipt confirms durable local channel or execution-event acceptance.
type EmitReceipt struct {
	ID               string          `json:"id"`
	Destination      EmitDestination `json:"destination"`
	Sequence         uint64          `json:"sequence,omitempty"`
	DeduplicationKey string          `json:"deduplicationKey,omitempty"`
	AcceptedAt       time.Time       `json:"acceptedAt"`
}

type EmitWork struct {
	Destination       EmitDestination `json:"destination"`
	Kind              string          `json:"kind,omitempty"`
	Data              CanonicalValue  `json:"data"`
	Correlation       CanonicalValue  `json:"correlation,omitempty"`
	DeduplicationKey  string          `json:"deduplicationKey,omitempty"`
	EventSource       string          `json:"eventSource,omitempty"`
	DataSchema        CanonicalValue  `json:"dataSchema,omitempty"`
	CorrelationSchema CanonicalValue  `json:"correlationSchema,omitempty"`
	MaxMessages       int             `json:"maxMessages,omitempty"`
	MaxBytes          int             `json:"maxBytes,omitempty"`
}

type CompensationWork struct {
	OriginalWorkItemID WorkItemID     `json:"originalWorkItemID"`
	EffectID           EffectRecordID `json:"effectID"`
	Input              CanonicalValue `json:"input,omitempty"`
}

type SemanticBinding struct {
	Version           string                `json:"version"`
	Artifacts         []ArtifactPin         `json:"artifacts,omitempty"`
	Plan              *ModelDispatchPlan    `json:"plan,omitempty"`
	Service           *ServiceDispatchPlan  `json:"service,omitempty"`
	Activity          *ActivityDispatchPlan `json:"activity,omitempty"`
	TargetFingerprint string                `json:"targetFingerprint"`
	BoundAt           time.Time             `json:"boundAt"`
}

type WorkItemOutput struct {
	LLM            *LLMOutput            `json:"llm,omitempty"`
	Tool           *ToolOutput           `json:"tool,omitempty"`
	Value          CanonicalValue        `json:"value,omitempty"`
	Emit           *EmitReceipt          `json:"emit,omitempty"`
	App            *AppOutput            `json:"app,omitempty"`
	AppInstanceID  AppInstanceID         `json:"appInstanceID,omitempty"`
	TargetResolver *TargetResolverOutput `json:"targetResolver,omitempty"`
	Receipt        *ExternalReceipt      `json:"receipt,omitempty"`
}

// AppOutput is the durable semantic result of an MCP Tool response that
// exposes an App. The Tool binding is retained on the originating WorkItem.
type AppOutput struct {
	ServerID    string `json:"serverID"`
	ResourceURI string `json:"resourceURI,omitempty"`
}

type AppInstance struct {
	ID                AppInstanceID  `json:"id"`
	ExecutionID       ExecutionID    `json:"executionID"`
	OriginWorkItemID  WorkItemID     `json:"originWorkItemID"`
	OriginFrameID     ProgramFrameID `json:"originFrameID"`
	ServerID          string         `json:"serverID"`
	ResourceURI       string         `json:"resourceURI,omitempty"`
	Binding           ToolBinding    `json:"binding"`
	CreatedAt         time.Time      `json:"createdAt"`
	Views             []AppViewID    `json:"views,omitempty"`
	UnavailableReason string         `json:"unavailableReason,omitempty"`
}

type ContinuationInput struct {
	Revision           Revision           `json:"revision"`
	ConsumedRevision   Revision           `json:"consumedRevision,omitempty"`
	Text               string             `json:"text,omitempty"`
	Attachments        []Attachment       `json:"attachments,omitempty"`
	AppContext         []AppContextUpdate `json:"appContext,omitempty"`
	Metadata           CanonicalValue     `json:"metadata,omitempty"`
	InvocationDefaults *SessionDefaults   `json:"invocationDefaults,omitempty"`
	InvocationBudget   *ExecutionBudget   `json:"invocationBudget,omitempty"`
	UpdatedAt          time.Time          `json:"updatedAt,omitzero"`
}

// LLMStreamState is the durable incremental output observed before a final
// provider result. It contains no provider callback, client, or token.
type LLMStreamState struct {
	Text      string    `json:"text,omitempty"`
	Thinking  string    `json:"thinking,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type LLMOutput struct {
	Text            string           `json:"text,omitempty"`
	Thinking        string           `json:"thinking,omitempty"`
	ToolCalls       []ToolCall       `json:"toolCalls,omitempty"`
	Usage           *InferenceUsage  `json:"usage,omitempty"`
	ProtocolOutputs []CanonicalValue `json:"protocolOutputs,omitempty"`
	DebugDetails    CanonicalValue   `json:"debugDetails,omitempty"`
}

type ToolOutputKind string

const (
	ToolOutputKindSuccess ToolOutputKind = "success"
	ToolOutputKindError   ToolOutputKind = "error"
	ToolOutputKindSkipped ToolOutputKind = "skipped"
)

type ToolOutput struct {
	Kind       ToolOutputKind `json:"kind"`
	Text       string         `json:"text,omitempty"`
	Structured CanonicalValue `json:"structured,omitempty"`
	Error      string         `json:"error,omitempty"`
}

type ExternalReceipt struct {
	System      string      `json:"system"`
	Reference   string      `json:"reference"`
	ConfirmedAt time.Time   `json:"confirmedAt"`
	EffectState EffectState `json:"effectState,omitempty"`
}

type AttemptState string

const (
	AttemptStateIntent     AttemptState = "intent"
	AttemptStateDispatched AttemptState = "dispatched"
	AttemptStateConfirmed  AttemptState = "confirmed"
	AttemptStateFailed     AttemptState = "failed"
	AttemptStateUncertain  AttemptState = "uncertain"
	AttemptStateCancelled  AttemptState = "cancelled"
)

type Attempt struct {
	ID             AttemptID        `json:"id"`
	WorkItemID     WorkItemID       `json:"workItemID"`
	State          AttemptState     `json:"state"`
	Number         int              `json:"number"`
	IdempotencyKey string           `json:"idempotencyKey,omitempty"`
	EffectState    EffectState      `json:"effectState,omitempty"`
	StartedAt      time.Time        `json:"startedAt"`
	Deadline       *time.Time       `json:"deadline,omitempty"`
	FinishedAt     *time.Time       `json:"finishedAt,omitempty"`
	Receipt        *ExternalReceipt `json:"receipt,omitempty"`
	Error          *SanitizedError  `json:"error,omitempty"`
}

type WaitKind string

const (
	WaitKindApproval WaitKind = "approval"
	WaitKindManual   WaitKind = "manual"
	WaitKindHuman    WaitKind = "human"
	WaitKindTimer    WaitKind = "timer"
	WaitKindEvent    WaitKind = "event"
	WaitKindChannel  WaitKind = "channel"
	WaitKindChild    WaitKind = "child"
	WaitKindRecovery WaitKind = "recovery"
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

type WaitResolution string

const (
	WaitResolutionApproveOnce   WaitResolution = "approveOnce"
	WaitResolutionApproveAlways WaitResolution = "approveAlways"
	WaitResolutionDenyOnce      WaitResolution = "denyOnce"
	WaitResolutionDenyAlways    WaitResolution = "denyAlways"
	WaitResolutionContinue      WaitResolution = "continue"
	WaitResolutionCancel        WaitResolution = "cancel"
	WaitResolutionExecute       WaitResolution = "execute"
	WaitResolutionRespond       WaitResolution = "respond"
	WaitResolutionSignal        WaitResolution = "signal"
)

type WaitRequest struct {
	Kind               WaitKind         `json:"kind"`
	Summary            CanonicalValue   `json:"summary,omitempty"`
	InputSchema        CanonicalValue   `json:"inputSchema,omitempty"`
	AllowedResolutions []WaitResolution `json:"allowedResolutions"`
	Deadline           *time.Time       `json:"deadline,omitempty"`
	RuntimeBound       bool             `json:"runtimeBound"`

	Correlation       CanonicalValue      `json:"correlation,omitempty"`
	CorrelationSchema CanonicalValue      `json:"correlationSchema,omitempty"`
	DataSchema        CanonicalValue      `json:"dataSchema,omitempty"`
	EventSource       string              `json:"eventSource,omitempty"`
	Filter            *WorkflowWaitFilter `json:"filter,omitempty"`

	Subscription        string            `json:"subscription,omitempty"`
	SubscriptionFrameID ProgramFrameID    `json:"subscriptionFrameID,omitempty"`
	Signal              string            `json:"signal,omitempty"`
	Channel             string            `json:"channel,omitempty"`
	Cursor              uint64            `json:"cursor,omitempty"`
	Consume             bool              `json:"consume,omitempty"`
	Completion          *CompletionHandle `json:"completion,omitempty"`
	Timer               *TimerWait        `json:"timer,omitempty"`

	Assignees *WaitAssignees `json:"assignees,omitempty"`
	Request   CanonicalValue `json:"request,omitempty"`
}

type WaitAssignees struct {
	Users []string `json:"users,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

type Wait struct {
	ID         WaitID          `json:"id"`
	WorkItemID WorkItemID      `json:"workItemID"`
	Status     WaitStatus      `json:"status"`
	Request    WaitRequest     `json:"request"`
	Resolution WaitResolution  `json:"resolution,omitempty"`
	Response   CanonicalValue  `json:"response,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
	ResolvedBy *HumanActor     `json:"resolvedBy,omitempty"`
	Error      *SanitizedError `json:"error,omitempty"`
}

type SanitizedError struct {
	Code      string           `json:"code"`
	Category  ErrorCategory    `json:"category,omitempty"`
	Message   string           `json:"message"`
	Retryable bool             `json:"retryable,omitempty"`
	StepPath  string           `json:"stepPath,omitempty"`
	Effect    EffectState      `json:"effectState,omitempty"`
	Details   CanonicalValue   `json:"details,omitempty"`
	Causes    []SanitizedError `json:"causes,omitempty"`
}

type ProgramEventKind string

const (
	ProgramEventStarted      ProgramEventKind = "started"
	ProgramEventActivity     ProgramEventKind = "activity"
	ProgramEventWorkSettled  ProgramEventKind = "workSettled"
	ProgramEventChildSettled ProgramEventKind = "childSettled"
	ProgramEventContinue     ProgramEventKind = "continue"
	ProgramEventCancelled    ProgramEventKind = "cancelled"
)

type ProgramEvent struct {
	Kind               ProgramEventKind `json:"kind"`
	At                 time.Time        `json:"at"`
	ActivityID         ActivityID       `json:"activityID,omitempty"`
	WorkItemID         WorkItemID       `json:"workItemID,omitempty"`
	ChildFrameID       ProgramFrameID   `json:"childFrameID,omitempty"`
	InvocationDefaults *SessionDefaults `json:"invocationDefaults,omitempty"`
	InvocationBudget   *ExecutionBudget `json:"invocationBudget,omitempty"`
}

type ProgramFrameIntent struct {
	ID          ProgramFrameID             `json:"id,omitempty"`
	ExecutionID ExecutionID                `json:"executionID,omitempty"`
	Path        string                     `json:"path,omitempty"`
	Program     ProgramSnapshot            `json:"program"`
	Deadline    *time.Time                 `json:"deadline,omitempty"`
	Budget      *ExecutionBudget           `json:"budget,omitempty"`
	Lifetime    ChildFrameLifetime         `json:"lifetime,omitempty"`
	Ownership   *ChildExecutionOwnership   `json:"ownership,omitempty"`
	Ports       []PortPlan                 `json:"ports,omitempty"`
	Handoff     *ConversationHandoffIntent `json:"handoff,omitempty"`
}

type WorkItemIntent struct {
	ID                   WorkItemID              `json:"id,omitempty"`
	ProgramPath          string                  `json:"programPath,omitempty"`
	Type                 WorkItemType            `json:"type"`
	OriginWorkItemID     WorkItemID              `json:"originWorkItemID,omitempty"`
	Automatic            bool                    `json:"automatic"`
	Deadline             *time.Time              `json:"deadline,omitempty"`
	Effect               EffectContract          `json:"effect"`
	Retry                *RetryPolicy            `json:"retry,omitempty"`
	EffectPlan           *WorkflowEffectPlan     `json:"effectPlan,omitempty"`
	Budget               *ExecutionBudget        `json:"budget,omitempty"`
	Cleanup              bool                    `json:"cleanup,omitempty"`
	SagaID               string                  `json:"sagaID,omitempty"`
	CompensationPolicy   CompensationErrorPolicy `json:"compensationPolicy,omitempty"`
	Claims               []ResourceClaim         `json:"claims,omitempty"`
	Dependencies         []WorkItemID            `json:"dependencies,omitempty"`
	RetainClaimsForFrame bool                    `json:"retainClaimsForFrame,omitempty"`
	Ports                []PortPlan              `json:"ports,omitempty"`
	Input                WorkItemPayload         `json:"input"`
}

type ProgramTransition struct {
	State                   ProgramState               `json:"state"`
	Status                  *ProgramFrameState         `json:"status,omitempty"`
	Output                  CanonicalValue             `json:"output,omitempty"`
	Error                   *SanitizedError            `json:"error,omitempty"`
	Activities              []Activity                 `json:"activities,omitempty"`
	Frames                  []ProgramFrameIntent       `json:"frames,omitempty"`
	WorkItems               []WorkItemIntent           `json:"workItems,omitempty"`
	ForkClaims              []ResourceClaim            `json:"forkClaims,omitempty"`
	ReleaseFrameClaims      bool                       `json:"releaseFrameClaims,omitempty"`
	CancelChildPaths        []string                   `json:"cancelChildPaths,omitempty"`
	CancelChildFrameIDs     []ProgramFrameID           `json:"cancelChildFrameIDs,omitempty"`
	CancelChildExecutionIDs []ExecutionID              `json:"cancelChildExecutionIDs,omitempty"`
	CancelWorkItemIDs       []WorkItemID               `json:"cancelWorkItemIDs,omitempty"`
	Reconciliations         []ReconciliationSettlement `json:"reconciliations,omitempty"`
	Compensations           []CompensationSettlement   `json:"compensations,omitempty"`
	ContinueParent          bool                       `json:"continueParent,omitempty"`
}

type ProgramInterpreter interface {
	ID() string
	Version() uint32
	ValidateSnapshot(snapshot ProgramSnapshot) error
	ValidateState(state ProgramState) error
	InitialState(snapshot ProgramSnapshot) (ProgramState, error)
	Advance(frame ProgramFrame, execution Execution, event ProgramEvent) (ProgramTransition, error)
}

// WorkPortStream is the narrow runtime-owned callback used by a bound
// external domain to append a value to one declared streamed output port.
// The callback is process-private and is never serialized.
type WorkPortStream struct {
	Emit func(name string, value CanonicalValue) error `json:"-"`
}

type WorkDispatchRequest struct {
	Session    Session
	Execution  Execution
	WorkItem   WorkItem
	Stream     WorkStream     `json:"-"`
	PortStream WorkPortStream `json:"-"`
}

type WorkStream struct {
	OnText     func(text string) error `json:"-"`
	OnThinking func(text string) error `json:"-"`
}

type BindResult struct {
	SemanticBinding SemanticBinding
	Wait            *WaitRequest
	ImmediateOutput *WorkItemOutput
	Private         any `json:"-"`
}

type BoundWork struct {
	SemanticBinding SemanticBinding
	Private         any `json:"-"`
}

type WorkExecutionResult struct {
	Output    *WorkItemOutput
	Error     *SanitizedError
	Confirmed bool
}

type WaitResolutionRequest struct {
	Session    Session
	Execution  Execution
	WorkItem   WorkItem
	Wait       Wait
	Resolution WaitResolution
	Response   CanonicalValue
	Private    any `json:"-"`
}

type WaitResolutionResult struct {
	Resume    bool
	Bound     *BoundWork
	Output    *WorkItemOutput
	Error     *SanitizedError
	Confirmed bool
}

type WorkBinder interface {
	BindWork(ctx context.Context, req WorkDispatchRequest) (BindResult, error)
	ExecuteWork(ctx context.Context, req WorkDispatchRequest, work BoundWork) (WorkExecutionResult, error)
	ResolveWait(ctx context.Context, req WaitResolutionRequest) (WaitResolutionResult, error)
}

type UserMessageSkillTemplatePreparer interface {
	PrepareUserMessageSkillTemplate(
		ctx context.Context,
		request UserMessageSkillTemplatePreparationRequest,
	) (UserMessageSkillTemplateResult, error)
}

type InstructionSkillPreviewer interface {
	PreviewInstructionSkill(
		ctx context.Context,
		request InstructionSkillPreviewRequest,
	) (InstructionSkillPreviewResult, error)
}

// AppExecutor is an optional setup-owned bridge for Session-scoped MCP App
// operations. Session Runtime owns App identity, views, staged context, and
// command idempotency. Setup owns MCP policy, discovery, and transport.
type AppExecutor interface {
	PrepareAppTool(
		ctx context.Context,
		request AppToolPreparationRequest,
	) (ToolCall, error)
	ReadAppResource(
		ctx context.Context,
		request AppResourceReadRequest,
	) (AppResourceReadResult, error)
	ValidateAppContext(
		ctx context.Context,
		request AppContextStagingRequest,
	) (AppContextUpdate, error)
}

type ExecutionBindingCloser interface {
	CloseExecutionBinding(ctx context.Context, executionID ExecutionID) error
}

type WorkflowAdmissionRequest struct {
	SessionID   SessionID         `json:"sessionID"`
	ExecutionID ExecutionID       `json:"executionID"`
	Workflow    ArtifactSelection `json:"workflow"`
	Input       CanonicalValue    `json:"input"`
}

type WorkflowAdmissionResult struct {
	Program ProgramSnapshot `json:"program"`
}

type WorkflowAdmission interface {
	AdmitWorkflow(ctx context.Context, w WorkflowAdmissionRequest) (WorkflowAdmissionResult, error)
}

// WorkflowTriggerRegistrationAdmission compiles and pins a Workflow program
// template for trigger registration. Unlike WorkflowAdmission, it does not
// require invocation input and does not create an Execution.
type WorkflowTriggerRegistrationAdmission interface {
	AdmitWorkflowTriggers(
		ctx context.Context,
		request WorkflowTriggerRegistrationRequest,
	) (WorkflowAdmissionResult, error)
}

type CheckpointView struct {
	Reason    string    `json:"reason"`
	Session   Session   `json:"session"`
	CreatedAt time.Time `json:"createdAt"`
}

type CheckpointSink interface {
	HandleCheckpoint(ctx context.Context, checkpoint CheckpointView) error
}

type SessionDeletionSink interface {
	HandleSessionDeleted(ctx context.Context, sessionID SessionID) error
}

// SessionTombstoneDeletionSink preserves deletion-command idempotency with
// the durable deletion barrier. It is optional for compatibility with older
// checkpoint sinks.
type SessionTombstoneDeletionSink interface {
	HandleSessionTombstone(
		ctx context.Context,
		tombstone SessionTombstone,
	) error
}

type SessionTombstoneRestorer interface {
	RestoreSessionTombstone(
		ctx context.Context,
		tombstone SessionTombstone,
	) error
}

type RuntimeRestorer interface {
	RestoreSession(ctx context.Context, session Session) error
}

type RetiredSessionRestorer interface {
	RestoreRetiredSession(ctx context.Context, sessionID SessionID) error
}

type RecoverySource interface {
	Restore(ctx context.Context, restorer RuntimeRestorer) error
}

type RuntimeEventKind string

const (
	RuntimeEventSessionCreated   RuntimeEventKind = "sessionCreated"
	RuntimeEventSessionChanged   RuntimeEventKind = "sessionChanged"
	RuntimeEventExecutionChanged RuntimeEventKind = "executionChanged"
	RuntimeEventWorkChanged      RuntimeEventKind = "workChanged"
	RuntimeEventWaitChanged      RuntimeEventKind = "waitChanged"
	RuntimeEventRecoveryApplied  RuntimeEventKind = "recoveryApplied"
)

type RuntimeEvent struct {
	SessionID   SessionID        `json:"sessionID"`
	ExecutionID ExecutionID      `json:"executionID,omitempty"`
	Sequence    uint64           `json:"sequence"`
	Kind        RuntimeEventKind `json:"kind"`
	At          time.Time        `json:"at"`
}

type EventPublisher interface {
	PublishRuntimeEvent(ctx context.Context, e RuntimeEvent) error
}

type CommandMeta struct {
	ID                        CommandID `json:"id"`
	ExpectedSessionRevision   *Revision `json:"expectedSessionRevision,omitempty"`
	ExpectedExecutionRevision *Revision `json:"expectedExecutionRevision,omitempty"`
}

type CommandRecord struct {
	ID          CommandID      `json:"id"`
	Fingerprint string         `json:"fingerprint"`
	Receipt     CommandReceipt `json:"receipt"`
	RecordedAt  time.Time      `json:"recordedAt"`
}

// SessionTombstone is the durable deletion barrier retained after a Session
// document is deleted. Commands are retained so an accepted deletion command
// remains idempotent across Runtime restart.
type SessionTombstone struct {
	SessionID SessionID       `json:"sessionID"`
	DeletedAt time.Time       `json:"deletedAt"`
	Commands  []CommandRecord `json:"commands,omitempty"`
}

type CommandReceipt struct {
	CommandID         CommandID       `json:"commandID"`
	Accepted          bool            `json:"accepted"`
	Duplicate         bool            `json:"duplicate,omitempty"`
	SessionID         SessionID       `json:"sessionID,omitempty"`
	ExecutionID       ExecutionID     `json:"executionID,omitempty"`
	SessionRevision   Revision        `json:"sessionRevision,omitempty"`
	ExecutionRevision Revision        `json:"executionRevision,omitempty"`
	ActivityIDs       []ActivityID    `json:"activityIDs,omitempty"`
	WorkItemIDs       []WorkItemID    `json:"workItemIDs,omitempty"`
	WaitIDs           []WaitID        `json:"waitIDs,omitempty"`
	AppInstanceIDs    []AppInstanceID `json:"appInstanceIDs,omitempty"`
	Message           string          `json:"message,omitempty"`
}

type CreateSessionCommand struct {
	Meta      CommandMeta     `json:"meta"`
	SessionID SessionID       `json:"sessionID,omitempty"`
	Title     string          `json:"title,omitempty"`
	Defaults  SessionDefaults `json:"defaults"`
	Metadata  CanonicalValue  `json:"metadata,omitempty"`
}

type RenameSessionCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
	Title     string      `json:"title"`
}

type UpdateSessionDefaultsCommand struct {
	Meta      CommandMeta     `json:"meta"`
	SessionID SessionID       `json:"sessionID"`
	Defaults  SessionDefaults `json:"defaults"`
}

type DeleteSessionCommand struct {
	Meta      CommandMeta `json:"meta"`
	SessionID SessionID   `json:"sessionID"`
}

type SubmitMessageCommand struct {
	Meta      CommandMeta  `json:"meta"`
	SessionID SessionID    `json:"sessionID"`
	Input     MessageInput `json:"input"`
}

type StartWorkflowCommand struct {
	Meta        CommandMeta       `json:"meta"`
	SessionID   SessionID         `json:"sessionID"`
	ExecutionID ExecutionID       `json:"executionID,omitempty"`
	Workflow    ArtifactSelection `json:"workflow"`
	Input       CanonicalValue    `json:"input"`
}

type WorkflowTriggerRegistrationRequest struct {
	SessionID SessionID         `json:"sessionID"`
	Workflow  ArtifactSelection `json:"workflow"`
}

type RegisterWorkflowTriggersCommand struct {
	Meta      CommandMeta       `json:"meta"`
	SessionID SessionID         `json:"sessionID"`
	Workflow  ArtifactSelection `json:"workflow"`
}

type SetTriggerRegistrationStateCommand struct {
	Meta      CommandMeta              `json:"meta"`
	SessionID SessionID                `json:"sessionID"`
	TriggerID TriggerID                `json:"triggerID"`
	State     TriggerRegistrationState `json:"state"`
}

// AcceptTriggerEventCommand supplies one externally accepted event to the
// durable Workflow trigger registry inside an existing Session document.
//
// Source and EventID form the external deduplication identity. Command
// idempotency remains independent and is retained by Session.Commands.
type AcceptTriggerEventCommand struct {
	Meta        CommandMeta    `json:"meta"`
	SessionID   SessionID      `json:"sessionID"`
	Source      string         `json:"source"`
	EventID     string         `json:"eventID"`
	Name        string         `json:"name"`
	Data        CanonicalValue `json:"data"`
	Correlation CanonicalValue `json:"correlation,omitempty"`
	OccurredAt  time.Time      `json:"occurredAt"`
}

type SignalExecutionCommand struct {
	Meta        CommandMeta    `json:"meta"`
	SessionID   SessionID      `json:"sessionID"`
	ExecutionID ExecutionID    `json:"executionID"`
	SignalID    string         `json:"signalID,omitempty"`
	Source      string         `json:"source,omitempty"`
	Name        string         `json:"name"`
	Correlation CanonicalValue `json:"correlation,omitempty"`
	Data        CanonicalValue `json:"data"`
}

type MessageInput struct {
	Text              string                   `json:"text,omitempty"`
	Attachments       []Attachment             `json:"attachments,omitempty"`
	AttachmentSources []AttachmentSource       `json:"attachmentSources,omitempty"`
	Metadata          CanonicalValue           `json:"metadata,omitempty"`
	Defaults          *SessionDefaultsOverride `json:"defaults,omitempty"`
	InvocationBudget  *ExecutionBudget         `json:"invocationBudget,omitempty"`
}

// UserMessageSkillTemplatePreparationRequest renders a selected Skill with
// user-message insertion behavior. It is read-only and never creates an
// Execution-owned Skill session.
type UserMessageSkillTemplatePreparationRequest struct {
	SessionID SessionID         `json:"sessionID,omitempty"`
	Skill     ArtifactSelection `json:"skill"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type UserMessageSkillTemplateResult struct {
	Skill ArtifactPin `json:"skill"`
	Text  string      `json:"text"`
}

// InstructionSkillPreviewRequest renders an instructions Skill without
// creating a Session, Execution, provider request, or Skill session.
type InstructionSkillPreviewRequest struct {
	Skill     ArtifactSelection `json:"skill"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type InstructionSkillPreviewResult struct {
	Skill ArtifactPin `json:"skill"`
	Text  string      `json:"text"`
}

type UpdateContinuationInputCommand struct {
	Meta                  CommandMeta  `json:"meta"`
	SessionID             SessionID    `json:"sessionID"`
	ExecutionID           ExecutionID  `json:"executionID"`
	ExpectedInputRevision *Revision    `json:"expectedInputRevision,omitempty"`
	Input                 MessageInput `json:"input"`
}

type ExecuteToolCallsCommand struct {
	Meta        CommandMeta  `json:"meta"`
	SessionID   SessionID    `json:"sessionID"`
	ExecutionID ExecutionID  `json:"executionID"`
	WorkItemIDs []WorkItemID `json:"workItemIDs,omitempty"`
	Continue    bool         `json:"continue,omitempty"`
}

type RetryToolCallCommand struct {
	Meta        CommandMeta `json:"meta"`
	SessionID   SessionID   `json:"sessionID"`
	ExecutionID ExecutionID `json:"executionID"`
	WorkItemID  WorkItemID  `json:"workItemID"`
}

type SkipToolCallCommand struct {
	Meta        CommandMeta `json:"meta"`
	SessionID   SessionID   `json:"sessionID"`
	ExecutionID ExecutionID `json:"executionID"`
	WorkItemID  WorkItemID  `json:"workItemID"`
}

type CompleteToolCallAsErrorCommand struct {
	Meta        CommandMeta `json:"meta"`
	SessionID   SessionID   `json:"sessionID"`
	ExecutionID ExecutionID `json:"executionID"`
	WorkItemID  WorkItemID  `json:"workItemID"`
	Error       string      `json:"error"`
}

type ExecuteWorkItemsCommand struct {
	Meta        CommandMeta  `json:"meta"`
	SessionID   SessionID    `json:"sessionID"`
	ExecutionID ExecutionID  `json:"executionID"`
	WorkItemIDs []WorkItemID `json:"workItemIDs"`
}

type ContinueExecutionCommand struct {
	Meta        CommandMeta `json:"meta"`
	SessionID   SessionID   `json:"sessionID"`
	ExecutionID ExecutionID `json:"executionID"`
}

type ResolveWaitCommand struct {
	Meta        CommandMeta    `json:"meta"`
	SessionID   SessionID      `json:"sessionID"`
	ExecutionID ExecutionID    `json:"executionID"`
	WaitID      WaitID         `json:"waitID"`
	Resolution  WaitResolution `json:"resolution"`
	Response    CanonicalValue `json:"response,omitempty"`
	Actor       *HumanActor    `json:"actor,omitempty"`
}

type PauseExecutionCommand struct {
	Meta        CommandMeta `json:"meta"`
	SessionID   SessionID   `json:"sessionID"`
	ExecutionID ExecutionID `json:"executionID"`
}

type ResumeExecutionCommand struct {
	Meta        CommandMeta `json:"meta"`
	SessionID   SessionID   `json:"sessionID"`
	ExecutionID ExecutionID `json:"executionID"`
}

type CancelExecutionCommand struct {
	Meta        CommandMeta `json:"meta"`
	SessionID   SessionID   `json:"sessionID"`
	ExecutionID ExecutionID `json:"executionID"`
	Reason      string      `json:"reason,omitempty"`
}

type SetAutomationHoldCommand struct {
	Meta         CommandMeta         `json:"meta"`
	SessionID    SessionID           `json:"sessionID"`
	ExecutionID  ExecutionID         `json:"executionID"`
	Tools        bool                `json:"tools"`
	Continuation bool                `json:"continuation"`
	Enabled      bool                `json:"enabled"`
	Scope        AutomationHoldScope `json:"scope,omitempty"`
}

type OpenAppViewCommand struct {
	Meta          CommandMeta   `json:"meta"`
	SessionID     SessionID     `json:"sessionID"`
	ExecutionID   ExecutionID   `json:"executionID"`
	AppInstanceID AppInstanceID `json:"appInstanceID"`
	ViewID        AppViewID     `json:"viewID"`
}

type ExecuteAppToolCommand struct {
	Meta          CommandMeta    `json:"meta"`
	SessionID     SessionID      `json:"sessionID"`
	ExecutionID   ExecutionID    `json:"executionID"`
	AppInstanceID AppInstanceID  `json:"appInstanceID"`
	ViewID        AppViewID      `json:"viewID"`
	ToolName      string         `json:"toolName"`
	Arguments     CanonicalValue `json:"arguments,omitempty"`
}

type StageAppContextCommand struct {
	Meta          CommandMeta      `json:"meta"`
	SessionID     SessionID        `json:"sessionID"`
	ExecutionID   ExecutionID      `json:"executionID"`
	AppInstanceID AppInstanceID    `json:"appInstanceID"`
	ViewID        AppViewID        `json:"viewID"`
	Update        AppContextUpdate `json:"update"`
}

type AppResourceReadRequest struct {
	SessionID     SessionID     `json:"sessionID"`
	ExecutionID   ExecutionID   `json:"executionID"`
	AppInstanceID AppInstanceID `json:"appInstanceID"`
	ViewID        AppViewID     `json:"viewID"`
	ResourceURI   string        `json:"resourceURI,omitempty"`

	App AppInstance `json:"-"`
}

type AppContextStagingRequest struct {
	SessionID   SessionID
	ExecutionID ExecutionID
	App         AppInstance
	ViewID      AppViewID
	Update      AppContextUpdate
}

type AppResourceReadResult struct {
	Content CanonicalValue `json:"content"`
}

type EditAndResendCommand struct {
	Meta                    CommandMeta  `json:"meta"`
	SessionID               SessionID    `json:"sessionID"`
	ReplaceActivityID       ActivityID   `json:"replaceActivityID"`
	ExecutionID             ExecutionID  `json:"executionID,omitempty"`
	Input                   MessageInput `json:"input"`
	CarryForwardWorkItemIDs []WorkItemID `json:"carryForwardWorkItemIDs,omitempty"`
}

type ConversationProjection struct {
	SchemaVersion   string     `json:"schemaVersion"`
	SessionID       SessionID  `json:"sessionID"`
	SessionRevision Revision   `json:"sessionRevision"`
	Title           string     `json:"title,omitempty"`
	Messages        []Activity `json:"messages"`
}

type ActivityProjection struct {
	SchemaVersion   string     `json:"schemaVersion"`
	SessionID       SessionID  `json:"sessionID"`
	SessionRevision Revision   `json:"sessionRevision"`
	Activities      []Activity `json:"activities"`
}

// SessionSnapshotProjection is the frontend-ready V3 read. Every field is
// derived from the durable Session document, with no provider, Artifact,
// browser, Wails, credential, or process-private binding dependency.
type SessionSnapshotProjection struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Session       Session                  `json:"session"`
	Conversation  ConversationProjection   `json:"conversation"`
	Activities    ActivityProjection       `json:"activities"`
	Runtime       SessionRuntimeProjection `json:"runtime"`
	Workflow      WorkflowProjection       `json:"workflow"`
}

type WorkflowProjection struct {
	SchemaVersion   string                          `json:"schemaVersion"`
	SessionID       SessionID                       `json:"sessionID"`
	SessionRevision Revision                        `json:"sessionRevision"`
	Activities      []Activity                      `json:"activities"`
	Executions      []Execution                     `json:"executions"`
	StepOutcomes    []WorkflowStepOutcomeProjection `json:"stepOutcomes"`
}

type WorkflowStepOutcomeProjection struct {
	ExecutionID ExecutionID           `json:"executionID"`
	FrameID     ProgramFrameID        `json:"frameID"`
	Outcomes    []WorkflowStepOutcome `json:"outcomes"`
}

func (id WorkItemID) String() string     { return string(id) }
func (id ActivityID) String() string     { return string(id) }
func (id ProgramFrameID) String() string { return string(id) }
func (id ExecutionID) String() string    { return string(id) }
func (id SessionID) String() string      { return string(id) }
func (id WaitID) String() string         { return string(id) }
