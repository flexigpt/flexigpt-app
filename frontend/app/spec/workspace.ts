import type {
	ArtifactCollectionRef,
	ArtifactDiagnostic,
	ArtifactDigest,
	ArtifactKind,
	ArtifactLocator,
	ArtifactOccurrenceState,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceBinding,
	ArtifactSourceID,
	ArtifactSourceSummary,
	ArtifactState,
	CapabilityPlan,
	StoreArtifact,
	StoreArtifactDefinition,
	StoreArtifactRoot,
	StoreArtifactSourceSummary,
} from '@/spec/artifact';
import type { MCPResolvedServer } from '@/spec/mcp';
import type { SkillDocumentInput, SkillInsert } from '@/spec/skill';
import { SkillInsert as WorkspaceSkillInsert } from '@/spec/skill';

export { WorkspaceSkillInsert };

export type WorkspaceRef = ArtifactCollectionRef;

export enum WorkspaceMode {
	Empty = 'empty',
	Filesystem = 'filesystem',
}

export enum WorkspaceAttachmentRole {
	Primary = 'primary',
	Library = 'library',
	AttachedPackage = 'attached-package',
	Overlay = 'overlay',
}

export enum WorkspaceContextRole {
	AgentInstructions = 'agent-instructions',
	AssistantInstructions = 'assistant-instructions',
	ProjectReadme = 'project-readme',
	ProjectContext = 'project-context',
}

export enum WorkspaceContextMediaType {
	Markdown = 'text/markdown',
}

export enum WorkspaceContextCompositionStatus {
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

export enum WorkspaceConversationSkillUsageStatus {
	Available = 'available',
	Unavailable = 'unavailable',
}

export interface WorkspaceDiscoveryRoot {
	root: ArtifactLocator;
	recursive: boolean;
	includePatterns?: string[];
}

export interface WorkspaceDiscovery {
	additionalLocators?: ArtifactLocator[];
	additionalRoots?: WorkspaceDiscoveryRoot[];
	includeReadme?: boolean;
}

export interface WorkspaceAttachmentSettings {
	recursive?: boolean;
	authoritative?: boolean;
}

export interface WorkspaceAttachmentView {
	sourceID: ArtifactSourceID;
	revision: number;
	role: WorkspaceAttachmentRole;
	enabled: boolean;
	sourceDisplayName?: string;
	sourceKind?: string;
	path?: string;
	settings: WorkspaceAttachmentSettings;
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceView {
	workspace: WorkspaceRef;
	revision: number;
	displayName: string;
	description?: string;
	enabled: boolean;
	mode: WorkspaceMode;
	primarySourceID?: ArtifactSourceID;
	primaryPath?: string;
	discovery: WorkspaceDiscovery;
	attachments: WorkspaceAttachmentView[];
}

export interface WorkspaceArtifactSettings {
	runtimeDisabled: boolean;
}

export interface WorkspaceOccurrenceRef {
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
}

export interface WorkspaceSuppressionView {
	workspace: WorkspaceRef;
	binding: ArtifactSourceBinding;
	revision: number;
	createdAt: string;
	modifiedAt: string;
}

export interface WorkspaceOccurrenceView {
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
	kind?: ArtifactKind;
	logicalName?: string;
	logicalVersion?: string;
	definitionDigest?: ArtifactDigest;
	sourceContentDigest?: ArtifactDigest;
	state: ArtifactOccurrenceState;
	recorded: boolean;
	artifact?: ArtifactRef;
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceResourceView {
	artifact: WorkspaceArtifactView;
	definitionDigest: ArtifactDigest;
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	catalogCurrent: boolean;
	projectionValid: boolean;
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceResourceGroupView {
	kind: ArtifactKind;
	resources: WorkspaceResourceView[];
	unrecorded: WorkspaceOccurrenceView[];
}

export interface WorkspaceCatalogView {
	workspace: WorkspaceView;
	catalogRevision: number;
	catalogCurrent: boolean;
	diagnostics?: ArtifactDiagnostic[];
	resources: WorkspaceResourceView[];
	groups: WorkspaceResourceGroupView[];
	occurrences: WorkspaceOccurrenceView[];
	validOccurrences: WorkspaceOccurrenceView[];
	invalidOccurrences: WorkspaceOccurrenceView[];
	missingOccurrences: WorkspaceOccurrenceView[];
	unrecordedOccurrences: WorkspaceOccurrenceView[];
	unresolvedArtifacts: WorkspaceArtifactView[];
	unrecordedCount: number;
	unresolvedArtifactCount: number;
}

export interface WorkspaceRefreshResult {
	workspace: WorkspaceRef;
	catalogRevision: number;
	createdArtifacts: ArtifactRef[];
	updatedArtifacts: ArtifactRef[];
	diagnostics?: ArtifactDiagnostic[];
	candidates: number;
}

export interface WorkspaceContextContribution {
	artifact: ArtifactRef;
	recordRevision: number;
	definitionDigest: ArtifactDigest;
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	name: string;
	role: WorkspaceContextRole;
	mediaType: WorkspaceContextMediaType;
	content: string;
	conventionOrder: number;
	originalBytes: number;
	includedBytes: number;
	truncated: boolean;
}

export interface WorkspaceContextDecision {
	artifact: ArtifactRef;
	status: WorkspaceContextCompositionStatus;
	code?: string;
	originalBytes: number;
	includedBytes: number;
}

export interface WorkspaceContextView {
	artifact: ArtifactRef;
	recordRevision: number;
	definitionDigest: ArtifactDigest;
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	name: string;
	role: WorkspaceContextRole;
	mediaType: WorkspaceContextMediaType;
	enabled: boolean;
	state: ArtifactState;
	catalogCurrent: boolean;
	projectionValid: boolean;
	runtimeDisabled: boolean;
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceContextInspectionView {
	workspace: WorkspaceRef;
	catalogRevision: number;
	contributions: WorkspaceContextContribution[];
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceContextLoadPlan {
	workspace: WorkspaceRef;
	catalogRevision: number;
	contributions: WorkspaceContextContribution[];
	prompt: string;
	diagnostics?: ArtifactDiagnostic[];
	decisions: WorkspaceContextDecision[];
	promptBytes: number;
}

export interface WorkspaceSkillArgument {
	name: string;
	description?: string;
	default?: string;
}

export interface WorkspaceSkillSummary {
	schemaVersion: string;
	id: string;
	slug: string;
	name: string;
	displayName: string;
	description: string;
	tags?: string[];
	insert: WorkspaceSkillInsert;
	arguments?: WorkspaceSkillArgument[];
	isEnabled: boolean;
	createdAt: string;
	modifiedAt: string;
}

export interface WorkspaceSkillView {
	workspace: WorkspaceRef;
	artifact: ArtifactRef;
	definitionDigest: ArtifactDigest;
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	skill: WorkspaceSkillSummary;
	markdownBody?: string;
	recordRevision: number;
	state: ArtifactState;
	projectionValid: boolean;
	catalogCurrent: boolean;
	runtimeDisabled: boolean;
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceSkillLoadView {
	workspace: WorkspaceRef;
	catalogRevision: number;
	skills: WorkspaceSkillView[];
	diagnostics?: ArtifactDiagnostic[];
}

export interface CreateFilesystemWorkspaceInput {
	displayName: string;
	description?: string;
	rootPath: string;
	discovery: WorkspaceDiscovery;
}

export interface CreateEmptyWorkspaceInput {
	displayName: string;
	description?: string;
	discovery: WorkspaceDiscovery;
}

export interface UpdateWorkspaceBody {
	expectedRevision: number;
	displayName: string;
	description?: string;
	enabled: boolean;
	discovery: WorkspaceDiscovery;
}

export interface SetWorkspacePrimarySourceBody {
	expectedCollectionRevision: number;
	sourceID?: ArtifactSourceID;
	clear?: boolean;
}

export interface AttachWorkspaceSourceBody {
	expectedCollectionRevision: number;
	sourceID: ArtifactSourceID;
	role: WorkspaceAttachmentRole;
	enabled: boolean;
	settings: WorkspaceAttachmentSettings;
}

export interface UpdateWorkspaceAttachmentBody {
	expectedCollectionRevision: number;
	expectedAttachmentRevision: number;
	role: WorkspaceAttachmentRole;
	enabled: boolean;
	settings: WorkspaceAttachmentSettings;
}

export interface DetachWorkspaceSourceBody {
	expectedCollectionRevision: number;
	expectedAttachmentRevision: number;
}

export interface AdoptWorkspaceOccurrenceBody {
	expectedCatalogRevision: number;
	occurrence: WorkspaceOccurrenceRef;
	artifactID: string;
	name?: string;
	enabled: boolean;
	settings: WorkspaceArtifactSettings;
}

export interface PinWorkspaceArtifactBody {
	expectedCollectionRevision: number;
	artifactID: string;
	binding: ArtifactSourceBinding;
	name: string;
	enabled: boolean;
	settings: WorkspaceArtifactSettings;
}

export interface SuppressWorkspaceBindingBody {
	expectedCollectionRevision: number;
	binding: ArtifactSourceBinding;
}

export interface SetWorkspaceArtifactEnabledBody {
	expectedRevision: number;
	enabled: boolean;
}

export interface SetWorkspaceArtifactRuntimeDisabledBody {
	expectedRevision: number;
	runtimeDisabled: boolean;
}

export interface UnadoptWorkspaceArtifactBody {
	expectedRevision: number;
	suppress: boolean;
}

export interface UnadoptWorkspaceArtifactResult {
	artifact: ArtifactRef;
}

export interface UnsuppressWorkspaceBindingResult {
	workspace: WorkspaceRef;
	binding: ArtifactSourceBinding;
}

export interface RetireWorkspaceResult {
	workspace: WorkspaceRef;
	revision: number;
}

export interface RegisterWorkspaceDirectoryInput {
	expectedCollectionRevision: number;
	displayName: string;
	rootPath: string;
	role: WorkspaceAttachmentRole;
	settings: WorkspaceAttachmentSettings;
}

export interface WorkspaceDirectoryRegistrationResult {
	source: ArtifactSourceSummary;
	workspace: WorkspaceView;
}

export type WorkspaceSourceSummary = ArtifactSourceSummary;

/**
 * Persisted conversation selection for the Artifact Store Workspace model.
 */
export interface WorkspaceConversationResourceSelectionRef {
	artifact: ArtifactRef;
	name?: string;
	locator?: ArtifactLocator;
	definitionDigest?: ArtifactDigest;
	artifactRevision?: number;
}

export interface WorkspaceConversationSkillSelectionRef extends WorkspaceConversationResourceSelectionRef {
	displayName?: string;
	insert?: WorkspaceSkillInsert;
}

export interface WorkspaceConversationSelection {
	workspace: WorkspaceRef;
	displayName?: string;
	workspaceRevision?: number;
	catalogRevision?: number;
	contextRefs?: WorkspaceConversationResourceSelectionRef[];
	skillRefs?: WorkspaceConversationSkillSelectionRef[];
}

export interface WorkspaceConversationContextUsage {
	artifact: ArtifactRef;
	name?: string;
	locator?: ArtifactLocator;
	selectedDefinitionDigest?: ArtifactDigest;
	usedDefinitionDigest?: ArtifactDigest;
	usedArtifactRevision?: number;
	status: WorkspaceContextCompositionStatus;
	code?: string;
	originalBytes?: number;
	includedBytes?: number;
	changed?: boolean;
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceConversationSkillUsage {
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
	workspace: WorkspaceRef;
	displayName?: string;
	workspaceRevision?: number;
	catalogRevision?: number;
	status: WorkspaceConversationSelectionStatus;
	contexts?: WorkspaceConversationContextUsage[];
	skills?: WorkspaceConversationSkillUsage[];
	diagnostics?: ArtifactDiagnostic[];
}

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
