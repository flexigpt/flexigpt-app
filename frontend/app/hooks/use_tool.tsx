import { useCallback } from 'react';

import type { ToolListItem } from '@/spec/tool';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { toolManagementAPI } from '@/apis/baseapi';

export function useTools() {
	const load = useCallback(async (_signal: AbortSignal): Promise<ToolListItem[]> => {
		return toolManagementAPI.listComposerSelectableTools();
	}, []);

	const resource = useAsyncResource(load, {
		initialData: [] as ToolListItem[],
	});

	const refresh = useCallback(() => {
		toolManagementAPI.invalidateComposerSelectableTools();
		return resource.reloadOrThrow();
	}, [resource]);

	return {
		data: resource.data,
		error: resource.error,
		loading: resource.isLoading,
		isRefreshing: resource.isRefreshing,
		hasResolved: resource.hasResolved,
		ready: (resource.hasResolved && !resource.isLoading && !resource.isRefreshing && !resource.error) satisfies boolean,
		refresh,
	};
}

export type ToolCatalogState = ReturnType<typeof useTools>;
