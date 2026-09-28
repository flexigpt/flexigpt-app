import { useState } from 'react';
import { FiChevronDown, FiChevronUp, FiRefreshCw } from 'react-icons/fi';

import type { CollectionListItem } from '@/spec/collection';
import type { ToolStoreListItem } from '@/spec/tool';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { toolCollectionDisplayName, toolDisplayName } from '@/apis/tool_management';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard as ManagementCollectionCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

interface ToolCollectionCardProps {
	collection: CollectionListItem;
	tools: ToolStoreListItem[];
	toolsLoaded: boolean;
	isLoadingTools: boolean;
	toolLoadError?: string;
	onLoadTools: () => Promise<void>;
	onRefreshTools: () => Promise<void>;
	onToggleCollectionEnable: (collection: CollectionListItem, enabled: boolean) => Promise<void>;
	onToggleToolEnable: (collection: CollectionListItem, tool: ToolStoreListItem, enabled: boolean) => Promise<void>;
}

export function ToolCollectionCard({
	collection,
	tools,
	toolsLoaded,
	isLoadingTools,
	toolLoadError,
	onLoadTools,
	onRefreshTools,
	onToggleCollectionEnable,
	onToggleToolEnable,
}: ToolCollectionCardProps) {
	const [expanded, setExpanded] = useState(false);
	const [alertMessage, setAlertMessage] = useState('');
	const { isPending, runAction } = usePendingActions();

	const run = async (key: string, action: () => Promise<void>) => {
		try {
			await runAction(key, action);
		} catch (error) {
			setAlertMessage(error instanceof Error ? error.message : 'The tool operation failed.');
		}
	};

	const toggleExpanded = () => {
		const next = !expanded;
		setExpanded(next);
		if (next && !toolsLoaded && !isLoadingTools) {
			void onLoadTools();
		}
	};

	return (
		<>
			<ManagementCollectionCard
				title={toolCollectionDisplayName(collection)}
				identity={<span className="font-mono">{collection.name}</span>}
				description={collection.description}
				status={
					<>
						<StatusBadge tone={collection.enabled ? 'success' : 'neutral'}>
							{collection.enabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						{collection.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={expanded}
						onClick={() => {
							toggleExpanded();
						}}
					>
						<span>{toolsLoaded ? `Tools: ${tools.length}` : `Tools: ${collection.memberCount}`}</span>
						{expanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`tool-collection-${collection.ref.rootID}-${collection.ref.artifactID}`}
						checked={collection.enabled}
						onChange={enabled => {
							void run('collection:toggle', () => onToggleCollectionEnable(collection, enabled));
						}}
						disabled={isPending('collection:toggle')}
						busy={isPending('collection:toggle')}
						compact={false}
					/>
				}
				actions={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						disabled={isPending('collection:refresh')}
						onClick={() => {
							void run('collection:refresh', onRefreshTools);
						}}
					>
						<FiRefreshCw className={isPending('collection:refresh') ? 'animate-spin' : undefined} size={15} />
						<span>Refresh</span>
					</button>
				}
			>
				{toolLoadError ? (
					<output className="alert alert-warning mt-4 rounded-2xl text-sm">
						<span className="min-w-0 grow">
							<span className="block font-semibold">Tools could not be loaded</span>
							<span className="block wrap-break-word">{toolLoadError}</span>
						</span>
						<button
							type="button"
							className="btn btn-sm rounded-xl"
							disabled={isPending('collection:refresh')}
							onClick={() => {
								void run('collection:refresh', onRefreshTools);
							}}
						>
							Retry
						</button>
					</output>
				) : null}

				{expanded && !toolsLoaded && !toolLoadError ? (
					<ManagementEmptyState>
						{isLoadingTools ? 'Loading tools in this Collection...' : 'Tool contents have not been loaded.'}
					</ManagementEmptyState>
				) : null}

				{expanded ? (
					<div className="mt-6 space-y-3">
						{tools.map(tool => {
							const toggleKey = `${tool.ref.artifactID}:toggle`;
							const effectiveEnabled = collection.enabled && tool.enabled;

							return (
								<ManagementItemCard
									key={`${tool.ref.rootID}:${tool.ref.artifactID}`}
									title={toolDisplayName(tool)}
									subtitle={tool.name}
									description={tool.description}
									status={
										<>
											<StatusBadge tone={effectiveEnabled ? 'success' : 'neutral'}>
												{effectiveEnabled ? 'Enabled' : tool.enabled ? 'Collection disabled' : 'Disabled'}
											</StatusBadge>
											{tool.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
										</>
									}
									metadata={
										<>
											<MetadataPill label="State">{tool.state}</MetadataPill>
											{tool.definitionDigest ? <MetadataPill label="Definition">Available</MetadataPill> : null}
										</>
									}
								>
									<div className="mt-4 flex flex-wrap items-center gap-3">
										<EnabledControl
											id={`tool-${tool.ref.rootID}-${tool.ref.artifactID}`}
											checked={tool.enabled}
											onChange={enabled => {
												void run(toggleKey, () => onToggleToolEnable(collection, tool, enabled));
											}}
											disabled={isPending(toggleKey)}
											busy={isPending(toggleKey)}
											title="Change built-in tool availability"
										/>
										<span className="text-base-content/70 text-xs">Built-in tool definitions are read-only.</span>
									</div>
								</ManagementItemCard>
							);
						})}

						{toolsLoaded && tools.length === 0 && !toolLoadError ? (
							<ManagementEmptyState>No tools are registered in this Collection.</ManagementEmptyState>
						) : null}
					</div>
				) : null}
			</ManagementCollectionCard>

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
