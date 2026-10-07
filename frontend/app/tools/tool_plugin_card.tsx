import { useState } from 'react';
import { FiChevronDown, FiChevronUp, FiRefreshCw } from 'react-icons/fi';

import type { PluginListItem } from '@/spec/plugin';
import type { ToolStoreListItem } from '@/spec/tool';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { toolDisplayName, toolPluginDisplayName } from '@/apis/tool_management';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard as ManagementPluginCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

interface ToolPluginCardProps {
	plugin: PluginListItem;
	tools: ToolStoreListItem[];
	toolsLoaded: boolean;
	isLoadingTools: boolean;
	toolLoadError?: string;
	onLoadTools: () => Promise<void>;
	onRefreshTools: () => Promise<void>;
	onTogglePluginEnable: (plugin: PluginListItem, enabled: boolean) => Promise<void>;
	onToggleToolEnable: (plugin: PluginListItem, tool: ToolStoreListItem, enabled: boolean) => Promise<void>;
}

export function ToolPluginCard({
	plugin,
	tools,
	toolsLoaded,
	isLoadingTools,
	toolLoadError,
	onLoadTools,
	onRefreshTools,
	onTogglePluginEnable,
	onToggleToolEnable,
}: ToolPluginCardProps) {
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
			<ManagementPluginCard
				title={toolPluginDisplayName(plugin)}
				identity={<span className="font-mono">{plugin.name}</span>}
				description={plugin.description}
				status={
					<>
						<StatusBadge tone={plugin.enabled ? 'success' : 'neutral'}>
							{plugin.enabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						{plugin.builtIn ? <StatusBadge>Built-in</StatusBadge> : null}
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
						<span>{toolsLoaded ? `Tools: ${tools.length}` : `Tools: ${plugin.memberCount}`}</span>
						{expanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`tool-plugin-${plugin.ref.rootID}-${plugin.ref.artifactID}`}
						checked={plugin.enabled}
						onChange={enabled => {
							void run('plugin:toggle', () => onTogglePluginEnable(plugin, enabled));
						}}
						disabled={isPending('plugin:toggle')}
						busy={isPending('plugin:toggle')}
						compact={false}
					/>
				}
				actions={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						disabled={isPending('plugin:refresh')}
						onClick={() => {
							void run('plugin:refresh', onRefreshTools);
						}}
					>
						<FiRefreshCw className={isPending('plugin:refresh') ? 'animate-spin' : undefined} size={15} />
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
							disabled={isPending('plugin:refresh')}
							onClick={() => {
								void run('plugin:refresh', onRefreshTools);
							}}
						>
							Retry
						</button>
					</output>
				) : null}

				{expanded && !toolsLoaded && !toolLoadError ? (
					<ManagementEmptyState>
						{isLoadingTools ? 'Loading tools in this Plugin...' : 'Tool contents have not been loaded.'}
					</ManagementEmptyState>
				) : null}

				{expanded ? (
					<div className="mt-6 space-y-3">
						{tools.map(tool => {
							const toggleKey = `${tool.ref.artifactID}:toggle`;
							const effectiveEnabled = plugin.enabled && tool.enabled;

							return (
								<ManagementItemCard
									key={`${tool.ref.rootID}:${tool.ref.artifactID}`}
									title={toolDisplayName(tool)}
									subtitle={tool.name}
									description={tool.description}
									status={
										<>
											<StatusBadge tone={effectiveEnabled ? 'success' : 'neutral'}>
												{effectiveEnabled ? 'Enabled' : tool.enabled ? 'Plugin disabled' : 'Disabled'}
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
												void run(toggleKey, () => onToggleToolEnable(plugin, tool, enabled));
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
							<ManagementEmptyState>No tools are registered in this Plugin.</ManagementEmptyState>
						) : null}
					</div>
				) : null}
			</ManagementPluginCard>

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
