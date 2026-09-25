import type { ArtifactRef } from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';
import type { ToolView } from '@/spec/tool';

import type { IToolStoreAPI } from '@/apis/interface';
import { jsonSchemaFromWails, requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
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

	return {
		...tool,
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
	async listToolCollections(): Promise<CollectionView[]> {
		return wailsObjectArrayOrEmpty<CollectionView>(await ListToolCollections(), 'ListToolCollections');
	}

	async getToolCollection(collection: ArtifactRef): Promise<CollectionView> {
		return requiredObject<CollectionView>(await GetToolCollection(collection), 'GetToolCollection');
	}

	async listCollectionTools(collection: ArtifactRef): Promise<ToolView[]> {
		return wailsObjectArrayOrEmpty<ToolView>(await ListCollectionTools(collection), 'ListCollectionTools').map(
			(tool, index) => toolViewFromWails(tool, `ListCollectionTools[${index}]`)
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
		return requiredObject<CollectionView>(
			await SetToolCollectionEnabled(collection, expectedRevision, enabled),
			'SetToolCollectionEnabled'
		);
	}
}
