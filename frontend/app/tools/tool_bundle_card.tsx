import { useState } from 'react';

import { FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import type { Tool, ToolBundle } from '@/spec/tool';
import { ToolImplType } from '@/spec/tool';

import { usePendingActions } from '@/hooks/use_pending_actions';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

import { AddEditToolModal } from '@/tools/tool_add_edit_modal';
import { ToolBundleDetailsModal } from '@/tools/tool_bundle_details_modal';

type ToolModalMode = 'add' | 'edit' | 'view';

interface ToolBundleCardProps {
	bundle: ToolBundle;
	tools: Tool[];
	toolLoadError?: string;
	onRefreshTools: () => Promise<void>;

	onToggleBundleEnable: (bundleID: string, enabled: boolean) => Promise<void>;
	onToggleToolEnable: (bundleID: string, tool: Tool, enabled: boolean) => Promise<void>;
	onDeleteTool: (bundleID: string, tool: Tool) => Promise<void>;
	onSubmitTool: (bundleID: string, partial: Partial<Tool>, toolToEdit?: Tool) => Promise<void>;
	onRequestDeleteBundle: (bundle: ToolBundle) => void;
}

export function ToolBundleCard({
	bundle,
	tools,
	toolLoadError,
	onRefreshTools,
	onToggleBundleEnable,
	onToggleToolEnable,
	onDeleteTool,
	onSubmitTool,
	onRequestDeleteBundle,
}: ToolBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);

	const [isDeleteToolModalOpen, setIsDeleteToolModalOpen] = useState(false);
	const [toolToDelete, setToolToDelete] = useState<Tool | null>(null);

	const [isToolModalOpen, setIsToolModalOpen] = useState(false);
	const [toolModalMode, setToolModalMode] = useState<ToolModalMode>('add');
	const [toolToEdit, setToolToEdit] = useState<Tool | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const { isPending, runAction } = usePendingActions();

	const showError = (err: unknown, fallback: string) => {
		const message = err instanceof Error && err.message.trim() ? err.message : fallback;
		setAlertMsg(message);
		setShowAlert(true);
	};

	const refreshTools = async () => {
		try {
			await runAction('bundle:refresh', onRefreshTools);
		} catch (error) {
			showError(error, 'Failed to reload tools.');
		}
	};

	const toggleBundleEnable = async (enabled: boolean) => {
		try {
			await runAction('bundle:toggle', () => onToggleBundleEnable(bundle.id, enabled));
		} catch (err) {
			console.error('Toggle bundle enable failed:', err);
			showError(err, 'Failed to toggle bundle enable state.');
		}
	};

	const patchToolEnable = async (tool: Tool, enabled: boolean) => {
		try {
			await runAction(`${tool.id}:toggle`, () => onToggleToolEnable(bundle.id, tool, enabled));
		} catch (err) {
			console.error('Toggle tool failed:', err);
			showError(err, 'Failed to toggle tool.');
		}
	};

	const requestDeleteTool = (tool: Tool) => {
		if (bundle.isBuiltIn) {
			setAlertMsg('Cannot delete tools from a built-in bundle.');
			setShowAlert(true);
			return;
		}
		if (tool.isBuiltIn) {
			setAlertMsg('Cannot delete built-in tool.');
			setShowAlert(true);
			return;
		}
		setToolToDelete(tool);
		setIsDeleteToolModalOpen(true);
	};

	const confirmDeleteTool = async () => {
		if (!toolToDelete) {
			return;
		}

		const target = toolToDelete;

		try {
			const deleted = await runAction(`${target.id}:delete`, async () => {
				await onDeleteTool(bundle.id, target);
				return true;
			});

			if (deleted) {
				setIsDeleteToolModalOpen(false);
				setToolToDelete(null);
			}
		} catch (err) {
			console.error('Delete tool failed:', err);
			showError(err, 'Failed to delete tool.');
		}
	};

	const openToolModal = (mode: ToolModalMode, tool?: Tool) => {
		if ((mode === 'add' || mode === 'edit') && bundle.isBuiltIn) {
			setAlertMsg('Cannot add or edit tools in a built-in bundle.');
			setShowAlert(true);
			return;
		}
		if (mode === 'edit' && tool?.isBuiltIn) {
			setAlertMsg('Built-in tools cannot be edited.');
			setShowAlert(true);
			return;
		}
		if (mode === 'edit' && tool && tool.type !== ToolImplType.HTTP) {
			setAlertMsg('Only HTTP tools can create new versions in this UI. Go and SDK tools are view-only here.');
			setShowAlert(true);
			return;
		}
		if ((mode === 'add' || mode === 'edit') && !bundle.isEnabled) {
			setAlertMsg('Enable the bundle before adding or creating a new tool version.');
			setShowAlert(true);
			return;
		}
		setToolModalMode(mode);
		setToolToEdit(tool);
		setIsToolModalOpen(true);
	};

	const handleModifySubmitTool = async (partial: Partial<Tool>) => {
		await onSubmitTool(bundle.id, partial, toolToEdit);
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
						<StatusBadge>{bundle.isBuiltIn ? 'Built-in' : 'Custom'}</StatusBadge>
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
							void toggleBundleEnable(enabled);
						}}
						disabled={bundle.isBuiltIn}
						busy={isPending('bundle:toggle')}
						compact={false}
					/>
				}
				actions={
					<>
						<button
							type="button"
							className="btn btn-sm btn-ghost rounded-xl"
							title="View bundle details"
							onClick={() => {
								setIsBundleDetailsOpen(true);
							}}
						>
							<FiEye size={16} />
							<span>Details</span>
						</button>
						{!bundle.isBuiltIn ? (
							<>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										openToolModal('add', undefined);
									}}
									disabled={!bundle.isEnabled || Boolean(toolLoadError)}
								>
									<FiPlus size={16} />
									<span>Add Tool</span>
								</button>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									onClick={() => {
										onRequestDeleteBundle(bundle);
									}}
									disabled={tools.length > 0 || Boolean(toolLoadError)}
								>
									<FiTrash2 size={16} />
									<span>Delete Bundle</span>
								</button>
							</>
						) : null}
					</>
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
							onClick={() => {
								void refreshTools();
							}}
							disabled={isPending('bundle:refresh')}
						>
							<FiRefreshCw size={14} />
							<span>{isPending('bundle:refresh') ? 'Reloading' : 'Retry'}</span>
						</button>
					</output>
				) : null}

				{isExpanded ? (
					<div className="mt-6 space-y-3">
						{tools.map(tool => {
							const toggleKey = `${tool.id}:toggle`;
							const deleteKey = `${tool.id}:delete`;

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
											{tool.isBuiltIn ? <StatusBadge>Built-in</StatusBadge> : null}
										</>
									}
									metadata={
										<>
											<MetadataPill label="Type">{tool.type}</MetadataPill>
											<MetadataPill label="User callable">{tool.userCallable ? 'Yes' : 'No'}</MetadataPill>
											<MetadataPill label="Model callable">{tool.llmCallable ? 'Yes' : 'No'}</MetadataPill>
											<MetadataPill label="Auto execute">{tool.autoExecReco ? 'Recommended' : 'No'}</MetadataPill>
										</>
									}
								>
									<ActionRow
										leading={
											<EnabledControl
												id={`tool-${tool.id}`}
												checked={tool.isEnabled}
												onChange={enabled => {
													void patchToolEnable(tool, enabled);
												}}
												disabled={!bundle.isEnabled}
												busy={isPending(toggleKey)}
												title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
											/>
										}
									>
										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											onClick={() => {
												openToolModal('view', tool);
											}}
										>
											<FiEye size={15} />
											<span>View</span>
										</button>
										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											onClick={() => {
												openToolModal('edit', tool);
											}}
											disabled={
												!bundle.isEnabled || tool.isBuiltIn || bundle.isBuiltIn || tool.type !== ToolImplType.HTTP
											}
											title={
												tool.type !== ToolImplType.HTTP
													? 'Only HTTP tools can create new versions in this UI.'
													: 'Create a new version'
											}
										>
											<FiGitBranch size={15} />
											<span>New version</span>
										</button>
										<button
											type="button"
											className="btn btn-sm btn-ghost rounded-xl"
											onClick={() => {
												requestDeleteTool(tool);
											}}
											disabled={isPending(deleteKey) || tool.isBuiltIn || bundle.isBuiltIn}
										>
											<FiTrash2 size={15} />
											<span>Delete</span>
										</button>
									</ActionRow>
								</ManagementItemCard>
							);
						})}

						{tools.length === 0 ? (
							<ManagementEmptyState>
								{toolLoadError ? 'Tool contents are unavailable.' : 'No tools in this bundle.'}
							</ManagementEmptyState>
						) : null}
					</div>
				) : null}
			</ManagementBundleCard>

			<DeleteConfirmationModal
				isOpen={isDeleteToolModalOpen}
				onClose={() => {
					if (!toolToDelete || !isPending(`${toolToDelete.id}:delete`)) {
						setIsDeleteToolModalOpen(false);
						setToolToDelete(null);
					}
				}}
				onConfirm={confirmDeleteTool}
				title="Delete Tool"
				message={`Delete tool "${toolToDelete?.displayName ?? ''}"? This cannot be undone.`}
				confirmButtonText="Delete"
			/>

			<AddEditToolModal
				isOpen={isToolModalOpen}
				onClose={() => {
					setIsToolModalOpen(false);
					setToolToEdit(undefined);
				}}
				onSubmit={handleModifySubmitTool}
				mode={toolModalMode}
				initialData={
					toolToEdit
						? {
								tool: toolToEdit,
								bundleID: bundle.id,
								toolSlug: toolToEdit.slug,
							}
						: undefined
				}
				existingTools={tools.map(t => ({
					tool: t,
					bundleID: bundle.id,
					toolSlug: t.slug,
				}))}
			/>

			<ToolBundleDetailsModal
				isOpen={isBundleDetailsOpen}
				onClose={() => {
					setIsBundleDetailsOpen(false);
				}}
				bundle={bundle}
			/>

			<ActionDeniedAlertModal
				isOpen={showAlert}
				onClose={() => {
					setShowAlert(false);
					setAlertMsg('');
				}}
				message={alertMsg}
			/>
		</>
	);
}
