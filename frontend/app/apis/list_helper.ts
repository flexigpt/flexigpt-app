import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';
import type { ToolBundle, ToolListItem } from '@/spec/tool';

import { modelPresetStoreAPI, toolStoreAPI } from '@/apis/baseapi';
import { collectAllPages } from '@/apis/wailsapi/transport';

export async function getAllProviderPresetsMap(
	includeDisabled?: boolean
): Promise<Record<ProviderName, ProviderPreset>> {
	const result = Object.create(null) as Record<ProviderName, ProviderPreset>;
	const providers = await collectAllPages(async pageToken => {
		const page = await modelPresetStoreAPI.listProviderPresets(undefined, includeDisabled, undefined, pageToken);
		return {
			items: page.providers,
			nextPageToken: page.nextPageToken,
		};
	}, 20);

	for (const preset of providers) {
		if (Object.hasOwn(result, preset.name)) {
			throw new Error(`Provider preset ${preset.name} was returned more than once.`);
		}

		result[preset.name] = preset;
	}

	return result;
}

export async function getAllTools(
	bundleIDs?: string[],
	tags?: string[],
	includeDisabled?: boolean
): Promise<ToolListItem[]> {
	const recommendedPageSize = 25;

	return collectAllPages(async pageToken => {
		const page = await toolStoreAPI.listTools(bundleIDs, tags, includeDisabled, recommendedPageSize, pageToken);
		return {
			items: page.toolListItems,
			nextPageToken: page.nextPageToken,
		};
	});
}

export async function getAllToolBundles(bundleIDs?: string[], includeDisabled?: boolean): Promise<ToolBundle[]> {
	const pageSize = 25;

	return collectAllPages(async pageToken => {
		const page = await toolStoreAPI.listToolBundles(bundleIDs, includeDisabled, pageSize, pageToken);
		return {
			items: page.toolBundles,
			nextPageToken: page.nextPageToken,
		};
	});
}
