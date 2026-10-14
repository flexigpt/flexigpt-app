import type { ArtifactRef, ArtifactState, StoreArtifact, StoreArtifactAddress } from '@/spec/artifact';
import type { PluginListItem, PluginView } from '@/spec/plugin';
import type { MCPAuthHealth, MCPRuntimeServerID, MCPServerRuntimeSnapshot } from '@/spec/session_mcp';

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

	// Runtime-managed only. Put and Delete intentionally reject this value.
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

/**
 * Management aggregate result. Runtime observations are typed in
 * session_mcp.ts because they are process-local Session/MCP runtime state.
 */
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

/**
 * UI-facing management projection over one Plugin-backed MCP Server.
 *
 * Durable Server settings remain in this module. Process-local runtime fields
 * are imported from session_mcp.ts.
 */

/**
 * UI-facing projection over a Plugin-backed MCP grouping.
 */
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
