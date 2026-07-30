import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import {
	FiChevronDown,
	FiChevronUp,
	FiEdit2,
	FiEye,
	FiFileText,
	FiPlus,
	FiRefreshCw,
	FiSettings,
	FiTrash2,
} from 'react-icons/fi';

import { ArtifactOccurrenceState } from '@/spec/artifact';
import type {
	UpdateWorkspaceBody,
	WorkspaceArtifactView,
	WorkspaceContextView,
	WorkspaceOccurrenceView,
	WorkspaceSkillView,
	WorkspaceSuppressionView,
	WorkspaceView,
} from '@/spec/workspace';
import { WorkspaceMode } from '@/spec/workspace';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { workspaceAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalConfirmDialog } from '@/components/modal/modal_confirm_dialog';

import { artifactRefKey, workspaceRefKey } from '@/workspaces/lib/workspace_api_utils';
import type { WorkspaceCatalogData } from '@/workspaces/lib/workspace_utils';
import {
	collectWorkspaceDiagnostics,
	getArtifactKindLabel,
	getArtifactStateTone,
	getErrorMessage,
	getOccurrenceStateTone,
	getWorkspaceArtifacts,
	normalizeWorkspaceCatalog,
	removeWorkspaceArtifact,
	replaceWorkspaceArtifact,
	WORKSPACE_CONTEXT_ARTIFACT_KIND,
	WORKSPACE_SKILL_ARTIFACT_KIND,
	workspaceArtifactMatchesSearch,
} from '@/workspaces/lib/workspace_utils';
import { WorkspaceArtifactBindingModal } from '@/workspaces/workspace_artifact_binding_modal';
import { WorkspaceContextPreview } from '@/workspaces/workspace_context_preview';
import { WorkspaceDiagnostics } from '@/workspaces/workspace_diagnostics';
import { WorkspaceResourceDetailsModal } from '@/workspaces/workspace_resource_details_modal';
import type { WorkspaceSetupSubmission } from '@/workspaces/workspace_setup_modal';
import { WorkspaceSetupModal } from '@/workspaces/workspace_setup_modal';
import { WorkspaceSourcesModal } from '@/workspaces/workspace_sources_modal';

type WorkspaceTab = 'records' | 'observations' | 'contexts' | 'skills' | 'sources' | 'suppressions' | 'diagnostics';

interface WorkspaceCardProps {
	workspace: WorkspaceView;
	existingDisplayNames: readonly string[];
	onWorkspaceChange: (workspace: WorkspaceView) => void;
	onUpdateWorkspace: (payload: UpdateWorkspaceBody) => Promise<WorkspaceView>;
	onRequestDelete: (workspace: WorkspaceView) => void;
}

interface WorkspaceCatalogState {
	workspaceVersion: string;
	data: WorkspaceCatalogData | null;
	error: unknown;
	isLoading: boolean;
}

function workspaceVersionFor(workspace: WorkspaceView): string {
	return `${workspaceRefKey(workspace.workspace)}:${workspace.revision}`;
}

function workspaceSuppressionKey(suppression: WorkspaceSuppressionView): string {
	const { binding } = suppression;
	return `${binding.sourceID}:${binding.locator}:${binding.subresourceLocator ?? ''}:${binding.expectedKind}`;
}

function workspaceOccurrenceKey(occurrence: WorkspaceOccurrenceView): string {
	return `${occurrence.sourceID}:${occurrence.locator}:${occurrence.subresourceLocator ?? ''}:${occurrence.kind ?? ''}`;
}

function workspaceOccurrencePendingKey(occurrence: WorkspaceOccurrenceView): string {
	return `occurrence:${workspaceOccurrenceKey(occurrence)}:adopt`;
}

async function loadWorkspaceCatalogData(workspace: WorkspaceView): Promise<WorkspaceCatalogData> {
	const workspaceRef = workspace.workspace;
	const catalog = normalizeWorkspaceCatalog(await workspaceAPI.getWorkspaceCatalog(workspaceRef));
	const [contextResult, skillResult, suppressionResult] = await Promise.allSettled([
		workspaceAPI.listWorkspaceContexts(workspaceRef),
		workspaceAPI.listWorkspaceSkills(workspaceRef),
		workspaceAPI.listWorkspaceSuppressions(workspaceRef),
	]);

	return {
		catalog,
		contexts: contextResult.status === 'fulfilled' ? contextResult.value : [],
		skills: skillResult.status === 'fulfilled' ? skillResult.value : [],
		suppressions: suppressionResult.status === 'fulfilled' ? suppressionResult.value : [],
		contextLoadError:
			contextResult.status === 'rejected'
				? getErrorMessage(contextResult.reason, 'Workspace contexts could not be loaded.')
				: undefined,
		skillLoadError:
			skillResult.status === 'rejected'
				? getErrorMessage(skillResult.reason, 'Workspace skills could not be loaded.')
				: undefined,
		suppressionLoadError:
			suppressionResult.status === 'rejected'
				? getErrorMessage(suppressionResult.reason, 'Workspace suppressions could not be loaded.')
				: undefined,
	};
}

function RecordControls({
	workspace,
	record,
	isPending,
	onToggleEnabled,
	onSetRuntimeDisabled,
	onView,
	onDelete,
}: {
	workspace: WorkspaceView;
	record: WorkspaceArtifactView;
	isPending: (key: string) => boolean;
	onToggleEnabled: (artifact: WorkspaceArtifactView, enabled: boolean) => void;
	onSetRuntimeDisabled: (artifact: WorkspaceArtifactView, disabled: boolean) => void;
	onView: (artifact: WorkspaceArtifactView) => void;
	onDelete: (artifact: WorkspaceArtifactView) => void;
}) {
	const artifactID = record.artifact.artifactID;
	const runtimeRelevant =
		record.kind === WORKSPACE_CONTEXT_ARTIFACT_KIND || record.kind === WORKSPACE_SKILL_ARTIFACT_KIND;

	return (
		<ActionRow
			leading={
				<div className="flex gap-8">
					<EnabledControl
						id={`workspace-artifact-${workspace.workspace.collectionID}-${artifactID}`}
						checked={record.enabled}
						onChange={enabled => {
							onToggleEnabled(record, enabled);
						}}
						disabled={!workspace.enabled}
						busy={isPending(`${artifactID}:enabled`)}
						title={!workspace.enabled ? 'Enable the workspace first.' : undefined}
					/>
					{runtimeRelevant ? (
						<EnabledControl
							id={`workspace-runtime-${workspace.workspace.collectionID}-${artifactID}`}
							label="Use in conversations"
							checked={!record.runtimeDisabled}
							onChange={allowed => {
								onSetRuntimeDisabled(record, !allowed);
							}}
							disabled={!workspace.enabled}
							busy={isPending(`${artifactID}:runtime`)}
							title="Allows this discovered item to be used when the workspace is selected for a conversation."
						/>
					) : null}
				</div>
			}
		>
			<button
				type="button"
				className="btn btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onView(record);
				}}
			>
				<FiEye size={14} />
				<span>Inspect</span>
			</button>

			<button
				type="button"
				className="btn btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onDelete(record);
				}}
				disabled={isPending(`${artifactID}:remove`)}
			>
				<FiTrash2 size={14} />
				<span>Remove Artifact</span>
			</button>
		</ActionRow>
	);
}

export function WorkspaceCard({
	workspace,
	existingDisplayNames,
	onWorkspaceChange,
	onUpdateWorkspace,
	onRequestDelete,
}: WorkspaceCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const [activeTab, setActiveTab] = useState<WorkspaceTab>('contexts');
	const [catalogState, setCatalogState] = useState<WorkspaceCatalogState | null>(null);
	const [recordSearch, setRecordSearch] = useState('');
	const [refreshSummary, setRefreshSummary] = useState('');
	const [alertMessage, setAlertMessage] = useState('');

	const [isEditOpen, setIsEditOpen] = useState(false);
	const [recordToInspect, setRecordToInspect] = useState<WorkspaceArtifactView | null>(null);
	const [recordToDelete, setRecordToDelete] = useState<WorkspaceArtifactView | null>(null);
	const [suppressRemovedBinding, setSuppressRemovedBinding] = useState(true);
	const [isContextPreviewOpen, setIsContextPreviewOpen] = useState(false);
	const [isSourcesOpen, setIsSourcesOpen] = useState(false);
	const [isArtifactBindingOpen, setIsArtifactBindingOpen] = useState(false);

	const requestIDRef = useRef(0);
	const mountedRef = useRef(true);
	const { isPending, runAction } = usePendingActions();
	const workspaceVersion = workspaceVersionFor(workspace);
	const currentCatalogState = catalogState?.workspaceVersion === workspaceVersion ? catalogState : null;
	const catalogData = currentCatalogState?.data ?? null;
	const catalogError = currentCatalogState?.error ?? null;
	const isCatalogLoading = currentCatalogState?.isLoading ?? false;

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
			requestIDRef.current += 1;
		};
	}, []);

	const sourceLabelFor = useCallback(
		(sourceID: string) => {
			const attachment = workspace.attachments.find(item => item.sourceID === sourceID);
			return attachment?.path ?? attachment?.sourceDisplayName ?? sourceID;
		},
		[workspace.attachments]
	);

	const reloadCatalog = useCallback(async () => {
		const requestID = requestIDRef.current + 1;
		requestIDRef.current = requestID;
		const requestedWorkspace = workspace;
		const requestedWorkspaceVersion = workspaceVersion;

		setCatalogState(previous => ({
			workspaceVersion: requestedWorkspaceVersion,
			data: previous?.workspaceVersion === requestedWorkspaceVersion ? previous.data : null,
			error: null,
			isLoading: true,
		}));

		try {
			const next = await loadWorkspaceCatalogData(requestedWorkspace);
			if (mountedRef.current && requestIDRef.current === requestID) {
				setCatalogState({
					workspaceVersion: workspaceVersionFor(next.catalog.workspace),
					data: next,
					error: null,
					isLoading: false,
				});
				onWorkspaceChange(next.catalog.workspace);
			}
		} catch (error) {
			if (mountedRef.current && requestIDRef.current === requestID) {
				setCatalogState(previous => ({
					workspaceVersion: requestedWorkspaceVersion,
					data: previous?.workspaceVersion === requestedWorkspaceVersion ? previous.data : null,
					error,
					isLoading: false,
				}));
			}
			throw error;
		}
	}, [onWorkspaceChange, workspace, workspaceVersion]);

	const artifacts = useMemo(() => (catalogData ? getWorkspaceArtifacts(catalogData.catalog) : []), [catalogData]);
	const artifactsByRef = useMemo(
		() => new Map(artifacts.map(artifact => [artifactRefKey(artifact.artifact), artifact] as const)),
		[artifacts]
	);

	const invalidateCatalog = useCallback((message?: string) => {
		requestIDRef.current += 1;
		setCatalogState(null);
		if (message) {
			setRefreshSummary(message);
		}
	}, []);

	useEffect(() => {
		if (!isExpanded || catalogData || catalogError || isCatalogLoading) {
			return;
		}

		void reloadCatalog().catch(() => undefined);
	}, [catalogData, catalogError, isCatalogLoading, isExpanded, reloadCatalog]);

	const visibleArtifacts = useMemo(
		() => artifacts.filter(artifact => workspaceArtifactMatchesSearch(artifact, recordSearch)),
		[artifacts, recordSearch]
	);
	const diagnostics = useMemo(() => (catalogData ? collectWorkspaceDiagnostics(catalogData) : []), [catalogData]);

	const showFailure = useCallback((error: unknown, fallback: string) => {
		if (mountedRef.current) {
			setAlertMessage(getErrorMessage(error, fallback));
		}
	}, []);

	const showCatalogReloadFailure = useCallback((completedAction: string, error: unknown) => {
		if (!mountedRef.current) {
			return;
		}

		const details = getErrorMessage(error, '');
		setAlertMessage(details ? `${completedAction} ${details}` : completedAction);
	}, []);

	const updateArtifactLocally = (artifact: WorkspaceArtifactView) => {
		setCatalogState(previous => {
			if (!previous?.data || previous.workspaceVersion !== workspaceVersion) {
				return previous;
			}

			return {
				...previous,
				data: replaceWorkspaceArtifact(previous.data, artifact),
			};
		});
		setRecordToInspect(previous =>
			previous && artifactRefKey(previous.artifact) === artifactRefKey(artifact.artifact) ? artifact : previous
		);
	};

	const runArtifactMutation = async (key: string, action: () => Promise<WorkspaceArtifactView>, fallback: string) => {
		try {
			let updated: WorkspaceArtifactView | undefined;
			await runAction(key, async () => {
				updated = await action();
			});
			if (updated && mountedRef.current) {
				updateArtifactLocally(updated);
			}
		} catch (error) {
			showFailure(error, fallback);
		}
	};

	const toggleWorkspace = async (enabled: boolean) => {
		try {
			await runAction('workspace:enabled', async () => {
				const updated = await onUpdateWorkspace({
					expectedRevision: workspace.revision,
					displayName: workspace.displayName,
					description: workspace.description,
					enabled,
					discovery: workspace.discovery,
				});
				void updated;
			});
		} catch (error) {
			showFailure(error, 'Failed to update workspace enable state.');
		}
	};

	const refreshWorkspace = async () => {
		if (mountedRef.current) {
			setRefreshSummary('');
		}

		try {
			await runAction('workspace:refresh', async () => {
				const result = await workspaceAPI.refreshWorkspace(workspace.workspace);
				if (mountedRef.current) {
					setRefreshSummary(
						`Scanned ${result.candidates} candidates. Created ${result.createdArtifacts.length} and updated ${result.updatedArtifacts.length} artifacts.`
					);
				}

				try {
					await reloadCatalog();
				} catch (reloadError) {
					showCatalogReloadFailure(
						'Workspace discovery was refreshed, but the catalog could not be reloaded.',
						reloadError
					);
				}
			});
		} catch (error) {
			showFailure(error, 'Failed to refresh workspace discovery.');
		}
	};

	const saveWorkspace = async (submission: WorkspaceSetupSubmission) => {
		if (submission.kind !== 'update') {
			throw new Error('Expected a workspace update.');
		}

		await onUpdateWorkspace(submission.payload);
		if (mountedRef.current) {
			invalidateCatalog('Workspace settings changed. Refresh the Workspace to publish a new discovery catalog.');
		}
	};

	const requestArtifactRemoval = (record: WorkspaceArtifactView) => {
		setSuppressRemovedBinding(true);
		setRecordToDelete(record);
	};

	const removeArtifact = async () => {
		if (!recordToDelete) {
			return;
		}

		const deletingRecord = recordToDelete;
		await workspaceAPI.unadoptWorkspaceArtifact(
			workspace.workspace,
			deletingRecord.artifact,
			deletingRecord.revision,
			suppressRemovedBinding
		);

		if (!mountedRef.current) {
			return;
		}

		const deletingArtifactKey = artifactRefKey(deletingRecord.artifact);
		setCatalogState(previous => {
			if (!previous?.data || previous.workspaceVersion !== workspaceVersion) {
				return previous;
			}

			return {
				...previous,
				data: removeWorkspaceArtifact(previous.data, deletingRecord.artifact.artifactID),
			};
		});
		setRecordToInspect(previous =>
			previous && artifactRefKey(previous.artifact) === deletingArtifactKey ? null : previous
		);
		setRecordToDelete(null);
		setSuppressRemovedBinding(true);

		try {
			await reloadCatalog();
		} catch (reloadError) {
			showCatalogReloadFailure('Artifact was removed, but the Workspace catalog could not be reloaded.', reloadError);
		}
	};

	const adoptOccurrence = async (occurrence: WorkspaceOccurrenceView) => {
		if (!catalogData || occurrence.state !== ArtifactOccurrenceState.Valid || occurrence.recorded) {
			return;
		}

		await workspaceAPI.adoptWorkspaceOccurrence(workspace.workspace, {
			expectedCatalogRevision: catalogData.catalog.catalogRevision,
			occurrence: {
				sourceID: occurrence.sourceID,
				locator: occurrence.locator,
				subresourceLocator: occurrence.subresourceLocator,
			},
			name: occurrence.logicalName || undefined,
			enabled: true,
			settings: { runtimeDisabled: false },
		});

		try {
			await reloadCatalog();
		} catch (reloadError) {
			showCatalogReloadFailure('Artifact was adopted, but the Workspace catalog could not be reloaded.', reloadError);
		}
	};

	const unsuppressBinding = async (suppression: WorkspaceSuppressionView) => {
		const suppressionKey = workspaceSuppressionKey(suppression);
		const pendingKey = `suppression:${suppressionKey}`;

		try {
			await runAction(pendingKey, async () => {
				await workspaceAPI.unsuppressWorkspaceBinding(workspace.workspace, suppression.binding, suppression.revision);
				if (mountedRef.current) {
					setCatalogState(previous => {
						if (!previous?.data || previous.workspaceVersion !== workspaceVersion) {
							return previous;
						}

						return {
							...previous,
							data: {
								...previous.data,
								suppressions: previous.data.suppressions.filter(
									item => workspaceSuppressionKey(item) !== suppressionKey
								),
							},
						};
					});
					setRefreshSummary('Source binding was unsuppressed. Refresh the Workspace to discover and adopt it again.');
				}
			});
		} catch (error) {
			showFailure(error, 'Failed to unsuppress the Workspace source binding.');
		}
	};

	const renderArtifact = (record: WorkspaceArtifactView) => (
		<ManagementItemCard
			key={artifactRefKey(record.artifact)}
			title={record.name}
			subtitle={
				<span className="font-mono">
					{record.locator}
					{record.subresourceLocator ? ` / ${record.subresourceLocator}` : ''}
				</span>
			}
			status={
				<>
					<StatusBadge tone={getArtifactStateTone(record.state)}>{record.state}</StatusBadge>
					<StatusBadge tone={record.enabled ? 'success' : 'neutral'}>
						{record.enabled ? 'Enabled' : 'Disabled'}
					</StatusBadge>
				</>
			}
			metadata={
				<>
					<MetadataPill label="Kind">{getArtifactKindLabel(record.kind)}</MetadataPill>
					<MetadataPill label="Source">{sourceLabelFor(record.sourceID)}</MetadataPill>
					{record.diagnostics?.length ? (
						<MetadataPill label="Diagnostics">{record.diagnostics.length}</MetadataPill>
					) : null}
				</>
			}
		>
			<RecordControls
				workspace={workspace}
				record={record}
				isPending={isPending}
				onToggleEnabled={(current, enabled) => {
					void runArtifactMutation(
						`${current.artifact.artifactID}:enabled`,
						() =>
							workspaceAPI.setWorkspaceArtifactEnabled(workspace.workspace, current.artifact, {
								expectedRevision: current.revision,
								enabled,
							}),
						'Failed to update Artifact enable state.'
					);
				}}
				onSetRuntimeDisabled={(current, disabled) => {
					void runArtifactMutation(
						`${current.artifact.artifactID}:runtime`,
						() =>
							workspaceAPI.setWorkspaceArtifactRuntimeDisabled(workspace.workspace, current.artifact, {
								expectedRevision: current.revision,
								runtimeDisabled: disabled,
							}),
						'Failed to update runtime permission.'
					);
				}}
				onView={setRecordToInspect}
				onDelete={requestArtifactRemoval}
			/>
		</ManagementItemCard>
	);

	const contextArtifact = (context: WorkspaceContextView): WorkspaceArtifactView | undefined =>
		artifactsByRef.get(artifactRefKey(context.artifact));

	const skillArtifact = (skill: WorkspaceSkillView): WorkspaceArtifactView | undefined =>
		artifactsByRef.get(artifactRefKey(skill.artifact));

	const tabs: Array<{ key: WorkspaceTab; label: string; count?: number }> = [
		{ key: 'contexts', label: 'Contexts', count: catalogData?.contexts.length ?? 0 },
		{ key: 'skills', label: 'Skills', count: catalogData?.skills.length ?? 0 },
		{ key: 'sources', label: 'Sources', count: workspace.attachments.length },
		{ key: 'records', label: 'All Artifacts', count: artifacts.length },
		{ key: 'observations', label: 'Observations', count: catalogData?.catalog.occurrences.length ?? 0 },
		{ key: 'suppressions', label: 'Suppressions', count: catalogData?.suppressions.length ?? 0 },
		{ key: 'diagnostics', label: 'Diagnostics', count: diagnostics.length },
	];

	const workspacePanelID = `workspace-panel-${workspace.workspace.rootID}-${workspace.workspace.collectionID}`;
	const suppressRemovedBindingID = `workspace-remove-suppress-${workspace.workspace.collectionID}-${
		recordToDelete?.artifact.artifactID ?? 'none'
	}`;

	return (
		<>
			<ManagementBundleCard
				title={workspace.displayName}
				identity={
					workspace.primaryPath ? (
						<span className="font-mono break-all">{workspace.primaryPath}</span>
					) : workspace.mode === WorkspaceMode.Empty ? (
						'No project folder attached'
					) : (
						'Project folder path unavailable'
					)
				}
				description={workspace.description}
				status={
					<>
						<StatusBadge tone={workspace.enabled ? 'success' : 'neutral'}>
							{workspace.enabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>{workspace.mode}</StatusBadge>
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						onClick={() => {
							setIsExpanded(previous => !previous);
						}}
						aria-expanded={isExpanded}
						aria-controls={workspacePanelID}
					>
						<span>{isExpanded ? 'Hide' : 'Manage'}</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				metadata={
					<>
						<MetadataPill label="Context files">{workspace.discovery.additionalLocators?.length ?? 0}</MetadataPill>
						<MetadataPill label="Skill folders">{workspace.discovery.additionalRoots?.length ?? 0}</MetadataPill>
						<MetadataPill label="Sources">{workspace.attachments.length}</MetadataPill>
					</>
				}
				actionLeading={
					<EnabledControl
						id={`workspace-${workspace.workspace.collectionID}-enabled`}
						checked={workspace.enabled}
						onChange={enabled => {
							void toggleWorkspace(enabled);
						}}
						busy={isPending('workspace:enabled')}
					/>
				}
				actions={
					<>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								setIsEditOpen(true);
							}}
						>
							<FiEdit2 size={15} />
							<span>Edit Workspace</span>
						</button>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								setIsSourcesOpen(true);
							}}
						>
							<FiSettings size={15} />
							<span>Manage Sources</span>
						</button>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								void refreshWorkspace();
							}}
							disabled={isPending('workspace:refresh')}
						>
							<FiRefreshCw size={15} />
							<span>{isPending('workspace:refresh') ? 'Refreshing...' : 'Refresh'}</span>
						</button>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								onRequestDelete(workspace);
							}}
						>
							<FiTrash2 size={15} />
							<span>Delete</span>
						</button>
					</>
				}
			>
				{refreshSummary ? (
					<output className="alert alert-success mt-4 rounded-2xl text-sm" aria-live="polite">
						{refreshSummary}
					</output>
				) : null}

				{isExpanded ? (
					<div id={workspacePanelID} className="mt-6 space-y-4">
						<div className="flex flex-wrap gap-2" aria-label={`${workspace.displayName} Workspace sections`}>
							{tabs.map(tab => (
								<button
									key={tab.key}
									type="button"
									className={`btn btn-sm rounded-xl ${activeTab === tab.key ? 'bg-base-300' : 'btn-ghost'}`}
									onClick={() => {
										setActiveTab(tab.key);
									}}
									aria-pressed={activeTab === tab.key}
								>
									<span>{tab.label}</span>
									{tab.count !== undefined ? (
										<span className="border-base-content/20 rounded-lg border px-1.5 py-0.5 text-xs">{tab.count}</span>
									) : null}
								</button>
							))}
						</div>

						{catalogError ? (
							<ManagementResourceError
								title="Workspace catalog could not be loaded"
								error={catalogError}
								isRetrying={isCatalogLoading}
								onRetry={reloadCatalog}
							/>
						) : null}

						{isCatalogLoading && !catalogData ? (
							<output className="flex items-center justify-center gap-2 py-10 text-sm" aria-live="polite">
								<span className="loading loading-spinner loading-sm" />
								<span>Loading workspace catalog...</span>
							</output>
						) : null}

						{catalogData && activeTab === 'records' ? (
							<div className="space-y-3">
								<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
									<div className="flex w-full flex-col gap-2 sm:max-w-xl sm:flex-row">
										<label
											htmlFor={`workspace-artifact-search-${workspace.workspace.collectionID}`}
											className="sr-only"
										>
											Search Workspace Artifacts
										</label>
										<input
											id={`workspace-artifact-search-${workspace.workspace.collectionID}`}
											type="search"
											className="input input-sm min-w-0 grow rounded-xl"
											value={recordSearch}
											onChange={event => {
												setRecordSearch(event.currentTarget.value);
											}}
											placeholder="Search Workspace Artifacts..."
											aria-label="Search Workspace Artifacts"
										/>
										<button
											type="button"
											className="btn btn-sm btn-ghost shrink-0 rounded-xl"
											onClick={() => {
												setIsArtifactBindingOpen(true);
											}}
										>
											<FiPlus size={14} />
											<span>Pin or Suppress</span>
										</button>
									</div>
									<div className="text-base-content/60 text-xs">
										Catalog revision {catalogData.catalog.catalogRevision}
										{catalogData.catalog.catalogCurrent ? '' : ' · catalog is stale'}
									</div>
								</div>

								{visibleArtifacts.map(a => {
									return renderArtifact(a);
								})}

								{visibleArtifacts.length === 0 ? (
									<ManagementEmptyState>
										{artifacts.length === 0
											? 'No Workspace Artifacts were adopted. Refresh the Workspace after adding paths.'
											: 'No Artifacts match the current search.'}
									</ManagementEmptyState>
								) : null}

								{catalogData.catalog.unrecordedOccurrences.length > 0 ? (
									<div className="space-y-3 pt-3">
										<div className="text-sm font-semibold">Discovered but not recorded</div>
										{catalogData.catalog.unrecordedOccurrences.map((occurrence, index) => (
											<ManagementItemCard
												key={`${workspaceOccurrenceKey(occurrence)}:${index}`}
												title={occurrence.logicalName || occurrence.locator}
												subtitle={<span className="font-mono">{occurrence.locator}</span>}
												status={<StatusBadge tone={getOccurrenceStateTone(occurrence)}>{occurrence.state}</StatusBadge>}
												metadata={
													<>
														<MetadataPill label="Source">{sourceLabelFor(occurrence.sourceID)}</MetadataPill>
														{occurrence.kind ? (
															<MetadataPill label="Kind">{getArtifactKindLabel(occurrence.kind)}</MetadataPill>
														) : null}
													</>
												}
											>
												{occurrence.state === ArtifactOccurrenceState.Valid ? (
													<ActionRow>
														<button
															type="button"
															className="btn btn-sm btn-ghost rounded-xl"
															disabled={isPending(workspaceOccurrencePendingKey(occurrence))}
															onClick={() => {
																void runAction(workspaceOccurrencePendingKey(occurrence), () =>
																	adoptOccurrence(occurrence)
																).catch((error: unknown) => {
																	showFailure(error, 'Failed to adopt the discovered Artifact.');
																});
															}}
														>
															<FiPlus size={14} />
															<span>Adopt Artifact</span>
														</button>
													</ActionRow>
												) : null}
											</ManagementItemCard>
										))}
									</div>
								) : null}
							</div>
						) : null}

						{catalogData && activeTab === 'observations' ? (
							<div className="space-y-3">
								<div className="text-base-content/60 px-1 text-xs">
									Observations describe current source content. Recorded observations have durable Workspace Artifacts;
									unrecorded observations do not.
								</div>

								{catalogData.catalog.occurrences.map((occurrence, index) => {
									const occurrenceKey = `${workspaceOccurrenceKey(occurrence)}:${index}`;
									const pendingKey = workspaceOccurrencePendingKey(occurrence);

									return (
										<ManagementItemCard
											key={occurrenceKey}
											title={occurrence.logicalName || occurrence.locator}
											subtitle={
												<span className="font-mono">
													{occurrence.locator}
													{occurrence.subresourceLocator ? ` / ${occurrence.subresourceLocator}` : ''}
												</span>
											}
											status={
												<>
													<StatusBadge tone={getOccurrenceStateTone(occurrence)}>{occurrence.state}</StatusBadge>
													<StatusBadge tone={occurrence.recorded ? 'success' : 'neutral'}>
														{occurrence.recorded ? 'Recorded' : 'Unrecorded'}
													</StatusBadge>
												</>
											}
											metadata={
												<>
													<MetadataPill label="Source">{sourceLabelFor(occurrence.sourceID)}</MetadataPill>
													{occurrence.kind ? (
														<MetadataPill label="Kind">{getArtifactKindLabel(occurrence.kind)}</MetadataPill>
													) : null}
													{occurrence.diagnostics?.length ? (
														<MetadataPill label="Diagnostics">{occurrence.diagnostics.length}</MetadataPill>
													) : null}
												</>
											}
										>
											{!occurrence.recorded && occurrence.state === ArtifactOccurrenceState.Valid ? (
												<ActionRow>
													<button
														type="button"
														className="btn btn-sm btn-ghost rounded-xl"
														disabled={isPending(pendingKey)}
														onClick={() => {
															void runAction(pendingKey, () => adoptOccurrence(occurrence)).catch((error: unknown) => {
																showFailure(error, 'Failed to adopt the discovered Artifact.');
															});
														}}
													>
														<FiPlus size={14} />
														<span>Adopt Artifact</span>
													</button>
												</ActionRow>
											) : null}
										</ManagementItemCard>
									);
								})}

								{catalogData.catalog.occurrences.length === 0 ? (
									<ManagementEmptyState>No source observations are available.</ManagementEmptyState>
								) : null}
							</div>
						) : null}

						{catalogData && activeTab === 'contexts' ? (
							<div className="space-y-3">
								<div className="flex justify-end">
									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										onClick={() => {
											setIsContextPreviewOpen(true);
										}}
										disabled={!workspace.enabled}
									>
										<FiFileText size={14} />
										<span>Preview Composed Context</span>
									</button>
								</div>

								{catalogData.contextLoadError ? (
									<div className="alert alert-warning rounded-2xl text-sm">{catalogData.contextLoadError}</div>
								) : null}

								{catalogData.contexts.map(context => {
									const record = contextArtifact(context);

									return (
										<ManagementItemCard
											key={artifactRefKey(context.artifact)}
											title={context.name}
											subtitle={
												context.name !== context.locator ? <span className="font-mono">{context.locator}</span> : null
											}
											status={
												<>
													<StatusBadge tone={getArtifactStateTone(context.state)}>{context.state}</StatusBadge>
													<StatusBadge tone={context.enabled ? 'success' : 'neutral'}>
														{context.enabled ? 'Enabled' : 'Disabled'}
													</StatusBadge>
												</>
											}
											metadata={<MetadataPill label="Role">{context.role}</MetadataPill>}
										>
											{record ? (
												<RecordControls
													workspace={workspace}
													record={record}
													isPending={isPending}
													onToggleEnabled={(current, enabled) => {
														void runArtifactMutation(
															`${current.artifact.artifactID}:enabled`,
															() =>
																workspaceAPI.setWorkspaceArtifactEnabled(workspace.workspace, current.artifact, {
																	expectedRevision: current.revision,
																	enabled,
																}),
															'Failed to update context enable state.'
														);
													}}
													onSetRuntimeDisabled={(current, disabled) => {
														void runArtifactMutation(
															`${current.artifact.artifactID}:runtime`,
															() =>
																workspaceAPI.setWorkspaceArtifactRuntimeDisabled(
																	workspace.workspace,
																	current.artifact,
																	{
																		expectedRevision: current.revision,
																		runtimeDisabled: disabled,
																	}
																),
															'Failed to update context runtime permission.'
														);
													}}
													onView={setRecordToInspect}
													onDelete={requestArtifactRemoval}
												/>
											) : null}
										</ManagementItemCard>
									);
								})}

								{catalogData.contexts.length === 0 ? (
									<ManagementEmptyState>No workspace contexts discovered.</ManagementEmptyState>
								) : null}
							</div>
						) : null}

						{catalogData && activeTab === 'skills' ? (
							<div className="space-y-3">
								{catalogData.skillLoadError ? (
									<div className="alert alert-warning rounded-2xl text-sm">{catalogData.skillLoadError}</div>
								) : null}

								{catalogData.skills.map(skill => {
									const record = skillArtifact(skill);

									return (
										<ManagementItemCard
											key={artifactRefKey(skill.artifact)}
											title={skill.skill.displayName || skill.skill.name}
											subtitle={<span className="font-mono">{skill.locator}</span>}
											description={skill.skill.description}
											status={
												<>
													<StatusBadge tone={getArtifactStateTone(skill.state)}>{skill.state}</StatusBadge>
													<StatusBadge tone={skill.skill.isEnabled ? 'success' : 'neutral'}>
														{skill.skill.isEnabled ? 'Enabled' : 'Disabled'}
													</StatusBadge>
												</>
											}
											metadata={
												<>
													<MetadataPill label="Slug">{skill.skill.slug}</MetadataPill>
													<MetadataPill label="Insert">{skill.skill.insert}</MetadataPill>
													<MetadataPill label="Arguments">{skill.skill.arguments?.length ?? 0}</MetadataPill>
													{skill.skill.tags?.map(tag => (
														<MetadataPill key={tag} label="Tag">
															{tag}
														</MetadataPill>
													))}
												</>
											}
										>
											{record ? (
												<RecordControls
													workspace={workspace}
													record={record}
													isPending={isPending}
													onToggleEnabled={(current, enabled) => {
														void runArtifactMutation(
															`${current.artifact.artifactID}:enabled`,
															() =>
																workspaceAPI.setWorkspaceArtifactEnabled(workspace.workspace, current.artifact, {
																	expectedRevision: current.revision,
																	enabled,
																}),
															'Failed to update skill enable state.'
														);
													}}
													onSetRuntimeDisabled={(current, disabled) => {
														void runArtifactMutation(
															`${current.artifact.artifactID}:runtime`,
															() =>
																workspaceAPI.setWorkspaceArtifactRuntimeDisabled(
																	workspace.workspace,
																	current.artifact,
																	{
																		expectedRevision: current.revision,
																		runtimeDisabled: disabled,
																	}
																),
															'Failed to update skill runtime permission.'
														);
													}}
													onView={setRecordToInspect}
													onDelete={requestArtifactRemoval}
												/>
											) : null}
										</ManagementItemCard>
									);
								})}

								{catalogData.skills.length === 0 ? (
									<ManagementEmptyState>No workspace skills discovered.</ManagementEmptyState>
								) : null}
							</div>
						) : null}

						{activeTab === 'sources' ? (
							<div className="space-y-3">
								<div className="flex justify-end">
									<button
										type="button"
										className="btn btn-sm btn-ghost rounded-xl"
										onClick={() => {
											setIsSourcesOpen(true);
										}}
									>
										<FiSettings size={14} />
										<span>Manage Sources</span>
									</button>
								</div>

								{workspace.attachments.map(attachment => (
									<ManagementItemCard
										key={attachment.sourceID}
										title={attachment.path ?? attachment.sourceDisplayName ?? 'Attached source'}
										subtitle={attachment.path ? attachment.sourceDisplayName : attachment.sourceKind}
										status={
											<>
												<StatusBadge tone={attachment.enabled ? 'success' : 'neutral'}>
													{attachment.enabled ? 'Enabled' : 'Disabled'}
												</StatusBadge>
												<StatusBadge>{attachment.role}</StatusBadge>
											</>
										}
										metadata={
											attachment.sourceKind ? <MetadataPill label="Type">{attachment.sourceKind}</MetadataPill> : null
										}
									/>
								))}

								{workspace.attachments.length === 0 ? (
									<ManagementEmptyState>No sources attached.</ManagementEmptyState>
								) : null}

								<div className="text-base-content/60 rounded-2xl px-1 text-xs">
									Workspace attachment roles determine how each Artifact Store Source participates in discovery. Source
									configuration remains private to Artifact Store and is never copied into Workspace data.
								</div>
							</div>
						) : null}

						{catalogData && activeTab === 'suppressions' ? (
							<div className="space-y-3">
								{catalogData.suppressionLoadError ? (
									<div className="alert alert-warning rounded-2xl text-sm">{catalogData.suppressionLoadError}</div>
								) : null}

								<div className="text-base-content/60 px-1 text-xs">
									Suppressed source bindings are not automatically adopted during refresh. Unsuppressing a binding does
									not modify source content.
								</div>

								{catalogData.suppressions.map(suppression => {
									const pendingKey = `suppression:${workspaceSuppressionKey(suppression)}`;

									return (
										<ManagementItemCard
											key={`${pendingKey}:${suppression.revision}`}
											title={suppression.binding.locator}
											subtitle={
												suppression.binding.subresourceLocator ? (
													<span className="font-mono">{suppression.binding.subresourceLocator}</span>
												) : null
											}
											status={<StatusBadge tone="warning">Suppressed</StatusBadge>}
											metadata={
												<>
													<MetadataPill label="Kind">
														{getArtifactKindLabel(suppression.binding.expectedKind)}
													</MetadataPill>
													<MetadataPill label="Source">{sourceLabelFor(suppression.binding.sourceID)}</MetadataPill>
												</>
											}
										>
											<ActionRow>
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={isPending(pendingKey)}
													onClick={() => {
														void unsuppressBinding(suppression);
													}}
												>
													<FiRefreshCw size={14} />
													<span>Unsuppress</span>
												</button>
											</ActionRow>
										</ManagementItemCard>
									);
								})}

								{catalogData.suppressions.length === 0 && !catalogData.suppressionLoadError ? (
									<ManagementEmptyState>No source bindings are suppressed.</ManagementEmptyState>
								) : null}
							</div>
						) : null}

						{catalogData && activeTab === 'diagnostics' ? <WorkspaceDiagnostics diagnostics={diagnostics} /> : null}
					</div>
				) : null}
			</ManagementBundleCard>

			<WorkspaceSetupModal
				isOpen={isEditOpen}
				onClose={() => {
					setIsEditOpen(false);
				}}
				onSubmit={saveWorkspace}
				workspace={workspace}
				existingDisplayNames={existingDisplayNames}
			/>

			<WorkspaceSourcesModal
				isOpen={isSourcesOpen}
				onClose={() => {
					setIsSourcesOpen(false);
				}}
				workspace={workspace}
				onWorkspaceChange={onWorkspaceChange}
				onCatalogInvalidated={() => {
					invalidateCatalog('Workspace Sources changed. Refresh the Workspace to publish a new discovery catalog.');
				}}
			/>

			<WorkspaceArtifactBindingModal
				isOpen={isArtifactBindingOpen}
				onClose={() => {
					setIsArtifactBindingOpen(false);
				}}
				workspace={workspace}
				onChanged={async () => {
					try {
						await reloadCatalog();
					} catch (reloadError) {
						showCatalogReloadFailure(
							'Workspace Artifact binding was changed, but the catalog could not be reloaded.',
							reloadError
						);
					}
				}}
			/>

			<WorkspaceResourceDetailsModal
				isOpen={recordToInspect !== null}
				onClose={() => {
					setRecordToInspect(null);
				}}
				workspace={workspace}
				record={recordToInspect}
			/>

			<WorkspaceContextPreview
				isOpen={isContextPreviewOpen}
				onClose={() => {
					setIsContextPreviewOpen(false);
				}}
				workspace={workspace}
			/>

			<ModalConfirmDialog
				isOpen={recordToDelete !== null}
				onClose={() => {
					setRecordToDelete(null);
					setSuppressRemovedBinding(true);
				}}
				title="Remove Workspace Artifact"
				message={
					<div className="space-y-3 text-sm">
						<p>
							Remove Artifact <span className="font-semibold">{recordToDelete?.name}</span>?
						</p>
						<p className="text-base-content/70">The source content is not deleted.</p>
						<label
							htmlFor={suppressRemovedBindingID}
							className="border-base-content/10 bg-base-100 flex cursor-pointer items-start gap-3 rounded-xl border p-3"
							aria-label="Suppress Binding"
						>
							<input
								id={suppressRemovedBindingID}
								type="checkbox"
								className="checkbox checkbox-sm mt-0.5"
								checked={suppressRemovedBinding}
								aria-describedby={`${suppressRemovedBindingID}-help`}
								onChange={event => {
									setSuppressRemovedBinding(event.currentTarget.checked);
								}}
							/>
							<span className="min-w-0">
								<span className="block font-medium">Prevent automatic re-adoption</span>
								<span id={`${suppressRemovedBindingID}-help`} className="text-base-content/70 mt-1 block text-xs">
									Suppress this typed source binding. It can be restored later from the Suppressions tab.
								</span>
							</span>
						</label>
						{!suppressRemovedBinding ? (
							<p className="text-warning text-xs">
								A later Workspace refresh may automatically adopt this Artifact again.
							</p>
						) : null}
					</div>
				}
				confirmLabel={suppressRemovedBinding ? 'Remove and Suppress' : 'Remove Artifact'}
				busyLabel="Removing..."
				confirmTone="error"
				onConfirm={removeArtifact}
				blockCancel
			/>

			<ActionDeniedAlertModal
				isOpen={Boolean(alertMessage)}
				onClose={() => {
					setAlertMessage('');
				}}
				message={alertMessage}
			/>
		</>
	);
}
