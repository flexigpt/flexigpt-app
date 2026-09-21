import type { SubmitEventHandler } from 'react';
import { useCallback, useMemo, useState } from 'react';
import { FiAlertCircle, FiCheck, FiFolder, FiLink, FiPlus, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import type { ArtifactSourceSummary } from '@/spec/artifact';
import type {
	UpdateWorkspaceAttachmentBody,
	WorkspaceAttachmentSettings,
	WorkspaceAttachmentView,
	WorkspaceView,
} from '@/spec/workspace';
import { WorkspaceAttachmentRole } from '@/spec/workspace';

import { throwIfAborted } from '@/lib/async_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';
import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { backendAPI, workspaceManagementAPI } from '@/apis/baseapi';

import { Dropdown } from '@/components/dropdown';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { workspaceRefKey } from '@/workspaces/lib/workspace_api_utils';
import { getErrorMessage } from '@/workspaces/lib/workspace_utils';

type AttachmentEdit = Pick<UpdateWorkspaceAttachmentBody, 'role' | 'enabled' | 'settings'>;

const FILESYSTEM_SOURCE_KIND = 'fs-directory';

const NON_PRIMARY_ROLES: WorkspaceAttachmentRole[] = [
	WorkspaceAttachmentRole.Library,
	WorkspaceAttachmentRole.AttachedPackage,
	WorkspaceAttachmentRole.Overlay,
];

const NON_PRIMARY_ROLE_ITEMS = Object.fromEntries(NON_PRIMARY_ROLES.map(role => [role, { isEnabled: true }])) as Record<
	WorkspaceAttachmentRole,
	{ isEnabled: boolean }
>;

interface WorkspaceSourceCatalogData {
	sources: ArtifactSourceSummary[];
}

const EMPTY_WORKSPACE_SOURCE_CATALOG: WorkspaceSourceCatalogData = {
	sources: [],
};

function attachmentRoleLabel(role: WorkspaceAttachmentRole): string {
	switch (role) {
		case WorkspaceAttachmentRole.Primary:
			return 'Primary project';
		case WorkspaceAttachmentRole.Library:
			return 'Library';
		case WorkspaceAttachmentRole.AttachedPackage:
			return 'Attached package';
		case WorkspaceAttachmentRole.Overlay:
			return 'Overlay';
		default:
			return role;
	}
}

function sourceLabel(source: ArtifactSourceSummary): string {
	return `${source.displayName} (${source.kind})`;
}

function sourceDisplayNameFromPath(path: string): string {
	const normalized = path.trim().replaceAll('\\', '/').replaceAll(/\/+$/g, '');
	const name = normalized.slice(normalized.lastIndexOf('/') + 1);

	return name || 'Workspace folder';
}

function attachmentSettings(recursive: boolean, authoritative: boolean): WorkspaceAttachmentSettings {
	return {
		recursive,
		authoritative,
	};
}

function sortSources(sources: ArtifactSourceSummary[]): ArtifactSourceSummary[] {
	return [...sources].toSorted((left, right) => {
		const displayNameOrder = left.displayName.localeCompare(right.displayName, undefined, {
			sensitivity: 'base',
		});
		return displayNameOrder !== 0 ? displayNameOrder : left.id.localeCompare(right.id);
	});
}

async function loadWorkspaceSourceCatalog(signal: AbortSignal): Promise<WorkspaceSourceCatalogData> {
	const sources = await workspaceManagementAPI.listWorkspaceSourcesForManagement();
	throwIfAborted(signal);

	return {
		sources: sortSources(sources),
	};
}

interface WorkspaceSourceAttachmentCardProps {
	attachment: WorkspaceAttachmentView;
	source?: ArtifactSourceSummary;
	busy: boolean;
	onSave: (attachment: WorkspaceAttachmentView, edit: AttachmentEdit) => Promise<void>;
	onRequestDetach: (attachment: WorkspaceAttachmentView) => void;
}

function WorkspaceSourceAttachmentCard({
	attachment,
	source,
	busy,
	onSave,
	onRequestDetach,
}: WorkspaceSourceAttachmentCardProps) {
	const isPrimary = attachment.role === WorkspaceAttachmentRole.Primary;
	const [role, setRole] = useState<WorkspaceAttachmentRole>(attachment.role);
	const [enabled, setEnabled] = useState(attachment.enabled);
	const [recursive, setRecursive] = useState(attachment.settings.recursive ?? false);
	const [authoritative, setAuthoritative] = useState(attachment.settings.authoritative ?? false);
	const title = attachment.path ?? attachment.sourceDisplayName ?? source?.displayName ?? 'Attached Source';
	const subtitle = source ? `${source.kind} · ${source.id}` : (attachment.sourceKind ?? attachment.sourceID);

	const save = async () => {
		await onSave(attachment, {
			role: isPrimary ? WorkspaceAttachmentRole.Primary : role,
			enabled: isPrimary ? true : enabled,
			settings: isPrimary ? {} : attachmentSettings(recursive, authoritative),
		});
	};

	return (
		<div className="border-base-content/10 bg-base-100 rounded-2xl border p-3">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
				<div className="min-w-0">
					<div className="flex flex-wrap items-center gap-2">
						<div className="min-w-0 truncate text-sm font-semibold">{title}</div>
						<span className="badge badge-ghost badge-xs">{attachmentRoleLabel(attachment.role)}</span>
						<span className={`badge badge-xs ${attachment.enabled ? 'badge-success' : 'badge-warning'}`}>
							{attachment.enabled ? 'Attached' : 'Attachment disabled'}
						</span>
						{source && !source.enabled ? <span className="badge badge-warning badge-xs">Source disabled</span> : null}
					</div>
					<div className="text-base-content/60 mt-1 truncate font-mono text-xs">{subtitle}</div>
					<div className="text-base-content/60 mt-1 text-xs">
						Source ID: <span className="font-mono">{attachment.sourceID}</span>
					</div>
				</div>

				<button
					type="button"
					className="btn btn-sm btn-ghost text-error rounded-xl"
					disabled={busy}
					onClick={() => {
						onRequestDetach(attachment);
					}}
				>
					<FiTrash2 size={14} />
					<span>{isPrimary ? 'Clear Primary' : 'Detach'}</span>
				</button>
			</div>

			<details className="border-base-content/10 mt-3 rounded-xl border p-3">
				<summary className="cursor-pointer text-sm font-medium">Attachment settings</summary>

				<div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
					<label className="space-y-1">
						<span className="text-xs font-medium">Workspace role</span>
						{isPrimary ? (
							<div className="input input-sm flex items-center rounded-xl">{attachmentRoleLabel(attachment.role)}</div>
						) : (
							<Dropdown<WorkspaceAttachmentRole>
								dropdownItems={NON_PRIMARY_ROLE_ITEMS}
								orderedKeys={NON_PRIMARY_ROLES}
								selectedKey={role}
								onChange={setRole}
								disabled={busy}
								title="Select workspace source role"
								getDisplayName={attachmentRoleLabel}
							/>
						)}
					</label>

					<label className="flex items-center gap-3 pt-5 text-sm">
						<input
							type="checkbox"
							className="toggle toggle-accent toggle-sm"
							checked={isPrimary ? true : enabled}
							disabled={busy || isPrimary}
							onChange={event => {
								setEnabled(event.currentTarget.checked);
							}}
						/>
						<span>{isPrimary ? 'Primary Sources must remain enabled' : 'Enable this Source attachment'}</span>
					</label>

					{isPrimary ? (
						<div className="text-base-content/70 text-xs sm:col-span-2">
							Primary project discovery uses Workspace discovery settings. Attachment overrides apply only to
							non-primary Sources.
						</div>
					) : (
						<>
							<label className="flex items-center gap-3 text-sm">
								<input
									type="checkbox"
									className="toggle toggle-accent toggle-sm"
									checked={recursive}
									disabled={busy}
									onChange={event => {
										setRecursive(event.currentTarget.checked);
									}}
								/>
								<span>Discover recursively when supported</span>
							</label>

							<label className="flex items-center gap-3 text-sm">
								<input
									type="checkbox"
									className="toggle toggle-accent toggle-sm"
									checked={authoritative}
									disabled={busy}
									onChange={event => {
										setAuthoritative(event.currentTarget.checked);
									}}
								/>
								<span>Use as authoritative input</span>
							</label>
						</>
					)}
				</div>

				{!isPrimary ? (
					<div className="mt-3 flex justify-end">
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={busy}
							onClick={() => {
								void save();
							}}
						>
							Save attachment settings
						</button>
					</div>
				) : null}
			</details>
		</div>
	);
}

export interface WorkspaceSourcesModalProps {
	isOpen: boolean;
	onClose: () => void;
	workspace: WorkspaceView;
	onWorkspaceChange: (workspace: WorkspaceView) => void;
	onCatalogInvalidated: () => void;
}

function WorkspaceSourcesModalContent({
	workspace,
	onWorkspaceChange,
	onCatalogInvalidated,
}: Omit<WorkspaceSourcesModalProps, 'isOpen' | 'onClose'>) {
	const { requestClose, unmountingRef } = useModalDialogController();
	const [currentWorkspace, setCurrentWorkspace] = useState(workspace);
	const [pendingAction, setPendingAction] = useState<string | null>(null);
	const [actionError, setActionError] = useState('');
	const [attachmentToDetach, setAttachmentToDetach] = useState<WorkspaceAttachmentView | null>(null);

	const {
		data: sourceCatalog,
		error: sourceCatalogError,
		isLoading: isInitialSourceLoading,
		isRefreshing: isRefreshingSources,
		reloadOrThrow: reloadSourceCatalog,
		setData: setSourceCatalog,
	} = useAsyncResource(loadWorkspaceSourceCatalog, {
		initialData: EMPTY_WORKSPACE_SOURCE_CATALOG,
	});

	const [selectedExistingSourceID, setSelectedExistingSourceID] = useState('');
	const [selectedAttachmentRole, setSelectedAttachmentRole] = useState<WorkspaceAttachmentRole>(
		WorkspaceAttachmentRole.Library
	);
	const [selectedRecursive, setSelectedRecursive] = useState(true);
	const [selectedAuthoritative, setSelectedAuthoritative] = useState(false);
	const [selectedPrimarySourceID, setSelectedPrimarySourceID] = useState('');

	const [newSourceDisplayName, setNewSourceDisplayName] = useState('');
	const [newSourcePath, setNewSourcePath] = useState('');
	const [registerAsPrimary, setRegisterAsPrimary] = useState(false);

	const sources = sourceCatalog.sources;
	const isSourceLoading = isInitialSourceLoading || isRefreshingSources;
	const sourceLoadError = sourceCatalogError
		? getErrorMessage(sourceCatalogError, 'Workspace Sources could not be loaded.')
		: '';

	const sourceByID = useMemo(() => new Map(sources.map(source => [source.id, source] as const)), [sources]);

	const attachedSourceIDs = useMemo(
		() => new Set(currentWorkspace.attachments.map(attachment => attachment.sourceID)),
		[currentWorkspace.attachments]
	);

	const primaryAttachment = useMemo(
		() => currentWorkspace.attachments.find(attachment => attachment.role === WorkspaceAttachmentRole.Primary),
		[currentWorkspace.attachments]
	);

	const attachableSources = useMemo(
		() => sources.filter(source => !attachedSourceIDs.has(source.id)),
		[attachedSourceIDs, sources]
	);

	const primaryCandidates = useMemo(
		() => sources.filter(source => source.kind === FILESYSTEM_SOURCE_KIND && !attachedSourceIDs.has(source.id)),
		[attachedSourceIDs, sources]
	);

	const selectedExistingSource = sourceByID.get(selectedExistingSourceID);
	const selectedPrimarySource = sourceByID.get(selectedPrimarySourceID);

	const attachableSourceItems = useMemo(
		() => Object.fromEntries(attachableSources.map(source => [source.id, { isEnabled: source.enabled }])),
		[attachableSources]
	);

	const primarySourceItems = useMemo(
		() => Object.fromEntries(primaryCandidates.map(source => [source.id, { isEnabled: source.enabled }])),
		[primaryCandidates]
	);

	const applyUpdatedWorkspace = useCallback(
		(updated: WorkspaceView) => {
			setCurrentWorkspace(updated);
			onWorkspaceChange(updated);
			onCatalogInvalidated();
		},
		[onCatalogInvalidated, onWorkspaceChange]
	);

	const refreshSources = useCallback(async () => {
		try {
			await reloadSourceCatalog();
		} catch {
			// `useAsyncResource` publishes the failure through sourceCatalogError.
		}
	}, [reloadSourceCatalog]);

	const runWorkspaceMutation = useCallback(
		async (actionKey: string, mutation: (current: WorkspaceView) => Promise<WorkspaceView>): Promise<boolean> => {
			if (pendingAction) {
				return false;
			}

			setActionError('');
			setPendingAction(actionKey);

			try {
				const updated = await mutation(currentWorkspace);
				if (!unmountingRef.current) {
					applyUpdatedWorkspace(updated);
				}
				return true;
			} catch (error) {
				if (!unmountingRef.current) {
					setActionError(getErrorMessage(error, 'Workspace Source settings could not be changed.'));
				}
				return false;
			} finally {
				if (!unmountingRef.current) {
					setPendingAction(null);
				}
			}
		},
		[applyUpdatedWorkspace, currentWorkspace, pendingAction, unmountingRef]
	);

	const saveAttachment = useCallback(
		async (attachment: WorkspaceAttachmentView, edit: AttachmentEdit) => {
			await runWorkspaceMutation(`attachment:${attachment.sourceID}:save`, current => {
				const latestAttachment = current.attachments.find(item => item.sourceID === attachment.sourceID);
				if (!latestAttachment) {
					throw new Error('The Source attachment no longer belongs to this Workspace.');
				}

				return workspaceManagementAPI.updateWorkspaceAttachment(current.workspace, latestAttachment.sourceID, {
					expectedCollectionRevision: current.revision,
					expectedAttachmentRevision: latestAttachment.revision,
					...edit,
				});
			});
		},
		[runWorkspaceMutation]
	);

	const setPrimarySource = useCallback(
		async (sourceID: string): Promise<boolean> => {
			const source = sourceByID.get(sourceID);
			if (!source) {
				setActionError('Select a registered filesystem Source.');
				return false;
			}
			if (!source.enabled) {
				setActionError('Enable the Source before using it as the project Source.');
				return false;
			}

			return runWorkspaceMutation(`primary:${sourceID}`, current =>
				workspaceManagementAPI.setWorkspacePrimarySource(current.workspace, {
					expectedCollectionRevision: current.revision,
					sourceID,
				})
			);
		},
		[runWorkspaceMutation, sourceByID]
	);

	const clearPrimarySource = useCallback(async (): Promise<boolean> => {
		if (!primaryAttachment) {
			return true;
		}

		return runWorkspaceMutation(`primary:${primaryAttachment.sourceID}:clear`, current =>
			workspaceManagementAPI.setWorkspacePrimarySource(current.workspace, {
				expectedCollectionRevision: current.revision,
				clear: true,
			})
		);
	}, [primaryAttachment, runWorkspaceMutation]);

	const detachAttachment = useCallback(
		async (attachment: WorkspaceAttachmentView): Promise<boolean> => {
			if (attachment.role === WorkspaceAttachmentRole.Primary) {
				return clearPrimarySource();
			}

			return runWorkspaceMutation(`attachment:${attachment.sourceID}:detach`, current => {
				const latestAttachment = current.attachments.find(item => item.sourceID === attachment.sourceID);
				if (!latestAttachment) {
					throw new Error('The Source attachment no longer belongs to this Workspace.');
				}

				return workspaceManagementAPI.detachWorkspaceSource(current.workspace, latestAttachment.sourceID, {
					expectedCollectionRevision: current.revision,
					expectedAttachmentRevision: latestAttachment.revision,
				});
			});
		},
		[clearPrimarySource, runWorkspaceMutation]
	);

	const attachExistingSource = useCallback(async () => {
		if (!selectedExistingSource) {
			setActionError('Select a registered Source to attach.');
			return;
		}
		if (!selectedExistingSource.enabled) {
			setActionError('Enable the Source before attaching it to this Workspace.');
			return;
		}

		const attached = await runWorkspaceMutation(`attachment:${selectedExistingSource.id}:attach`, current =>
			workspaceManagementAPI.attachWorkspaceSource(current.workspace, {
				expectedCollectionRevision: current.revision,
				sourceID: selectedExistingSource.id,
				role: selectedAttachmentRole,
				enabled: true,
				settings: attachmentSettings(selectedRecursive, selectedAuthoritative),
			})
		);

		if (attached && !unmountingRef.current) {
			setSelectedExistingSourceID('');
		}
	}, [
		runWorkspaceMutation,
		selectedAttachmentRole,
		selectedAuthoritative,
		selectedExistingSource,
		selectedRecursive,
		unmountingRef,
	]);

	const toggleSourceEnabled = useCallback(
		async (source: ArtifactSourceSummary) => {
			if (pendingAction) {
				return;
			}

			setActionError('');
			setPendingAction(`source:${source.id}:enabled`);

			try {
				const updated = await workspaceManagementAPI.setWorkspaceSourceEnabled(
					source.id,
					source.revision,
					!source.enabled
				);

				if (!unmountingRef.current) {
					setSourceCatalog(previous => ({
						sources: sortSources(previous.sources.map(item => (item.id === updated.id ? updated : item))),
					}));

					if (attachedSourceIDs.has(source.id)) {
						onCatalogInvalidated();
					}
				}
			} catch (error) {
				if (!unmountingRef.current) {
					setActionError(getErrorMessage(error, 'Workspace Source enablement could not be changed.'));
				}
			} finally {
				if (!unmountingRef.current) {
					setPendingAction(null);
				}
			}
		},
		[attachedSourceIDs, onCatalogInvalidated, pendingAction, setSourceCatalog, unmountingRef]
	);

	const chooseDirectory = useCallback(async () => {
		setActionError('');

		try {
			const selectedPath = await backendAPI.pickDirectoryPath();
			if (!selectedPath || unmountingRef.current) {
				return;
			}

			setNewSourcePath(selectedPath);
			setNewSourceDisplayName(previous => (previous.trim() ? previous : sourceDisplayNameFromPath(selectedPath)));
		} catch (error) {
			if (!unmountingRef.current) {
				setActionError(getErrorMessage(error, 'Could not open the folder picker.'));
			}
		}
	}, [unmountingRef]);

	const registerDirectory: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (pendingAction) {
			return;
		}

		const displayName = newSourceDisplayName.trim();
		const rootPath = newSourcePath.trim();

		if (!displayName) {
			setActionError('Source display name is required.');
			return;
		}
		if (!rootPath) {
			setActionError('Choose or enter a folder path.');
			return;
		}

		setActionError('');
		setPendingAction('source:register');

		void (async () => {
			try {
				const result = await workspaceManagementAPI.registerWorkspaceDirectory(currentWorkspace.workspace, {
					expectedCollectionRevision: currentWorkspace.revision,
					displayName,
					rootPath,
					role: registerAsPrimary ? WorkspaceAttachmentRole.Primary : selectedAttachmentRole,
					settings: registerAsPrimary ? {} : attachmentSettings(selectedRecursive, selectedAuthoritative),
				});

				if (!unmountingRef.current) {
					setSourceCatalog(previous => {
						const hasExisting = previous.sources.some(source => source.id === result.source.id);
						const nextSources = hasExisting
							? previous.sources.map(source => (source.id === result.source.id ? result.source : source))
							: [...previous.sources, result.source];

						return {
							sources: sortSources(nextSources),
						};
					});

					applyUpdatedWorkspace(result.workspace);
					setNewSourceDisplayName('');
					setNewSourcePath('');
					setRegisterAsPrimary(false);
				}
			} catch (error) {
				if (!unmountingRef.current) {
					setActionError(getErrorMessage(error, 'Could not add the folder Source to this Workspace.'));
				}
			} finally {
				if (!unmountingRef.current) {
					setPendingAction(null);
				}
			}
		})();
	};

	const confirmDetach = async () => {
		if (!attachmentToDetach) {
			return;
		}

		const detached = await detachAttachment(attachmentToDetach);
		if (detached && !unmountingRef.current) {
			setAttachmentToDetach(null);
		}
	};

	return (
		<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-5xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title="Manage Workspace Sources"
				description={`Attach registered Sources to ${currentWorkspace.displayName}. Folder Source registration and attachment are one backend operation.`}
				onClose={requestClose}
				closeDisabled={pendingAction !== null}
			/>

			<div className="app-scrollbar-thin min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-6">
				{actionError ? (
					<div className="alert alert-error rounded-2xl text-sm" role="alert">
						<FiAlertCircle size={14} />
						<span>{actionError}</span>
					</div>
				) : null}

				{sourceLoadError ? (
					<div className="alert alert-warning rounded-2xl text-sm" role="alert">
						<FiAlertCircle size={14} />
						<span>{sourceLoadError}</span>
					</div>
				) : null}

				<ModalSection
					title="Attached Sources"
					description="Attachment roles and discovery settings belong to this Workspace. A registered Source can be attached to multiple Workspace Collections in the same Workspace Root."
				>
					<div className="space-y-3">
						{currentWorkspace.attachments.map(attachment => (
							<WorkspaceSourceAttachmentCard
								key={`${attachment.sourceID}:${attachment.revision}`}
								attachment={attachment}
								source={sourceByID.get(attachment.sourceID)}
								busy={pendingAction !== null}
								onSave={saveAttachment}
								onRequestDetach={setAttachmentToDetach}
							/>
						))}

						{currentWorkspace.attachments.length === 0 ? (
							<div className="border-base-content/10 text-base-content/60 rounded-2xl border border-dashed p-4 text-sm">
								No Sources are attached. Add a folder Source below or attach an existing registered Source.
							</div>
						) : null}
					</div>

					{attachmentToDetach ? (
						<div className="alert alert-warning mt-3 flex flex-wrap items-center gap-3 rounded-2xl text-sm">
							<div className="grow">
								<div className="font-semibold">
									{attachmentToDetach.role === WorkspaceAttachmentRole.Primary
										? 'Clear the primary Source?'
										: 'Detach this Source?'}
								</div>
								<div>
									{attachmentToDetach.role === WorkspaceAttachmentRole.Primary
										? 'The Workspace will have no primary project Source. No files are deleted.'
										: 'The Source remains registered and can still be used by other Workspaces.'}
								</div>
							</div>

							<button
								type="button"
								className="btn btn-sm btn-ghost rounded-xl"
								disabled={pendingAction !== null}
								onClick={() => {
									setAttachmentToDetach(null);
								}}
							>
								Cancel
							</button>

							<button
								type="button"
								className="btn btn-sm btn-error rounded-xl"
								disabled={pendingAction !== null}
								onClick={() => {
									void confirmDetach();
								}}
							>
								{attachmentToDetach.role === WorkspaceAttachmentRole.Primary ? 'Clear Primary' : 'Detach'}
							</button>
						</div>
					) : null}
				</ModalSection>

				<ModalSection
					title="Attach an existing Source"
					description="Attach an enabled Source already registered in the Workspace Root."
				>
					<div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<ModalField label="Registered Source" htmlFor="workspace-existing-source">
							<Dropdown<string>
								dropdownItems={attachableSourceItems}
								orderedKeys={attachableSources.map(source => source.id)}
								selectedKey={selectedExistingSourceID}
								onChange={setSelectedExistingSourceID}
								disabled={pendingAction !== null || attachableSources.length === 0}
								placeholderLabel="Select a Source"
								title="Select a Source to attach"
								getDisplayName={key => {
									const source = sourceByID.get(key);
									return source ? sourceLabel(source) : '';
								}}
							/>
						</ModalField>

						<ModalField label="Workspace role" htmlFor="workspace-existing-source-role">
							<Dropdown<WorkspaceAttachmentRole>
								dropdownItems={NON_PRIMARY_ROLE_ITEMS}
								orderedKeys={NON_PRIMARY_ROLES}
								selectedKey={selectedAttachmentRole}
								onChange={setSelectedAttachmentRole}
								disabled={pendingAction !== null}
								title="Select Workspace Source role"
								getDisplayName={attachmentRoleLabel}
							/>
						</ModalField>
					</div>

					<div className="mt-3 flex flex-wrap items-center gap-4">
						<label className="flex items-center gap-2 text-sm">
							<input
								type="checkbox"
								className="checkbox checkbox-sm"
								checked={selectedRecursive}
								disabled={pendingAction !== null}
								onChange={event => {
									setSelectedRecursive(event.currentTarget.checked);
								}}
							/>
							Discover recursively
						</label>

						<label className="flex items-center gap-2 text-sm">
							<input
								type="checkbox"
								className="checkbox checkbox-sm"
								checked={selectedAuthoritative}
								disabled={pendingAction !== null}
								onChange={event => {
									setSelectedAuthoritative(event.currentTarget.checked);
								}}
							/>
							Authoritative attachment
						</label>

						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl sm:ml-auto"
							disabled={pendingAction !== null || !selectedExistingSource || !selectedExistingSource.enabled}
							onClick={() => {
								void attachExistingSource();
							}}
						>
							<FiLink size={14} />
							<span>Attach Source</span>
						</button>
					</div>
				</ModalSection>

				<ModalSection
					title="Primary Source"
					description="A Workspace can have one enabled filesystem Source as its primary project Source."
				>
					<div className="flex flex-col gap-3 sm:flex-row sm:items-end">
						<Dropdown<string>
							dropdownItems={primarySourceItems}
							orderedKeys={primaryCandidates.map(source => source.id)}
							selectedKey={selectedPrimarySourceID}
							onChange={setSelectedPrimarySourceID}
							disabled={pendingAction !== null || primaryCandidates.length === 0}
							placeholderLabel="Select a filesystem Source"
							title="Select the project Source"
							getDisplayName={key => {
								const source = sourceByID.get(key);
								return source ? sourceLabel(source) : '';
							}}
						/>

						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={pendingAction !== null || !selectedPrimarySource || !selectedPrimarySource.enabled}
							onClick={() => {
								if (!selectedPrimarySourceID) {
									return;
								}
								void setPrimarySource(selectedPrimarySourceID).then(updated => {
									if (updated && !unmountingRef.current) {
										setSelectedPrimarySourceID('');
									}
								});
							}}
						>
							<FiFolder size={14} />
							<span>Set Primary</span>
						</button>
					</div>

					{primaryAttachment ? (
						<div className="text-base-content/70 mt-3 text-xs">
							Current primary Source: <span className="font-mono">{primaryAttachment.sourceID}</span>
						</div>
					) : (
						<div className="text-base-content/70 mt-3 text-xs">No primary Source is attached.</div>
					)}
				</ModalSection>

				<ModalSection
					title="Add a folder Source"
					description="The backend creates the filesystem Source and attaches it atomically. Source adapter configuration is not exposed to the frontend."
				>
					<form className="space-y-4" onSubmit={registerDirectory}>
						<ModalField label="Folder" htmlFor="workspace-new-source-path" required>
							<div className="flex flex-col gap-2 sm:flex-row">
								<input
									id="workspace-new-source-path"
									type="text"
									className="input input-sm min-w-0 grow rounded-xl font-mono text-xs"
									value={newSourcePath}
									disabled={pendingAction !== null}
									onChange={event => {
										setNewSourcePath(event.currentTarget.value);
									}}
									placeholder="/path/to/folder"
									spellCheck="false"
									autoComplete="off"
								/>

								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={pendingAction !== null}
									onClick={() => {
										void chooseDirectory();
									}}
								>
									<FiFolder size={14} />
									<span>Choose Folder</span>
								</button>
							</div>
						</ModalField>

						<ModalField label="Display name" htmlFor="workspace-new-source-name" required>
							<input
								id="workspace-new-source-name"
								type="text"
								className="input input-sm w-full rounded-xl"
								value={newSourceDisplayName}
								disabled={pendingAction !== null}
								onChange={event => {
									setNewSourceDisplayName(event.currentTarget.value);
								}}
								autoComplete="off"
							/>
						</ModalField>

						<label className="flex items-center gap-3 text-sm">
							<input
								type="checkbox"
								className="checkbox checkbox-sm"
								checked={registerAsPrimary}
								disabled={pendingAction !== null}
								onChange={event => {
									setRegisterAsPrimary(event.currentTarget.checked);
								}}
							/>
							<span>Use this folder as the primary project Source</span>
						</label>

						{!registerAsPrimary ? (
							<>
								<ModalField label="Workspace role" htmlFor="workspace-new-source-role">
									<Dropdown<WorkspaceAttachmentRole>
										dropdownItems={NON_PRIMARY_ROLE_ITEMS}
										orderedKeys={NON_PRIMARY_ROLES}
										selectedKey={selectedAttachmentRole}
										onChange={setSelectedAttachmentRole}
										disabled={pendingAction !== null}
										title="Select Workspace Source role"
										getDisplayName={attachmentRoleLabel}
									/>
								</ModalField>

								<div className="flex flex-wrap items-center gap-4">
									<label className="flex items-center gap-2 text-sm">
										<input
											type="checkbox"
											className="checkbox checkbox-sm"
											checked={selectedRecursive}
											disabled={pendingAction !== null}
											onChange={event => {
												setSelectedRecursive(event.currentTarget.checked);
											}}
										/>
										Discover recursively
									</label>

									<label className="flex items-center gap-2 text-sm">
										<input
											type="checkbox"
											className="checkbox checkbox-sm"
											checked={selectedAuthoritative}
											disabled={pendingAction !== null}
											onChange={event => {
												setSelectedAuthoritative(event.currentTarget.checked);
											}}
										/>
										Authoritative attachment
									</label>
								</div>
							</>
						) : (
							<div className="text-base-content/70 text-xs">
								Primary Sources always remain enabled and use Workspace-level discovery settings.
							</div>
						)}

						<div className="flex justify-end">
							<button type="submit" className="btn btn-sm btn-ghost rounded-xl" disabled={pendingAction !== null}>
								<FiPlus size={14} />
								<span>{pendingAction === 'source:register' ? 'Adding...' : 'Add Folder Source'}</span>
							</button>
						</div>
					</form>
				</ModalSection>

				<ModalSection
					title="Source availability"
					description="Source enablement is shared by Workspace Collections in this Workspace Root."
				>
					<div className="max-h-56 space-y-2 overflow-y-auto">
						{isSourceLoading && sources.length === 0 ? (
							<div className="flex items-center gap-2 py-3 text-sm">
								<span className="loading loading-spinner loading-sm" />
								<span>Loading Workspace Sources...</span>
							</div>
						) : null}

						{sources.map(source => {
							const attached = attachedSourceIDs.has(source.id);
							const hasEnabledAttachment = currentWorkspace.attachments.some(
								attachment => attachment.sourceID === source.id && attachment.enabled
							);

							return (
								<div
									key={`${source.id}:${source.revision}`}
									className="border-base-content/10 flex items-center gap-3 rounded-xl border p-2"
								>
									<FiFolder size={14} className="shrink-0" />

									<div className="min-w-0 grow">
										<div className="truncate text-sm">{source.displayName}</div>
										<div className="text-base-content/60 truncate font-mono text-xs">
											{source.kind} · {source.id}
										</div>
									</div>

									{attached ? (
										<span className="text-success inline-flex items-center" title="Attached to this Workspace">
											<FiCheck size={14} aria-hidden="true" />
											<span className="sr-only">Attached to this Workspace</span>
										</span>
									) : null}

									<button
										type="button"
										className="btn btn-xs btn-ghost rounded-lg"
										disabled={pendingAction !== null || (source.enabled && hasEnabledAttachment)}
										title={
											source.enabled && hasEnabledAttachment
												? 'Disable enabled Workspace attachments before disabling this Source.'
												: undefined
										}
										onClick={() => {
											void toggleSourceEnabled(source);
										}}
									>
										{source.enabled ? 'Disable' : 'Enable'}
									</button>
								</div>
							);
						})}

						{!isSourceLoading && sources.length === 0 && !sourceLoadError ? (
							<div className="text-base-content/60 text-sm">No Workspace Sources are registered.</div>
						) : null}
					</div>

					<div className="mt-3 flex justify-end">
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={isSourceLoading || pendingAction !== null}
							onClick={() => {
								void refreshSources();
							}}
						>
							<FiRefreshCw size={14} />
							<span>Reload Sources</span>
						</button>
					</div>
				</ModalSection>
			</div>

			<ModalActions className="-mx-4 -mb-4 sm:-mx-6 sm:-mb-6">
				<button
					type="button"
					className="btn bg-base-300 rounded-xl"
					disabled={pendingAction !== null}
					onClick={() => {
						requestClose();
					}}
				>
					Close
				</button>
			</ModalActions>
		</div>
	);
}

export function WorkspaceSourcesModal(props: WorkspaceSourcesModalProps) {
	if (!props.isOpen) {
		return null;
	}

	return (
		<ModalDialog isOpen={props.isOpen} onClose={props.onClose} blockCancel>
			<WorkspaceSourcesModalContent
				key={`${workspaceRefKey(props.workspace.workspace)}:${props.workspace.revision}`}
				workspace={props.workspace}
				onWorkspaceChange={props.onWorkspaceChange}
				onCatalogInvalidated={props.onCatalogInvalidated}
			/>
		</ModalDialog>
	);
}
