import type { SubmitEventHandler } from 'react';
import { useCallback, useMemo, useState } from 'react';

import { FiAlertCircle, FiCheck, FiFolder, FiLink, FiPlus, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import type { ArtifactSourceKind, ArtifactSourceSummary } from '@/spec/artifact';
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

import { artifactStoreAPI, workspaceAPI } from '@/apis/baseapi';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';

import { workspaceRefKey } from '@/workspaces/lib/workspace_api_utils';
import { getErrorMessage } from '@/workspaces/lib/workspace_utils';

type AttachmentEdit = Pick<UpdateWorkspaceAttachmentBody, 'role' | 'enabled' | 'settings'>;

const NON_PRIMARY_ROLES: WorkspaceAttachmentRole[] = [
	WorkspaceAttachmentRole.BuiltIn,
	WorkspaceAttachmentRole.Library,
	WorkspaceAttachmentRole.AttachedPackage,
	WorkspaceAttachmentRole.Overlay,
];

function attachmentRoleLabel(role: WorkspaceAttachmentRole): string {
	switch (role) {
		case WorkspaceAttachmentRole.Primary:
			return 'Primary project';
		case WorkspaceAttachmentRole.BuiltIn:
			return 'Built-in';
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

function attachmentSettings(recursive: boolean, authoritative: boolean): WorkspaceAttachmentSettings {
	return {
		recursive,
		authoritative,
	};
}

function sortSources(sources: ArtifactSourceSummary[]): ArtifactSourceSummary[] {
	return [...sources].toSorted((left, right) => {
		const nameOrder = left.displayName.localeCompare(right.displayName, undefined, {
			sensitivity: 'base',
		});
		return nameOrder !== 0 ? nameOrder : left.id.localeCompare(right.id);
	});
}

interface WorkspaceSourceCatalogData {
	sources: ArtifactSourceSummary[];
	sourceKinds: ArtifactSourceKind[];
	sourceLoadError?: string;
	sourceKindsError?: string;
}

const EMPTY_WORKSPACE_SOURCE_CATALOG: WorkspaceSourceCatalogData = {
	sources: [],
	sourceKinds: [],
};

async function loadWorkspaceSourceCatalog(rootID: string, signal: AbortSignal): Promise<WorkspaceSourceCatalogData> {
	const [sourcesResult, kindsResult] = await Promise.allSettled([
		artifactStoreAPI.listArtifactSources(rootID),
		artifactStoreAPI.listArtifactSourceKinds(),
	]);
	throwIfAborted(signal);

	return {
		sources: sourcesResult.status === 'fulfilled' ? sortSources(sourcesResult.value) : [],
		sourceKinds: kindsResult.status === 'fulfilled' ? kindsResult.value : [],
		sourceLoadError:
			sourcesResult.status === 'rejected'
				? getErrorMessage(sourcesResult.reason, 'Artifact Store Sources could not be loaded.')
				: undefined,
		sourceKindsError:
			kindsResult.status === 'rejected'
				? getErrorMessage(kindsResult.reason, 'Artifact Source kinds could not be loaded.')
				: undefined,
	};
}

function parseSourceConfig(raw: string): string {
	let parsed: unknown;

	try {
		parsed = JSON.parse(raw);
	} catch {
		throw new Error('Source configuration must be valid JSON.');
	}

	if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
		throw new Error('Source configuration must be a JSON object.');
	}

	const normalized = JSON.stringify(parsed);
	if (!normalized) {
		throw new Error('Source configuration could not be normalized.');
	}

	return normalized;
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
	const [recursive, setRecursive] = useState(Boolean(attachment.settings?.recursive));
	const [authoritative, setAuthoritative] = useState(Boolean(attachment.settings?.authoritative));
	const title = attachment.path ?? attachment.sourceDisplayName ?? source?.displayName ?? 'Attached Source';
	const subtitle = source ? `${source.kind} · ${source.id}` : (attachment.sourceKind ?? attachment.sourceID);

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
							<select
								className="select select-sm w-full rounded-xl"
								value={role}
								disabled={busy}
								onChange={event => {
									setRole(event.currentTarget.value as WorkspaceAttachmentRole);
								}}
							>
								{NON_PRIMARY_ROLES.map(option => (
									<option key={option} value={option}>
										{attachmentRoleLabel(option)}
									</option>
								))}
							</select>
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
						<span>Discover recursively when the Source supports it</span>
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
						<span>Use this attachment as authoritative input</span>
					</label>
				</div>

				<div className="mt-3 flex justify-end">
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						disabled={busy}
						onClick={() => {
							void onSave(attachment, {
								role: isPrimary ? WorkspaceAttachmentRole.Primary : role,
								enabled: isPrimary ? true : enabled,
								settings: attachmentSettings(recursive, authoritative),
							});
						}}
					>
						Save attachment settings
					</button>
				</div>
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

	const loadSourceCatalog = useCallback(
		(signal: AbortSignal) => loadWorkspaceSourceCatalog(currentWorkspace.workspace.rootID, signal),
		[currentWorkspace.workspace.rootID]
	);
	const {
		data: sourceCatalog,
		error: sourceCatalogError,
		isLoading: isInitialSourceLoading,
		isRefreshing: isRefreshingSources,
		reloadOrThrow: reloadSourceCatalog,
		setData: setSourceCatalog,
	} = useAsyncResource(loadSourceCatalog, {
		initialData: EMPTY_WORKSPACE_SOURCE_CATALOG,
	});
	const sources = sourceCatalog.sources;
	const sourceKinds = sourceCatalog.sourceKinds;
	const isSourceLoading = isInitialSourceLoading || isRefreshingSources;
	const sourceLoadError =
		sourceCatalog.sourceLoadError ??
		(sourceCatalogError ? getErrorMessage(sourceCatalogError, 'Artifact Store Sources could not be loaded.') : '');
	const sourceKindsError =
		sourceCatalog.sourceKindsError ??
		(sourceCatalogError ? getErrorMessage(sourceCatalogError, 'Artifact Source kinds could not be loaded.') : '');

	const refreshSources = useCallback(async () => {
		try {
			await reloadSourceCatalog();
		} catch {
			// `useAsyncResource` exposes the resource failure in sourceLoadError.
		}
	}, [reloadSourceCatalog]);

	const [selectedExistingSourceID, setSelectedExistingSourceID] = useState('');
	const [selectedAttachmentRole, setSelectedAttachmentRole] = useState<WorkspaceAttachmentRole>(
		WorkspaceAttachmentRole.Library
	);
	const [selectedRecursive, setSelectedRecursive] = useState(true);
	const [selectedAuthoritative, setSelectedAuthoritative] = useState(false);
	const [selectedPrimarySourceID, setSelectedPrimarySourceID] = useState('');

	const [newSourceKind, setNewSourceKind] = useState('');
	const [newSourceDisplayName, setNewSourceDisplayName] = useState('');
	const [newSourceConfig, setNewSourceConfig] = useState('{}');
	const [newSourceEnabled, setNewSourceEnabled] = useState(true);
	const [attachNewSource, setAttachNewSource] = useState(true);
	const [attachNewSourceAsPrimary, setAttachNewSourceAsPrimary] = useState(false);

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
		() => sources.filter(source => source.id !== primaryAttachment?.sourceID),
		[primaryAttachment?.sourceID, sources]
	);
	const selectedExistingSource = sourceByID.get(selectedExistingSourceID);
	const selectedPrimarySource = sourceByID.get(selectedPrimarySourceID);
	const effectiveNewSourceKind = newSourceKind || sourceKinds[0] || '';

	const applyUpdatedWorkspace = useCallback(
		(updated: WorkspaceView) => {
			setCurrentWorkspace(updated);
			onWorkspaceChange(updated);
			onCatalogInvalidated();
		},
		[onCatalogInvalidated, onWorkspaceChange]
	);

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

				return workspaceAPI.updateWorkspaceAttachment(current.workspace, latestAttachment.sourceID, {
					expectedCollectionRevision: current.revision,
					expectedAttachmentRevision: latestAttachment.revision,
					...edit,
				});
			});
		},
		[runWorkspaceMutation]
	);

	const setPrimarySource = useCallback(
		async (sourceID: string) => {
			const source = sourceByID.get(sourceID);
			if (source && !source.enabled) {
				setActionError('Enable the Artifact Store Source before using it as a Workspace primary Source.');
				return false;
			}

			return runWorkspaceMutation(`primary:${sourceID}`, current => {
				const currentPrimary = current.attachments.find(
					attachment => attachment.role === WorkspaceAttachmentRole.Primary
				);

				if (currentPrimary?.sourceID === sourceID) {
					return Promise.resolve(current);
				}

				return workspaceAPI.setWorkspacePrimarySource(current.workspace, {
					expectedCollectionRevision: current.revision,
					...(currentPrimary
						? {
								previousSourceID: currentPrimary.sourceID,
								expectedPreviousAttachmentRevision: currentPrimary.revision,
							}
						: {}),
					sourceID,
				});
			});
		},
		[runWorkspaceMutation, sourceByID]
	);

	const clearPrimarySource = useCallback(async (): Promise<boolean> => {
		if (!primaryAttachment) {
			return true;
		}

		return runWorkspaceMutation(`primary:${primaryAttachment.sourceID}:clear`, current => {
			const currentPrimary = current.attachments.find(
				attachment => attachment.role === WorkspaceAttachmentRole.Primary
			);
			if (!currentPrimary) {
				return Promise.resolve(current);
			}

			return workspaceAPI.setWorkspacePrimarySource(current.workspace, {
				expectedCollectionRevision: current.revision,
				previousSourceID: currentPrimary.sourceID,
				expectedPreviousAttachmentRevision: currentPrimary.revision,
				clear: true,
			});
		});
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

				return workspaceAPI.detachWorkspaceSource(
					current.workspace,
					latestAttachment.sourceID,
					current.revision,
					latestAttachment.revision
				);
			});
		},
		[clearPrimarySource, runWorkspaceMutation]
	);

	const attachExistingSource = useCallback(async () => {
		if (!selectedExistingSource) {
			setActionError('Select an Artifact Store Source to attach.');
			return;
		}
		if (!selectedExistingSource.enabled) {
			setActionError('Enable the Artifact Store Source before attaching it to this Workspace.');
			return;
		}

		const attached = await runWorkspaceMutation(`attachment:${selectedExistingSource.id}:attach`, current =>
			workspaceAPI.attachWorkspaceSource(current.workspace, {
				expectedCollectionRevision: current.revision,
				sourceID: selectedExistingSource.id,
				role: selectedAttachmentRole,
				enabled: true,
				settings: attachmentSettings(selectedRecursive, selectedAuthoritative),
			})
		);

		if (attached) {
			setSelectedExistingSourceID('');
		}
	}, [runWorkspaceMutation, selectedAttachmentRole, selectedAuthoritative, selectedExistingSource, selectedRecursive]);

	const toggleSourceEnabled = useCallback(
		async (source: ArtifactSourceSummary) => {
			if (pendingAction) {
				return;
			}

			setActionError('');
			setPendingAction(`source:${source.id}:enabled`);
			try {
				const updated = await artifactStoreAPI.updateArtifactSource(currentWorkspace.workspace.rootID, source.id, {
					expectedRevision: source.revision,
					displayName: source.displayName,
					enabled: !source.enabled,
				});

				if (!unmountingRef.current) {
					setSourceCatalog(previous => ({
						...previous,
						sources: sortSources(previous.sources.map(item => (item.id === updated.id ? updated : item))),
						sourceLoadError: undefined,
					}));
					if (attachedSourceIDs.has(source.id)) {
						onCatalogInvalidated();
					}
				}
			} catch (error) {
				if (!unmountingRef.current) {
					setActionError(getErrorMessage(error, 'Artifact Store Source enablement could not be changed.'));
				}
			} finally {
				if (!unmountingRef.current) {
					setPendingAction(null);
				}
			}
		},
		[
			attachedSourceIDs,
			setSourceCatalog,
			currentWorkspace.workspace.rootID,
			onCatalogInvalidated,
			pendingAction,
			unmountingRef,
		]
	);

	const registerSource: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (pendingAction) {
			return;
		}

		let config: string;
		try {
			config = parseSourceConfig(newSourceConfig);
		} catch (error) {
			setActionError(getErrorMessage(error, 'Source configuration is invalid.'));
			return;
		}

		const displayName = newSourceDisplayName.trim();
		if (!displayName) {
			setActionError('Source display name is required.');
			return;
		}
		if (!effectiveNewSourceKind) {
			setActionError('Choose or enter an Artifact Source kind.');
			return;
		}
		if (attachNewSourceAsPrimary && !newSourceEnabled) {
			setActionError('A primary Workspace Source must be enabled.');
			return;
		}
		if (attachNewSource && !newSourceEnabled) {
			setActionError('Enable the Artifact Store Source before attaching it to this Workspace.');
			return;
		}

		setActionError('');
		setPendingAction('source:create');

		void (async () => {
			let created: ArtifactSourceSummary | undefined;

			try {
				created = await artifactStoreAPI.createArtifactSource(currentWorkspace.workspace.rootID, {
					kind: effectiveNewSourceKind,
					displayName,
					enabled: newSourceEnabled,
					config,
				});

				if (!unmountingRef.current) {
					setSourceCatalog(previous => ({
						...previous,
						sources: sortSources([...previous.sources, created as ArtifactSourceSummary]),
						sourceLoadError: undefined,
					}));
				}

				if (attachNewSource) {
					let updated: WorkspaceView;

					if (attachNewSourceAsPrimary) {
						const currentPrimary = currentWorkspace.attachments.find(
							attachment => attachment.role === WorkspaceAttachmentRole.Primary
						);
						updated = await workspaceAPI.setWorkspacePrimarySource(currentWorkspace.workspace, {
							expectedCollectionRevision: currentWorkspace.revision,
							...(currentPrimary
								? {
										previousSourceID: currentPrimary.sourceID,
										expectedPreviousAttachmentRevision: currentPrimary.revision,
									}
								: {}),
							sourceID: created.id,
						});
					} else {
						updated = await workspaceAPI.attachWorkspaceSource(currentWorkspace.workspace, {
							expectedCollectionRevision: currentWorkspace.revision,
							sourceID: created.id,
							role: selectedAttachmentRole,
							enabled: true,
							settings: attachmentSettings(selectedRecursive, selectedAuthoritative),
						});
					}

					if (!unmountingRef.current) {
						applyUpdatedWorkspace(updated);
					}
				}

				if (!unmountingRef.current) {
					setNewSourceDisplayName('');
					setNewSourceConfig('{}');
					setNewSourceKind(sourceKinds[0] ?? '');
					setAttachNewSourceAsPrimary(false);
				}
			} catch (error) {
				if (!unmountingRef.current) {
					const failure = getErrorMessage(error, 'Artifact Store Source could not be created.');
					setActionError(
						created ? `Source "${created.displayName}" was created, but it could not be attached. ${failure}` : failure
					);
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
		if (detached) {
			setAttachmentToDetach(null);
		}
	};

	return (
		<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-5xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title="Manage Workspace Sources"
				description={`Attach root-scoped Artifact Store Sources to ${currentWorkspace.displayName}. Source configuration is private and never stored in Workspace data.`}
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
					description="Attachment roles and settings belong to this Workspace Collection. The same Source can participate in another Collection in the same Root."
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
								No Sources are attached. This is an empty Workspace Collection until a Source is attached.
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
										? 'The Workspace will have no primary project Source. Other attachments, source files, and the Artifact Store Source are not deleted.'
										: 'The Source remains available to other Collections and is not deleted.'}
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
					description="Attach an enabled Source already registered in this Workspace Root. Primary Source changes use the separate typed primary-source operation."
				>
					<div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<label className="space-y-1">
							<span className="text-xs font-medium">Artifact Store Source</span>
							<select
								className="select select-sm w-full rounded-xl"
								value={selectedExistingSourceID}
								disabled={pendingAction !== null || attachableSources.length === 0}
								onChange={event => {
									setSelectedExistingSourceID(event.currentTarget.value);
								}}
							>
								<option value="">Select a Source</option>
								{attachableSources.map(source => (
									<option key={source.id} value={source.id}>
										{sourceLabel(source)}
										{source.enabled ? '' : ' [disabled]'}
									</option>
								))}
							</select>
						</label>

						<label className="space-y-1">
							<span className="text-xs font-medium">Workspace role</span>
							<select
								className="select select-sm w-full rounded-xl"
								value={selectedAttachmentRole}
								disabled={pendingAction !== null}
								onChange={event => {
									setSelectedAttachmentRole(event.currentTarget.value as WorkspaceAttachmentRole);
								}}
							>
								{NON_PRIMARY_ROLES.map(role => (
									<option key={role} value={role}>
										{attachmentRoleLabel(role)}
									</option>
								))}
							</select>
						</label>
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
					description="A filesystem Workspace has one enabled primary filesystem Source. The backend validates Source eligibility and preserves Workspace identity when the primary Source changes."
				>
					<div className="flex flex-col gap-3 sm:flex-row sm:items-end">
						<label className="grow space-y-1">
							<span className="text-xs font-medium">Replace or set primary Source</span>
							<select
								className="select select-sm w-full rounded-xl"
								value={selectedPrimarySourceID}
								disabled={pendingAction !== null || primaryCandidates.length === 0}
								onChange={event => {
									setSelectedPrimarySourceID(event.currentTarget.value);
								}}
							>
								<option value="">Select a Source</option>
								{primaryCandidates.map(source => (
									<option key={source.id} value={source.id}>
										{sourceLabel(source)}
										{source.enabled ? '' : ' [disabled]'}
									</option>
								))}
							</select>
						</label>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={pendingAction !== null || !selectedPrimarySource || !selectedPrimarySource.enabled}
							onClick={() => {
								if (!selectedPrimarySourceID) {
									return;
								}
								void setPrimarySource(selectedPrimarySourceID).then(updated => {
									if (updated) {
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

				<details className="border-base-content/10 bg-base-100 rounded-2xl border">
					<summary className="cursor-pointer p-4 text-sm font-semibold">Register a new Artifact Store Source</summary>
					<form className="border-base-content/10 space-y-4 border-t p-4" onSubmit={registerSource}>
						<div className="text-base-content/70 text-xs">
							Source configuration is sent only to Artifact Store and is write-only. It is never included in Workspace
							views, Workspace selections, portable definitions, or conversation state.
						</div>

						{sourceKindsError ? (
							<div className="alert alert-warning rounded-xl text-xs">
								<FiAlertCircle size={13} />
								<span>{sourceKindsError}</span>
							</div>
						) : null}

						<div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
							<label className="space-y-1">
								<span className="text-xs font-medium">Source kind</span>
								{sourceKinds.length > 0 ? (
									<select
										className="select select-sm w-full rounded-xl"
										value={effectiveNewSourceKind}
										disabled={pendingAction !== null}
										onChange={event => {
											setNewSourceKind(event.currentTarget.value);
										}}
									>
										{sourceKinds.map(kind => (
											<option key={kind} value={kind}>
												{kind}
											</option>
										))}
									</select>
								) : (
									<input
										type="text"
										className="input input-sm w-full rounded-xl"
										value={newSourceKind}
										disabled={pendingAction !== null}
										onChange={event => {
											setNewSourceKind(event.currentTarget.value);
										}}
										placeholder="Source kind"
										spellCheck="false"
									/>
								)}
							</label>

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
						</div>

						<ModalField
							label="Private Source configuration JSON"
							htmlFor="workspace-new-source-config"
							hint="The required shape depends on the selected Source adapter."
						>
							<textarea
								id="workspace-new-source-config"
								className="textarea min-h-28 w-full rounded-xl font-mono text-xs"
								value={newSourceConfig}
								disabled={pendingAction !== null}
								onChange={event => {
									setNewSourceConfig(event.currentTarget.value);
								}}
								spellCheck="false"
							/>
						</ModalField>

						<div className="flex flex-wrap items-center gap-4 text-sm">
							<label className="flex items-center gap-2">
								<input
									type="checkbox"
									className="checkbox checkbox-sm"
									checked={newSourceEnabled}
									disabled={pendingAction !== null}
									onChange={event => {
										setNewSourceEnabled(event.currentTarget.checked);
										if (!event.currentTarget.checked) {
											setAttachNewSource(false);
											setAttachNewSourceAsPrimary(false);
										}
									}}
								/>
								Enable Source after registration
							</label>
							<label className="flex items-center gap-2">
								<input
									type="checkbox"
									className="checkbox checkbox-sm"
									checked={attachNewSource}
									disabled={pendingAction !== null}
									onChange={event => {
										setAttachNewSource(event.currentTarget.checked);
										if (!event.currentTarget.checked) {
											setAttachNewSourceAsPrimary(false);
										}
									}}
								/>
								Attach to this Workspace
							</label>
							<label className="flex items-center gap-2">
								<input
									type="checkbox"
									className="checkbox checkbox-sm"
									checked={attachNewSourceAsPrimary}
									disabled={pendingAction !== null || !attachNewSource}
									onChange={event => {
										setAttachNewSourceAsPrimary(event.currentTarget.checked);
										if (event.currentTarget.checked) {
											setAttachNewSource(true);
											setNewSourceEnabled(true);
										}
									}}
								/>
								Use as primary Source
							</label>
						</div>

						<div className="flex justify-end">
							<button type="submit" className="btn btn-sm btn-ghost rounded-xl" disabled={pendingAction !== null}>
								<FiPlus size={14} />
								<span>{pendingAction === 'source:create' ? 'Registering...' : 'Register Source'}</span>
							</button>
						</div>
					</form>
				</details>

				<ModalSection
					title="Root Source availability"
					description="Source enablement is shared by every Collection in this Root. Changing it can make attached catalogs stale."
				>
					<div className="max-h-56 space-y-2 overflow-y-auto">
						{isSourceLoading && sources.length === 0 ? (
							<div className="flex items-center gap-2 py-3 text-sm">
								<span className="loading loading-spinner loading-sm" />
								<span>Loading Artifact Store Sources...</span>
							</div>
						) : null}

						{sources.map(source => (
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
								{attachedSourceIDs.has(source.id) ? (
									<span className="text-success inline-flex items-center" title="Attached to this Workspace">
										<FiCheck size={14} aria-hidden="true" />
										<span className="sr-only">Attached to this Workspace</span>
									</span>
								) : null}
								<button
									type="button"
									className="btn btn-xs btn-ghost rounded-lg"
									disabled={pendingAction !== null}
									onClick={() => {
										void toggleSourceEnabled(source);
									}}
								>
									{source.enabled ? 'Disable' : 'Enable'}
								</button>
							</div>
						))}

						{!isSourceLoading && sources.length === 0 && !sourceLoadError ? (
							<div className="text-base-content/60 text-sm">No Artifact Store Sources are registered in this Root.</div>
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
