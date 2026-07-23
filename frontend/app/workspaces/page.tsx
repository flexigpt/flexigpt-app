import { useCallback, useMemo, useRef, useState } from 'react';

import { FiFolderPlus, FiSearch, FiX } from 'react-icons/fi';

import type { UpdateWorkspacePayload, WorkspaceView } from '@/spec/workspace';
import { WorkspaceMode } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { Loader } from '@/components/loader';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { ModalConfirmDialog } from '@/components/modal/modal_confirm_dialog';
import { PageFrame } from '@/components/page_frame';

import { getErrorMessage, sortWorkspaces, workspaceMatchesSearch } from '@/workspaces/lib/workspace_utils';
import { WorkspaceCard } from '@/workspaces/workspace_card';
import type { WorkspaceUpsertSubmission } from '@/workspaces/workspace_upsert_modal';
import { WorkspaceUpsertModal } from '@/workspaces/workspace_upsert_modal';

async function loadWorkspaces(signal: AbortSignal): Promise<WorkspaceView[]> {
	const workspaces = await workspaceAPI.listWorkspaces();
	throwIfAborted(signal);
	return sortWorkspaces(workspaces);
}

// oxlint-disable-next-line no-restricted-exports
export default function WorkspacesPage() {
	const loadPageData = useCallback((signal: AbortSignal) => loadWorkspaces(signal), []);
	const {
		data: workspaces,
		error: pageLoadError,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setWorkspaces,
	} = useAsyncResource(loadPageData, { initialData: [] as WorkspaceView[] });

	const [searchQuery, setSearchQuery] = useState('');
	const [isCreateOpen, setIsCreateOpen] = useState(false);
	const [workspaceToDelete, setWorkspaceToDelete] = useState<WorkspaceView | null>(null);
	const [alertMessage, setAlertMessage] = useState('');
	const mountedRef = useRef(true);

	const existingDisplayNames = useMemo(() => workspaces.map(workspace => workspace.displayName), [workspaces]);

	const visibleWorkspaces = useMemo(
		() => workspaces.filter(workspace => workspaceMatchesSearch(workspace, searchQuery)),
		[searchQuery, workspaces]
	);

	const enabledCount = workspaces.filter(workspace => workspace.enabled).length;
	const filesystemCount = workspaces.filter(workspace => workspace.mode === WorkspaceMode.Filesystem).length;
	const sourceCount = workspaces.reduce((total, workspace) => total + workspace.attachments.length, 0);

	const replaceWorkspace = useCallback(
		(nextWorkspace: WorkspaceView) => {
			setWorkspaces(previous =>
				sortWorkspaces(
					previous.some(workspace => workspace.rootID === nextWorkspace.rootID)
						? previous.map(workspace => (workspace.rootID === nextWorkspace.rootID ? nextWorkspace : workspace))
						: [...previous, nextWorkspace]
				)
			);
		},
		[setWorkspaces]
	);

	const createWorkspace = async (submission: WorkspaceUpsertSubmission) => {
		if (submission.kind === 'update') {
			throw new Error('Expected a new workspace payload.');
		}

		const created =
			submission.kind === 'filesystem'
				? await workspaceAPI.createFilesystemWorkspace(submission.payload)
				: await workspaceAPI.createEmptyWorkspace(submission.payload);

		replaceWorkspace(created);

		try {
			await workspaceAPI.refreshWorkspace(created.rootID);
			const refreshed = await workspaceAPI.getWorkspace(created.rootID);
			replaceWorkspace(refreshed);
		} catch (error) {
			setAlertMessage(
				`Workspace was created, but initial discovery failed. Open the workspace and retry Refresh. ${getErrorMessage(
					error,
					''
				)}`.trim()
			);
		}
	};

	const updateWorkspace = useCallback(
		async (workspace: WorkspaceView, payload: UpdateWorkspacePayload): Promise<WorkspaceView> => {
			const updated = await workspaceAPI.updateWorkspace(workspace.rootID, payload);
			replaceWorkspace(updated);
			return updated;
		},
		[replaceWorkspace]
	);

	const deleteWorkspace = async () => {
		if (!workspaceToDelete) {
			return;
		}

		await workspaceAPI.deleteWorkspace(workspaceToDelete.rootID, workspaceToDelete.revision);

		if (mountedRef.current) {
			setWorkspaces(previous => previous.filter(workspace => workspace.rootID !== workspaceToDelete.rootID));
		}
	};

	if (isLoading && !hasResolved) {
		return <Loader text="Loading workspaces..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Workspaces"
					description="Manage project roots, discovery paths, context, skills, sources, and runtime permissions."
					width="wide"
					actions={
						<button
							type="button"
							className="btn btn-ghost rounded-xl"
							onClick={() => {
								setIsCreateOpen(true);
							}}
						>
							<FiFolderPlus size={18} />
							<span>Add Workspace</span>
						</button>
					}
				/>

				<ManagementPageContent width="wide">
					{pageLoadError ? (
						<ManagementResourceError
							title="Workspaces could not be loaded"
							error={pageLoadError}
							isRetrying={isRefreshing}
							onRetry={reloadOrThrow}
						/>
					) : null}

					<div className="border-base-content/10 bg-base-100 rounded-2xl border p-4 text-sm">
						<div className="font-semibold">How workspace discovery works</div>
						<ul className="text-base-content/70 mt-2 list-disc space-y-1 pl-5 text-xs">
							<li>Add a project root path when creating a filesystem workspace.</li>
							<li>Default context files, README files, and skill folders are discovered by the backend.</li>
							<li>Add project-specific files or folders from Edit Paths, then refresh the workspace.</li>
						</ul>
					</div>

					<div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
						<div className="bg-base-100 border-base-content/10 rounded-2xl border p-3">
							<div className="text-sm font-semibold">Workspaces</div>
							<div className="text-base-content/70 mt-1 text-xs">{workspaces.length} configured</div>
						</div>
						<div className="bg-base-100 border-base-content/10 rounded-2xl border p-3">
							<div className="text-sm font-semibold">Enabled</div>
							<div className="text-base-content/70 mt-1 text-xs">{enabledCount} available</div>
						</div>
						<div className="bg-base-100 border-base-content/10 rounded-2xl border p-3">
							<div className="text-sm font-semibold">Filesystem roots</div>
							<div className="text-base-content/70 mt-1 text-xs">{filesystemCount} configured</div>
						</div>
						<div className="bg-base-100 border-base-content/10 rounded-2xl border p-3">
							<div className="text-sm font-semibold">Attached sources</div>
							<div className="text-base-content/70 mt-1 text-xs">{sourceCount} total</div>
						</div>
					</div>

					<div className="border-base-content/10 bg-base-100 flex flex-col gap-3 rounded-2xl border p-3 sm:flex-row sm:items-center">
						<label className="input input-sm flex grow items-center gap-2 rounded-xl">
							<FiSearch size={14} />
							<input
								type="search"
								className="grow"
								value={searchQuery}
								onChange={event => {
									setSearchQuery(event.currentTarget.value);
								}}
								placeholder="Search workspaces, IDs, paths, and sources..."
								spellCheck="false"
							/>
							{searchQuery ? (
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									onClick={() => {
										setSearchQuery('');
									}}
									aria-label="Clear workspace search"
								>
									<FiX size={12} />
								</button>
							) : null}
						</label>

						<div className="text-base-content/70 shrink-0 text-xs">
							{visibleWorkspaces.length} of {workspaces.length} workspaces
						</div>
					</div>

					<div className="pb-8">
						{visibleWorkspaces.map(workspace => (
							<WorkspaceCard
								key={workspace.rootID}
								workspace={workspace}
								existingDisplayNames={existingDisplayNames.filter(
									name => name.toLowerCase() !== workspace.displayName.toLowerCase()
								)}
								onWorkspaceChange={replaceWorkspace}
								onUpdateWorkspace={payload => updateWorkspace(workspace, payload)}
								onRequestDelete={setWorkspaceToDelete}
							/>
						))}

						{workspaces.length === 0 ? (
							<ManagementEmptyState className="mt-4">
								No workspaces configured. Add a project root to get started.
							</ManagementEmptyState>
						) : null}

						{workspaces.length > 0 && visibleWorkspaces.length === 0 ? (
							<ManagementEmptyState className="mt-4">No workspaces match the current search.</ManagementEmptyState>
						) : null}
					</div>
				</ManagementPageContent>

				<WorkspaceUpsertModal
					isOpen={isCreateOpen}
					onClose={() => {
						setIsCreateOpen(false);
					}}
					onSubmit={createWorkspace}
					existingDisplayNames={existingDisplayNames}
				/>

				<ModalConfirmDialog
					isOpen={workspaceToDelete !== null}
					onClose={() => {
						setWorkspaceToDelete(null);
					}}
					title="Delete Workspace"
					message={
						<div className="space-y-2 text-sm">
							<p>
								Delete workspace <span className="font-semibold">{workspaceToDelete?.displayName}</span>?
							</p>
							<p className="text-base-content/70">
								This removes the workspace catalog and stored records. It does not delete project files.
							</p>
						</div>
					}
					confirmLabel="Delete Workspace"
					busyLabel="Deleting..."
					confirmTone="error"
					onConfirm={deleteWorkspace}
					blockCancel
				/>

				<ActionDeniedAlertModal
					isOpen={Boolean(alertMessage)}
					onClose={() => {
						setAlertMessage('');
					}}
					message={alertMessage}
				/>
			</div>
		</PageFrame>
	);
}
