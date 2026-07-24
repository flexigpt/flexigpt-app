export type WorkspaceRootID = string;
export type WorkspaceSourceID = string;
export type WorkspaceRecordID = string;
export type WorkspaceDigest = string;
export type WorkspaceLocator = string;

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

export enum WorkspaceArtifactKind {
	Definition = 'workspace.definition',
	Context = 'workspace.context',
	Skill = 'workspace.skill',
}

export enum WorkspaceRecordState {
	Available = 'available',
	Missing = 'missing',
	Invalid = 'invalid',
	Incompatible = 'incompatible',
}

export enum WorkspaceOccurrenceState {
	Valid = 'valid',
	Invalid = 'invalid',
	Missing = 'missing',
}

export enum WorkspaceDiagnosticSeverity {
	Error = 'error',
	Warning = 'warning',
	Info = 'info',
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

export interface WorkspaceDiagnosticLocation {
	locator?: WorkspaceLocator;
	subresourceLocator?: WorkspaceLocator;
	line?: number;
	column?: number;
}

export interface WorkspaceDiagnostic {
	severity: WorkspaceDiagnosticSeverity;
	code: string;
	message: string;
	location?: WorkspaceDiagnosticLocation;
}

export interface WorkspaceDiscoveryRoot {
	root: WorkspaceLocator;
	recursive: boolean;
	includePatterns?: string[];
}

export interface WorkspaceDiscovery {
	additionalLocators?: WorkspaceLocator[];
	additionalRoots?: WorkspaceDiscoveryRoot[];
	includeReadme?: boolean;
}

export interface WorkspaceAttachmentSettings {
	recursive?: boolean;
	authoritative?: boolean;
}

export interface WorkspaceAttachmentView {
	sourceID: WorkspaceSourceID;
	revision: number;
	sourceDisplayName?: string;
	sourceKind?: string;
	path?: string;
	role: WorkspaceAttachmentRole;
	enabled: boolean;
	settings: WorkspaceAttachmentSettings;
}

/**
 * Local desktop Workspace projection.
 *
 * Filesystem paths are intentionally exposed to the local management UI.
 * Source configuration, raw source data, attachment data, and trust-reference
 * contents remain excluded.
 */
export interface WorkspaceView {
	rootID: WorkspaceRootID;
	revision: number;
	displayName: string;
	description?: string;
	enabled: boolean;
	mode: WorkspaceMode;
	primarySourceID?: WorkspaceSourceID;
	primaryPath?: string;
	discovery: WorkspaceDiscovery;
	attachments: WorkspaceAttachmentView[];
}

export interface WorkspaceRecordView {
	id: WorkspaceRecordID;
	revision: number;
	name: string;
	kind: WorkspaceArtifactKind;
	enabled: boolean;
	state: WorkspaceRecordState;
	resolvedDefinition?: WorkspaceDigest;
	sourceID: WorkspaceSourceID;
	locator: WorkspaceLocator;
	subresourceLocator?: WorkspaceLocator;
	runtimeDisabled: boolean;
	diagnostics?: WorkspaceDiagnostic[];
}

export interface WorkspaceResourceView {
	record: WorkspaceRecordView;
	definitionDigest: WorkspaceDigest;
	sourceID: WorkspaceSourceID;
	locator: WorkspaceLocator;
	catalogCurrent: boolean;
	projectionValid: boolean;
	diagnostics?: WorkspaceDiagnostic[];
}

export interface WorkspaceOccurrenceView {
	sourceID: WorkspaceSourceID;
	locator: WorkspaceLocator;
	subresourceLocator?: WorkspaceLocator;
	kind?: WorkspaceArtifactKind;
	logicalName?: string;
	logicalVersion?: string;
	definitionDigest?: WorkspaceDigest;
	sourceContentDigest?: WorkspaceDigest;
	state: WorkspaceOccurrenceState;
	recorded: boolean;
	recordID?: WorkspaceRecordID;
	diagnostics?: WorkspaceDiagnostic[];
}

export interface WorkspaceResourceGroupView {
	kind: WorkspaceArtifactKind;
	resources: WorkspaceResourceView[];
	unrecorded: WorkspaceOccurrenceView[];
}

export interface WorkspaceCatalogView {
	workspace: WorkspaceView;
	catalogRevision: number;
	catalogCurrent: boolean;
	diagnostics?: WorkspaceDiagnostic[];
	resources: WorkspaceResourceView[];
	groups: WorkspaceResourceGroupView[];
	occurrences: WorkspaceOccurrenceView[];
	validOccurrences: WorkspaceOccurrenceView[];
	invalidOccurrences: WorkspaceOccurrenceView[];
	missingOccurrences: WorkspaceOccurrenceView[];
	unrecordedOccurrences: WorkspaceOccurrenceView[];
	unresolvedRecords: WorkspaceRecordView[];
	unrecordedCount: number;
	unresolvedRecordCount: number;
}

export interface WorkspaceRefreshResult {
	rootID: WorkspaceRootID;
	catalogRevision: number;
	createdRecords: WorkspaceRecordID[];
	updatedRecords: WorkspaceRecordID[];
	diagnostics?: WorkspaceDiagnostic[];
	candidates: number;
}

export interface WorkspaceContextContribution {
	recordID: WorkspaceRecordID;
	definitionDigest: WorkspaceDigest;
	sourceID: WorkspaceSourceID;
	locator: WorkspaceLocator;
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
	recordID: WorkspaceRecordID;
	status: WorkspaceContextCompositionStatus;
	code?: string;
	originalBytes: number;
	includedBytes: number;
}

export interface WorkspaceContextLoadPlan {
	rootID: WorkspaceRootID;
	catalogRevision: number;
	contributions: WorkspaceContextContribution[];
	prompt: string;
	diagnostics?: WorkspaceDiagnostic[];
	decisions: WorkspaceContextDecision[];
	promptBytes: number;
}

export interface WorkspaceContextView {
	recordID: WorkspaceRecordID;
	recordRevision: number;
	definitionDigest: WorkspaceDigest;
	sourceID: WorkspaceSourceID;
	locator: WorkspaceLocator;
	name: string;
	role: WorkspaceContextRole;
	mediaType: WorkspaceContextMediaType;
	enabled: boolean;
	state: WorkspaceRecordState;
	catalogCurrent: boolean;
	runtimeDisabled: boolean;
	diagnostics?: WorkspaceDiagnostic[];
}

export interface WorkspaceContextInspectionView {
	rootID: WorkspaceRootID;
	catalogRevision: number;
	contributions: WorkspaceContextContribution[];
	diagnostics?: WorkspaceDiagnostic[];
}

export interface WorkspaceSkillArgument {
	name: string;
	description?: string;
	default?: string;
}

export interface WorkspaceSkillSummary {
	schemaVersion: string;
	id: WorkspaceRecordID;
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
	rootID: WorkspaceRootID;
	recordID: WorkspaceRecordID;
	definitionDigest: WorkspaceDigest;
	sourceID: WorkspaceSourceID;
	locator: WorkspaceLocator;
	skill: WorkspaceSkillSummary;
	markdownBody?: string;
	recordRevision: number;
	state: WorkspaceRecordState;
	projectionValid: boolean;
	catalogCurrent: boolean;
	runtimeDisabled: boolean;
	diagnostics?: WorkspaceDiagnostic[];
}

export interface WorkspaceSkillLoadView {
	rootID: WorkspaceRootID;
	catalogRevision: number;
	skills: WorkspaceSkillView[];
	diagnostics?: WorkspaceDiagnostic[];
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

export interface WorkspaceConversationResourceSelectionRef {
	recordID: WorkspaceRecordID;
	name?: string;
	locator?: WorkspaceLocator;
	definitionDigest?: WorkspaceDigest;
	recordRevision?: number;
}

export interface WorkspaceConversationSkillSelectionRef extends WorkspaceConversationResourceSelectionRef {
	identity: string;
	displayName?: string;
	insert?: WorkspaceSkillInsert;
}

export interface WorkspaceConversationSelection {
	rootID: WorkspaceRootID;
	displayName?: string;
	workspaceRevision?: number;
	catalogRevision?: number;
	contextRefs?: WorkspaceConversationResourceSelectionRef[];
	skillRefs?: WorkspaceConversationSkillSelectionRef[];
}

export interface WorkspaceConversationContextUsage {
	recordID: WorkspaceRecordID;
	name?: string;
	locator?: WorkspaceLocator;
	selectedDefinitionDigest?: WorkspaceDigest;
	usedDefinitionDigest?: WorkspaceDigest;
	status: WorkspaceContextCompositionStatus;
	code?: string;
	originalBytes?: number;
	includedBytes?: number;
	changed?: boolean;
	diagnostics?: WorkspaceDiagnostic[];
}

export interface WorkspaceConversationSkillUsage {
	recordID: WorkspaceRecordID;
	identity: string;
	name?: string;
	displayName?: string;
	locator?: WorkspaceLocator;
	selectedDefinitionDigest?: WorkspaceDigest;
	usedDefinitionDigest?: WorkspaceDigest;
	status: WorkspaceConversationSkillUsageStatus;
	changed?: boolean;
	sessionAvailable?: boolean;
	active?: boolean;
	advertised?: boolean;
	diagnostics?: WorkspaceDiagnostic[];
}

export interface WorkspaceConversationUsage {
	rootID: WorkspaceRootID;
	displayName?: string;
	workspaceRevision?: number;
	catalogRevision?: number;
	status: WorkspaceConversationSelectionStatus;
	contexts?: WorkspaceConversationContextUsage[];
	skills?: WorkspaceConversationSkillUsage[];
	diagnostics?: WorkspaceDiagnostic[];
}

export function workspaceSkillIdentity(rootID: WorkspaceRootID, recordID: WorkspaceRecordID): string {
	return `workspace/${rootID}/${recordID}`;
}

export function parseWorkspaceSkillIdentity(
	identity?: string
): { rootID: WorkspaceRootID; recordID: WorkspaceRecordID } | undefined {
	const parts = identity?.trim().split('/') ?? [];
	if (parts.length !== 3 || parts[0] !== 'workspace' || !parts[1] || !parts[2]) {
		return undefined;
	}

	return { rootID: parts[1], recordID: parts[2] };
}

export function isWorkspaceSkillIdentity(identity?: string): boolean {
	return parseWorkspaceSkillIdentity(identity) !== undefined;
}

export interface CreateFilesystemWorkspacePayload {
	displayName: string;
	description?: string;
	rootPath: string;
	discovery: WorkspaceDiscovery;
}

export interface CreateEmptyWorkspacePayload {
	displayName: string;
	description?: string;
	discovery: WorkspaceDiscovery;
}

export interface UpdateWorkspacePayload {
	expectedRevision: number;
	displayName: string;
	description?: string;
	enabled: boolean;
	discovery: WorkspaceDiscovery;
}

export interface WorkspaceAttachmentPayload {
	role: WorkspaceAttachmentRole;
	enabled: boolean;
	settings: WorkspaceAttachmentSettings;
}

export interface AttachWorkspaceSourcePayload extends WorkspaceAttachmentPayload {
	expectedRootRevision: number;
}

export interface UpdateWorkspaceAttachmentPayload extends WorkspaceAttachmentPayload {
	expectedRootRevision: number;
	expectedAttachmentRevision: number;
}

export interface DeleteWorkspaceResult {
	rootID: WorkspaceRootID;
	revision: number;
}
