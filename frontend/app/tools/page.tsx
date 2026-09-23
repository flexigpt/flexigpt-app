import { useCallback } from 'react';
import { FiRefreshCw } from 'react-icons/fi';

import type { Tool, ToolBundle } from '@/spec/tool';

import { mapWithConcurrency, throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { toolStoreAPI } from '@/apis/baseapi';
import { getAllToolBundles, getAllTools } from '@/apis/list_helper';

import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { ToolBundleCard } from '@/tools/tool_bundle_card';

interface BundleData {
	bundle: ToolBundle;
	tools: Tool[];
	toolLoadError?: string;
}

async function loadBuiltInToolBundleData(signal: AbortSignal): Promise<BundleData[]> {
	const toolBundles = await getAllToolBundles(undefined, true);
	throwIfAborted(signal);

	if (toolBundles.some(bundle => !bundle.isBuiltIn)) {
		throw new Error('The tool catalogue returned a non-built-in bundle.');
	}

	return mapWithConcurrency(
		toolBundles,
		4,
		async bundle => {
			try {
				const toolListItems = await getAllTools([bundle.id], undefined, true);
				throwIfAborted(signal);

				const tools = toolListItems.map(item => item.toolDefinition);
				if (tools.some(tool => !tool.isBuiltIn)) {
					throw new Error('The tool catalogue returned a non-built-in tool.');
				}

				return {
					bundle,
					tools,
				};
			} catch (error) {
				throwIfAborted(signal);
				return {
					bundle,
					tools: [],
					toolLoadError: getErrorMessage(error, 'Failed to load built-in tools for this bundle.'),
				};
			}
		},
		signal
	);
}

// oxlint-disable-next-line no-restricted-exports
export default function ToolsPage() {
	const loadPageData = useCallback((signal: AbortSignal) => loadBuiltInToolBundleData(signal), []);
	const {
		data: bundles,
		error: pageLoadError,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setBundles,
	} = useAsyncResource(loadPageData, { initialData: [] as BundleData[] });

	const refreshBundleTools = useCallback(
		async (bundleID: string) => {
			const toolListItems = await getAllTools([bundleID], undefined, true);
			const freshTools = toolListItems.map(item => item.toolDefinition);

			if (freshTools.some(tool => !tool.isBuiltIn)) {
				throw new Error('The tool catalogue returned a non-built-in tool.');
			}

			setBundles(previous =>
				(previous ?? []).map(bundleData =>
					bundleData.bundle.id === bundleID
						? Object.assign(bundleData, {
								tools: freshTools,
								toolLoadError: undefined,
							})
						: bundleData
				)
			);
		},
		[setBundles]
	);

	const handleToggleBundleEnable = useCallback(
		async (bundleID: string, enabled: boolean) => {
			const bundleData = (bundles ?? []).find(item => item.bundle.id === bundleID);
			if (!bundleData?.bundle.isBuiltIn) {
				throw new Error('Only built-in tool bundles can be changed.');
			}

			await toolStoreAPI.patchToolBundle(bundleID, enabled);

			setBundles(previous =>
				(previous ?? []).map(item =>
					item.bundle.id === bundleID
						? Object.assign(item, {
								bundle: {
									...item.bundle,
									isEnabled: enabled,
								},
							})
						: item
				)
			);
		},
		[bundles, setBundles]
	);

	const handleToggleToolEnable = useCallback(
		async (bundleID: string, tool: Tool, enabled: boolean) => {
			const bundleData = (bundles ?? []).find(item => item.bundle.id === bundleID);
			if (!bundleData?.bundle.isBuiltIn || !tool.isBuiltIn) {
				throw new Error('Only built-in tools can be changed.');
			}
			if (!bundleData.tools.some(candidate => candidate.id === tool.id)) {
				throw new Error('Tool not found.');
			}

			await toolStoreAPI.patchTool(bundleID, tool.slug, tool.version, enabled);

			setBundles(previous =>
				(previous ?? []).map(item =>
					item.bundle.id === bundleID
						? Object.assign(item, {
								tools: item.tools.map(candidate =>
									candidate.id === tool.id
										? {
												...candidate,
												isEnabled: enabled,
											}
										: candidate
								),
							})
						: item
				)
			);
		},
		[bundles, setBundles]
	);

	const refreshPage = () => {
		void reloadOrThrow().catch((error: unknown) => {
			console.error('Failed to refresh built-in tools:', error);
		});
	};

	if (isLoading && !hasResolved) {
		return <Loader text="Loading built-in tools..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Built-in Tools"
					description="Inspect and enable built-in Go tools and provider API tools. Custom and HTTP tool authoring are not supported."
					actions={
						<button type="button" className="btn btn-ghost rounded-xl" onClick={refreshPage} disabled={isRefreshing}>
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

					{bundles.length === 0 ? <p className="mt-8 text-center text-sm">No built-in tools are available.</p> : null}

					{bundles.map(bundleData => (
						<ToolBundleCard
							key={bundleData.bundle.id}
							bundle={bundleData.bundle}
							tools={bundleData.tools}
							toolLoadError={bundleData.toolLoadError}
							onRefreshTools={() => refreshBundleTools(bundleData.bundle.id)}
							onToggleBundleEnable={handleToggleBundleEnable}
							onToggleToolEnable={handleToggleToolEnable}
						/>
					))}
				</ManagementPageContent>
			</div>
		</PageFrame>
	);
}
