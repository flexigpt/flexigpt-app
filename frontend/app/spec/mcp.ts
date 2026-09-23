import type { ArtifactRef, CapabilityPlan, StoreArtifact, StoreArtifactAddress } from '@/spec/artifact';
import type { CollectionCapabilityPlan, CollectionView } from '@/spec/collection';

import type { JSONRawString } from '@/lib/jsonschema_utils';

export const MCP_SCHEMA_VERSION = 'v1';
export const MCP_APP_HTML_MIME_TYPE = 'text/html;profile=mcp-app';

type MCPTimestamp = string;

/**
 * Runtime-owned opaque MCP identities. Frontend code may carry these values
 * between Runtime calls but must not construct or parse them.
 */
export type MCPRuntimeServerID = string;
export type MCPRuntimeCatalogID = string;

export interface MCPManagementPage<T> {
	items: T[];
	nextPageToken?: string;
}

export interface MCPDiscoveryPage<T> {
	items: T[];
	nextPageToken?: string;
}

export enum MCPServerType {
	Stdio = 'stdio',
	HTTP = 'http',
}

export enum MCPTransportType {
	StreamableHTTP = 'streamableHTTP',
	Stdio = 'stdio',
}

export enum MCPHTTPAuthMode {
	None = 'none',
	APIKey = 'apiKey',
	OAuth = 'oauth',
	ClientCredentials = 'clientCredentials',
}

export enum MCPInputKind {
	Text = 'text',
	Secret = 'secret',
	Path = 'path',
	OAuthClientCredentials = 'oauthClientCredentials',
}

export enum MCPSecretKind {
	StdioEnv = 'stdioEnv',
	OAuthClientCredentials = 'oauthClientCredentials',
	HTTPHeader = 'httpHeader',

	// Runtime-managed only. PutMCPServerSecret and DeleteMCPServerSecret
	// intentionally reject this value.
	OAuthToken = 'oauthToken',
}

export enum MCPTrustLevel {
	Untrusted = 'untrusted',
	Trusted = 'trusted',
}

export enum MCPApprovalRule {
	Ask = 'ask',
	Allow = 'allow',
	Deny = 'deny',
}

export enum MCPExecutionMode {
	Manual = 'manual',
	Auto = 'auto',
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

enum MCPTaskSupport {
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

export enum MCPToolExposure {
	None = 'none',
	All = 'all',
	Selected = 'selected',
}

export enum MCPCompletionRefType {
	Resource = 'resource',
	Prompt = 'prompt',
}

interface MCPServerPolicy {
	defaultApprovalRule: MCPApprovalRule;
	defaultExecutionMode: MCPExecutionMode;
	requireApprovalForUnknownRisk: boolean;
	requireApprovalForWrite: boolean;
	requireApprovalForDestructive: boolean;
}

interface MCPToolPolicyOverride {
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

export interface MCPPolicy {
	trustLevel: MCPTrustLevel;
	defaultPolicy: MCPServerPolicy;
	toolPolicies?: Record<string, MCPToolPolicyOverride>;
	appsPolicy: MCPAppsPolicy;
}

export interface MCPInputBinding {
	value?: string;
	secretRef?: string;
}

export interface MCPServerData {
	schemaVersion: string;
	selectedConnectionProfile?: string;
	inputs?: Record<string, MCPInputBinding>;
	additionalPolicies?: ArtifactRef[];
}

interface MCPServerCapabilitiesSummary {
	tools?: boolean;
	toolsListChanged?: boolean;
	resources?: boolean;
	resourcesSubscribe?: boolean;
	resourcesListChanged?: boolean;
	prompts?: boolean;
	promptsListChanged?: boolean;
	completions?: boolean;
	experimental?: Record<string, any>;
	extensions?: Record<string, any>;
}

interface MCPToolAnnotations {
	destructiveHint?: boolean;
	idempotentHint: boolean;
	openWorldHint?: boolean;
	readOnlyHint: boolean;
	title?: string;
}

interface MCPToolAppInfo {
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
	annotations?: Record<string, any>;
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
	annotations?: Record<string, any>;
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
	approvalRule: MCPApprovalRule;
	executionMode: MCPExecutionMode;
	appResourceUri?: string;
	visibility?: MCPAppVisibility[];
}

export interface MCPServerSelection {
	server: MCPRuntimeServerID;
	snapshotDigest?: string;
	toolExposure: MCPToolExposure;
	selectedTools?: MCPToolSelection[];
	includeServerInstructions?: boolean;
}

export interface MCPConversationContext {
	servers: MCPServerSelection[];
	resources?: MCPResourceRef[];
	resourceTemplates?: MCPResourceTemplateSelection[];
	prompts?: MCPPromptSelection[];
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

interface MCPPromptMessage {
	role: string;
	content: MCPContent;
}

interface MCPToolCallProvenance {
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

export interface InvokeMCPToolRequestBody {
	source: MCPInvocationSource;
	toolName: string;
	providerToolName?: string;
	choiceID?: string;
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
	content?: MCPContent[];
	structuredContent?: any;
	isError?: boolean;
}

export interface InvokeMCPToolResponseBody {
	server: MCPRuntimeServerID;
	toolName: string;
	providerToolName?: string;
	content?: MCPContent[];
	structuredContent?: any;
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
	arguments?: JSONRawString;
}

export interface MCPApprovalEvaluation {
	decision: MCPApprovalDecision;
	reason?: string;
	approvalID?: string;
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
export interface MCPAppModelContextUpdate {
	instanceID?: string;
	server: MCPRuntimeServerID;
	resourceUri?: string;
	content?: MCPContent[];
	structuredContent?: any;
	updatedAt?: MCPTimestamp;
	rawArguments?: JSONRawString;
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

export interface MCPOAuthAuthorization {
	server: MCPRuntimeServerID;
	authorizationURL: string;
	expiresAt?: MCPTimestamp;
}

export interface MCPGlobalSettings {
	settings: MCPAuthSettings;
	revision: number;
	oauthRedirectURL?: string;
	oauthLoopbackListenAddr?: string;
	oauthRestartRequired: boolean;
	oauthLoopbackReady: boolean;
	oauthLoopbackError?: string;
}

export interface MCPSecretWriteResult {
	secretRef: string;
	sha256?: string;
	nonEmpty: boolean;
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

enum MCPPlatform {
	Linux = 'linux',
	Darwin = 'darwin',
	Windows = 'windows',
}

interface MCPServerCore {
	type: MCPServerType;
	command?: string;
	args?: string[];
	env?: Record<string, string>;
	url?: string;
	headers?: Record<string, string>;
}

interface MCPServerInclude {
	tools?: string[];
	resources?: string[];
	prompts?: string[];
}

export interface MCPAuthenticationDeclaration {
	mode: MCPHTTPAuthMode;
	clientCredentialsInput?: string;
	clientIDMetadataDocumentURL?: string;
}

interface MCPInputDeclaration {
	kind: MCPInputKind;
	label?: string;
	description?: string;
	note?: string;
	placeholder?: string;
	required?: boolean;
	default?: string;
	clientSecretRequired?: boolean;
}

interface MCPInstallationDeclaration {
	note?: string;
	inputs?: Record<string, MCPInputDeclaration>;
	allowEnvironment?: string[];
}

interface MCPStdioProfile {
	command?: string;
	args?: string[];
	env?: Record<string, string>;
	removeEnv?: string[];
}

interface MCPHTTPProfile {
	url?: string;
	headers?: Record<string, string>;
	removeHeaders?: string[];
}

interface MCPConnectionProfile {
	platforms?: MCPPlatform[];
	stdio?: MCPStdioProfile;
	http?: MCPHTTPProfile;
}

interface MCPServerPolicyReference {
	name: string;
	required: boolean;
}

interface MCPServerConfiguration {
	timeoutMS?: number;
	auth: MCPAuthenticationDeclaration;
	install: MCPInstallationDeclaration;
	connectionProfiles?: Record<string, MCPConnectionProfile>;
	policy?: MCPServerPolicyReference;
}

// Exact frontend projection of Go server.ServerDocument.
//
// It is used for mutable user Servers and protected built-in Servers alike.
// It is not a portable declaration and does not contain raw secret values.
export interface MCPServerDocument {
	logicalName: string;
	logicalVersion?: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;
	mcpServer: MCPServerCore;
	include?: MCPServerInclude;
	configuration: MCPServerConfiguration;
}

export interface ManagedMCPCreateRequest {
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	document: MCPServerDocument;
	enabled: boolean;
}

export interface ManagedMCPCreateResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	collection: CollectionView;
	membershipCreated: boolean;
}

export interface ManagedMCPPolicyUpsertRequest {
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	name: string;
	description?: string;
	policy: MCPPolicy;
	enabled: boolean;
}

export interface ManagedMCPPolicyUpsertResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	collection: CollectionView;
	membershipCreated: boolean;
}

export interface MCPStoreServerInstallationView {
	artifact: StoreArtifact;
	document: MCPServerDocument;
	installation: MCPServerData;
	installationRevision: number;
	builtIn: boolean;
}

export interface MCPStorePolicyView {
	artifact: StoreArtifact;
	body: MCPPolicy;
	builtIn: boolean;
}

export interface MCPEffectivePolicy {
	body: MCPPolicy;
	conflicts?: Record<string, string>;
	digest: string;
}

interface MCPRuntimeImplementationInfo {
	name?: string;
	version?: string;
}

export interface MCPServerRuntimeSnapshot {
	server: MCPRuntimeServerID;
	catalog: MCPRuntimeCatalogID;
	status: MCPServerStatus;
	negotiatedProtocolVersion?: string;
	serverInfo?: MCPRuntimeImplementationInfo;
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

export type MCPRuntimeInvokeToolResponse = InvokeMCPToolResponseBody;

export interface MCPAuthSettings {
	oauthLoopbackListenAddr?: string;
}

export interface MCPCollectionManagementView {
	collection: CollectionView;
	capabilities: CollectionCapabilityPlan;
}

export interface MCPServerManagementView {
	installation: MCPStoreServerInstallationView;
	capabilities: CapabilityPlan;
	runtimeServerID: MCPRuntimeServerID;
	policy: MCPEffectivePolicy;
	authHealth?: MCPAuthHealth;
	runtime?: MCPServerRuntimeSnapshot;
	authHealthError?: string;
	runtimeError?: string;
}

export interface MCPPolicyManagementView {
	policy: MCPStorePolicyView;
	capabilities: CapabilityPlan;
}

export interface ManagedMCPReplaceRequest {
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	artifact: ArtifactRef;
	expectedArtifactRevision: number;
	document: MCPServerDocument;
	enabled: boolean;
}

export interface ManagedMCPReplaceResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	collection: CollectionView;
}

export interface MCPBundleView {
	collection: CollectionView;
	ref: ArtifactRef;
	displayName: string;
	logicalName: string;
	description?: string;
	enabled: boolean;
	builtIn: boolean;
	editable: boolean;
	deletable: boolean;
	baseline: boolean;
}

export interface MCPServerView {
	ref: ArtifactRef;
	runtimeServerID?: MCPRuntimeServerID;
	artifact: StoreArtifact;
	bundle: ArtifactRef;
	logicalName: string;
	displayName: string;
	document?: MCPServerDocument;
	installation?: MCPServerData;
	installationRevision?: number;
	enabled: boolean;
	builtIn: boolean;
	policy?: MCPEffectivePolicy;
	policyRef?: ArtifactRef;
	loadError?: string;
}

export interface MCPSetupSecretTarget {
	kind: MCPSecretKind;
	slot: string;
}

export interface MCPSetupInputView {
	name: string;
	declaration: MCPInputDeclaration;
	target?: MCPSetupSecretTarget;
	boundValue?: string;
	boundSecretRef?: string;
}

export interface MCPSetupSubmissionValue {
	value?: string;
	clientID?: string;
	clientSecret?: string;
}

export interface MCPStdioSecretDraft {
	inputName?: string;
	envName: string;
	existingSecretRef?: string;
	secretValue: string;
	deleteExisting: boolean;
}

export interface MCPHTTPSecretDraft {
	inputName?: string;
	headerName: string;
	valuePrefix: string;
	valueSuffix: string;
	existingSecretRef?: string;
	secretValue: string;
	deleteExisting: boolean;
}

interface MCPOAuthClientCredentialsDraft {
	inputName?: string;
	existingSecretRef?: string;
	secretJSON: string;
	deleteExisting: boolean;
	useClientCredentials: boolean;
}

export interface MCPServerDraft {
	logicalName: string;
	displayName: string;
	enabled: boolean;
	transport: MCPTransportType;
	trustLevel: MCPTrustLevel;

	stdioCommand: string;
	stdioArgs: string[];
	stdioEnv: Record<string, string>;
	stdioStartupTimeoutMS?: number;
	stdioSecrets: MCPStdioSecretDraft[];

	httpURL: string;
	httpHeaders: Record<string, string>;
	httpTimeoutMS?: number;
	httpAuthMode: MCPHTTPAuthMode;
	httpAPIKey?: MCPHTTPSecretDraft;
	httpOAuthClientCredentials: MCPOAuthClientCredentialsDraft;
	httpClientIDMetadataDocumentURL: string;

	defaultPolicy: MCPPolicy['defaultPolicy'];
	toolPolicies: Record<string, MCPToolPolicyOverride>;
	appsPolicy: MCPAppsPolicy;
}
