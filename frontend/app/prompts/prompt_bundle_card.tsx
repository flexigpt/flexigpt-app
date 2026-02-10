import { useEffect, useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import type { PromptBundle, PromptTemplate } from '@/spec/prompt';

import { promptStoreAPI } from '@/apis/baseapi';
import { getAllPromptTemplates } from '@/apis/list_helper';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { PromptBundleDetailsModal } from '@/prompts/prompt_bundle_details_modal';
import { AddEditPromptTemplateModal } from '@/prompts/prompt_template_add_edit_modal';

type TemplateModalMode = 'add' | 'edit' | 'view';

interface PromptBundleCardProps {
	bundle: PromptBundle;
	templates: PromptTemplate[];

	onTemplatesChange: (bundleID: string, newTemplates: PromptTemplate[]) => void;
	onBundleEnableChange: (bundleID: string, enabled: boolean) => void;
	onBundleDeleted: (bundle: PromptBundle) => void;
}

export function PromptBundleCard({
	bundle,
	templates,
	onTemplatesChange,
	onBundleEnableChange,
	onBundleDeleted,
}: PromptBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);
	const [localTemplates, setLocalTemplates] = useState<PromptTemplate[]>(templates);

	const [isBundleEnabled, setIsBundleEnabled] = useState(bundle.isEnabled);

	useEffect(() => {
		setIsBundleEnabled(bundle.isEnabled);
	}, [bundle.isEnabled]);

	useEffect(() => {
		setLocalTemplates(templates);
	}, [templates]);

	const [isDeleteTemplateModalOpen, setIsDeleteTemplateModalOpen] = useState(false);
	const [templateToDelete, setTemplateToDelete] = useState<PromptTemplate | null>(null);

	const [isTemplateModalOpen, setIsTemplateModalOpen] = useState(false);
	const [templateModalMode, setTemplateModalMode] = useState<TemplateModalMode>('add');
	const [templateToEdit, setTemplateToEdit] = useState<PromptTemplate | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const toggleBundleEnable = async () => {
		try {
			const newVal = !isBundleEnabled;
			await promptStoreAPI.patchPromptBundle(bundle.id, newVal);
			setIsBundleEnabled(newVal);
			onBundleEnableChange(bundle.id, newVal);
		} catch (err) {
			console.error('Failed to toggle bundle:', err);
			setAlertMsg('Failed to toggle bundle enable state.');
			setShowAlert(true);
		}
	};

	const patchTemplateEnable = async (tpl: PromptTemplate) => {
		try {
			await promptStoreAPI.patchPromptTemplate(bundle.id, tpl.slug, tpl.version, !tpl.isEnabled);
			const updated: PromptTemplate = { ...tpl, isEnabled: !tpl.isEnabled };
			const newArr = localTemplates.map(t => (t.id === tpl.id ? updated : t));
			setLocalTemplates(newArr);
			onTemplatesChange(bundle.id, newArr);
		} catch (err) {
			console.error('Toggle template failed:', err);
			setAlertMsg('Failed to toggle template.');
			setShowAlert(true);
		}
	};

	const requestDeleteTemplate = (tpl: PromptTemplate) => {
		if (tpl.isBuiltIn) {
			setAlertMsg('Cannot delete built-in template.');
			setShowAlert(true);
			return;
		}
		setTemplateToDelete(tpl);
		setIsDeleteTemplateModalOpen(true);
	};

	const confirmDeleteTemplate = async () => {
		if (!templateToDelete) return;
		try {
			await promptStoreAPI.deletePromptTemplate(bundle.id, templateToDelete.slug, templateToDelete.version);
			const newArr = localTemplates.filter(t => t.id !== templateToDelete.id);
			setLocalTemplates(newArr);
			onTemplatesChange(bundle.id, newArr);
		} catch (err) {
			console.error('Delete template failed:', err);
			setAlertMsg('Failed to delete template.');
			setShowAlert(true);
		} finally {
			setIsDeleteTemplateModalOpen(false);
			setTemplateToDelete(null);
		}
	};

	const openTemplateModal = (mode: TemplateModalMode, tpl?: PromptTemplate) => {
		if ((mode === 'add' || mode === 'edit') && bundle.isBuiltIn) {
			setAlertMsg('Cannot add or edit templates in a built-in bundle.');
			setShowAlert(true);
			return;
		}
		if (mode === 'edit' && tpl?.isBuiltIn) {
			setAlertMsg('Built-in templates cannot be edited.');
			setShowAlert(true);
			return;
		}
		setTemplateModalMode(mode);
		setTemplateToEdit(tpl);
		setIsTemplateModalOpen(true);
	};

	const handleModifySubmit = async (partial: Partial<PromptTemplate>) => {
		// Defensive: NEVER allow overwriting an existing (slug, version).
		const slug = (templateToEdit?.slug ?? partial.slug ?? '').trim();
		const version = (partial.version ?? '').trim();
		if (!slug) throw new Error('Missing template slug.');
		if (!version) throw new Error('Version is required.');

		const exists = localTemplates.some(t => t.slug === slug && t.version === version);
		if (exists) {
			throw new Error(`Version "${version}" already exists for slug "${slug}". Create a different version.`);
		}

		if (templateToEdit) {
			await promptStoreAPI.putPromptTemplate(
				bundle.id,
				templateToEdit.slug,
				partial.displayName ?? templateToEdit.displayName,
				partial.isEnabled ?? templateToEdit.isEnabled,
				partial.blocks ?? templateToEdit.blocks,
				version,
				partial.description ?? templateToEdit.description,
				partial.tags ?? templateToEdit.tags,
				partial.variables ?? templateToEdit.variables
			);
		} else {
			const display = partial.displayName?.trim() ?? '';
			await promptStoreAPI.putPromptTemplate(
				bundle.id,
				slug,
				display,
				partial.isEnabled ?? true,
				partial.blocks ?? [],
				version,
				partial.description,
				partial.tags,
				partial.variables
			);
		}

		const promptTemplateListItems = await getAllPromptTemplates([bundle.id], undefined, true);
		const tplPromises = promptTemplateListItems.map(li =>
			promptStoreAPI.getPromptTemplate(li.bundleID, li.templateSlug, li.templateVersion)
		);
		const fresh = (await Promise.all(tplPromises)).filter((t): t is PromptTemplate => t !== undefined);
		setLocalTemplates(fresh);
		onTemplatesChange(bundle.id, fresh);
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
							checked={isBundleEnabled}
							onChange={toggleBundleEnable}
						/>
					</div>

					<div
						className="flex cursor-pointer items-center gap-1"
						onClick={() => {
							setIsExpanded(p => !p);
						}}
					>
						<label className="text-sm whitespace-nowrap">Templates:&nbsp;{localTemplates.length}</label>
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
								{localTemplates.map(tpl => (
									<tr key={tpl.id} className="hover:bg-base-300">
										<td>{tpl.displayName}</td>
										<td className="text-center">{tpl.slug}</td>
										<td className="text-center align-middle">
											<input
												type="checkbox"
												className="toggle toggle-accent"
												checked={tpl.isEnabled}
												onChange={() => patchTemplateEnable(tpl)}
											/>
										</td>
										<td className="text-center">{tpl.version}</td>
										<td className="text-center">
											{tpl.isBuiltIn ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
										</td>
										<td className="justify-end text-center">
											<div className="inline-flex items-center gap-2">
												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														openTemplateModal('view', tpl);
													}}
													title="View"
													aria-label="View"
												>
													<FiEye size={16} />
												</button>

												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														openTemplateModal('edit', tpl);
													}}
													disabled={tpl.isBuiltIn || bundle.isBuiltIn}
													title={
														tpl.isBuiltIn || bundle.isBuiltIn
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
														requestDeleteTemplate(tpl);
													}}
													disabled={tpl.isBuiltIn || bundle.isBuiltIn}
													title={tpl.isBuiltIn || bundle.isBuiltIn ? 'Deleting disabled for built-in items' : 'Delete'}
													aria-label="Delete"
												>
													<FiTrash2 size={16} />
												</button>
											</div>
										</td>
									</tr>
								))}
								{localTemplates.length === 0 && (
									<tr>
										<td colSpan={6} className="py-3 text-center text-sm">
											No templates in this bundle.
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
									openTemplateModal('add', undefined);
								}}
							>
								<FiPlus /> <span className="ml-1">Add Template</span>
							</button>
						</div>
					)}
				</div>
			)}

			<DeleteConfirmationModal
				isOpen={isDeleteTemplateModalOpen}
				onClose={() => {
					setIsDeleteTemplateModalOpen(false);
				}}
				onConfirm={confirmDeleteTemplate}
				title="Delete Prompt Template"
				message={`Delete template "${templateToDelete?.displayName ?? ''}"? This cannot be undone.`}
				confirmButtonText="Delete"
			/>

			<AddEditPromptTemplateModal
				isOpen={isTemplateModalOpen}
				onClose={() => {
					setIsTemplateModalOpen(false);
					setTemplateToEdit(undefined);
				}}
				onSubmit={handleModifySubmit}
				mode={templateModalMode}
				initialData={
					templateToEdit
						? {
								template: templateToEdit,
								bundleID: bundle.id,
								templateSlug: templateToEdit.slug,
							}
						: undefined
				}
				existingTemplates={localTemplates.map(t => ({
					template: t,
					bundleID: bundle.id,
					templateSlug: t.slug,
				}))}
			/>

			<PromptBundleDetailsModal
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
