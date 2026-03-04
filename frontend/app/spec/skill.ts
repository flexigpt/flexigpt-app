import type { ToolOutputUnion } from '@/spec/tool';

// This is the runtime identity.
export interface SkillDef {
	type: string;
	name: string;
	location: string;
}

// Store identity for selection/persistence (NOT runtime identity).
export interface SkillRef {
	bundleID: string;
	skillSlug: string;
	skillID: string;
}

export interface RuntimeSkillFilter {
	types?: string[];
	namePrefix?: string;
	locationPrefix?: string;
	// Store identity allowlist. Backend resolves to SkillDef internally.
	allowSkillRefs?: SkillRef[];
	sessionID?: string;
	activity?: string;
}

export type SkillSessionID = string;

export interface SkillSession {
	sessionID: SkillSessionID;
	activeSkills: SkillDef[];
}

export interface SkillRecord {
	def: SkillDef;
	description: string;
	properties?: Record<string, any>;
	digest?: string;
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

/**
 * @public
 *
 * Mirrors Go: spec.SkillPresence (time.Time -> string)
 */
export interface SkillPresence {
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
