import type { ArtifactRef } from '@/spec/artifact';

export type SessionID = string;
export type RunID = string;
export type CommandID = string;
export type OperationID = string;
export type ToolCallID = string;
export type ToolAttemptID = string;
export type ToolResultID = string;
export type WaitID = string;
export type AppInstanceID = string;
export type AppViewID = string;
export type SessionRevision = number;

export type SessionOpaqueJSON = unknown;

export enum SessionMessageRole {
	System = 'system',
	Developer = 'developer',
	User = 'user',
	Assistant = 'assistant',
	Tool = 'tool',
}

export enum SessionRunState {
	Queued = 'queued',
	Running = 'running',
	Waiting = 'waiting',
	Paused = 'paused',
	Cancelling = 'cancelling',
	Completed = 'completed',
	Failed = 'failed',
	Cancelled = 'cancelled',
}

export enum SessionOperationState {
	Queued = 'queued',
	Preparing = 'preparing',
	Running = 'running',
	Completed = 'completed',
	Failed = 'failed',
	Interrupted = 'interrupted',
	Cancelled = 'cancelled',
}

export enum SessionToolCallState {
	Proposed = 'proposed',
	Pending = 'pending',
	EvaluationQueued = 'evaluationQueued',
	Evaluating = 'evaluating',
	Ready = 'ready',
	ExecutionQueued = 'executionQueued',
	Waiting = 'waiting',
	Running = 'running',
	Resolved = 'resolved',
	Failed = 'failed',
	Cancelled = 'cancelled',
	Observed = 'observed',
}

export enum SessionToolAttemptState {
	Queued = 'queued',
	Running = 'running',
	Completed = 'completed',
	Failed = 'failed',
	Cancelled = 'cancelled',
	Uncertain = 'uncertain',
}

export enum SessionToolResultKind {
	Success = 'success',
	Error = 'error',
	Skipped = 'skipped',
	Uncertain = 'uncertain',
}

export enum SessionToolBindingKind {
	Go = 'go',
	MCP = 'mcp',
	Skill = 'skill',
	Provider = 'provider',
	Unresolved = 'unresolved',
}

export enum SessionToolCallSource {
	Model = 'model',
	App = 'app',
}

enum SessionWaitKind {
	Approval = 'approval',
	ManualToolExecution = 'manualToolExecution',
	UserInput = 'userInput',
	Configuration = 'configuration',
	Authentication = 'authentication',
	ContinuationReview = 'continuationReview',
	Recovery = 'recovery',
}

export enum SessionWaitStatus {
	Pending = 'pending',
	Resolving = 'resolving',
	Resolved = 'resolved',
	Expired = 'expired',
	Cancelled = 'cancelled',
	Invalid = 'invalid',
}

export enum SessionWaitResolution {
	ApproveOnce = 'approveOnce',
	ApproveAlways = 'approveAlways',
	DenyOnce = 'denyOnce',
	DenyAlways = 'denyAlways',
	Continue = 'continue',
	Cancel = 'cancel',
}

export enum SessionAutomationHoldScope {
	OneBatch = 'oneBatch',
	Indefinite = 'indefinite',
}

export enum SessionAttachmentReferenceKind {
	File = 'file',
	Directory = 'directory',
	Image = 'image',
	URL = 'url',
	Generic = 'generic',
}

export enum SessionCollectionOverrideMode {
	Inherit = 'inherit',
	Add = 'add',
	Remove = 'remove',
	Replace = 'replace',
}

export enum SessionActivityOverrideMode {
	Inherit = 'inherit',
	Activate = 'activate',
	Deactivate = 'deactivate',
}

enum SessionConfigurationOwner {
	Platform = 'platform',
	Model = 'model',
	Agent = 'agent',
	Workspace = 'workspace',
	Session = 'session',
	Run = 'run',
	User = 'user',
}

enum SessionResolutionState {
	Included = 'included',
	Excluded = 'excluded',
	Unavailable = 'unavailable',
	Changed = 'changed',
	Blocked = 'blocked',
	Stale = 'stale',
}

enum SessionDiagnosticSeverity {
	Info = 'info',
	Warning = 'warning',
	Error = 'error',
}

enum SessionPreparationState {
	Pending = 'pending',
	Prepared = 'prepared',
	Failed = 'failed',
}

export enum SessionRunOutcomeKind {
	Completed = 'completed',
	Failed = 'failed',
	Cancelled = 'cancelled',
}

export enum SessionRuntimeEventKind {
	SessionCreated = 'sessionCreated',
	SessionRenamed = 'sessionRenamed',
	SessionDeleted = 'sessionDeleted',
	SessionConfigurationChanged = 'sessionConfigurationChanged',
	RunAccepted = 'runAccepted',
	RunStateChanged = 'runStateChanged',
	ModelOperationQueued = 'modelOperationQueued',
	ModelOperationStarted = 'modelOperationStarted',
	ModelOutputAppended = 'modelOutputAppended',
	ModelOperationFinished = 'modelOperationFinished',
	ToolCallProposed = 'toolCallProposed',
	ToolEvaluationStarted = 'toolEvaluationStarted',
	ToolCallsExecutionRequested = 'toolCallsExecutionRequested',
	ToolCallRetried = 'toolCallRetried',
	ToolAttemptStarted = 'toolAttemptStarted',
	ToolResultRecorded = 'toolResultRecorded',
	ToolCallFailed = 'toolCallFailed',
	WaitRequested = 'waitRequested',
	WaitResolutionRequested = 'waitResolutionRequested',
	WaitResolved = 'waitResolved',
	WaitExpired = 'waitExpired',
	ContinuationUpdated = 'continuationUpdated',
	ContinuationConsumed = 'continuationConsumed',
	AutomationChanged = 'automationChanged',
	SkillSessionChanged = 'skillSessionChanged',
	RunCompleted = 'runCompleted',
	RunFailed = 'runFailed',
	RunCancelled = 'runCancelled',
	RecoveryApplied = 'recoveryApplied',
	ModelOperationPrepared = 'modelOperationPrepared',
	AppInstanceCreated = 'appInstanceCreated',
	AppViewOpened = 'appViewOpened',
	AppContextStaged = 'appContextStaged',
}

export interface SessionArtifactSelection extends ArtifactRef {
	expectedRevision?: SessionRevision;
	expectedDigest?: string;
}

export interface SessionCommandMeta {
	id: CommandID;
	expectedSessionRevision?: SessionRevision;
	expectedRunRevision?: SessionRevision;
}

interface SessionResolvedArtifact extends ArtifactRef {
	revision?: SessionRevision;
	digest?: string;
}

interface SessionResolutionItem {
	key: string;
	kind: string;
	owner: SessionConfigurationOwner;
	state: SessionResolutionState;
	selected?: SessionArtifactSelection;
	used?: SessionResolvedArtifact;
	reason?: string;
}

interface SessionResolutionDiagnostic {
	severity: SessionDiagnosticSeverity;
	code: string;
	message: string;
	key?: string;
}

interface SessionOperationResolutionSummary {
	version: number;
	agent?: SessionResolutionItem;
	workspace?: SessionResolutionItem;
	model?: SessionResolutionItem;
	instructions?: SessionResolutionItem[];
	tools?: SessionResolutionItem[];
	skills?: SessionResolutionItem[];
	mcp?: SessionResolutionItem[];
	attachments?: SessionResolutionItem[];
	diagnostics?: SessionResolutionDiagnostic[];
}

interface SessionOperationPreparationFailureDetails {
	code: string;
	message: string;
	resolution: SessionOperationResolutionSummary;
}

export interface SessionOperationPreparation {
	state: SessionPreparationState;
	fingerprint?: string;
	preparedAt?: string;
	resolution: SessionOperationResolutionSummary;
	effectiveSkills?: ArtifactRef[];
	activeSkills?: ArtifactRef[];
}

export interface SessionOperationError {
	code: string;
	message: string;
}

export interface SessionCommandReceipt {
	commandID: CommandID;
	accepted: boolean;
	duplicate?: boolean;
	sessionID?: SessionID;
	runID?: RunID;
	sessionRevision?: SessionRevision;
	runRevision?: SessionRevision;
	operationIDs?: OperationID[];
	toolCallIDs?: ToolCallID[];
	waitIDs?: WaitID[];
	alreadyActiveToolCallIDs?: ToolCallID[];
	starterText?: string;
	message?: string;
	preparationFailure?: SessionOperationPreparationFailureDetails;
}

export interface SessionWaitingRequest {
	id: WaitID;
	kind: SessionWaitKind;
	status: SessionWaitStatus;
	operationID?: OperationID;
	toolCallID?: ToolCallID;
	summary?: SessionOpaqueJSON;
	inputSchema?: SessionOpaqueJSON;
	allowedResolutions?: SessionWaitResolution[];
	createdAt: string;
	updatedAt: string;
	expiresAt?: string;
	error?: string;
}

export interface SessionRuntimeEvent {
	sessionID: SessionID;
	runID?: RunID;
	sequence: number;
	runSequence?: number;
	commandID?: CommandID;
	operationID?: OperationID;
	toolCallID?: ToolCallID;
	toolAttemptID?: ToolAttemptID;
	waitID?: WaitID;
	kind: SessionRuntimeEventKind;
	at: string;
	data?: SessionOpaqueJSON;
}
