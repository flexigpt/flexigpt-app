import type { MappedTarget } from '@/spec/artifact';
import type { Tool, ToolBundle, ToolListItem, ToolRef } from '@/spec/tool';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import type { IToolStoreAPI } from '@/apis/interface';
import type { spec } from '@/apis/wailsjs/go/models';
import {
	enumFromWails,
	jsonObjectFromWails,
	optionalWailsBody,
	optionalWailsString,
	requiredObject,
	requireWailsBody,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	GetTool,
	ListToolBundles,
	ListTools,
	PatchTool,
	PatchToolBundle,
	ResolveMappedToolTarget,
} from '@/apis/wailsjs/go/main/ToolStoreWrapper';

function toolFromWails(toolValue: Tool, field: string): Tool {
	const tool = requireWailsBody(toolValue, field);
	return {
		...tool,
		argSchema: jsonObjectFromWails(tool.argSchema, `${field}.argSchema`),
		userArgSchema:
			tool.userArgSchema === null || tool.userArgSchema === undefined
				? undefined
				: jsonObjectFromWails(tool.userArgSchema, `${field}.userArgSchema`),
		llmToolType: enumFromWails(tool.llmToolType, ToolStoreChoiceType, `${field}.llmToolType`),
		type: enumFromWails(tool.type, ToolImplType, `${field}.type`),
	};
}

export class WailsToolStoreAPI implements IToolStoreAPI {
	async listToolBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ toolBundles: ToolBundle[]; nextPageToken?: string }> {
		const req = {
			BundleIDs: bundleIDs,
			IncludeDisabled: includeDisabled,
			PageSize: pageSize,
			PageToken: pageToken,
		};
		const resp = await ListToolBundles(req as spec.ListToolBundlesRequest);
		const body = requireWailsBody(resp.Body, 'ListToolBundles');
		return {
			toolBundles: wailsObjectArrayOrEmpty<ToolBundle>(body.toolBundles, 'ListToolBundles.toolBundles'),
			nextPageToken: optionalWailsString(body.nextPageToken, 'ListToolBundles.nextPageToken') || undefined,
		};
	}

	async patchToolBundle(bundleID: string, isEnabled: boolean): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				isEnabled: isEnabled,
			},
		};
		await PatchToolBundle(req as spec.PatchToolBundleRequest);
	}

	async listTools(
		bundleIDs?: string[],
		tags?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ toolListItems: ToolListItem[]; nextPageToken?: string }> {
		const req = {
			BundleIDs: bundleIDs,
			Tags: tags,
			IncludeDisabled: includeDisabled,
			RecommendedPageSize: pageSize,
			PageToken: pageToken,
		};
		const resp = await ListTools(req as spec.ListToolsRequest);
		const body = requireWailsBody(resp.Body, 'ListTools');
		const items = wailsObjectArrayOrEmpty<ToolListItem>(body.toolListItems, 'ListTools.toolListItems');

		return {
			toolListItems: items.map((item, index) => {
				return Object.assign(item, {
					toolDefinition: toolFromWails(
						item.toolDefinition as Tool,
						`ListTools.toolListItems[${index}].toolDefinition`
					),
				});
			}),
			nextPageToken: optionalWailsString(body.nextPageToken, 'ListTools.nextPageToken') || undefined,
		};
	}

	async patchTool(bundleID: string, toolSlug: string, version: string, isEnabled: boolean): Promise<void> {
		const req = {
			BundleID: bundleID,
			ToolSlug: toolSlug,
			Version: version,
			Body: {
				isEnabled: isEnabled,
			},
		};
		await PatchTool(req as spec.PatchToolRequest);
	}

	async getTool(bundleID: string, toolSlug: string, version: string): Promise<Tool | undefined> {
		const req: spec.GetToolRequest = {
			BundleID: bundleID,
			ToolSlug: toolSlug,
			Version: version,
		};
		const resp = await GetTool(req);
		const body = optionalWailsBody(resp.Body, 'GetTool');
		return body === undefined ? undefined : toolFromWails(body as Tool, 'GetTool');
	}

	async resolveMappedToolTarget(target: MappedTarget): Promise<ToolRef> {
		/*
		 * The generated declaration currently incorrectly shares the model
		 * resolver response body. The Tool Store endpoint must return toolRef.
		 * Do not derive a ToolRef from MappedTarget.identifier.
		 */
		const response = (await ResolveMappedToolTarget({
			target,
		} as Parameters<typeof ResolveMappedToolTarget>[0])) as unknown as {
			Body?: {
				toolRef?: ToolRef;
			};
		};

		const body = requiredObject<{ toolRef?: ToolRef }>(response.Body, 'ResolveMappedToolTarget');
		return requiredObject<ToolRef>(body.toolRef, 'ResolveMappedToolTarget.toolRef');
	}
}
