import type { ArtifactRef } from '@/spec/artifact';
import type { CollectionListItem, CollectionView } from '@/spec/collection';
import type { ToolImplementationView, ToolStoreListItem, ToolView } from '@/spec/tool';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import type { IToolStoreAPI } from '@/apis/interface';
import {
	collectionListItemFromWails,
	collectionViewFromWails,
	toolStoreListItemFromWails,
} from '@/apis/wailsapi/list_item_projection';
import {
	enumFromWails,
	jsonSchemaFromWails,
	optionalWailsString,
	requiredObject,
	requireNonBlankString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	GetTool,
	GetToolCollection,
	ListCollectionTools,
	ListToolCollections,
	SetToolCollectionEnabled,
	SetToolEnabled,
} from '@/apis/wailsjs/go/main/ToolStoreWrapper';

/**
 * The only Tool-specific transport adaptation:
 * Go json.RawMessage can arrive as a number[] through Wails.
 */
export function toolViewFromWails(value: unknown, operation: string): ToolView {
	const tool = requiredObject<ToolView>(value, operation);
	const rawImplementation = requiredObject<Record<string, unknown>>(tool.implementation, `${operation}.implementation`);
	const kind = enumFromWails(rawImplementation.kind, ToolImplType, `${operation}.implementation.kind`);
	let implementation: ToolImplementationView;

	if (kind === ToolImplType.SDK) {
		implementation = {
			kind,
			sdkType: requireNonBlankString(rawImplementation.sdkType, `${operation}.implementation.sdkType`),
			sdkToolType: enumFromWails(
				rawImplementation.sdkToolType,
				ToolStoreChoiceType,
				`${operation}.implementation.sdkToolType`
			),
		};
	} else {
		implementation = {
			kind,
			function: optionalWailsString(rawImplementation.function, `${operation}.implementation.function`),
		};
	}

	return {
		...tool,
		implementation,
		inputSchema: jsonSchemaFromWails(tool.inputSchema, `${operation}.inputSchema`),
		userArgSchema:
			tool.userArgSchema === null || tool.userArgSchema === undefined
				? undefined
				: jsonSchemaFromWails(tool.userArgSchema, `${operation}.userArgSchema`),
		outputSchema:
			tool.outputSchema === null || tool.outputSchema === undefined
				? undefined
				: jsonSchemaFromWails(tool.outputSchema, `${operation}.outputSchema`),
	};
}

export class WailsToolStoreAPI implements IToolStoreAPI {
	async listToolCollections(): Promise<CollectionListItem[]> {
		return wailsObjectArrayOrEmpty(await ListToolCollections(), 'ListToolCollections').map((value, index) =>
			collectionListItemFromWails(value, `ListToolCollections[${index}]`)
		);
	}

	async getToolCollection(collection: ArtifactRef): Promise<CollectionView> {
		return collectionViewFromWails(await GetToolCollection(collection), 'GetToolCollection');
	}

	async listCollectionTools(collection: ArtifactRef): Promise<ToolStoreListItem[]> {
		return wailsObjectArrayOrEmpty(await ListCollectionTools(collection), 'ListCollectionTools').map((tool, index) =>
			toolStoreListItemFromWails(tool, `ListCollectionTools[${index}]`)
		);
	}

	async getTool(tool: ArtifactRef): Promise<ToolView> {
		return toolViewFromWails(await GetTool(tool), 'GetTool');
	}

	async setToolEnabled(tool: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<ToolView> {
		return toolViewFromWails(await SetToolEnabled(tool, expectedRevision, enabled), 'SetToolEnabled');
	}

	async setToolCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView> {
		return collectionViewFromWails(
			await SetToolCollectionEnabled(collection, expectedRevision, enabled),
			'SetToolCollectionEnabled'
		);
	}
}
