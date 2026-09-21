// oxlint-disable typescript/parameter-properties
import type { AgentManagementView } from '@/spec/agent';
import type { ArtifactRef, MappedTarget } from '@/spec/artifact';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { ToolRef } from '@/spec/tool';

import type { IAgentManagementAPI, IAgentStoreAPI, IModelPresetStoreAPI, IToolStoreAPI } from '@/apis/interface';

export class AgentManagementAPI implements IAgentManagementAPI {
	constructor(
		private readonly store: IAgentStoreAPI,
		private readonly toolStore: IToolStoreAPI,
		private readonly modelPresetStore: IModelPresetStoreAPI
	) {}

	async getAgentManagementView(agent: ArtifactRef): Promise<AgentManagementView> {
		const [agentView, directMemberships, capabilities] = await Promise.all([
			this.store.getAgent(agent),
			this.store.listDirectAgentMemberships(agent),
			this.store.resolveAgentCapabilities(agent),
		]);

		return {
			agent: agentView,
			directMemberships,
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
