import type { ArtifactRef } from '@/spec/artifact';
import type { MCPHTTPAuthMode } from '@/spec/mcp';
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
import { MCPApprovalRule, MCPExecutionMode } from '@/spec/mcp';

/**
 * Process-local MCP runtime identity.
 *
 * This is opaque to frontend callers. It is obtained through MCP management
 * resolution and is never constructed from an ArtifactRef in the frontend.
 */
export type MCPRuntimeServerID = string;
export type MCPRuntimeCatalogID = string;
export type MCPTimestamp = string;

export interface MCPDiscoveryPage<T> {
	items: T[];
	nextPageToken?: string;
}

export enum MCPServerStatus {
	Disabled = 'disabled',
	Disconnected = 'disconnected',
	Connecting = 'connecting',
	Ready = 'ready',
	Error = 'error',
}

export enum MCPAuthHealthState {
	NotRequired = 'notRequired',
	NotConfigured = 'notConfigured',
	AuthorizationNeeded = 'authorizationNeeded',
	AuthorizationPending = 'authorizationPending',
	Authorized = 'authorized',
	Expired = 'expired',
	InsufficientScope = 'insufficientScope',
	Error = 'error',
}

export enum MCPToolRisk {
	Unknown = 'unknown',
	Read = 'read',
	Write = 'write',
	Destructive = 'destructive',
	OpenWorld = 'openWorld',
}

export enum MCPTaskSupport {
	Forbidden = 'forbidden',
	Optional = 'optional',
	Required = 'required',
}

export enum MCPInvocationSource {
	Model = 'model',
	User = 'user',
	App = 'app',
}

export enum MCPApprovalDecision {
	Allowed = 'allowed',
	Denied = 'denied',
	ApprovalRequired = 'approvalRequired',
}

export enum MCPApprovalResolution {
	AllowOnce = 'allowOnce',
	AllowAlways = 'allowAlways',
	DenyOnce = 'denyOnce',
	DenyAlways = 'denyAlways',
}

export enum MCPContentType {
	Text = 'text',
	Image = 'image',
	Audio = 'audio',
	ResourceLink = 'resource_link',
	Resource = 'resource',
}

export enum MCPAppVisibility {
	Model = 'model',
	App = 'app',
}

export enum MCPCompletionRefType {
	Resource = 'resource',
	Prompt = 'prompt',
}

export interface MCPServerCapabilitiesSummary {
	tools?: boolean;
	toolsListChanged?: boolean;
	resources?: boolean;
	resourcesSubscribe?: boolean;
	resourcesListChanged?: boolean;
	prompts?: boolean;
	promptsListChanged?: boolean;
	completions?: boolean;
	experimental?: Record<string, unknown>;
	extensions?: Record<string, unknown>;
}

export interface MCPToolAnnotations {
	destructiveHint?: boolean;
	idempotentHint: boolean;
	openWorldHint?: boolean;
	readOnlyHint: boolean;
	title?: string;
}

export interface MCPToolAppInfo {
	resourceUri?: string;
	visibility?: MCPAppVisibility[];
}

export interface MCPToolCapability {
	server: MCPRuntimeServerID;
	toolName: string;
	providerToolName: string;
	choiceID: string;

	title?: string;
	displayName: string;
	description?: string;

	inputSchema?: Record<string, unknown>;
	outputSchema?: Record<string, unknown>;

	annotations?: MCPToolAnnotations;
	inferredRisk: MCPToolRisk;
	approvalRule: MCPApprovalRule;
	executionMode: MCPExecutionMode;
	taskSupport: MCPTaskSupport;
	app?: MCPToolAppInfo;

	digest: string;
	enabled: boolean;
	stale?: boolean;
}

export interface MCPArgumentDefinition {
	name: string;
	title?: string;
	description?: string;
	required?: boolean;
}

export interface MCPResourceRef {
	server: MCPRuntimeServerID;
	uri: string;
	name?: string;
	title?: string;
	displayName: string;
	description?: string;
	mimeType?: string;
	size?: number;
	annotations?: Record<string, unknown>;
	digest?: string;
}

export interface MCPResourceTemplateRef {
	server: MCPRuntimeServerID;
	uriTemplate: string;
	name?: string;
	title?: string;
	displayName: string;
	description?: string;
	mimeType?: string;
	arguments?: Record<string, MCPArgumentDefinition>;
	annotations?: Record<string, unknown>;
	digest?: string;
}

export interface MCPPromptRef {
	server: MCPRuntimeServerID;
	promptName: string;
	title?: string;
	displayName: string;
	description?: string;
	arguments?: Record<string, MCPArgumentDefinition>;
	digest?: string;
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

export interface MCPIcon {
	src: string;
	mimeType?: string;
	sizes?: string[];
	theme?: string;
}

export interface MCPResourceContents {
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

export interface MCPPromptMessage {
	role: string;
	content: MCPContent;
}

export interface InvokeMCPToolRequestBody {
	source: MCPInvocationSource;
	toolName: string;
	providerToolName?: string;
	choiceID?: string;
	toolDigest?: string;
	arguments?: Record<string, unknown>;
	approvalID?: string;
	approvalToken?: string;
	conversationID?: string;
	messageID?: string;
	toolUseID?: string;
	appInstanceID?: string;
}

export interface MCPToolCallProvenance {
	server: MCPRuntimeServerID;
	catalog: MCPRuntimeCatalogID;
	serverDisplayName?: string;
	toolName: string;
	providerToolName: string;
	toolDigest?: string;
	choiceID?: string;
	toolUseID?: string;
	approvalID?: string;
	appResourceUri?: string;
	appInstanceID?: string;
}

export interface MCPToolAppRenderInfo {
	resourceUri?: string;
	mimeType?: string;
	content?: MCPContent[];
	structuredContent?: SessionOpaqueJSON;
	isError?: boolean;
}

export interface InvokeMCPToolResponseBody {
	server: MCPRuntimeServerID;
	toolName: string;
	providerToolName?: string;
	content?: MCPContent[];
	structuredContent?: SessionOpaqueJSON;
	isError?: boolean;
	provenance: MCPToolCallProvenance;
	app?: MCPToolAppRenderInfo;
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

export interface MCPApprovalSummary {
	server: MCPRuntimeServerID;
	serverDisplayName?: string;
	source: MCPInvocationSource;
	appInstanceID?: string;
	toolName: string;
	toolDigest?: string;
	risk: MCPToolRisk;
	arguments?: SessionOpaqueJSON;
}

export interface MCPApprovalEvaluation {
	decision: MCPApprovalDecision;
	reason?: string;
	approvalID?: string;
	expiresAt?: MCPTimestamp;
	summary?: MCPApprovalSummary;
}

export interface MCPApprovalResolutionResult {
	approvalID: string;
	resolution: MCPApprovalResolution;
	decision: MCPApprovalDecision;
	rememberedForSession?: boolean;
	token?: string;
	expiresAt?: string;
}

export enum MCPAuthState {
	NotRequired = 'notRequired',
	Required = 'required',
	Authorized = 'authorized',
	Expired = 'expired',
	InsufficientScope = 'insufficientScope',
	Error = 'error',
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

export interface MCPAuthHealth {
	server: MCPRuntimeServerID;
	authMode: MCPHTTPAuthMode;
	state: MCPAuthHealthState;
	configured: boolean;
	resource?: string;
	scopes?: string[];
	expiresAt?: MCPTimestamp;
	authorizationPending?: boolean;
	authorizationURL?: string;
	authorizationExpiresAt?: MCPTimestamp;
	oauthRedirectURL?: string;
	oauthLoopbackListenAddr?: string;
	oauthLoopbackReady?: boolean;
	oauthLoopbackError?: string;
	lastError?: string;
}

export interface MCPServerRuntimeSnapshot {
	server: MCPRuntimeServerID;
	catalog: MCPRuntimeCatalogID;
	status: MCPServerStatus;
	generation: number;
	negotiatedProtocolVersion?: string;
	serverInfo?: {
		name?: string;
		version?: string;
	};
	serverCapabilities?: MCPServerCapabilitiesSummary;
	instructions?: string;
	lastError?: string;
	lastConnectedAt?: string;
	lastSyncedAt?: string;
	toolCount: number;
	resourceCount: number;
	resourceTemplateCount: number;
	promptCount: number;
	snapshotDigest?: string;
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

export interface MCPServerToolSelection {
	toolName: string;
	expectedDigest?: string;
	approvalRule?: MCPApprovalRule;
	executionMode?: MCPExecutionMode;
}

export interface MCPServerResourceSelection {
	uri: string;
	expectedDigest?: string;
}

export interface MCPServerResourceTemplateSelection {
	uriTemplate: string;
	expectedDigest?: string;
	arguments?: Record<string, string>;
}

export interface MCPServerPromptSelection {
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
export interface MCPRuntimeServerSelection {
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

export interface MCPAppContextContent {
	type: string;
	text?: string;
	uri?: string;
	name?: string;
	description?: string;
	mimeType?: string;
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
}

/**
 * App-side proposal shape used before the owning MCP App view stages a
 * Session-scoped MCPAppContextUpdate.
 */
export interface MCPAppModelContextUpdate {
	instanceID?: string;
	server: MCPRuntimeServerID;
	resourceUri?: string;
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

export function isMCPAppVisibility(value: unknown): value is MCPAppVisibility {
	return typeof value === 'string' && Object.values(MCPAppVisibility).includes(value as MCPAppVisibility);
}

export function isMCPExecutionMode(value: unknown): value is MCPExecutionMode {
	return typeof value === 'string' && Object.values(MCPExecutionMode).includes(value as MCPExecutionMode);
}

export function isMCPApprovalRule(value: unknown): value is MCPApprovalRule {
	return typeof value === 'string' && Object.values(MCPApprovalRule).includes(value as MCPApprovalRule);
}
