import type {
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceID,
	ArtifactState,
	CapabilityOccurrence,
	StoreArtifact,
} from '@/spec/artifact';

interface DeclarationLocator {
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

interface CollectionMemberReference {
	type: string;
	name: string;
	insert?: string;
	locator?: DeclarationLocator;
	scope?: string;
	server?: string;
}

interface CollectionMemberView {
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

/**
 * Exact frontend projection of generated `collection.ListItem`.
 *
 * List endpoints intentionally return this small shape. Call a collection
 * read endpoint only when a workflow actually needs members or Artifact
 * metadata that is absent here.
 */
export interface CollectionListItem {
	ref: ArtifactRef;
	sourceID: ArtifactSourceID;
	name: string;
	displayName: string;
	description?: string;
	state: ArtifactState;
	enabled: boolean;
	revision: number;
	memberCount: number;
	builtIn: boolean;
	editable: boolean;
	deletable: boolean;
	baseline: boolean;
}

export function collectionListItemFromCollectionView(collection: CollectionView): CollectionListItem {
	return {
		ref: {
			rootID: collection.artifact.rootID,
			artifactID: collection.artifact.id,
		},
		sourceID: collection.artifact.binding.sourceID,
		name: collection.name,
		displayName: collection.displayName,
		description: collection.description,
		state: collection.artifact.state,
		enabled: collection.artifact.enabled,
		revision: collection.artifact.revision,
		memberCount: collection.members.length,
		builtIn: !collection.baseline && !collection.editable && !collection.deletable,
		editable: collection.editable,
		deletable: collection.deletable,
		baseline: collection.baseline,
	};
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
