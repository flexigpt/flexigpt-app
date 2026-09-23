import type {
	AgentExportResult,
	AgentImportCommitRequest,
	AgentImportCommitResult,
	AgentImportDestination,
	AgentImportPreview,
	AgentImportPreviewRequest,
	AgentResolution,
	AgentTextMaterialization,
	AgentView,
	ListAgentsRequest,
} from '@/spec/agent';
import type { ArtifactRef, ArtifactRootID, CapabilityPlan, StoreArtifact } from '@/spec/artifact';
import type {
	CollectionCapabilityPlan,
	CollectionView,
	CreateCollectionRequest,
	UpdateCollectionRequest,
} from '@/spec/collection';

import type { IAgentStoreAPI } from '@/apis/interface';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	CommitAgentImport,
	CreateAgentCollection,
	DeleteAgentCollection,
	DeleteManagedAgent,
	ExportAgent,
	GetAgent,
	GetAgentCollection,
	ListAgentCollectionMembers,
	ListAgentCollections,
	ListAgentImportDestinations,
	ListAgents,
	ListAgentsForManagement,
	MaterializeAgentText,
	PreviewAgentImport,
	ResolveAgent,
	ResolveAgentCapabilities,
	SetAgentCollectionEnabled,
	SetAgentEnabled,
	UpdateAgentCollection,
} from '@/apis/wailsjs/go/main/AgentStoreWrapper';

export class WailsAgentStoreAPI implements IAgentStoreAPI {
	async listAgents(request: ListAgentsRequest): Promise<AgentView[]> {
		return wailsObjectArrayOrEmpty<AgentView>(
			await ListAgents(request as Parameters<typeof ListAgents>[0]),
			'ListAgents'
		);
	}

	async listAgentsForManagement(): Promise<AgentView[]> {
		return wailsObjectArrayOrEmpty<AgentView>(await ListAgentsForManagement(), 'ListAgentsForManagement');
	}

	async getAgent(agent: ArtifactRef): Promise<AgentView> {
		return requiredObject<AgentView>(await GetAgent(agent as Parameters<typeof GetAgent>[0]), 'GetAgent');
	}

	async materializeAgentText(text: ArtifactRef): Promise<AgentTextMaterialization> {
		return requiredObject<AgentTextMaterialization>(
			await MaterializeAgentText(text as Parameters<typeof MaterializeAgentText>[0]),
			'MaterializeAgentText'
		);
	}

	async resolveAgent(agent: ArtifactRef): Promise<AgentResolution> {
		return requiredObject<AgentResolution>(
			await ResolveAgent(agent as Parameters<typeof ResolveAgent>[0]),
			'ResolveAgent'
		);
	}

	async resolveAgentCapabilities(agent: ArtifactRef): Promise<CapabilityPlan> {
		return requiredObject<CapabilityPlan>(
			await ResolveAgentCapabilities(agent as Parameters<typeof ResolveAgentCapabilities>[0]),
			'ResolveAgentCapabilities'
		);
	}

	async setAgentEnabled(agent: ArtifactRef, expectedRevision: number, enabled: boolean): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(
			await SetAgentEnabled(agent as Parameters<typeof SetAgentEnabled>[0], expectedRevision, enabled),
			'SetAgentEnabled'
		);
	}

	async listAgentCollectionMembers(collection: ArtifactRef): Promise<CollectionCapabilityPlan> {
		return requiredObject<CollectionCapabilityPlan>(
			await ListAgentCollectionMembers(collection as Parameters<typeof ListAgentCollectionMembers>[0]),
			'ListAgentCollectionMembers'
		);
	}

	async createAgentCollection(request: CreateCollectionRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await CreateAgentCollection(request as Parameters<typeof CreateAgentCollection>[0]),
			'CreateAgentCollection'
		);
	}

	async getAgentCollection(collection: ArtifactRef): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await GetAgentCollection(collection as Parameters<typeof GetAgentCollection>[0]),
			'GetAgentCollection'
		);
	}

	async listAgentCollections(rootID: ArtifactRootID): Promise<CollectionView[]> {
		return wailsObjectArrayOrEmpty<CollectionView>(
			await ListAgentCollections(rootID as Parameters<typeof ListAgentCollections>[0]),
			'ListAgentCollections'
		);
	}

	async updateAgentCollection(request: UpdateCollectionRequest): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await UpdateAgentCollection(request as Parameters<typeof UpdateAgentCollection>[0]),
			'UpdateAgentCollection'
		);
	}

	async setAgentCollectionEnabled(
		collection: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<CollectionView> {
		return requiredObject<CollectionView>(
			await SetAgentCollectionEnabled(
				collection as Parameters<typeof SetAgentCollectionEnabled>[0],
				expectedRevision,
				enabled
			),
			'SetAgentCollectionEnabled'
		);
	}

	async deleteAgentCollection(collection: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteAgentCollection(collection as Parameters<typeof DeleteAgentCollection>[0], expectedRevision);
	}

	async listAgentImportDestinations(): Promise<AgentImportDestination[]> {
		return wailsObjectArrayOrEmpty<AgentImportDestination>(
			await ListAgentImportDestinations(),
			'ListAgentImportDestinations'
		);
	}

	async previewAgentImport(request: AgentImportPreviewRequest): Promise<AgentImportPreview> {
		return requiredObject<AgentImportPreview>(
			await PreviewAgentImport(request as Parameters<typeof PreviewAgentImport>[0]),
			'PreviewAgentImport'
		);
	}

	async commitAgentImport(request: AgentImportCommitRequest): Promise<AgentImportCommitResult> {
		return requiredObject<AgentImportCommitResult>(
			await CommitAgentImport(request as Parameters<typeof CommitAgentImport>[0]),
			'CommitAgentImport'
		);
	}

	async exportAgent(agent: ArtifactRef): Promise<AgentExportResult> {
		return requiredObject<AgentExportResult>(
			await ExportAgent({ agent } as Parameters<typeof ExportAgent>[0]),
			'ExportAgent'
		);
	}

	async deleteManagedAgent(agent: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteManagedAgent({
			agent,
			expectedRevision,
		} as Parameters<typeof DeleteManagedAgent>[0]);
	}
}
