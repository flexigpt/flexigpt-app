import { useCallback } from 'react';
import { FiFileText } from 'react-icons/fi';

import type { WorkspaceDefaultPolicyView } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';

function WorkspaceDefaultPolicyDisclosureContent() {
	const loadPolicy = useCallback(async (signal: AbortSignal): Promise<WorkspaceDefaultPolicyView> => {
		const policy = await workspaceManagementAPI.getWorkspaceDefaultPolicy();
		throwIfAborted(signal);
		return policy;
	}, []);

	const {
		data: policy,
		error,
		isLoading,
		isRefreshing,
		reloadOrThrow,
	} = useAsyncResource(loadPolicy, {
		initialData: null as WorkspaceDefaultPolicyView | null,
	});

	return (
		<div className="border-base-content/10 max-h-96 overflow-y-auto border-t p-3">
			{error ? (
				<ManagementResourceError
					title="Automatic Workspace setup could not be loaded"
					error={error}
					isRetrying={isRefreshing}
					onRetry={reloadOrThrow}
				/>
			) : null}

			{isLoading && !policy ? <Loader text="Loading automatic Workspace setup..." /> : null}

			{policy ? (
				<pre className="bg-base-200 max-h-72 overflow-auto rounded-xl p-3 text-xs whitespace-pre-wrap">
					{policy.yaml}
				</pre>
			) : null}
		</div>
	);
}

export function WorkspaceDefaultPolicyDisclosure() {
	return (
		<details className="border-base-content/10 bg-base-200/50 w-full rounded-xl border">
			<summary className="btn btn-ghost btn-sm w-full list-none justify-start rounded-xl">
				<FiFileText size={14} />
				View automatic setup
			</summary>
			<WorkspaceDefaultPolicyDisclosureContent />
		</details>
	);
}
