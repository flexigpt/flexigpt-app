import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiFolderPlus, FiSearch, FiX } from 'react-icons/fi';

import type { UpdateWorkspaceBody, WorkspaceView } from '@/spec/workspace';
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

import {
	createEmptyWorkspaceCollection,
	createFilesystemWorkspaceCollection,
	listAllWorkspaces,
	workspaceRefKey,
} from '@/workspaces/lib/workspace_api_utils';
import { getErrorMessage, sortWorkspaces, workspaceMatchesSearch } from '@/workspaces/lib/workspace_utils';
import { WorkspaceCard } from '@/workspaces/workspace_card';
import type { WorkspaceSetupSubmission } from '@/workspaces/workspace_setup_modal';
import { WorkspaceSetupModal } from '@/workspaces/workspace_setup_modal';

async function loadWorkspaces(signal: AbortSignal): Promise<WorkspaceView[]> {
	const workspaces = await listAllWorkspaces();
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
	const retiredWorkspaceRevisionsRef = useRef(new Map<string, number>());

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
		};
	}, []);

	const existingDisplayNames = useMemo(() => workspaces.map(workspace => workspace.displayName), [workspaces]);

	const visibleWorkspaces = useMemo(
		() => workspaces.filter(workspace => workspaceMatchesSearch(workspace, searchQuery)),
		[searchQuery, workspaces]
	);

	const enabledCount = useMemo(() => workspaces.filter(workspace => workspace.enabled).length, [workspaces]);
	const filesystemCount = workspaces.filter(workspace => workspace.mode === WorkspaceMode.Filesystem).length;
	const sourceCount = workspaces.reduce((total, workspace) => total + workspace.attachments.length, 0);

	const replaceWorkspace = useCallback(
		(nextWorkspace: WorkspaceView) => {
			const nextKey = workspaceRefKey(nextWorkspace.workspace);
			setWorkspaces(previous => {
				const currentWorkspace = previous.find(workspace => workspaceRefKey(workspace.workspace) === nextKey);

				// Catalog reads can complete after a newer Workspace mutation. Never
				// replace a newer Collection revision with an older response.
				if (currentWorkspace && currentWorkspace.revision > nextWorkspace.revision) {
					return previous;
				}

				const nextWorkspaces = currentWorkspace
					? previous.map(workspace => (workspaceRefKey(workspace.workspace) === nextKey ? nextWorkspace : workspace))
					: [...previous, nextWorkspace];

				return sortWorkspaces(nextWorkspaces);
			});
		},
		[setWorkspaces]
	);

	const createWorkspace = useCallback(
		async (submission: WorkspaceSetupSubmission) => {
			const preferredRootID = workspaces[0]?.workspace.rootID;
			let created: WorkspaceView;

			if (submission.kind === 'filesystem') {
				created = await createFilesystemWorkspaceCollection(submission.payload, preferredRootID);
			} else if (submission.kind === 'empty') {
				created = await createEmptyWorkspaceCollection(submission.payload, preferredRootID);
			} else {
				throw new Error('Expected a new Workspace payload.');
			}

			if (mountedRef.current) {
				replaceWorkspace(created);
			}

			if (submission.kind === 'empty') {
				return;
			}

			try {
				await workspaceAPI.refreshWorkspace(created.workspace);
				const refreshed = await workspaceAPI.getWorkspace(created.workspace);
				if (mountedRef.current) {
					replaceWorkspace(refreshed);
				}
			} catch (error) {
				if (mountedRef.current) {
					setAlertMessage(
						`Workspace was created, but initial discovery failed. Open the workspace and retry Refresh. ${getErrorMessage(
							error,
							''
						)}`.trim()
					);
				}
			}
		},
		[replaceWorkspace, workspaces]
	);

	const updateWorkspace = useCallback(
		async (workspace: WorkspaceView, payload: UpdateWorkspaceBody): Promise<WorkspaceView> => {
			const updated = await workspaceAPI.updateWorkspace(workspace.workspace, payload);
			if (mountedRef.current) {
				replaceWorkspace(updated);
			}
			return updated;
		},
		[replaceWorkspace]
	);

	const deleteWorkspace = async () => {
		if (!workspaceToDelete) {
			return;
		}

		const deletingWorkspace = workspaceToDelete;
		const deletingRef = deletingWorkspace.workspace;
		const deletingKey = workspaceRefKey(deletingRef);
		let purgeRevision = retiredWorkspaceRevisionsRef.current.get(deletingKey);

		if (purgeRevision === undefined) {
			const retired = await workspaceAPI.retireWorkspace(deletingRef, deletingWorkspace.revision);
			purgeRevision = retired.revision;
			retiredWorkspaceRevisionsRef.current.set(deletingKey, purgeRevision);
		}

		try {
			await workspaceAPI.purgeWorkspace(deletingRef, purgeRevision);
		} catch (error) {
			const details = getErrorMessage(error, '');
			throw new Error(
				`Workspace was retired, but its stored records could not be purged. Retry Delete Workspace to finish cleanup. ${details}`.trim(),
				{ cause: error }
			);
		}

		retiredWorkspaceRevisionsRef.current.delete(deletingKey);

		if (mountedRef.current) {
			setWorkspaces(previous => previous.filter(workspace => workspaceRefKey(workspace.workspace) !== deletingKey));
			setWorkspaceToDelete(null);
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
					description="Manage project sources, discovered context documents, workspace skills, and conversation permissions."
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
							<li>
								Create a filesystem workspace from a project root, or create an empty workspace and attach sources
								later.
							</li>
							<li>AGENTS.md, CLAUDE.md, optional README.md, and .skills folders are discovered automatically.</li>
							<li>Add project-specific Context files or Skill folders from Edit Workspace, then refresh discovery.</li>
							<li>Manage library, package, overlay, and primary source attachments from each workspace.</li>
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
						<div className="input input-sm flex grow items-center gap-2 rounded-xl">
							<label htmlFor="workspace-search" className="sr-only">
								Search Workspaces
							</label>
							<FiSearch size={14} aria-hidden="true" />
							<input
								id="workspace-search"
								type="search"
								className="grow"
								value={searchQuery}
								onChange={event => {
									setSearchQuery(event.currentTarget.value);
								}}
								placeholder="Search workspaces, project paths, and discovery paths..."
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
						</div>

						<div className="text-base-content/70 shrink-0 text-xs">
							{visibleWorkspaces.length} of {workspaces.length} workspaces
						</div>
					</div>

					<div className="pb-8">
						{visibleWorkspaces.map(workspace => (
							<WorkspaceCard
								key={workspaceRefKey(workspace.workspace)}
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
								No workspaces configured. Add a project root or create an empty workspace to get started.
							</ManagementEmptyState>
						) : null}

						{workspaces.length > 0 && visibleWorkspaces.length === 0 ? (
							<ManagementEmptyState className="mt-4">No workspaces match the current search.</ManagementEmptyState>
						) : null}
					</div>
				</ManagementPageContent>

				<WorkspaceSetupModal
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
								This removes the workspace index and saved workspace settings. It does not delete project files.
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
