package sessionv3

import (
	"context"
	"time"

	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const SchemaVersion = "session-runtime/v3"

type (
	SessionID      string
	ExecutionID    string
	ActivityID     string
	ProgramFrameID string
	WorkItemID     string
	AttemptID      string
	WaitID         string
	CommandID      string
	Revision       uint64
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
}

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
	Role        MessageRole     `json:"role"`
	Status      string          `json:"status,omitempty"`
	Text        string          `json:"text,omitempty"`
	Attachments []Attachment    `json:"attachments,omitempty"`
	Metadata    CanonicalValue  `json:"metadata,omitempty"`
	Error       *SanitizedError `json:"error,omitempty"`
}

type AttachmentKind string

const (
	AttachmentKindText  AttachmentKind = "text"
	AttachmentKindImage AttachmentKind = "image"
	AttachmentKindFile  AttachmentKind = "file"
	AttachmentKindURL   AttachmentKind = "url"
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
	ID                ExecutionID            `json:"id"`
	SessionID         SessionID              `json:"sessionID"`
	Revision          Revision               `json:"revision"`
	State             ExecutionState         `json:"state"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
	ParentExecutionID ExecutionID            `json:"parentExecutionID,omitempty"`
	RootExecutionID   ExecutionID            `json:"rootExecutionID,omitempty"`
	ParentFrameID     ProgramFrameID         `json:"parentFrameID,omitempty"`
	Lifetime          ExecutionChildLifetime `json:"lifetime,omitempty"`
	Owner             string                 `json:"owner,omitempty"`
	CancelWithParent  bool                   `json:"cancelWithParent,omitempty"`
	ChildBudget       ExecutionBudget        `json:"childBudget"`
	ChildExecutionIDs []ExecutionID          `json:"childExecutionIDs,omitempty"`
	HandoffID         string                 `json:"handoffID,omitempty"`
	ProgramFrames     []ProgramFrame         `json:"programFrames"`
	WorkItems         []WorkItem             `json:"workItems"`
	Attempts          []Attempt              `json:"attempts"`
	Waits             []Wait                 `json:"waits"`
	Signals           []ExecutionSignal      `json:"signals,omitempty"`
	Channels          []ChannelLog           `json:"channels,omitempty"`
	EmitReceipts      []EmitReceipt          `json:"emitReceipts,omitempty"`
	Controls          ExecutionControls      `json:"controls"`
	Outcome           *ExecutionOutcome      `json:"outcome,omitempty"`
	EventSequence     uint64                 `json:"eventSequence"`
}

type ExecutionControls struct {
	Paused                bool `json:"paused"`
	CancellationRequested bool `json:"cancellationRequested"`
	ContinueRequested     bool `json:"continueRequested"`
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
	Program           ProgramSnapshot    `json:"program"`
	State             ProgramState       `json:"state"`
	Status            ProgramFrameState  `json:"status"`
	Path              string             `json:"path,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
	Deadline          *time.Time         `json:"deadline,omitempty"`
	ChildFrameIDs     []ProgramFrameID   `json:"childFrameIDs,omitempty"`
	ChildExecutionIDs []ExecutionID      `json:"childExecutionIDs,omitempty"`
	WorkItemIDs       []WorkItemID       `json:"workItemIDs,omitempty"`
	Output            CanonicalValue     `json:"output,omitempty"`
	Error             *SanitizedError    `json:"error,omitempty"`
	HandoffID         string             `json:"handoffID,omitempty"`
}

type WorkItemType string

const (
	WorkItemTypeLLM          WorkItemType = "llm"
	WorkItemTypeTool         WorkItemType = "tool"
	WorkItemTypeService      WorkItemType = "service"
	WorkItemTypeActivity     WorkItemType = "activity"
	WorkItemTypeTransform    WorkItemType = "transform"
	WorkItemTypeCondition    WorkItemType = "condition"
	WorkItemTypeStateUpdate  WorkItemType = "stateUpdate"
	WorkItemTypeFork         WorkItemType = "fork"
	WorkItemTypeJoin         WorkItemType = "join"
	WorkItemTypeWait         WorkItemType = "wait"
	WorkItemTypeEmit         WorkItemType = "emit"
	WorkItemTypeCompensation WorkItemType = "compensation"
)

type WorkItemState string

const (
	WorkItemStateReady        WorkItemState = "ready"
	WorkItemStateBinding      WorkItemState = "binding"
	WorkItemStateDispatched   WorkItemState = "dispatched"
	WorkItemStateRetryWaiting WorkItemState = "retryWaiting"
	WorkItemStateWaiting      WorkItemState = "waiting"
	WorkItemStateCompleted    WorkItemState = "completed"
	WorkItemStateFailed       WorkItemState = "failed"
	WorkItemStateUncertain    WorkItemState = "uncertain"
	WorkItemStateCancelled    WorkItemState = "cancelled"
	WorkItemStateBlocked      WorkItemState = "blocked"
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

// WorkflowStepEffect is the evaluated effect constraint attached to one
// Workflow call. Key remains a typed value template until the interpreter
// prepares the concrete WorkItem.
type WorkflowStepEffect struct {
	Kind EffectKind     `json:"kind"`
	Key  *WorkflowValue `json:"key,omitempty"`
}

type RetryMatcher struct {
	Categories []ErrorCategory `json:"categories,omitempty"`
	Codes      []string        `json:"codes,omitempty"`
	Retryable  *bool           `json:"retryable,omitempty"`
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

type WorkItem struct {
	ID           WorkItemID       `json:"id"`
	FrameID      ProgramFrameID   `json:"frameID"`
	Type         WorkItemType     `json:"type"`
	State        WorkItemState    `json:"state"`
	ProgramPath  string           `json:"programPath,omitempty"`
	Dependencies []WorkItemID     `json:"dependencies,omitempty"`
	AttemptIDs   []AttemptID      `json:"attemptIDs,omitempty"`
	Deadline     *time.Time       `json:"deadline,omitempty"`
	Effect       EffectContract   `json:"effect"`
	Automatic    bool             `json:"automatic"`
	Retry        *RetryPolicy     `json:"retry,omitempty"`
	RetryState   *RetryState      `json:"retryState,omitempty"`
	Requested    bool             `json:"requested"`
	Input        WorkItemPayload  `json:"input"`
	Binding      *SemanticBinding `json:"binding,omitempty"`
	Output       *WorkItemOutput  `json:"output,omitempty"`
	Error        *SanitizedError  `json:"error,omitempty"`
	LLMStream    *LLMStreamState  `json:"llmStream,omitempty"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
	DispatchedAt *time.Time       `json:"dispatchedAt,omitempty"`
	CompletedAt  *time.Time       `json:"completedAt,omitempty"`
	WaitID       WaitID           `json:"waitID,omitempty"`
}

// WorkItemPayload is a closed tagged union. It must contain exactly one
// non-nil member that matches WorkItem.Type.
type WorkItemPayload struct {
	LLM          *LLMWork          `json:"llm,omitempty"`
	Tool         *ToolWork         `json:"tool,omitempty"`
	Service      *ServiceWork      `json:"service,omitempty"`
	Activity     *ActivityWork     `json:"activity,omitempty"`
	Transform    *TransformWork    `json:"transform,omitempty"`
	Condition    *ConditionWork    `json:"condition,omitempty"`
	StateUpdate  *StateUpdateWork  `json:"stateUpdate,omitempty"`
	Fork         *ForkWork         `json:"fork,omitempty"`
	Join         *JoinWork         `json:"join,omitempty"`
	Wait         *WaitWork         `json:"wait,omitempty"`
	Emit         *EmitWork         `json:"emit,omitempty"`
	Compensation *CompensationWork `json:"compensation,omitempty"`
}

type LLMWork struct {
	Model                   *ArtifactSelection `json:"model,omitempty"`
	Agent                   *ArtifactSelection `json:"agent,omitempty"`
	InputActivityID         ActivityID         `json:"inputActivityID"`
	ContinuationWorkItemIDs []WorkItemID       `json:"continuationWorkItemIDs,omitempty"`
	SystemInstructions      []Instruction      `json:"systemInstructions,omitempty"`
	Temperature             *float64           `json:"temperature,omitempty"`
	Invocation              *LLMInvocationPlan `json:"invocation,omitempty"`
}

type ToolWork struct {
	Call   ToolCall       `json:"call"`
	Target *ArtifactPin   `json:"target,omitempty"`
	MCP    *MCPToolTarget `json:"mcp,omitempty"`
}

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
	Type string `json:"type"`
	Name string `json:"name"`
}

type UnresolvedToolBinding struct {
	Reason string `json:"reason"`
}

type ServiceWork struct {
	ServiceID      string         `json:"serviceID"`
	BindingVersion string         `json:"bindingVersion"`
	Input          CanonicalValue `json:"input"`
}

type ActivityWork struct {
	ActivityID     string         `json:"activityID"`
	BindingVersion string         `json:"bindingVersion"`
	Input          CanonicalValue `json:"input"`
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
	ID                    string         `json:"id"`
	Name                  string         `json:"name"`
	Source                string         `json:"source,omitempty"`
	Data                  CanonicalValue `json:"data"`
	Correlation           CanonicalValue `json:"correlation,omitempty"`
	CreatedAt             time.Time      `json:"createdAt"`
	ConsumedBy            []WaitID       `json:"consumedBy,omitempty"`
	ConsumedSubscriptions []string       `json:"consumedSubscriptions,omitempty"`
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
	Input              CanonicalValue `json:"input,omitempty"`
}

type SemanticBinding struct {
	Version           string        `json:"version"`
	Artifacts         []ArtifactPin `json:"artifacts,omitempty"`
	TargetFingerprint string        `json:"targetFingerprint"`
	BoundAt           time.Time     `json:"boundAt"`
}

type WorkItemOutput struct {
	LLM     *LLMOutput       `json:"llm,omitempty"`
	Tool    *ToolOutput      `json:"tool,omitempty"`
	Value   CanonicalValue   `json:"value,omitempty"`
	Emit    *EmitReceipt     `json:"emit,omitempty"`
	Receipt *ExternalReceipt `json:"receipt,omitempty"`
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
	System      string    `json:"system"`
	Reference   string    `json:"reference"`
	ConfirmedAt time.Time `json:"confirmedAt"`
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

	Subscription string            `json:"subscription,omitempty"`
	Signal       string            `json:"signal,omitempty"`
	Channel      string            `json:"channel,omitempty"`
	Cursor       uint64            `json:"cursor,omitempty"`
	Consume      bool              `json:"consume,omitempty"`
	Completion   *CompletionHandle `json:"completion,omitempty"`
	Timer        *TimerWait        `json:"timer,omitempty"`

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
	Error      *SanitizedError `json:"error,omitempty"`
}

type SanitizedError struct {
	Code      string           `json:"code"`
	Category  ErrorCategory    `json:"category,omitempty"`
	Message   string           `json:"message"`
	Retryable bool             `json:"retryable,omitempty"`
	StepPath  string           `json:"stepPath,omitempty"`
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
	Kind         ProgramEventKind `json:"kind"`
	At           time.Time        `json:"at"`
	ActivityID   ActivityID       `json:"activityID,omitempty"`
	WorkItemID   WorkItemID       `json:"workItemID,omitempty"`
	ChildFrameID ProgramFrameID   `json:"childFrameID,omitempty"`
}

type ProgramFrameIntent struct {
	ID          ProgramFrameID             `json:"id,omitempty"`
	ExecutionID ExecutionID                `json:"executionID,omitempty"`
	Path        string                     `json:"path,omitempty"`
	Program     ProgramSnapshot            `json:"program"`
	Deadline    *time.Time                 `json:"deadline,omitempty"`
	Lifetime    ChildFrameLifetime         `json:"lifetime,omitempty"`
	Ownership   *ChildExecutionOwnership   `json:"ownership,omitempty"`
	Handoff     *ConversationHandoffIntent `json:"handoff,omitempty"`
}

type WorkItemIntent struct {
	ProgramPath string          `json:"programPath,omitempty"`
	Type        WorkItemType    `json:"type"`
	Automatic   bool            `json:"automatic"`
	Deadline    *time.Time      `json:"deadline,omitempty"`
	Effect      EffectContract  `json:"effect"`
	Retry       *RetryPolicy    `json:"retry,omitempty"`
	Input       WorkItemPayload `json:"input"`
}

type ProgramTransition struct {
	State                   ProgramState         `json:"state"`
	Status                  *ProgramFrameState   `json:"status,omitempty"`
	Output                  CanonicalValue       `json:"output,omitempty"`
	Error                   *SanitizedError      `json:"error,omitempty"`
	Activities              []Activity           `json:"activities,omitempty"`
	Frames                  []ProgramFrameIntent `json:"frames,omitempty"`
	WorkItems               []WorkItemIntent     `json:"workItems,omitempty"`
	CancelChildPaths        []string             `json:"cancelChildPaths,omitempty"`
	CancelChildFrameIDs     []ProgramFrameID     `json:"cancelChildFrameIDs,omitempty"`
	CancelChildExecutionIDs []ExecutionID        `json:"cancelChildExecutionIDs,omitempty"`
	ContinueParent          bool                 `json:"continueParent,omitempty"`
}

type ProgramInterpreter interface {
	ID() string
	Version() uint32
	ValidateSnapshot(snapshot ProgramSnapshot) error
	ValidateState(state ProgramState) error
	InitialState(snapshot ProgramSnapshot) (ProgramState, error)
	Advance(frame ProgramFrame, execution Execution, event ProgramEvent) (ProgramTransition, error)
}

type WorkDispatchRequest struct {
	Session   Session
	Execution Execution
	WorkItem  WorkItem
	Stream    WorkStream `json:"-"`
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

type CommandReceipt struct {
	CommandID         CommandID    `json:"commandID"`
	Accepted          bool         `json:"accepted"`
	Duplicate         bool         `json:"duplicate,omitempty"`
	SessionID         SessionID    `json:"sessionID,omitempty"`
	ExecutionID       ExecutionID  `json:"executionID,omitempty"`
	SessionRevision   Revision     `json:"sessionRevision,omitempty"`
	ExecutionRevision Revision     `json:"executionRevision,omitempty"`
	ActivityIDs       []ActivityID `json:"activityIDs,omitempty"`
	WorkItemIDs       []WorkItemID `json:"workItemIDs,omitempty"`
	WaitIDs           []WaitID     `json:"waitIDs,omitempty"`
	Message           string       `json:"message,omitempty"`
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
	Text        string         `json:"text,omitempty"`
	Attachments []Attachment   `json:"attachments,omitempty"`
	Metadata    CanonicalValue `json:"metadata,omitempty"`
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

type WorkflowProjection struct {
	SchemaVersion   string      `json:"schemaVersion"`
	SessionID       SessionID   `json:"sessionID"`
	SessionRevision Revision    `json:"sessionRevision"`
	Activities      []Activity  `json:"activities"`
	Executions      []Execution `json:"executions"`
}

func (id WorkItemID) String() string     { return string(id) }
func (id ActivityID) String() string     { return string(id) }
func (id ProgramFrameID) String() string { return string(id) }
func (id ExecutionID) String() string    { return string(id) }
func (id SessionID) String() string      { return string(id) }
func (id WaitID) String() string         { return string(id) }
func (v CanonicalValue) MarshalJSON() ([]byte, error) {
	if len(v) == 0 {
		return []byte("null"), nil
	}
	return append([]byte(nil), v...), nil
}

func (v *CanonicalValue) UnmarshalJSON(raw []byte) error {
	value, err := CanonicalValueFrom(raw)
	if err != nil {
		return err
	}
	*v = value
	return nil
}
