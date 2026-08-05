import type { ProviderName } from '@/spec/inference';
import type {
	MCPBundle,
	MCPPromptRef,
	MCPResourceRef,
	MCPResourceTemplateRef,
	MCPServerConfig,
	MCPServerID,
	MCPToolCapability,
} from '@/spec/mcp';
import type { ProviderPreset } from '@/spec/modelpreset';
import type { SkillBundle, SkillListItem } from '@/spec/skill';
import type { ToolBundle, ToolListItem } from '@/spec/tool';

import { mcpAPI, modelPresetStoreAPI, skillManagementAPI, toolStoreAPI } from '@/apis/baseapi';
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
	_includeRuntimeMetadata = true
): Promise<SkillListItem[]> {
	return skillManagementAPI.listSkills(bundleIDs, includeDisabled);
}

export async function getAllMCPBundles(bundleIDs?: string[], includeDisabled?: boolean): Promise<MCPBundle[]> {
	const pageSize = 25;

	return collectAllPages(async pageToken => {
		const page = await mcpAPI.listMCPBundles(bundleIDs, includeDisabled, pageSize, pageToken);
		return { items: page.bundles, nextPageToken: page.nextPageToken };
	});
}

export async function getAllMCPServers(
	bundleID: string,
	serverIDs?: MCPServerID[],
	enabled?: boolean,
	includeDisabled?: boolean
): Promise<MCPServerConfig[]> {
	const pageSize = 25;

	return collectAllPages(async pageToken => {
		const page = await mcpAPI.listMCPServers(bundleID, serverIDs, enabled, includeDisabled, pageSize, pageToken);
		return { items: page.servers, nextPageToken: page.nextPageToken };
	});
}

export async function getAllMCPServerTools(bundleID: string, serverID: MCPServerID): Promise<MCPToolCapability[]> {
	const pageSize = 25;

	return collectAllPages(async pageToken => {
		const page = await mcpAPI.listMCPServerTools(bundleID, serverID, pageSize, pageToken);
		return { items: page.tools, nextPageToken: page.nextPageToken };
	});
}

export async function getAllMCPServerResources(bundleID: string, serverID: MCPServerID): Promise<MCPResourceRef[]> {
	const pageSize = 25;

	return collectAllPages(async pageToken => {
		const page = await mcpAPI.listMCPServerResources(bundleID, serverID, pageSize, pageToken);
		return { items: page.resources, nextPageToken: page.nextPageToken };
	});
}

export async function getAllMCPServerResourceTemplates(
	bundleID: string,
	serverID: MCPServerID
): Promise<MCPResourceTemplateRef[]> {
	const pageSize = 25;

	return collectAllPages(async pageToken => {
		const page = await mcpAPI.listMCPServerResourceTemplates(bundleID, serverID, pageSize, pageToken);
		return {
			items: page.resourceTemplates,
			nextPageToken: page.nextPageToken,
		};
	});
}

export async function getAllMCPServerPrompts(bundleID: string, serverID: MCPServerID): Promise<MCPPromptRef[]> {
	const pageSize = 25;

	return collectAllPages(async pageToken => {
		const page = await mcpAPI.listMCPServerPrompts(bundleID, serverID, pageSize, pageToken);
		return { items: page.prompts, nextPageToken: page.nextPageToken };
	});
}
