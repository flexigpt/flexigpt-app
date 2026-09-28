import { useCallback } from 'react';
import { FiRefreshCw } from 'react-icons/fi';

import type { CollectionListItem } from '@/spec/collection';
import type { ToolStoreListItem } from '@/spec/tool';

import { useAsyncResource } from '@/hooks/use_async_resource';

import type { ToolCollectionData } from '@/apis/tool_management';
import { toolManagementAPI } from '@/apis/baseapi';
import { toolArtifactKey, toolArtifactRef, toolCollectionRef } from '@/apis/tool_management';

import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { PageFrame } from '@/components/page_frame';

import { ToolCollectionCard } from '@/tools/tool_collection_card';

// oxlint-disable-next-line no-restricted-exports
export default function ToolsPage() {
	const loadPageData = useCallback((signal: AbortSignal) => toolManagementAPI.loadManagementPageData(signal), []);
	const {
		data: collections,
		error: pageLoadError,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setCollections,
	} = useAsyncResource(loadPageData, { initialData: [] as ToolCollectionData[] });

	const refreshCollection = useCallback(
		async (collection: CollectionListItem) => {
			const result = await toolManagementAPI.loadCollectionTools(collection, new AbortController().signal);
			const key = toolArtifactKey(toolCollectionRef(collection));

			setCollections(previous =>
				(previous ?? []).map(item => {
					if (toolArtifactKey(toolCollectionRef(item.collection)) !== key) {
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
		[setCollections]
	);

	const loadCollectionTools = useCallback(
		async (collection: CollectionListItem) => {
			const key = toolArtifactKey(toolCollectionRef(collection));

			setCollections(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolCollectionRef(item.collection)) === key
						? {
								...item,
								isLoadingTools: true,
								toolLoadError: undefined,
							}
						: item
				)
			);

			const result = await toolManagementAPI.loadCollectionTools(collection, new AbortController().signal);

			setCollections(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolCollectionRef(item.collection)) === key
						? {
								...item,
								...result,
								isLoadingTools: false,
							}
						: item
				)
			);
		},
		[setCollections]
	);

	const toggleCollection = useCallback(
		async (collection: CollectionListItem, enabled: boolean) => {
			const ref = toolCollectionRef(collection);
			const updated = await toolManagementAPI.setToolCollectionEnabled(ref, collection.revision, enabled);
			const key = toolArtifactKey(ref);

			setCollections(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolCollectionRef(item.collection)) === key
						? {
								...item,
								collection: toolManagementAPI.collectionListItemFromView(updated),
							}
						: item
				)
			);
		},
		[setCollections]
	);

	const toggleTool = useCallback(
		async (collection: CollectionListItem, tool: ToolStoreListItem, enabled: boolean) => {
			const updated = await toolManagementAPI.setToolEnabled(toolArtifactRef(tool), tool.revision, enabled);
			const collectionKey = toolArtifactKey(toolCollectionRef(collection));
			const toolKey = toolArtifactKey(toolArtifactRef(tool));

			setCollections(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolCollectionRef(item.collection)) === collectionKey
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
		[setCollections]
	);

	if (isLoading && !hasResolved) {
		return <Loader text="Loading built-in tools..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Built-in Tools"
					description="Inspect and enable built-in Go tools and provider API tools, organized into Collections."
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

					{collections.length === 0 && !pageLoadError ? (
						<p className="mt-8 text-center text-sm">No built-in tools are available.</p>
					) : null}

					{collections.map(item => (
						<ToolCollectionCard
							key={toolArtifactKey(toolCollectionRef(item.collection))}
							collection={item.collection}
							tools={item.tools}
							toolsLoaded={item.toolsLoaded}
							isLoadingTools={item.isLoadingTools}
							toolLoadError={item.toolLoadError}
							onLoadTools={() => loadCollectionTools(item.collection)}
							onRefreshTools={() => refreshCollection(item.collection)}
							onToggleCollectionEnable={toggleCollection}
							onToggleToolEnable={toggleTool}
						/>
					))}
				</ManagementPageContent>
			</div>
		</PageFrame>
	);
}
