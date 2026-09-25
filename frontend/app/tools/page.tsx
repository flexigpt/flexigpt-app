import { useCallback } from 'react';
import { FiRefreshCw } from 'react-icons/fi';

import type { CollectionView } from '@/spec/collection';
import type { ToolView } from '@/spec/tool';

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
		async (collection: CollectionView) => {
			const ref = toolCollectionRef(collection);
			const [freshCollection, freshTools] = await Promise.all([
				toolManagementAPI.getToolCollection(ref),
				toolManagementAPI.listCollectionTools(ref),
			]);
			const key = toolArtifactKey(ref);

			setCollections(previous =>
				(previous ?? []).map(item => {
					if (toolArtifactKey(toolCollectionRef(item.collection)) !== key) {
						return item;
					}
					const oldTools = new Map(item.tools.map(tool => [toolArtifactKey(toolArtifactRef(tool)), tool]));
					return {
						collection:
							item.collection.artifact.revision > freshCollection.artifact.revision ? item.collection : freshCollection,
						tools: freshTools.map(tool => {
							const old = oldTools.get(toolArtifactKey(toolArtifactRef(tool)));
							return old && old.artifact.revision > tool.artifact.revision ? old : tool;
						}),
					};
				})
			);
		},
		[setCollections]
	);

	const toggleCollection = useCallback(
		async (collection: CollectionView, enabled: boolean) => {
			const ref = toolCollectionRef(collection);
			const updated = await toolManagementAPI.setToolCollectionEnabled(ref, collection.artifact.revision, enabled);
			const key = toolArtifactKey(ref);

			setCollections(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolCollectionRef(item.collection)) === key
						? {
								...item,
								collection: item.collection.artifact.revision > updated.artifact.revision ? item.collection : updated,
							}
						: item
				)
			);
		},
		[setCollections]
	);

	const toggleTool = useCallback(
		async (collection: CollectionView, tool: ToolView, enabled: boolean) => {
			const updated = await toolManagementAPI.setToolEnabled(toolArtifactRef(tool), tool.artifact.revision, enabled);
			const collectionKey = toolArtifactKey(toolCollectionRef(collection));
			const toolKey = toolArtifactKey(toolArtifactRef(tool));

			setCollections(previous =>
				(previous ?? []).map(item =>
					toolArtifactKey(toolCollectionRef(item.collection)) === collectionKey
						? {
								...item,
								tools: item.tools.map(candidate =>
									toolArtifactKey(toolArtifactRef(candidate)) === toolKey &&
									candidate.artifact.revision <= updated.artifact.revision
										? updated
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
							toolLoadError={item.toolLoadError}
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
