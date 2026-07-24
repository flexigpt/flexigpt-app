import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { FiChevronDown, FiChevronUp, FiEdit2, FiEye, FiFileText, FiLink, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import type {
	UpdateWorkspacePayload,
	WorkspaceContextView,
	WorkspaceRecordView,
	WorkspaceSkillView,
	WorkspaceView,
} from '@/spec/workspace';
import { WorkspaceArtifactKind, WorkspaceMode, WorkspaceRecordMode } from '@/spec/workspace';

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

import type { WorkspaceCatalogData } from '@/workspaces/lib/workspace_utils';
import {
	collectWorkspaceDiagnostics,
	getArtifactKindLabel,
	getErrorMessage,
	getOccurrenceStateTone,
	getRecordModeLabel,
	getRecordStateTone,
	getWorkspaceRecords,
	normalizeWorkspaceCatalog,
	removeWorkspaceRecord,
	replaceWorkspaceRecord,
	workspaceRecordMatchesSearch,
} from '@/workspaces/lib/workspace_utils';
import { WorkspaceContextPreview } from '@/workspaces/workspace_context_preview';
import { WorkspaceDiagnostics } from '@/workspaces/workspace_diagnostics';
import { WorkspaceResourceDetailsModal } from '@/workspaces/workspace_resource_details_modal';
import type { WorkspaceSetupSubmission } from '@/workspaces/workspace_setup_modal';
import { WorkspaceSetupModal } from '@/workspaces/workspace_setup_modal';

type WorkspaceTab = 'records' | 'contexts' | 'skills' | 'sources' | 'diagnostics';

interface WorkspaceCardProps {
	workspace: WorkspaceView;
	existingDisplayNames: readonly string[];
	onWorkspaceChange: (workspace: WorkspaceView) => void;
	onUpdateWorkspace: (payload: UpdateWorkspacePayload) => Promise<WorkspaceView>;
	onRequestDelete: (workspace: WorkspaceView) => void;
}

async function loadWorkspaceCatalogData(rootID: string): Promise<WorkspaceCatalogData> {
	const catalog = normalizeWorkspaceCatalog(await workspaceAPI.getWorkspaceCatalog(rootID));
	const [contextResult, skillResult] = await Promise.allSettled([
		workspaceAPI.listWorkspaceContexts(rootID),
		workspaceAPI.listWorkspaceSkills(rootID),
	]);

	return {
		catalog,
		contexts: contextResult.status === 'fulfilled' ? contextResult.value : [],
		skills: skillResult.status === 'fulfilled' ? skillResult.value : [],
		contextLoadError:
			contextResult.status === 'rejected'
				? getErrorMessage(contextResult.reason, 'Workspace contexts could not be loaded.')
				: undefined,
		skillLoadError:
			skillResult.status === 'rejected'
				? getErrorMessage(skillResult.reason, 'Workspace skills could not be loaded.')
				: undefined,
	};
}

function RecordControls({
	workspace,
	record,
	isPending,
	onToggleEnabled,
	onSetRuntimeDisabled,
	onPin,
	onFollow,
	onView,
	onDelete,
}: {
	workspace: WorkspaceView;
	record: WorkspaceRecordView;
	isPending: (key: string) => boolean;
	onToggleEnabled: (record: WorkspaceRecordView, enabled: boolean) => void;
	onSetRuntimeDisabled: (record: WorkspaceRecordView, disabled: boolean) => void;
	onPin: (record: WorkspaceRecordView) => void;
	onFollow: (record: WorkspaceRecordView) => void;
	onView: (record: WorkspaceRecordView) => void;
	onDelete: (record: WorkspaceRecordView) => void;
}) {
	const runtimeRelevant = record.kind === WorkspaceArtifactKind.Context || record.kind === WorkspaceArtifactKind.Skill;

	return (
		<ActionRow
			leading={
				<div className="flex gap-8">
					<EnabledControl
						id={`workspace-record-${workspace.rootID}-${record.id}`}
						checked={record.enabled}
						onChange={enabled => {
							onToggleEnabled(record, enabled);
						}}
						disabled={!workspace.enabled}
						busy={isPending(`${record.id}:enabled`)}
						title={!workspace.enabled ? 'Enable the workspace first.' : undefined}
					/>
					{runtimeRelevant ? (
						<EnabledControl
							id={`workspace-runtime-${workspace.rootID}-${record.id}`}
							label="Use in conversations"
							checked={!record.runtimeDisabled}
							onChange={allowed => {
								onSetRuntimeDisabled(record, !allowed);
							}}
							disabled={!workspace.enabled}
							busy={isPending(`${record.id}:runtime`)}
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

			{record.mode === WorkspaceRecordMode.Linked && record.resolvedDefinition ? (
				<button
					type="button"
					className="btn btn-sm btn-ghost rounded-xl"
					onClick={() => {
						onPin(record);
					}}
					disabled={isPending(`${record.id}:pin`)}
				>
					<FiLink size={14} />
					<span>Pin Current</span>
				</button>
			) : null}

			{record.mode === WorkspaceRecordMode.Pinned ? (
				<button
					type="button"
					className="btn btn-sm btn-ghost rounded-xl"
					onClick={() => {
						onFollow(record);
					}}
					disabled={isPending(`${record.id}:follow`)}
				>
					<FiRefreshCw size={14} />
					<span>Follow Latest</span>
				</button>
			) : null}

			<button
				type="button"
				className="btn btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onDelete(record);
				}}
				disabled={isPending(`${record.id}:delete`)}
			>
				<FiTrash2 size={14} />
				<span>Delete Record</span>
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
	const [catalogData, setCatalogData] = useState<WorkspaceCatalogData | null>(null);
	const [catalogError, setCatalogError] = useState<unknown>(null);
	const [isCatalogLoading, setIsCatalogLoading] = useState(false);
	const [recordSearch, setRecordSearch] = useState('');
	const [refreshSummary, setRefreshSummary] = useState('');
	const [alertMessage, setAlertMessage] = useState('');

	const [isEditOpen, setIsEditOpen] = useState(false);
	const [recordToInspect, setRecordToInspect] = useState<WorkspaceRecordView | null>(null);
	const [recordToDelete, setRecordToDelete] = useState<WorkspaceRecordView | null>(null);
	const [isContextPreviewOpen, setIsContextPreviewOpen] = useState(false);

	const requestIDRef = useRef(0);
	const mountedRef = useRef(true);
	const { isPending, runAction } = usePendingActions();

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
			return attachment?.path ?? attachment?.sourceDisplayName ?? 'Workspace source';
		},
		[workspace.attachments]
	);

	const reloadCatalog = useCallback(async () => {
		const requestID = requestIDRef.current + 1;
		requestIDRef.current = requestID;
		setIsCatalogLoading(true);
		setCatalogError(null);

		try {
			const next = await loadWorkspaceCatalogData(workspace.rootID);
			if (mountedRef.current && requestIDRef.current === requestID) {
				setCatalogData(next);
				onWorkspaceChange(next.catalog.workspace);
			}
		} catch (error) {
			if (mountedRef.current && requestIDRef.current === requestID) {
				setCatalogError(error);
			}
			throw error;
		} finally {
			if (mountedRef.current && requestIDRef.current === requestID) {
				setIsCatalogLoading(false);
			}
		}
	}, [onWorkspaceChange, workspace.rootID]);

	const records = useMemo(() => (catalogData ? getWorkspaceRecords(catalogData.catalog) : []), [catalogData]);
	const visibleRecords = useMemo(
		() => records.filter(record => workspaceRecordMatchesSearch(record, recordSearch)),
		[recordSearch, records]
	);
	const diagnostics = useMemo(() => (catalogData ? collectWorkspaceDiagnostics(catalogData) : []), [catalogData]);

	const showFailure = (error: unknown, fallback: string) => {
		setAlertMessage(getErrorMessage(error, fallback));
	};

	const updateRecordLocally = (record: WorkspaceRecordView) => {
		setCatalogData(previous => (previous ? replaceWorkspaceRecord(previous, record) : previous));
		setRecordToInspect(previous => (previous?.id === record.id ? record : previous));
	};

	const runRecordMutation = async (key: string, action: () => Promise<WorkspaceRecordView>, fallback: string) => {
		try {
			let updated: WorkspaceRecordView | undefined;
			await runAction(key, async () => {
				updated = await action();
			});
			if (updated) {
				updateRecordLocally(updated);
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
				onWorkspaceChange(updated);
			});
		} catch (error) {
			showFailure(error, 'Failed to update workspace enable state.');
		}
	};

	const refreshWorkspace = async () => {
		setRefreshSummary('');

		try {
			await runAction('workspace:refresh', async () => {
				const result = await workspaceAPI.refreshWorkspace(workspace.rootID);
				setRefreshSummary(
					`Scanned ${result.candidates} candidates. Created ${result.createdRecords.length} and updated ${result.updatedRecords.length} records.`
				);
				await reloadCatalog();
			});
		} catch (error) {
			showFailure(error, 'Failed to refresh workspace discovery.');
		}
	};

	const saveWorkspace = async (submission: WorkspaceSetupSubmission) => {
		if (submission.kind !== 'update') {
			throw new Error('Expected a workspace update.');
		}
		const updated = await onUpdateWorkspace(submission.payload);
		onWorkspaceChange(updated);
		setCatalogData(null);
	};

	const deleteRecord = async () => {
		if (!recordToDelete) {
			return;
		}

		await workspaceAPI.deleteWorkspaceRecord(workspace.rootID, recordToDelete.id, recordToDelete.revision);

		setCatalogData(previous => (previous ? removeWorkspaceRecord(previous, recordToDelete.id) : previous));
	};

	const renderRecord = (record: WorkspaceRecordView) => (
		<ManagementItemCard
			key={record.id}
			title={record.name}
			subtitle={
				<span className="font-mono">
					{record.locator}
					{record.subresourceLocator ? ` / ${record.subresourceLocator}` : ''}
				</span>
			}
			status={
				<>
					<StatusBadge tone={getRecordStateTone(record.state)}>{record.state}</StatusBadge>
					<StatusBadge tone={record.enabled ? 'success' : 'neutral'}>
						{record.enabled ? 'Enabled' : 'Disabled'}
					</StatusBadge>
				</>
			}
			metadata={
				<>
					<MetadataPill label="Kind">{getArtifactKindLabel(record.kind)}</MetadataPill>
					<MetadataPill label="Mode">{getRecordModeLabel(record.mode)}</MetadataPill>
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
					void runRecordMutation(
						`${current.id}:enabled`,
						() => workspaceAPI.setWorkspaceRecordEnabled(workspace.rootID, current.id, current.revision, enabled),
						'Failed to update record enable state.'
					);
				}}
				onSetRuntimeDisabled={(current, disabled) => {
					void runRecordMutation(
						`${current.id}:runtime`,
						() =>
							workspaceAPI.setWorkspaceRecordRuntimeDisabled(workspace.rootID, current.id, current.revision, disabled),
						'Failed to update runtime permission.'
					);
				}}
				onPin={current => {
					if (!current.resolvedDefinition) {
						showFailure(undefined, 'The record does not have a resolved definition to pin.');
						return;
					}

					void runRecordMutation(
						`${current.id}:pin`,
						() =>
							workspaceAPI.pinWorkspaceRecord(
								workspace.rootID,
								current.id,
								current.revision,
								current.resolvedDefinition as string
							),
						'Failed to pin workspace record.'
					);
				}}
				onFollow={current => {
					void runRecordMutation(
						`${current.id}:follow`,
						() => workspaceAPI.followWorkspaceRecord(workspace.rootID, current.id, current.revision),
						'Failed to make the record follow the latest definition.'
					);
				}}
				onView={setRecordToInspect}
				onDelete={setRecordToDelete}
			/>
		</ManagementItemCard>
	);

	const contextRecord = (context: WorkspaceContextView): WorkspaceRecordView | undefined =>
		records.find(record => record.id === context.recordID);

	const skillRecord = (skill: WorkspaceSkillView): WorkspaceRecordView | undefined =>
		records.find(record => record.id === skill.recordID);

	const tabs: Array<{ key: WorkspaceTab; label: string; count?: number }> = [
		{ key: 'contexts', label: 'Contexts', count: catalogData?.contexts.length ?? 0 },
		{ key: 'skills', label: 'Skills', count: catalogData?.skills.length ?? 0 },
		{ key: 'sources', label: 'Sources', count: workspace.attachments.length },
		{ key: 'records', label: 'All Catalog Records', count: records.length },
		{ key: 'diagnostics', label: 'Diagnostics', count: diagnostics.length },
	];

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
							const nextExpanded = !isExpanded;
							setIsExpanded(nextExpanded);
							if (nextExpanded && !catalogData && !isCatalogLoading && !catalogError) {
								void reloadCatalog().catch(() => undefined);
							}
						}}
						aria-expanded={isExpanded}
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
						id={`workspace-${workspace.rootID}-enabled`}
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
							<span>Edit Paths</span>
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
				{refreshSummary ? <div className="alert alert-success mt-4 rounded-2xl text-sm">{refreshSummary}</div> : null}

				{isExpanded ? (
					<div className="mt-6 space-y-4">
						<div className="flex flex-wrap gap-2">
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
							<div className="flex items-center justify-center gap-2 py-10 text-sm">
								<span className="loading loading-spinner loading-sm" />
								<span>Loading workspace catalog...</span>
							</div>
						) : null}

						{catalogData && activeTab === 'records' ? (
							<div className="space-y-3">
								<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
									<input
										type="search"
										className="input input-sm w-full rounded-xl sm:max-w-md"
										value={recordSearch}
										onChange={event => {
											setRecordSearch(event.currentTarget.value);
										}}
										placeholder="Search workspace records..."
									/>
									<div className="text-base-content/60 text-xs">
										Catalog revision {catalogData.catalog.catalogRevision}
										{catalogData.catalog.catalogCurrent ? '' : ' · catalog is stale'}
									</div>
								</div>

								{visibleRecords.map(r => {
									return renderRecord(r);
								})}

								{visibleRecords.length === 0 ? (
									<ManagementEmptyState>
										{records.length === 0
											? 'No workspace records were discovered. Refresh the workspace after adding paths.'
											: 'No records match the current search.'}
									</ManagementEmptyState>
								) : null}

								{catalogData.catalog.unrecordedOccurrences.length > 0 ? (
									<div className="space-y-3 pt-3">
										<div className="text-sm font-semibold">Discovered but not recorded</div>
										{catalogData.catalog.unrecordedOccurrences.map((occurrence, index) => (
											<ManagementItemCard
												key={`${occurrence.sourceID}:${occurrence.locator}:${index}`}
												title={occurrence.logicalName || occurrence.locator}
												subtitle={<span className="font-mono">{occurrence.locator}</span>}
												status={<StatusBadge tone={getOccurrenceStateTone(occurrence)}>{occurrence.state}</StatusBadge>}
												metadata={
													<>
														<MetadataPill label="Source">{occurrence.sourceID}</MetadataPill>
														{occurrence.kind ? (
															<MetadataPill label="Kind">{getArtifactKindLabel(occurrence.kind)}</MetadataPill>
														) : null}
													</>
												}
											/>
										))}
									</div>
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
									const record = contextRecord(context);

									return (
										<ManagementItemCard
											key={context.recordID}
											title={context.name}
											subtitle={
												context.name !== context.locator ? <span className="font-mono">{context.locator}</span> : null
											}
											status={
												<>
													<StatusBadge tone={getRecordStateTone(context.state)}>{context.state}</StatusBadge>
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
														void runRecordMutation(
															`${current.id}:enabled`,
															() =>
																workspaceAPI.setWorkspaceRecordEnabled(
																	workspace.rootID,
																	current.id,
																	current.revision,
																	enabled
																),
															'Failed to update context enable state.'
														);
													}}
													onSetRuntimeDisabled={(current, disabled) => {
														void runRecordMutation(
															`${current.id}:runtime`,
															() =>
																workspaceAPI.setWorkspaceRecordRuntimeDisabled(
																	workspace.rootID,
																	current.id,
																	current.revision,
																	disabled
																),
															'Failed to update context runtime permission.'
														);
													}}
													onPin={current => {
														if (current.resolvedDefinition) {
															void runRecordMutation(
																`${current.id}:pin`,
																() =>
																	workspaceAPI.pinWorkspaceRecord(
																		workspace.rootID,
																		current.id,
																		current.revision,
																		current.resolvedDefinition as string
																	),
																'Failed to pin context.'
															);
														}
													}}
													onFollow={current => {
														void runRecordMutation(
															`${current.id}:follow`,
															() => workspaceAPI.followWorkspaceRecord(workspace.rootID, current.id, current.revision),
															'Failed to follow context.'
														);
													}}
													onView={setRecordToInspect}
													onDelete={setRecordToDelete}
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
									const record = skillRecord(skill);

									return (
										<ManagementItemCard
											key={skill.recordID}
											title={skill.skill.displayName || skill.skill.name}
											subtitle={<span className="font-mono">{skill.locator}</span>}
											description={skill.skill.description}
											status={
												<>
													<StatusBadge tone={getRecordStateTone(skill.state)}>{skill.state}</StatusBadge>
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
														void runRecordMutation(
															`${current.id}:enabled`,
															() =>
																workspaceAPI.setWorkspaceRecordEnabled(
																	workspace.rootID,
																	current.id,
																	current.revision,
																	enabled
																),
															'Failed to update skill enable state.'
														);
													}}
													onSetRuntimeDisabled={(current, disabled) => {
														void runRecordMutation(
															`${current.id}:runtime`,
															() =>
																workspaceAPI.setWorkspaceRecordRuntimeDisabled(
																	workspace.rootID,
																	current.id,
																	current.revision,
																	disabled
																),
															'Failed to update skill runtime permission.'
														);
													}}
													onPin={current => {
														if (current.resolvedDefinition) {
															void runRecordMutation(
																`${current.id}:pin`,
																() =>
																	workspaceAPI.pinWorkspaceRecord(
																		workspace.rootID,
																		current.id,
																		current.revision,
																		current.resolvedDefinition as string
																	),
																'Failed to pin skill.'
															);
														}
													}}
													onFollow={current => {
														void runRecordMutation(
															`${current.id}:follow`,
															() => workspaceAPI.followWorkspaceRecord(workspace.rootID, current.id, current.revision),
															'Failed to follow skill.'
														);
													}}
													onView={setRecordToInspect}
													onDelete={setRecordToDelete}
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
									Additional project folders are currently configured as discovery paths. A path-based external source
									attachment API should be added before exposing source attachment management here.
								</div>
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
				}}
				title="Delete Workspace Record"
				message={
					<div className="space-y-2 text-sm">
						<p>
							Delete record <span className="font-semibold">{recordToDelete?.name}</span>?
						</p>
						<p className="text-base-content/70">
							The source file is not deleted. A later workspace refresh may discover and record it again.
						</p>
					</div>
				}
				confirmLabel="Delete Record"
				busyLabel="Deleting..."
				confirmTone="error"
				onConfirm={deleteRecord}
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
