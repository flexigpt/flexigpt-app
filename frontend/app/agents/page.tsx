import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
	FiChevronDown,
	FiChevronUp,
	FiDownload,
	FiEdit2,
	FiEye,
	FiPlus,
	FiRefreshCw,
	FiTrash2,
	FiUpload,
} from 'react-icons/fi';

import type { AgentImportCommitResult, AgentImportDestination, AgentView } from '@/spec/agent';
import type { PluginListItem, PluginView } from '@/spec/plugin';
import { ArtifactState } from '@/spec/artifact';
import { pluginListItemFromPluginView } from '@/spec/plugin';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';
import { usePendingActions } from '@/hooks/use_pending_actions';

import type { AgentManagementPageData, AgentPluginData } from '@/apis/agent_management';
import {
	agentArtifactRef,
	agentDisplayName,
	agentPluginKey,
	canDeleteAgentPlugin,
	canEditAgentPluginMetadata,
	EMPTY_AGENT_MANAGEMENT_PAGE_DATA,
	isBuiltInAgentPlugin,
	pluginDisplayName,
} from '@/apis/agent_management';
import { agentManagementAPI, agentStoreAPI, backendAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementBundleCreateModal } from '@/components/managementui/management_bundle_create_modal';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementInfoGrid } from '@/components/managementui/management_info_grid';
import { ManagementInfoRow } from '@/components/managementui/management_info_row';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { ManagementResourceError } from '@/components/managementui/management_resource_error';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';
import { PageFrame } from '@/components/page_frame';

import { AgentDetailsModal } from '@/agents/agent_details_modal';
import { AgentImportModal } from '@/agents/agent_import_modal';
import { formatDateish, textToBase64 } from '@/agents/lib/agent_management_utils';

const AGENT_MANAGEMENT_PAGE_CACHE_TTL_MS = 5 * 60 * 1000;

interface AgentManagementPageDataCache {
	data: AgentManagementPageData;
	loadedAt: number;
}

interface AgentManagementPageDataLoad {
	generation: number;
	promise: Promise<AgentManagementPageData>;
}

interface PluginDetailsState {
	plugin: PluginListItem;
	view?: PluginView;
	error?: string;
	loading: boolean;
}

let agentManagementPageDataCache: AgentManagementPageDataCache | undefined;
let agentManagementPageDataLoad: AgentManagementPageDataLoad | undefined;
let agentManagementPageDataCacheGeneration = 0;

function invalidateAgentManagementPageDataCache(): void {
	agentManagementPageDataCacheGeneration += 1;
	agentManagementPageDataCache = undefined;
}

function rememberAgentManagementPageData(data: AgentManagementPageData): void {
	agentManagementPageDataCache = {
		data,
		loadedAt: Date.now(),
	};
}

function agentRefKey(agent: AgentView): string {
	return `${agent.ref.rootID}:${agent.ref.artifactID}`;
}

function pluginRefKey(plugin: PluginListItem): string {
	return `${plugin.ref.rootID}:${plugin.ref.artifactID}`;
}

async function loadAgentManagementPageData(signal: AbortSignal): Promise<AgentManagementPageData> {
	throwIfAborted(signal);

	const cached = agentManagementPageDataCache;
	if (cached && Date.now() - cached.loadedAt <= AGENT_MANAGEMENT_PAGE_CACHE_TTL_MS) {
		return cached.data;
	}

	const generation = agentManagementPageDataCacheGeneration;
	let load = agentManagementPageDataLoad;

	if (!load || load.generation !== generation) {
		const promise = agentManagementAPI.loadManagementPageData(new AbortController().signal).then(data => {
			if (agentManagementPageDataCacheGeneration === generation) {
				agentManagementPageDataCache = {
					data,
					loadedAt: Date.now(),
				};
			}

			return data;
		});

		load = {
			generation,
			promise,
		};
		agentManagementPageDataLoad = load;

		const clear = () => {
			if (agentManagementPageDataLoad?.promise === promise) {
				agentManagementPageDataLoad = undefined;
			}
		};
		void promise.then(clear, clear);
	}

	const data = await load.promise;
	throwIfAborted(signal);
	return data;
}

interface AgentPluginCardProps {
	data: AgentPluginData;
	onLoadAgents: (plugin: PluginListItem) => Promise<void>;
	onViewPlugin: (plugin: PluginListItem) => void;
	onEditPlugin: (plugin: PluginListItem) => void;
	onDeletePlugin: (plugin: PluginListItem) => void;
	onImportAgent: (destination: AgentImportDestination) => void;
	onTogglePluginEnabled: (plugin: PluginListItem, enabled: boolean) => Promise<void>;
	onViewAgent: (agent: AgentView) => void;
	onExportAgent: (agent: AgentView) => Promise<void>;
	onDeleteAgent: (agent: AgentView) => void;
	onToggleAgentEnabled: (agent: AgentView, enabled: boolean) => Promise<void>;
	onActionError: (message: string) => void;
}

function AgentPluginCard({
	data,
	onLoadAgents,
	onViewPlugin,
	onEditPlugin,
	onDeletePlugin,
	onImportAgent,
	onTogglePluginEnabled,
	onViewAgent,
	onExportAgent,
	onDeleteAgent,
	onToggleAgentEnabled,
	onActionError,
}: AgentPluginCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const { isPending, runAction } = usePendingActions();

	const plugin = data.plugin;
	const builtIn = isBuiltInAgentPlugin(plugin);
	const displayName = pluginDisplayName(plugin);
	const secondaryName = displayName === plugin.name ? undefined : plugin.name;
	const pluginAvailable = plugin.state === ArtifactState.Available;

	const run = (key: string, action: () => Promise<void>, fallback: string) => {
		void runAction(key, action).catch((error: unknown) => {
			onActionError(getErrorMessage(error, fallback));
		});
	};

	const loadAgents = () => {
		if (data.agentsLoaded || data.isLoadingAgents) {
			return;
		}

		void onLoadAgents(plugin).catch((error: unknown) => {
			onActionError(getErrorMessage(error, 'Agents could not be loaded for this Plugin.'));
		});
	};

	const toggleExpanded = () => {
		const next = !isExpanded;
		setIsExpanded(next);

		if (next) {
			loadAgents();
		}
	};

	return (
		<ManagementBundleCard
			title={displayName}
			identity={secondaryName ? <span className="font-mono">{secondaryName}</span> : null}
			description={plugin.description}
			status={
				<>
					<StatusBadge tone={plugin.enabled ? 'success' : 'neutral'}>
						{plugin.enabled ? 'Enabled' : 'Disabled'}
					</StatusBadge>
					{plugin.state !== ArtifactState.Available ? <StatusBadge tone="warning">{plugin.state}</StatusBadge> : null}
					{builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
					{plugin.baseline ? <StatusBadge>Baseline</StatusBadge> : null}
				</>
			}
			disclosure={
				<button
					type="button"
					className="btn btn-sm btn-ghost rounded-xl"
					aria-expanded={isExpanded}
					onClick={toggleExpanded}
				>
					<span>
						{data.isLoadingAgents
							? 'Loading Agents...'
							: data.agentsLoaded
								? `Agents: ${data.agents.length}`
								: `Agents: ${plugin.memberCount}`}
					</span>
					{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
				</button>
			}
			actionLeading={
				<EnabledControl
					id={`agent-plugin-${plugin.ref.rootID}-${plugin.ref.artifactID}`}
					checked={plugin.enabled}
					disabled={!pluginAvailable || isPending('plugin:toggle')}
					busy={isPending('plugin:toggle')}
					title={!pluginAvailable ? 'This Agent Plugin is unavailable.' : undefined}
					onChange={enabled => {
						run(
							'plugin:toggle',
							() => onTogglePluginEnabled(plugin, enabled),
							'Failed to update Agent Plugin enablement.'
						);
					}}
				/>
			}
			actions={
				<>
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						onClick={() => {
							onViewPlugin(plugin);
						}}
					>
						<FiEye size={16} />
						<span>Details</span>
					</button>

					{canEditAgentPluginMetadata(plugin) ? (
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								onEditPlugin(plugin);
							}}
						>
							<FiEdit2 size={16} />
							<span>Edit Plugin</span>
						</button>
					) : null}

					{data.importDestination !== undefined && data.importDestination !== null ? (
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={!pluginAvailable}
							onClick={() => {
								onImportAgent(data.importDestination as AgentImportDestination);
							}}
						>
							<FiUpload size={16} />
							<span>Import Agent</span>
						</button>
					) : null}

					{canDeleteAgentPlugin(plugin) ? (
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={!data.agentsLoaded || data.agents.length > 0 || Boolean(data.agentLoadError)}
							title={
								!data.agentsLoaded
									? 'Load Plugin Agents before deletion.'
									: data.agentLoadError
										? 'Reload Plugin Agents before deletion.'
										: data.agents.length > 0
											? 'Remove all Agents before deleting this Plugin.'
											: undefined
							}
							onClick={() => {
								onDeletePlugin(plugin);
							}}
						>
							<FiTrash2 size={16} />
							<span>Delete Plugin</span>
						</button>
					) : null}
				</>
			}
		>
			{data.agentLoadError ? (
				<div className="alert alert-warning mt-3 rounded-2xl text-sm">
					<div className="grow">
						<div className="font-semibold">Agents could not be loaded for this Plugin</div>
						<div>{data.agentLoadError}</div>
					</div>
					<button
						type="button"
						className="btn btn-sm rounded-xl"
						disabled={data.isLoadingAgents}
						onClick={() => {
							void onLoadAgents(plugin).catch((error: unknown) => {
								onActionError(getErrorMessage(error, 'Agents could not be loaded for this Plugin.'));
							});
						}}
					>
						Retry
					</button>
				</div>
			) : null}

			{isExpanded ? (
				<div className="mt-6 space-y-3">
					{!data.agentsLoaded ? (
						<ManagementEmptyState>
							{data.isLoadingAgents
								? 'Loading Agents in this Plugin...'
								: data.agentLoadError
									? 'Agent contents are unavailable.'
									: 'Agent contents have not been loaded.'}
						</ManagementEmptyState>
					) : null}

					{data.agentsLoaded
						? data.agents.map(agent => {
								const agentAvailable = agent.state === ArtifactState.Available;
								const toggleKey = `agent:${agent.ref.rootID}:${agent.ref.artifactID}:toggle`;
								const exportKey = `agent:${agent.ref.rootID}:${agent.ref.artifactID}:export`;

								return (
									<ManagementItemCard
										key={agentRefKey(agent)}
										title={agentDisplayName(agent)}
										subtitle={agent.name === agentDisplayName(agent) ? undefined : agent.name}
										description={agent.description}
										status={
											<>
												<StatusBadge tone={agent.enabled ? 'success' : 'neutral'}>
													{agent.enabled ? 'Enabled' : 'Disabled'}
												</StatusBadge>
												{agent.state !== ArtifactState.Available ? (
													<StatusBadge tone="warning">{agent.state}</StatusBadge>
												) : null}
												{agent.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
												{agent.managed ? <StatusBadge>Managed</StatusBadge> : null}
											</>
										}
										metadata={
											<>
												<MetadataPill label="Revision">{agent.revision}</MetadataPill>
												{agent.definitionDigest ? <MetadataPill label="Definition">Available</MetadataPill> : null}
											</>
										}
									>
										<ActionRow
											leading={
												<EnabledControl
													id={`agent-${agent.ref.rootID}-${agent.ref.artifactID}`}
													checked={agent.enabled}
													disabled={!agentAvailable || isPending(toggleKey)}
													busy={isPending(toggleKey)}
													title={!agentAvailable ? 'This Agent Artifact is unavailable.' : undefined}
													onChange={enabled => {
														run(
															toggleKey,
															() => onToggleAgentEnabled(agent, enabled),
															'Failed to update Agent enablement.'
														);
													}}
												/>
											}
										>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													onViewAgent(agent);
												}}
											>
												<FiEye size={15} />
												<span>View</span>
											</button>

											{agentAvailable ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													disabled={isPending(exportKey)}
													onClick={() => {
														run(exportKey, () => onExportAgent(agent), 'Failed to export Agent YAML.');
													}}
												>
													<FiDownload size={15} />
													<span>{isPending(exportKey) ? 'Exporting...' : 'Export'}</span>
												</button>
											) : null}

											{agent.managed && !agent.builtIn ? (
												<button
													type="button"
													className="btn btn-sm btn-ghost rounded-xl"
													onClick={() => {
														onDeleteAgent(agent);
													}}
												>
													<FiTrash2 size={15} />
													<span>Delete</span>
												</button>
											) : null}
										</ActionRow>
									</ManagementItemCard>
								);
							})
						: null}

					{data.agentsLoaded && data.agents.length === 0 ? (
						<ManagementEmptyState>No resolved Agents are currently available in this Plugin.</ManagementEmptyState>
					) : null}
				</div>
			) : null}
		</ManagementBundleCard>
	);
}

function AgentPluginEditModalContent({
	plugin,
	onClose,
	onSubmit,
}: {
	plugin: PluginListItem;
	onClose: () => void;
	onSubmit: (displayName: string, description?: string) => Promise<void>;
}) {
	const [displayName, setDisplayName] = useState(plugin.displayName);
	const [description, setDescription] = useState(plugin.description ?? '');
	const [error, setError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const submit = async () => {
		const nextDisplayName = displayName.trim();

		if (!nextDisplayName) {
			setError('Plugin display name is required.');
			return;
		}

		setIsSubmitting(true);
		setError('');

		try {
			await onSubmit(nextDisplayName, description.trim() || undefined);
			onClose();
		} catch (submitError) {
			setError(getErrorMessage(submitError, 'Could not update the Agent Plugin.'));
		} finally {
			setIsSubmitting(false);
		}
	};

	return (
		<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
			<ModalHeader
				title="Edit Agent Plugin"
				description="The logical Plugin name remains stable. Imported Agents remain immutable."
				onClose={onClose}
				closeDisabled={isSubmitting}
			/>

			<form
				className="space-y-4 p-4 sm:p-6"
				onSubmit={event => {
					event.preventDefault();
					void submit();
				}}
			>
				{error ? <div className="alert alert-error rounded-2xl text-sm">{error}</div> : null}

				<ModalSection title="Plugin details">
					<ModalField label="Plugin name" htmlFor="agent-plugin-name">
						<input id="agent-plugin-name" className="input w-full rounded-xl font-mono" value={plugin.name} readOnly />
					</ModalField>

					<ModalField label="Display name" htmlFor="agent-plugin-display-name" required>
						<input
							id="agent-plugin-display-name"
							className="input w-full rounded-xl"
							value={displayName}
							onChange={event => {
								setDisplayName(event.currentTarget.value);
							}}
							disabled={isSubmitting}
							maxLength={256}
						/>
					</ModalField>

					<ModalField label="Description" htmlFor="agent-plugin-description">
						<textarea
							id="agent-plugin-description"
							className="textarea min-h-24 w-full rounded-xl"
							value={description}
							onChange={event => {
								setDescription(event.currentTarget.value);
							}}
							disabled={isSubmitting}
							maxLength={2000}
						/>
					</ModalField>
				</ModalSection>

				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" disabled={isSubmitting} onClick={onClose}>
						Cancel
					</button>
					<button type="submit" className="btn btn-primary rounded-xl" disabled={isSubmitting}>
						{isSubmitting ? 'Saving...' : 'Save Plugin'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

function AgentPluginEditModal({
	plugin,
	onClose,
	onSubmit,
}: {
	plugin: PluginListItem | null;
	onClose: () => void;
	onSubmit: (plugin: PluginListItem, displayName: string, description?: string) => Promise<void>;
}) {
	if (!plugin) {
		return null;
	}

	return (
		<ModalDialog isOpen={true} onClose={onClose} blockCancel>
			<AgentPluginEditModalContent
				key={`${plugin.ref.rootID}:${plugin.ref.artifactID}:${plugin.revision}`}
				plugin={plugin}
				onClose={onClose}
				onSubmit={(displayName, description) => onSubmit(plugin, displayName, description)}
			/>
		</ModalDialog>
	);
}

function AgentPluginDetailsModal({ state, onClose }: { state: PluginDetailsState | null; onClose: () => void }) {
	if (!state) {
		return null;
	}

	const plugin = state.view ?? state.plugin;
	const modalKey = state.view
		? `agent-plugin:${state.view.artifact.rootID}:${state.view.artifact.id}:${state.view.artifact.revision}`
		: `agent-plugin:${state.plugin.ref.rootID}:${state.plugin.ref.artifactID}:${state.plugin.revision}`;

	return (
		<ManagementDetailsModal isOpen={true} onClose={onClose} title="Agent Plugin Details" modalKey={modalKey}>
			{state.loading ? <Loader text="Loading Agent Plugin details..." /> : null}

			{state.error ? (
				<ManagementResourceError
					title="Agent Plugin details could not be loaded"
					error={state.error}
					isRetrying={false}
					onRetry={async () => undefined}
				/>
			) : null}

			<ManagementInfoGrid>
				<ManagementInfoRow label="Display Name">{pluginDisplayName(plugin)}</ManagementInfoRow>
				<ManagementInfoRow label="Name" mono>
					{plugin.name}
				</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{isBuiltInAgentPlugin(plugin) ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Baseline">{plugin.baseline ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Enabled">
					{'artifact' in plugin ? (plugin.artifact.enabled ? 'Yes' : 'No') : plugin.enabled ? 'Yes' : 'No'}
				</ManagementInfoRow>
				<ManagementInfoRow label="State">
					{'artifact' in plugin ? plugin.artifact.state : plugin.state}
				</ManagementInfoRow>
				<ManagementInfoRow label="Revision">
					{'artifact' in plugin ? plugin.artifact.revision : plugin.revision}
				</ManagementInfoRow>
				{'artifact' in plugin ? (
					<>
						<ManagementInfoRow label="Source" mono>
							{plugin.artifact.binding.sourceID}
						</ManagementInfoRow>
						<ManagementInfoRow label="Created">{formatDateish(plugin.artifact.createdAt)}</ManagementInfoRow>
						<ManagementInfoRow label="Modified">{formatDateish(plugin.artifact.modifiedAt)}</ManagementInfoRow>
					</>
				) : (
					<>
						<ManagementInfoRow label="Source" mono>
							{plugin.sourceID}
						</ManagementInfoRow>
						<ManagementInfoRow label="Declared members">{plugin.memberCount}</ManagementInfoRow>
					</>
				)}
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{plugin.description || '—'}</span>
				</ManagementInfoRow>
			</ManagementInfoGrid>
		</ManagementDetailsModal>
	);
}

// oxlint-disable-next-line no-restricted-exports
export default function AgentsPage() {
	const loadPageData = useCallback((signal: AbortSignal) => loadAgentManagementPageData(signal), []);
	const {
		data: pageData,
		error: pageLoadError,
		isLoading,
		isRefreshing,
		hasResolved,
		reloadOrThrow,
		setData: setPageData,
	} = useAsyncResource(loadPageData, {
		initialData: agentManagementPageDataCache?.data ?? EMPTY_AGENT_MANAGEMENT_PAGE_DATA,
	});

	const [isCreatePluginOpen, setIsCreatePluginOpen] = useState(false);
	const [isImportModalOpen, setIsImportModalOpen] = useState(false);
	const [importDestination, setImportDestination] = useState<AgentImportDestination | null>(null);
	const [agentToView, setAgentToView] = useState<AgentView | null>(null);
	const [agentToDelete, setAgentToDelete] = useState<AgentView | null>(null);
	const [pluginToDelete, setPluginToDelete] = useState<PluginListItem | null>(null);
	const [pluginToEdit, setPluginToEdit] = useState<PluginListItem | null>(null);
	const [pluginDetails, setPluginDetails] = useState<PluginDetailsState | null>(null);
	const [isDeletingAgent, setIsDeletingAgent] = useState(false);
	const [isDeletingPlugin, setIsDeletingPlugin] = useState(false);
	const [alertMessage, setAlertMessage] = useState('');
	const [pluginCreationRootID, setPluginCreationRootID] = useState('');

	const mountedRef = useRef(false);
	const agentLoadRequestIDRef = useRef<Record<string, number>>({});
	const agentLoadEpochRef = useRef(0);
	const pluginDetailsRequestIDRef = useRef(0);

	const pluginCreationRoots = useMemo(
		() =>
			[
				...new Map(
					pageData.plugins
						.map(value => value.plugin)
						.filter(plugin => plugin.baseline && !isBuiltInAgentPlugin(plugin))
						.map(
							plugin =>
								[
									plugin.ref.rootID,
									{
										rootID: plugin.ref.rootID,
										label: pluginDisplayName(plugin),
									},
								] as const
						)
				).values(),
			].toSorted((left, right) => left.label.localeCompare(right.label)),
		[pageData.plugins]
	);

	const effectivePluginCreationRootID = pluginCreationRoots.some(value => value.rootID === pluginCreationRootID)
		? pluginCreationRootID
		: (pluginCreationRoots[0]?.rootID ?? '');

	const showAlert = useCallback((message: string) => {
		setAlertMessage(message);
	}, []);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
			agentLoadRequestIDRef.current = {};
			agentLoadEpochRef.current += 1;
			pluginDetailsRequestIDRef.current += 1;
		};
	}, []);

	useEffect(() => {
		if (
			hasResolved &&
			!pageLoadError &&
			!isLoading &&
			!isRefreshing &&
			pageData.plugins.every(plugin => !plugin.isLoadingAgents)
		) {
			rememberAgentManagementPageData(pageData);
		}
	}, [hasResolved, isLoading, isRefreshing, pageData, pageLoadError]);

	const loadPluginAgents = useCallback(
		async (plugin: PluginListItem) => {
			const key = agentPluginKey(plugin);
			const epoch = agentLoadEpochRef.current;
			const requestID = (agentLoadRequestIDRef.current[key] ?? 0) + 1;
			agentLoadRequestIDRef.current[key] = requestID;

			setPageData(previous => ({
				...previous,
				plugins: previous.plugins.map(value =>
					agentPluginKey(value.plugin) === key
						? {
								...value,
								isLoadingAgents: true,
								agentLoadError: undefined,
							}
						: value
				),
			}));

			let result: Pick<AgentPluginData, 'agents' | 'agentsLoaded' | 'agentLoadError'>;

			try {
				result = await agentManagementAPI.loadPluginAgents(plugin, new AbortController().signal);
			} catch (error) {
				result = {
					agents: [],
					agentsLoaded: false,
					agentLoadError: getErrorMessage(error, 'Agents could not be loaded for this Plugin.'),
				};
			}

			if (
				!mountedRef.current ||
				agentLoadEpochRef.current !== epoch ||
				agentLoadRequestIDRef.current[key] !== requestID
			) {
				return;
			}

			setPageData(previous => ({
				...previous,
				plugins: previous.plugins.map(value =>
					agentPluginKey(value.plugin) === key
						? {
								...value,
								...result,
								isLoadingAgents: false,
							}
						: value
				),
			}));
		},
		[setPageData]
	);

	const refreshPage = useCallback(async () => {
		invalidateAgentManagementPageDataCache();
		agentManagementAPI.invalidateAgentCatalog();
		agentLoadEpochRef.current += 1;
		agentLoadRequestIDRef.current = {};

		try {
			await reloadOrThrow();
		} catch (error) {
			showAlert(getErrorMessage(error, 'Agent management data could not be refreshed.'));
		}
	}, [reloadOrThrow, showAlert]);

	const updatePluginSummary = useCallback(
		(updatedView: PluginView) => {
			const updated = pluginListItemFromPluginView(updatedView, false);
			const key = pluginRefKey(updated);

			setPageData(previous => ({
				...previous,
				plugins: previous.plugins.map(data =>
					pluginRefKey(data.plugin) === key
						? {
								...data,
								plugin: updated,
							}
						: data
				),
			}));

			setPluginDetails(previous => {
				if (!previous || pluginRefKey(previous.plugin) !== key) {
					return previous;
				}

				return {
					...previous,
					plugin: updated,
					view: updatedView,
					error: undefined,
					loading: false,
				};
			});
		},
		[setPageData]
	);

	const openPluginDetails = useCallback((plugin: PluginListItem) => {
		const requestID = pluginDetailsRequestIDRef.current + 1;
		pluginDetailsRequestIDRef.current = requestID;

		setPluginDetails({
			plugin,
			loading: true,
		});

		void agentStoreAPI
			.getAgentPlugin(plugin.ref)
			.then(view => {
				if (!mountedRef.current || pluginDetailsRequestIDRef.current !== requestID) {
					return;
				}

				setPluginDetails({
					plugin: pluginListItemFromPluginView(view, plugin.builtIn),
					view,
					loading: false,
				});
			})
			.catch((error: unknown) => {
				if (!mountedRef.current || pluginDetailsRequestIDRef.current !== requestID) {
					return;
				}

				setPluginDetails({
					plugin,
					loading: false,
					error: getErrorMessage(error, 'Agent Plugin details could not be loaded.'),
				});
			});
	}, []);

	const togglePluginEnabled = useCallback(
		async (plugin: PluginListItem, enabled: boolean) => {
			const updated = await agentStoreAPI.setAgentPluginEnabled(plugin.ref, plugin.revision, enabled);

			updatePluginSummary(updated);
			agentManagementAPI.invalidateAgentCatalog();
		},
		[updatePluginSummary]
	);

	const toggleAgentEnabled = useCallback(
		async (agent: AgentView, enabled: boolean) => {
			const updated = await agentStoreAPI.setAgentEnabled(agentArtifactRef(agent), agent.revision, enabled);
			const key = agentRefKey(updated);

			setPageData(previous => ({
				...previous,
				plugins: previous.plugins.map(plugin => ({
					...plugin,
					agents: plugin.agents.map(candidate => (agentRefKey(candidate) === key ? updated : candidate)),
				})),
			}));

			setAgentToView(previous => (previous && agentRefKey(previous) === key ? updated : previous));
			agentManagementAPI.invalidateAgentCatalog();
		},
		[setPageData]
	);

	const createPlugin = useCallback(
		async (slug: string, displayName: string, description?: string) => {
			await agentStoreAPI.createAgentPlugin({
				rootID: effectivePluginCreationRootID,
				name: slug,
				displayName,
				description,
			});

			await refreshPage();
		},
		[effectivePluginCreationRootID, refreshPage]
	);

	const updatePlugin = useCallback(
		async (plugin: PluginListItem, displayName: string, description?: string) => {
			const updated = await agentStoreAPI.updateAgentPlugin({
				plugin: plugin.ref,
				expectedRevision: plugin.revision,
				displayName,
				description,
			});

			updatePluginSummary(updated);
			agentManagementAPI.invalidateAgentCatalog();
		},
		[updatePluginSummary]
	);

	const deleteAgent = useCallback(async () => {
		if (!agentToDelete || isDeletingAgent) {
			return;
		}

		setIsDeletingAgent(true);

		try {
			await agentStoreAPI.deleteManagedAgent(agentArtifactRef(agentToDelete), agentToDelete.revision);
			setAgentToDelete(null);
			await refreshPage();
		} catch (error) {
			showAlert(getErrorMessage(error, 'Failed to delete Agent.'));
		} finally {
			if (mountedRef.current) {
				setIsDeletingAgent(false);
			}
		}
	}, [agentToDelete, isDeletingAgent, refreshPage, showAlert]);

	const deletePlugin = useCallback(async () => {
		if (!pluginToDelete || isDeletingPlugin) {
			return;
		}

		const pluginData = pageData.plugins.find(value => pluginRefKey(value.plugin) === pluginRefKey(pluginToDelete));

		if (!canDeleteAgentPlugin(pluginToDelete)) {
			showAlert('This Agent Plugin cannot be deleted.');
			setPluginToDelete(null);
			return;
		}

		if (!pluginData?.agentsLoaded || pluginData.agentLoadError || pluginData.agents.length > 0) {
			showAlert('Load the Plugin and remove all resolved Agents before deleting it.');
			setPluginToDelete(null);
			return;
		}

		setIsDeletingPlugin(true);

		try {
			await agentStoreAPI.deleteAgentPlugin(pluginToDelete.ref, pluginToDelete.revision);
			setPluginToDelete(null);
			await refreshPage();
		} catch (error) {
			showAlert(
				getErrorMessage(
					error,
					'Failed to delete Agent Plugin. It may still contain unresolved or dangling Agent memberships.'
				)
			);
		} finally {
			if (mountedRef.current) {
				setIsDeletingPlugin(false);
			}
		}
	}, [pluginToDelete, isDeletingPlugin, pageData.plugins, refreshPage, showAlert]);

	const exportAgent = useCallback(async (agent: AgentView) => {
		const exported = await agentStoreAPI.exportAgent(agentArtifactRef(agent));

		await backendAPI.saveFile(exported.suggestedFileName, textToBase64(exported.content), [
			{
				DisplayName: 'YAML',
				Extensions: ['yaml', 'yml'],
			},
		]);
	}, []);

	const importCommitted = useCallback(
		async (result: AgentImportCommitResult) => {
			await refreshPage();
			setAgentToView(result.agent);
		},
		[refreshPage]
	);

	const closePluginDetails = useCallback(() => {
		pluginDetailsRequestIDRef.current += 1;
		setPluginDetails(null);
	}, []);

	if (isLoading && !hasResolved) {
		return <Loader text="Loading Agent Plugins..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Agent Plugins"
					description="Import immutable Agent recipes, inspect their resolved capabilities, configure MCP dependencies, and apply them as Composer starters."
					actions={
						<>
							{pluginCreationRoots.length > 1 ? (
								<select
									className="select select-sm max-w-72 rounded-xl"
									aria-label="Agent Plugin Root"
									value={effectivePluginCreationRootID}
									onChange={event => {
										setPluginCreationRootID(event.currentTarget.value);
									}}
								>
									{pluginCreationRoots.map(value => (
										<option key={value.rootID} value={value.rootID}>
											{value.label}
										</option>
									))}
								</select>
							) : null}

							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								disabled={isRefreshing}
								onClick={() => {
									void refreshPage();
								}}
							>
								<FiRefreshCw className={isRefreshing ? 'animate-spin' : undefined} size={18} />
								<span>{isRefreshing ? 'Refreshing' : 'Refresh'}</span>
							</button>

							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								disabled={pageData.importDestinations.length === 0}
								onClick={() => {
									setImportDestination(null);
									setIsImportModalOpen(true);
								}}
							>
								<FiUpload size={18} />
								<span>Import Agent</span>
							</button>

							<button
								type="button"
								className="btn btn-ghost rounded-xl"
								onClick={() => {
									setIsCreatePluginOpen(true);
								}}
							>
								<FiPlus size={18} />
								<span>Add Plugin</span>
							</button>
						</>
					}
				/>

				<ManagementPageContent>
					{pageLoadError ? (
						<ManagementResourceError
							title="Agent Plugins could not be loaded"
							error={pageLoadError}
							isRetrying={isRefreshing}
							onRetry={refreshPage}
						/>
					) : null}

					{pageData.plugins.length === 0 && !pageLoadError ? (
						<p className="mt-8 text-center text-sm">No Agent Plugins are currently available.</p>
					) : null}

					{pageData.plugins.map(data => (
						<AgentPluginCard
							key={agentPluginKey(data.plugin)}
							data={data}
							onLoadAgents={loadPluginAgents}
							onViewPlugin={openPluginDetails}
							onEditPlugin={setPluginToEdit}
							onDeletePlugin={setPluginToDelete}
							onImportAgent={destination => {
								setImportDestination(destination);
								setIsImportModalOpen(true);
							}}
							onTogglePluginEnabled={togglePluginEnabled}
							onViewAgent={setAgentToView}
							onExportAgent={exportAgent}
							onDeleteAgent={setAgentToDelete}
							onToggleAgentEnabled={toggleAgentEnabled}
							onActionError={showAlert}
						/>
					))}
				</ManagementPageContent>

				<ManagementBundleCreateModal
					isOpen={isCreatePluginOpen}
					title="Add Agent Plugin"
					entityLabel="Agent Plugin"
					onClose={() => {
						setIsCreatePluginOpen(false);
					}}
					onSubmit={createPlugin}
					existingSlugs={pageData.plugins
						.filter(data => data.plugin.ref.rootID === effectivePluginCreationRootID)
						.map(data => data.plugin.name)}
					existingDisplayNames={pageData.plugins
						.filter(data => data.plugin.ref.rootID === effectivePluginCreationRootID)
						.map(data => pluginDisplayName(data.plugin))}
					failureMessage="Failed to create Agent Plugin."
				/>

				<AgentImportModal
					isOpen={isImportModalOpen}
					destinations={pageData.importDestinations}
					initialDestination={importDestination}
					onClose={() => {
						setIsImportModalOpen(false);
						setImportDestination(null);
					}}
					onCommitted={importCommitted}
				/>

				<AgentDetailsModal
					isOpen={agentToView !== null}
					agent={agentToView}
					onClose={() => {
						setAgentToView(null);
					}}
				/>

				<AgentPluginEditModal
					plugin={pluginToEdit}
					onClose={() => {
						setPluginToEdit(null);
					}}
					onSubmit={updatePlugin}
				/>

				<AgentPluginDetailsModal state={pluginDetails} onClose={closePluginDetails} />

				<DeleteConfirmationModal
					isOpen={agentToDelete !== null}
					onClose={() => {
						if (!isDeletingAgent) {
							setAgentToDelete(null);
						}
					}}
					onConfirm={deleteAgent}
					title="Delete Managed Agent"
					message={`Delete "${agentToDelete ? agentDisplayName(agentToDelete) : ''}"? Existing Plugin relationships remain declared and become unavailable until an exact Agent is restored.`}
					confirmButtonText={isDeletingAgent ? 'Deleting...' : 'Delete'}
				/>

				<DeleteConfirmationModal
					isOpen={pluginToDelete !== null}
					onClose={() => {
						if (!isDeletingPlugin) {
							setPluginToDelete(null);
						}
					}}
					onConfirm={deletePlugin}
					title="Delete Agent Plugin"
					message={`Delete empty Agent Plugin "${pluginToDelete ? pluginDisplayName(pluginToDelete) : ''}"?`}
					confirmButtonText={isDeletingPlugin ? 'Deleting...' : 'Delete'}
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
