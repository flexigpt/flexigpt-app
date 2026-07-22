import type { ToolOutputUnion } from '@/spec/tool';

export const SKILLS_AUTOEXEC_TOOL_CHOICES = new Set([
	'builtin.skills-load',
	'builtin.skills-unload',
	'builtin.skills-readresource',
]);

export type SkillInsert = 'instructions' | 'user-message';

// Store identity for selection/persistence (NOT runtime identity).
export interface SkillRef {
	bundleID: string;
	skillSlug: string;
	skillID: string;
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
	sel: SkillSelection;
	skillDefinition: Skill;

	bundleSlug: string;
	bundleDisplayName: string;

	isBuiltIn: boolean;
	isSelectable: boolean;
	isBundleEnabled: boolean;
	isSkillEnabled: boolean;
	availabilityReason?: string;
}
