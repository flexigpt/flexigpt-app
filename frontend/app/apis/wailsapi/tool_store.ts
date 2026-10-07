import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { PluginListItem, PluginView } from '@/spec/plugin';
import type { ResolvedToolView, ToolImplementationView, ToolStoreListItem, ToolView } from '@/spec/tool';
import type { InvokeToolResponse } from '@/spec/toolruntime';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { IToolStoreAPI } from '@/apis/interface';
import {
	capabilityTargetFromWails,
	pluginListItemFromWails,
	pluginViewFromWails,
	toolStoreListItemFromWails,
} from '@/apis/wailsapi/list_item_projection';
import {
	enumFromWails,
	jsonSchemaFromWails,
	optionalWailsString,
	rawJSONToWails,
	requiredObject,
	requireNonBlankString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	GetTool,
	GetToolPlugin,
	InvokeGoToolTarget,
	ListPluginTools,
	ListToolPlugins,
	MapToolTarget,
	ResolveToolTarget,
	SetToolEnabled,
	SetToolPluginEnabled,
} from '@/apis/wailsjs/go/main/ToolStoreWrapper';

/**
 * The only Tool-specific transport adaptation:
 * Go json.RawMessage can arrive as a number[] through Wails.
 */
function toolViewFromWails(value: unknown, operation: string): ToolView {
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

function resolvedToolViewFromWails(value: unknown, operation: string): ResolvedToolView {
	const resolved = requiredObject<ResolvedToolView>(value, operation);

	return {
		tool: toolViewFromWails(resolved.tool, `${operation}.tool`),
		plugin: pluginViewFromWails(resolved.plugin, `${operation}.plugin`),
	};
}

export class WailsToolStoreAPI implements IToolStoreAPI {
	async listToolPlugins(): Promise<PluginListItem[]> {
		return wailsObjectArrayOrEmpty(await ListToolPlugins(), 'ListToolPlugins').map((value, index) =>
			pluginListItemFromWails(value, `ListToolPlugins[${index}]`)
		);
	}

	async getToolPlugin(plugin: ArtifactRef): Promise<PluginView> {
		return pluginViewFromWails(await GetToolPlugin(plugin as Parameters<typeof GetToolPlugin>[0]), 'GetToolPlugin');
	}

	async listPluginTools(plugin: ArtifactRef): Promise<ToolStoreListItem[]> {
		return wailsObjectArrayOrEmpty(
			await ListPluginTools(plugin as Parameters<typeof ListPluginTools>[0]),
			'ListPluginTools'
		).map((tool, index) => toolStoreListItemFromWails(tool, `ListPluginTools[${index}]`));
	}

	async getTool(tool: ArtifactRef): Promise<ToolView> {
		return toolViewFromWails(await GetTool(tool as Parameters<typeof GetTool>[0]), 'GetTool');
	}

	async setToolEnabled(tool: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<ToolView> {
		return toolViewFromWails(
			await SetToolEnabled(tool as Parameters<typeof SetToolEnabled>[0], expectedRevision, enabled),
			'SetToolEnabled'
		);
	}

	async setToolPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView> {
		return pluginViewFromWails(
			await SetToolPluginEnabled(plugin as Parameters<typeof SetToolPluginEnabled>[0], expectedRevision, enabled),
			'SetToolPluginEnabled'
		);
	}

	async mapToolTarget(tool: ArtifactRef): Promise<CapabilityTarget> {
		return capabilityTargetFromWails(await MapToolTarget(tool as Parameters<typeof MapToolTarget>[0]), 'MapToolTarget');
	}

	async resolveToolTarget(target: CapabilityTarget): Promise<ResolvedToolView> {
		return resolvedToolViewFromWails(
			await ResolveToolTarget(target as Parameters<typeof ResolveToolTarget>[0]),
			'ResolveToolTarget'
		);
	}

	async invokeGoToolTarget(
		target: CapabilityTarget,
		args?: JSONRawString,
		timeoutMS?: number
	): Promise<InvokeToolResponse> {
		return requiredObject<InvokeToolResponse>(
			await InvokeGoToolTarget(
				target as Parameters<typeof InvokeGoToolTarget>[0],
				args === undefined ? '' : rawJSONToWails(args, 'Tool target arguments'),
				timeoutMS ?? 0
			),
			'InvokeGoToolTarget'
		);
	}
}
