import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import type { StoreArtifact, StoreArtifactAddress } from '@/spec/artifact_store';
import type {
	ArtifactMembershipView,
	CollectionCapabilityPlan,
	CollectionView,
	DeclarationLocator,
	MemberMutationResult,
} from '@/spec/collection';
import type { CapabilityPlan } from '@/spec/resolution';

export interface AgentDocument {
	type: string;
	name: string;
	displayName?: string;
	description?: string;
	labels?: Record<string, string>;
	locator?: DeclarationLocator;
	metadata?: Record<string, number[]>;
	members?: unknown[];
	loop?: unknown;
	workflow?: unknown;
}

export interface AgentView {
	artifact: StoreArtifact;
	name: string;
	displayName: string;
	description?: string;
	builtIn: boolean;
	managed: boolean;
}

export interface ListAgentsRequest {
	rootID: ArtifactRootID;
	logicalNames?: string[];
	collection?: ArtifactRef;
	includeBuiltin?: boolean;
	enabled?: boolean;
}

export interface AttachAgentToCollectionRequest {
	collection: ArtifactRef;
	expectedRevision: number;
	agent: ArtifactRef;
}

export interface ManagedAgentCreateRequest {
	collection: ArtifactRef;
	expectedCollectionRevision: number;
	document: AgentDocument;
	enabled: boolean;
}

export interface ManagedAgentCreateResult {
	agent: StoreArtifact;
	address: StoreArtifactAddress;
	collection: CollectionView;
	membershipCreated: boolean;
}

export interface ManagedAgentDeleteRequest {
	agent: ArtifactRef;
	expectedRevision: number;
}

export interface ManagedAgentReplaceRequest {
	agent: ArtifactRef;
	expectedRevision: number;
	document: AgentDocument;
}

export interface ManagedAgentReplaceResult {
	agent: StoreArtifact;
	address: StoreArtifactAddress;
}

export interface AgentManagementView {
	agent: AgentView;
	directMemberships: ArtifactMembershipView[];
	capabilities: CapabilityPlan;
}

export interface AgentCollectionManagementView {
	collection: CollectionView;
	capabilities: CollectionCapabilityPlan;
	mutation?: MemberMutationResult;
}
