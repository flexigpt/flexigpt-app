import type { ArtifactRef } from '@/spec/artifact';
import type {
	MCPApprovalRule,
	MCPAppVisibility,
	MCPAuthState,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPPromptRef,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPRuntimeServerID,
	MCPServerRuntimeSnapshot,
	MCPTimestamp,
} from '@/spec/mcp';
import type {
	AppInstanceID,
	AppViewID,
	OperationID,
	RunID,
	SessionArtifactSelection,
	SessionCollectionOverrideMode,
	SessionCommandMeta,
	SessionID,
	SessionOpaqueJSON,
	SessionRevision,
	ToolCallID,
} from '@/spec/session_base';

export interface MCPDiscoveryPage<T> {
	items: T[];
	nextPageToken?: string;
}

export enum MCPContentType {
	Text = 'text',
	Image = 'image',
	Audio = 'audio',
	ResourceLink = 'resource_link',
	Resource = 'resource',
}

export enum MCPCompletionRefType {
	Resource = 'resource',
	Prompt = 'prompt',
}

export interface MCPResourceTemplateSelection extends MCPResourceTemplateRef {
	argumentValues?: Record<string, string>;
}

export interface MCPPromptSelection extends MCPPromptRef {
	argumentValues?: Record<string, string>;
}

/**
 * Runtime-discovered MCP Tool selection. This is used by Composer discovery,
 * tool output rendering, and provider-tool binding projections.
 */
export interface MCPToolSelection {
	server: MCPRuntimeServerID;
	toolName: string;
	providerToolName?: string;
	choiceID?: string;
	digest?: string;
	approvalRule?: MCPApprovalRule;
	executionMode?: MCPExecutionMode;
	appResourceUri?: string;
	visibility?: MCPAppVisibility[];
}

export interface MCPProviderToolMapping {
	server: MCPRuntimeServerID;
	providerToolName: string;
	choiceID: string;
	toolName: string;
	toolDigest: string;
	approvalRule?: MCPApprovalRule;
	executionMode?: MCPExecutionMode;
	appResourceUri?: string;
	visibility?: MCPAppVisibility[];
}

interface MCPIcon {
	src: string;
	mimeType?: string;
	sizes?: string[];
	theme?: string;
}

interface MCPResourceContents {
	uri: string;
	mimeType?: string;
	text?: string;
	blob?: number[];
	_meta?: Record<string, unknown>;
}

export interface MCPContent {
	type: MCPContentType;
	text?: string;
	data?: number[];
	mimeType?: string;
	uri?: string;
	name?: string;
	title?: string;
	description?: string;
	size?: number;
	resource?: MCPResourceContents;
	annotations?: Record<string, unknown>;
	_meta?: Record<string, unknown>;
	icons?: MCPIcon[];
}

interface MCPPromptMessage {
	role: string;
	content: MCPContent;
}

export interface MCPToolAppRenderInfo {
	resourceUri?: string;
	mimeType?: string;
	content?: MCPContent[];
	structuredContent?: SessionOpaqueJSON;
	isError?: boolean;
	allowAppInitiatedToolCalls?: boolean;
	requireApprovalForOpenLink?: boolean;
	requireApprovalForContextUpdates?: boolean;
}

export interface MCPReadResourceResponseBody {
	server: MCPRuntimeServerID;
	uri: string;
	contents?: MCPContent[];
}

export interface MCPGetPromptResponseBody {
	server: MCPRuntimeServerID;
	promptName: string;
	description?: string;
	messages?: MCPPromptMessage[];
}

export interface MCPCompletionResult {
	values?: string[];
	total?: number;
	hasMore?: boolean;
}

export interface MCPAuthStatus {
	server: MCPRuntimeServerID;
	authMode: MCPHTTPAuthMode;
	state: MCPAuthState;
	scopes?: string[];
	expiresAt?: MCPTimestamp;
	lastError?: string;
	authorizationServer?: string;
	resource?: string;
}

export interface MCPOAuthAuthorization {
	server: MCPRuntimeServerID;
	authorizationURL: string;
	expiresAt?: MCPTimestamp;
}

export interface MCPCompleteArgumentRequestBody {
	refType: MCPCompletionRefType;
	name: string;
	argumentName: string;
	argumentValue?: string;
	context?: Record<string, string>;
}

export interface MCPServerRuntimeDetails {
	ref: ArtifactRef;
	connection: MCPServerRuntimeSnapshot;
	authorization?: MCPAuthStatus;
	pendingAuthorization?: MCPOAuthAuthorization;
}

export interface MCPRuntimeServerView {
	ref: ArtifactRef;
	runtimeServerID: MCPRuntimeServerID;
	authorization?: MCPAuthStatus;
	pendingAuthorization?: MCPOAuthAuthorization;
	runtime: MCPServerRuntimeSnapshot;
}

/**
 * Session-owned MCP selection and override intent.
 *
 * These records are artifact-backed and durable. They do not carry a
 * process-local MCP runtime server ID.
 */
export enum MCPToolExposure {
	None = 'none',
	All = 'all',
	Selected = 'selected',
}

interface MCPServerToolSelection {
	toolName: string;
	expectedDigest?: string;
	approvalRule?: MCPApprovalRule;
	executionMode?: MCPExecutionMode;
}

interface MCPServerResourceSelection {
	uri: string;
	expectedDigest?: string;
}

interface MCPServerResourceTemplateSelection {
	uriTemplate: string;
	expectedDigest?: string;
	arguments?: Record<string, string>;
}

interface MCPServerPromptSelection {
	promptName: string;
	expectedDigest?: string;
	arguments?: Record<string, string>;
}

export interface MCPServerSelection {
	key?: string;
	server: SessionArtifactSelection;
	includeServerInstructions?: boolean;
	toolExposure: MCPToolExposure;
	selectedTools?: MCPServerToolSelection[];
	resources?: MCPServerResourceSelection[];
	resourceTemplates?: MCPServerResourceTemplateSelection[];
	prompts?: MCPServerPromptSelection[];
}

export interface MCPServerOverride {
	key: string;
	mode: SessionCollectionOverrideMode;
	value?: MCPServerSelection;
}

/**
 * Composer/runtime MCP selection. Runtime server IDs are process-local and
 * are converted to durable MCPServerOverride records only at Session-command
 * admission.
 */
interface MCPRuntimeServerSelection {
	server: MCPRuntimeServerID;
	snapshotDigest?: string;
	toolExposure: MCPToolExposure;
	selectedTools?: MCPToolSelection[];
	includeServerInstructions?: boolean;
}

export interface MCPConversationContext {
	servers: MCPRuntimeServerSelection[];
	resources?: MCPResourceRef[];
	resourceTemplates?: MCPResourceTemplateSelection[];
	prompts?: MCPPromptSelection[];
}

interface MCPAppContextContent {
	type: string;
	text?: string;
	uri?: string;
	name?: string;
	description?: string;
	mimeType?: string;
	data?: number[];
	displayName?: string;
}

/**
 * Durable Session/Run continuation context accepted by StageAppContext.
 */
export interface MCPAppContextUpdate {
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
	serverID: MCPRuntimeServerID;
	resourceURI?: string;
	content?: MCPAppContextContent[];
	structuredContent?: SessionOpaqueJSON;
	updatedAt?: string;
	serverDisplayName?: string;
	appDisplayName?: string;
	contextLabel?: string;
	structuredContentLabel?: string;
}

/**
 * App-side proposal shape used before the owning MCP App view stages a
 * Session-scoped MCPAppContextUpdate.
 */
export interface MCPAppModelContextUpdate {
	instanceID?: string;
	server: MCPRuntimeServerID;
	resourceUri?: string;
	viewID?: AppViewID;
	content?: MCPContent[];
	structuredContent?: SessionOpaqueJSON;
	updatedAt?: MCPTimestamp;
	rawArguments?: SessionOpaqueJSON;
}

export interface MCPAppInstance {
	id: AppInstanceID;
	sessionID: SessionID;
	runID: RunID;
	originOperationID: OperationID;
	originToolCallID: ToolCallID;
	serverID: MCPRuntimeServerID;
	resourceURI?: string;
	createdAt: string;
	views?: AppViewID[];
	unavailableReason?: string;
}

export interface MCPAppRenderTarget {
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	serverID: MCPRuntimeServerID;
	resourceURI?: string;
	sessionRevision: SessionRevision;
	runRevision: SessionRevision;
}

export interface MCPAppOpenViewCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
}

export interface MCPAppExecuteToolCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
	toolName: string;
	arguments?: SessionOpaqueJSON;
}

export interface MCPAppStageContextCommand {
	meta: SessionCommandMeta;
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
	update: MCPAppContextUpdate;
}

export interface MCPAppReadResourceRequest {
	sessionID: SessionID;
	runID: RunID;
	appInstanceID: AppInstanceID;
	viewID: AppViewID;
	resourceURI?: string;
}

export interface MCPAppResourceReadResult {
	content: SessionOpaqueJSON;
}
