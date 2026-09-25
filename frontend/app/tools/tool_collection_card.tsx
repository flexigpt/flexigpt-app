import { useState } from 'react';
import { FiChevronDown, FiChevronUp, FiRefreshCw } from 'react-icons/fi';

import type { CollectionView } from '@/spec/collection';
import type { ToolView } from '@/spec/tool';
import { ToolImplType } from '@/spec/tool';

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
	collection: CollectionView;
	tools: ToolView[];
	toolLoadError?: string;
	onRefreshTools: () => Promise<void>;
	onToggleCollectionEnable: (collection: CollectionView, enabled: boolean) => Promise<void>;
	onToggleToolEnable: (collection: CollectionView, tool: ToolView, enabled: boolean) => Promise<void>;
}

export function ToolCollectionCard({
	collection,
	tools,
	toolLoadError,
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

	return (
		<>
			<ManagementCollectionCard
				title={toolCollectionDisplayName(collection)}
				identity={<span className="font-mono">{collection.name}</span>}
				description={collection.description}
				status={
					<>
						<StatusBadge tone={collection.artifact.enabled ? 'success' : 'neutral'}>
							{collection.artifact.enabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>Built-in</StatusBadge>
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={expanded}
						onClick={() => {
							setExpanded(previous => !previous);
						}}
					>
						<span>Tools: {collection.members.length}</span>
						{expanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`tool-collection-${collection.artifact.id}`}
						checked={collection.artifact.enabled}
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

				{expanded ? (
					<div className="mt-6 space-y-3">
						{tools.map(tool => {
							const implementation = tool.implementation;
							const toggleKey = `${tool.artifact.id}:toggle`;
							const effectiveEnabled = collection.artifact.enabled && tool.artifact.enabled;

							return (
								<ManagementItemCard
									key={tool.artifact.id}
									title={toolDisplayName(tool)}
									subtitle={`${tool.name} / version ${tool.version}`}
									description={tool.description}
									status={
										<>
											<StatusBadge tone={effectiveEnabled ? 'success' : 'neutral'}>
												{effectiveEnabled ? 'Enabled' : tool.artifact.enabled ? 'Collection disabled' : 'Disabled'}
											</StatusBadge>
											<StatusBadge>Built-in</StatusBadge>
										</>
									}
									metadata={
										<>
											<MetadataPill label="Implementation">
												{implementation.kind === ToolImplType.Go ? 'Go' : 'Provider API'}
											</MetadataPill>
											<MetadataPill label="Execution">
												{implementation.kind === ToolImplType.Go ? 'Local runtime' : 'Provider inference'}
											</MetadataPill>
											{implementation.kind === ToolImplType.SDK ? (
												<MetadataPill label="SDK">{implementation.sdkType}</MetadataPill>
											) : (
												<MetadataPill label="Auto-execute default">{tool.autoExecute ? 'Yes' : 'No'}</MetadataPill>
											)}
										</>
									}
								>
									<div className="mt-4 flex flex-wrap items-center gap-3">
										<EnabledControl
											id={`tool-${tool.artifact.id}`}
											checked={tool.artifact.enabled}
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

						{tools.length === 0 && !toolLoadError ? (
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
