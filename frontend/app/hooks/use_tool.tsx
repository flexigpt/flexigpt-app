import { useCallback } from 'react';

import type { ToolListItem } from '@/spec/tool';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { toolManagementAPI } from '@/apis/baseapi';

export function useTools() {
	const load = useCallback((signal: AbortSignal) => toolManagementAPI.listSelectableTools(signal), []);
	const resource = useAsyncResource(load, { initialData: [] as ToolListItem[] });

	return {
		data: resource.data,
		error: resource.error,
		loading: resource.isLoading,
		isRefreshing: resource.isRefreshing,
		hasResolved: resource.hasResolved,
		ready: resource.hasResolved && !resource.isLoading && !resource.isRefreshing && !resource.error,
		refresh: resource.reloadOrThrow,
	};
}

export type ToolCatalogState = ReturnType<typeof useTools>;
