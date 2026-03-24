import { useCallback, useEffect, useState } from 'react';

import { assistantPresetStoreAPI } from '@/apis/baseapi';

import {
	getAllAssistantPresetBundles,
	getAllAssistantPresetListItems,
} from '@/assistantpresets/lib/assistant_preset_store_list_utils';
import {
	type AssistantPresetOptionItem,
	buildAssistantPresetIdentityKey,
} from '@/chats/inputarea/assitantcontexts/assistant_preset_runtime';

function getErrorMessage(error: unknown, fallback: string): string {
	if (error instanceof Error && error.message.trim().length > 0) {
		return error.message;
	}
	return fallback;
}

function getBundleDisplayName(bundle: { displayName?: string; slug: string }, fallbackID: string): string {
	return bundle.displayName || bundle.slug || fallbackID;
}

export function useAssistantPresets() {
	const [presetOptions, setPresetOptions] = useState<AssistantPresetOptionItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const refreshPresets = useCallback(async () => {
		setLoading(true);
		setError(null);

		try {
			const bundles = await getAllAssistantPresetBundles(undefined, false);
			const bundleByID = new Map(bundles.map(bundle => [bundle.id, bundle]));

			const listItems = await getAllAssistantPresetListItems(
				bundles.map(bundle => bundle.id),
				false
			);

			const fullResults = await Promise.all(
				listItems.map(async item => {
					const preset = await assistantPresetStoreAPI.getAssistantPreset(
						item.bundleID,
						item.assistantPresetSlug,
						item.assistantPresetVersion
					);

					return {
						item,
						preset,
					};
				})
			);

			const nextOptions: AssistantPresetOptionItem[] = fullResults.flatMap(({ item, preset }) => {
				if (!preset) {
					return [];
				}

				const bundle = bundleByID.get(item.bundleID);
				const bundleDisplayName = bundle
					? getBundleDisplayName(bundle, item.bundleID)
					: item.bundleSlug || item.bundleID;

				const displayName = preset.displayName || preset.slug;
				const label = `${displayName} — ${bundleDisplayName} (${preset.slug}@${preset.version})`;

				return [
					{
						key: buildAssistantPresetIdentityKey(item.bundleID, item.assistantPresetSlug, item.assistantPresetVersion),
						bundleID: item.bundleID,
						bundleSlug: item.bundleSlug,
						bundleDisplayName,
						displayName,
						description: preset.description,
						preset,
						label,
					},
				];
			});

			setPresetOptions(nextOptions);
		} catch (refreshError) {
			console.error('Failed to load assistant presets:', refreshError);
			setError(getErrorMessage(refreshError, 'Failed to load assistant presets.'));
			setPresetOptions([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		void refreshPresets();
	}, [refreshPresets]);

	return {
		presetOptions,
		loading,
		error,
		refreshPresets,
	};
}
