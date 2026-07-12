import type { AssistantPreset, AssistantPresetBundle, AssistantPresetListItem } from '@/spec/assistantpreset';

import { compareVersionStrings } from '@/lib/version_utils';

import { assistantPresetStoreAPI } from '@/apis/baseapi';

const MAX_PAGE_COUNT = 1_000;

async function collectAllPages<TResponse, TItem>(
	fetchPage: (pageToken?: string) => Promise<TResponse>,
	pickItems: (response: TResponse) => TItem[],
	pickNextToken: (response: TResponse) => string | undefined
): Promise<TItem[]> {
	const items: TItem[] = [];
	const seenPageTokens = new Set<string>();
	let nextPageToken: string | undefined = undefined;

	for (let page = 0; page < MAX_PAGE_COUNT; page += 1) {
		if (nextPageToken) {
			if (seenPageTokens.has(nextPageToken)) {
				throw new Error('Assistant preset pagination returned a repeated page token.');
			}
			seenPageTokens.add(nextPageToken);
		}

		const response = await fetchPage(nextPageToken);
		items.push(...pickItems(response));

		nextPageToken = pickNextToken(response);
		if (!nextPageToken) {
			return items;
		}
	}

	throw new Error(`Assistant preset pagination exceeded ${MAX_PAGE_COUNT} pages.`);
}

function getBundleLabel(bundle: Pick<AssistantPresetBundle, 'displayName' | 'slug'>): string {
	return (bundle.displayName || bundle.slug).toLowerCase();
}

function getPresetLabel(preset: Pick<AssistantPreset, 'displayName' | 'slug'>): string {
	return (preset.displayName || preset.slug).toLowerCase();
}

function sortAssistantPresetBundles(bundles: AssistantPresetBundle[]): AssistantPresetBundle[] {
	return [...bundles].toSorted((a, b) => {
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
	return [...presets].toSorted((a, b) => {
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

		return compareVersionStrings(a.version, b.version);
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

	return [...items].toSorted((a, b) => {
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

		return compareVersionStrings(a.assistantPresetVersion, b.assistantPresetVersion);
	});
}
