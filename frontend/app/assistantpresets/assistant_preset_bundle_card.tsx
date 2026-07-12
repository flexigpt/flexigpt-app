import { useState } from 'react';

import { FiChevronDown, FiChevronUp, FiEye, FiGitBranch, FiPlus, FiTrash2 } from 'react-icons/fi';

import type { AssistantPreset, AssistantPresetBundle } from '@/spec/assistantpreset';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { ActionRow } from '@/components/managementui/action_row';
import { EnabledControl } from '@/components/managementui/enabled_control';
import { ManagementBundleCard } from '@/components/managementui/management_bundle_card';
import { ManagementEmptyState } from '@/components/managementui/management_empty_state';
import { ManagementItemCard } from '@/components/managementui/management_item_card';
import { MetadataPill } from '@/components/managementui/metadata_pill';
import { StatusBadge } from '@/components/managementui/status_badge';

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
						<span className="whitespace-nowrap">Presets: {presets.length}</span>
						{isExpanded ? <FiChevronUp /> : <FiChevronDown />}
					</button>
				}
				actionLeading={
					<EnabledControl
						id={`assistant-bundle-${bundle.id}`}
						checked={bundle.isEnabled}
						onChange={() => {
							void handleToggleBundleEnable();
						}}
						busy={isBundleTogglePending}
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
									disabled={!bundle.isEnabled || Boolean(presetLoadError)}
									onClick={() => {
										openPresetModal('add');
									}}
								>
									<FiPlus size={16} />
									<span>Add Preset</span>
								</button>
								<button
									type="button"
									className="btn btn-sm btn-ghost rounded-xl"
									disabled={presets.length > 0 || Boolean(presetLoadError)}
									onClick={() => {
										onDeleteBundleRequested(bundle.id);
									}}
								>
									<FiTrash2 size={16} />
									<span>Delete Bundle</span>
								</button>
							</>
						) : null}
					</>
				}
			>
				{presetLoadError ? (
					<div className="alert alert-warning mt-3 rounded-2xl text-sm">
						<div className="grow">
							<div className="font-semibold">Assistant presets could not be loaded for this bundle</div>
							<div>{presetLoadError}</div>
						</div>
						<button
							id={`assistant-bundle-${bundle.id}`}
							type="button"
							className="btn btn-sm rounded-xl"
							onClick={() => {
								void refreshPresets();
							}}
							disabled={isRefreshingPresets}
						>
							{isRefreshingPresets ? 'Reloading...' : 'Retry'}
						</button>
					</div>
				) : null}

				{isExpanded ? (
					<div className="mt-6 space-y-4">
						<div className="space-y-3">
							{presets.map(preset => {
								const counts = getAssistantPresetCounts(preset);
								const model = formatAssistantPresetModelRef(preset.startingModelPresetRef);

								return (
									<ManagementItemCard
										key={preset.id}
										title={preset.displayName || preset.slug}
										subtitle={`${preset.slug} / version ${preset.version}`}
										description={preset.description}
										status={
											<>
												<StatusBadge tone={preset.isEnabled ? 'success' : 'neutral'}>
													{preset.isEnabled ? 'Enabled' : 'Disabled'}
												</StatusBadge>
												{preset.isBuiltIn ? <StatusBadge>Built-in</StatusBadge> : null}
											</>
										}
										metadata={
											<>
												<MetadataPill label="Model">{model}</MetadataPill>
												<MetadataPill label="Instructions">{counts.instructions}</MetadataPill>
												<MetadataPill label="Skills">{counts.skills}</MetadataPill>
												<MetadataPill label="Tools">{counts.tools}</MetadataPill>
												<MetadataPill label="MCP">{counts.mcp}</MetadataPill>
											</>
										}
									>
										<ActionRow
											leading={
												<EnabledControl
													id={`assistant-preset-${preset.id}`}
													checked={preset.isEnabled}
													onChange={() => {
														void handlePresetEnableToggle(preset);
													}}
													disabled={pendingPresetToggleIDs.has(preset.id) || !bundle.isEnabled}
													busy={pendingPresetToggleIDs.has(preset.id)}
													title={!bundle.isEnabled ? 'Enable the bundle first.' : undefined}
												/>
											}
										>
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
										</ActionRow>
									</ManagementItemCard>
								);
							})}

							{presets.length === 0 ? (
								<ManagementEmptyState>No assistant presets in this bundle.</ManagementEmptyState>
							) : null}
						</div>
					</div>
				) : null}
			</ManagementBundleCard>

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
		</>
	);
}
