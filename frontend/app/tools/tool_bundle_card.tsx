import { useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import { type Tool, type ToolBundle } from '@/spec/tool';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { AddEditToolModal } from '@/tools/tool_add_edit_modal';
import { ToolBundleDetailsModal } from '@/tools/tool_bundle_details_modal';

type ToolModalMode = 'add' | 'edit' | 'view';

interface ToolBundleCardProps {
	bundle: ToolBundle;
	tools: Tool[];
	onToggleBundleEnable: (bundleID: string, enabled: boolean) => Promise<void>;
	onToggleToolEnable: (bundleID: string, tool: Tool, enabled: boolean) => Promise<void>;
	onDeleteTool: (bundleID: string, tool: Tool) => Promise<void>;
	onSubmitTool: (bundleID: string, partial: Partial<Tool>, toolToEdit?: Tool) => Promise<void>;
	onRequestDeleteBundle: (bundle: ToolBundle) => void;
}

export function ToolBundleCard({
	bundle,
	tools,
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

	const showError = (err: unknown, fallback: string) => {
		const message = err instanceof Error && err.message.trim() ? err.message : fallback;
		setAlertMsg(message);
		setShowAlert(true);
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
		if (!toolToDelete) return;

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
		<div className="bg-base-100 mb-8 rounded-2xl p-4 shadow-lg">
			<div className="flex items-center justify-between">
				<div className="flex items-center">
					<h3 className="gap-2 text-sm font-semibold">
						<span className="capitalize">{bundle.displayName || bundle.slug}</span>
						<span className="text-base-content/60 ml-1">({bundle.slug})</span>
					</h3>
				</div>

				<div className="flex items-center justify-end gap-4">
					<button
						className="btn btn-sm btn-ghost p-0"
						title="View bundle details"
						onClick={e => {
							e.stopPropagation();
							setIsBundleDetailsOpen(true);
						}}
					>
						<FiEye size={16} />
					</button>

					<span className="text-base-content/60 text-xs tracking-wide uppercase">
						{bundle.isBuiltIn ? 'Built-in' : 'Custom'}
					</span>

					<div className="flex items-center gap-1">
						<label className="text-sm">Enabled</label>
						<input
							type="checkbox"
							className="toggle toggle-accent"
							checked={bundle.isEnabled}
							disabled={isTogglingBundle}
							onChange={e => {
								void toggleBundleEnable(e.currentTarget.checked);
							}}
						/>
					</div>

					<div
						className="flex cursor-pointer items-center gap-1"
						onClick={() => {
							setIsExpanded(p => !p);
						}}
					>
						<label className="text-sm whitespace-nowrap">Tools:&nbsp;{tools.length}</label>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</div>
				</div>
			</div>

			{isExpanded && (
				<div className="mt-8 space-y-4">
					<div className="border-base-content/10 overflow-x-auto rounded-2xl border">
						<table className="table-zebra table w-full">
							<thead>
								<tr className="bg-base-300 text-sm font-semibold">
									<th className="w-full">Display Name</th>
									<th className="min-w-32 text-center">Slug</th>
									<th className="text-center whitespace-nowrap">Enabled</th>
									<th className="text-center whitespace-nowrap">Version</th>
									<th className="text-center whitespace-nowrap">Built-In</th>
									<th className="text-center whitespace-nowrap">Actions</th>
								</tr>
							</thead>
							<tbody>
								{tools.map(tool => {
									const isBusy = busyToolID === tool.id;

									return (
										<tr key={tool.id} className="hover:bg-base-300">
											<td>{tool.displayName}</td>
											<td className="text-center">{tool.slug}</td>
											<td className="text-center align-middle">
												<input
													type="checkbox"
													className="toggle toggle-accent"
													checked={tool.isEnabled}
													disabled={isBusy}
													onChange={e => {
														void patchToolEnable(tool, e.currentTarget.checked);
													}}
												/>
											</td>
											<td className="text-center">{tool.version}</td>
											<td className="text-center">
												{tool.isBuiltIn ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
											</td>
											<td className="text-center">
												<div className="inline-flex items-center gap-2">
													<button
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															openToolModal('view', tool);
														}}
														disabled={isBusy}
														title="View"
														aria-label="View"
													>
														<FiEye size={16} />
													</button>

													<button
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															openToolModal('edit', tool);
														}}
														disabled={isBusy || tool.isBuiltIn || bundle.isBuiltIn}
														title={
															tool.isBuiltIn || bundle.isBuiltIn
																? 'Built-in items cannot create new versions'
																: 'New Version'
														}
														aria-label="New Version"
													>
														<FiGitBranch size={16} />
													</button>

													<button
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															requestDeleteTool(tool);
														}}
														disabled={isBusy || tool.isBuiltIn || bundle.isBuiltIn}
														title={
															tool.isBuiltIn || bundle.isBuiltIn ? 'Deleting disabled for built-in items' : 'Delete'
														}
														aria-label="Delete"
													>
														<FiTrash2 size={16} />
													</button>
												</div>
											</td>
										</tr>
									);
								})}

								{tools.length === 0 && (
									<tr>
										<td colSpan={6} className="py-3 text-center text-sm">
											No tools in this bundle.
										</td>
									</tr>
								)}
							</tbody>
						</table>
					</div>

					{!bundle.isBuiltIn && (
						<div className="flex items-center justify-between">
							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								onClick={() => {
									onRequestDeleteBundle(bundle);
								}}
							>
								<FiTrash2 /> <span className="ml-1">Delete Bundle</span>
							</button>
							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								onClick={() => {
									openToolModal('add', undefined);
								}}
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
		</div>
	);
}
