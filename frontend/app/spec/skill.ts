import type { ToolOutputUnion } from '@/spec/tool';

export const SKILLS_AUTOEXEC_TOOL_CHOICES = new Set([
	'builtin.skills-load',
	'builtin.skills-unload',
	'builtin.skills-readresource',
]);

export type SkillInsert = 'instructions' | 'user-message';

export enum SkillSessionSyncMode {
	None = 'none',
	IfSessionExists = 'if-session-exists',
	EnsureIfEnabled = 'ensure-if-enabled',
}

/**
 * Source classification for a normalized runtime-facing SkillRef.
 *
 * `SkillRef` remains wire-compatible with the current Go API. Consumers must
 * use the identity helpers instead of inferring source from missing fields.
 */
export enum SkillRefKind {
	Installed = 'installed',
	Workspace = 'workspace',
	Invalid = 'invalid',
}

export interface InstalledSkillRef {
	bundleID: string;
	skillSlug: string;
	skillID: string;
	identity?: never;
}

export interface WorkspaceSkillRef {
	identity: string;
	bundleID?: never;
	skillSlug?: never;
	skillID?: never;
}

export interface InstalledSkillSelection {
	skillRef: InstalledSkillRef;
	preLoadAsActive: boolean;
	useAsInstructions: boolean;
}

// Store identity for selection/persistence (NOT runtime identity).
export interface SkillRef {
	/**
	 * External provider identity, for example workspace/<rootID>/<recordID>.
	 * Mutually exclusive with the installed Skill Store fields below.
	 */
	identity?: string;
	bundleID?: string;
	skillSlug?: string;
	skillID?: string;
}

export enum SkillProviderOrigin {
	Installed = 'installed',
	Workspace = 'workspace',
}

/**
 * @public
 */
export interface SkillProviderDiagnostic {
	severity: 'error' | 'warning' | 'info';
	code: string;
	message: string;
	location?: {
		locator?: string;
		subresourceLocator?: string;
		line?: number;
		column?: number;
	};
}

/**
 * Read-only aggregate provider projection.
 *
 * This is intentionally separate from `SkillListItem`: Workspace Skills are
 * not installed-store records and must not be forced into that model.
 */
export interface ProvidedSkill {
	identity: string;
	origin: SkillProviderOrigin;
	installedRef?: SkillRef;
	workspaceRootID?: string;
	workspaceRecordID?: string;
	recordRevision?: number;
	name: string;
	displayName?: string;
	description?: string;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	enabled: boolean;
	available: boolean;
	runtimeAllowed: boolean;
	catalogCurrent: boolean;
	shadowed: boolean;
	shadowedBy?: string;
	definitionDigest?: string;
	sourceID?: string;
	locator?: string;
	state?: string;
	diagnostics?: SkillProviderDiagnostic[];
}

export interface SkillSelection {
	skillRef: SkillRef;
	preLoadAsActive: boolean;
	useAsInstructions: boolean;
}

export interface RuntimeSkillFilter {
	types?: string[];
	inserts?: SkillInsert[];
	locationPrefix?: string;
	// Store identity allowlist. Backend resolves to SkillDef internally.
	allowSkillRefs?: SkillRef[];
	sessionID?: string;
	activity?: string;
}

type SkillSessionID = string;

export interface SkillSession {
	sessionID: SkillSessionID;
	activeSkillRefs: SkillRef[];
}

export interface SkillResourceInfo {
	hasResources: boolean;
	totalCount: number;
	locations?: string[];
	moreLocations: boolean;
}

export interface RuntimeSkillListItem {
	skillRef: SkillRef;
	type?: string;
	name?: string;
	displayName?: string;
	description?: string;
	digest?: string;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	sourceTags?: string[];
	resources: SkillResourceInfo;
	rawFrontmatter?: Record<string, any>;
	warnings?: string[];
	isActive: boolean;
	errorMessage?: string;
}

export interface RenderProvidedSkillResponse {
	skill: ProvidedSkill;
	available: boolean;
	text?: string;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	appliedArguments?: Record<string, string>;
	diagnostics?: SkillProviderDiagnostic[];
}

export interface ListSkillsRequest {
	bundleIDs?: string[];
	types?: SkillType[];
	inserts?: SkillInsert[];
	tags?: string[];
	includeDisabled?: boolean;
	includeMissing?: boolean;
	recommendedPageSize?: number;
	pageToken?: string;
}

export interface PutSkillArtifactPayload {
	name?: string;
	isEnabled: boolean;
	displayName?: string;
	description?: string;
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	tags?: string[];
	markdownBody: string;
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
	rawFrontmatter?: Record<string, any>;
	warnings?: string[];
}
// Mirrors Go: spec.SkillType
export enum SkillType {
	FS = 'fs',
	EmbeddedFS = 'embeddedfs',
}

// Mirrors Go: spec.SkillPresenceStatus
export enum SkillPresenceStatus {
	Unknown = 'unknown',
	Present = 'present',
	Missing = 'missing',
	Error = 'error',
}

export interface SkillArgument {
	name: string;
	description?: string;
	default?: string;
}

interface SkillPresence {
	status: SkillPresenceStatus;
	lastCheckedAt?: string;
	lastSeenAt?: string;
	missingSince?: string;
	lastCheckError?: string;
}

// Mirrors Go: spec.Skill (time.Time -> string)
export interface Skill {
	schemaVersion: string;
	id: string;
	slug: string;

	type: SkillType;
	location: string;
	name: string;

	displayName?: string;
	description?: string;
	tags?: string[];
	insert?: SkillInsert;
	arguments?: SkillArgument[];
	resources: SkillResourceInfo;
	rawFrontmatter?: Record<string, any>;
	runtimeWarnings?: string[];
	digest?: string;

	presence?: SkillPresence;

	isEnabled: boolean;
	isBuiltIn: boolean;

	createdAt: Date;
	modifiedAt: Date;
}

// Mirrors Go: spec.SkillBundle (time.Time -> string)
export interface SkillBundle {
	schemaVersion: string;
	id: string;
	slug: string;

	displayName?: string;
	description?: string;

	isEnabled: boolean;
	isBuiltIn: boolean;

	createdAt: Date;
	modifiedAt: Date;

	softDeletedAt?: string;
}

// Mirrors Go: spec.SkillListItem
export interface SkillListItem {
	bundleID: string;
	bundleSlug: string;

	skillSlug: string;
	isBuiltIn: boolean;

	skillDefinition: Skill;
}

export interface InvokeSkillToolResponse {
	outputs?: ToolOutputUnion[];
	meta?: Record<string, any>;
	isBuiltIn: boolean;
	isError?: boolean;
	errorMessage?: string;
}

export interface AssistantSkillOption {
	key: string;
	label: string;
	sel: InstalledSkillSelection;
	skillDefinition: Skill;

	bundleSlug: string;
	bundleDisplayName: string;

	isBuiltIn: boolean;
	isSelectable: boolean;
	isBundleEnabled: boolean;
	isSkillEnabled: boolean;
	availabilityReason?: string;
}
