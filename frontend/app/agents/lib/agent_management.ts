import type { AgentImportDestination, AgentView } from '@/spec/agent';
import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';

import { mapWithConcurrency, throwIfAborted } from '@/lib/async_utils';

import { agentStoreAPI } from '@/apis/baseapi';

const COLLECTION_LOAD_CONCURRENCY = 4;

export interface AgentCollectionData {
	collection: CollectionView;
	agents: AgentView[];
	importDestination?: AgentImportDestination;
	agentLoadError?: string;
}

export interface AgentManagementPageData {
	collections: AgentCollectionData[];
	importDestinations: AgentImportDestination[];
}

export const EMPTY_AGENT_MANAGEMENT_PAGE_DATA: AgentManagementPageData = {
	collections: [],
	importDestinations: [],
};

export function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}

	return fallback;
}

export function agentArtifactRef(agent: AgentView): ArtifactRef {
	return {
		rootID: agent.artifact.rootID,
		artifactID: agent.artifact.id,
	};
}

export function agentCollectionRef(collection: CollectionView): ArtifactRef {
	return {
		rootID: collection.artifact.rootID,
		artifactID: collection.artifact.id,
	};
}

export function agentCollectionKey(collection: CollectionView): string {
	const ref = agentCollectionRef(collection);
	return `${ref.rootID}:${ref.artifactID}`;
}

export function formatArtifactRef(ref?: ArtifactRef): string {
	if (!ref) {
		return '—';
	}

	return `${ref.rootID}/${ref.artifactID}`;
}

export function formatDateish(value: string | Date | undefined | null): string {
	if (!value) {
		return '—';
	}

	return value instanceof Date ? value.toISOString() : value;
}

export function agentDisplayName(agent: AgentView): string {
	return agent.displayName || agent.name;
}

export function collectionDisplayName(collection: CollectionView): string {
	return collection.displayName || collection.name;
}

export function isBuiltInAgentCollection(collection: CollectionView): boolean {
	return !collection.editable && !collection.deletable;
}

export function canEditAgentCollectionMetadata(collection: CollectionView): boolean {
	return collection.editable && !collection.baseline;
}

export function canDeleteAgentCollection(collection: CollectionView): boolean {
	return collection.deletable && !collection.baseline;
}

export function textToBase64(value: string): string {
	const bytes = new TextEncoder().encode(value);
	let binary = '';

	for (let offset = 0; offset < bytes.length; offset += 0x8000) {
		binary += String.fromCodePoint(...bytes.subarray(offset, offset + 0x8000));
	}

	return globalThis.btoa(binary);
}

function compareCollections(left: CollectionView, right: CollectionView): number {
	const leftBuiltIn = isBuiltInAgentCollection(left);
	const rightBuiltIn = isBuiltInAgentCollection(right);

	if (leftBuiltIn !== rightBuiltIn) {
		return leftBuiltIn ? -1 : 1;
	}

	const byName = collectionDisplayName(left).localeCompare(collectionDisplayName(right), undefined, {
		sensitivity: 'base',
	});

	if (byName !== 0) {
		return byName;
	}

	return agentCollectionKey(left).localeCompare(agentCollectionKey(right));
}

export async function loadAgentManagementPageData(signal: AbortSignal): Promise<AgentManagementPageData> {
	const [allAgents, importDestinations] = await Promise.all([
		agentStoreAPI.listAgentsForManagement(),
		agentStoreAPI.listAgentImportDestinations(),
	]);
	throwIfAborted(signal);

	const rootIDs = new Set<ArtifactRootID>();

	for (const agent of allAgents) {
		rootIDs.add(agent.artifact.rootID);
	}

	for (const destination of importDestinations) {
		rootIDs.add(destination.rootID);
		rootIDs.add(destination.collection.artifact.rootID);
	}

	const collectionLists = await mapWithConcurrency(
		[...rootIDs],
		COLLECTION_LOAD_CONCURRENCY,
		rootID => agentStoreAPI.listAgentCollections(rootID),
		signal
	);
	throwIfAborted(signal);

	const collectionsByKey = new Map<string, CollectionView>();

	for (const collection of collectionLists.flat()) {
		collectionsByKey.set(agentCollectionKey(collection), collection);
	}

	for (const destination of importDestinations) {
		const key = agentCollectionKey(destination.collection);

		if (!collectionsByKey.has(key)) {
			collectionsByKey.set(key, destination.collection);
		}
	}

	const destinationByCollectionKey = new Map(
		importDestinations.map(destination => [agentCollectionKey(destination.collection), destination] as const)
	);

	const collections = await mapWithConcurrency(
		[...collectionsByKey.values()],
		COLLECTION_LOAD_CONCURRENCY,
		async collection => {
			try {
				const agents = await agentStoreAPI.listAgents({
					rootID: collection.artifact.rootID,
					collection: agentCollectionRef(collection),
				});
				throwIfAborted(signal);

				return {
					collection,
					agents,
					importDestination: destinationByCollectionKey.get(agentCollectionKey(collection)),
				};
			} catch (error) {
				throwIfAborted(signal);

				return {
					collection,
					agents: [],
					importDestination: destinationByCollectionKey.get(agentCollectionKey(collection)),
					agentLoadError: getErrorMessage(error, 'Agents could not be loaded for this Collection.'),
				};
			}
		},
		signal
	);

	return {
		collections: collections.toSorted((left, right) => compareCollections(left.collection, right.collection)),
		importDestinations: [...importDestinations].toSorted((left, right) => {
			const rootCompare = (left.rootDisplayName || left.rootID).localeCompare(
				right.rootDisplayName || right.rootID,
				undefined,
				{ sensitivity: 'base' }
			);

			if (rootCompare !== 0) {
				return rootCompare;
			}

			return collectionDisplayName(left.collection).localeCompare(collectionDisplayName(right.collection), undefined, {
				sensitivity: 'base',
			});
		}),
	};
}
