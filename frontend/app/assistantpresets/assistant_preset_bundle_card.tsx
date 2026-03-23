import { useState } from 'react';

import { FiCheck, FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiTrash2, FiX } from 'react-icons/fi';

import type { AssistantPreset, AssistantPresetBundle } from '@/spec/assistantpreset';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { AddEditAssistantPresetModal } from '@/assistantpresets/assistant_preset_add_edit_modal';
import { AssistantPresetBundleDetailsModal } from '@/assistantpresets/assistant_preset_bundle_details_modal';
import {
	type AssistantPresetUpsertInput,
	formatAssistantPresetModelRef,
	getAssistantPresetCounts,
} from '@/assistantpresets/lib/assistant_preset_utils';

type PresetModalMode = 'add' | 'edit' | 'view';

interface AssistantPresetBundleCardProps {
	bundle: AssistantPresetBundle;
	presets: AssistantPreset[];

	onToggleBundleEnabled: (bundleID: string, enabled: boolean) => Promise<void>;
	onTogglePresetEnabled: (bundleID: string, presetID: string) => Promise<void>;
	onDeletePreset: (bundleID: string, presetID: string) => Promise<void>;
	onSubmitPreset: (
		bundleID: string,
		presetToEditID: string | undefined,
		partial: AssistantPresetUpsertInput
	) => Promise<void>;
	onDeleteBundleRequested: (bundleID: string) => void;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

export function AssistantPresetBundleCard({
	bundle,
	presets,
	onToggleBundleEnabled,
	onTogglePresetEnabled,
	onDeletePreset,
	onSubmitPreset,
	onDeleteBundleRequested,
}: AssistantPresetBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);

	const [isDeletePresetModalOpen, setIsDeletePresetModalOpen] = useState(false);
	const [presetToDelete, setPresetToDelete] = useState<AssistantPreset | null>(null);

	const [isPresetModalOpen, setIsPresetModalOpen] = useState(false);
	const [presetModalMode, setPresetModalMode] = useState<PresetModalMode>('add');
	const [presetToEdit, setPresetToEdit] = useState<AssistantPreset | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [isBundleTogglePending, setIsBundleTogglePending] = useState(false);
	const [pendingPresetToggleIDs, setPendingPresetToggleIDs] = useState<Set<string>>(new Set());

	const openAlert = (message: string) => {
		setAlertMsg(message);
		setShowAlert(true);
	};

	const setPresetTogglePending = (presetID: string, pending: boolean) => {
		setPendingPresetToggleIDs(prev => {
			const next = new Set(prev);

			if (pending) {
				next.add(presetID);
			} else {
				next.delete(presetID);
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

	const handlePresetEnableToggle = async (preset: AssistantPreset) => {
		try {
			setPresetTogglePending(preset.id, true);
			await onTogglePresetEnabled(bundle.id, preset.id);
		} catch (error) {
			console.error('Toggle preset failed:', error);
			openAlert(getErrorMessage(error, 'Failed to toggle assistant preset.'));
		} finally {
			setPresetTogglePending(preset.id, false);
		}
	};

	const requestDeletePreset = (preset: AssistantPreset) => {
		if (bundle.isBuiltIn) {
			openAlert('Cannot delete assistant presets from a built-in bundle.');
			return;
		}

		if (preset.isBuiltIn) {
			openAlert('Cannot delete built-in assistant preset.');
			return;
		}

		setPresetToDelete(preset);
		setIsDeletePresetModalOpen(true);
	};

	const confirmDeletePreset = async () => {
		if (!presetToDelete) {
			return;
		}

		try {
			await onDeletePreset(bundle.id, presetToDelete.id);
		} catch (error) {
			console.error('Delete assistant preset failed:', error);
			openAlert(getErrorMessage(error, 'Failed to delete assistant preset.'));
		} finally {
			setIsDeletePresetModalOpen(false);
			setPresetToDelete(null);
		}
	};

	const openPresetModal = (mode: PresetModalMode, preset?: AssistantPreset) => {
		if ((mode === 'add' || mode === 'edit') && bundle.isBuiltIn) {
			openAlert('Cannot add or edit assistant presets in a built-in bundle.');
			return;
		}

		if ((mode === 'add' || mode === 'edit') && !bundle.isEnabled) {
			openAlert('Enable the bundle before adding or editing assistant presets.');
			return;
		}

		if (mode === 'edit' && preset?.isBuiltIn) {
			openAlert('Built-in assistant presets cannot be edited.');
			return;
		}

		setPresetModalMode(mode);
		setPresetToEdit(preset);
		setIsPresetModalOpen(true);
	};

	const handleModifySubmit = async (partial: AssistantPresetUpsertInput) => {
		await onSubmitPreset(bundle.id, presetToEdit?.id, partial);
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
						<label className="text-sm whitespace-nowrap">Presets:&nbsp;{presets.length}</label>
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
									<th className="text-center whitespace-nowrap">Model</th>
									<th className="text-center whitespace-nowrap">Instructions</th>
									<th className="text-center whitespace-nowrap">Tools</th>
									<th className="text-center whitespace-nowrap">Skills</th>
									<th className="text-center whitespace-nowrap">Version</th>
									<th className="text-center whitespace-nowrap">Built-In</th>
									<th className="text-center whitespace-nowrap">Actions</th>
								</tr>
							</thead>
							<tbody>
								{presets.map(preset => {
									const counts = getAssistantPresetCounts(preset);

									return (
										<tr key={preset.id} className="hover:bg-base-300">
											<td>
												<div className="flex items-center gap-2">
													{preset.icon?.trim() ? <span className="text-lg leading-none">{preset.icon}</span> : null}
													<span>{preset.displayName}</span>
												</div>
											</td>
											<td className="text-center">{preset.slug}</td>
											<td className="text-center align-middle">
												<input
													type="checkbox"
													className="toggle toggle-accent"
													checked={preset.isEnabled}
													disabled={pendingPresetToggleIDs.has(preset.id) || !bundle.isEnabled}
													title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
													onChange={() => {
														void handlePresetEnableToggle(preset);
													}}
												/>
											</td>
											<td className="text-center">{formatAssistantPresetModelRef(preset.startingModelPresetRef)}</td>
											<td className="text-center">{counts.instructions}</td>
											<td className="text-center">{counts.tools}</td>
											<td className="text-center">{counts.skills}</td>
											<td className="text-center whitespace-nowrap">{preset.version}</td>
											<td className="text-center">
												{preset.isBuiltIn ? <FiCheck className="mx-auto" /> : <FiX className="mx-auto" />}
											</td>
											<td className="justify-end text-center">
												<div className="inline-flex items-center gap-2">
													<button
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															openPresetModal('view', preset);
														}}
														title="View"
														aria-label="View"
													>
														<FiEye size={16} />
													</button>

													<button
														className="btn btn-sm btn-ghost rounded-2xl"
														onClick={() => {
															openPresetModal('edit', preset);
														}}
														disabled={preset.isBuiltIn || bundle.isBuiltIn || !bundle.isEnabled}
														title={
															preset.isBuiltIn || bundle.isBuiltIn
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
															requestDeletePreset(preset);
														}}
														disabled={preset.isBuiltIn || bundle.isBuiltIn}
														title={
															preset.isBuiltIn || bundle.isBuiltIn ? 'Deleting disabled for built-in items' : 'Delete'
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

								{presets.length === 0 && (
									<tr>
										<td colSpan={10} className="py-3 text-center text-sm">
											No assistant presets in this bundle.
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
								disabled={presets.length > 0}
								title={presets.length > 0 ? 'Delete all assistant presets from this bundle first.' : 'Delete Bundle'}
								onClick={() => {
									onDeleteBundleRequested(bundle.id);
								}}
							>
								<FiTrash2 /> <span className="ml-1">Delete Bundle</span>
							</button>

							<button
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								disabled={!bundle.isEnabled}
								title={!bundle.isEnabled ? 'Enable the bundle first.' : 'Add Assistant Preset'}
								onClick={() => {
									openPresetModal('add');
								}}
							>
								<FiPlus /> <span className="ml-1">Add Assistant Preset</span>
							</button>
						</div>
					)}
				</div>
			)}

			<DeleteConfirmationModal
				isOpen={isDeletePresetModalOpen}
				onClose={() => {
					setIsDeletePresetModalOpen(false);
					setPresetToDelete(null);
				}}
				onConfirm={confirmDeletePreset}
				title="Delete Assistant Preset"
				message={`Delete assistant preset "${presetToDelete?.displayName ?? ''}"? This cannot be undone.`}
				confirmButtonText="Delete"
			/>

			<AddEditAssistantPresetModal
				isOpen={isPresetModalOpen}
				onClose={() => {
					setIsPresetModalOpen(false);
					setPresetToEdit(undefined);
				}}
				onSubmit={handleModifySubmit}
				mode={presetModalMode}
				initialData={
					presetToEdit
						? {
								preset: presetToEdit,
								bundleID: bundle.id,
								assistantPresetSlug: presetToEdit.slug,
							}
						: undefined
				}
				existingPresets={presets.map(preset => ({
					preset,
					bundleID: bundle.id,
					assistantPresetSlug: preset.slug,
				}))}
			/>

			<AssistantPresetBundleDetailsModal
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
