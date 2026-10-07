import { useCallback } from 'react';
import { FiRefreshCw } from 'react-icons/fi';

import type { PluginListItem } from '@/spec/plugin';
import type { ToolStoreListItem } from '@/spec/tool';
import { pluginListItemFromPluginView } from '@/spec/plugin';

import { useAsyncResource } from '@/hooks/use_async_resource';

import type { ToolPluginData } from '@/apis/tool_management';
import { toolManagementAPI } from '@/apis/baseapi';
import { toolArtifactKey, toolArtifactRef, toolPluginRef } from '@/apis/tool_management';

import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { ToolPluginCard } from '@/tools/tool_plugin_card';

// oxlint-disable-next-line no-restricted-exports
export default function ToolsPage() {
	const loadPageData = useCallback((signal: AbortSignal) => toolManagementAPI.loadManagementPageData(signal), []);
	const {
		data: plugins,
		error: pageLoadError,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setPlugins,
	} = useAsyncResource(loadPageData, { initialData: [] as ToolPluginData[] });

	const refreshPlugin = useCallback(
		async (plugin: PluginListItem) => {
			const result = await toolManagementAPI.loadPluginTools(plugin, new AbortController().signal);
			const key = toolArtifactKey(toolPluginRef(plugin));

			setPlugins(previous =>
				(previous ?? []).map(item => {
					if (toolArtifactKey(toolPluginRef(item.plugin)) !== key) {
						return item;
					}

					return {
						...item,
						...result,
						isLoadingTools: false,
					};
				})
			);
		},
		[setPlugins]
	);

	const loadPluginTools = useCallback(
		async (plugin: PluginListItem) => {
			const key = toolArtifactKey(toolPluginRef(plugin));

			setPlugins(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolPluginRef(item.plugin)) === key
						? {
								...item,
								isLoadingTools: true,
								toolLoadError: undefined,
							}
						: item
				)
			);

			const result = await toolManagementAPI.loadPluginTools(plugin, new AbortController().signal);

			setPlugins(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolPluginRef(item.plugin)) === key
						? {
								...item,
								...result,
								isLoadingTools: false,
							}
						: item
				)
			);
		},
		[setPlugins]
	);

	const togglePlugin = useCallback(
		async (plugin: PluginListItem, enabled: boolean) => {
			const ref = toolPluginRef(plugin);
			const updated = await toolManagementAPI.setToolPluginEnabled(ref, plugin.revision, enabled);
			const key = toolArtifactKey(ref);

			setPlugins(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolPluginRef(item.plugin)) === key
						? {
								...item,
								plugin: pluginListItemFromPluginView(updated, plugin.builtIn),
							}
						: item
				)
			);
		},
		[setPlugins]
	);

	const toggleTool = useCallback(
		async (plugin: PluginListItem, tool: ToolStoreListItem, enabled: boolean) => {
			const updated = await toolManagementAPI.setToolEnabled(toolArtifactRef(tool), tool.revision, enabled);
			const pluginKey = toolArtifactKey(toolPluginRef(plugin));
			const toolKey = toolArtifactKey(toolArtifactRef(tool));

			setPlugins(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolPluginRef(item.plugin)) === pluginKey
						? {
								...item,
								tools: item.tools.map(candidate =>
									toolArtifactKey(toolArtifactRef(candidate)) === toolKey &&
									candidate.revision <= updated.artifact.revision
										? toolManagementAPI.toolListItemFromView(updated)
										: candidate
								),
							}
						: item
				)
			);
		},
		[setPlugins]
	);

	if (isLoading && !hasResolved) {
		return <Loader text="Loading built-in tools..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Built-in Tools"
					description="Inspect and enable built-in Go tools and provider API tools, organized into Plugins."
					actions={
						<button
							type="button"
							className="btn btn-ghost rounded-xl"
							disabled={isRefreshing}
							onClick={() => {
								void reloadOrThrow().catch((error: unknown) => {
									console.error('Failed to refresh built-in tools:', error);
								});
							}}
						>
							<FiRefreshCw className={isRefreshing ? 'animate-spin' : undefined} size={18} />
							<span>{isRefreshing ? 'Refreshing' : 'Refresh'}</span>
						</button>
					}
				/>

				<ManagementPageContent>
					{pageLoadError ? (
						<ManagementResourceError
							title="Built-in tools could not be loaded"
							error={pageLoadError}
							isRetrying={isRefreshing}
							onRetry={reloadOrThrow}
						/>
					) : null}

					{plugins.length === 0 && !pageLoadError ? (
						<p className="mt-8 text-center text-sm">No built-in tools are available.</p>
					) : null}

					{plugins.map(item => (
						<ToolPluginCard
							key={toolArtifactKey(toolPluginRef(item.plugin))}
							plugin={item.plugin}
							tools={item.tools}
							toolsLoaded={item.toolsLoaded}
							isLoadingTools={item.isLoadingTools}
							toolLoadError={item.toolLoadError}
							onLoadTools={() => loadPluginTools(item.plugin)}
							onRefreshTools={() => refreshPlugin(item.plugin)}
							onTogglePluginEnable={togglePlugin}
							onToggleToolEnable={toggleTool}
						/>
					))}
				</ManagementPageContent>
			</div>
		</PageFrame>
	);
}
