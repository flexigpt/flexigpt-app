export const SecretRefVersion = 'mcpv1';
export const DefaultMCPPageSize = 25;
export const MaxMCPServerPageSize = 256;

export type MCPServerID = string;
export type JSONRawString = string;
export type MCPTimestamp = string;

export enum MCPToolExposure {
	MCPToolExposureNone = 'none',
	MCPToolExposureAll = 'all',
	MCPToolExposureSelected = 'selected',
}

export enum MCPRefType {
	MCPRefTypeResource = 'resource',
	MCPRefTypePrompt = 'prompt',
}

export enum MCPSecretKind {
	MCPSecretKindStdioEnv = 'stdioEnv',
	MCPSecretKindOAuthClientCredentials = 'oauthClientCredentials',
}

export enum MCPTransportType {
	MCPTransportTypeStreamableHTTP = 'streamableHttp',
	MCPTransportTypeStdio = 'stdio',
}

export enum MCPServerAvailability {
	MCPServerAvailabilityManual = 'manual',
	MCPServerAvailabilityAutoAttach = 'autoAttach',
}

export enum MCPTrustLevel {
	MCPTrustLevelUntrusted = 'untrusted',
	MCPTrustLevelTrusted = 'trusted',
}

export enum MCPServerStatus {
	MCPServerStatusDisabled = 'disabled',
	MCPServerStatusDisconnected = 'disconnected',
	MCPServerStatusConnecting = 'connecting',
	MCPServerStatusReady = 'ready',
	MCPServerStatusError = 'error',
}

export enum MCPHTTPAuthMode {
	MCPHTTPAuthNone = 'none',
	MCPHTTPAuthOAuth = 'oauth',
	MCPHTTPAuthClientCredentials = 'clientCredentials',
}

export enum MCPAuthState {
	MCPAuthStateNotRequired = 'notRequired',
	MCPAuthStateRequired = 'required',
	MCPAuthStateAuthorized = 'authorized',
	MCPAuthStateExpired = 'expired',
	MCPAuthStateInsufficientScope = 'insufficientScope',
	MCPAuthStateError = 'error',
}

export enum MCPAuthHealthState {
	MCPAuthHealthStateNotRequired = 'notRequired',
	MCPAuthHealthStateNotConfigured = 'notConfigured',
	MCPAuthHealthStateAuthorizationNeeded = 'authorizationNeeded',
	MCPAuthHealthStateAuthorizationPending = 'authorizationPending',
	MCPAuthHealthStateAuthorized = 'authorized',
	MCPAuthHealthStateExpired = 'expired',
	MCPAuthHealthStateInsufficientScope = 'insufficientScope',
	MCPAuthHealthStateError = 'error',
}

export enum MCPApprovalRule {
	MCPApprovalRuleAsk = 'ask',
	MCPApprovalRuleAllow = 'allow',
	MCPApprovalRuleDeny = 'deny',
}

export enum MCPExecutionMode {
	MCPExecutionModeManual = 'manual',
	MCPExecutionModeAuto = 'auto',
}

export enum MCPToolRisk {
	MCPToolRiskUnknown = 'unknown',
	MCPToolRiskRead = 'read',
	MCPToolRiskWrite = 'write',
	MCPToolRiskDestructive = 'destructive',
	MCPToolRiskOpenWorld = 'openWorld',
}

export enum MCPTaskSupport {
	MCPTaskSupportForbidden = 'forbidden',
	MCPTaskSupportOptional = 'optional',
	MCPTaskSupportRequired = 'required',
}

export enum MCPInvocationSource {
	MCPInvocationSourceModel = 'model',
	MCPInvocationSourceUser = 'user',
	MCPInvocationSourceApp = 'app',
}

export enum MCPApprovalDecision {
	MCPApprovalDecisionAllowed = 'allowed',
	MCPApprovalDecisionDenied = 'denied',
	MCPApprovalDecisionApprovalRequired = 'approvalRequired',
}

export enum MCPApprovalResolution {
	MCPApprovalResolutionAllowOnce = 'allowOnce',
	MCPApprovalResolutionAllowAlways = 'allowAlways',
	MCPApprovalResolutionDenyOnce = 'denyOnce',
	MCPApprovalResolutionDenyAlways = 'denyAlways',
}

export enum MCPContentType {
	MCPContentTypeText = 'text',
	MCPContentTypeImage = 'image',
	MCPContentTypeAudio = 'audio',
	MCPContentTypeResourceLink = 'resource_link',
	MCPContentTypeResource = 'resource',
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
	_meta?: Record<string, any>;
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

	annotations?: Record<string, any>;
	_meta?: Record<string, any>;
	icons?: MCPIcon[];
}

export interface MCPPromptMessage {
	role: string;
	content: MCPContent;
}

export interface MCPToolAnnotations {
	destructiveHint?: boolean;
	idempotentHint: boolean;
	openWorldHint?: boolean;
	readOnlyHint: boolean;
	title?: string;
}

export interface MCPImplementationInfo {
	name?: string;
	version?: string;
}

export interface MCPServerCapabilitiesSummary {
	tools?: boolean;
	toolsListChanged?: boolean;
	resources?: boolean;
	resourcesSubscribe?: boolean;
	resourcesListChanged?: boolean;
	prompts?: boolean;
	promptsListChanged?: boolean;
	logging?: boolean;
	completions?: boolean;
	experimental?: Record<string, any>;
	extensions?: Record<string, any>;
}

export interface MCPStdioConfig {
	command: string;
	args?: string[];
	workingDir?: string;
	env?: Record<string, string>;
	secretEnvRefs?: Record<string, string>;
	startupTimeoutMS?: number;
}

export interface MCPStreamableHTTPConfig {
	url: string;
	timeoutMS?: number;
	authMode: MCPHTTPAuthMode;

	clientCredentialRef?: string;
	clientIDMetadataDocumentURL?: string;
}

export interface MCPAuthStatus {
	bundleID: string;
	serverID: MCPServerID;
	authMode: MCPHTTPAuthMode;
	state: MCPAuthState;

	scopes?: string[];
	expiresAt?: MCPTimestamp;
	lastError?: string;
	authorizationServer?: string;
	resource?: string;
}

export interface MCPAuthHealth {
	bundleID?: string;
	serverID: MCPServerID;
	authMode: MCPHTTPAuthMode;
	state: MCPAuthHealthState;

	configured: boolean;

	resource?: string;
	scopes?: string[];
	expiresAt?: MCPTimestamp;

	authorizationPending?: boolean;
	authorizationURL?: string;
	authorizationExpiresAt?: MCPTimestamp;

	lastError?: string;
}

export interface MCPServerPolicy {
	defaultApprovalRule: MCPApprovalRule;
	defaultExecutionMode: MCPExecutionMode;

	requireApprovalForUnknownRisk: boolean;
	requireApprovalForWrite: boolean;
	requireApprovalForDestructive: boolean;
}

export interface MCPToolPolicyOverride {
	toolName: string;

	approvalRule?: MCPApprovalRule;
	executionMode?: MCPExecutionMode;

	allowStaleDigest?: boolean;
	expectedDigest?: string;
}

export interface MCPAppsPolicy {
	enabled: boolean;
	allowAppInitiatedToolCalls: boolean;
	requireApprovalForOpenLink: boolean;
	requireApprovalForContextUpdates: boolean;
}

export interface MCPBundle {
	schemaVersion: string;
	id: string;
	slug: string;
	displayName?: string;
	description?: string;
	isEnabled: boolean;
	createdAt: MCPTimestamp;
	modifiedAt: MCPTimestamp;
	isBuiltIn: boolean;
	softDeletedAt?: MCPTimestamp;
}

export interface PutMCPServerPayload {
	displayName: string;
	enabled: boolean;
	transport: MCPTransportType;

	stdio?: MCPStdioConfig;
	streamableHttp?: MCPStreamableHTTPConfig;

	availability?: MCPServerAvailability;
	trustLevel?: MCPTrustLevel;

	defaultPolicy?: MCPServerPolicy;
	toolPolicies?: Record<string, MCPToolPolicyOverride>;
	appsPolicy?: MCPAppsPolicy;
}

export interface PatchMCPServerPolicyPayload {
	defaultPolicy?: MCPServerPolicy;
	toolPolicies?: Record<string, MCPToolPolicyOverride>;
	appsPolicy?: MCPAppsPolicy;
}

export interface PutMCPServerSecretRequestBody {
	kind: MCPSecretKind;
	slot: string;
	secret: string;
}

export interface PutMCPServerSecretResponseBody {
	secretRef: string;
	sha256?: string;
	nonEmpty: boolean;
}

export interface PatchMCPServerEnabledRequestBody {
	enabled: boolean;
}

export interface MCPServerRuntimeSnapshot {
	bundleID: string;
	serverID: MCPServerID;
	status: MCPServerStatus;

	negotiatedProtocolVersion?: string;
	serverInfo?: MCPImplementationInfo;
	serverCapabilities?: MCPServerCapabilitiesSummary;
	instructions?: string;

	lastError?: string;
	lastConnectedAt?: MCPTimestamp;
	lastSyncedAt?: MCPTimestamp;

	toolCount: number;
	resourceCount: number;
	resourceTemplateCount: number;
	promptCount: number;

	snapshotDigest?: string;
}

// Opaque cursor for pagination.
export interface MCPDiscoveryPageToken {
	sid: MCPServerID;
	dig: string;
	k: string;
	ps: number;
	i: number;
}

export interface MCPToolAppInfo {
	resourceUri?: string;
	visibility?: string[];
}

export interface MCPToolCapability {
	bundleID: string;
	serverID: MCPServerID;
	toolName: string;
	providerToolName: string;
	choiceID: string;

	title?: string;
	displayName: string;
	description?: string;

	inputSchema?: Record<string, any>;
	outputSchema?: Record<string, any>;

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

export interface MCPResourceRef {
	bundleID: string;
	serverID: MCPServerID;
	uri: string;
	name?: string;
	title?: string;
	displayName: string;
	description?: string;
	mimeType?: string;
	size?: number;
	annotations?: Record<string, any>;
	digest?: string;
}

export interface MCPResourceTemplateRef {
	bundleID: string;
	serverID: MCPServerID;
	uriTemplate: string;
	name?: string;
	title?: string;
	displayName: string;
	description?: string;
	mimeType?: string;
	arguments?: Record<string, string>;
	annotations?: Record<string, any>;
	digest?: string;
}

export interface MCPPromptRef {
	bundleID: string;
	serverID: MCPServerID;
	promptName: string;
	title?: string;
	displayName: string;
	description?: string;
	arguments?: Record<string, string>;
	digest?: string;
}

export interface MCPDiscoverySnapshot {
	serverID: MCPServerID;

	negotiatedProtocolVersion?: string;
	serverInfo?: MCPImplementationInfo;
	serverCapabilities?: MCPServerCapabilitiesSummary;
	instructions?: string;

	tools?: MCPToolCapability[];
	resources?: MCPResourceRef[];
	resourceTemplates?: MCPResourceTemplateRef[];
	prompts?: MCPPromptRef[];

	digest?: string;
	syncedAt?: MCPTimestamp;
}

export interface MCPToolSelection {
	bundleID: string;
	serverID: MCPServerID;
	toolName: string;
	providerToolName?: string;
	choiceID?: string;
	digest?: string;

	approvalRule?: MCPApprovalRule;
	executionMode?: MCPExecutionMode;
}

export interface MCPProviderToolMapping {
	providerToolName: string;
	choiceID: string;
	serverID: MCPServerID;
	toolName: string;
	toolDigest: string;
	appResourceUri?: string;
	visibility?: string[];
}

export interface MCPServerSelection {
	bundleID: string;
	serverID: MCPServerID;
	snapshotDigest?: string;

	toolExposure: MCPToolExposure;
	selectedTools?: MCPToolSelection[];

	includeServerInstructions?: boolean;
}

export interface MCPConversationContext {
	servers: MCPServerSelection[];
	resources?: MCPResourceRef[];
	resourceTemplates?: MCPResourceTemplateRef[];
	prompts?: MCPPromptRef[];
}

export interface MCPToolCallProvenance {
	bundleID: string;
	serverID: MCPServerID;
	serverDisplayName?: string;

	toolName: string;
	providerToolName: string;
	toolDigest?: string;

	toolUseID?: string;
	approvalID?: string;

	appResourceUri?: string;
	appInstanceID?: string;
}

export interface MCPSecretRef {
	serverID: MCPServerID;
	kind: MCPSecretKind;
	slot?: string;
}

export interface MCPOAuthAuthorization {
	bundleID: string;
	serverID: string;
	authorizationURL: string;
	expiresAt?: MCPTimestamp;
}

export interface MCPApprovalSummary {
	bundleID: string;
	serverID: MCPServerID;
	serverDisplayName?: string;
	toolName: string;
	toolDigest?: string;
	risk: MCPToolRisk;
	arguments?: JSONRawString;
}

export interface MCPApprovalEvaluation {
	decision: MCPApprovalDecision;
	reason?: string;
	approvalID?: string;
	summary?: MCPApprovalSummary;
}

export interface MCPApprovalToken {
	approvalID: string;
	token: string;
	expiresAt: MCPTimestamp;
}

export interface InvokeMCPToolRequestBody {
	source: MCPInvocationSource;

	serverID: MCPServerID;
	toolName: string;
	providerToolName?: string;
	toolDigest?: string;

	arguments?: Record<string, any>;

	approvalID?: string;
	approvalToken?: string;

	conversationID?: string;
	messageID?: string;
	toolUseID?: string;

	appInstanceID?: string;
}

export interface MCPToolAppRenderInfo {
	resourceUri?: string;
	mimeType?: string;
}

export interface InvokeMCPToolResponseBody {
	bundleID: string;
	serverID: string;
	toolName: string;
	providerToolName?: string;

	content?: MCPContent[];
	structuredContent?: any;
	isError?: boolean;

	provenance: MCPToolCallProvenance;
	app?: MCPToolAppRenderInfo;
}

export interface MCPReadResourceRequestBody {
	serverID: MCPServerID;
	uri: string;
}

export interface MCPReadResourceResponseBody {
	bundleID: string;
	serverID: string;
	uri: string;
	contents?: MCPContent[];
}

export interface MCPGetPromptRequestBody {
	serverID: MCPServerID;
	promptName: string;
	arguments?: Record<string, string>;
}

export interface MCPGetPromptResponseBody {
	bundleID: string;
	serverID: string;
	promptName: string;
	description?: string;
	messages?: MCPPromptMessage[];
}

export interface MCPCompleteArgumentRequestBody {
	serverID: MCPServerID;
	refType: MCPRefType;
	name: string;
	argumentName: string;
	argumentValue?: string;
	context?: Record<string, string>;
}

export interface MCPCompletionResult {
	values?: string[];
	total?: number;
	hasMore?: boolean;
}

export interface MCPServerConfig {
	bundleID: string;
	schemaVersion: string;
	id: string;
	displayName: string;
	enabled: boolean;
	transport: MCPTransportType;
	stdio?: MCPStdioConfig;
	streamableHttp?: MCPStreamableHTTPConfig;
	availability: MCPServerAvailability;
	trustLevel: MCPTrustLevel;
	defaultPolicy: MCPServerPolicy;
	toolPolicies?: Record<string, MCPToolPolicyOverride>;
	appsPolicy?: MCPAppsPolicy;
	isBuiltIn: boolean;
	createdAt: MCPTimestamp;
	modifiedAt: MCPTimestamp;
	softDeletedAt?: MCPTimestamp;
}
