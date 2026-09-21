import type { ArtifactRef } from '@/spec/artifact';
import type { StoreArtifact, StoreArtifactAddress, StoreArtifactDefinition } from '@/spec/artifact_store';
import type { CollectionCapabilityPlan, CollectionView } from '@/spec/collection';
import type {
	MCPApprovalRule,
	MCPAppVisibility,
	MCPAuthenticationDeclaration,
	MCPAuthHealth,
	MCPAuthHealthState,
	MCPCompletionRefType,
	MCPCompletionResult,
	MCPContent,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPInputBinding,
	MCPInstallationDeclaration,
	MCPInvocationSource,
	MCPPolicy,
	MCPProviderToolMapping,
	MCPServerCapabilitiesSummary,
	MCPServerStatus,
	MCPServerType,
	MCPToolAppRenderInfo,
	MCPToolCapability,
	MCPToolRisk,
} from '@/spec/mcp_artifact';
import type { CapabilityPlan } from '@/spec/resolution';

export enum MCPPlatform {
	Linux = 'linux',
	Darwin = 'darwin',
	Windows = 'windows',
}

export interface MCPManagedCoreServer {
	type: MCPServerType;
	command?: string;
	args?: string[];
	env?: Record<string, string>;
	url?: string;
	headers?: Record<string, string>;
}

export interface MCPManagedInclude {
	tools?: string[];
	resources?: string[];
	prompts?: string[];
}

export interface MCPManagedStdioProfile {
	command?: string;
	args?: string[];
	env?: Record<string, string>;
	removeEnv?: string[];
}

export interface MCPManagedHTTPProfile {
	url?: string;
	headers?: Record<string, string>;
	removeHeaders?: string[];
}

export interface MCPManagedConnectionProfile {
	platforms?: MCPPlatform[];
	stdio?: MCPManagedStdioProfile;
	http?: MCPManagedHTTPProfile;
}

export interface MCPManagedPolicyReference {
	name: string;
	required: boolean;
}

export interface MCPManagedServerConfiguration {
	timeoutMS?: number;
	auth: MCPAuthenticationDeclaration;
	install: MCPInstallationDeclaration;
	connectionProfiles?: Record<string, MCPManagedConnectionProfile>;
	policy?: MCPManagedPolicyReference;
}

export interface MCPManagedServerDocument {
	logicalName: string;
	logicalVersion?: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;
	mcpServer: MCPManagedCoreServer;
	include?: MCPManagedInclude;
	configuration: MCPManagedServerConfiguration;
}

export interface MCPStoreServerData {
	schemaVersion: string;
	selectedConnectionProfile?: string;
	inputs?: Record<string, MCPInputBinding>;
	additionalPolicies?: ArtifactRef[];
}

export interface ManagedMCPCreateRequest {
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	document: MCPManagedServerDocument;
	enabled: boolean;
}

export interface ManagedMCPCreateResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	collection: CollectionView;
	membershipCreated: boolean;
}

export interface MCPManagedPolicyUpsertRequest {
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	name: string;
	description?: string;
	policy: MCPPolicy;
	enabled: boolean;
}

export interface MCPManagedPolicyUpsertResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	collection: CollectionView;
	membershipCreated: boolean;
}

export interface MCPStoreServerInstallationView {
	artifact: StoreArtifact;
	definition: StoreArtifactDefinition;
	document: MCPManagedServerDocument;
	installation: MCPStoreServerData;
	installationRevision: number;
	builtIn: boolean;
}

export interface MCPStorePolicyView {
	artifact: StoreArtifact;
	definition: StoreArtifactDefinition;
	body: MCPPolicy;
	builtIn: boolean;
}

export interface MCPEffectivePolicy {
	body: MCPPolicy;
	conflicts?: Record<string, string>;
	digest: string;
}

export interface MCPResolvedServer {
	server: ArtifactRef;
	artifactRevision: number;
	definitionDigest: string;
	sourceContentDigest: string;
	sourceGeneration: string;
	document: MCPManagedServerDocument;
	installation: MCPStoreServerData;
	policy: MCPEffectivePolicy;
	installationRevision: number;
	builtIn: boolean;
	version: string;
}

export interface MCPRuntimeImplementationInfo {
	name?: string;
	version?: string;
}

export interface MCPRuntimeServerSnapshot {
	server: string;
	catalog: string;
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

export interface MCPRuntimeInvokeToolRequest {
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

export interface MCPRuntimeToolCallProvenance {
	server: string;
	catalog: string;
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

export interface MCPRuntimeInvokeToolResponse {
	server: string;
	toolName: string;
	providerToolName?: string;
	content?: MCPContent[];
	structuredContent?: unknown;
	isError?: boolean;
	provenance: MCPRuntimeToolCallProvenance;
	app?: MCPToolAppRenderInfo;
}

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
	runtimeServerID: string;
	authHealth?: MCPAuthHealth;
	runtime?: MCPRuntimeServerSnapshot;
}

export interface MCPPolicyManagementView {
	policy: MCPStorePolicyView;
	capabilities: CapabilityPlan;
}

export interface MCPRuntimeStatusView {
	snapshot?: MCPRuntimeServerSnapshot;
	authState?: MCPAuthHealthState;
	toolCount?: number;
	toolRisk?: MCPToolRisk;
	appVisibility?: MCPAppVisibility[];
	executionMode?: MCPExecutionMode;
	approvalRule?: MCPApprovalRule;
	authMode?: MCPHTTPAuthMode;
	tool?: MCPToolCapability;
	providerMapping?: MCPProviderToolMapping;
	completion?: MCPCompletionResult;
}
