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

export enum SessionWaitKind {
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

export enum SessionMCPToolExposure {
	None = 'none',
	All = 'all',
	Selected = 'selected',
}

export enum SessionConfigurationOwner {
	Platform = 'platform',
	Model = 'model',
	Agent = 'agent',
	Workspace = 'workspace',
	Session = 'session',
	Run = 'run',
	User = 'user',
}

export enum SessionResolutionState {
	Included = 'included',
	Excluded = 'excluded',
	Unavailable = 'unavailable',
	Changed = 'changed',
	Blocked = 'blocked',
	Stale = 'stale',
}

export enum SessionDiagnosticSeverity {
	Info = 'info',
	Warning = 'warning',
	Error = 'error',
}

export enum SessionPreparationState {
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

export interface SessionDirectCapabilitySelection {
	providerIdentity: string;
	providerLocalID: string;
	name: string;
	type: string;
}

export interface SessionCapabilitySelection {
	artifact?: SessionArtifactSelection;
	direct?: SessionDirectCapabilitySelection;
}

export interface SessionArtifactOverride {
	key: string;
	mode: SessionCollectionOverrideMode;
	value?: SessionArtifactSelection;
}

export interface SessionInstructionSelection {
	key?: string;
	artifact?: SessionArtifactSelection;
	text?: string;
	label?: string;
}

export interface SessionInstructionOverride {
	key: string;
	mode: SessionCollectionOverrideMode;
	value?: SessionInstructionSelection;
}

export interface SessionToolSelectionIntent {
	key?: string;
	capability: SessionCapabilitySelection;
	autoExecute?: boolean;
	userArguments?: SessionOpaqueJSON;
}

export interface SessionToolOverride {
	key: string;
	mode: SessionCollectionOverrideMode;
	value?: SessionToolSelectionIntent;
}

export interface SessionActiveSkillOverride {
	key: string;
	mode: SessionActivityOverrideMode;
	value?: SessionArtifactSelection;
}

export interface SessionMCPToolSelection {
	toolName: string;
	expectedDigest?: string;
	approvalRule?: string;
	executionMode?: string;
}

export interface SessionMCPResourceSelection {
	uri: string;
	expectedDigest?: string;
}

export interface SessionMCPResourceTemplateSelection {
	uriTemplate: string;
	expectedDigest?: string;
	arguments?: Record<string, string>;
}

export interface SessionMCPPromptSelection {
	promptName: string;
	expectedDigest?: string;
	arguments?: Record<string, string>;
}

export interface SessionMCPServerSelection {
	key?: string;
	server: SessionArtifactSelection;
	includeServerInstructions?: boolean;
	toolExposure: SessionMCPToolExposure;
	selectedTools?: SessionMCPToolSelection[];
	resources?: SessionMCPResourceSelection[];
	resourceTemplates?: SessionMCPResourceTemplateSelection[];
	prompts?: SessionMCPPromptSelection[];
}

export interface SessionMCPServerOverride {
	key: string;
	mode: SessionCollectionOverrideMode;
	value?: SessionMCPServerSelection;
}

export interface SessionWorkspaceSelectionIntent {
	workspace: SessionArtifactSelection;
	context?: SessionArtifactOverride[];
	skills?: SessionArtifactOverride[];
	mcp?: SessionMCPServerOverride[];
}

export interface SessionModelReasoningOverride {
	type?: string;
	level?: string;
	tokens?: number;
	summaryStyle?: string;
	context?: string;
	mode?: string;
}

export interface SessionModelCacheControlOverride {
	kind?: string;
	ttl?: string;
	key?: string;
}

export interface SessionModelOutputJSONSchemaOverride {
	name?: string;
	description?: string;
	schema?: SessionOpaqueJSON;
	strict?: boolean;
}

export interface SessionModelOutputFormatOverride {
	kind?: string;
	jsonSchema?: SessionModelOutputJSONSchemaOverride;
}

export interface SessionModelOutputOverride {
	verbosity?: string;
	format?: SessionModelOutputFormatOverride;
}

export enum SessionModelDefaultField {
	AdapterParameters = 'adapterParameters',
	CacheControl = 'cacheControl',
	MaxOutputTokens = 'maxOutputTokens',
	MaxPromptTokens = 'maxPromptTokens',
	Output = 'output',
	Reasoning = 'reasoning',
	StopSequences = 'stopSequences',
	Stream = 'stream',
	SystemPrompt = 'systemPrompt',
	Temperature = 'temperature',
	TimeoutMS = 'timeoutMS',
}

export interface SessionModelDefaultsOverride {
	stream?: boolean;
	maxPromptTokens?: number;
	maxOutputTokens?: number;
	temperature?: number;
	systemPrompt?: string;
	timeoutMS?: number;
	reasoning?: SessionModelReasoningOverride;
	cacheControl?: SessionModelCacheControlOverride;
	output?: SessionModelOutputOverride;
	stopSequences?: string[];
	adapterParameters?: Record<string, SessionOpaqueJSON>;
	clear?: SessionModelDefaultField[];
}

export interface SessionHistorySelection {
	all?: boolean;
	includePreviousUserTurns?: number;
}

export interface SessionConfiguration {
	agent?: SessionArtifactSelection;
	workspace?: SessionWorkspaceSelectionIntent;
	model?: SessionArtifactSelection;
	modelOverrides: SessionModelDefaultsOverride;
	instructions?: SessionInstructionOverride[];
	tools?: SessionToolOverride[];
	skills?: SessionArtifactOverride[];
	activeSkills?: SessionActiveSkillOverride[];
	mcp?: SessionMCPServerOverride[];
	history: SessionHistorySelection;
	metadata?: SessionOpaqueJSON;
}

export interface SessionLocalAttachmentReference {
	path?: string;
	originalPath?: string;
	name?: string;
	exists: boolean;
	isDirectory?: boolean;
	size?: number;
	originalSize?: number;
	modifiedAt?: string;
	originalModifiedAt?: string;
	mimeType?: string;
	width?: number;
	height?: number;
	format?: string;
}

export interface SessionURLAttachmentReference {
	url: string;
	normalized?: string;
	original?: string;
}

export interface SessionGenericAttachmentReference {
	handle: string;
	originalHandle?: string;
}

export interface SessionDirectoryAttachmentReference {
	path: string;
	originalPath?: string;
	name?: string;
	scannedFileCount?: number;
	hasMore?: boolean;
	expanded?: SessionAttachmentReference[];
}

export interface SessionAttachmentReference {
	kind: SessionAttachmentReferenceKind;
	label?: string;
	mode?: string;
	availableModes?: string[];
	file?: SessionLocalAttachmentReference;
	image?: SessionLocalAttachmentReference;
	url?: SessionURLAttachmentReference;
	generic?: SessionGenericAttachmentReference;
	directory?: SessionDirectoryAttachmentReference;
}

export interface SessionAppContextContent {
	type: string;
	text?: string;
	uri?: string;
	name?: string;
	description?: string;
	mimeType?: string;
}

export interface SessionAppContextUpdate {
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
	serverID: string;
	resourceURI?: string;
	content?: SessionAppContextContent[];
	structuredContent?: SessionOpaqueJSON;
	updatedAt?: string;
}

export interface SessionUserInput {
	text?: string;
	attachments?: SessionAttachmentReference[];
	mcpAppContext?: SessionAppContextUpdate[];
	metadata?: SessionOpaqueJSON;
}

export interface SessionContinuationInput {
	revision: SessionRevision;
	consumedRevision?: SessionRevision;
	text?: string;
	attachments?: SessionAttachmentReference[];
	mcpAppContext?: SessionAppContextUpdate[];
	configuration?: SessionConfiguration;
	metadata?: SessionOpaqueJSON;
	updatedAt?: string;
}

export interface SessionCommandMeta {
	id: CommandID;
	expectedSessionRevision?: SessionRevision;
	expectedRunRevision?: SessionRevision;
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
}

export interface SessionUserMessageSkillTemplateCommand {
	sessionID?: SessionID;
	skill: SessionArtifactSelection;
	arguments?: Record<string, string>;
}

export interface SessionUserMessageSkillTemplateResult {
	skill: ArtifactRef;
	text: string;
}

export interface SessionRunLimits {
	maxModelOperations?: number;
	maxToolAttempts?: number;
}

export interface SessionToolBinding {
	kind: SessionToolBindingKind;
	data?: SessionOpaqueJSON;
}

export interface SessionToolResult {
	id: ToolResultID;
	attemptID?: ToolAttemptID;
	callID: ToolCallID;
	providerCallID?: string;
	name: string;
	choiceID?: string;
	type?: string;
	kind: SessionToolResultKind;
	text?: string;
	structured?: SessionOpaqueJSON;
	app?: SessionOpaqueJSON;
	error?: string;
	appInstanceID?: AppInstanceID;
	createdAt: string;
}

export interface SessionToolAttempt {
	id: ToolAttemptID;
	callID: ToolCallID;
	state: SessionToolAttemptState;
	automatic: boolean;
	startedAt?: string;
	finishedAt?: string;
	error?: string;
	uncertain?: boolean;
}

export interface SessionToolCall {
	id: ToolCallID;
	originOperationID: OperationID;
	providerCallID?: string;
	name: string;
	choiceID?: string;
	type?: string;
	arguments?: SessionOpaqueJSON;
	binding: SessionToolBinding;
	required: boolean;
	providerManaged?: boolean;
	automatic?: boolean;
	source?: SessionToolCallSource;
	appInstanceID?: AppInstanceID;
	state: SessionToolCallState;
	attempts?: SessionToolAttempt[];
	results?: SessionToolResult[];
	selectedResultID?: ToolResultID;
	waitID?: WaitID;
	lastFailure?: string;
	consumedByOperationID?: OperationID;
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

export interface SessionAutomationHold {
	enabled: boolean;
	scope?: SessionAutomationHoldScope;
	targetOperationID?: OperationID;
	setByCommandID?: CommandID;
}

export interface SessionAutomationControls {
	automaticTools: SessionAutomationHold;
	automaticContinuation: SessionAutomationHold;
	paused: boolean;
	cancellationRequested: boolean;
}

export interface SessionSkillSessionState {
	id?: string;
	generation: SessionRevision;
	establishedGeneration?: SessionRevision;
	enabled?: ArtifactRef[];
	active?: ArtifactRef[];
}

export interface SessionResolvedArtifact extends ArtifactRef {
	revision?: SessionRevision;
	digest?: string;
}

export interface SessionResolutionItem {
	key: string;
	kind: string;
	owner: SessionConfigurationOwner;
	state: SessionResolutionState;
	selected?: SessionArtifactSelection;
	used?: SessionResolvedArtifact;
	reason?: string;
}

export interface SessionResolutionDiagnostic {
	severity: SessionDiagnosticSeverity;
	code: string;
	message: string;
	key?: string;
}

export interface SessionOperationResolutionSummary {
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

export interface SessionTranscriptMessage {
	id: string;
	createdAt: string;
	role: SessionMessageRole;
	status?: string;
	text?: string;
	attachments?: SessionAttachmentReference[];
	attachmentsHydrated?: boolean;
	protocolInputs?: SessionOpaqueJSON[];
	protocolOutputs?: SessionOpaqueJSON[];
	toolResults?: SessionToolResult[];
	mcpAppContext?: SessionAppContextUpdate[];
	operationID?: OperationID;
	error?: SessionOperationError;
	metadata?: SessionOpaqueJSON;
	runtimeMetadata?: SessionOpaqueJSON;
}

export interface SessionModelOperationInput {
	configuration: SessionConfiguration;
	history: SessionTranscriptMessage[];
	current: SessionTranscriptMessage;
	continuationRevision?: SessionRevision;
}

export interface SessionModelOperation {
	id: OperationID;
	state: SessionOperationState;
	startedAt?: string;
	finishedAt?: string;
	request: SessionModelOperationInput;
	streamedText?: string;
	streamedThinking?: string;
	protocolOutputs?: SessionOpaqueJSON[];
	toolCallIDs?: ToolCallID[];
	error?: SessionOperationError;
	preparation: SessionOperationPreparation;
	usable: boolean;
	metadata?: SessionOpaqueJSON;
	runtimeMetadata?: SessionOpaqueJSON;
}

export interface SessionDriverState {
	kind: string;
	version: number;
	data?: SessionOpaqueJSON;
}

export interface SessionRunUsage {
	modelOperations: number;
	toolAttempts: number;
}

export interface SessionRunOutcome {
	kind: SessionRunOutcomeKind;
	code?: string;
	message?: string;
}

export interface SessionAppInstance {
	id: AppInstanceID;
	sessionID: SessionID;
	runID: RunID;
	originOperationID: OperationID;
	originToolCallID: ToolCallID;
	serverID: string;
	resourceURI?: string;
	createdAt: string;
	views?: AppViewID[];
	unavailableReason?: string;
}

/**
 * Runtime-only identity required to route an MCP App through its originating
 * Session and Run. It is intentionally attached only to UI projections and is
 * never persisted as provider or MCP binding data.
 */
export interface SessionAppRenderTarget {
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	serverID: string;
	resourceURI?: string;
	sessionRevision: SessionRevision;
	runRevision: SessionRevision;
}

export interface SessionRun {
	id: RunID;
	sessionID: SessionID;
	parentRunID?: RunID;
	rootRunID?: RunID;
	revision: SessionRevision;
	state: SessionRunState;
	createdAt: string;
	updatedAt: string;
	driver: SessionDriverState;
	configuration: SessionConfiguration;
	skillSession: SessionSkillSessionState;
	controls: SessionAutomationControls;
	limits: SessionRunLimits;
	usage: SessionRunUsage;
	continuation: SessionContinuationInput;
	modelOperations?: SessionModelOperation[];
	toolCalls?: SessionToolCall[];
	waitingRequests?: SessionWaitingRequest[];
	apps?: SessionAppInstance[];
	requestedCallIDs?: ToolCallID[];
	continuationRequested?: boolean;
	activeModelOperationID?: OperationID;
	outcome?: SessionRunOutcome;
	eventSequence: number;
}

export interface Session {
	id: SessionID;
	revision: SessionRevision;
	title?: string;
	createdAt: string;
	updatedAt: string;
	configuration: SessionConfiguration;
	transcript?: SessionTranscriptMessage[];
	runIDs?: RunID[];
	eventSequence: number;
}

export interface SessionSnapshot {
	session: Session;
	runs: SessionRun[];
	sequence: number;
}

export interface SessionRunSnapshot {
	sessionID: SessionID;
	sessionRevision: SessionRevision;
	run: SessionRun;
}

export interface SessionRuntimeListItem {
	id: SessionID;
	revision: SessionRevision;
	title?: string;
	updatedAt: string;
	activeRuns?: RunID[];
}

export interface SessionAppResourceReadResult {
	content: SessionOpaqueJSON;
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

/**
 * Flat snapshot-observer result. Event payloads are notification metadata;
 * consumers render only the authoritative snapshot.
 */
export enum SessionSnapshotObservationKind {
	Snapshot = 'snapshot',
	Deleted = 'deleted',
	Error = 'error',
}

export interface SessionSnapshotObservation {
	kind: SessionSnapshotObservationKind;
	snapshot?: SessionSnapshot;
	event?: SessionRuntimeEvent;
	error?: unknown;
}

export enum SessionRunObservationKind {
	Snapshot = 'snapshot',
	Unavailable = 'unavailable',
	Deleted = 'deleted',
	Error = 'error',
}

export interface SessionRunObservation {
	kind: SessionRunObservationKind;
	snapshot?: SessionRunSnapshot;
	sessionID?: SessionID;
	runID?: RunID;
	event?: SessionRuntimeEvent;
	error?: unknown;
}

export type SessionSnapshotObserver = (observation: SessionSnapshotObservation) => void;

export type SessionRunObserver = (observation: SessionRunObservation) => void;

export interface CreateSessionCommand {
	meta: SessionCommandMeta;
	sessionID?: SessionID;
	title?: string;
	configuration: SessionConfiguration;
}

export interface UpdateSessionConfigurationCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	configuration: SessionConfiguration;
}

export interface RenameSessionCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	title: string;
}

export interface DeleteSessionCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
}

export interface StartRunCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID?: RunID;
	input: SessionUserInput;
	configuration?: SessionConfiguration;
	limits?: SessionRunLimits;
}

export interface UpdateContinuationInputCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	expectedInputRevision?: SessionRevision;
	input: SessionContinuationInput;
}

export interface ContinueRunCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
}

export interface ExecuteCallsCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	callIDs?: ToolCallID[];
	continue: boolean;
}

export interface RetryCallCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	callID: ToolCallID;
}

export interface SkipCallCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	callID: ToolCallID;
}

export interface CompleteCallAsErrorCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	callID: ToolCallID;
	error: string;
}

export interface ResolveWaitCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	waitID: WaitID;
	resolution: SessionWaitResolution;
	input?: SessionOpaqueJSON;
}

export interface SetAutomationHoldCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	tools: boolean;
	continuation: boolean;
	enabled: boolean;
	scope?: SessionAutomationHoldScope;
}

export interface PauseRunCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
}

export interface ResumeRunCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
}

export interface CancelRunCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	reason?: string;
}

export interface EditAndResendCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	replaceMessageID: string;
	runID?: RunID;
	input: SessionUserInput;
	configuration?: SessionConfiguration;
	limits?: SessionRunLimits;
}

export interface OpenAppViewCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
}

export interface ExecuteAppToolCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
	toolName: string;
	arguments?: SessionOpaqueJSON;
}

export interface StageAppContextCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
	update: SessionAppContextUpdate;
}

export interface ReadAppResourceRequest {
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
	resourceURI?: string;
}

export function emptySessionConfiguration(): SessionConfiguration {
	return {
		modelOverrides: {},
		history: {},
	};
}
