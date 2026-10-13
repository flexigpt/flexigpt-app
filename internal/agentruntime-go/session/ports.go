package session

import (
	"context"
	"encoding/json"
	"time"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID() string
}

type ModelStream struct {
	OnText     func(string) error
	OnThinking func(string) error
}

type ModelExecutionRequest struct {
	SessionID      SessionID            `json:"sessionID"`
	RunID          RunID                `json:"runID"`
	OperationID    OperationID          `json:"operationID"`
	Configuration  SessionConfiguration `json:"configuration"`
	History        []TranscriptMessage  `json:"history"`
	Current        TranscriptMessage    `json:"current"`
	SkillSessionID string               `json:"skillSessionID,omitempty"`
}

type ModelExecutionResult struct {
	Usable                bool               `json:"usable"`
	Text                  string             `json:"text,omitempty"`
	Thinking              string             `json:"thinking,omitempty"`
	ProtocolOutputs       []json.RawMessage  `json:"protocolOutputs,omitempty"`
	HydratedCurrentInputs []json.RawMessage  `json:"hydratedCurrentInputs,omitempty"`
	AttachmentsHydrated   bool               `json:"attachmentsHydrated,omitempty"`
	ToolCalls             []ToolCallProposal `json:"toolCalls,omitempty"`
	Error                 *OperationError    `json:"error,omitempty"`
	Metadata              json.RawMessage    `json:"metadata,omitempty"`
	RuntimeMetadata       json.RawMessage    `json:"runtimeMetadata,omitempty"`
}

type ModelExecutor interface {
	ExecuteModel(
		ctx context.Context,
		req ModelExecutionRequest,
		s ModelStream,
	) (ModelExecutionResult, error)
}

type SubmissionPreparationRequest struct {
	SessionID            SessionID
	Configuration        SessionConfiguration
	Input                UserInput
	PersistConfiguration bool
}

type PreparedSubmission struct {
	Configuration        SessionConfiguration
	Input                UserInput
	PersistConfiguration bool
}

type SubmissionPreparer interface {
	PrepareSubmission(
		ctx context.Context,
		request SubmissionPreparationRequest,
	) (PreparedSubmission, error)
}

type ToolEvaluationDecision string

const (
	ToolEvaluationAllowed ToolEvaluationDecision = "allowed"
	ToolEvaluationDenied  ToolEvaluationDecision = "denied"
	ToolEvaluationWait    ToolEvaluationDecision = "wait"
)

type ToolWaitSpec struct {
	Kind               WaitKind         `json:"kind"`
	Summary            json.RawMessage  `json:"summary,omitempty"`
	InputSchema        json.RawMessage  `json:"inputSchema,omitempty"`
	AllowedResolutions []WaitResolution `json:"allowedResolutions,omitempty"`
	ExpiresAt          *time.Time       `json:"expiresAt,omitempty"`

	// ExecutorReference is intentionally runtime-private. The Session core
	// stores it in memory only and does not include it in snapshots/checkpoints.
	ExecutorReference string `json:"-"`
}

type ToolEvaluation struct {
	Decision      ToolEvaluationDecision `json:"decision"`
	Reason        string                 `json:"reason,omitempty"`
	Wait          *ToolWaitSpec          `json:"wait,omitempty"`
	Result        *ToolExecutionResult   `json:"result,omitempty"`
	Authorization json.RawMessage        `json:"-"`
}

type ToolExecutionRequest struct {
	SessionID      SessionID   `json:"sessionID"`
	RunID          RunID       `json:"runID"`
	ToolCall       ToolCall    `json:"toolCall"`
	Attempt        ToolAttempt `json:"attempt"`
	SkillSessionID string      `json:"skillSessionID,omitempty"`

	// Authorization is opaque executor state. The core does not interpret it.
	Authorization json.RawMessage `json:"-"`
}

type ToolExecutionResult struct {
	HasResult  bool            `json:"hasResult"`
	Kind       ToolResultKind  `json:"kind"`
	Text       string          `json:"text,omitempty"`
	Structured json.RawMessage `json:"structured,omitempty"`
	App        json.RawMessage `json:"app,omitempty"`
	Error      string          `json:"error,omitempty"`

	SkillSessionActive      []ArtifactReference `json:"skillSessionActive,omitempty"`
	SkillSessionActiveKnown bool                `json:"skillSessionActiveKnown,omitempty"`
}

type ToolWaitResolutionStatus string

const (
	ToolWaitResolutionApproved ToolWaitResolutionStatus = "approved"
	ToolWaitResolutionDenied   ToolWaitResolutionStatus = "denied"
)

type ToolWaitResolutionRequest struct {
	SessionID  SessionID       `json:"sessionID"`
	RunID      RunID           `json:"runID"`
	Wait       WaitingRequest  `json:"wait"`
	ToolCall   ToolCall        `json:"toolCall"`
	Resolution WaitResolution  `json:"resolution"`
	Input      json.RawMessage `json:"input,omitempty"`

	// ExecutorReference is the in-memory reference captured while creating the
	// wait. It is not persisted and is not visible to clients.
	ExecutorReference string `json:"-"`
}

type ToolWaitResolution struct {
	Status        ToolWaitResolutionStatus `json:"status"`
	Authorization json.RawMessage          `json:"-"`
	Result        *ToolExecutionResult     `json:"result,omitempty"`
}

type ToolExecutor interface {
	EvaluateTool(ctx context.Context, req ToolExecutionRequest) (ToolEvaluation, error)
	ExecuteTool(ctx context.Context, req ToolExecutionRequest) (ToolExecutionResult, error)
	ResolveToolWait(ctx context.Context, req ToolWaitResolutionRequest) (ToolWaitResolution, error)
}

type SkillSessionRequest struct {
	SessionID string              `json:"sessionID,omitempty"`
	Enabled   []ArtifactReference `json:"enabled,omitempty"`
	Active    []ArtifactReference `json:"active,omitempty"`
}

type SkillSessionResult struct {
	SessionID string              `json:"sessionID"`
	Active    []ArtifactReference `json:"active,omitempty"`
}

type SkillSessionController interface {
	EnsureSkillSession(ctx context.Context, req SkillSessionRequest) (SkillSessionResult, error)
	CloseSkillSession(ctx context.Context, s string) error
}

type AgentResolveRequest struct {
	SessionID            SessionID            `json:"sessionID"`
	Agent                ArtifactReference    `json:"agent"`
	CurrentConfiguration SessionConfiguration `json:"currentConfiguration"`
}

type AgentResolver interface {
	ResolveAgent(ctx context.Context, req AgentResolveRequest) (AgentApplication, error)
}

type WorkspaceSkillResolutionRequest struct {
	SessionID               SessionID            `json:"sessionID"`
	RunID                   RunID                `json:"runID"`
	Configuration           SessionConfiguration `json:"configuration"`
	RequestedWorkspaceSkills []ArtifactReference `json:"requestedWorkspaceSkills"`
}

type WorkspaceSkillResolution struct {
	EnabledWorkspaceSkills []ArtifactReference `json:"enabledWorkspaceSkills,omitempty"`
}

type WorkspaceResolver interface {
	ResolveWorkspaceSkills(
		ctx context.Context,
		req WorkspaceSkillResolutionRequest,
	) (WorkspaceSkillResolution, error)
}

type AttachmentPreparer interface {
	PrepareAttachments(
		ctx context.Context,
		refs []AttachmentReference,
	) ([]AttachmentReference, error)
}

type RuntimeEventKind string

const (
	RuntimeEventSessionCreated              RuntimeEventKind = "sessionCreated"
	RuntimeEventSessionRenamed              RuntimeEventKind = "sessionRenamed"
	RuntimeEventSessionDeleted              RuntimeEventKind = "sessionDeleted"
	RuntimeEventSessionConfigurationChanged RuntimeEventKind = "sessionConfigurationChanged"
	RuntimeEventRunAccepted                 RuntimeEventKind = "runAccepted"
	RuntimeEventRunStateChanged             RuntimeEventKind = "runStateChanged"
	RuntimeEventModelOperationQueued        RuntimeEventKind = "modelOperationQueued"
	RuntimeEventModelOperationStarted       RuntimeEventKind = "modelOperationStarted"
	RuntimeEventModelOutputAppended         RuntimeEventKind = "modelOutputAppended"
	RuntimeEventModelOperationFinished      RuntimeEventKind = "modelOperationFinished"
	RuntimeEventToolCallProposed            RuntimeEventKind = "toolCallProposed"
	RuntimeEventToolEvaluationStarted       RuntimeEventKind = "toolEvaluationStarted"
	RuntimeEventToolCallsExecutionRequested RuntimeEventKind = "toolCallsExecutionRequested"
	RuntimeEventToolCallRetried             RuntimeEventKind = "toolCallRetried"
	RuntimeEventToolAttemptStarted          RuntimeEventKind = "toolAttemptStarted"
	RuntimeEventToolResultRecorded          RuntimeEventKind = "toolResultRecorded"
	RuntimeEventToolCallFailed              RuntimeEventKind = "toolCallFailed"
	RuntimeEventWaitRequested               RuntimeEventKind = "waitRequested"
	RuntimeEventWaitResolutionRequested     RuntimeEventKind = "waitResolutionRequested"
	RuntimeEventWaitResolved                RuntimeEventKind = "waitResolved"
	RuntimeEventWaitExpired                 RuntimeEventKind = "waitExpired"
	RuntimeEventContinuationUpdated         RuntimeEventKind = "continuationUpdated"
	RuntimeEventContinuationConsumed        RuntimeEventKind = "continuationConsumed"
	RuntimeEventAutomationChanged           RuntimeEventKind = "automationChanged"
	RuntimeEventSkillSessionChanged         RuntimeEventKind = "skillSessionChanged"
	RuntimeEventRunCompleted                RuntimeEventKind = "runCompleted"
	RuntimeEventRunFailed                   RuntimeEventKind = "runFailed"
	RuntimeEventRunCancelled                RuntimeEventKind = "runCancelled"
	RuntimeEventRecoveryApplied             RuntimeEventKind = "recoveryApplied"
)

type RuntimeEvent struct {
	SessionID     SessionID        `json:"sessionID"`
	RunID         RunID            `json:"runID,omitempty"`
	Sequence      uint64           `json:"sequence"`
	RunSequence   uint64           `json:"runSequence,omitempty"`
	CommandID     CommandID        `json:"commandID,omitempty"`
	OperationID   OperationID      `json:"operationID,omitempty"`
	ToolCallID    ToolCallID       `json:"toolCallID,omitempty"`
	ToolAttemptID ToolAttemptID    `json:"toolAttemptID,omitempty"`
	WaitID        WaitID           `json:"waitID,omitempty"`
	Kind          RuntimeEventKind `json:"kind"`
	At            time.Time        `json:"at"`
	Data          json.RawMessage  `json:"data,omitempty"`
}

// runtimeEventData is intentionally closed to this package.
//
// Runtime events are transport notifications, and Data remains raw JSON at
// that final transport boundary. Inside Session, however, every event payload
// must be a declared domain type. This prevents map-shaped or arbitrary event
// payloads from becoming an accidental Session protocol.
type runtimeEventData interface {
	runtimeEventData()
}

type SessionRenamedEventData struct {
	Title string `json:"title"`
}

type RunStateChangedEventData struct {
	State RunState `json:"state"`
}

type ToolCallsExecutionRequestedEventData struct {
	CallIDs  []ToolCallID `json:"callIDs"`
	Continue bool         `json:"continue"`
}

type WaitResolutionRequestedEventData struct {
	Resolution WaitResolution `json:"resolution"`
}

type ModelOperationStateEventData struct {
	State OperationState `json:"state"`
}

type ModelOutputAppendedEventData struct {
	Thinking bool   `json:"thinking"`
	Text     string `json:"text"`
}

type ModelOperationFinishedEventData struct {
	State  OperationState  `json:"state"`
	Usable bool            `json:"usable"`
	Error  *OperationError `json:"error,omitempty"`
}

type ToolEvaluationStartedEventData struct {
	Automatic bool `json:"automatic"`
}

type ToolCallFailedEventData struct {
	Error string `json:"error"`
}

type ContinuationRevisionEventData struct {
	Revision Revision `json:"revision"`
}

type SkillSessionChangedEventData struct {
	SkillSessionID string              `json:"skillSessionID"`
	ActiveSkills   []ArtifactReference `json:"activeSkills,omitempty"`
}

type RecoveryWaitSummary struct {
	Message string `json:"message"`
}

func (Session) runtimeEventData()                  {}
func (SessionConfiguration) runtimeEventData()     {}
func (SessionRenamedEventData) runtimeEventData()  {}
func (RunStateChangedEventData) runtimeEventData() {}
func (ToolCallsExecutionRequestedEventData) runtimeEventData() {
}

func (WaitResolutionRequestedEventData) runtimeEventData() {
}

func (ModelOperationStateEventData) runtimeEventData() {
}

func (ModelOutputAppendedEventData) runtimeEventData() {
}

func (ModelOperationFinishedEventData) runtimeEventData() {
}

func (ToolEvaluationStartedEventData) runtimeEventData() {
}
func (ToolCallFailedEventData) runtimeEventData() {}
func (ContinuationRevisionEventData) runtimeEventData() {
}

func (SkillSessionChangedEventData) runtimeEventData() {
}
func (ToolCall) runtimeEventData()           {}
func (ToolAttempt) runtimeEventData()        {}
func (ToolResult) runtimeEventData()         {}
func (WaitingRequest) runtimeEventData()     {}
func (AutomationControls) runtimeEventData() {}
func (RunOutcome) runtimeEventData()         {}

type EventPublisher interface {
	PublishRuntimeEvent(ctx context.Context, e RuntimeEvent) error
}

type RuntimeCheckpointView struct {
	Reason    RuntimeEventKind `json:"reason"`
	Session   Session          `json:"session"`
	Run       *Run             `json:"run,omitempty"`
	CreatedAt time.Time        `json:"createdAt"`
}

type CheckpointSink interface {
	HandleCheckpoint(ctx context.Context, v RuntimeCheckpointView) error
}

type SessionDeletionSink interface {
	HandleSessionDeleted(ctx context.Context, sessionID SessionID) error
}

type RuntimeDiagnostic struct {
	SessionID SessionID `json:"sessionID,omitempty"`
	RunID     RunID     `json:"runID,omitempty"`
	Operation string    `json:"operation"`
	Message   string    `json:"message"`
	Cause     string    `json:"cause,omitempty"`
	At        time.Time `json:"at"`
}

type DiagnosticSink interface {
	ReportRuntimeDiagnostic(ctx context.Context, d RuntimeDiagnostic)
}

type RuntimeRestorer interface {
	RestoreSession(ctx context.Context, s Session) error
	RestoreRun(ctx context.Context, r Run) error
}

type RecoverySource interface {
	Restore(ctx context.Context, rr RuntimeRestorer) error
}

type RuntimeDependencies struct {
	Preparation SubmissionPreparer
	Model       ModelExecutor
	Tools       ToolExecutor
	Skills      SkillSessionController
	Agents      AgentResolver
	Workspace   WorkspaceResolver
	Attachments AttachmentPreparer
	Events      EventPublisher
	Checkpoints CheckpointSink
	Diagnostics DiagnosticSink
	Clock       Clock
	IDs         IDGenerator
	Recovery    RecoverySource
}
