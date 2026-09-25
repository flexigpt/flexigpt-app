import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';

import { modelPresetStoreAPI } from '@/apis/baseapi';
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
