import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import type {
	StoreArtifact,
	StoreArtifactAddress,
	StoreArtifactSourceSummary,
	StoreManagedPackageFile,
} from '@/spec/artifact_store';
import type { ArtifactMembershipView, CollectionCapabilityPlan, CollectionView } from '@/spec/collection';
import type { CapabilityPlan } from '@/spec/resolution';
import type {
	InvokeSkillToolResponse,
	RuntimeSkillActivity,
	RuntimeSkillDefinition,
	RuntimeSkillListItem,
	RuntimeSkillRecord,
	RuntimeSkillRenderResult,
	RuntimeSkillSession,
	RuntimeSkillSessionOptions,
	SkillDocumentInput,
	SkillInsert,
} from '@/spec/skill';

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

export type {
	InvokeSkillToolResponse,
	RuntimeSkillDefinition,
	RuntimeSkillListItem,
	RuntimeSkillRecord,
	RuntimeSkillRenderResult,
	RuntimeSkillSession,
	RuntimeSkillSessionOptions,
};
