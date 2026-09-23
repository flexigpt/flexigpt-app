import type {
	ArtifactDiagnostic,
	ArtifactDigest,
	ArtifactKind,
	ArtifactLocator,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactState,
	CapabilityPlan,
	StoreArtifact,
	StoreArtifactRoot,
	StoreArtifactSourceSummary,
} from '@/spec/artifact';

export type WorkspaceRef = ArtifactRef;

export enum WorkspaceDirectoryOrigin {
	Default = 'default',
	Manifest = 'manifest',
}

export enum WorkspaceInsertTarget {
	Instructions = 'instructions',
	UserMessage = 'user-message',
}

export enum WorkspacePromptCompositionStatus {
	Included = 'included',
	Truncated = 'truncated',
	Excluded = 'excluded',
	Denied = 'denied',
	Unavailable = 'unavailable',
}

export enum WorkspaceConversationSelectionStatus {
	Ready = 'ready',
	Partial = 'partial',
	Unavailable = 'unavailable',
}

export enum WorkspaceConversationContextUsageStatus {
	Included = 'included',
	Truncated = 'truncated',
	Excluded = 'excluded',
	Denied = 'denied',
	Unavailable = 'unavailable',
}

export enum WorkspaceConversationSkillUsageStatus {
	Available = 'available',
	Unavailable = 'unavailable',
}

export interface WorkspaceDirectoryRef {
	rootID: ArtifactRootID;
}

interface WorkspaceView {
	artifact: StoreArtifact;
	description?: string;
}

export interface WorkspaceDirectoryWorkspace {
	workspace: WorkspaceView;
	origin: WorkspaceDirectoryOrigin;
	manifestLocator?: ArtifactLocator;
}

export interface WorkspaceDirectoryView {
	ref: WorkspaceDirectoryRef;
	root: StoreArtifactRoot;
	directorySource: StoreArtifactSourceSummary;
	enabled: boolean;
	policyID: string;
	policyVersion: string;
	policyDigest: ArtifactDigest;
	workspaces: WorkspaceDirectoryWorkspace[];
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspacePageRequest {
	cursor?: string;
	limit?: number;
}

export interface WorkspacePage {
	items: WorkspaceDirectoryView[];
	nextCursor?: string;
}

export interface WorkspaceDefaultPolicyView {
	policyID: string;
	policyVersion: string;
	policyDigest: ArtifactDigest;
	yaml: string;
}

export interface WorkspaceArtifactView {
	artifact: ArtifactRef;
	revision: number;
	displayName: string;
	kind: ArtifactKind;
	logicalName: string;
	logicalVersion?: string;
	enabled: boolean;
	state: ArtifactState;
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
}

export interface WorkspacePromptContribution {
	artifact: ArtifactRef;
	artifactRevision: number;
	definitionDigest: ArtifactDigest;
	kind: ArtifactKind;
	name: string;
	insert: WorkspaceInsertTarget;
	mediaType?: string;
	locator?: ArtifactLocator;
	originalBytes: number;
	includedBytes: number;
	truncated: boolean;
}

interface WorkspacePromptDecision {
	artifact: ArtifactRef;
	status: WorkspacePromptCompositionStatus;
	code?: string;
	originalBytes: number;
	includedBytes: number;
}

export interface WorkspacePromptPlan {
	workspace: ArtifactRef;
	contributions: WorkspacePromptContribution[];
	instructions: string;
	userMessage: string;
	diagnostics?: ArtifactDiagnostic[];
	decisions: WorkspacePromptDecision[];
}

export interface WorkspaceSkill {
	artifact: ArtifactRef;
	artifactRevision: number;
	definitionDigest: ArtifactDigest;
	name: string;
	displayName?: string;
	insert?: WorkspaceInsertTarget;
	locator?: ArtifactLocator;
	version: string;
}

export interface WorkspaceSkillLoadPlan {
	workspace: ArtifactRef;
	skills: WorkspaceSkill[];
}

export interface WorkspaceMCPServer {
	artifact: ArtifactRef;
	artifactRevision: number;
	definitionDigest: ArtifactDigest;
	name: string;
	displayName?: string;
	builtIn: boolean;
	version: ArtifactDigest;
}

export interface WorkspaceMCPServerLoadPlan {
	workspace: ArtifactRef;
	servers: WorkspaceMCPServer[];
}

export interface WorkspaceRuntimeSelection {
	promptArtifacts?: ArtifactRef[];
	skillArtifacts?: ArtifactRef[];
	mcpArtifacts?: ArtifactRef[];
	requireComplete?: boolean;
}

export interface WorkspaceRuntimePlan {
	workspace: WorkspaceView;
	capabilities: CapabilityPlan;
	prompt: WorkspacePromptPlan;
	skills: WorkspaceSkillLoadPlan;
	mcpServers: WorkspaceMCPServerLoadPlan;
}

export interface WorkspaceConversationResourceSelectionRef {
	artifact: ArtifactRef;
	name?: string;
	locator?: ArtifactLocator;
	definitionDigest?: ArtifactDigest;
	artifactRevision?: number;
}

export interface WorkspaceConversationSelection {
	workspace: ArtifactRef;
	displayName?: string;
	workspaceRevision?: number;
	contextRefs?: WorkspaceConversationResourceSelectionRef[];
	skillRefs?: WorkspaceConversationResourceSelectionRef[];
}

interface WorkspaceConversationContextUsage {
	artifact: ArtifactRef;
	name?: string;
	locator?: ArtifactLocator;
	selectedDefinitionDigest?: ArtifactDigest;
	usedDefinitionDigest?: ArtifactDigest;
	usedArtifactRevision?: number;
	status: WorkspaceConversationContextUsageStatus;
	code?: string;
	originalBytes?: number;
	includedBytes?: number;
	changed?: boolean;
	diagnostics?: ArtifactDiagnostic[];
}

interface WorkspaceConversationSkillUsage {
	artifact: ArtifactRef;
	name?: string;
	displayName?: string;
	locator?: ArtifactLocator;
	selectedDefinitionDigest?: ArtifactDigest;
	usedDefinitionDigest?: ArtifactDigest;
	usedArtifactRevision?: number;
	status: WorkspaceConversationSkillUsageStatus;
	changed?: boolean;
	sessionAvailable?: boolean;
	active?: boolean;
	advertised?: boolean;
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceConversationUsage {
	workspace: ArtifactRef;
	displayName?: string;
	workspaceRevision?: number;
	status: WorkspaceConversationSelectionStatus;
	contexts?: WorkspaceConversationContextUsage[];
	skills?: WorkspaceConversationSkillUsage[];
	diagnostics?: ArtifactDiagnostic[];
}
