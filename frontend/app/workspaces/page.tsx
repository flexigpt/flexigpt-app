import { useCallback, useMemo, useState } from 'react';
import { FiFolderPlus, FiSearch, FiX } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import type { WorkspaceDirectoryListItem, WorkspaceDirectoryView } from '@/spec/workspace';
import { ArtifactState } from '@/spec/artifact';
import { workspaceDirectoryListItemFromView } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';
import { omitManyKeys } from '@/lib/obj_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';
import { usePendingActions } from '@/hooks/use_pending_actions';

import { workspaceManagementAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { Loader } from '@/components/loader';
import { DefaultArtifactControl } from '@/components/managementui/default_artifact_control';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { ModalConfirmDialog } from '@/components/modal/modal_confirm_dialog';
import { PageFrame } from '@/components/page_frame';

import { WorkspaceDirectoryCard } from '@/workspaces/workspace_directory_card';
import { WorkspaceDirectoryManagementModal } from '@/workspaces/workspace_directory_management_modal';
import { WorkspaceDirectoryRegistrationModal } from '@/workspaces/workspace_directory_registration_modal';

const DIRECTORY_PAGE_SIZE = 100;
const MAX_DIRECTORY_PAGE_HOPS = 10_000;

function directoryKey(directory: Pick<WorkspaceDirectoryListItem, 'ref'>): string {
	return directory.ref.rootID;
}

interface WorkspaceDirectoryRemovalTarget {
	ref: WorkspaceDirectoryListItem['ref'];
	displayName: string;
	expectedRevision: number;
}

interface DefaultWorkspaceOption {
	ref: ArtifactRef;
	displayName: string;
	isSelectable: boolean;
	availabilityReason?: string;
}

interface WorkspaceDefaultsData {
	defaultWorkspace: ArtifactRef | null;
	directories: WorkspaceDirectoryView[];
}

function refsEqual(left: ArtifactRef | null | undefined, right: ArtifactRef | null | undefined): boolean {
	return left?.rootID === right?.rootID && left?.artifactID === right?.artifactID;
}

async function loadAllWorkspaceDirectories(signal: AbortSignal): Promise<WorkspaceDirectoryListItem[]> {
	const items: WorkspaceDirectoryListItem[] = [];
	const seenCursors = new Set<string>();
	let cursor: string | undefined;

	for (let hop = 0; hop < MAX_DIRECTORY_PAGE_HOPS; hop += 1) {
		const page = await workspaceManagementAPI.listWorkspaceDirectories({
			cursor,
			limit: DIRECTORY_PAGE_SIZE,
		});
		throwIfAborted(signal);

		items.push(...(page?.items ?? []));

		if (!page?.nextCursor) {
			return items.toSorted((left, right) =>
				left.rootDisplayName.localeCompare(right.rootDisplayName, undefined, {
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
		initialData: [] as WorkspaceDirectoryListItem[],
	});

	const loadWorkspaceDefaults = useCallback(async (signal: AbortSignal): Promise<WorkspaceDefaultsData> => {
		const [defaultWorkspace, fullDirectories] = await Promise.all([
			workspaceManagementAPI.getDefaultWorkspace().catch(() => null),
			workspaceManagementAPI.listComposerWorkspaceDirectories(true),
		]);
		throwIfAborted(signal);

		return {
			defaultWorkspace,
			directories: fullDirectories,
		};
	}, []);
	const {
		data: workspaceDefaults,
		error: workspaceDefaultsError,
		isRefreshing: isRefreshingWorkspaceDefaults,
		reloadOrThrow: reloadWorkspaceDefaultsOrThrow,
	} = useAsyncResource(loadWorkspaceDefaults, {
		initialData: { defaultWorkspace: null, directories: [] } satisfies WorkspaceDefaultsData,
	});

	const [search, setSearch] = useState('');
	const [isRegisterOpen, setIsRegisterOpen] = useState(false);
	const [managedDirectoryKey, setManagedDirectoryKey] = useState<string | null>(null);
	const [directoryToRemove, setDirectoryToRemove] = useState<WorkspaceDirectoryRemovalTarget | null>(null);
	const [actionError, setActionError] = useState('');
	const [detailsByDirectoryKey, setDetailsByDirectoryKey] = useState<Record<string, WorkspaceDirectoryView>>({});
	const [loadingDirectoryKeys, setLoadingDirectoryKeys] = useState<Set<string>>(new Set());
	const [directoryErrors, setDirectoryErrors] = useState<Record<string, string>>({});
	const { isPending: isDefaultWorkspacePending, runAction: runDefaultWorkspaceAction } = usePendingActions();

	const visibleDirectories = useMemo(() => {
		const query = search.trim().toLowerCase();
		if (!query) {
			return directories;
		}

		return directories.filter(directory => directory.rootDisplayName.toLowerCase().includes(query));
	}, [directories, search]);

	const managedDirectory = managedDirectoryKey ? detailsByDirectoryKey[managedDirectoryKey] : undefined;

	const defaultWorkspaceOptions = useMemo<DefaultWorkspaceOption[]>(
		() =>
			workspaceDefaults.directories
				.flatMap(directory =>
					directory.workspaces.map(workspace => {
						const artifact = workspace.workspace.artifact;
						const ref = {
							rootID: artifact.rootID,
							artifactID: artifact.id,
						};
						const isSelectable = directory.enabled && artifact.enabled && artifact.state === ArtifactState.Available;

						return {
							ref,
							displayName: artifact.displayName || artifact.logicalName || 'Workspace',
							isSelectable,
							availabilityReason: isSelectable
								? undefined
								: 'Enable an available Workspace directory and Workspace before setting it as default.',
						};
					})
				)
				.toSorted((left, right) =>
					left.displayName.localeCompare(right.displayName, undefined, { sensitivity: 'base' })
				),
		[workspaceDefaults.directories]
	);

	const defaultWorkspaceDirectoryKeys = useMemo(
		() =>
			new Set(
				workspaceDefaults.directories.flatMap(directory =>
					directory.workspaces.some(workspace =>
						refsEqual(workspaceDefaults.defaultWorkspace, {
							rootID: workspace.workspace.artifact.rootID,
							artifactID: workspace.workspace.artifact.id,
						})
					)
						? [directory.ref.rootID]
						: []
				)
			),
		[workspaceDefaults.defaultWorkspace, workspaceDefaults.directories]
	);

	const replaceDirectory = useCallback(
		(next: WorkspaceDirectoryView) => {
			const summary = workspaceDirectoryListItemFromView(next);
			setDirectories(previous =>
				previous
					.map(value => (directoryKey(value) === directoryKey(summary) ? summary : value))
					.toSorted((left, right) =>
						left.rootDisplayName.localeCompare(right.rootDisplayName, undefined, {
							sensitivity: 'base',
						})
					)
			);
			setDetailsByDirectoryKey(previous => ({
				...previous,
				[directoryKey(summary)]: next,
			}));
		},
		[setDirectories]
	);

	const changeDefaultWorkspace = useCallback(
		async (workspace?: ArtifactRef) => {
			if (workspace) {
				await workspaceManagementAPI.setDefaultWorkspace(workspace);
			} else {
				await workspaceManagementAPI.clearDefaultWorkspace();
			}

			await reloadWorkspaceDefaultsOrThrow();
		},
		[reloadWorkspaceDefaultsOrThrow]
	);

	const handleDirectoryChanged = useCallback(
		(next: WorkspaceDirectoryView) => {
			replaceDirectory(next);
			void reloadWorkspaceDefaultsOrThrow().catch(() => undefined);
		},
		[reloadWorkspaceDefaultsOrThrow, replaceDirectory]
	);

	const loadDirectoryDetails = useCallback(
		async (directory: WorkspaceDirectoryListItem) => {
			const key = directoryKey(directory);
			if (detailsByDirectoryKey[key]) {
				setManagedDirectoryKey(key);
				return;
			}
			if (loadingDirectoryKeys.has(key)) {
				return;
			}

			setLoadingDirectoryKeys(previous => new Set(previous).add(key));
			setDirectoryErrors(previous => {
				let next = { ...previous };
				next = omitManyKeys(next, [key]);
				return next;
			});

			try {
				const detail = await workspaceManagementAPI.getWorkspaceDirectory(directory.ref);
				replaceDirectory(detail);
				setManagedDirectoryKey(key);
			} catch (cause) {
				setDirectoryErrors(previous => ({
					...previous,
					[key]: cause instanceof Error ? cause.message : 'Workspace directory details could not be loaded.',
				}));
			} finally {
				setLoadingDirectoryKeys(previous => {
					const next = new Set(previous);
					next.delete(key);
					return next;
				});
			}
		},
		[detailsByDirectoryKey, replaceDirectory, loadingDirectoryKeys]
	);

	const registerDirectory = async (path: string) => {
		const directory = await workspaceManagementAPI.registerWorkspaceDirectory(path);
		const summary = workspaceDirectoryListItemFromView(directory);

		setDirectories(previous => {
			const withoutCurrent = previous.filter(value => directoryKey(value) !== directoryKey(summary));
			return [...withoutCurrent, summary].toSorted((left, right) =>
				left.rootDisplayName.localeCompare(right.rootDisplayName, undefined, {
					sensitivity: 'base',
				})
			);
		});
		setDetailsByDirectoryKey(previous => ({
			...previous,
			[directoryKey(summary)]: directory,
		}));
		await reloadWorkspaceDefaultsOrThrow().catch(() => undefined);
	};

	const removeDirectory = async () => {
		if (!directoryToRemove) {
			return;
		}

		try {
			await workspaceManagementAPI.removeWorkspaceDirectory(directoryToRemove.ref, directoryToRemove.expectedRevision);
			setDirectories(previous => previous.filter(value => directoryKey(value) !== directoryToRemove.ref.rootID));
			setDetailsByDirectoryKey(previous => omitManyKeys(previous, [directoryToRemove.ref.rootID]));
			setManagedDirectoryKey(previous => (previous === directoryToRemove.ref.rootID ? null : previous));
			setDirectoryToRemove(null);
			await reloadWorkspaceDefaultsOrThrow().catch(() => undefined);
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
					description="Add repositories and review the exact files, Skills, and integrations each Workspace contributes."
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

					<DefaultArtifactControl
						label="Default Workspace"
						automaticLabel="Use no default Workspace"
						options={defaultWorkspaceOptions}
						defaultRef={workspaceDefaults.defaultWorkspace}
						pending={isDefaultWorkspacePending('default-workspace')}
						onChange={ref => {
							void runDefaultWorkspaceAction('default-workspace', () => changeDefaultWorkspace(ref)).catch(
								(cause: unknown) => {
									setActionError(
										cause instanceof Error ? cause.message : 'The default Workspace could not be changed.'
									);
								}
							);
						}}
					/>

					{workspaceDefaultsError ? (
						<ManagementResourceError
							title="Workspace defaults could not be loaded"
							error={workspaceDefaultsError}
							isRetrying={isRefreshingWorkspaceDefaults}
							onRetry={reloadWorkspaceDefaultsOrThrow}
						/>
					) : null}

					<div className="border-base-content/10 bg-base-100 rounded-2xl border p-4 text-sm">
						<div className="font-semibold">How repository Workspaces work</div>
						<ul className="text-base-content/70 mt-2 list-disc space-y-1 pl-5 text-xs">
							<li>Automatic setup detects instruction files such as `AGENTS.md` and `CLAUDE.md`.</li>
							<li>README and documentation files are shown separately from model instructions.</li>
							<li>Repository Skills and MCP servers are listed with their relative source paths.</li>
							<li>Add `workspace.yaml` only when the repository needs a customized setup.</li>
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
								placeholder="Search repositories..."
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
						{visibleDirectories.map(directory => {
							const key = directoryKey(directory);

							return (
								<WorkspaceDirectoryCard
									key={directory.ref.rootID}
									directory={directory}
									isLoading={loadingDirectoryKeys.has(key)}
									error={directoryErrors[key]}
									isDefaultWorkspace={defaultWorkspaceDirectoryKeys.has(directory.ref.rootID)}
									onManage={() => {
										void loadDirectoryDetails(directory);
									}}
									onRequestRemove={() => {
										setDirectoryToRemove({
											ref: directory.ref,
											displayName: directory.rootDisplayName,
											expectedRevision: directory.directorySourceRevision,
										});
									}}
								/>
							);
						})}

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

			{managedDirectory ? (
				<WorkspaceDirectoryManagementModal
					isOpen
					directory={managedDirectory}
					onClose={() => {
						setManagedDirectoryKey(null);
					}}
					onChanged={handleDirectoryChanged}
					onRequestRemove={value => {
						setManagedDirectoryKey(null);
						setDirectoryToRemove({
							ref: value.ref,
							displayName: value.root.displayName,
							expectedRevision: value.directorySource.revision,
						});
					}}
				/>
			) : null}

			<ModalConfirmDialog
				isOpen={directoryToRemove !== null}
				onClose={() => {
					setDirectoryToRemove(null);
				}}
				title="Remove Workspace Directory"
				message={
					<div className="space-y-2 text-sm">
						<p>
							Remove <span className="font-semibold">{directoryToRemove?.displayName}</span>?
						</p>
						<p className="text-base-content/70">
							This removes the directory from FlexiGPT. Repository files are never deleted.
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
