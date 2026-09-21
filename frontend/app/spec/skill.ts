import type {
	ArtifactAddress,
	ArtifactAdoptionMode,
	ArtifactCollectionRef,
	ArtifactDiagnostic,
	ArtifactDigest,
	ArtifactKind,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceBinding,
	ArtifactSourceID,
	ArtifactState,
	CapabilityPlan,
	StoreArtifact,
	StoreArtifactAddress,
	StoreArtifactSourceSummary,
	StoreManagedPackageFile,
} from '@/spec/artifact';
import type { ArtifactMembershipView, CollectionCapabilityPlan, CollectionView } from '@/spec/collection';
import type { ToolOutputUnion } from '@/spec/tool';

export type SkillRef = ArtifactRef;
export type SkillBundleRef = ArtifactCollectionRef;

export const SKILLS_AUTOEXEC_TOOL_CHOICES = new Set([
	'builtin.skills-load',
	'builtin.skills-unload',
	'builtin.skills-readresource',
]);

export enum SkillSessionSyncMode {
	None = 'none',
	IfSessionExists = 'if-session-exists',
	EnsureIfEnabled = 'ensure-if-enabled',
}

export enum SkillInsert {
	Instructions = 'instructions',
	UserMessage = 'user-message',
}

export enum SkillBundleAttachmentRole {
	Managed = 'managed',
	BuiltIn = 'builtin',
	External = 'external',
	Imported = 'imported',
	Library = 'library',
}

export interface SkillSelection {
	artifact: ArtifactRef;
	preLoadAsActive: boolean;
	useAsInstructions: boolean;
}

export interface SkillArgument {
	name: string;
	description?: string;
	default?: string;
}

interface SkillBundleAttachmentView {
	sourceID: ArtifactSourceID;
	revision: number;
	role: SkillBundleAttachmentRole;
	enabled: boolean;
	sourceDisplayName?: string;
	sourceKind?: string;
}

export interface SkillArtifactView {
	artifact: ArtifactRef;
	address: ArtifactAddress;
	revision: number;
	name: string;
	kind: ArtifactKind;
	enabled: boolean;
	adoption: ArtifactAdoptionMode;
	state: ArtifactState;
	binding: ArtifactSourceBinding;
	definitionDigest?: ArtifactDigest;
	diagnostics?: ArtifactDiagnostic[];
	createdAt: string;
	modifiedAt: string;
}

export interface SkillDocumentInput {
	name: string;
	displayName?: string;
	description: string;
	insert: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	markdownBody: string;
	rawFrontmatter?: Record<string, unknown>;
}

/**
 * Editable managed Skill source projected from its canonical definition.
 * This deliberately exposes no source configuration or filesystem path.
 */
export interface ManagedSkillDocumentView {
	artifact: SkillArtifactView;
	document: SkillDocumentInput;
}

export enum RuntimeSkillActivity {
	Any = 'any',
	Active = 'active',
	Inactive = 'inactive',
}

export interface SkillResourceInfo {
	hasResources: boolean;
	totalCount: number;
	locations?: string[];
	moreLocations: boolean;
}

export interface RuntimeSkillListItem {
	skillRef: ArtifactRef;
	name?: string;
	displayName?: string;
	type?: string;
	description?: string;
	digest?: ArtifactDigest;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	sourceTags?: string[];
	resources: SkillResourceInfo;
	rawFrontmatter?: Record<string, unknown>;
	warnings?: string[];
	isActive?: boolean;
	errorMessage?: string;
}

/**
 * Runtime-native skill identity. Provider type remains a string because
 * Agent Skills providers are registry-extensible.
 */
export interface RuntimeSkillDefinition {
	type: string;
	name: string;
	location: string;
}

export interface RuntimeSkillSessionOptions {
	closeSessionID?: string;
	maxActivePerSession?: number;

	/**
	 * Undefined preserves the native unrestricted Agent Skills session path.
	 * An explicit empty array creates a deny-all allowed-skills session.
	 */
	allowedSkills?: RuntimeSkillDefinition[];
	activeSkills?: RuntimeSkillDefinition[];
}

export interface RuntimeSkillSession {
	sessionID: string;
	activeSkills: RuntimeSkillDefinition[];
}

export interface RuntimeSkillRecord {
	def: RuntimeSkillDefinition;
	name: string;
	description: string;
	displayName?: string;
	insert: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	resources: SkillResourceInfo;
	rawFrontmatter?: Record<string, unknown>;
	warnings?: string[];
	digest?: ArtifactDigest;
}

export interface RuntimeSkillRenderResult {
	text: string;
	insert: SkillInsert;
	name: string;
	description?: string;
	displayName?: string;
	tags?: string[];
	resources: SkillResourceInfo;
	arguments?: SkillArgument[];
	appliedArguments?: Record<string, string>;
	rawFrontmatter?: Record<string, unknown>;
	warnings?: string[];
}

/**
 * A selectable installed Skill Bundle Artifact for management UIs such as
 * Assistant Presets. Workspace Skills are deliberately not represented here:
 * they are selected through a Workspace conversation selection instead.
 */
export interface AssistantSkillOption {
	key: string;
	sel: SkillSelection;
	skillDefinition: Skill;

	bundleSlug: string;
	bundleDisplayName: string;

	isBuiltIn: boolean;
	isBundleEnabled: boolean;
	isSkillEnabled: boolean;
	isSelectable: boolean;
	availabilityReason?: string;
	label: string;
}

export interface RenderSkillResponse {
	text: string;
	insert: SkillInsert;
	name: string;
	description?: string;
	displayName?: string;
	sourceTags?: string[];
	resources: SkillResourceInfo;
	arguments?: SkillArgument[];
	appliedArguments?: Record<string, string>;
	rawFrontmatter?: Record<string, unknown>;
	warnings?: string[];
}

export interface InvokeSkillToolResponse {
	outputs?: ToolOutputUnion[];
	meta?: Record<string, unknown>;
	isBuiltIn: boolean;
	isError?: boolean;
	errorMessage?: string;
}

/**
 * Management-only projection over Artifact Store Skill entities.
 *
 * Durable identity remains `ArtifactRef` and `SkillBundleRef`. These views
 * exist so management components do not need to duplicate joins between
 * Bundle, Artifact, and runtime metadata.
 */
export enum SkillType {
	FS = 'fs',
	EmbeddedFS = 'embeddedfs',
}

export enum SkillPresenceStatus {
	Present = 'present',
	Missing = 'missing',
	Error = 'error',
	Unknown = 'unknown',
}

interface SkillPresence {
	status: SkillPresenceStatus;
	lastCheckedAt?: string;
	lastSeenAt?: string;
	missingSince?: string;
	lastCheckError?: string;
}

export interface SkillArtifactCreateInput {
	name: string;
	displayName?: string;
	description: string;
	insert: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	markdownBody: string;
	isEnabled: boolean;
}

export interface Skill {
	schemaVersion: string;
	id: string;
	ref: ArtifactRef;
	revision: number;
	slug: string;
	name: string;
	displayName?: string;
	description?: string;
	type: SkillType;
	location: string;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	resources: SkillResourceInfo;
	digest?: ArtifactDigest;
	rawFrontmatter?: Record<string, unknown>;
	runtimeWarnings?: string[];
	runtimeError?: string;
	presence?: SkillPresence;
	isEnabled: boolean;
	isBuiltIn: boolean;
	isManaged: boolean;
	adoption: ArtifactAdoptionMode;
	state: ArtifactState;
	diagnostics?: ArtifactDiagnostic[];
	createdAt: string;
	modifiedAt: string;
}

export interface SkillBundle {
	schemaVersion: string;
	id: string;
	rootID: string;
	ref: SkillBundleRef;
	revision: number;
	slug: string;
	logicalVersion?: string;
	labels?: Record<string, string>;
	managedSourceID?: ArtifactSourceID;
	displayName?: string;
	description?: string;
	isEnabled: boolean;
	isBuiltIn: boolean;
	isEditable: boolean;
	isDeletable: boolean;
	isBaseline: boolean;
	sourceID: ArtifactSourceID;

	attachments: SkillBundleAttachmentView[];
	createdAt: string;
	modifiedAt: string;
}

export interface SkillListItem {
	bundleID: string;
	bundleSlug: string;
	skillSlug: string;
	skillDefinition: Skill;
}

export interface ManagedSkillCreateRequest {
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	skillName: string;
	skillMD?: number[];
	files?: StoreManagedPackageFile[];
	enabled: boolean;
}

export interface ManagedSkillCreateResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	collection: CollectionView;
	membershipCreated: boolean;
}

export interface ManagedSkillReplaceRequest {
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	artifact: ArtifactRef;
	expectedArtifactRevision: number;
	skillName: string;
	skillMD?: number[];
	files?: StoreManagedPackageFile[];
	enabled: boolean;
}

export interface ManagedSkillReplaceResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	collection: CollectionView;
}

export interface StoreManagedSkillDocument {
	artifact: StoreArtifact;
	document: SkillDocumentInput;
}

export interface SkillDirectoryRegistration {
	rootID: ArtifactRootID;
	rootPath: string;
	sourceDisplayName: string;
}

export interface SkillPathRegistration {
	rootID: ArtifactRootID;
	path: string;
	sourceDisplayName?: string;
	enabled: boolean;
}

export interface SkillPathRegistrationResult {
	source: StoreArtifactSourceSummary;
	artifact: StoreArtifact;
}

/**
 * Aggregate-owned Artifact Skill filter. This is the frontend application
 * contract for `SkillAggregateWrapper`, not a runtime-native filter.
 */
export interface ArtifactSkillFilter {
	types?: string[];
	inserts?: SkillInsert[];
	namePrefix?: string;
	locationPrefix?: string;
	allowArtifacts?: ArtifactRef[];
	sessionID?: string;
	activity?: RuntimeSkillActivity;
}

/**
 * Direct result of `SkillAggregateWrapper.ResolveArtifactSkill`.
 *
 * The uppercase property names are intentional. The Go aggregate result has
 * no JSON field tags, so Wails preserves exported Go field names.
 */
export interface ResolvedArtifactSkill {
	Artifact: ArtifactRef;
	Definition: RuntimeSkillDefinition;
	Version: string;
	Enabled: boolean;
}

/**
 * Direct result of `SkillAggregateWrapper.DescribeArtifactSkill`.
 *
 * The uppercase property names are intentional for the same reason as
 * `ResolvedArtifactSkill`.
 */
export interface ArtifactSkillSummary {
	Artifact: ArtifactRef;
	IsEnabled: boolean;
	Insert: SkillInsert;
	HasArguments: boolean;
	HasResources: boolean;
}

/**
 * Runtime-native prompt filter. It intentionally accepts Skill definitions,
 * never ArtifactRefs.
 */
export interface RuntimeSkillPromptFilter {
	types?: string[];
	namePrefix?: string;
	locationPrefix?: string;
	allowSkills?: RuntimeSkillDefinition[];
	sessionID?: string;
	activity?: RuntimeSkillActivity;
}

/**
 * Runtime-native list filter. It intentionally accepts Skill definitions,
 * never ArtifactRefs.
 */
export interface RuntimeSkillListFilter {
	types?: string[];
	inserts?: SkillInsert[];
	namePrefix?: string;
	locationPrefix?: string;
	allowSkills?: RuntimeSkillDefinition[];
	sessionID?: string;
	activity?: RuntimeSkillActivity;
}

/**
 * Artifact-oriented session request exposed by Skill management.
 */
export interface ArtifactSkillSessionOptions {
	closeSessionID?: string;
	maxActivePerSession?: number;
	allowArtifacts?: ArtifactRef[];
	activeArtifacts?: ArtifactRef[];
}

/**
 * Artifact-oriented session result exposed by Skill management.
 */
export interface ArtifactSkillSession {
	sessionID: string;
	activeArtifacts: ArtifactRef[];
}

/**
 * Runtime record joined with its durable ArtifactRef by Skill management.
 */
export interface ArtifactRuntimeSkillListItem extends RuntimeSkillListItem {
	skillRef: ArtifactRef;
}

export interface SkillCollectionManagementView {
	collection: CollectionView;
	capabilities: CollectionCapabilityPlan;
}

export interface SkillManagementView {
	artifact: StoreArtifact;
	memberships: ArtifactMembershipView[];
	capabilities: CapabilityPlan;
	runtimeSummary?: ArtifactSkillSummary;
	runtimeError?: string;
}

export interface SkillRuntimeManagementView {
	session?: RuntimeSkillSession;
	skills: RuntimeSkillRecord[];
	rendered?: RuntimeSkillRenderResult;
	invocation?: InvokeSkillToolResponse;
}
