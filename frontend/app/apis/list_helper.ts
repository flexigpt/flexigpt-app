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
import type { PromptBundle, PromptTemplateKind, PromptTemplateListItem } from '@/spec/prompt';
import type { SkillBundle, SkillListItem, SkillType } from '@/spec/skill';
import type { ToolBundle, ToolListItem } from '@/spec/tool';

import { mcpAPI, modelPresetStoreAPI, promptStoreAPI, skillStoreAPI, toolStoreAPI } from '@/apis/baseapi';

export async function getAllProviderPresetsMap(
	includeDisabled?: boolean
): Promise<Record<ProviderName, ProviderPreset>> {
	let pageToken: string | undefined = undefined;
	const result: Record<ProviderName, ProviderPreset> = {};
	let pageCount = 0;
	const MAX_PAGES = 20;

	do {
		const { providers, nextPageToken } = await modelPresetStoreAPI.listProviderPresets(
			undefined,
			includeDisabled,
			undefined,
			pageToken
		);
		for (const preset of providers) {
			result[preset.name] = preset;
		}
		pageToken = nextPageToken;
		pageCount++;
		if (pageCount >= MAX_PAGES) break;
	} while (pageToken);

	return result;
}

export async function getAllPromptTemplates(
	bundleIDs?: string[],
	tags?: string[],
	includeDisabled?: boolean,
	kinds?: PromptTemplateKind[],
	onlyResolved?: boolean
): Promise<PromptTemplateListItem[]> {
	const all: PromptTemplateListItem[] = [];
	let pageToken: string | undefined;
	const recommendedPageSize = 25;
	do {
		const { promptTemplateListItems, nextPageToken } = await promptStoreAPI.listPromptTemplates(
			bundleIDs,
			tags,
			includeDisabled,
			kinds,
			onlyResolved,
			recommendedPageSize,
			pageToken
		);

		all.push(...promptTemplateListItems);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllPromptBundles(bundleIDs?: string[], includeDisabled?: boolean): Promise<PromptBundle[]> {
	const all: PromptBundle[] = [];
	let pageToken: string | undefined;
	const pageSize = 25;
	do {
		const { promptBundles, nextPageToken } = await promptStoreAPI.listPromptBundles(
			bundleIDs,
			includeDisabled,
			pageSize,
			pageToken
		);

		all.push(...promptBundles);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllTools(
	bundleIDs?: string[],
	tags?: string[],
	includeDisabled?: boolean
): Promise<ToolListItem[]> {
	const all: ToolListItem[] = [];
	let pageToken: string | undefined;
	const recommendedPageSize = 25;
	do {
		const { toolListItems, nextPageToken } = await toolStoreAPI.listTools(
			bundleIDs,
			tags,
			includeDisabled,
			recommendedPageSize,
			pageToken
		);

		all.push(...toolListItems);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllToolBundles(bundleIDs?: string[], includeDisabled?: boolean): Promise<ToolBundle[]> {
	const all: ToolBundle[] = [];

	let pageToken: string | undefined;
	const pageSize = 25;
	do {
		const { toolBundles, nextPageToken } = await toolStoreAPI.listToolBundles(
			bundleIDs,
			includeDisabled,
			pageSize,
			pageToken
		);

		all.push(...toolBundles);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllSkillBundles(bundleIDs?: string[], includeDisabled?: boolean): Promise<SkillBundle[]> {
	const all: SkillBundle[] = [];
	let pageToken: string | undefined = undefined;
	const pageSize = 25;

	do {
		const { skillBundles, nextPageToken } = await skillStoreAPI.listSkillBundles(
			bundleIDs,
			includeDisabled,
			pageSize,
			pageToken
		);
		all.push(...skillBundles);
		if (!nextPageToken) break;
		pageToken = nextPageToken;
	} while (pageToken);
	return all;
}

export async function getAllSkills(
	bundleIDs?: string[],
	types?: SkillType[],
	includeDisabled?: boolean,
	includeMissing?: boolean
): Promise<SkillListItem[]> {
	const all: SkillListItem[] = [];
	let pageToken: string | undefined = undefined;
	const pageSize = 25;

	do {
		const { skillListItems, nextPageToken } = await skillStoreAPI.listSkills(
			bundleIDs,
			types,
			includeDisabled,
			includeMissing,
			pageSize,
			pageToken
		);
		all.push(...skillListItems);
		if (!nextPageToken) break;
		pageToken = nextPageToken;
	} while (pageToken);
	return all;
}

export async function getAllMCPBundles(bundleIDs?: string[], includeDisabled?: boolean): Promise<MCPBundle[]> {
	const all: MCPBundle[] = [];
	let pageToken: string | undefined = undefined;
	const pageSize = 25;

	do {
		const { bundles, nextPageToken } = await mcpAPI.listMCPBundles(bundleIDs, includeDisabled, pageSize, pageToken);

		all.push(...bundles);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllMCPServers(
	bundleID: string,
	serverIDs?: MCPServerID[],
	enabled?: boolean,
	includeDisabled?: boolean
): Promise<MCPServerConfig[]> {
	const all: MCPServerConfig[] = [];
	let pageToken: string | undefined = undefined;
	const pageSize = 25;

	do {
		const { servers, nextPageToken } = await mcpAPI.listMCPServers(
			bundleID,
			serverIDs,
			enabled,
			includeDisabled,
			pageSize,
			pageToken
		);

		all.push(...servers);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllMCPServerTools(bundleID: string, serverID: MCPServerID): Promise<MCPToolCapability[]> {
	const all: MCPToolCapability[] = [];
	let pageToken: string | undefined = undefined;
	const pageSize = 25;

	do {
		const { tools, nextPageToken } = await mcpAPI.listMCPServerTools(bundleID, serverID, pageSize, pageToken);

		all.push(...tools);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllMCPServerResources(bundleID: string, serverID: MCPServerID): Promise<MCPResourceRef[]> {
	const all: MCPResourceRef[] = [];
	let pageToken: string | undefined = undefined;
	const pageSize = 25;

	do {
		const { resources, nextPageToken } = await mcpAPI.listMCPServerResources(bundleID, serverID, pageSize, pageToken);

		all.push(...resources);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllMCPServerResourceTemplates(
	bundleID: string,
	serverID: MCPServerID
): Promise<MCPResourceTemplateRef[]> {
	const all: MCPResourceTemplateRef[] = [];
	let pageToken: string | undefined = undefined;
	const pageSize = 25;

	do {
		const { resourceTemplates, nextPageToken } = await mcpAPI.listMCPServerResourceTemplates(
			bundleID,
			serverID,
			pageSize,
			pageToken
		);

		all.push(...resourceTemplates);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}

export async function getAllMCPServerPrompts(bundleID: string, serverID: MCPServerID): Promise<MCPPromptRef[]> {
	const all: MCPPromptRef[] = [];
	let pageToken: string | undefined = undefined;
	const pageSize = 25;

	do {
		const { prompts, nextPageToken } = await mcpAPI.listMCPServerPrompts(bundleID, serverID, pageSize, pageToken);

		all.push(...prompts);
		pageToken = nextPageToken;
	} while (pageToken);

	return all;
}
