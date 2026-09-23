import { useState } from 'react';
import { FiChevronDown, FiChevronUp, FiRefreshCw } from 'react-icons/fi';

import type { Tool, ToolBundle } from '@/spec/tool';
import { ToolImplType } from '@/spec/tool';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

interface ToolBundleCardProps {
	bundle: ToolBundle;
	tools: Tool[];
	toolLoadError?: string;
	onRefreshTools: () => Promise<void>;
	onToggleBundleEnable: (bundleID: string, enabled: boolean) => Promise<void>;
	onToggleToolEnable: (bundleID: string, tool: Tool, enabled: boolean) => Promise<void>;
}

function implementationLabel(tool: Tool): string {
	switch (tool.type) {
		case ToolImplType.Go:
			return 'Go';
		case ToolImplType.SDK:
			return 'Provider API';
		default:
			return tool.type;
	}
}

function implementationIdentity(tool: Tool): string | undefined {
	switch (tool.type) {
		case ToolImplType.Go:
			return tool.goImpl?.func;
		case ToolImplType.SDK:
			return tool.sdkImpl?.sdkType;
		default:
			return undefined;
	}
}

export function ToolBundleCard({
	bundle,
	tools,
	toolLoadError,
	onRefreshTools,
	onToggleBundleEnable,
	onToggleToolEnable,
}: ToolBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const [alertMessage, setAlertMessage] = useState('');
	const [showAlert, setShowAlert] = useState(false);
	const { isPending, runAction } = usePendingActions();

	const showError = (error: unknown, fallback: string) => {
		const message = error instanceof Error && error.message.trim() ? error.message : fallback;
		setAlertMessage(message);
		setShowAlert(true);
	};

	const refreshTools = async () => {
		try {
			await runAction('bundle:refresh', onRefreshTools);
		} catch (error) {
			showError(error, 'Failed to refresh built-in tools.');
		}
	};

	const toggleBundle = async (enabled: boolean) => {
		try {
			await runAction('bundle:toggle', () => onToggleBundleEnable(bundle.id, enabled));
		} catch (error) {
			showError(error, 'Failed to change built-in bundle availability.');
		}
	};

	const toggleTool = async (tool: Tool, enabled: boolean) => {
		try {
			await runAction(`${tool.id}:toggle`, () => onToggleToolEnable(bundle.id, tool, enabled));
		} catch (error) {
			showError(error, 'Failed to change built-in tool availability.');
		}
	};

	return (
		<>
			<ManagementBundleCard
				title={bundle.displayName || bundle.slug}
				identity={
					<span className="font-mono">
						{bundle.slug} / {bundle.id}
					</span>
				}
				description={bundle.description}
				status={
					<>
						<StatusBadge tone={bundle.isEnabled ? 'success' : 'neutral'}>
							{bundle.isEnabled ? 'Enabled' : 'Disabled'}
						</StatusBadge>
						<StatusBadge>Built-in</StatusBadge>
					</>
				}
				disclosure={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={isExpanded}
						onClick={() => {
							setIsExpanded(previous => !previous);
						}}
					>
						<span className="whitespace-nowrap">Tools: {tools.length}</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`tool-bundle-${bundle.id}`}
						checked={bundle.isEnabled}
						onChange={enabled => {
							void toggleBundle(enabled);
						}}
						disabled={isPending('bundle:toggle')}
						busy={isPending('bundle:toggle')}
						compact={false}
					/>
				}
				actions={
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						onClick={() => {
							void refreshTools();
						}}
						disabled={isPending('bundle:refresh')}
					>
						<FiRefreshCw className={isPending('bundle:refresh') ? 'animate-spin' : undefined} size={15} />
						<span>{isPending('bundle:refresh') ? 'Refreshing' : 'Refresh'}</span>
					</button>
				}
			>
				{toolLoadError ? (
					<output className="alert alert-warning mt-4 rounded-2xl text-sm">
						<span className="min-w-0 grow">
							<span className="block font-semibold">Built-in tools could not be loaded</span>
							<span className="block wrap-break-word">{toolLoadError}</span>
						</span>
						<button
							type="button"
							className="btn btn-sm rounded-xl"
							onClick={() => {
								void refreshTools();
							}}
							disabled={isPending('bundle:refresh')}
						>
							Retry
						</button>
					</output>
				) : null}

				{isExpanded ? (
					<div className="mt-6 space-y-3">
						{tools.map(tool => {
							const toolToggleKey = `${tool.id}:toggle`;
							const identity = implementationIdentity(tool);

							return (
								<ManagementItemCard
									key={tool.id}
									title={tool.displayName || tool.slug}
									subtitle={`${tool.slug} / version ${tool.version}`}
									description={tool.description}
									status={
										<>
											<StatusBadge tone={tool.isEnabled ? 'success' : 'neutral'}>
												{tool.isEnabled ? 'Enabled' : 'Disabled'}
											</StatusBadge>
											<StatusBadge>Built-in</StatusBadge>
										</>
									}
									metadata={
										<>
											<MetadataPill label="Implementation">{implementationLabel(tool)}</MetadataPill>
											<MetadataPill label="User callable">{tool.userCallable ? 'Yes' : 'No'}</MetadataPill>
											<MetadataPill label="Model callable">{tool.llmCallable ? 'Yes' : 'No'}</MetadataPill>
											{identity ? (
												<MetadataPill label="Built-in ID">
													<span className="font-mono">{identity}</span>
												</MetadataPill>
											) : null}
										</>
									}
								>
									<div className="mt-4 flex flex-wrap items-center gap-3">
										<EnabledControl
											id={`tool-${tool.id}`}
											checked={tool.isEnabled}
											onChange={enabled => {
												void toggleTool(tool, enabled);
											}}
											disabled={isPending(toolToggleKey)}
											busy={isPending(toolToggleKey)}
											title="Change built-in tool availability"
										/>
										<span className="text-base-content/70 text-xs">Built-in tool definitions are read-only.</span>
									</div>
								</ManagementItemCard>
							);
						})}

						{tools.length === 0 ? (
							<ManagementEmptyState>No built-in tools are registered in this bundle.</ManagementEmptyState>
						) : null}
					</div>
				) : null}
			</ManagementBundleCard>

			<ActionDeniedAlertModal
				isOpen={showAlert}
				onClose={() => {
					setShowAlert(false);
					setAlertMessage('');
				}}
				message={alertMessage}
			/>
		</>
	);
}
