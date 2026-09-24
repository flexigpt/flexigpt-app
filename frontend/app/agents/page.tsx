import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiChevronDown, FiChevronUp, FiDownload, FiEdit2, FiEye, FiPlus, FiTrash2, FiUpload } from 'react-icons/fi';

import type { AgentImportCommitResult, AgentImportDestination, AgentView } from '@/spec/agent';
import type { CollectionView } from '@/spec/collection';
import { ArtifactState } from '@/spec/artifact';

import { throwIfAborted } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { useAsyncResource } from '@/hooks/use_async_resource';

import type { AgentCollectionData, AgentManagementPageData } from '@/apis/agent_management';
import {
	agentArtifactRef,
	agentCollectionKey,
	agentCollectionRef,
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

let agentManagementPageDataCache: AgentManagementPageDataCache | undefined;
let agentManagementPageDataLoad: AgentManagementPageDataLoad | undefined;
let agentManagementPageDataCacheGeneration = 0;

function invalidateAgentManagementPageDataCache() {
	agentManagementPageDataCacheGeneration += 1;
	agentManagementPageDataCache = undefined;
}

function rememberAgentManagementPageData(data: AgentManagementPageData) {
	agentManagementPageDataCacheGeneration += 1;
	agentManagementPageDataCache = {
		data,
		loadedAt: Date.now(),
	};
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
				agentManagementPageDataCache = { data, loadedAt: Date.now() };
			}
			return data;
		});
		load = { generation, promise };
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
	onLoadAgents: (collection: CollectionView) => Promise<void>;
	onViewCollection: (collection: CollectionView) => void;
	onEditCollection: (collection: CollectionView) => void;
	onDeleteCollection: (collection: CollectionView) => void;
	onImportAgent: (destination: AgentImportDestination) => void;
	onToggleCollectionEnabled: (collection: CollectionView, enabled: boolean) => Promise<void>;
	onViewAgent: (agent: AgentView) => void;
	onExportAgent: (agent: AgentView) => Promise<void>;
	onDeleteAgent: (agent: AgentView) => void;
	onToggleAgentEnabled: (agent: AgentView, enabled: boolean) => Promise<void>;
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
}: AgentCollectionCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const collection = data.collection;
	const builtIn = isBuiltInAgentCollection(collection);
	const displayName = collectionDisplayName(collection);
	const secondaryName = displayName === collection.name ? undefined : collection.name;

	const loadAgents = () => {
		if (data.agentsLoaded || data.isLoadingAgents) {
			return;
		}
		void onLoadAgents(collection);
	};

	return (
		<ManagementBundleCard
			title={displayName}
			identity={secondaryName ? <span className="font-mono">{secondaryName}</span> : null}
			description={collection.description}
			status={
				<>
					<StatusBadge tone={collection.artifact.enabled ? 'success' : 'neutral'}>
						{collection.artifact.enabled ? 'Enabled' : 'Disabled'}
					</StatusBadge>
					{builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
					{collection.baseline ? <StatusBadge>Baseline</StatusBadge> : null}
				</>
			}
			disclosure={
				<button
					type="button"
					className="btn btn-sm btn-ghost rounded-xl"
					aria-expanded={isExpanded}
					onClick={() => {
						const next = !isExpanded;
						setIsExpanded(next);
						if (next) {
							loadAgents();
						}
					}}
				>
					<span>
						{data.isLoadingAgents
							? 'Loading Agents...'
							: data.agentsLoaded
								? `Agents: ${data.agents.length}`
								: 'Agents: not loaded'}
					</span>
					{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
				</button>
			}
			actionLeading={
				<EnabledControl
					id={`agent-collection-${collection.artifact.rootID}-${collection.artifact.id}`}
					checked={collection.artifact.enabled}
					onChange={() => {
						void onToggleCollectionEnabled(collection, !collection.artifact.enabled);
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

					{data.importDestination ? (
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							onClick={() => {
								if (data.importDestination) {
									onImportAgent(data.importDestination);
								}
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
						onClick={() => {
							void onLoadAgents(collection);
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
									: 'Expand this Collection to load its Agents.'}
						</ManagementEmptyState>
					) : null}

					{data.agentsLoaded
						? data.agents.map(agent => (
								<ManagementItemCard
									key={`${agent.artifact.rootID}:${agent.artifact.id}`}
									title={agentDisplayName(agent)}
									subtitle={agent.name === agentDisplayName(agent) ? undefined : agent.name}
									description={agent.description}
									status={
										<>
											<StatusBadge tone={agent.artifact.enabled ? 'success' : 'neutral'}>
												{agent.artifact.enabled ? 'Enabled' : 'Disabled'}
											</StatusBadge>
											{agent.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
											{agent.managed ? <StatusBadge>Managed</StatusBadge> : null}
										</>
									}
									metadata={<MetadataPill label="State">{agent.artifact.state}</MetadataPill>}
								>
									<ActionRow
										leading={
											<EnabledControl
												id={`agent-${agent.artifact.rootID}-${agent.artifact.id}`}
												checked={agent.artifact.enabled}
												onChange={() => {
													void onToggleAgentEnabled(agent, !agent.artifact.enabled);
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

										{agent.artifact.state === ArtifactState.Available ? (
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													void onExportAgent(agent);
												}}
											>
												<FiDownload size={15} />
												<span>Export</span>
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
							))
						: null}

					{data.agentsLoaded && data.agents.length === 0 ? (
						<ManagementEmptyState>No currently available Agents in this Collection.</ManagementEmptyState>
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
	collection: CollectionView;
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
				description="The logical Collection name remains stable. Agents themselves remain immutable after import."
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
	collection: CollectionView | null;
	onClose: () => void;
	onSubmit: (collection: CollectionView, displayName: string, description?: string) => Promise<void>;
}) {
	if (!collection) {
		return null;
	}

	return (
		<ModalDialog isOpen={true} onClose={onClose}>
			<AgentCollectionEditModalContent
				key={`${collection.artifact.rootID}:${collection.artifact.id}:${collection.artifact.revision}`}
				collection={collection}
				onClose={onClose}
				onSubmit={(displayName, description) => onSubmit(collection, displayName, description)}
			/>
		</ModalDialog>
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
		initialData: agentManagementPageDataCache?.data ?? (EMPTY_AGENT_MANAGEMENT_PAGE_DATA as AgentManagementPageData),
	});

	const [isCreateCollectionOpen, setIsCreateCollectionOpen] = useState(false);
	const [isImportModalOpen, setIsImportModalOpen] = useState(false);
	const [importDestination, setImportDestination] = useState<AgentImportDestination | null>(null);
	const [agentToView, setAgentToView] = useState<AgentView | null>(null);
	const [agentToDelete, setAgentToDelete] = useState<AgentView | null>(null);
	const [collectionToDelete, setCollectionToDelete] = useState<CollectionView | null>(null);
	const [collectionToEdit, setCollectionToEdit] = useState<CollectionView | null>(null);
	const [collectionToView, setCollectionToView] = useState<CollectionView | null>(null);
	const [isDeletingAgent, setIsDeletingAgent] = useState(false);
	const [isDeletingCollection, setIsDeletingCollection] = useState(false);
	const [alertMessage, setAlertMessage] = useState('');
	const [collectionCreationRootID, setCollectionCreationRootID] = useState('');

	const mountedRef = useRef(false);
	const agentLoadRequestIDRef = useRef<Record<string, number>>({});
	const agentLoadEpochRef = useRef(0);
	const agentPrefetchKeysRef = useRef<Set<string>>(new Set());
	const agentCatalogWarmupStartedRef = useRef(false);

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
									collection.artifact.rootID,
									{
										rootID: collection.artifact.rootID,
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

	useEffect(() => {
		const prefetchedKeys = agentPrefetchKeysRef.current;
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
			agentLoadRequestIDRef.current = {};
			agentLoadEpochRef.current += 1;
			prefetchedKeys.clear();
		};
	}, []);

	useEffect(() => {
		if (
			hasResolved &&
			!pageLoadError &&
			!isLoading &&
			!isRefreshing &&
			pageData.collections.every(value => !value.isLoadingAgents)
		) {
			rememberAgentManagementPageData(pageData);
		}
	}, [hasResolved, isLoading, isRefreshing, pageData, pageLoadError]);

	const showAlert = (message: string) => {
		setAlertMessage(message);
	};

	const loadCollectionAgents = useCallback(
		async (collection: CollectionView) => {
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

	useEffect(() => {
		if (!hasResolved || pageLoadError) {
			return;
		}

		if (!agentCatalogWarmupStartedRef.current) {
			agentCatalogWarmupStartedRef.current = true;
			void agentManagementAPI.preloadAgentCatalog().catch(() => undefined);
		}

		const candidates = pageData.collections.filter(value => {
			const key = agentCollectionKey(value.collection);
			return (
				!value.agentsLoaded && !value.isLoadingAgents && !value.agentLoadError && !agentPrefetchKeysRef.current.has(key)
			);
		});
		if (candidates.length === 0) {
			return;
		}

		for (const value of candidates) {
			agentPrefetchKeysRef.current.add(agentCollectionKey(value.collection));
		}

		let cursor = 0;
		const worker = async () => {
			while (cursor < candidates.length) {
				const index = cursor;
				cursor += 1;
				await loadCollectionAgents(candidates[index].collection);
			}
		};

		const workerCount = Math.min(3, candidates.length);
		void Promise.all(Array.from({ length: workerCount }, () => worker())).catch(() => undefined);
	}, [hasResolved, loadCollectionAgents, pageData.collections, pageLoadError]);

	const refreshPage = useCallback(async () => {
		try {
			invalidateAgentManagementPageDataCache();
			agentManagementAPI.invalidateAgentCatalog();
			agentLoadEpochRef.current += 1;
			agentPrefetchKeysRef.current.clear();
			agentCatalogWarmupStartedRef.current = false;
			await reloadOrThrow();
		} catch (error) {
			showAlert(getErrorMessage(error, 'Agent management data could not be refreshed.'));
		}
	}, [reloadOrThrow]);

	const toggleCollectionEnabled = useCallback(
		async (collection: CollectionView, enabled: boolean) => {
			try {
				await agentStoreAPI.setAgentCollectionEnabled(
					agentCollectionRef(collection),
					collection.artifact.revision,
					enabled
				);
				await refreshPage();
			} catch (error) {
				showAlert(getErrorMessage(error, 'Failed to update Agent Collection enablement.'));
			}
		},
		[refreshPage]
	);

	const toggleAgentEnabled = useCallback(
		async (agent: AgentView, enabled: boolean) => {
			try {
				await agentStoreAPI.setAgentEnabled(agentArtifactRef(agent), agent.artifact.revision, enabled);
				await refreshPage();
			} catch (error) {
				showAlert(getErrorMessage(error, 'Failed to update Agent enablement.'));
			}
		},
		[refreshPage]
	);

	const createCollection = useCallback(
		async (slug: string, displayName: string, description?: string) => {
			await agentStoreAPI.createAgentCollection({
				rootID: effectiveCollectionCreationRootID ? effectiveCollectionCreationRootID : '',
				name: slug,
				displayName,
				description,
			});

			await refreshPage();
		},
		[effectiveCollectionCreationRootID, refreshPage]
	);

	const updateCollection = useCallback(
		async (collection: CollectionView, displayName: string, description?: string) => {
			await agentStoreAPI.updateAgentCollection({
				collection: agentCollectionRef(collection),
				expectedRevision: collection.artifact.revision,
				displayName,
				description,
			});

			await refreshPage();
		},
		[refreshPage]
	);

	const deleteAgent = useCallback(async () => {
		if (!agentToDelete || isDeletingAgent) {
			return;
		}

		setIsDeletingAgent(true);

		try {
			await agentStoreAPI.deleteManagedAgent(agentArtifactRef(agentToDelete), agentToDelete.artifact.revision);
			setAgentToDelete(null);
			await refreshPage();
		} catch (error) {
			showAlert(getErrorMessage(error, 'Failed to delete Agent.'));
		} finally {
			setIsDeletingAgent(false);
		}
	}, [agentToDelete, isDeletingAgent, refreshPage]);

	const deleteCollection = useCallback(async () => {
		if (!collectionToDelete || isDeletingCollection) {
			return;
		}

		const collectionData = pageData.collections.find(
			value => agentCollectionKey(value.collection) === agentCollectionKey(collectionToDelete)
		);

		if (!canDeleteAgentCollection(collectionToDelete)) {
			showAlert('This Agent Collection cannot be deleted.');
			setCollectionToDelete(null);
			return;
		}

		if (!collectionData?.agentsLoaded || collectionData.agentLoadError || collectionData.agents.length > 0) {
			showAlert('Reload and remove all currently available Agents before deleting this Collection.');
			setCollectionToDelete(null);
			return;
		}

		setIsDeletingCollection(true);

		try {
			await agentStoreAPI.deleteAgentCollection(
				agentCollectionRef(collectionToDelete),
				collectionToDelete.artifact.revision
			);
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
			setIsDeletingCollection(false);
		}
	}, [collectionToDelete, isDeletingCollection, pageData.collections, refreshPage]);

	const exportAgent = useCallback(async (agent: AgentView) => {
		try {
			const exported = await agentStoreAPI.exportAgent(agentArtifactRef(agent));

			await backendAPI.saveFile(exported.suggestedFileName, textToBase64(exported.content), [
				{
					DisplayName: 'YAML',
					Extensions: ['yaml', 'yml'],
				},
			]);
		} catch (error) {
			showAlert(getErrorMessage(error, 'Failed to export Agent YAML.'));
		}
	}, []);

	const importCommitted = useCallback(
		async (result: AgentImportCommitResult) => {
			await refreshPage();
			setAgentToView(result.agent);
		},
		[refreshPage]
	);

	if (isLoading && !hasResolved) {
		return <Loader text="Loading Agent Collections..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Agent Collections"
					description="Import immutable Agent JSON or YAML recipes, inspect their declarations, configure MCP dependencies, and use them as reusable conversation starters."
					actions={
						<>
							{collectionCreationRoots.length > 1 ? (
								<select
									className="select select-sm max-w-72 rounded-xl"
									aria-label="Agent Collection group"
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
							onRetry={reloadOrThrow}
						/>
					) : null}

					{pageData.collections.length === 0 ? (
						<p className="mt-8 text-center text-sm">No Agent Collections are currently available.</p>
					) : null}

					{pageData.collections.map(data => (
						<AgentCollectionCard
							key={agentCollectionKey(data.collection)}
							data={data}
							onLoadAgents={loadCollectionAgents}
							onViewCollection={setCollectionToView}
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
						.filter(data => data.collection.artifact.rootID === effectiveCollectionCreationRootID)
						.map(data => data.collection.name)}
					existingDisplayNames={pageData.collections
						.filter(data => data.collection.artifact.rootID === effectiveCollectionCreationRootID)
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

				<ManagementDetailsModal
					isOpen={collectionToView !== null}
					onClose={() => {
						setCollectionToView(null);
					}}
					title="Agent Collection Details"
					modalKey={
						collectionToView
							? `agent-collection:${collectionToView.artifact.rootID}:${collectionToView.artifact.id}:${collectionToView.artifact.revision}`
							: 'agent-collection'
					}
				>
					{collectionToView ? (
						<ManagementInfoGrid>
							<ManagementInfoRow label="Display Name">{collectionDisplayName(collectionToView)}</ManagementInfoRow>
							<ManagementInfoRow label="Name" mono>
								{collectionToView.name}
							</ManagementInfoRow>
							<ManagementInfoRow label="Built-in">
								{isBuiltInAgentCollection(collectionToView) ? 'Yes' : 'No'}
							</ManagementInfoRow>
							<ManagementInfoRow label="Baseline">{collectionToView.baseline ? 'Yes' : 'No'}</ManagementInfoRow>
							<ManagementInfoRow label="Enabled">{collectionToView.artifact.enabled ? 'Yes' : 'No'}</ManagementInfoRow>
							<ManagementInfoRow label="Description">
								<span className="whitespace-pre-wrap">{collectionToView.description || '—'}</span>
							</ManagementInfoRow>
							<ManagementInfoRow label="Created">
								{formatDateish(collectionToView.artifact.createdAt)}
							</ManagementInfoRow>
							<ManagementInfoRow label="Modified">
								{formatDateish(collectionToView.artifact.modifiedAt)}
							</ManagementInfoRow>
						</ManagementInfoGrid>
					) : null}
				</ManagementDetailsModal>

				<DeleteConfirmationModal
					isOpen={agentToDelete !== null}
					onClose={() => {
						if (!isDeletingAgent) {
							setAgentToDelete(null);
						}
					}}
					onConfirm={deleteAgent}
					title="Delete Managed Agent"
					message={`Delete "${agentToDelete ? agentDisplayName(agentToDelete) : ''}"? Its Collection memberships remain declared and become unavailable until an exact Agent is restored.`}
					confirmButtonText="Delete"
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
					confirmButtonText="Delete"
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
