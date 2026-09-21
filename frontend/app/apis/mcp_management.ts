// oxlint-disable typescript/parameter-properties
import type { ArtifactRef } from '@/spec/artifact';
import type { MCPCollectionManagementView, MCPPolicyManagementView, MCPServerManagementView } from '@/spec/mcp';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { MappedTarget } from '@/spec/resolution';
import type { ToolRef } from '@/spec/tool';

import type {
	IMCPAggregateAPI,
	IMCPManagementAPI,
	IMCPRuntimeAPI,
	IMCPStoreAPI,
	IModelPresetStoreAPI,
	IToolStoreAPI,
} from '@/apis/interface';

export class MCPManagementAPI implements IMCPManagementAPI {
	constructor(
		public readonly store: IMCPStoreAPI,
		public readonly aggregate: IMCPAggregateAPI,
		public readonly runtime: IMCPRuntimeAPI,
		private readonly toolStore: IToolStoreAPI,
		private readonly modelPresetStore: IModelPresetStoreAPI
	) {}

	async getMCPCollectionManagementView(collection: ArtifactRef): Promise<MCPCollectionManagementView> {
		const [view, capabilities] = await Promise.all([
			this.store.getMCPCollection(collection),
			this.store.resolveMCPCollection(collection),
		]);

		return {
			collection: view,
			capabilities,
		};
	}

	async getMCPServerManagementView(server: ArtifactRef): Promise<MCPServerManagementView> {
		const [installation, capabilities, runtimeServerID] = await Promise.all([
			this.store.getMCPServerInstallation(server),
			this.store.resolveMCPArtifactCapabilities(server),
			this.aggregate.runtimeServerIDForArtifact(server),
		]);

		const authHealth = await this.aggregate.getMCPServerAuthHealth(server).catch(() => undefined);

		const runtime = await this.runtime.getMCPServerStatus(runtimeServerID).catch(() => undefined);

		return {
			installation,
			capabilities,
			runtimeServerID,
			authHealth,
			runtime,
		};
	}

	async getMCPPolicyManagementView(policy: ArtifactRef): Promise<MCPPolicyManagementView> {
		const [view, capabilities] = await Promise.all([
			this.store.getMCPPolicy(policy),
			this.store.resolveMCPArtifactCapabilities(policy),
		]);

		return {
			policy: view,
			capabilities,
		};
	}

	resolveMappedToolTarget(target: MappedTarget): Promise<ToolRef> {
		return this.toolStore.resolveMappedToolTarget(target);
	}

	resolveMappedModelTarget(target: MappedTarget): Promise<ModelPresetRef> {
		return this.modelPresetStore.resolveMappedModelTarget(target);
	}
}
