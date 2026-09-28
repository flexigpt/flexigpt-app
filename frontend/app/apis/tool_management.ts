// oxlint-disable typescript/parameter-properties
import type { ArtifactRef, MappedTarget } from '@/spec/artifact';
import type { CollectionListItem, CollectionView } from '@/spec/collection';
import type { ToolChoice } from '@/spec/inference';
import type {
	ResolvedToolView,
	ToolListItem,
	ToolSelection,
	ToolStoreChoice,
	ToolStoreListItem,
	ToolView,
} from '@/spec/tool';
import type { InvokeToolResponse } from '@/spec/toolruntime';
import { ArtifactState } from '@/spec/artifact';
import { collectionListItemFromCollectionView } from '@/spec/collection';
import { ToolImplType, ToolStoreChoiceType, toolStoreListItemFromView } from '@/spec/tool';

import type { JSONRawString } from '@/lib/jsonschema_utils';
import { mapWithConcurrency, throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';
import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IToolAggregateAPI, IToolRuntimeAPI, IToolStoreAPI } from '@/apis/interface';

export interface ToolCollectionData {
	collection: CollectionListItem;
	tools: ToolStoreListItem[];
	toolsLoaded: boolean;
	isLoadingTools: boolean;
	toolLoadError?: string;
}

type ToolCollection = CollectionListItem | CollectionView;

export function toolArtifactRef(tool: ToolView | ToolStoreListItem): ArtifactRef {
	if ('ref' in tool) {
		return tool.ref;
	}

	return {
		rootID: tool.artifact.rootID,
		artifactID: tool.artifact.id,
	};
}

export function toolCollectionRef(collection: ToolCollection): ArtifactRef {
	if ('ref' in collection) {
		return collection.ref;
	}

	return {
		rootID: collection.artifact.rootID,
		artifactID: collection.artifact.id,
	};
}

export function toolArtifactKey(ref: ArtifactRef): string {
	return JSON.stringify([ref.rootID, ref.artifactID]);
}

export function toolDisplayName(tool: Pick<ToolView | ToolStoreListItem, 'displayName' | 'name'>): string {
	return tool.displayName || tool.name;
}

export function toolCollectionDisplayName(collection: CollectionListItem): string {
	return collection.displayName || collection.name;
}

function toolChoiceType(tool: ToolView): ToolStoreChoiceType {
	return tool.implementation.kind === ToolImplType.Go
		? ToolStoreChoiceType.Function
		: (tool.implementation.sdkToolType ?? ToolStoreChoiceType.Function);
}

export function toolStoreChoiceFromSelection(selection: ToolSelection, resolved: ResolvedToolView): ToolStoreChoice {
	const tool = resolved.tool;
	return {
		choiceID: selection.choiceID,
		target: selection.target,
		autoExecute: selection.autoExecute,
		userArgSchemaInstance: selection.userArgSchemaInstance,
		toolType: toolChoiceType(tool),
		implementationKind: tool.implementation.kind,
		sdkType: tool.implementation.kind === ToolImplType.SDK ? tool.implementation.sdkType : undefined,
		displayName: toolDisplayName(tool),
		description: tool.description,
		toolVersion: tool.version,
		collectionRef: toolCollectionRef(resolved.collection),
		collectionName: resolved.collection.name,
	};
}

export function toolChoiceFromListItem(
	item: ToolListItem,
	autoExecute = item.toolDefinition.autoExecute
): ToolStoreChoice {
	const tool = item.toolDefinition;
	return {
		choiceID: getUUIDv7(),
		target: item.target,
		autoExecute: tool.implementation.kind === ToolImplType.Go && autoExecute,
		toolType: toolChoiceType(tool),
		implementationKind: tool.implementation.kind,
		sdkType: tool.implementation.kind === ToolImplType.SDK ? tool.implementation.sdkType : undefined,
		displayName: toolDisplayName(tool),
		description: tool.description,
		toolVersion: tool.version,
		collectionRef: item.collectionRef,
		collectionName: item.collectionName,
	};
}

export class ToolManagementAPI {
	constructor(
		private readonly store: IToolStoreAPI,
		private readonly aggregate: IToolAggregateAPI,
		private readonly runtime: IToolRuntimeAPI
	) {}

	private readonly composerSelectableToolsCatalog = createSharedAsyncCatalog<ToolListItem[]>(() =>
		this.listSelectableToolsUncached()
	);

	listToolCollections(): Promise<CollectionListItem[]> {
		return this.store.listToolCollections();
	}

	getToolCollection(collection: ArtifactRef): Promise<CollectionView> {
		return this.store.getToolCollection(collection);
	}

	listCollectionTools(collection: ArtifactRef): Promise<ToolStoreListItem[]> {
		return this.store.listCollectionTools(collection);
	}

	getTool(tool: ArtifactRef): Promise<ToolView> {
		return this.store.getTool(tool);
	}

	async setToolEnabled(tool: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<ToolView> {
		const updated = await this.store.setToolEnabled(tool, expectedRevision, enabled);
		this.invalidateComposerSelectableTools();
		return updated;
	}

	async setToolCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView> {
		const updated = await this.store.setToolCollectionEnabled(collection, expectedRevision, enabled);
		this.invalidateComposerSelectableTools();
		return updated;
	}

	async loadManagementPageData(signal: AbortSignal): Promise<ToolCollectionData[]> {
		const collections = await this.store.listToolCollections();
		throwIfAborted(signal);

		return collections
			.map(collection => ({
				collection,
				tools: [],
				toolsLoaded: false,
				isLoadingTools: false,
			}))
			.toSorted((left, right) =>
				toolCollectionDisplayName(left.collection).localeCompare(
					toolCollectionDisplayName(right.collection),
					undefined,
					{
						sensitivity: 'base',
					}
				)
			);
	}

	async loadCollectionTools(
		collection: CollectionListItem,
		signal: AbortSignal
	): Promise<Pick<ToolCollectionData, 'tools' | 'toolsLoaded' | 'toolLoadError'>> {
		try {
			const tools = await this.store.listCollectionTools(toolCollectionRef(collection));
			throwIfAborted(signal);

			return {
				tools,
				toolsLoaded: true,
			};
		} catch (error) {
			throwIfAborted(signal);

			return {
				tools: [],
				toolsLoaded: false,
				toolLoadError: getErrorMessage(error, 'Tools could not be loaded for this Collection.'),
			};
		}
	}

	/**
	 * Shared static catalog used by all mounted composer tabs.
	 * Runtime invocation and mapped-target resolution remain uncached.
	 */
	listComposerSelectableTools(force = false): Promise<ToolListItem[]> {
		return this.composerSelectableToolsCatalog.load(force);
	}

	invalidateComposerSelectableTools(): void {
		this.composerSelectableToolsCatalog.invalidate();
	}

	private async listSelectableToolsUncached(signal?: AbortSignal): Promise<ToolListItem[]> {
		const collections = await this.store.listToolCollections();
		if (signal) {
			throwIfAborted(signal);
		}

		const enabledCollections = collections.filter(value => value.enabled && value.state === ArtifactState.Available);
		const groups = await mapWithConcurrency(
			enabledCollections,
			4,
			async collection => {
				const tools = await this.store.listCollectionTools(toolCollectionRef(collection));
				return tools
					.filter(tool => tool.enabled && tool.state === ArtifactState.Available)
					.map(tool => ({ collection, tool }));
			},
			signal
		);

		const items = await mapWithConcurrency(
			groups.flat(),
			4,
			async ({ collection, tool: listed }) => {
				const [tool, target] = await Promise.all([
					this.store.getTool(toolArtifactRef(listed)),
					this.aggregate.mapToolTarget(toolArtifactRef(listed)),
				]);

				return {
					target,
					collectionRef: toolCollectionRef(collection),
					collectionName: collection.name,
					toolDefinition: tool,
				};
			},
			signal
		);

		if (signal) {
			throwIfAborted(signal);
		}
		return items;
	}

	listSelectableTools(signal?: AbortSignal): Promise<ToolListItem[]> {
		return this.listSelectableToolsUncached(signal);
	}

	mapToolTarget(tool: ArtifactRef): Promise<MappedTarget> {
		return this.aggregate.mapToolTarget(tool);
	}

	resolveMappedTool(target: MappedTarget): Promise<ResolvedToolView> {
		return this.aggregate.resolveMappedTool(target);
	}

	async getMappedTool(target: MappedTarget): Promise<ToolView> {
		return (await this.aggregate.resolveMappedTool(target)).tool;
	}

	async hydrateToolSelection(selection: ToolSelection): Promise<ToolStoreChoice> {
		return toolStoreChoiceFromSelection(selection, await this.aggregate.resolveMappedTool(selection.target));
	}

	hydrateToolSelections(selections: ToolSelection[]): Promise<ToolStoreChoice[]> {
		return mapWithConcurrency(selections, 4, selection => this.hydrateToolSelection(selection));
	}

	hydrateInferenceToolChoice(selection: ToolSelection): Promise<ToolChoice> {
		return this.aggregate.hydrateInferenceToolChoice(selection);
	}

	invokeMappedTool(target: MappedTarget, args?: JSONRawString, timeoutMS?: number): Promise<InvokeToolResponse> {
		return this.aggregate.invokeMappedTool(target, args, timeoutMS);
	}

	/**
	 * Explicit low-level runtime access. Normal composer execution must use
	 * invokeMappedTool so the backend verifies identity and enablement.
	 */
	invokeGoTool(functionName: string, args?: JSONRawString, timeoutMS?: number): Promise<InvokeToolResponse> {
		return this.runtime.invokeTool(functionName, args, timeoutMS);
	}

	collectionListItemFromView(collection: CollectionView): CollectionListItem {
		return collectionListItemFromCollectionView(collection);
	}

	toolListItemFromView(tool: ToolView): ToolStoreListItem {
		return toolStoreListItemFromView(tool);
	}
}
