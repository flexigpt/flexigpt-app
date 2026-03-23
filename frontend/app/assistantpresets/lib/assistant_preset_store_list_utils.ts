import type { AssistantPreset, AssistantPresetBundle, AssistantPresetListItem } from '@/spec/assistantpreset';

import { assistantPresetStoreAPI } from '@/apis/baseapi';

async function collectAllPages<TResponse, TItem>(
	fetchPage: (pageToken?: string) => Promise<TResponse>,
	pickItems: (response: TResponse) => TItem[],
	pickNextToken: (response: TResponse) => string | undefined
): Promise<TItem[]> {
	const items: TItem[] = [];
	let nextPageToken: string | undefined = undefined;

	while (true) {
		const response = await fetchPage(nextPageToken);
		items.push(...pickItems(response));

		nextPageToken = pickNextToken(response);
		if (!nextPageToken) {
			break;
		}
	}

	return items;
}

function getBundleLabel(bundle: Pick<AssistantPresetBundle, 'displayName' | 'slug'>): string {
	return (bundle.displayName || bundle.slug).toLowerCase();
}

function getPresetLabel(preset: Pick<AssistantPreset, 'displayName' | 'slug'>): string {
	return (preset.displayName || preset.slug).toLowerCase();
}

function sortAssistantPresetBundles(bundles: AssistantPresetBundle[]): AssistantPresetBundle[] {
	return [...bundles].sort((a, b) => {
		if (a.isBuiltIn !== b.isBuiltIn) {
			return a.isBuiltIn ? 1 : -1;
		}

		const byLabel = getBundleLabel(a).localeCompare(getBundleLabel(b));
		if (byLabel !== 0) {
			return byLabel;
		}

		return a.id.localeCompare(b.id);
	});
}

export function sortAssistantPresets(presets: AssistantPreset[]): AssistantPreset[] {
	return [...presets].sort((a, b) => {
		if (a.isBuiltIn !== b.isBuiltIn) {
			return a.isBuiltIn ? 1 : -1;
		}

		const byDisplay = getPresetLabel(a).localeCompare(getPresetLabel(b));
		if (byDisplay !== 0) {
			return byDisplay;
		}

		const bySlug = a.slug.localeCompare(b.slug);
		if (bySlug !== 0) {
			return bySlug;
		}

		return a.version.localeCompare(b.version);
	});
}

export async function getAllAssistantPresetBundles(
	bundleIDs?: string[],
	includeDisabled = true
): Promise<AssistantPresetBundle[]> {
	const bundles = await collectAllPages(
		pageToken => assistantPresetStoreAPI.listAssistantPresetBundles(bundleIDs, includeDisabled, 200, pageToken),
		response => response.assistantPresetBundles,
		response => response.nextPageToken
	);

	return sortAssistantPresetBundles(bundles);
}

export async function getAllAssistantPresetListItems(
	bundleIDs?: string[],
	includeDisabled = true
): Promise<AssistantPresetListItem[]> {
	const items = await collectAllPages(
		pageToken => assistantPresetStoreAPI.listAssistantPresets(bundleIDs, includeDisabled, 200, pageToken),
		response => response.assistantPresetListItems,
		response => response.nextPageToken
	);

	return [...items].sort((a, b) => {
		if (a.isBuiltIn !== b.isBuiltIn) {
			return a.isBuiltIn ? 1 : -1;
		}

		const byDisplay = (a.displayName || a.assistantPresetSlug).localeCompare(b.displayName || b.assistantPresetSlug);
		if (byDisplay !== 0) {
			return byDisplay;
		}

		const bySlug = a.assistantPresetSlug.localeCompare(b.assistantPresetSlug);
		if (bySlug !== 0) {
			return bySlug;
		}

		return a.assistantPresetVersion.localeCompare(b.assistantPresetVersion);
	});
}
