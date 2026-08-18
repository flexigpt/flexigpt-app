import type {
	ArtifactCollection,
	ArtifactCollectionAttachment,
	ArtifactCollectionRef,
	ArtifactDefinitionView,
	ArtifactDigest,
	ArtifactKind,
	ArtifactRecord,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactSourceSummary,
	ArtifactStorageKey,
	ManagedPackageAddress,
} from '@/spec/artifact';

import type { JSONRawString } from '@/lib/jsonschema_utils';

export const MCP_USER_ROOT_ID: ArtifactRootID = '0198f097-0d5b-7000-8000-000000000002';

export const MCP_SCHEMA_VERSION = 'v1';
export const MCP_APP_HTML_MIME_TYPE = 'text/html;profile=mcp-app';

type MCPTimestamp = string;

export enum MCPServerType {
	Stdio = 'stdio',
	HTTP = 'http',
}

export enum MCPTransportType {
	StreamableHTTP = 'streamableHttp',
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

enum MCPPromptRole {
	User = 'user',
	Assistant = 'assistant',
}

interface MCPCoreServer {
	type: MCPServerType;

	command?: string;
	args?: string[];
	env?: Record<string, string>;

	url?: string;
	headers?: Record<string, string>;
}

export interface MCPAuthenticationDeclaration {
	mode: MCPHTTPAuthMode;
	clientCredentialsInput?: string;
	clientIDMetadataDocumentURL?: string;
}

export interface MCPInputDeclaration {
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
	platforms?: string[];
	stdio?: MCPStdioProfile;
	http?: MCPHTTPProfile;
}

interface MCPPolicyReference {
	ref: string;
	required: boolean;
}

interface MCPServerExtension {
	logicalVersion?: string;
	displayName?: string;
	description?: string;
	timeoutMS?: number;
	labels?: Record<string, string>;

	auth: MCPAuthenticationDeclaration;
	install: MCPInstallationDeclaration;
	connectionProfiles?: Record<string, MCPConnectionProfile>;
	policy?: MCPPolicyReference;
}

export interface MCPServerDocument {
	kind: ArtifactKind;
	schemaID: string;
	schemaVersion: string;
	digest?: ArtifactDigest;

	logicalName: string;
	logicalVersion?: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;

	mcpServer: MCPCoreServer;
	extension: MCPServerExtension;
}

interface MCPServerPolicy {
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

export interface MCPPolicy {
	trustLevel: MCPTrustLevel;
	defaultPolicy: MCPServerPolicy;
	toolPolicies?: Record<string, MCPToolPolicyOverride>;
	appsPolicy: MCPAppsPolicy;
}

export interface MCPPolicyDocument {
	kind: ArtifactKind;
	schemaID: string;
	schemaVersion: string;
	digest?: ArtifactDigest;

	logicalName: string;
	logicalVersion?: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;

	body: MCPPolicy;
}

interface MCPEffectivePolicy {
	body: MCPPolicy;
	conflicts?: Record<string, string>;
	digest: ArtifactDigest;
}

interface MCPBundleExtension {
	servers?: Record<string, MCPServerExtension>;
	policies?: Record<string, MCPPolicyDocument>;
}

export interface MCPBundleDocument {
	kind: string;
	schemaID: string;
	schemaVersion: string;
	digest?: ArtifactDigest;

	logicalName: string;
	logicalVersion?: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;

	mcpServers: Record<string, MCPCoreServer>;
	bundleExtension: MCPBundleExtension;
}

interface MCPBundleData {
	schemaVersion: string;
	discoveryPolicyRevision: string;
	logicalName: string;
	logicalVersion?: string;
	labels?: Record<string, string>;
	managedSourceID?: ArtifactSourceID;
}

export interface MCPBundle {
	collection: ArtifactCollection;
	data: MCPBundleData;
	attachment: ArtifactCollectionAttachment;
	source: ArtifactSourceSummary;
	packageAddress: ManagedPackageAddress;
	documentLocator: string;
}

export interface MCPArtifactRegistration {
	artifactID: string;
	subresource: string;
	kind: ArtifactKind;
	enabled: boolean;
	data?: JSONRawString;
}

export interface MCPCreateBundleInput {
	rootID: ArtifactRootID;
	collectionID: string;
	sourceID: ArtifactSourceID;
	sourceStorageKey: ArtifactStorageKey;
	document: MCPBundleDocument;
	registrations: MCPArtifactRegistration[];
}

export interface MCPReplaceBundleDocumentInput {
	bundle: ArtifactCollectionRef;
	expectedCollectionRevision: number;
	document: MCPBundleDocument;
	registrations: MCPArtifactRegistration[];
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

export interface MCPBundleInstallation {
	bundle: ArtifactCollectionRef;
	builtIn: boolean;
	collectionRevision: number;
	overlayRevision: number;
	runtimeEnabled: boolean;
}

export interface MCPServerInstallation {
	artifact: ArtifactRecord;
	collection: ArtifactCollectionRef;
	catalogRevision: number;
	document: MCPServerDocument;
	installation: MCPServerData;
	installationRevision: number;
	installationEnabled: boolean;
	runtimeEnabled: boolean;
	builtIn: boolean;
}

export interface MCPPolicyView {
	artifact: ArtifactRecord;
	collection: ArtifactCollectionRef;
	catalogRevision: number;
	definition: ArtifactDefinitionView;
	body: MCPPolicy;
	effectiveEnabled: boolean;
	builtIn: boolean;
}

export interface MCPServerResolved {
	server: ArtifactRef;
	collection: ArtifactCollectionRef;
	artifactRevision: number;
	catalogRevision: number;
	definitionDigest: ArtifactDigest;
	sourceContentDigest: ArtifactDigest;
	sourceGeneration: string;
	document: MCPServerDocument;
	installation: MCPServerData;
	policy: MCPEffectivePolicy;
	installationRevision: number;
	runtimeEnabled: boolean;
	builtIn: boolean;
	version: ArtifactDigest;
}

interface MCPImplementationInfo {
	name?: string;
	version?: string;
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

export interface MCPServerRuntimeSnapshot {
	server: ArtifactRef;
	collection: ArtifactCollectionRef;
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
	server: ArtifactRef;
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
	server: ArtifactRef;
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
	server: ArtifactRef;
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
	server: ArtifactRef;
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
	server: ArtifactRef;
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
	server: ArtifactRef;
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
	server: ArtifactRef;
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
	role: MCPPromptRole;
	content: MCPContent;
}

interface MCPToolCallProvenance {
	server: ArtifactRef;
	collection: ArtifactCollectionRef;
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
	server: ArtifactRef;
	toolName: string;
	providerToolName?: string;
	content?: MCPContent[];
	structuredContent?: any;
	isError?: boolean;
	provenance: MCPToolCallProvenance;
	app?: MCPToolAppRenderInfo;
}

export interface MCPReadResourceResponseBody {
	server: ArtifactRef;
	uri: string;
	contents?: MCPContent[];
}

export interface MCPGetPromptResponseBody {
	server: ArtifactRef;
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
	server: ArtifactRef;
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

export interface MCPAppModelContextUpdate {
	instanceID?: string;
	server: ArtifactRef;
	resourceUri?: string;
	content?: MCPContent[];
	structuredContent?: any;
	updatedAt?: MCPTimestamp;
}

export interface MCPAuthHealth {
	server: ArtifactRef;
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
	server: ArtifactRef;
	authorizationURL: string;
	expiresAt?: MCPTimestamp;
}

interface MCPAuthSettings {
	oauthLoopbackListenAddr?: string;
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
