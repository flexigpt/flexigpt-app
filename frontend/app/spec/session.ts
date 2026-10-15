import type { ArtifactRef } from '@/spec/artifact';
import type { Attachment } from '@/spec/attachment';
import type {
	InferenceError,
	InferenceUsage,
	InputUnion,
	ModelParam,
	OutputUnion,
	ReasoningContent,
	RoleEnum,
	Status,
	ToolChoice,
	UIToolCall,
	UIToolOutput,
	URLCitation,
} from '@/spec/inference';
import type {
	AppInstanceID,
	CommandID,
	OperationID,
	RunID,
	SessionActivityOverrideMode,
	SessionArtifactSelection,
	SessionAttachmentReferenceKind,
	SessionAutomationHoldScope,
	SessionCollectionOverrideMode,
	SessionCommandMeta,
	SessionConfigurationOwner,
	SessionDiagnosticSeverity,
	SessionID,
	SessionMessageRole,
	SessionOpaqueJSON,
	SessionOperationState,
	SessionPreparationState,
	SessionResolutionState,
	SessionRevision,
	SessionRunOutcomeKind,
	SessionRunState,
	SessionRuntimeEvent,
	SessionToolAttemptState,
	SessionToolBindingKind,
	SessionToolCallSource,
	SessionToolCallState,
	SessionToolResultKind,
	SessionWaitingRequest,
	SessionWaitResolution,
	ToolAttemptID,
	ToolCallID,
	ToolResultID,
	WaitID,
} from '@/spec/session_base';
import type {
	MCPAppContextUpdate,
	MCPAppInstance,
	MCPAppModelContextUpdate,
	MCPConversationContext,
	MCPProviderToolMapping,
	MCPServerOverride,
} from '@/spec/session_mcp';
import type { ToolSelection, ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import type { WorkspaceConversationSelection, WorkspaceConversationUsage } from '@/spec/workspace';

interface SessionDirectCapabilitySelection {
	providerIdentity: string;
	providerLocalID: string;
	name: string;
	type: string;
}

interface SessionCapabilitySelection {
	artifact?: SessionArtifactSelection;
	direct?: SessionDirectCapabilitySelection;
}

export interface SessionArtifactOverride {
	key: string;
	mode: SessionCollectionOverrideMode;
	value?: SessionArtifactSelection;
}

interface SessionInstructionSelection {
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

interface SessionToolSelectionIntent {
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

interface SessionModelReasoningOverride {
	type?: string;
	level?: string;
	tokens?: number;
	summaryStyle?: string;
	context?: string;
	mode?: string;
}

interface SessionModelCacheControlOverride {
	kind?: string;
	ttl?: string;
	key?: string;
}

interface SessionModelOutputJSONSchemaOverride {
	name?: string;
	description?: string;
	schema?: SessionOpaqueJSON;
	strict?: boolean;
}

interface SessionModelOutputFormatOverride {
	kind?: string;
	jsonSchema?: SessionModelOutputJSONSchemaOverride;
}

interface SessionModelOutputOverride {
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

interface SessionHistorySelection {
	all?: boolean;
	includePreviousUserTurns?: number;
}

export interface SessionWorkspaceSelectionIntent {
	workspace: SessionArtifactSelection;
	context?: SessionArtifactOverride[];
	skills?: SessionArtifactOverride[];
	mcp?: MCPServerOverride[];
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
	mcp?: MCPServerOverride[];
	history: SessionHistorySelection;
	metadata?: SessionOpaqueJSON;
}

interface SessionLocalAttachmentReference {
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

interface SessionURLAttachmentReference {
	url: string;
	normalized?: string;
	original?: string;
}

interface SessionGenericAttachmentReference {
	handle: string;
	originalHandle?: string;
}

interface SessionDirectoryAttachmentReference {
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

export interface SessionUserInput {
	text?: string;
	attachments?: SessionAttachmentReference[];
	mcpAppContext?: MCPAppContextUpdate[];
	metadata?: SessionOpaqueJSON;
}

interface SessionContinuationInput {
	revision: SessionRevision;
	consumedRevision?: SessionRevision;
	text?: string;
	attachments?: SessionAttachmentReference[];
	mcpAppContext?: MCPAppContextUpdate[];
	configuration?: SessionConfiguration;
	metadata?: SessionOpaqueJSON;
	updatedAt?: string;
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

export interface SessionInstructionSkillPreviewRequest {
	skill: SessionArtifactSelection;
	arguments?: Record<string, string>;
}

export interface SessionInstructionSkillPreviewResult {
	skill: ArtifactRef;
	text: string;
}

interface SessionRunLimits {
	maxModelOperations?: number;
	maxToolAttempts?: number;
}

interface SessionToolBinding {
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

interface SessionToolAttempt {
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

interface SessionAutomationHold {
	enabled: boolean;
	scope?: SessionAutomationHoldScope;
	targetOperationID?: OperationID;
	setByCommandID?: CommandID;
}

interface SessionAutomationControls {
	automaticTools: SessionAutomationHold;
	automaticContinuation: SessionAutomationHold;
	paused: boolean;
	cancellationRequested: boolean;
}

interface SessionSkillSessionState {
	id?: string;
	generation: SessionRevision;
	establishedGeneration?: SessionRevision;
	enabled?: ArtifactRef[];
	active?: ArtifactRef[];
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

interface SessionOperationPreparation {
	state: SessionPreparationState;
	fingerprint?: string;
	preparedAt?: string;
	resolution: SessionOperationResolutionSummary;
	effectiveSkills?: ArtifactRef[];
	activeSkills?: ArtifactRef[];
}

interface SessionOperationError {
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
	mcpAppContext?: MCPAppContextUpdate[];
	operationID?: OperationID;
	error?: SessionOperationError;
	metadata?: SessionOpaqueJSON;
	runtimeMetadata?: SessionOpaqueJSON;
}

interface SessionModelOperationInput {
	configuration: SessionConfiguration;
	history: SessionTranscriptMessage[];
	current: SessionTranscriptMessage;
	continuationRevision?: SessionRevision;
}

interface SessionModelOperation {
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

interface SessionDriverState {
	kind: string;
	version: number;
	data?: SessionOpaqueJSON;
}

interface SessionRunUsage {
	modelOperations: number;
	toolAttempts: number;
}

export interface SessionRunOutcome {
	kind: SessionRunOutcomeKind;
	code?: string;
	message?: string;
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
	apps?: MCPAppInstance[];
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

/**
 * Session-derived chat projection.
 *
 * This is a frontend rendering model derived from Session snapshots. It is
 * not a durable Conversation aggregate and it has no direct persistence API.
 */
export interface ConversationMessage {
	id: string;
	createdAt: string;
	role: RoleEnum;
	status: Status;

	modelParam?: ModelParam;
	modelRef?: ArtifactRef;
	inputs?: InputUnion[];
	outputs?: OutputUnion[];

	toolChoices?: ToolChoice[];
	toolSelections?: ToolSelection[];
	mcpContext?: MCPConversationContext;
	mcpToolMappings?: MCPProviderToolMapping[];
	mcpAppContextUpdates?: MCPAppModelContextUpdate[];
	attachments?: Attachment[];
	enabledSkillRefs?: ArtifactRef[];
	activeSkillRefs?: ArtifactRef[];
	workspaceSelection?: WorkspaceConversationSelection;
	workspaceUsage?: WorkspaceConversationUsage;

	usage?: InferenceUsage;
	error?: InferenceError;
	meta?: Record<string, unknown>;
	debugDetails?: unknown;

	uiContent: string;
	uiDebugDetails?: string;
	uiReasoningContents?: ReasoningContent[];
	uiToolCalls?: UIToolCall[];
	uiToolOutputs?: UIToolOutput[];
	uiToolChoices?: ToolStoreChoice[];
	uiToolSelectionIssues?: ToolSelectionIssue[];
	uiCitations?: URLCitation[];
}

export interface Conversation {
	schemaVersion: string;
	id: string;
	title: string;
	createdAt: string;
	modifiedAt: string;
	messages: ConversationMessage[];
	meta?: Record<string, unknown>;
}

export interface RestorableConversationContext {
	modelRef?: ArtifactRef;
	modelParam?: ModelParam;
	toolChoices: ToolStoreChoice[];
	toolSelectionIssues?: ToolSelectionIssue[];
	mcpContext?: MCPConversationContext;
	mcpAppContextUpdates?: MCPAppModelContextUpdate[];
	webSearchChoices: ToolStoreChoice[];
	enabledSkillRefs: ArtifactRef[];
	activeSkillRefs: ArtifactRef[];
	workspaceSelection?: WorkspaceConversationSelection;
}

export function emptySessionConfiguration(): SessionConfiguration {
	return {
		modelOverrides: {},
		history: {},
	};
}
