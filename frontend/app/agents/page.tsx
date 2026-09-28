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
import type { CollectionListItem, CollectionView } from '@/spec/collection';
import { ArtifactState } from '@/spec/artifact';
import { collectionListItemFromCollectionView } from '@/spec/collection';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';
import { usePendingActions } from '@/hooks/use_pending_actions';

import type { AgentCollectionData, AgentManagementPageData } from '@/apis/agent_management';
import {
	agentArtifactRef,
	agentCollectionKey,
	agentDisplayName,
	canDeleteAgentCollection,
	canEditAgentCollectionMetadata,
	collectionDisplayName,
	EMPTY_AGENT_MANAGEMENT_PAGE_DATA,
	isBuiltInAgentCollection,
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

interface CollectionDetailsState {
	collection: CollectionListItem;
	view?: CollectionView;
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

function collectionRefKey(collection: CollectionListItem): string {
	return `${collection.ref.rootID}:${collection.ref.artifactID}`;
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

interface AgentCollectionCardProps {
	data: AgentCollectionData;
	onLoadAgents: (collection: CollectionListItem) => Promise<void>;
	onViewCollection: (collection: CollectionListItem) => void;
	onEditCollection: (collection: CollectionListItem) => void;
	onDeleteCollection: (collection: CollectionListItem) => void;
	onImportAgent: (destination: AgentImportDestination) => void;
	onToggleCollectionEnabled: (collection: CollectionListItem, enabled: boolean) => Promise<void>;
	onViewAgent: (agent: AgentView) => void;
	onExportAgent: (agent: AgentView) => Promise<void>;
	onDeleteAgent: (agent: AgentView) => void;
	onToggleAgentEnabled: (agent: AgentView, enabled: boolean) => Promise<void>;
	onActionError: (message: string) => void;
}

function AgentCollectionCard({
	data,
	onLoadAgents,
	onViewCollection,
	onEditCollection,
	onDeleteCollection,
	onImportAgent,
	onToggleCollectionEnabled,
	onViewAgent,
	onExportAgent,
	onDeleteAgent,
	onToggleAgentEnabled,
	onActionError,
}: AgentCollectionCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const { isPending, runAction } = usePendingActions();

	const collection = data.collection;
	const builtIn = isBuiltInAgentCollection(collection);
	const displayName = collectionDisplayName(collection);
	const secondaryName = displayName === collection.name ? undefined : collection.name;
	const collectionAvailable = collection.state === ArtifactState.Available;

	const run = (key: string, action: () => Promise<void>, fallback: string) => {
		void runAction(key, action).catch((error: unknown) => {
			onActionError(getErrorMessage(error, fallback));
		});
	};

	const loadAgents = () => {
		if (data.agentsLoaded || data.isLoadingAgents) {
			return;
		}

		void onLoadAgents(collection).catch((error: unknown) => {
			onActionError(getErrorMessage(error, 'Agents could not be loaded for this Collection.'));
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
			description={collection.description}
			status={
				<>
					<StatusBadge tone={collection.enabled ? 'success' : 'neutral'}>
						{collection.enabled ? 'Enabled' : 'Disabled'}
					</StatusBadge>
					{collection.state !== ArtifactState.Available ? (
						<StatusBadge tone="warning">{collection.state}</StatusBadge>
					) : null}
					{builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
					{collection.baseline ? <StatusBadge>Baseline</StatusBadge> : null}
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
								: `Agents: ${collection.memberCount}`}
					</span>
					{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
				</button>
			}
			actionLeading={
				<EnabledControl
					id={`agent-collection-${collection.ref.rootID}-${collection.ref.artifactID}`}
					checked={collection.enabled}
					disabled={!collectionAvailable || isPending('collection:toggle')}
					busy={isPending('collection:toggle')}
					title={!collectionAvailable ? 'This Agent Collection is unavailable.' : undefined}
					onChange={enabled => {
						run(
							'collection:toggle',
							() => onToggleCollectionEnabled(collection, enabled),
							'Failed to update Agent Collection enablement.'
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
							onViewCollection(collection);
						}}
					>
						<FiEye size={16} />
						<span>Details</span>
					</button>

					{canEditAgentCollectionMetadata(collection) ? (
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								onEditCollection(collection);
							}}
						>
							<FiEdit2 size={16} />
							<span>Edit Collection</span>
						</button>
					) : null}

					{data.importDestination !== undefined && data.importDestination !== null ? (
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={!collectionAvailable}
							onClick={() => {
								onImportAgent(data.importDestination as AgentImportDestination);
							}}
						>
							<FiUpload size={16} />
							<span>Import Agent</span>
						</button>
					) : null}

					{canDeleteAgentCollection(collection) ? (
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							disabled={!data.agentsLoaded || data.agents.length > 0 || Boolean(data.agentLoadError)}
							title={
								!data.agentsLoaded
									? 'Load Collection Agents before deletion.'
									: data.agentLoadError
										? 'Reload Collection Agents before deletion.'
										: data.agents.length > 0
											? 'Remove all Agents before deleting this Collection.'
											: undefined
							}
							onClick={() => {
								onDeleteCollection(collection);
							}}
						>
							<FiTrash2 size={16} />
							<span>Delete Collection</span>
						</button>
					) : null}
				</>
			}
		>
			{data.agentLoadError ? (
				<div className="alert alert-warning mt-3 rounded-2xl text-sm">
					<div className="grow">
						<div className="font-semibold">Agents could not be loaded for this Collection</div>
						<div>{data.agentLoadError}</div>
					</div>
					<button
						type="button"
						className="btn btn-sm rounded-xl"
						disabled={data.isLoadingAgents}
						onClick={() => {
							void onLoadAgents(collection).catch((error: unknown) => {
								onActionError(getErrorMessage(error, 'Agents could not be loaded for this Collection.'));
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
								? 'Loading Agents in this Collection...'
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
						<ManagementEmptyState>No resolved Agents are currently available in this Collection.</ManagementEmptyState>
					) : null}
				</div>
			) : null}
		</ManagementBundleCard>
	);
}

function AgentCollectionEditModalContent({
	collection,
	onClose,
	onSubmit,
}: {
	collection: CollectionListItem;
	onClose: () => void;
	onSubmit: (displayName: string, description?: string) => Promise<void>;
}) {
	const [displayName, setDisplayName] = useState(collection.displayName);
	const [description, setDescription] = useState(collection.description ?? '');
	const [error, setError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);

	const submit = async () => {
		const nextDisplayName = displayName.trim();

		if (!nextDisplayName) {
			setError('Collection display name is required.');
			return;
		}

		setIsSubmitting(true);
		setError('');

		try {
			await onSubmit(nextDisplayName, description.trim() || undefined);
			onClose();
		} catch (submitError) {
			setError(getErrorMessage(submitError, 'Could not update the Agent Collection.'));
		} finally {
			setIsSubmitting(false);
		}
	};

	return (
		<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-xl rounded-2xl p-0">
			<ModalHeader
				title="Edit Agent Collection"
				description="The logical Collection name remains stable. Imported Agents remain immutable."
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

				<ModalSection title="Collection details">
					<ModalField label="Collection name" htmlFor="agent-collection-name">
						<input
							id="agent-collection-name"
							className="input w-full rounded-xl font-mono"
							value={collection.name}
							readOnly
						/>
					</ModalField>

					<ModalField label="Display name" htmlFor="agent-collection-display-name" required>
						<input
							id="agent-collection-display-name"
							className="input w-full rounded-xl"
							value={displayName}
							onChange={event => {
								setDisplayName(event.currentTarget.value);
							}}
							disabled={isSubmitting}
							maxLength={256}
						/>
					</ModalField>

					<ModalField label="Description" htmlFor="agent-collection-description">
						<textarea
							id="agent-collection-description"
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
						{isSubmitting ? 'Saving...' : 'Save Collection'}
					</button>
				</ModalActions>
			</form>
		</div>
	);
}

function AgentCollectionEditModal({
	collection,
	onClose,
	onSubmit,
}: {
	collection: CollectionListItem | null;
	onClose: () => void;
	onSubmit: (collection: CollectionListItem, displayName: string, description?: string) => Promise<void>;
}) {
	if (!collection) {
		return null;
	}

	return (
		<ModalDialog isOpen={true} onClose={onClose} blockCancel>
			<AgentCollectionEditModalContent
				key={`${collection.ref.rootID}:${collection.ref.artifactID}:${collection.revision}`}
				collection={collection}
				onClose={onClose}
				onSubmit={(displayName, description) => onSubmit(collection, displayName, description)}
			/>
		</ModalDialog>
	);
}

function AgentCollectionDetailsModal({
	state,
	onClose,
}: {
	state: CollectionDetailsState | null;
	onClose: () => void;
}) {
	if (!state) {
		return null;
	}

	const collection = state.view ?? state.collection;
	const modalKey = state.view
		? `agent-collection:${state.view.artifact.rootID}:${state.view.artifact.id}:${state.view.artifact.revision}`
		: `agent-collection:${state.collection.ref.rootID}:${state.collection.ref.artifactID}:${state.collection.revision}`;

	return (
		<ManagementDetailsModal isOpen={true} onClose={onClose} title="Agent Collection Details" modalKey={modalKey}>
			{state.loading ? <Loader text="Loading Agent Collection details..." /> : null}

			{state.error ? (
				<ManagementResourceError
					title="Agent Collection details could not be loaded"
					error={state.error}
					isRetrying={false}
					onRetry={async () => undefined}
				/>
			) : null}

			<ManagementInfoGrid>
				<ManagementInfoRow label="Display Name">{collectionDisplayName(collection)}</ManagementInfoRow>
				<ManagementInfoRow label="Name" mono>
					{collection.name}
				</ManagementInfoRow>
				<ManagementInfoRow label="Built-in">{isBuiltInAgentCollection(collection) ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Baseline">{collection.baseline ? 'Yes' : 'No'}</ManagementInfoRow>
				<ManagementInfoRow label="Enabled">
					{'artifact' in collection ? (collection.artifact.enabled ? 'Yes' : 'No') : collection.enabled ? 'Yes' : 'No'}
				</ManagementInfoRow>
				<ManagementInfoRow label="State">
					{'artifact' in collection ? collection.artifact.state : collection.state}
				</ManagementInfoRow>
				<ManagementInfoRow label="Revision">
					{'artifact' in collection ? collection.artifact.revision : collection.revision}
				</ManagementInfoRow>
				{'artifact' in collection ? (
					<>
						<ManagementInfoRow label="Source" mono>
							{collection.artifact.binding.sourceID}
						</ManagementInfoRow>
						<ManagementInfoRow label="Created">{formatDateish(collection.artifact.createdAt)}</ManagementInfoRow>
						<ManagementInfoRow label="Modified">{formatDateish(collection.artifact.modifiedAt)}</ManagementInfoRow>
					</>
				) : (
					<>
						<ManagementInfoRow label="Source" mono>
							{collection.sourceID}
						</ManagementInfoRow>
						<ManagementInfoRow label="Declared members">{collection.memberCount}</ManagementInfoRow>
					</>
				)}
				<ManagementInfoRow label="Description">
					<span className="whitespace-pre-wrap">{collection.description || '—'}</span>
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

	const [isCreateCollectionOpen, setIsCreateCollectionOpen] = useState(false);
	const [isImportModalOpen, setIsImportModalOpen] = useState(false);
	const [importDestination, setImportDestination] = useState<AgentImportDestination | null>(null);
	const [agentToView, setAgentToView] = useState<AgentView | null>(null);
	const [agentToDelete, setAgentToDelete] = useState<AgentView | null>(null);
	const [collectionToDelete, setCollectionToDelete] = useState<CollectionListItem | null>(null);
	const [collectionToEdit, setCollectionToEdit] = useState<CollectionListItem | null>(null);
	const [collectionDetails, setCollectionDetails] = useState<CollectionDetailsState | null>(null);
	const [isDeletingAgent, setIsDeletingAgent] = useState(false);
	const [isDeletingCollection, setIsDeletingCollection] = useState(false);
	const [alertMessage, setAlertMessage] = useState('');
	const [collectionCreationRootID, setCollectionCreationRootID] = useState('');

	const mountedRef = useRef(false);
	const agentLoadRequestIDRef = useRef<Record<string, number>>({});
	const agentLoadEpochRef = useRef(0);
	const collectionDetailsRequestIDRef = useRef(0);

	const collectionCreationRoots = useMemo(
		() =>
			[
				...new Map(
					pageData.collections
						.map(value => value.collection)
						.filter(collection => collection.baseline && !isBuiltInAgentCollection(collection))
						.map(
							collection =>
								[
									collection.ref.rootID,
									{
										rootID: collection.ref.rootID,
										label: collectionDisplayName(collection),
									},
								] as const
						)
				).values(),
			].toSorted((left, right) => left.label.localeCompare(right.label)),
		[pageData.collections]
	);

	const effectiveCollectionCreationRootID = collectionCreationRoots.some(
		value => value.rootID === collectionCreationRootID
	)
		? collectionCreationRootID
		: (collectionCreationRoots[0]?.rootID ?? '');

	const showAlert = useCallback((message: string) => {
		setAlertMessage(message);
	}, []);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
			agentLoadRequestIDRef.current = {};
			agentLoadEpochRef.current += 1;
			collectionDetailsRequestIDRef.current += 1;
		};
	}, []);

	useEffect(() => {
		if (
			hasResolved &&
			!pageLoadError &&
			!isLoading &&
			!isRefreshing &&
			pageData.collections.every(collection => !collection.isLoadingAgents)
		) {
			rememberAgentManagementPageData(pageData);
		}
	}, [hasResolved, isLoading, isRefreshing, pageData, pageLoadError]);

	const loadCollectionAgents = useCallback(
		async (collection: CollectionListItem) => {
			const key = agentCollectionKey(collection);
			const epoch = agentLoadEpochRef.current;
			const requestID = (agentLoadRequestIDRef.current[key] ?? 0) + 1;
			agentLoadRequestIDRef.current[key] = requestID;

			setPageData(previous => ({
				...previous,
				collections: previous.collections.map(value =>
					agentCollectionKey(value.collection) === key
						? {
								...value,
								isLoadingAgents: true,
								agentLoadError: undefined,
							}
						: value
				),
			}));

			let result: Pick<AgentCollectionData, 'agents' | 'agentsLoaded' | 'agentLoadError'>;

			try {
				result = await agentManagementAPI.loadCollectionAgents(collection, new AbortController().signal);
			} catch (error) {
				result = {
					agents: [],
					agentsLoaded: false,
					agentLoadError: getErrorMessage(error, 'Agents could not be loaded for this Collection.'),
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
				collections: previous.collections.map(value =>
					agentCollectionKey(value.collection) === key
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

	const updateCollectionSummary = useCallback(
		(updatedView: CollectionView) => {
			const updated = collectionListItemFromCollectionView(updatedView);
			const key = collectionRefKey(updated);

			setPageData(previous => ({
				...previous,
				collections: previous.collections.map(data =>
					collectionRefKey(data.collection) === key
						? {
								...data,
								collection: updated,
							}
						: data
				),
			}));

			setCollectionDetails(previous => {
				if (!previous || collectionRefKey(previous.collection) !== key) {
					return previous;
				}

				return {
					...previous,
					collection: updated,
					view: updatedView,
					error: undefined,
					loading: false,
				};
			});
		},
		[setPageData]
	);

	const openCollectionDetails = useCallback((collection: CollectionListItem) => {
		const requestID = collectionDetailsRequestIDRef.current + 1;
		collectionDetailsRequestIDRef.current = requestID;

		setCollectionDetails({
			collection,
			loading: true,
		});

		void agentStoreAPI
			.getAgentCollection(collection.ref)
			.then(view => {
				if (!mountedRef.current || collectionDetailsRequestIDRef.current !== requestID) {
					return;
				}

				setCollectionDetails({
					collection: collectionListItemFromCollectionView(view),
					view,
					loading: false,
				});
			})
			.catch((error: unknown) => {
				if (!mountedRef.current || collectionDetailsRequestIDRef.current !== requestID) {
					return;
				}

				setCollectionDetails({
					collection,
					loading: false,
					error: getErrorMessage(error, 'Agent Collection details could not be loaded.'),
				});
			});
	}, []);

	const toggleCollectionEnabled = useCallback(
		async (collection: CollectionListItem, enabled: boolean) => {
			const updated = await agentStoreAPI.setAgentCollectionEnabled(collection.ref, collection.revision, enabled);

			updateCollectionSummary(updated);
			agentManagementAPI.invalidateAgentCatalog();
		},
		[updateCollectionSummary]
	);

	const toggleAgentEnabled = useCallback(
		async (agent: AgentView, enabled: boolean) => {
			const updated = await agentStoreAPI.setAgentEnabled(agentArtifactRef(agent), agent.revision, enabled);
			const key = agentRefKey(updated);

			setPageData(previous => ({
				...previous,
				collections: previous.collections.map(collection => ({
					...collection,
					agents: collection.agents.map(candidate => (agentRefKey(candidate) === key ? updated : candidate)),
				})),
			}));

			setAgentToView(previous => (previous && agentRefKey(previous) === key ? updated : previous));
			agentManagementAPI.invalidateAgentCatalog();
		},
		[setPageData]
	);

	const createCollection = useCallback(
		async (slug: string, displayName: string, description?: string) => {
			await agentStoreAPI.createAgentCollection({
				rootID: effectiveCollectionCreationRootID,
				name: slug,
				displayName,
				description,
			});

			await refreshPage();
		},
		[effectiveCollectionCreationRootID, refreshPage]
	);

	const updateCollection = useCallback(
		async (collection: CollectionListItem, displayName: string, description?: string) => {
			const updated = await agentStoreAPI.updateAgentCollection({
				collection: collection.ref,
				expectedRevision: collection.revision,
				displayName,
				description,
			});

			updateCollectionSummary(updated);
			agentManagementAPI.invalidateAgentCatalog();
		},
		[updateCollectionSummary]
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

	const deleteCollection = useCallback(async () => {
		if (!collectionToDelete || isDeletingCollection) {
			return;
		}

		const collectionData = pageData.collections.find(
			value => collectionRefKey(value.collection) === collectionRefKey(collectionToDelete)
		);

		if (!canDeleteAgentCollection(collectionToDelete)) {
			showAlert('This Agent Collection cannot be deleted.');
			setCollectionToDelete(null);
			return;
		}

		if (!collectionData?.agentsLoaded || collectionData.agentLoadError || collectionData.agents.length > 0) {
			showAlert('Load the Collection and remove all resolved Agents before deleting it.');
			setCollectionToDelete(null);
			return;
		}

		setIsDeletingCollection(true);

		try {
			await agentStoreAPI.deleteAgentCollection(collectionToDelete.ref, collectionToDelete.revision);
			setCollectionToDelete(null);
			await refreshPage();
		} catch (error) {
			showAlert(
				getErrorMessage(
					error,
					'Failed to delete Agent Collection. It may still contain unresolved or dangling Agent memberships.'
				)
			);
		} finally {
			if (mountedRef.current) {
				setIsDeletingCollection(false);
			}
		}
	}, [collectionToDelete, isDeletingCollection, pageData.collections, refreshPage, showAlert]);

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

	const closeCollectionDetails = useCallback(() => {
		collectionDetailsRequestIDRef.current += 1;
		setCollectionDetails(null);
	}, []);

	if (isLoading && !hasResolved) {
		return <Loader text="Loading Agent Collections..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Agent Collections"
					description="Import immutable Agent recipes, inspect their resolved capabilities, configure MCP dependencies, and apply them as Composer starters."
					actions={
						<>
							{collectionCreationRoots.length > 1 ? (
								<select
									className="select select-sm max-w-72 rounded-xl"
									aria-label="Agent Collection Root"
									value={effectiveCollectionCreationRootID}
									onChange={event => {
										setCollectionCreationRootID(event.currentTarget.value);
									}}
								>
									{collectionCreationRoots.map(value => (
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
									setIsCreateCollectionOpen(true);
								}}
							>
								<FiPlus size={18} />
								<span>Add Collection</span>
							</button>
						</>
					}
				/>

				<ManagementPageContent>
					{pageLoadError ? (
						<ManagementResourceError
							title="Agent Collections could not be loaded"
							error={pageLoadError}
							isRetrying={isRefreshing}
							onRetry={refreshPage}
						/>
					) : null}

					{pageData.collections.length === 0 && !pageLoadError ? (
						<p className="mt-8 text-center text-sm">No Agent Collections are currently available.</p>
					) : null}

					{pageData.collections.map(data => (
						<AgentCollectionCard
							key={agentCollectionKey(data.collection)}
							data={data}
							onLoadAgents={loadCollectionAgents}
							onViewCollection={openCollectionDetails}
							onEditCollection={setCollectionToEdit}
							onDeleteCollection={setCollectionToDelete}
							onImportAgent={destination => {
								setImportDestination(destination);
								setIsImportModalOpen(true);
							}}
							onToggleCollectionEnabled={toggleCollectionEnabled}
							onViewAgent={setAgentToView}
							onExportAgent={exportAgent}
							onDeleteAgent={setAgentToDelete}
							onToggleAgentEnabled={toggleAgentEnabled}
							onActionError={showAlert}
						/>
					))}
				</ManagementPageContent>

				<ManagementBundleCreateModal
					isOpen={isCreateCollectionOpen}
					title="Add Agent Collection"
					entityLabel="Agent Collection"
					onClose={() => {
						setIsCreateCollectionOpen(false);
					}}
					onSubmit={createCollection}
					existingSlugs={pageData.collections
						.filter(data => data.collection.ref.rootID === effectiveCollectionCreationRootID)
						.map(data => data.collection.name)}
					existingDisplayNames={pageData.collections
						.filter(data => data.collection.ref.rootID === effectiveCollectionCreationRootID)
						.map(data => collectionDisplayName(data.collection))}
					failureMessage="Failed to create Agent Collection."
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

				<AgentCollectionEditModal
					collection={collectionToEdit}
					onClose={() => {
						setCollectionToEdit(null);
					}}
					onSubmit={updateCollection}
				/>

				<AgentCollectionDetailsModal state={collectionDetails} onClose={closeCollectionDetails} />

				<DeleteConfirmationModal
					isOpen={agentToDelete !== null}
					onClose={() => {
						if (!isDeletingAgent) {
							setAgentToDelete(null);
						}
					}}
					onConfirm={deleteAgent}
					title="Delete Managed Agent"
					message={`Delete "${agentToDelete ? agentDisplayName(agentToDelete) : ''}"? Existing Collection relationships remain declared and become unavailable until an exact Agent is restored.`}
					confirmButtonText={isDeletingAgent ? 'Deleting...' : 'Delete'}
				/>

				<DeleteConfirmationModal
					isOpen={collectionToDelete !== null}
					onClose={() => {
						if (!isDeletingCollection) {
							setCollectionToDelete(null);
						}
					}}
					onConfirm={deleteCollection}
					title="Delete Agent Collection"
					message={`Delete empty Agent Collection "${collectionToDelete ? collectionDisplayName(collectionToDelete) : ''}"?`}
					confirmButtonText={isDeletingCollection ? 'Deleting...' : 'Delete'}
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
