import type {
	ArtifactAdoptionMode,
	ArtifactCollectionRef,
	ArtifactDefinitionView,
	ArtifactDiagnostic,
	ArtifactDigest,
	ArtifactKind,
	ArtifactLocator,
	ArtifactOccurrenceState,
	ArtifactRef,
	ArtifactSourceBinding,
	ArtifactSourceID,
	ArtifactState,
} from '@/spec/artifact';

export type WorkspaceRef = ArtifactCollectionRef;
export type WorkspaceRootID = string;
export type WorkspaceCollectionID = string;

export enum WorkspaceMode {
	Empty = 'empty',
	Filesystem = 'filesystem',
}

export enum WorkspaceAttachmentRole {
	Primary = 'primary',
	BuiltIn = 'built-in',
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

export enum WorkspaceSkillInsert {
	Instructions = 'instructions',
	UserMessage = 'user-message',
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

export interface WorkspaceArtifactView {
	artifact: ArtifactRef;
	revision: number;
	name: string;
	kind: ArtifactKind;
	enabled: boolean;
	state: ArtifactState;
	adoption: ArtifactAdoptionMode;
	resolvedDefinition?: ArtifactDigest;
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
	runtimeDisabled: boolean;
	diagnostics?: ArtifactDiagnostic[];
}

export interface WorkspaceSuppressionView {
	workspace: WorkspaceRef;
	binding: ArtifactSourceBinding;
	revision: number;
	createdAt: Date;
	modifiedAt: Date;
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

export interface WorkspaceLoadPlanItemView {
	artifact: WorkspaceArtifactView;
	definition: ArtifactDefinitionView;
	definitionDigest: ArtifactDigest;
	sourceID: ArtifactSourceID;
	sourceKind: string;
	locator: ArtifactLocator;
	catalogCurrent: boolean;
}

export interface WorkspaceLoadPlanView {
	workspace: WorkspaceRef;
	catalogRevision: number;
	items: WorkspaceLoadPlanItemView[];
	diagnostics?: ArtifactDiagnostic[];
}

export interface ResolveWorkspaceResourceResult {
	resource: WorkspaceResourceView;
	definition: ArtifactDefinitionView;
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
	createdAt: Date;
	modifiedAt: Date;
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

export interface CreateFilesystemWorkspaceBody {
	displayName: string;
	description?: string;
	rootPath: string;
	discovery: WorkspaceDiscovery;
}

export interface CreateEmptyWorkspaceBody {
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

export interface ReplaceWorkspacePrimarySourceBody {
	expectedCollectionRevision: number;
	previousSourceID: ArtifactSourceID;
	expectedPreviousAttachmentRevision: number;
	sourceID: ArtifactSourceID;
}

export interface SetWorkspacePrimarySourceBody {
	expectedCollectionRevision: number;
	previousSourceID?: ArtifactSourceID;
	expectedPreviousAttachmentRevision?: number;
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

export interface AdoptWorkspaceOccurrenceBody {
	expectedCatalogRevision: number;
	occurrence: WorkspaceOccurrenceRef;
	name?: string;
	enabled: boolean;
	settings: WorkspaceArtifactSettings;
}

export interface PinWorkspaceArtifactBody {
	expectedCollectionRevision: number;
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
