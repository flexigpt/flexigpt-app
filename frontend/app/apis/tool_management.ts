// oxlint-disable typescript/parameter-properties
import type { ArtifactRef, MappedTarget } from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';
import type { ToolChoice } from '@/spec/inference';
import type { ResolvedToolView, ToolListItem, ToolSelection, ToolStoreChoice, ToolView } from '@/spec/tool';
import type { InvokeToolResponse } from '@/spec/toolruntime';
import { ArtifactState } from '@/spec/artifact';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import type { JSONRawString } from '@/lib/jsonschema_utils';
import { mapWithConcurrency, throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IToolAggregateAPI, IToolRuntimeAPI, IToolStoreAPI } from '@/apis/interface';

export interface ToolCollectionData {
	collection: CollectionView;
	tools: ToolView[];
	toolLoadError?: string;
}

export function toolArtifactRef(tool: ToolView): ArtifactRef {
	return {
		rootID: tool.artifact.rootID,
		artifactID: tool.artifact.id,
	};
}

export function toolCollectionRef(collection: CollectionView): ArtifactRef {
	return {
		rootID: collection.artifact.rootID,
		artifactID: collection.artifact.id,
	};
}

export function toolArtifactKey(ref: ArtifactRef): string {
	return JSON.stringify([ref.rootID, ref.artifactID]);
}

export function toolDisplayName(tool: ToolView): string {
	return tool.displayName || tool.name;
}

export function toolCollectionDisplayName(collection: CollectionView): string {
	return collection.displayName || collection.name;
}

function toolChoiceType(tool: ToolView): ToolStoreChoiceType {
	return tool.implementation.kind === ToolImplType.Go ? ToolStoreChoiceType.Function : tool.implementation.sdkToolType;
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

	listToolCollections(): Promise<CollectionView[]> {
		return this.store.listToolCollections();
	}

	getToolCollection(collection: ArtifactRef): Promise<CollectionView> {
		return this.store.getToolCollection(collection);
	}

	listCollectionTools(collection: ArtifactRef): Promise<ToolView[]> {
		return this.store.listCollectionTools(collection);
	}

	getTool(tool: ArtifactRef): Promise<ToolView> {
		return this.store.getTool(tool);
	}

	setToolEnabled(tool: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<ToolView> {
		return this.store.setToolEnabled(tool, expectedRevision, enabled);
	}

	setToolCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView> {
		return this.store.setToolCollectionEnabled(collection, expectedRevision, enabled);
	}

	async loadManagementPageData(signal: AbortSignal): Promise<ToolCollectionData[]> {
		const collections = await this.store.listToolCollections();
		throwIfAborted(signal);

		return mapWithConcurrency(
			collections.toSorted((left, right) =>
				toolCollectionDisplayName(left).localeCompare(toolCollectionDisplayName(right), undefined, {
					sensitivity: 'base',
				})
			),
			4,
			async collection => {
				try {
					const tools = await this.store.listCollectionTools(toolCollectionRef(collection));
					throwIfAborted(signal);
					return { collection, tools };
				} catch (error) {
					throwIfAborted(signal);
					return {
						collection,
						tools: [],
						toolLoadError: getErrorMessage(error, 'Tools could not be loaded for this Collection.'),
					};
				}
			},
			signal
		);
	}

	async listSelectableTools(signal?: AbortSignal): Promise<ToolListItem[]> {
		const collections = await this.store.listToolCollections();
		if (signal) {
			throwIfAborted(signal);
		}

		const enabledCollections = collections.filter(
			value => value.artifact.enabled && value.artifact.state === ArtifactState.Available
		);
		const groups = await mapWithConcurrency(
			enabledCollections,
			4,
			async collection => {
				const tools = await this.store.listCollectionTools(toolCollectionRef(collection));
				return tools.filter(tool => tool.artifact.enabled && tool.artifact.state === ArtifactState.Available);
			},
			signal
		);

		const items = await mapWithConcurrency(
			groups.flat(),
			4,
			async tool => {
				const target = await this.aggregate.mapToolTarget(toolArtifactRef(tool));
				const resolved = await this.aggregate.resolveMappedTool(target);
				return {
					target,
					collectionRef: toolCollectionRef(resolved.collection),
					collectionName: resolved.collection.name,
					toolDefinition: resolved.tool,
				};
			},
			signal
		);

		if (signal) {
			throwIfAborted(signal);
		}
		return items;
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
}
