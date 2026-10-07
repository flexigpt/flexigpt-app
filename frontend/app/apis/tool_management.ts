// oxlint-disable typescript/parameter-properties
import type { ArtifactRef, CapabilityTarget } from '@/spec/artifact';
import type { PluginListItem, PluginView } from '@/spec/plugin';
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
import { ToolImplType, ToolStoreChoiceType, toolStoreListItemFromView } from '@/spec/tool';

import type { JSONRawString } from '@/lib/jsonschema_utils';
import { mapWithConcurrency, throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';
import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';
import { getUUIDv7 } from '@/lib/uuid_utils';

import type { IToolRuntimeAPI, IToolStoreAPI } from '@/apis/interface';

export interface ToolPluginData {
	plugin: PluginListItem;
	tools: ToolStoreListItem[];
	toolsLoaded: boolean;
	isLoadingTools: boolean;
	toolLoadError?: string;
}

type ToolPlugin = PluginListItem | PluginView;

export function toolArtifactRef(tool: ToolView | ToolStoreListItem): ArtifactRef {
	if ('ref' in tool) {
		return tool.ref;
	}

	return {
		rootID: tool.artifact.rootID,
		artifactID: tool.artifact.id,
	};
}

export function toolPluginRef(plugin: ToolPlugin): ArtifactRef {
	if ('ref' in plugin) {
		return plugin.ref;
	}

	return {
		rootID: plugin.artifact.rootID,
		artifactID: plugin.artifact.id,
	};
}

export function toolArtifactKey(ref: ArtifactRef): string {
	return JSON.stringify([ref.rootID, ref.artifactID]);
}

export function toolDisplayName(tool: Pick<ToolView | ToolStoreListItem, 'displayName' | 'name'>): string {
	return tool.displayName || tool.name;
}

export function toolPluginDisplayName(plugin: PluginListItem): string {
	return plugin.displayName || plugin.name;
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
		pluginRef: toolPluginRef(resolved.plugin),
		pluginName: resolved.plugin.name,
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
		pluginRef: item.pluginRef,
		pluginName: item.pluginName,
	};
}

export class ToolManagementAPI {
	constructor(
		private readonly store: IToolStoreAPI,
		private readonly runtime: IToolRuntimeAPI
	) {}

	private readonly composerSelectableToolsCatalog = createSharedAsyncCatalog<ToolListItem[]>(() =>
		this.listSelectableToolsUncached()
	);

	listToolPlugins(): Promise<PluginListItem[]> {
		return this.store.listToolPlugins();
	}

	getToolPlugin(plugin: ArtifactRef): Promise<PluginView> {
		return this.store.getToolPlugin(plugin);
	}

	listPluginTools(plugin: ArtifactRef): Promise<ToolStoreListItem[]> {
		return this.store.listPluginTools(plugin);
	}

	getTool(tool: ArtifactRef): Promise<ToolView> {
		return this.store.getTool(tool);
	}

	async setToolEnabled(tool: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<ToolView> {
		const updated = await this.store.setToolEnabled(tool, expectedRevision, enabled);
		this.invalidateComposerSelectableTools();
		return updated;
	}

	async setToolPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView> {
		const updated = await this.store.setToolPluginEnabled(plugin, expectedRevision, enabled);
		this.invalidateComposerSelectableTools();
		return updated;
	}

	async loadManagementPageData(signal: AbortSignal): Promise<ToolPluginData[]> {
		const plugins = await this.store.listToolPlugins();
		throwIfAborted(signal);

		return plugins
			.map(plugin => ({
				plugin,
				tools: [],
				toolsLoaded: false,
				isLoadingTools: false,
			}))
			.toSorted((left, right) =>
				toolPluginDisplayName(left.plugin).localeCompare(toolPluginDisplayName(right.plugin), undefined, {
					sensitivity: 'base',
				})
			);
	}

	async loadPluginTools(
		plugin: PluginListItem,
		signal: AbortSignal
	): Promise<Pick<ToolPluginData, 'tools' | 'toolsLoaded' | 'toolLoadError'>> {
		try {
			const tools = await this.store.listPluginTools(toolPluginRef(plugin));
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
				toolLoadError: getErrorMessage(error, 'Tools could not be loaded for this Plugin.'),
			};
		}
	}

	/**
	 * Shared static catalog used by all mounted composer tabs.
	 * Runtime invocation and target resolution remain uncached.
	 */
	listComposerSelectableTools(force = false): Promise<ToolListItem[]> {
		return this.composerSelectableToolsCatalog.load(force);
	}

	invalidateComposerSelectableTools(): void {
		this.composerSelectableToolsCatalog.invalidate();
	}

	private async listSelectableToolsUncached(signal?: AbortSignal): Promise<ToolListItem[]> {
		const plugins = await this.store.listToolPlugins();
		if (signal) {
			throwIfAborted(signal);
		}

		const enabledPlugins = plugins.filter(value => value.enabled && value.state === ArtifactState.Available);
		const groups = await mapWithConcurrency(
			enabledPlugins,
			4,
			async plugin => {
				const tools = await this.store.listPluginTools(toolPluginRef(plugin));

				return tools
					.filter(tool => tool.enabled && tool.state === ArtifactState.Available)
					.map(tool => ({ plugin, tool }));
			},
			signal
		);

		const items = await mapWithConcurrency(
			groups.flat(),
			4,
			async ({ plugin, tool: listed }) => {
				const [tool, target] = await Promise.all([
					this.store.getTool(toolArtifactRef(listed)),
					this.store.mapToolTarget(toolArtifactRef(listed)),
				]);

				return {
					target,
					pluginRef: toolPluginRef(plugin),
					pluginName: plugin.name,
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

	mapToolTarget(tool: ArtifactRef): Promise<CapabilityTarget> {
		return this.store.mapToolTarget(tool);
	}

	resolveToolTarget(target: CapabilityTarget): Promise<ResolvedToolView> {
		return this.store.resolveToolTarget(target);
	}

	async getToolTarget(target: CapabilityTarget): Promise<ToolView> {
		return (await this.store.resolveToolTarget(target)).tool;
	}

	async hydrateToolSelection(selection: ToolSelection): Promise<ToolStoreChoice> {
		return toolStoreChoiceFromSelection(selection, await this.store.resolveToolTarget(selection.target));
	}

	hydrateToolSelections(selections: ToolSelection[]): Promise<ToolStoreChoice[]> {
		return mapWithConcurrency(selections, 4, selection => this.hydrateToolSelection(selection));
	}

	invokeGoToolTarget(target: CapabilityTarget, args?: JSONRawString, timeoutMS?: number): Promise<InvokeToolResponse> {
		return this.store.invokeGoToolTarget(target, args, timeoutMS);
	}

	/**
	 * Explicit low-level runtime access. Normal Artifact-backed Tool execution
	 * should use invokeGoToolTarget so Tool and Plugin enablement are checked.
	 */
	invokeGoTool(functionName: string, args?: JSONRawString, timeoutMS?: number): Promise<InvokeToolResponse> {
		return this.runtime.invokeTool(functionName, args, timeoutMS);
	}

	toolListItemFromView(tool: ToolView): ToolStoreListItem {
		return toolStoreListItemFromView(tool);
	}
}
