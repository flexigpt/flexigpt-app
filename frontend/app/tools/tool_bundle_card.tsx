import { useEffect, useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import { type Tool, type ToolBundle, ToolImplType } from '@/spec/tool';

import { toolStoreAPI } from '@/apis/baseapi';
import { getAllTools } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { AddEditToolModal } from '@/tools/tool_add_edit_modal';
import { ToolBundleDetailsModal } from '@/tools/tool_bundle_details_modal';

type ToolModalMode = 'add' | 'edit' | 'view';

interface ToolBundleCardProps {
	bundle: ToolBundle;
	tools: Tool[];
	onToolsChange: (bundleID: string, newTools: Tool[]) => void;
	onBundleEnableChange: (bundleID: string, enabled: boolean) => void;
	onBundleDeleted: (bundle: ToolBundle) => void;
}

export function ToolBundleCard({
	bundle,
	tools,
	onToolsChange,
	onBundleEnableChange,
	onBundleDeleted,
}: ToolBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const [localTools, setLocalTools] = useState<Tool[]>(tools);
	const [isBundleEnabled, setIsBundleEnabled] = useState(bundle.isEnabled);

	useEffect(() => {
		setIsBundleEnabled(bundle.isEnabled);
	}, [bundle.isEnabled]);
	useEffect(() => {
		setLocalTools(tools);
	}, [tools]);

	const [isDeleteToolModalOpen, setIsDeleteToolModalOpen] = useState(false);
	const [toolToDelete, setToolToDelete] = useState<Tool | null>(null);

	const [isToolModalOpen, setIsToolModalOpen] = useState(false);
	const [toolModalMode, setToolModalMode] = useState<ToolModalMode>('add');
	const [toolToEdit, setToolToEdit] = useState<Tool | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const toggleBundleEnable = async () => {
		try {
			const newVal = !isBundleEnabled;
			await toolStoreAPI.patchToolBundle(bundle.id, newVal);
			setIsBundleEnabled(newVal);
			onBundleEnableChange(bundle.id, newVal);
		} catch (err) {
			console.error('Toggle bundle enable failed:', err);
			setAlertMsg('Failed to toggle bundle enable state.');
			setShowAlert(true);
		}
	};

	const patchToolEnable = async (tool: Tool) => {
		try {
			await toolStoreAPI.patchTool(bundle.id, tool.slug, tool.version, !tool.isEnabled);
			const updated: Tool = { ...tool, isEnabled: !tool.isEnabled };
			const newArr = localTools.map(t => (t.id === tool.id ? updated : t));
			setLocalTools(newArr);
			onToolsChange(bundle.id, newArr);
		} catch (err) {
			console.error('Toggle tool failed:', err);
			setAlertMsg('Failed to toggle tool.');
			setShowAlert(true);
		}
	};

	const requestDeleteTool = (tool: Tool) => {
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
		try {
			await toolStoreAPI.deleteTool(bundle.id, toolToDelete.slug, toolToDelete.version);
			const newArr = localTools.filter(t => t.id !== toolToDelete.id);
			setLocalTools(newArr);
			onToolsChange(bundle.id, newArr);
		} catch (err) {
			console.error('Delete tool failed:', err);
			setAlertMsg('Failed to delete tool.');
			setShowAlert(true);
		} finally {
			setIsDeleteToolModalOpen(false);
			setToolToDelete(null);
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
		// Defensive: NEVER allow overwriting an existing (slug, version).
		const slug = (toolToEdit?.slug ?? partial.slug ?? '').trim();
		const version = (partial.version ?? '').trim();
		if (!slug) throw new Error('Missing tool slug.');
		if (!version) throw new Error('Version is required.');

		const exists = localTools.some(t => t.slug === slug && t.version === version);
		if (exists) {
			throw new Error(`Version "${version}" already exists for slug "${slug}". Create a different version.`);
		}

		if (toolToEdit) {
			await toolStoreAPI.putTool(
				bundle.id,
				toolToEdit.slug,
				version,
				partial.displayName ?? toolToEdit.displayName,
				partial.isEnabled ?? toolToEdit.isEnabled,
				partial.userCallable ?? toolToEdit.userCallable,
				partial.llmCallable ?? toolToEdit.llmCallable,
				partial.autoExecReco ?? toolToEdit.autoExecReco,
				partial.argSchema ?? toolToEdit.argSchema,
				partial.type ?? toolToEdit.type,
				partial.httpImpl ?? toolToEdit.httpImpl,
				partial.description ?? toolToEdit.description,
				partial.tags ?? toolToEdit.tags
			);
		} else {
			const display = partial.displayName?.trim() ?? '';
			await toolStoreAPI.putTool(
				bundle.id,
				slug,
				version,
				display,
				partial.isEnabled ?? true,
				partial.userCallable ?? true,
				partial.llmCallable ?? true,
				partial.autoExecReco ?? false,
				partial.argSchema ?? {},
				partial.type ?? ToolImplType.HTTP,
				partial.httpImpl,
				partial.description,
				partial.tags
			);
		}

		const toolListItems = await getAllTools([bundle.id], undefined, true);
		const fresh = toolListItems.map(li => li.toolDefinition);
		setLocalTools(fresh);
		onToolsChange(bundle.id, fresh);
	};

	return (
		<div className="bg-base-100 mb-8 rounded-2xl p-4 shadow-lg">
			<div className="grid grid-cols-12 items-center gap-2">
				<div className="col-span-4 flex items-center gap-2">
					<h3 className="text-sm font-semibold">
						<span className="capitalize">{bundle.displayName || bundle.slug}</span>
						<span className="text-base-content/60 ml-1">({bundle.slug})</span>
					</h3>
				</div>

				<div className="col-span-1 flex items-center gap-2">
					<span className="text-base-content/60 text-xs tracking-wide uppercase">
						{bundle.isBuiltIn ? 'Built-in' : 'Custom'}
					</span>
				</div>

				<div className="col-span-3 flex items-center gap-2">
					<label className="text-sm">Enabled</label>
					<input
						type="checkbox"
						className="toggle toggle-accent"
						checked={isBundleEnabled}
						onChange={toggleBundleEnable}
					/>
				</div>

				<div className="col-span-2 flex items-center text-sm">
					<span>Tools:&nbsp;{localTools.length}</span>
				</div>

				<div className="col-span-2 flex items-center justify-end gap-2">
					<button
						className="btn btn-sm btn-ghost rounded-2xl"
						title="View bundle details"
						onClick={e => {
							e.stopPropagation();
							setIsBundleDetailsOpen(true);
						}}
					>
						<FiEye size={16} />
					</button>

					<div
						className="flex cursor-pointer items-center gap-1"
						onClick={() => {
							setIsExpanded(p => !p);
						}}
					>
						<label className="text-sm whitespace-nowrap">Tools</label>
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
									<th>Display Name</th>
									<th>Slug</th>
									<th className="text-center">Enabled</th>
									<th className="text-center">Version</th>
									<th className="text-center">Built-In</th>
									<th className="text-center">Actions</th>
								</tr>
							</thead>
							<tbody>
								{localTools.map(tool => (
									<tr key={tool.id} className="hover:bg-base-300">
										<td>{tool.displayName}</td>
										<td>{tool.slug}</td>
										<td className="text-center align-middle">
											<input
												type="checkbox"
												className="toggle toggle-accent"
												checked={tool.isEnabled}
												onChange={() => patchToolEnable(tool)}
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
													disabled={tool.isBuiltIn || bundle.isBuiltIn}
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
													disabled={tool.isBuiltIn || bundle.isBuiltIn}
													title={tool.isBuiltIn || bundle.isBuiltIn ? 'Deleting disabled for built-in items' : 'Delete'}
													aria-label="Delete"
												>
													<FiTrash2 size={16} />
												</button>
											</div>
										</td>
									</tr>
								))}
								{localTools.length === 0 && (
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
									onBundleDeleted(bundle);
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
				existingTools={localTools.map(t => ({
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
