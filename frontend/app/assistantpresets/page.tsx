import { useCallback, useEffect, useMemo, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { AssistantPreset, AssistantPresetBundle } from '@/spec/assistantpreset';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { assistantPresetStoreAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementBundleCreateModal } from '@/components/management_bundle_create_modal';
import { ManagementPageContent, ManagementPageHeader } from '@/components/management_ui';
import { PageFrame } from '@/components/page_frame';

import { AssistantPresetBundleCard } from '@/assistantpresets/assistant_preset_bundle_card';
import type { PresetItem } from '@/assistantpresets/lib/assistant_preset_editor_types';
import {
	getAllAssistantPresetBundles,
	getAllAssistantPresetListItems,
	sortAssistantPresets,
} from '@/assistantpresets/lib/assistant_preset_store_list_utils';
import type { AssistantPresetUpsertInput } from '@/assistantpresets/lib/assistant_preset_utils';
import { toPutAssistantPresetPayload } from '@/assistantpresets/lib/assistant_preset_utils';

interface BundleData {
	bundle: AssistantPresetBundle;
	presets: AssistantPreset[];
	presetLoadError?: string;
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

// oxlint-disable-next-line no-restricted-exports
export default function AssistantPresetsPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [loading, setLoading] = useState(true);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDeleteID, setBundleToDeleteID] = useState<string | null>(null);
	const [isDeletingBundle, setIsDeletingBundle] = useState(false);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);

	const bundleToDelete =
		bundleToDeleteID === null
			? null
			: (bundles.find(bundleData => bundleData.bundle.id === bundleToDeleteID)?.bundle ?? null);

	const allPresetItems = useMemo<PresetItem[]>(
		() =>
			bundles.flatMap(bundleData =>
				bundleData.presets.map(preset => ({
					preset,
					bundleID: bundleData.bundle.id,
					assistantPresetSlug: preset.slug,
				}))
			),
		[bundles]
	);

	const loadPresetsForBundle = useCallback(async (bundleID: string): Promise<AssistantPreset[]> => {
		const presetListItems = await getAllAssistantPresetListItems([bundleID], true);

		const presetPromises = presetListItems.map(item =>
			assistantPresetStoreAPI.getAssistantPreset(item.bundleID, item.assistantPresetSlug, item.assistantPresetVersion)
		);

		return sortAssistantPresets(
			(await Promise.all(presetPromises)).filter((preset): preset is AssistantPreset => preset !== undefined)
		);
	}, []);

	const refreshBundlePresets = useCallback(
		async (bundleID: string) => {
			try {
				const freshPresets = await loadPresetsForBundle(bundleID);

				setBundles(prev =>
					prev.map(bundleData =>
						bundleData.bundle.id === bundleID
							? { ...bundleData, presets: freshPresets, presetLoadError: undefined }
							: bundleData
					)
				);
			} catch (error) {
				const message = getErrorMessage(error, 'Failed to load assistant presets for this bundle.');

				setBundles(prev =>
					prev.map(bundleData =>
						bundleData.bundle.id === bundleID ? { ...bundleData, presetLoadError: message } : bundleData
					)
				);

				throw error;
			}
		},
		[loadPresetsForBundle]
	);

	const fetchAll = useCallback(async () => {
		setLoading(true);

		try {
			const assistantPresetBundles = await getAllAssistantPresetBundles(undefined, true);

			const bundleResults: BundleData[] = await Promise.all(
				assistantPresetBundles.map(async bundle => {
					try {
						const presets = await loadPresetsForBundle(bundle.id);
						return { bundle, presets };
					} catch (error) {
						return {
							bundle,
							presets: [],
							presetLoadError: getErrorMessage(error, 'Failed to load assistant presets for this bundle.'),
						};
					}
				})
			);

			setBundles(bundleResults);
		} catch (error) {
			console.error('Failed to load assistant preset bundles:', error);
			setAlertMsg(getErrorMessage(error, 'Failed to load assistant preset bundles. Please try again.'));
			setShowAlert(true);
		} finally {
			setLoading(false);
		}
	}, [loadPresetsForBundle]);

	useEffect(() => {
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void fetchAll();
	}, [fetchAll]);

	const handleToggleBundleEnabled = useCallback(
		async (bundleID: string, enabled: boolean) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('Bundle not found.');
			}

			await assistantPresetStoreAPI.patchAssistantPresetBundle(bundleID, enabled);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								bundle: {
									...item.bundle,
									isEnabled: enabled,
								},
							}
						: item
				)
			);
		},
		[bundles]
	);

	const handleTogglePresetEnabled = useCallback(
		async (bundleID: string, presetID: string) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('Bundle not found.');
			}

			if (!bundleData.bundle.isEnabled) {
				throw new Error('Enable the bundle before enabling or disabling assistant preset versions.');
			}

			const preset = bundleData.presets.find(item => item.id === presetID);

			if (!preset) {
				throw new Error('Assistant preset not found.');
			}

			const nextEnabled = !preset.isEnabled;

			await assistantPresetStoreAPI.patchAssistantPreset(bundleID, preset.slug, preset.version, nextEnabled);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								presets: item.presets.map(existingPreset =>
									existingPreset.id === presetID
										? {
												...existingPreset,
												isEnabled: nextEnabled,
											}
										: existingPreset
								),
							}
						: item
				)
			);
		},
		[bundles]
	);

	const handleDeletePreset = useCallback(
		async (bundleID: string, presetID: string) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('Bundle not found.');
			}

			if (bundleData.bundle.isBuiltIn) {
				throw new Error('Cannot delete assistant presets from a built-in bundle.');
			}

			const preset = bundleData.presets.find(item => item.id === presetID);

			if (!preset) {
				throw new Error('Assistant preset not found.');
			}

			if (preset.isBuiltIn) {
				throw new Error('Cannot delete built-in assistant preset.');
			}

			await assistantPresetStoreAPI.deleteAssistantPreset(bundleID, preset.slug, preset.version);

			setBundles(prev =>
				prev.map(item =>
					item.bundle.id === bundleID
						? {
								...item,
								presets: item.presets.filter(existingPreset => existingPreset.id !== presetID),
							}
						: item
				)
			);
		},
		[bundles]
	);

	const handleSubmitPreset = useCallback(
		async (bundleID: string, presetToEditID: string | undefined, partial: AssistantPresetUpsertInput) => {
			const bundleData = bundles.find(item => item.bundle.id === bundleID);

			if (!bundleData) {
				throw new Error('Bundle not found.');
			}

			if (bundleData.bundle.isBuiltIn) {
				throw new Error('Cannot add or edit assistant presets in a built-in bundle.');
			}

			if (!bundleData.bundle.isEnabled) {
				throw new Error('Enable the bundle before adding or editing assistant presets.');
			}

			const presetToEdit =
				presetToEditID === undefined ? undefined : bundleData.presets.find(item => item.id === presetToEditID);

			if (presetToEditID !== undefined && !presetToEdit) {
				throw new Error('Assistant preset not found.');
			}

			if (presetToEdit?.isBuiltIn) {
				throw new Error('Built-in assistant presets cannot be edited.');
			}

			const slug = (presetToEdit?.slug ?? partial.slug).trim();
			const version = partial.version.trim();

			if (!slug) {
				throw new Error('Missing assistant preset slug.');
			}

			if (!version) {
				throw new Error('Version is required.');
			}

			const exists = bundleData.presets.some(preset => preset.slug === slug && preset.version === version);

			if (exists) {
				throw new Error(`Version "${version}" already exists for slug "${slug}". Create a different version.`);
			}

			await assistantPresetStoreAPI.putAssistantPreset(
				bundleID,
				slug,
				version,
				toPutAssistantPresetPayload({
					...partial,
					slug,
					version,
				})
			);

			try {
				await refreshBundlePresets(bundleID);
			} catch (error) {
				console.error('Assistant preset was saved but preset list refresh failed:', error);
				setAlertMsg(
					'Assistant preset version was saved, but the bundle could not be refreshed. Use Retry on the bundle before making destructive changes.'
				);
				setShowAlert(true);
			}
		},
		[bundles, refreshBundlePresets]
	);

	const handleBundleDelete = useCallback(async () => {
		const deletingBundleID = bundleToDeleteID;
		if (!deletingBundleID || isDeletingBundle) {
			return;
		}

		const bundleData = bundles.find(item => item.bundle.id === deletingBundleID);
		if (!bundleData || bundleData.presetLoadError || bundleData.presets.length > 0) {
			setBundleToDeleteID(null);
			setAlertMsg(
				bundleData?.presetLoadError
					? 'Reload this bundle’s presets before deleting it.'
					: 'Remove all assistant presets from this bundle before deleting it.'
			);
			setShowAlert(true);
			return;
		}

		setIsDeletingBundle(true);
		try {
			await assistantPresetStoreAPI.deleteAssistantPresetBundle(deletingBundleID);

			setBundles(prev => prev.filter(item => item.bundle.id !== deletingBundleID));
		} catch (error) {
			console.error('Delete bundle failed:', error);
			setAlertMsg(getErrorMessage(error, 'Failed to delete assistant preset bundle.'));
			setShowAlert(true);
		} finally {
			setIsDeletingBundle(false);
			setBundleToDeleteID(null);
		}
	}, [bundleToDeleteID, bundles, isDeletingBundle]);

	const handleAddBundle = useCallback(
		async (slug: string, display: string, description?: string) => {
			const id = getUUIDv7();
			await assistantPresetStoreAPI.putAssistantPresetBundle(id, slug, display, true, description);
			await fetchAll();
		},
		[fetchAll]
	);

	if (loading) {
		return <Loader text="Loading assistant preset bundles…" />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col items-center overflow-hidden">
				<ManagementPageHeader
					title="Assistant Preset Bundles"
					description="Create reusable starting models, tools, skills, text, and MCP context."
					actions={
						<button
							type="button"
							className="btn btn-ghost rounded-xl"
							onClick={() => {
								setIsAddModalOpen(true);
							}}
						>
							<FiPlus size={18} />
							<span>Add Bundle</span>
						</button>
					}
				/>

				<ManagementPageContent>
					{bundles.length === 0 && (
						<p className="mt-8 text-center text-sm">No assistant preset bundles configured yet.</p>
					)}

					{bundles.map(bundleData => (
						<AssistantPresetBundleCard
							key={bundleData.bundle.id}
							bundle={bundleData.bundle}
							presets={bundleData.presets}
							presetLoadError={bundleData.presetLoadError}
							onRefreshPresets={() => refreshBundlePresets(bundleData.bundle.id)}
							onToggleBundleEnabled={handleToggleBundleEnabled}
							onTogglePresetEnabled={handleTogglePresetEnabled}
							onDeletePreset={handleDeletePreset}
							onSubmitPreset={handleSubmitPreset}
							onDeleteBundleRequested={bundleID => {
								setBundleToDeleteID(bundleID);
							}}
							copyablePresets={allPresetItems}
						/>
					))}
				</ManagementPageContent>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						if (!isDeletingBundle) {
							setBundleToDeleteID(null);
						}
					}}
					onConfirm={handleBundleDelete}
					title="Delete Assistant Preset Bundle"
					message={`Delete empty bundle "${bundleToDelete?.displayName ?? ''}"? Remove all assistant presets first.`}
					confirmButtonText="Delete"
				/>

				<ManagementBundleCreateModal
					isOpen={isAddModalOpen}
					title="Add Assistant Preset Bundle"
					entityLabel="Assistant preset bundle"
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={bundles.map(bundleData => bundleData.bundle.slug)}
					failureMessage="Failed to create assistant preset bundle."
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
		</PageFrame>
	);
}
