import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import type {
	MCPPluginPage,
	MCPPolicyListItem,
	MCPPolicyView,
	MCPServerListItem,
	MCPServerPage,
	MCPServerSecretsView,
	MCPSettings,
} from '@/spec/mcp';
import type {
	AddPluginArtifactMemberRequest,
	AddPluginMemberRequest,
	ArtifactPluginMembershipView,
	CreatePluginRequest,
	DeletePluginRequest,
	PluginListItem,
	PluginView,
	RemovePluginMemberRequest,
	UpdatePluginRequest,
} from '@/spec/plugin';

import type { IMCPStoreAPI } from '@/apis/interface';
import {
	artifactPluginMembershipFromWails,
	mcpPolicyListItemFromWails,
	mcpServerListItemFromWails,
	pluginListItemFromWails,
	pluginViewFromWails,
} from '@/apis/wailsapi/list_item_projection';
import { optionalWailsString, requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	AddMCPPluginMember,
	AddMCPServerToPlugin,
	CreateMCPPlugin,
	DeleteMCPPlugin,
	GetMCPPlugin,
	GetMCPPolicy,
	GetMCPServerSecrets,
	GetMCPSettings,
	ListMCPPluginMemberships,
	ListMCPPlugins,
	ListMCPPluginsPage,
	ListMCPPolicies,
	ListMCPServers,
	ListMCPServersPage,
	RemoveMCPPluginMember,
	SaveMCPSettings,
	SetMCPPluginEnabled,
	UpdateMCPPlugin,
} from '@/apis/wailsjs/go/main/MCPStoreWrapper';

function mcpServerSecretsViewFromWails(value: unknown, operation: string): MCPServerSecretsView {
	const view = requiredObject<MCPServerSecretsView>(value, operation);

	return {
		...view,
		inputs: wailsObjectArrayOrEmpty<MCPServerSecretsView['inputs'][number]>(view.inputs, `${operation}.inputs`),
	};
}

export class WailsMCPStoreAPI implements IMCPStoreAPI {
	async addMCPPluginMember(request: AddPluginMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await AddMCPPluginMember(request as Parameters<typeof AddMCPPluginMember>[0]),
			'AddMCPPluginMember'
		);
	}

	async addMCPServerToPlugin(request: AddPluginArtifactMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await AddMCPServerToPlugin(request as Parameters<typeof AddMCPServerToPlugin>[0]),
			'AddMCPServerToPlugin'
		);
	}

	async createMCPPlugin(request: CreatePluginRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await CreateMCPPlugin(request as Parameters<typeof CreateMCPPlugin>[0]),
			'CreateMCPPlugin'
		);
	}

	async deleteMCPPlugin(request: DeletePluginRequest): Promise<void> {
		await DeleteMCPPlugin(request as Parameters<typeof DeleteMCPPlugin>[0]);
	}

	async getMCPPlugin(plugin: ArtifactRef): Promise<PluginView> {
		return pluginViewFromWails(await GetMCPPlugin(plugin as Parameters<typeof GetMCPPlugin>[0]), 'GetMCPPlugin');
	}

	async getMCPPolicy(policy: ArtifactRef): Promise<MCPPolicyView> {
		return requiredObject<MCPPolicyView>(
			await GetMCPPolicy(policy as Parameters<typeof GetMCPPolicy>[0]),
			'GetMCPPolicy'
		);
	}

	async getMCPServerSecrets(server: ArtifactRef): Promise<MCPServerSecretsView> {
		return mcpServerSecretsViewFromWails(
			await GetMCPServerSecrets(server as Parameters<typeof GetMCPServerSecrets>[0]),
			'GetMCPServerSecrets'
		);
	}

	async getMCPSettings(): Promise<MCPSettings> {
		const value = requiredObject<{ settings: { oauthLoopbackListenAddr?: string }; revision: number }>(
			await GetMCPSettings(),
			'GetMCPSettings'
		);
		return { revision: value.revision, oauthLoopbackListenAddr: value.settings.oauthLoopbackListenAddr };
	}

	async listMCPPluginMemberships(artifact: ArtifactRef): Promise<ArtifactPluginMembershipView[]> {
		return wailsObjectArrayOrEmpty(
			await ListMCPPluginMemberships(artifact as Parameters<typeof ListMCPPluginMemberships>[0]),
			'ListMCPPluginMemberships'
		).map((value, index) => artifactPluginMembershipFromWails(value, `ListMCPPluginMemberships[${index}]`));
	}

	async listMCPPlugins(rootID: ArtifactRootID): Promise<PluginListItem[]> {
		return wailsObjectArrayOrEmpty(
			await ListMCPPlugins(rootID as Parameters<typeof ListMCPPlugins>[0]),
			'ListMCPPlugins'
		).map((value, index) => pluginListItemFromWails(value, `ListMCPPlugins[${index}]`));
	}

	async listMCPPluginsPage(pageSize: number, pageToken = ''): Promise<MCPPluginPage> {
		const page = requiredObject<Record<string, unknown>>(
			await ListMCPPluginsPage(pageSize, pageToken),
			'ListMCPPluginsPage'
		);

		return {
			items: wailsObjectArrayOrEmpty(page.items, 'ListMCPPluginsPage.items').map((value, index) =>
				pluginListItemFromWails(value, `ListMCPPluginsPage.items[${index}]`)
			),
			nextPageToken: optionalWailsString(page.nextPageToken, 'ListMCPPluginsPage.nextPageToken') || undefined,
		};
	}

	async listMCPPolicies(rootID: ArtifactRootID): Promise<MCPPolicyListItem[]> {
		return wailsObjectArrayOrEmpty(
			await ListMCPPolicies(rootID as Parameters<typeof ListMCPPolicies>[0]),
			'ListMCPPolicies'
		).map((value, index) => mcpPolicyListItemFromWails(value, `ListMCPPolicies[${index}]`));
	}

	async listMCPServers(rootID: ArtifactRootID): Promise<MCPServerListItem[]> {
		return wailsObjectArrayOrEmpty(
			await ListMCPServers(rootID as Parameters<typeof ListMCPServers>[0]),
			'ListMCPServers'
		).map((value, index) => mcpServerListItemFromWails(value, `ListMCPServers[${index}]`));
	}

	async listMCPServersPage(pageSize: number, pageToken = ''): Promise<MCPServerPage> {
		const page = requiredObject<Record<string, unknown>>(
			await ListMCPServersPage(pageSize, pageToken),
			'ListMCPServersPage'
		);

		return {
			items: wailsObjectArrayOrEmpty(page.items, 'ListMCPServersPage.items').map((value, index) =>
				mcpServerListItemFromWails(value, `ListMCPServersPage.items[${index}]`)
			),
			nextPageToken: optionalWailsString(page.nextPageToken, 'ListMCPServersPage.nextPageToken') || undefined,
		};
	}

	async removeMCPPluginMember(request: RemovePluginMemberRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await RemoveMCPPluginMember(request as Parameters<typeof RemoveMCPPluginMember>[0]),
			'RemoveMCPPluginMember'
		);
	}

	async setMCPPluginEnabled(plugin: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<PluginView> {
		return pluginViewFromWails(
			await SetMCPPluginEnabled(plugin as Parameters<typeof SetMCPPluginEnabled>[0], expectedRevision, enabled),
			'SetMCPPluginEnabled'
		);
	}

	async saveMCPSettings(
		expectedRevision: number,
		settings: { oauthLoopbackListenAddr?: string }
	): Promise<MCPSettings> {
		const value = requiredObject<{ settings: { oauthLoopbackListenAddr?: string }; revision: number }>(
			await SaveMCPSettings(expectedRevision, settings as Parameters<typeof SaveMCPSettings>[1]),
			'SaveMCPSettings'
		);
		return { revision: value.revision, oauthLoopbackListenAddr: value.settings.oauthLoopbackListenAddr };
	}

	async updateMCPPlugin(request: UpdatePluginRequest): Promise<PluginView> {
		return pluginViewFromWails(
			await UpdateMCPPlugin(request as Parameters<typeof UpdateMCPPlugin>[0]),
			'UpdateMCPPlugin'
		);
	}
}
