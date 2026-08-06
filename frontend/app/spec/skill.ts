import type {
	ArtifactAddress,
	ArtifactAdoptionMode,
	ArtifactCollectionID,
	ArtifactCollectionRef,
	ArtifactDiagnostic,
	ArtifactDigest,
	ArtifactID,
	ArtifactKind,
	ArtifactLocator,
	ArtifactRef,
	ArtifactSourceBinding,
	ArtifactSourceID,
	ArtifactState,
	ManagedSourcePackageFile,
} from '@/spec/artifact';
import type { ToolOutputUnion } from '@/spec/tool';

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

export type SkillBundleRef = ArtifactCollectionRef;

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

interface SkillBundleAttachmentDraft {
	sourceID: ArtifactSourceID;
	role: SkillBundleAttachmentRole;
	enabled: boolean;
	discoveryRoot: ArtifactLocator;
	expectedMemberDigests?: Record<ArtifactLocator, ArtifactDigest>;
}

interface SkillBundleAttachmentView {
	sourceID: ArtifactSourceID;
	revision: number;
	role: SkillBundleAttachmentRole;
	enabled: boolean;
	sourceDisplayName?: string;
	sourceKind?: string;
}

export interface SkillBundleView {
	bundle: SkillBundleRef;
	revision: number;
	displayName: string;
	description?: string;
	enabled: boolean;
	retiredAt?: Date;
	logicalName: string;
	logicalVersion?: string;
	labels?: Record<string, string>;
	portableDefinitionDigest?: ArtifactDigest;
	managedSourceID?: ArtifactSourceID;
	attachments: SkillBundleAttachmentView[];
	createdAt: Date;
	modifiedAt: Date;
}

export interface RetireSkillBundleResult {
	bundle: SkillBundleRef;
	revision: number;
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
	createdAt: Date;
	modifiedAt: Date;
}

export interface CreateSkillBundleBody {
	collectionID: ArtifactCollectionID;
	displayName: string;
	description?: string;
	enabled: boolean;
	logicalName: string;
	logicalVersion?: string;
	labels?: Record<string, string>;
	portableDefinitionDigest?: ArtifactDigest;

	// Requests Artifact Store to provision and exclusively assign a managed
	// source to this bundle. It must not also appear in `attachments`.
	managedSourceID?: ArtifactSourceID;

	attachments?: SkillBundleAttachmentDraft[];
}

export interface UpdateSkillBundleBody {
	expectedRevision: number;
	displayName: string;
	description?: string;
	enabled: boolean;
}

export interface AttachSkillBundleSourceBody extends SkillBundleAttachmentDraft {
	expectedCollectionRevision: number;
}

interface SkillOccurrenceRef {
	sourceID: ArtifactSourceID;
	locator: ArtifactLocator;
	subresourceLocator?: ArtifactLocator;
}

export interface AdoptSkillBody {
	expectedCatalogRevision: number;
	occurrence: SkillOccurrenceRef;
	artifactID: ArtifactID;
	name: string;
	enabled: boolean;
}

export interface PinSkillBody {
	expectedCollectionRevision: number;
	artifactID: ArtifactID;
	binding: ArtifactSourceBinding;
	name: string;
	enabled: boolean;
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

interface CreateManagedSkillCommon {
	expectedCollectionRevision: number;
	/**
	 * Required when replacing an existing managed skill package. New managed
	 * skills omit this value.
	 */
	expectedArtifactRevision?: number;
	artifactID: ArtifactID;
	skillName: string;
	files?: ManagedSourcePackageFile[];
	enabled: boolean;
}

/**
 * The backend accepts exactly one semantic authoring input. `files`, when
 * present, are the complete package-relative native payload and must contain
 * the same `SKILL.md` bytes as `skillMD`.
 */
export type CreateManagedSkillBody = CreateManagedSkillCommon &
	(
		| {
				skillMD: Uint8Array;
				document?: never;
		  }
		| {
				skillMD?: never;
				document: SkillDocumentInput;
		  }
	);

export interface CreateManagedSkillResult {
	artifact: SkillArtifactView;
	address: ArtifactAddress;
}

/**
 * Editable managed Skill source projected from its canonical definition.
 * This deliberately exposes no source configuration or filesystem path.
 */
export interface ManagedSkillDocumentView {
	artifact: SkillArtifactView;
	document: SkillDocumentInput;
}

export interface SetSkillEnabledBody {
	expectedRevision: number;
	enabled: boolean;
}

export enum RuntimeSkillActivity {
	Any = 'any',
	Active = 'active',
}

export interface RuntimeSkillFilter {
	types?: string[];
	inserts?: SkillInsert[];
	allowArtifacts: ArtifactRef[];
	locationPrefix?: string;
	sessionID?: string;
	activity?: RuntimeSkillActivity;
}

export interface CreateSkillSessionOptions {
	closeSessionID?: string;
	maxActivePerSession?: number;
	allowArtifacts: ArtifactRef[];
	activeArtifacts?: ArtifactRef[];
}

export interface SkillSession {
	sessionID: string;
	activeArtifacts: ArtifactRef[];
}

interface SkillResourceSummary {
	hasResources: boolean;
	totalCount: number;
	locations?: string[];
	moreLocations: boolean;
}

export interface RuntimeSkillListItem {
	artifact: ArtifactRef;
	name?: string;
	displayName?: string;
	type?: string;
	description?: string;
	definitionDigest?: ArtifactDigest;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	sourceTags?: string[];
	resources: SkillResourceSummary;
	rawFrontmatter?: Record<string, unknown>;
	warnings?: string[];
	isActive?: boolean;
	errorMessage?: string;
}

/**
 * A selectable installed Skill Bundle Artifact for management UIs such as
 * Assistant Presets. Workspace Skills are deliberately not represented here:
 * they are selected through a Workspace conversation selection instead.
 */
export interface AssistantSkillOption {
	key: string;
	sel: SkillSelection;
	skillDefinition: RuntimeSkillListItem;

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
	resources: SkillResourceSummary;
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
	lastCheckedAt?: Date;
	lastSeenAt?: Date;
	missingSince?: Date;
	lastCheckError?: string;
}

export type SkillResourceInfo = SkillResourceSummary;

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
	presence?: SkillPresence;
	isEnabled: boolean;
	isBuiltIn: boolean;
	isManaged: boolean;
	adoption: ArtifactAdoptionMode;
	state: ArtifactState;
	diagnostics?: ArtifactDiagnostic[];
	createdAt: Date;
	modifiedAt: Date;
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
	attachments: SkillBundleAttachmentView[];
	createdAt: Date;
	modifiedAt: Date;
}

export interface SkillListItem {
	bundleID: string;
	bundleSlug: string;
	skillSlug: string;
	skillDefinition: Skill;
}

export type SkillRef = ArtifactRef;
