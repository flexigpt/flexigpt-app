import type { ArtifactRef, ArtifactState, StoreArtifact, StoreArtifactAddress } from '@/spec/artifact';
import type { PluginListItem, PluginView } from '@/spec/plugin';

export type MCPTimestamp = string;
export type MCPRuntimeServerID = string;
type MCPRuntimeCatalogID = string;

export const MCP_SCHEMA_VERSION = 'v1';
export const MCP_APP_HTML_MIME_TYPE = 'text/html;profile=mcp-app';

export interface MCPPluginPage {
	items: PluginListItem[];
	nextPageToken?: string;
}

export interface MCPServerPage {
	items: MCPServerListItem[];
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

export function getMCPTransportLabel(value: MCPTransportType): string {
	switch (value) {
		case MCPTransportType.Stdio:
			return 'Stdio';
		case MCPTransportType.StreamableHTTP:
			return 'Streamable HTTP';
		default:
			return String(value);
	}
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

	// Runtime-managed only. Put and Delete intentionally reject this value.
	OAuthToken = 'oauthToken',
}

export enum MCPTrustLevel {
	Untrusted = 'untrusted',
	Trusted = 'trusted',
}

export function getMCPTrustLevelLabel(value: MCPTrustLevel): string {
	return value === MCPTrustLevel.Trusted ? 'Trusted' : 'Untrusted';
}

export enum MCPHTTPAuthMode {
	None = 'none',
	APIKey = 'apiKey',
	OAuth = 'oauth',
	ClientCredentials = 'clientCredentials',
}

export function getMCPHTTPAuthModeLabel(value: MCPHTTPAuthMode): string {
	switch (value) {
		case MCPHTTPAuthMode.None:
			return 'None';
		case MCPHTTPAuthMode.APIKey:
			return 'API Key';
		case MCPHTTPAuthMode.OAuth:
			return 'OAuth';
		case MCPHTTPAuthMode.ClientCredentials:
			return 'Client Credentials';
		default:
			return String(value);
	}
}

export enum MCPAuthState {
	NotRequired = 'notRequired',
	Required = 'required',
	Authorized = 'authorized',
	Expired = 'expired',
	InsufficientScope = 'insufficientScope',
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

export function getMCPServerAuthHealthStateLabel(state: MCPAuthHealthState | undefined): string {
	switch (state) {
		case MCPAuthHealthState.NotConfigured:
			return 'Auth: config needed';
		case MCPAuthHealthState.AuthorizationNeeded:
			return 'OAuth: required';
		case MCPAuthHealthState.AuthorizationPending:
			return 'OAuth: pending';
		case MCPAuthHealthState.Authorized:
			return 'OAuth: authorized';
		case MCPAuthHealthState.Expired:
			return 'OAuth: expired';
		case MCPAuthHealthState.InsufficientScope:
			return 'Auth: insufficient scope';
		case MCPAuthHealthState.Error:
			return 'Auth: error';
		default:
			return 'Auth: unknown';
	}
}

export enum MCPApprovalRule {
	Ask = 'ask',
	Allow = 'allow',
	Deny = 'deny',
}

export function getMCPApprovalRuleLabel(value: MCPApprovalRule): string {
	switch (value) {
		case MCPApprovalRule.Allow:
			return 'Allow';
		case MCPApprovalRule.Deny:
			return 'Deny';
		default:
			return 'Ask';
	}
}

export enum MCPExecutionMode {
	Manual = 'manual',
	Auto = 'auto',
}

export function getMCPExecutionModeLabel(value: MCPExecutionMode): string {
	return value === MCPExecutionMode.Auto ? 'Auto' : 'Manual';
}

export enum MCPServerStatus {
	Disabled = 'disabled',
	Disconnected = 'disconnected',
	Connecting = 'connecting',
	Ready = 'ready',
	Error = 'error',
}

export function getMCPStatusLabel(value: MCPServerStatus): string {
	switch (value) {
		case MCPServerStatus.Disabled:
			return 'Disabled';
		case MCPServerStatus.Disconnected:
			return 'Disconnected';
		case MCPServerStatus.Connecting:
			return 'Connecting';
		case MCPServerStatus.Ready:
			return 'Ready';
		case MCPServerStatus.Error:
			return 'Error';
		default:
			return 'Unknown';
	}
}

export enum MCPToolRisk {
	Unknown = 'unknown',
	Read = 'read',
	Write = 'write',
	Destructive = 'destructive',
	OpenWorld = 'openWorld',
}

export function getMCPToolRiskLabel(value: MCPToolRisk): string {
	switch (value) {
		case MCPToolRisk.Read:
			return 'Read';
		case MCPToolRisk.Write:
			return 'Write';
		case MCPToolRisk.Destructive:
			return 'Destructive';
		case MCPToolRisk.OpenWorld:
			return 'Open World';
		default:
			return 'Unknown';
	}
}

export enum MCPAppVisibility {
	Model = 'model',
	App = 'app',
}

export enum MCPTaskSupport {
	Forbidden = 'forbidden',
	Optional = 'optional',
	Required = 'required',
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

interface MCPAppsPolicy {
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

interface MCPInputBinding {
	value?: string;
}

export interface MCPServerData {
	schemaVersion: string;
	selectedConnectionProfile?: string;
	inputs?: Record<string, MCPInputBinding>;
	additionalPolicies?: ArtifactRef[];
}

interface MCPInstallationInputView {
	value?: string;
	secretConfigured: boolean;
}

export interface MCPServerInstallationDataView {
	selectedConnectionProfile?: string;
	inputs?: Record<string, MCPInstallationInputView>;
	additionalPolicies?: ArtifactRef[];
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
	platforms?: Array<'linux' | 'darwin' | 'windows'>;
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

/**
 * Exact frontend projection of Go server.ServerDocument.
 *
 * It is used for mutable user Servers and protected built-in Servers alike.
 * It is not a portable declaration and does not contain raw secret values.
 */
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
	plugin: ArtifactRef;
	expectedPluginRevision: number;
	document: MCPServerDocument;
	enabled: boolean;
}

export interface ManagedMCPCreateResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	plugin: PluginView;
	membershipCreated: boolean;
}

export interface ManagedMCPPolicyUpsertRequest {
	plugin: ArtifactRef;
	expectedPluginRevision: number;
	name: string;
	description?: string;
	policy: MCPPolicy;
	enabled: boolean;
}

export interface ManagedMCPPolicyUpsertResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	plugin: PluginView;
	membershipCreated: boolean;
}

export interface ManagedMCPReplaceRequest {
	plugin: ArtifactRef;
	expectedPluginRevision: number;
	artifact: ArtifactRef;
	expectedArtifactRevision: number;
	document: MCPServerDocument;
	enabled: boolean;
}

export interface ManagedMCPReplaceResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	plugin: PluginView;
}

export interface MCPServerInstallationView {
	artifact: StoreArtifact;
	document: MCPServerDocument;
	installation: MCPServerInstallationDataView;
	installationRevision: number;
	builtIn: boolean;
}

export interface MCPPolicyView {
	artifact: StoreArtifact;
	body: MCPPolicy;
	builtIn: boolean;
}

interface MCPEffectivePolicy {
	body: MCPPolicy;
	conflicts?: Record<string, string>;
	digest: string;
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

interface MCPServerCapabilitiesSummary {
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

export interface MCPServerDetails {
	settings: MCPServerInstallationView;
	policy: MCPEffectivePolicy;
	authorization: MCPAuthHealth;
	connection: MCPServerRuntimeSnapshot;
}

interface MCPServerSecretInput {
	name: string;
	label?: string;
	description?: string;
	required: boolean;
	kind: MCPInputKind;
	configured: boolean;
}

export interface MCPServerSecretsView {
	inputs: MCPServerSecretInput[];
}

export interface MCPAuthSettings {
	oauthLoopbackListenAddr?: string;
}

export interface MCPSettings {
	revision: number;
	oauthLoopbackListenAddr?: string;
}

export interface MCPServerListItem {
	ref: ArtifactRef;
	name: string;
	displayName: string;
	description?: string;
	state: ArtifactState;
	enabled: boolean;
	revision: number;
	definitionDigest?: string;
	builtIn: boolean;
}

export interface MCPPolicyListItem {
	ref: ArtifactRef;
	name: string;
	displayName: string;
	description?: string;
	state: ArtifactState;
	enabled: boolean;
	revision: number;
	definitionDigest?: string;
	builtIn: boolean;
}
export interface MCPPluginView {
	plugin: PluginListItem;
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
	plugin: ArtifactRef;
	logicalName: string;
	displayName: string;
	document?: MCPServerDocument;
	installation?: MCPServerInstallationDataView;
	installationRevision?: number;
	enabled: boolean;
	builtIn: boolean;
	policy?: MCPEffectivePolicy;
	policyRef?: ArtifactRef;
	loadError?: string;
	authHealth?: MCPAuthHealth;
	runtime?: MCPServerRuntimeSnapshot;
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
	secretConfigured: boolean;
}

export interface MCPSetupSubmissionValue {
	value?: string;
	clientID?: string;
	clientSecret?: string;
}

export interface MCPServerSetupView {
	server: ArtifactRef;
	displayName: string;
	builtIn: boolean;
	settingsRevision: number;
	note?: string;
	inputs: MCPSetupInputView[];
}

export interface MCPStdioSecretDraft {
	inputName?: string;
	envName: string;
	existingSecretConfigured?: boolean;
	secretValue: string;
	deleteExisting: boolean;
}

export interface MCPHTTPSecretDraft {
	inputName?: string;
	headerName: string;
	valuePrefix: string;
	valueSuffix: string;
	existingSecretConfigured?: boolean;
	secretValue: string;
	deleteExisting: boolean;
}

interface MCPOAuthClientCredentialsDraft {
	inputName?: string;
	existingSecretConfigured?: boolean;
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

export interface MCPArgumentDefinition {
	name: string;
	title?: string;
	description?: string;
	required?: boolean;
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

const STATUS_BADGE_LAYOUT = 'h-auto max-w-full whitespace-normal break-words px-2 py-1 text-center leading-tight';

export function getMCPStatusBadgeClass(value: MCPServerStatus): string {
	switch (value) {
		case MCPServerStatus.Ready:
			return `${STATUS_BADGE_LAYOUT} badge-success`;
		case MCPServerStatus.Connecting:
			return `${STATUS_BADGE_LAYOUT} badge-info`;
		case MCPServerStatus.Error:
			return `${STATUS_BADGE_LAYOUT} badge-error`;
		case MCPServerStatus.Disabled:
			return `${STATUS_BADGE_LAYOUT} badge-neutral`;
		default:
			return `${STATUS_BADGE_LAYOUT} badge-warning`;
	}
}

export function getMCPServerAuthHealthStateBadgeClass(state: MCPAuthHealthState | undefined): string {
	switch (state) {
		case MCPAuthHealthState.NotRequired:
			return `${STATUS_BADGE_LAYOUT} badge-ghost`;
		case MCPAuthHealthState.Authorized:
			return `${STATUS_BADGE_LAYOUT} badge-success`;
		case MCPAuthHealthState.AuthorizationPending:
			return `${STATUS_BADGE_LAYOUT} badge-info`;
		case MCPAuthHealthState.AuthorizationNeeded:
		case MCPAuthHealthState.NotConfigured:
		case MCPAuthHealthState.Expired:
			return `${STATUS_BADGE_LAYOUT} badge-warning`;
		case MCPAuthHealthState.Error:
		case MCPAuthHealthState.InsufficientScope:
			return `${STATUS_BADGE_LAYOUT} badge-error`;
		default:
			return `${STATUS_BADGE_LAYOUT} badge-neutral`;
	}
}
