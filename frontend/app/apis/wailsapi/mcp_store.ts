import type { ArtifactRef, ArtifactRootID, CapabilityPlan, StoreArtifact } from '@/spec/artifact';
import type {
	AddArtifactMemberRequest,
	AddMemberRequest,
	ArtifactMembershipView,
	CollectionCapabilityPlan,
	CollectionView,
	CreateCollectionRequest,
	DeleteCollectionRequest,
	RemoveMemberRequest,
	UpdateCollectionRequest,
} from '@/spec/collection';
import type {
	MCPEffectivePolicy,
	MCPManagementPage,
	MCPStorePolicyView,
	MCPStoreServerInstallationView,
} from '@/spec/mcp';

import type { IMCPStoreAPI } from '@/apis/interface';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	AddMCPCollectionMember,
	AttachMCPArtifactToCollection,
	CreateMCPCollection,
	DeleteMCPCollection,
	GetMCPCollection,
	GetMCPPolicy,
	GetMCPServerInstallation,
	ListMCPCollectionMemberships,
	ListMCPCollections,
	ListMCPCollectionServers,
	ListMCPCollectionsPage,
	ListMCPPolicies,
	ListMCPServers,
	ListMCPServersPage,
	RemoveMCPCollectionMember,
	ResolveMCPArtifactCapabilities,
	ResolveMCPCollection,
	SetMCPCollectionEnabled,
	SetMCPPolicyEnabled,
	SetMCPServerEnabled,
	UpdateMCPCollection,
} from '@/apis/wailsjs/go/main/MCPStoreWrapper';

export class WailsMCPStoreAPI implements IMCPStoreAPI {
	async addMCPCollectionMember(request: AddMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await AddMCPCollectionMember(request as Parameters<typeof AddMCPCollectionMember>[0]),
			'AddMCPCollectionMember'
		);
	}

	async attachMCPArtifactToCollection(request: AddArtifactMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await AttachMCPArtifactToCollection(request as Parameters<typeof AttachMCPArtifactToCollection>[0]),
			'AttachMCPArtifactToCollection'
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

	async getMCPServerInstallation(server: ArtifactRef): Promise<MCPStoreServerInstallationView> {
		return requiredObject<MCPStoreServerInstallationView>(
			await GetMCPServerInstallation(server as Parameters<typeof GetMCPServerInstallation>[0]),
			'GetMCPServerInstallation'
		);
	}

	async listMCPCollectionMemberships(artifact: ArtifactRef): Promise<ArtifactMembershipView[]> {
		return wailsObjectArrayOrEmpty<ArtifactMembershipView>(
			await ListMCPCollectionMemberships(artifact as Parameters<typeof ListMCPCollectionMemberships>[0]),
			'ListMCPCollectionMemberships'
		);
	}

	async listMCPCollections(rootID: ArtifactRootID): Promise<CollectionView[]> {
		return wailsObjectArrayOrEmpty<CollectionView>(
			await ListMCPCollections(rootID as Parameters<typeof ListMCPCollections>[0]),
			'ListMCPCollections'
		);
	}

	async listMCPCollectionsPage(pageSize: number, pageToken = ''): Promise<MCPManagementPage<CollectionView>> {
		return requiredObject<MCPManagementPage<CollectionView>>(
			await ListMCPCollectionsPage(pageSize, pageToken),
			'ListMCPCollectionsPage'
		);
	}

	async listMCPPolicies(rootID: ArtifactRootID): Promise<StoreArtifact[]> {
		return wailsObjectArrayOrEmpty<StoreArtifact>(
			await ListMCPPolicies(rootID as Parameters<typeof ListMCPPolicies>[0]),
			'ListMCPPolicies'
		);
	}

	async listMCPServers(rootID: ArtifactRootID): Promise<StoreArtifact[]> {
		return wailsObjectArrayOrEmpty<StoreArtifact>(
			await ListMCPServers(rootID as Parameters<typeof ListMCPServers>[0]),
			'ListMCPServers'
		);
	}

	async listMCPServersPage(pageSize: number, pageToken = ''): Promise<MCPManagementPage<StoreArtifact>> {
		return requiredObject<MCPManagementPage<StoreArtifact>>(
			await ListMCPServersPage(pageSize, pageToken),
			'ListMCPServersPage'
		);
	}

	async removeMCPCollectionMember(request: RemoveMemberRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await RemoveMCPCollectionMember(request as Parameters<typeof RemoveMCPCollectionMember>[0]),
			'RemoveMCPCollectionMember'
		);
	}

	async resolveMCPArtifactCapabilities(artifact: ArtifactRef): Promise<CapabilityPlan> {
		return requiredObject<CapabilityPlan>(
			await ResolveMCPArtifactCapabilities(artifact as Parameters<typeof ResolveMCPArtifactCapabilities>[0]),
			'ResolveMCPArtifactCapabilities'
		);
	}

	async resolveMCPCollection(collection: ArtifactRef): Promise<CollectionCapabilityPlan> {
		return requiredObject<CollectionCapabilityPlan>(
			await ResolveMCPCollection(collection as Parameters<typeof ResolveMCPCollection>[0]),
			'ResolveMCPCollection'
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

	async setMCPPolicyEnabled(policy: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(
			await SetMCPPolicyEnabled(policy as Parameters<typeof SetMCPPolicyEnabled>[0], expectedRevision, enabled),
			'SetMCPPolicyEnabled'
		);
	}

	async setMCPServerEnabled(server: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(
			await SetMCPServerEnabled(server as Parameters<typeof SetMCPServerEnabled>[0], expectedRevision, enabled),
			'SetMCPServerEnabled'
		);
	}

	async updateMCPCollection(request: UpdateCollectionRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await UpdateMCPCollection(request as Parameters<typeof UpdateMCPCollection>[0]),
			'UpdateMCPCollection'
		);
	}

	async listMCPCollectionServers(collection: ArtifactRef): Promise<
		Array<{
			installation: MCPStoreServerInstallationView;
			policy: MCPEffectivePolicy;
		}>
	> {
		return wailsObjectArrayOrEmpty<{
			installation: MCPStoreServerInstallationView;
			policy: MCPEffectivePolicy;
		}>(
			await ListMCPCollectionServers(collection as Parameters<typeof ListMCPCollectionServers>[0]),
			'ListMCPCollectionServers'
		);
	}
}
