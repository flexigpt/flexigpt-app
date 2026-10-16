import type {
	ArtifactDigest,
	ArtifactRef,
	ArtifactRootID,
	ArtifactState,
	StoreArtifact,
	StoreArtifactAddress,
	StoreArtifactSourceSummary,
	StoreManagedPackageFile,
} from '@/spec/artifact';
import type { PluginView } from '@/spec/plugin';

export type SkillRef = ArtifactRef;

/** @public */
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

export interface SkillArgument {
	name: string;
	description?: string;
	default?: string;
}

/**
 * Exact frontend projection of generated `consumerapi.SkillListItem`.
 *
 * Management listing paths must use this lightweight object. A full
 * `StoreArtifact` is fetched only by direct edit, delete, and source flows.
 */
export interface StoreSkillListItem {
	ref: ArtifactRef;
	name: string;
	displayName: string;
	description?: string;
	state: ArtifactState;
	enabled: boolean;
	revision: number;
	definitionDigest?: ArtifactDigest;
	builtIn: boolean;
	managed: boolean;
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

enum RuntimeSkillActivity {
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

/**
 * Runtime-native skill identity. Provider type remains a string because
 * Agent Skills providers are registry-extensible.
 */
interface RuntimeSkillDefinition {
	type: string;
	name: string;
	location: string;
}

export interface ManagedSkillCreateRequest {
	plugin: ArtifactRef;
	expectedPluginRevision: number;
	skillName: string;
	skillMD?: number[];
	files?: StoreManagedPackageFile[];
	enabled: boolean;
}

export interface ManagedSkillCreateResult {
	artifact: StoreArtifact;
	address: StoreArtifactAddress;
	plugin: PluginView;
	membershipCreated: boolean;
}

export interface ManagedSkillReplaceRequest {
	plugin: ArtifactRef;
	expectedPluginRevision: number;
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
	plugin: PluginView;
}

export interface StoreManagedSkillDocument {
	artifact: StoreArtifact;
	document: SkillDocumentInput;
}

export interface ArtifactSkillDetails {
	artifact: StoreArtifact;
	document: SkillDocumentInput;
	location: string;
	resources: SkillResourceInfo;
	runtimeWarnings?: string[];
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
	Revision: number;
	DefinitionDigest: ArtifactDigest;
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
