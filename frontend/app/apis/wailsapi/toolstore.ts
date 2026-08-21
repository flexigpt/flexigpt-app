import type { HTTPToolImpl, Tool, ToolBundle, ToolListItem } from '@/spec/tool';
import { HTTPBodyOutputMode, ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import type { JSONSchema } from '@/lib/jsonschema_utils';

import type { IToolStoreAPI } from '@/apis/interface';
import {
	enumFromWails,
	jsonObjectFromWails,
	jsonObjectToWails,
	optionalWailsBody,
	optionalWailsString,
	requireWailsBody,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	DeleteTool,
	DeleteToolBundle,
	GetTool,
	ListToolBundles,
	ListTools,
	PatchTool,
	PatchToolBundle,
	PutTool,
	PutToolBundle,
} from '@/apis/wailsjs/go/main/ToolStoreWrapper';
import type { spec } from '@/apis/wailsjs/go/models';

function httpImplFromWails(httpImpl: HTTPToolImpl | null | undefined, field: string): HTTPToolImpl | undefined {
	if (httpImpl === null || httpImpl === undefined) {
		return undefined;
	}

	const normalized = requireWailsBody(httpImpl, field);
	const response = requireWailsBody(normalized.response, `${field}.response`);

	return {
		...normalized,
		response: {
			...response,
			bodyOutputMode: response.bodyOutputMode
				? enumFromWails(response.bodyOutputMode, HTTPBodyOutputMode, `${field}.response.bodyOutputMode`)
				: undefined,
		},
	};
}

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
		httpImpl: httpImplFromWails(tool.httpImpl, `${field}.httpImpl`),
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

	async putToolBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				slug: slug,
				displayName: displayName,
				isEnabled: isEnabled,
				description: description,
			} as spec.PutToolBundleRequestBody,
		};
		await PutToolBundle(req as spec.PutToolBundleRequest);
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

	async deleteToolBundle(bundleID: string): Promise<void> {
		const req: spec.DeleteToolBundleRequest = {
			BundleID: bundleID,
		};
		await DeleteToolBundle(req);
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

	async putTool(
		bundleID: string,
		toolSlug: string,
		version: string,
		displayName: string,
		isEnabled: boolean,
		userCallable: boolean,
		llmCallable: boolean,
		autoExecReco: boolean,
		argSchema: JSONSchema,
		type: ToolImplType,
		httpImpl?: HTTPToolImpl,
		description?: string,
		tags?: string[]
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			ToolSlug: toolSlug,
			Version: version,
			Body: {
				displayName: displayName,
				isEnabled: isEnabled,
				description: description,
				tags: tags,
				userCallable: userCallable,
				llmCallable: llmCallable,
				autoExecReco: autoExecReco,
				argSchema: jsonObjectToWails(argSchema, 'tool argSchema'),
				type: type,
				httpImpl: httpImpl,
			} as spec.PutToolRequestBody,
		};
		await PutTool(req as spec.PutToolRequest);
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

	async deleteTool(bundleID: string, toolSlug: string, version: string): Promise<void> {
		const req: spec.DeleteToolRequest = {
			BundleID: bundleID,
			ToolSlug: toolSlug,
			Version: version,
		};
		await DeleteTool(req);
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
}
