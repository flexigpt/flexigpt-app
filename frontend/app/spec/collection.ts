import type {
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	CapabilityOccurrence,
	StoreArtifact,
} from '@/spec/artifact';

export interface DeclarationLocator {
	kind?: string;
	path?: string;
	url?: string;
	integrity?: string;
	repository?: string;
	revision?: string;
	manager?: string;
	package?: string;
	version?: string;
	registry?: string;
	command?: string;
}

export interface CollectionMemberReference {
	type: string;
	name?: string;
	insert?: string;
	locator?: DeclarationLocator;
	scope?: string;
	server?: string;
}

export interface CollectionMemberView {
	type: string;
	name?: string;
	insert?: string;
	locator?: DeclarationLocator;
	server?: string;
	contained: boolean;
	selector: boolean;
}

export interface CollectionView {
	artifact: StoreArtifact;
	name: string;
	displayName: string;
	description?: string;
	members: CollectionMemberReference[];
	entries: CollectionMemberView[];
	editable: boolean;
	deletable: boolean;
	baseline: boolean;
}

export interface ArtifactMembershipView {
	collection: ArtifactRef;
	collectionName: string;
	collectionRevision: number;
	memberIndex: number;
	member: CollectionMemberReference;
	status: string;
	resolvedArtifact?: ArtifactRef;
	resolvedToArtifact: boolean;
	code?: string;
	message?: string;
}

export interface CollectionCapabilityPlan {
	collection: CollectionView;
	occurrences: CapabilityOccurrence[];
	complete: boolean;
}

export interface MemberMutationResult {
	collection: CollectionView;
	index: number;
	created: boolean;
}

export interface CreateCollectionRequest {
	rootID: ArtifactRootID;
	sourceID?: ArtifactSourceID;
	name: string;
	displayName?: string;
	description?: string;
}

export interface UpdateCollectionRequest {
	collection: ArtifactRef;
	expectedRevision: number;
	description?: string;
	displayName?: string;
}

export interface DeleteCollectionRequest {
	collection: ArtifactRef;
	expectedRevision: number;
}

export interface AddEntryRequest {
	collection: ArtifactRef;
	expectedRevision: number;
	entry: unknown;
}

export interface AddMemberRequest {
	collection: ArtifactRef;
	expectedRevision: number;
	member: CollectionMemberReference;
}

export interface AddArtifactMemberRequest {
	collection: ArtifactRef;
	expectedRevision: number;
	artifact: ArtifactRef;
}

export interface RemoveMemberRequest {
	collection: ArtifactRef;
	expectedRevision: number;
	index: number;
}
