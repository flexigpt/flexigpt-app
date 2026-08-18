import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';
import type { SkillBundle, SkillListItem } from '@/spec/skill';
import type { ToolBundle, ToolListItem } from '@/spec/tool';

import { modelPresetStoreAPI, skillManagementAPI, toolStoreAPI } from '@/apis/baseapi';
import { collectAllPages } from '@/apis/wailsapi/transport';

export async function getAllProviderPresetsMap(
	includeDisabled?: boolean
): Promise<Record<ProviderName, ProviderPreset>> {
	const result: Record<ProviderName, ProviderPreset> = {};
	const providers = await collectAllPages(async pageToken => {
		const page = await modelPresetStoreAPI.listProviderPresets(undefined, includeDisabled, undefined, pageToken);
		return {
			items: page.providers,
			nextPageToken: page.nextPageToken,
		};
	}, 20);

	for (const preset of providers) {
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

export function getAllSkillBundles(bundleIDs?: string[], includeDisabled = true): Promise<SkillBundle[]> {
	return skillManagementAPI.listSkillBundles(bundleIDs, includeDisabled);
}

export function getAllSkills(
	bundleIDs?: string[],
	_tags?: string[],
	includeDisabled = true,
	includeRuntimeMetadata = true
): Promise<SkillListItem[]> {
	return skillManagementAPI.listSkills(bundleIDs, includeDisabled, includeRuntimeMetadata);
}
