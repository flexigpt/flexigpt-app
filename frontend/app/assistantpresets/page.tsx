import { useCallback, useEffect, useState } from 'react';

import { FiPlus } from 'react-icons/fi';

import type { AssistantPreset, AssistantPresetBundle } from '@/spec/assistantpreset';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { assistantPresetStoreAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { PageFrame } from '@/components/page_frame';

import { AddAssistantPresetBundleModal } from '@/assistantpresets/assistant_preset_bundle_add_modal';
import { AssistantPresetBundleCard } from '@/assistantpresets/assistant_preset_bundle_card';
import {
	getAllAssistantPresetBundles,
	getAllAssistantPresetListItems,
	sortAssistantPresets,
} from '@/assistantpresets/lib/assistant_preset_store_list_utils';
import {
	type AssistantPresetUpsertInput,
	toPutAssistantPresetPayload,
} from '@/assistantpresets/lib/assistant_preset_utils';

interface BundleData {
	bundle: AssistantPresetBundle;
	presets: AssistantPreset[];
}

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

// eslint-disable-next-line no-restricted-exports
export default function AssistantPresetsPage() {
	const [bundles, setBundles] = useState<BundleData[]>([]);
	const [loading, setLoading] = useState(true);

	const [showAlert, setShowAlert] = useState(false);
	const [alertMsg, setAlertMsg] = useState('');

	const [bundleToDeleteID, setBundleToDeleteID] = useState<string | null>(null);
	const [isAddModalOpen, setIsAddModalOpen] = useState(false);

	const bundleToDelete =
		bundleToDeleteID === null
			? null
			: (bundles.find(bundleData => bundleData.bundle.id === bundleToDeleteID)?.bundle ?? null);

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
			const freshPresets = await loadPresetsForBundle(bundleID);

			setBundles(prev =>
				prev.map(bundleData =>
					bundleData.bundle.id === bundleID ? { ...bundleData, presets: freshPresets } : bundleData
				)
			);
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
					} catch {
						return { bundle, presets: [] };
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

			await refreshBundlePresets(bundleID);
		},
		[bundles, refreshBundlePresets]
	);

	const handleBundleDelete = useCallback(async () => {
		if (!bundleToDeleteID) {
			return;
		}

		try {
			await assistantPresetStoreAPI.deleteAssistantPresetBundle(bundleToDeleteID);

			setBundles(prev => prev.filter(bundleData => bundleData.bundle.id !== bundleToDeleteID));
		} catch (error) {
			console.error('Delete bundle failed:', error);
			setAlertMsg(getErrorMessage(error, 'Failed to delete assistant preset bundle.'));
			setShowAlert(true);
		} finally {
			setBundleToDeleteID(null);
		}
	}, [bundleToDeleteID]);

	const handleAddBundle = useCallback(
		async (slug: string, display: string, description?: string) => {
			try {
				const id = getUUIDv7();
				await assistantPresetStoreAPI.putAssistantPresetBundle(id, slug, display, true, description);
				setIsAddModalOpen(false);
				await fetchAll();
			} catch (error) {
				console.error('Add bundle failed:', error);
				setAlertMsg(getErrorMessage(error, 'Failed to add assistant preset bundle.'));
				setShowAlert(true);
			}
		},
		[fetchAll]
	);

	if (loading) {
		return <Loader text="Loading assistant preset bundles…" />;
	}

	return (
		<PageFrame>
			<div className="flex h-full w-full flex-col items-center">
				<div className="fixed mt-8 flex w-10/12 items-center p-2 lg:w-2/3">
					<h1 className="flex grow items-center justify-center text-xl font-semibold">Assistant Preset Bundles</h1>
					<button
						className="btn btn-ghost flex items-center rounded-2xl"
						onClick={() => {
							setIsAddModalOpen(true);
						}}
					>
						<FiPlus size={20} /> <span className="ml-1">Add Bundle</span>
					</button>
				</div>

				<div
					className="mt-24 flex w-full grow flex-col items-center overflow-y-auto"
					style={{ maxHeight: `calc(100vh - 128px)` }}
				>
					<div className="flex w-11/12 flex-col space-y-4 xl:w-2/3">
						{bundles.length === 0 && (
							<p className="mt-8 text-center text-sm">No assistant preset bundles configured yet.</p>
						)}

						{bundles.map(bundleData => (
							<AssistantPresetBundleCard
								key={bundleData.bundle.id}
								bundle={bundleData.bundle}
								presets={bundleData.presets}
								onToggleBundleEnabled={handleToggleBundleEnabled}
								onTogglePresetEnabled={handleTogglePresetEnabled}
								onDeletePreset={handleDeletePreset}
								onSubmitPreset={handleSubmitPreset}
								onDeleteBundleRequested={bundleID => {
									setBundleToDeleteID(bundleID);
								}}
							/>
						))}
					</div>
				</div>

				<DeleteConfirmationModal
					isOpen={bundleToDelete !== null}
					onClose={() => {
						setBundleToDeleteID(null);
					}}
					onConfirm={handleBundleDelete}
					title="Delete Assistant Preset Bundle"
					message={`Delete empty bundle "${bundleToDelete?.displayName ?? ''}"? Remove all assistant presets first.`}
					confirmButtonText="Delete"
				/>

				<AddAssistantPresetBundleModal
					isOpen={isAddModalOpen}
					onClose={() => {
						setIsAddModalOpen(false);
					}}
					onSubmit={handleAddBundle}
					existingSlugs={bundles.map(bundleData => bundleData.bundle.slug)}
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
