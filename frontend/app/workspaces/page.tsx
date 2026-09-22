import { useCallback, useMemo, useState } from 'react';
import { FiFolderPlus, FiSearch, FiX } from 'react-icons/fi';

import type { WorkspaceDirectoryView } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import { workspaceManagementAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { Loader } from '@/components/loader';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { ModalConfirmDialog } from '@/components/modal/modal_confirm_dialog';
import { PageFrame } from '@/components/page_frame';

import { WorkspaceDefaultPolicyModal } from '@/workspaces/workspace_default_policy_modal';
import { WorkspaceDirectoryCard } from '@/workspaces/workspace_directory_card';
import { WorkspaceDirectoryRegistrationModal } from '@/workspaces/workspace_directory_registration_modal';

const DIRECTORY_PAGE_SIZE = 100;
const MAX_DIRECTORY_PAGE_HOPS = 10_000;

function directoryKey(directory: WorkspaceDirectoryView): string {
	return directory.ref.rootID;
}

async function loadAllWorkspaceDirectories(signal: AbortSignal): Promise<WorkspaceDirectoryView[]> {
	const items: WorkspaceDirectoryView[] = [];
	const seenCursors = new Set<string>();
	let cursor: string | undefined;

	for (let hop = 0; hop < MAX_DIRECTORY_PAGE_HOPS; hop += 1) {
		const page = await workspaceManagementAPI.listWorkspaceDirectories({
			cursor,
			limit: DIRECTORY_PAGE_SIZE,
		});
		throwIfAborted(signal);

		items.push(...page.items);

		if (!page.nextCursor) {
			return items.toSorted((left, right) =>
				left.root.displayName.localeCompare(right.root.displayName, undefined, {
					sensitivity: 'base',
				})
			);
		}
		if (seenCursors.has(page.nextCursor)) {
			throw new Error('Workspace directory pagination returned a repeated cursor.');
		}

		seenCursors.add(page.nextCursor);
		cursor = page.nextCursor;
	}

	throw new Error(`Workspace directory pagination exceeded ${MAX_DIRECTORY_PAGE_HOPS} pages.`);
}

// oxlint-disable-next-line no-restricted-exports
export default function WorkspaceDirectoryManagement() {
	const loadDirectories = useCallback((signal: AbortSignal) => loadAllWorkspaceDirectories(signal), []);
	const {
		data: directories,
		error,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setDirectories,
	} = useAsyncResource(loadDirectories, {
		initialData: [] as WorkspaceDirectoryView[],
	});

	const [search, setSearch] = useState('');
	const [isRegisterOpen, setIsRegisterOpen] = useState(false);
	const [isPolicyOpen, setIsPolicyOpen] = useState(false);
	const [directoryToRemove, setDirectoryToRemove] = useState<WorkspaceDirectoryView | null>(null);
	const [actionError, setActionError] = useState('');

	const visibleDirectories = useMemo(() => {
		const query = search.trim().toLowerCase();
		if (!query) {
			return directories;
		}

		return directories.filter(directory =>
			[
				directory.root.displayName,
				directory.root.id,
				directory.directorySource.displayName,
				directory.policyID,
				...directory.workspaces.map(entry => entry.workspace.artifact.displayName),
				...directory.workspaces.map(entry => entry.workspace.artifact.logicalName),
				...directory.workspaces.map(entry => entry.manifestLocator),
			]
				.filter(Boolean)
				.join('\n')
				.toLowerCase()
				.includes(query)
		);
	}, [directories, search]);

	const replaceDirectory = (next: WorkspaceDirectoryView) => {
		setDirectories(previous =>
			previous
				.map(value => (directoryKey(value) === directoryKey(next) ? next : value))
				.toSorted((left, right) =>
					left.root.displayName.localeCompare(right.root.displayName, undefined, {
						sensitivity: 'base',
					})
				)
		);
	};

	const registerDirectory = async (path: string) => {
		const directory = await workspaceManagementAPI.registerWorkspaceDirectory(path);

		setDirectories(previous => {
			const withoutCurrent = previous.filter(value => directoryKey(value) !== directoryKey(directory));
			return [...withoutCurrent, directory].toSorted((left, right) =>
				left.root.displayName.localeCompare(right.root.displayName, undefined, {
					sensitivity: 'base',
				})
			);
		});
	};

	const removeDirectory = async () => {
		if (!directoryToRemove) {
			return;
		}

		try {
			await workspaceManagementAPI.removeWorkspaceDirectory(
				directoryToRemove.ref,
				directoryToRemove.directorySource.revision
			);
			setDirectories(previous => previous.filter(value => directoryKey(value) !== directoryKey(directoryToRemove)));
			setDirectoryToRemove(null);
		} catch (cause) {
			setActionError(cause instanceof Error ? cause.message : 'Workspace directory removal failed.');
		}
	};

	if (isLoading && !hasResolved) {
		return <Loader text="Loading Workspace directories..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Workspaces"
					description="Select repository directories and manage the effective Workspace declarations discovered in each one."
					width="wide"
					actions={
						<button
							type="button"
							className="btn btn-ghost rounded-xl"
							onClick={() => {
								setIsRegisterOpen(true);
							}}
						>
							<FiFolderPlus size={18} />
							Add Workspace Directory
						</button>
					}
				/>

				<ManagementPageContent width="wide">
					{error ? (
						<ManagementResourceError
							title="Workspace directories could not be loaded"
							error={error}
							isRetrying={isRefreshing}
							onRetry={reloadOrThrow}
						/>
					) : null}

					<div className="border-base-content/10 bg-base-100 rounded-2xl border p-4 text-sm">
						<div className="font-semibold">How Workspace declarations work</div>
						<ul className="text-base-content/70 mt-2 list-disc space-y-1 pl-5 text-xs">
							<li>A directory without a Workspace manifest receives the compiled default policy.</li>
							<li>
								Add `workspace.yaml`, `workspace.yml`, `workspace.json`, or `*.workspace.*` to replace the default
								policy with repository declarations.
							</li>
							<li>Multiple valid Workspace manifests become peer Workspaces that conversations can select.</li>
							<li>Invalid Workspace manifests intentionally block default-policy fallback until fixed.</li>
						</ul>
					</div>

					<div className="border-base-content/10 bg-base-100 flex flex-col gap-3 rounded-2xl border p-3 sm:flex-row sm:items-center">
						<div className="input input-sm flex grow items-center gap-2 rounded-xl">
							<label htmlFor="workspace-directory-search" className="sr-only">
								Search Workspace directories
							</label>
							<FiSearch size={14} />
							<input
								id="workspace-directory-search"
								type="search"
								className="grow"
								value={search}
								onChange={event => {
									setSearch(event.currentTarget.value);
								}}
								placeholder="Search directory roots and Workspace declarations..."
							/>
							{search ? (
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									onClick={() => {
										setSearch('');
									}}
									aria-label="Clear Workspace directory search"
								>
									<FiX size={12} />
								</button>
							) : null}
						</div>

						<div className="text-base-content/60 text-xs">
							{visibleDirectories.length} of {directories.length} directories
						</div>
					</div>

					<div className="space-y-4 pb-8">
						{visibleDirectories.map(directory => (
							<WorkspaceDirectoryCard
								key={directory.ref.rootID}
								directory={directory}
								onChanged={replaceDirectory}
								onRequestRemove={setDirectoryToRemove}
								onShowDefaultPolicy={() => {
									setIsPolicyOpen(true);
								}}
							/>
						))}

						{directories.length === 0 ? (
							<ManagementEmptyState>
								No Workspace directories are configured. Add a repository directory to get started.
							</ManagementEmptyState>
						) : null}

						{directories.length > 0 && visibleDirectories.length === 0 ? (
							<ManagementEmptyState>No Workspace directories match the current search.</ManagementEmptyState>
						) : null}
					</div>
				</ManagementPageContent>
			</div>

			<WorkspaceDirectoryRegistrationModal
				isOpen={isRegisterOpen}
				onClose={() => {
					setIsRegisterOpen(false);
				}}
				onRegister={registerDirectory}
			/>

			<WorkspaceDefaultPolicyModal
				isOpen={isPolicyOpen}
				onClose={() => {
					setIsPolicyOpen(false);
				}}
			/>

			<ModalConfirmDialog
				isOpen={directoryToRemove !== null}
				onClose={() => {
					setDirectoryToRemove(null);
				}}
				title="Remove Workspace Directory"
				message={
					<div className="space-y-2 text-sm">
						<p>
							Remove <span className="font-semibold">{directoryToRemove?.root.displayName}</span>?
						</p>
						<p className="text-base-content/70">
							This removes the local Workspace Root and Source registrations. Repository files are never deleted.
						</p>
					</div>
				}
				confirmLabel="Remove Directory"
				busyLabel="Removing..."
				confirmTone="error"
				onConfirm={removeDirectory}
				blockCancel
			/>

			<ActionDeniedAlertModal
				isOpen={Boolean(actionError)}
				onClose={() => {
					setActionError('');
				}}
				message={actionError}
			/>
		</PageFrame>
	);
}
