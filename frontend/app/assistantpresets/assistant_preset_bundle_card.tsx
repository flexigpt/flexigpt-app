import { useState } from 'react';

import { FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiTrash2 } from 'react-icons/fi';

import type { AssistantPreset, AssistantPresetBundle } from '@/spec/assistantpreset';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { AddEditAssistantPresetModal } from '@/assistantpresets/assistant_preset_add_edit_modal';
import { AssistantPresetBundleDetailsModal } from '@/assistantpresets/assistant_preset_bundle_details_modal';
import type { PresetItem } from '@/assistantpresets/lib/assistant_preset_editor_types';
import type { AssistantPresetUpsertInput } from '@/assistantpresets/lib/assistant_preset_utils';
import { formatAssistantPresetModelRef, getAssistantPresetCounts } from '@/assistantpresets/lib/assistant_preset_utils';

type PresetModalMode = 'add' | 'edit' | 'view';

interface AssistantPresetBundleCardProps {
	bundle: AssistantPresetBundle;
	presets: AssistantPreset[];
	presetLoadError?: string;

	onRefreshPresets: () => Promise<void>;
	onToggleBundleEnabled: (bundleID: string, enabled: boolean) => Promise<void>;
	onTogglePresetEnabled: (bundleID: string, presetID: string) => Promise<void>;
	onDeletePreset: (bundleID: string, presetID: string) => Promise<void>;
	onSubmitPreset: (
		bundleID: string,
		presetToEditID: string | undefined,
		partial: AssistantPresetUpsertInput
	) => Promise<void>;
	onDeleteBundleRequested: (bundleID: string) => void;
	copyablePresets: PresetItem[];
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
	presetLoadError,
	onRefreshPresets,
	onToggleBundleEnabled,
	onTogglePresetEnabled,
	onDeletePreset,
	onSubmitPreset,
	onDeleteBundleRequested,
	copyablePresets,
}: AssistantPresetBundleCardProps) {
	const [isExpanded, setIsExpanded] = useState(false);

	const [isDeletePresetModalOpen, setIsDeletePresetModalOpen] = useState(false);
	const [presetToDelete, setPresetToDelete] = useState<AssistantPreset | null>(null);
	const [isDeletePresetPending, setIsDeletePresetPending] = useState(false);
	const [isRefreshingPresets, setIsRefreshingPresets] = useState(false);

	const [isPresetModalOpen, setIsPresetModalOpen] = useState(false);
	const [presetModalMode, setPresetModalMode] = useState<PresetModalMode>('add');
	const [presetToEdit, setPresetToEdit] = useState<AssistantPreset | undefined>(undefined);

	const [isBundleDetailsOpen, setIsBundleDetailsOpen] = useState(false);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [isBundleTogglePending, setIsBundleTogglePending] = useState(false);
	const [pendingPresetToggleIDs, setPendingPresetToggleIDs] = useState(new Set());

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

	const refreshPresets = async () => {
		if (isRefreshingPresets) {
			return;
		}

		setIsRefreshingPresets(true);
		try {
			await onRefreshPresets();
		} catch (error) {
			console.error('Reload assistant presets failed:', error);
			openAlert(getErrorMessage(error, 'Failed to reload assistant presets.'));
		} finally {
			setIsRefreshingPresets(false);
		}
	};

	const handleToggleBundleEnable = async () => {
		if (isBundleTogglePending) {
			return;
		}

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
		if (pendingPresetToggleIDs.has(preset.id)) {
			return;
		}

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
		if (!presetToDelete || isDeletePresetPending) {
			return;
		}

		setIsDeletePresetPending(true);
		let deleted = false;
		try {
			await onDeletePreset(bundle.id, presetToDelete.id);
			deleted = true;
		} catch (error) {
			console.error('Delete assistant preset failed:', error);
			openAlert(getErrorMessage(error, 'Failed to delete assistant preset.'));
		} finally {
			setIsDeletePresetPending(false);
			if (deleted) {
				setIsDeletePresetModalOpen(false);
				setPresetToDelete(null);
			}
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
						<label htmlFor={`assistant-bundle-${bundle.id}`} className="text-sm">
							Enabled
						</label>
						<input
							id={`assistant-bundle-${bundle.id}`}
							type="checkbox"
							className="toggle toggle-accent"
							checked={bundle.isEnabled}
							disabled={isBundleTogglePending}
							aria-label={`Enable ${bundle.displayName || bundle.slug}`}
							onChange={() => {
								void handleToggleBundleEnable();
							}}
						/>
					</div>

					<button
						type="button"
						className="btn btn-sm btn-ghost rounded-xl"
						aria-expanded={isExpanded}
						onClick={() => {
							setIsExpanded(prev => !prev);
						}}
					>
						<span className="whitespace-nowrap">Presets: {presets.length}</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				</div>
			</div>

			{presetLoadError ? (
				<div className="alert alert-warning mt-3 rounded-2xl text-sm">
					<div className="grow">
						<div className="font-semibold">Assistant presets could not be loaded for this bundle</div>
						<div>{presetLoadError}</div>
					</div>
					<button
						type="button"
						className="btn btn-sm rounded-xl"
						onClick={() => {
							void refreshPresets();
						}}
						disabled={isRefreshingPresets}
					>
						{isRefreshingPresets ? 'Reloading…' : 'Retry'}
					</button>
				</div>
			) : null}

			{isExpanded && (
				<div className="mt-6 space-y-4">
					<div className="space-y-3">
						{presets.map(preset => {
							const counts = getAssistantPresetCounts(preset);
							const model = formatAssistantPresetModelRef(preset.startingModelPresetRef);

							return (
								<article
									key={preset.id}
									className="border-base-content/10 hover:border-base-content/20 rounded-2xl border p-4 transition-colors"
								>
									<div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
										<div className="min-w-0">
											<div className="truncate font-medium" title={preset.displayName}>
												{preset.displayName || preset.slug}
											</div>
											<div className="text-base-content/60 mt-1 text-xs break-all">
												{preset.slug} · version {preset.version}
											</div>
											{preset.description ? (
												<p className="text-base-content/70 mt-2 max-h-10 overflow-hidden text-sm">
													{preset.description}
												</p>
											) : null}
										</div>

										<div className="flex shrink-0 items-center gap-2">
											<span
												className={`badge h-auto max-w-full px-2 py-1 text-center whitespace-normal ${
													preset.isEnabled ? 'badge-success' : 'badge-neutral'
												}`}
											>
												{preset.isEnabled ? 'Enabled' : 'Disabled'}
											</span>
											{pendingPresetToggleIDs.has(preset.id) ? (
												<span className="loading loading-spinner loading-xs" aria-label="Updating preset" />
											) : null}
										</div>
									</div>

									<dl className="mt-4 grid grid-cols-1 gap-2 text-xs sm:grid-cols-2 lg:grid-cols-3">
										<div className="border-base-content/10 min-w-0 rounded-xl border px-3 py-2">
											<dt className="text-base-content/60">Model</dt>
											<dd className="mt-1 truncate" title={model}>
												{model}
											</dd>
										</div>
										<div className="border-base-content/10 rounded-xl border px-3 py-2">
											<dt className="text-base-content/60">Starting content</dt>
											<dd className="mt-1">
												{counts.instructions} instructions · {counts.skills} skills
											</dd>
										</div>
										<div className="border-base-content/10 rounded-xl border px-3 py-2">
											<dt className="text-base-content/60">Integrations</dt>
											<dd className="mt-1">
												{counts.tools} tools · {counts.mcp} MCP items
											</dd>
										</div>
									</dl>

									<div className="border-base-content/10 mt-4 flex flex-col gap-3 border-t pt-3 sm:flex-row sm:items-center sm:justify-between">
										<div className="flex items-center gap-3">
											<label htmlFor={`assistant-preset-${preset.id}`} className="text-sm">
												Enabled
											</label>
											<input
												id={`assistant-preset-${preset.id}`}
												type="checkbox"
												className="toggle toggle-accent toggle-sm"
												checked={preset.isEnabled}
												disabled={pendingPresetToggleIDs.has(preset.id) || !bundle.isEnabled}
												title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
												aria-label={`Enable ${preset.displayName || preset.slug}`}
												onChange={() => {
													void handlePresetEnableToggle(preset);
												}}
											/>
											{preset.isBuiltIn ? (
												<span className="border-base-content/20 rounded-xl border px-2 py-1 text-xs">Built-in</span>
											) : null}
										</div>

										<div className="flex flex-wrap justify-end gap-2">
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													openPresetModal('view', preset);
												}}
												title="View assistant preset"
											>
												<FiEye size={15} />
												<span>View</span>
											</button>
											<button
												type="button"
												className="btn btn-sm btn-ghost rounded-xl"
												onClick={() => {
													openPresetModal('edit', preset);
												}}
												disabled={preset.isBuiltIn || bundle.isBuiltIn || !bundle.isEnabled}
												title={
													preset.isBuiltIn || bundle.isBuiltIn
														? 'Built-in items cannot create new versions'
														: !bundle.isEnabled
															? 'Enable the bundle first.'
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
													requestDeletePreset(preset);
												}}
												disabled={preset.isBuiltIn || bundle.isBuiltIn}
												title={preset.isBuiltIn || bundle.isBuiltIn ? 'Built-in items cannot be deleted' : 'Delete'}
											>
												<FiTrash2 size={15} />
												<span>Delete</span>
											</button>
										</div>
									</div>
								</article>
							);
						})}

						{presets.length === 0 ? (
							<div className="border-base-content/10 rounded-2xl border py-6 text-center text-sm">
								No assistant presets in this bundle.
							</div>
						) : null}
					</div>

					{!bundle.isBuiltIn && (
						<div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
							<button
								type="button"
								className="btn btn-md btn-ghost flex items-center rounded-2xl"
								disabled={presets.length > 0 || Boolean(presetLoadError)}
								title={
									presetLoadError
										? 'Reload assistant presets before deleting this bundle.'
										: presets.length > 0
											? 'Delete all assistant presets from this bundle first.'
											: 'Delete Bundle'
								}
								onClick={() => {
									onDeleteBundleRequested(bundle.id);
								}}
							>
								<FiTrash2 /> <span className="ml-1">Delete Bundle</span>
							</button>

							<button
								type="button"
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
					if (!isDeletePresetPending) {
						setIsDeletePresetModalOpen(false);
						setPresetToDelete(null);
					}
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
				copyablePresets={copyablePresets}
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
		</section>
	);
}
