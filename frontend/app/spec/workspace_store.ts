import type {
	ArtifactDiagnostic,
	ArtifactKind,
	ArtifactLocator,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactState,
} from '@/spec/artifact';
import type {
	StoreArtifact,
	StoreArtifactDefinition,
	StoreArtifactRoot,
	StoreArtifactSourceSummary,
} from '@/spec/artifact_store';
import type { MCPResolvedServer } from '@/spec/mcp';
import type { CapabilityPlan } from '@/spec/resolution';
import type { SkillDocumentInput, SkillInsert } from '@/spec/skill';
import type { WorkspaceContextCompositionStatus } from '@/spec/workspace';

export enum WorkspaceDeclarationType {
	Workspace = 'workspace',
}

export interface WorkspaceDeclarationLocator {
	kind?: string;
	path?: string;
	url?: string;
	integrity?: string;
	repository?: string;
	revision?: string;
	manager?: string;
	package?: string;
	version?: string;
	registry?: string;
	command?: string;
}

export interface WorkspaceDocument {
	type: WorkspaceDeclarationType;
	name: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;
	locator?: WorkspaceDeclarationLocator;
	metadata?: Record<string, number[]>;
	members?: unknown[];
}

export interface Workspace {
	Artifact: StoreArtifact;
	Definition: StoreArtifactDefinition;
	Document: WorkspaceDocument;
}

export interface WorkspacePathRegistration {
	rootID: ArtifactRootID;
	path: string;
	sourceDisplayName?: string;
	workspaceName?: string;
}

export interface WorkspacePathRegistrationResult {
	source: StoreArtifactSourceSummary;
	workspace: Workspace;
	load: WorkspaceLoad;
}

export interface FilesystemSourceRegistration {
	rootID: ArtifactRootID;
	rootPath: string;
	sourceDisplayName: string;
}

export interface WorkspaceLoad {
	Workspace: Workspace;
	Members: unknown[];
	capabilities: CapabilityPlan;
}

export interface WorkspaceRefresh {
	Workspace: ArtifactRef;
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
	runtimeDisabled: boolean;
}

export interface WorkspacePromptContribution {
	Artifact: ArtifactRef;
	ArtifactRevision: number;
	DefinitionDigest: string;
	Kind: ArtifactKind;
	Name: string;
	Insert: SkillInsert;
	MediaType: string;
	Locator: ArtifactLocator;
	Content: string;
	OriginalBytes: number;
	IncludedBytes: number;
	Truncated: boolean;
}

export interface WorkspacePromptDecision {
	Artifact: ArtifactRef;
	Status: WorkspaceContextCompositionStatus;
	Code: string;
	OriginalBytes: number;
	IncludedBytes: number;
}

export interface WorkspacePromptPlan {
	Workspace: ArtifactRef;
	Contributions: WorkspacePromptContribution[];
	Prompt: string;
	Diagnostics: ArtifactDiagnostic[];
	Decisions: WorkspacePromptDecision[];
}

export interface WorkspaceSkill {
	Artifact: ArtifactRef;
	ArtifactRevision: number;
	DefinitionDigest: string;
	SourceID: ArtifactSourceID;
	Locator: ArtifactLocator;
	Document: SkillDocumentInput;
	RuntimeLocation: string;
	Version: string;
	RuntimeDisabled: boolean;
}

export interface WorkspaceSkillLoadPlan {
	Workspace: ArtifactRef;
	Skills: WorkspaceSkill[];
}

export interface WorkspaceMCPServer {
	artifact: ArtifactRef;
	server: MCPResolvedServer;
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
	workspace: Workspace;
	capabilities: CapabilityPlan;
	prompt: WorkspacePromptPlan;
	skills: WorkspaceSkillLoadPlan;
	mcpServers: WorkspaceMCPServerLoadPlan;
}

export interface WorkspaceManagementSnapshot {
	workspace: Workspace;
	load: WorkspaceLoad;
	artifacts: StoreArtifact[];
	root?: StoreArtifactRoot;
}
