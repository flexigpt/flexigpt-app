import { useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import type { PromptBundle, PromptTemplate } from '@/spec/prompt';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { PromptBundleDetailsModal } from '@/prompts/prompt_bundle_details_modal';
import { AddEditPromptTemplateModal } from '@/prompts/prompt_template_add_edit_modal';
import {
	getPromptTemplateKindLabel,
	getPromptTemplateResolutionLabel,
	type PromptTemplateUpsertInput,
} from '@/prompts/prompt_template_utils';

type TemplateModalMode = 'add' | 'edit' | 'view';

interface PromptBundleCardProps {
	bundle: PromptBundle;
	templates: PromptTemplate[];

	onToggleBundleEnabled: (bundleID: string, enabled: boolean) => Promise<void>;
	onToggleTemplateEnabled: (bundleID: string, templateID: string) => Promise<void>;
	onDeleteTemplate: (bundleID: string, templateID: string) => Promise<void>;
	onSubmitTemplate: (
		bundleID: string,
		templateToEditID: string | undefined,
		partial: PromptTemplateUpsertInput
	) => Promise<void>;
	onDeleteBundleRequested: (bundleID: string) => void;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

export function PromptBundleCard({
	bundle,
	templates,
	onToggleBundleEnabled,
	onToggleTemplateEnabled,
	onDeleteTemplate,
	onSubmitTemplate,
	onDeleteBundleRequested,
}: PromptBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);

	const [isDeleteTemplateModalOpen, setIsDeleteTemplateModalOpen] = useState(false);
	const [templateToDelete, setTemplateToDelete] = useState<PromptTemplate | null>(null);

	const [isTemplateModalOpen, setIsTemplateModalOpen] = useState(false);
	const [templateModalMode, setTemplateModalMode] = useState<TemplateModalMode>('add');
	const [templateToEdit, setTemplateToEdit] = useState<PromptTemplate | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [isBundleTogglePending, setIsBundleTogglePending] = useState(false);
	const [pendingTemplateToggleIDs, setPendingTemplateToggleIDs] = useState<Set<string>>(new Set());

	const openAlert = (message: string) => {
		setAlertMsg(message);
		setShowAlert(true);
	};

	const setTemplateTogglePending = (templateID: string, pending: boolean) => {
		setPendingTemplateToggleIDs(prev => {
			const next = new Set(prev);

			if (pending) {
				next.add(templateID);
			} else {
				next.delete(templateID);
			}

			return next;
		});
	};

	const handleToggleBundleEnable = async () => {
		try {
			setIsBundleTogglePending(true);
			await onToggleBundleEnabled(bundle.id, !bundle.isEnabled);
		} catch (error) {
			console.error('Failed to toggle bundle:', error);
			openAlert(getErrorMessage(error, 'Failed to toggle bundle enable state.'));
		} finally {
			setIsBundleTogglePending(false);
		}
	};

	const handleTemplateEnableToggle = async (template: PromptTemplate) => {
		try {
			setTemplateTogglePending(template.id, true);
			await onToggleTemplateEnabled(bundle.id, template.id);
		} catch (error) {
			console.error('Toggle template failed:', error);
			openAlert(getErrorMessage(error, 'Failed to toggle template.'));
		} finally {
			setTemplateTogglePending(template.id, false);
		}
	};

	const requestDeleteTemplate = (template: PromptTemplate) => {
		if (bundle.isBuiltIn) {
			openAlert('Cannot delete templates from a built-in bundle.');
			return;
		}

		if (template.isBuiltIn) {
			openAlert('Cannot delete built-in template.');
			return;
		}

		setTemplateToDelete(template);
		setIsDeleteTemplateModalOpen(true);
	};

	const confirmDeleteTemplate = async () => {
		if (!templateToDelete) {
			return;
		}

		try {
			await onDeleteTemplate(bundle.id, templateToDelete.id);
		} catch (error) {
			console.error('Delete template failed:', error);
			openAlert(getErrorMessage(error, 'Failed to delete template.'));
		} finally {
			setIsDeleteTemplateModalOpen(false);
			setTemplateToDelete(null);
		}
	};

	const openTemplateModal = (mode: TemplateModalMode, template?: PromptTemplate) => {
		if ((mode === 'add' || mode === 'edit') && bundle.isBuiltIn) {
			openAlert('Cannot add or edit templates in a built-in bundle.');
			return;
		}

		if ((mode === 'add' || mode === 'edit') && !bundle.isEnabled) {
			openAlert('Enable the bundle before adding or editing templates.');
			return;
		}

		if (mode === 'edit' && template?.isBuiltIn) {
			openAlert('Built-in templates cannot be edited.');
			return;
		}

		setTemplateModalMode(mode);
		setTemplateToEdit(template);
		setIsTemplateModalOpen(true);
	};

	const handleModifySubmit = async (partial: PromptTemplateUpsertInput) => {
		await onSubmitTemplate(bundle.id, templateToEdit?.id, partial);
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
							disabled={isBundleTogglePending}
							onChange={() => {
								void handleToggleBundleEnable();
							}}
						/>
					</div>

					<div
						className="flex cursor-pointer items-center gap-1"
						onClick={() => {
							setIsExpanded(prev => !prev);
						}}
					>
						<label className="text-sm whitespace-nowrap">Templates:&nbsp;{templates.length}</label>
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
									<th className="text-center">Slug</th>
									<th className="text-center whitespace-nowrap">Enabled</th>
									<th className="text-center whitespace-nowrap">Kind</th>
									<th className="text-center whitespace-nowrap">Resolved</th>
									<th className="text-center whitespace-nowrap">Version</th>
									<th className="text-center whitespace-nowrap">Built-In</th>
									<th className="text-center whitespace-nowrap">Actions</th>
								</tr>
							</thead>
							<tbody>
								{templates.map(template => (
									<tr key={template.id} className="hover:bg-base-300">
										<td>{template.displayName}</td>
										<td className="text-center">{template.slug}</td>
										<td className="text-center align-middle">
											<input
												type="checkbox"
												className="toggle toggle-accent"
												checked={template.isEnabled}
												disabled={pendingTemplateToggleIDs.has(template.id) || !bundle.isEnabled}
												title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
												onChange={() => {
													void handleTemplateEnableToggle(template);
												}}
											/>
										</td>
										<td className="text-center">{getPromptTemplateKindLabel(template.kind)}</td>
										<td className="text-center" title={getPromptTemplateResolutionLabel(template.isResolved)}>
											{template.isResolved ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
										</td>
										<td className="text-center">{template.version}</td>
										<td className="text-center">
											{template.isBuiltIn ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
										</td>
										<td className="justify-end text-center">
											<div className="inline-flex items-center gap-2">
												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														openTemplateModal('view', template);
													}}
													title="View"
													aria-label="View"
												>
													<FiEye size={16} />
												</button>

												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														openTemplateModal('edit', template);
													}}
													disabled={template.isBuiltIn || bundle.isBuiltIn || !bundle.isEnabled}
													title={
														template.isBuiltIn || bundle.isBuiltIn
															? 'Built-in items cannot create new versions'
															: !bundle.isEnabled
																? 'Enable the bundle first.'
																: 'New Version'
													}
													aria-label="New Version"
												>
													<FiGitBranch size={16} />
												</button>

												<button
													className="btn btn-sm btn-ghost rounded-2xl"
													onClick={() => {
														requestDeleteTemplate(template);
													}}
													disabled={template.isBuiltIn || bundle.isBuiltIn}
													title={
														template.isBuiltIn || bundle.isBuiltIn ? 'Deleting disabled for built-in items' : 'Delete'
													}
													aria-label="Delete"
												>
													<FiTrash2 size={16} />
												</button>
											</div>
										</td>
									</tr>
								))}

								{templates.length === 0 && (
									<tr>
										<td colSpan={8} className="py-3 text-center text-sm">
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
								disabled={templates.length > 0}
								title={templates.length > 0 ? 'Delete all templates from this bundle first.' : 'Delete Bundle'}
								onClick={() => {
									onDeleteBundleRequested(bundle.id);
								}}
							>
								<FiTrash2 /> <span className="ml-1">Delete Bundle</span>
							</button>

							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								disabled={!bundle.isEnabled}
								title={!bundle.isEnabled ? 'Enable the bundle first.' : 'Add Template'}
								onClick={() => {
									openTemplateModal('add');
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
					setTemplateToDelete(null);
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
				existingTemplates={templates.map(template => ({
					template,
					bundleID: bundle.id,
					templateSlug: template.slug,
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
