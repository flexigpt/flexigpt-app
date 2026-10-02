import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import type {
	AddArtifactMemberRequest,
	AddMemberRequest,
	ArtifactMembershipView,
	CollectionListItem,
	CollectionView,
	CreateCollectionRequest,
	DeleteCollectionRequest,
	RemoveMemberRequest,
	UpdateCollectionRequest,
} from '@/spec/collection';
import type {
	MCPManagementPage,
	MCPPolicyListItem,
	MCPServerListItem,
	MCPServerSecretsView,
	MCPSettings,
	MCPStorePolicyView,
} from '@/spec/mcp';

import type { IMCPStoreAPI } from '@/apis/interface';
import {
	collectionListItemFromWails,
	mcpPolicyListItemFromWails,
	mcpServerListItemFromWails,
} from '@/apis/wailsapi/list_item_projection';
import { optionalWailsString, requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	AddMCPCollectionMember,
	AddMCPServerToCollection,
	CreateMCPCollection,
	DeleteMCPCollection,
	GetMCPCollection,
	GetMCPPolicy,
	GetMCPServerSecrets,
	GetMCPSettings,
	ListMCPCollectionMemberships,
	ListMCPCollections,
	ListMCPCollectionsPage,
	ListMCPPolicies,
	ListMCPServers,
	ListMCPServersPage,
	RemoveMCPCollectionMember,
	SaveMCPSettings,
	SetMCPCollectionEnabled,
	UpdateMCPCollection,
} from '@/apis/wailsjs/go/main/MCPStoreWrapper';

export class WailsMCPStoreAPI implements IMCPStoreAPI {
	async addMCPCollectionMember(request: AddMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await AddMCPCollectionMember(request as Parameters<typeof AddMCPCollectionMember>[0]),
			'AddMCPCollectionMember'
		);
	}

	async addMCPServerToCollection(request: AddArtifactMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await AddMCPServerToCollection(request as Parameters<typeof AddMCPServerToCollection>[0]),
			'AddMCPServerToCollection'
		);
	}

	async createMCPCollection(request: CreateCollectionRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await CreateMCPCollection(request as Parameters<typeof CreateMCPCollection>[0]),
			'CreateMCPCollection'
		);
	}

	async deleteMCPCollection(request: DeleteCollectionRequest): Promise<void> {
		await DeleteMCPCollection(request as Parameters<typeof DeleteMCPCollection>[0]);
	}

	async getMCPCollection(collection: ArtifactRef): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await GetMCPCollection(collection as Parameters<typeof GetMCPCollection>[0]),
			'GetMCPCollection'
		);
	}

	async getMCPPolicy(policy: ArtifactRef): Promise<MCPStorePolicyView> {
		return requiredObject<MCPStorePolicyView>(
			await GetMCPPolicy(policy as Parameters<typeof GetMCPPolicy>[0]),
			'GetMCPPolicy'
		);
	}

	async getMCPServerSecrets(server: ArtifactRef): Promise<MCPServerSecretsView> {
		return requiredObject<MCPServerSecretsView>(
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

	async listMCPCollectionMemberships(artifact: ArtifactRef): Promise<ArtifactMembershipView[]> {
		return wailsObjectArrayOrEmpty<ArtifactMembershipView>(
			await ListMCPCollectionMemberships(artifact as Parameters<typeof ListMCPCollectionMemberships>[0]),
			'ListMCPCollectionMemberships'
		);
	}

	async listMCPCollections(rootID: ArtifactRootID): Promise<CollectionListItem[]> {
		return wailsObjectArrayOrEmpty(
			await ListMCPCollections(rootID as Parameters<typeof ListMCPCollections>[0]),
			'ListMCPCollections'
		).map((value, index) => collectionListItemFromWails(value, `ListMCPCollections[${index}]`));
	}

	async listMCPCollectionsPage(pageSize: number, pageToken = ''): Promise<MCPManagementPage<CollectionListItem>> {
		const page = requiredObject<Record<string, unknown>>(
			await ListMCPCollectionsPage(pageSize, pageToken),
			'ListMCPCollectionsPage'
		);

		return {
			items: wailsObjectArrayOrEmpty(page.items, 'ListMCPCollectionsPage.items').map((value, index) =>
				collectionListItemFromWails(value, `ListMCPCollectionsPage.items[${index}]`)
			),
			nextPageToken: optionalWailsString(page.nextPageToken, 'ListMCPCollectionsPage.nextPageToken') || undefined,
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

	async listMCPServersPage(pageSize: number, pageToken = ''): Promise<MCPManagementPage<MCPServerListItem>> {
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

	async removeMCPCollectionMember(request: RemoveMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await RemoveMCPCollectionMember(request as Parameters<typeof RemoveMCPCollectionMember>[0]),
			'RemoveMCPCollectionMember'
		);
	}

	async setMCPCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await SetMCPCollectionEnabled(
				collection as Parameters<typeof SetMCPCollectionEnabled>[0],
				expectedRevision,
				enabled
			),
			'SetMCPCollectionEnabled'
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

	async updateMCPCollection(request: UpdateCollectionRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await UpdateMCPCollection(request as Parameters<typeof UpdateMCPCollection>[0]),
			'UpdateMCPCollection'
		);
	}
}
