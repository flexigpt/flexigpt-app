import { useState } from 'react';

import { FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiRefreshCw, FiTrash2 } from 'react-icons/fi';

import type { Tool, ToolBundle } from '@/spec/tool';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

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

	const [isTogglingBundle, setIsTogglingBundle] = useState(false);
	const [busyToolID, setBusyToolID] = useState<string | null>(null);
	const [isRefreshingTools, setIsRefreshingTools] = useState(false);

	const showError = (err: unknown, fallback: string) => {
		const message = err instanceof Error && err.message.trim() ? err.message : fallback;
		setAlertMsg(message);
		setShowAlert(true);
	};

	const refreshTools = async () => {
		if (isRefreshingTools) {
			return;
		}

		setIsRefreshingTools(true);
		try {
			await onRefreshTools();
		} catch (error) {
			showError(error, 'Failed to reload tools.');
		} finally {
			setIsRefreshingTools(false);
		}
	};

	const toggleBundleEnable = async (enabled: boolean) => {
		setIsTogglingBundle(true);
		try {
			await onToggleBundleEnable(bundle.id, enabled);
		} catch (err) {
			console.error('Toggle bundle enable failed:', err);
			showError(err, 'Failed to toggle bundle enable state.');
		} finally {
			setIsTogglingBundle(false);
		}
	};

	const patchToolEnable = async (tool: Tool, enabled: boolean) => {
		setBusyToolID(tool.id);
		try {
			await onToggleToolEnable(bundle.id, tool, enabled);
		} catch (err) {
			console.error('Toggle tool failed:', err);
			showError(err, 'Failed to toggle tool.');
		} finally {
			setBusyToolID(null);
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
		setBusyToolID(target.id);

		try {
			await onDeleteTool(bundle.id, target);
			setIsDeleteToolModalOpen(false);
			setToolToDelete(null);
		} catch (err) {
			console.error('Delete tool failed:', err);
			showError(err, 'Failed to delete tool.');
		} finally {
			setBusyToolID(null);
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
		setToolModalMode(mode);
		setToolToEdit(tool);
		setIsToolModalOpen(true);
	};

	const handleModifySubmitTool = async (partial: Partial<Tool>) => {
		await onSubmitTool(bundle.id, partial, toolToEdit);
	};

	return (
		<section className="bg-base-100 border-base-content/10 mb-6 rounded-2xl border p-4 shadow-sm">
			<div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div className="min-w-0">
					<h3 className="truncate text-sm font-semibold">
						<span className="capitalize">{bundle.displayName || bundle.slug}</span>
						<span className="text-base-content/60 ml-1">({bundle.slug})</span>
					</h3>
					<div className="text-base-content/60 mt-1 text-xs">
						{bundle.isBuiltIn ? 'Built-in bundle' : 'Custom bundle'}
					</div>
				</div>

				<div className="flex flex-wrap items-center justify-end gap-3">
					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						title="View bundle details"
						onClick={e => {
							e.stopPropagation();
							setIsBundleDetailsOpen(true);
						}}
					>
						<FiEye size={16} />
						<span>Details</span>
					</button>

					<div className="flex items-center gap-1">
						<label htmlFor={`tool-bundle-${bundle.id}`} className="text-sm">
							Enabled
						</label>
						<input
							id={`tool-bundle-${bundle.id}`}
							type="checkbox"
							className="toggle toggle-accent"
							checked={bundle.isEnabled}
							disabled={isTogglingBundle}
							aria-label={`Enable ${bundle.displayName || bundle.slug}`}
							onChange={e => {
								void toggleBundleEnable(e.currentTarget.checked);
							}}
						/>
					</div>

					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={isExpanded}
						onClick={() => {
							setIsExpanded(p => !p);
						}}
					>
						<span className="whitespace-nowrap">Tools: {tools.length}</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				</div>
			</div>

			{toolLoadError ? (
				<div className="alert alert-warning mt-4 rounded-2xl text-sm" role="status">
					<div className="min-w-0 grow">
						<div className="font-semibold">Tools could not be loaded</div>
						<div className="wrap-break-word">{toolLoadError}</div>
					</div>
					<button
						type="button"
						className="btn btn-sm rounded-xl"
						onClick={() => void refreshTools()}
						disabled={isRefreshingTools}
					>
						<FiRefreshCw size={14} />
						<span>{isRefreshingTools ? 'Reloading' : 'Retry'}</span>
					</button>
				</div>
			) : null}

			{isExpanded && (
				<div className="mt-6 space-y-4">
					<div className="space-y-3">
						{tools.map(tool => {
							const isBusy = busyToolID === tool.id;

							return (
								<article
									key={tool.id}
									className="border-base-content/10 hover:border-base-content/20 rounded-2xl border p-4 transition-colors"
								>
									<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
										<div className="min-w-0">
											<div className="truncate font-medium" title={tool.displayName}>
												{tool.displayName || tool.slug}
											</div>
											<div className="text-base-content/60 mt-1 text-xs break-all">
												{tool.slug} · version {tool.version}
											</div>
											{tool.description ? (
												<p className="text-base-content/70 mt-2 max-h-10 overflow-hidden text-sm">{tool.description}</p>
											) : null}
										</div>

										<span
											className={`badge h-auto px-2 py-1 text-center whitespace-normal ${
												tool.isEnabled ? 'badge-success' : 'badge-neutral'
											}`}
										>
											{tool.isEnabled ? 'Enabled' : 'Disabled'}
										</span>
									</div>

									<div className="mt-3 flex flex-wrap gap-2 text-xs">
										<span className="border-base-content/20 rounded-xl border px-2 py-1">{tool.type}</span>
										<span className="border-base-content/20 rounded-xl border px-2 py-1">
											{tool.userCallable ? 'User callable' : 'Not user callable'}
										</span>
										<span className="border-base-content/20 rounded-xl border px-2 py-1">
											{tool.llmCallable ? 'Model callable' : 'Not model callable'}
										</span>
										{tool.isBuiltIn ? (
											<span className="border-base-content/20 rounded-xl border px-2 py-1">Built-in</span>
										) : null}
									</div>

									<div className="border-base-content/10 mt-4 flex flex-col gap-3 border-t pt-3 sm:flex-row sm:items-center sm:justify-between">
										<div className="flex items-center gap-3">
											<label htmlFor={`tool-${tool.id}`} className="text-sm">
												Enabled
											</label>
											<input
												id={`tool-${tool.id}`}
												type="checkbox"
												className="toggle toggle-accent toggle-sm"
												checked={tool.isEnabled}
												disabled={isBusy}
												aria-label={`Enable ${tool.displayName || tool.slug}`}
												onChange={e => {
													void patchToolEnable(tool, e.currentTarget.checked);
												}}
											/>
											{isBusy ? <span className="loading loading-spinner loading-xs" /> : null}
										</div>

										<div className="flex flex-wrap justify-end gap-2">
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													openToolModal('view', tool);
												}}
												disabled={isBusy}
												title="View tool"
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
												disabled={isBusy || tool.isBuiltIn || bundle.isBuiltIn}
												title={
													tool.isBuiltIn || bundle.isBuiltIn
														? 'Built-in items cannot create new versions'
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
												disabled={isBusy || tool.isBuiltIn || bundle.isBuiltIn}
												title={tool.isBuiltIn || bundle.isBuiltIn ? 'Built-in items cannot be deleted' : 'Delete'}
											>
												<FiTrash2 size={15} />
												<span>Delete</span>
											</button>
										</div>
									</div>
								</article>
							);
						})}

						{tools.length === 0 ? (
							<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
								{toolLoadError ? 'Tool contents are unavailable.' : 'No tools in this bundle.'}
							</div>
						) : null}
					</div>

					{!bundle.isBuiltIn && (
						<div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
							<button
								type="button"
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								onClick={() => {
									onRequestDeleteBundle(bundle);
								}}
								disabled={tools.length > 0 || Boolean(toolLoadError)}
								title={
									toolLoadError
										? 'Reload tools before deleting this bundle.'
										: tools.length > 0
											? 'Delete all tools from this bundle first.'
											: 'Delete bundle'
								}
							>
								<FiTrash2 /> <span className="ml-1">Delete Bundle</span>
							</button>
							<button
								type="button"
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								onClick={() => {
									openToolModal('add', undefined);
								}}
								disabled={!bundle.isEnabled || Boolean(toolLoadError)}
								title={!bundle.isEnabled ? 'Enable the bundle first.' : 'Add tool'}
							>
								<FiPlus /> <span className="ml-1">Add Tool</span>
							</button>
						</div>
					)}
				</div>
			)}

			<DeleteConfirmationModal
				isOpen={isDeleteToolModalOpen}
				onClose={() => {
					setIsDeleteToolModalOpen(false);
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
		</section>
	);
}
