import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { PluginListItem, PluginView } from '@/spec/plugin';
import type { ResolvedToolView, ToolImplementationView, ToolStoreListItem, ToolView } from '@/spec/tool';
import { CapabilityTargetForm, CapabilityTargetProvenance } from '@/spec/artifact';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import type { IToolStoreAPI } from '@/apis/interface';
import type { CapabilityTarget as WailsCapabilityTarget } from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/models';
import {
	capabilityTargetFromWails,
	pluginListItemFromWails,
	pluginViewFromWails,
	toolStoreListItemFromWails,
} from '@/apis/wailsapi/list_item_projection';
import {
	enumFromWails,
	enumToWails,
	jsonSchemaFromWails,
	nonBlankStringToWails,
	optionalWailsString,
	requiredObject,
	requireNonBlankString,
	wailsObjectArrayOrEmpty,
} from '@/apis/wailsapi/transport';
import {
	GetTool,
	GetToolPlugin,
	ListPluginTools,
	ListToolPlugins,
	MapToolTarget,
	ResolveToolTarget,
	SetToolEnabled,
	SetToolPluginEnabled,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/toolstorewrapper';

function toolViewFromWails(value: unknown, operation: string): ToolView {
	const tool = requiredObject<ToolView>(value, operation);
	const rawImplementation = requiredObject<Record<string, unknown>>(tool.implementation, `${operation}.implementation`);
	const kind = enumFromWails(rawImplementation.kind, ToolImplType, `${operation}.implementation.kind`);

	const implementation: ToolImplementationView =
		kind === ToolImplType.SDK
			? {
					kind,
					sdkType: requireNonBlankString(rawImplementation.sdkType, `${operation}.implementation.sdkType`),
					sdkToolType: enumFromWails(
						rawImplementation.sdkToolType,
						ToolStoreChoiceType,
						`${operation}.implementation.sdkToolType`
					),
				}
			: {
					kind,
					function: optionalWailsString(rawImplementation.function, `${operation}.implementation.function`),
				};

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

function capabilityTargetToWails(target: CapabilityTarget): WailsCapabilityTarget {
	const operation = 'ResolveToolTarget.target';
	const form = enumToWails<CapabilityTargetForm, WailsCapabilityTarget['form']>(
		target.form,
		CapabilityTargetForm,
		`${operation}.form`
	);
	const provenance = enumToWails<CapabilityTargetProvenance, WailsCapabilityTarget['provenance']>(
		target.provenance,
		CapabilityTargetProvenance,
		`${operation}.provenance`
	);
	const base = {
		form,
		type: nonBlankStringToWails<WailsCapabilityTarget['type']>(target.type, `${operation}.type`),
		name: requireNonBlankString(target.name, `${operation}.name`),
		provenance,
	};

	if (target.form === CapabilityTargetForm.Artifact) {
		if (!target.artifact) {
			throw new Error('Artifact capability targets require an Artifact reference.');
		}

		return {
			...base,
			artifact: {
				rootID: target.artifact.rootID,
				artifactID: target.artifact.artifactID,
			},
		};
	}

	return {
		...base,
		providerIdentity: requireNonBlankString(target.providerIdentity, `${operation}.providerIdentity`),
		providerLocalID: requireNonBlankString(target.providerLocalID, `${operation}.providerLocalID`),
		...(target.evidence === undefined ? {} : { evidence: target.evidence }),
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
		return resolvedToolViewFromWails(await ResolveToolTarget(capabilityTargetToWails(target)), 'ResolveToolTarget');
	}
}
