import type { ArtifactRef } from '@/spec/artifact';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { MappedTarget } from '@/spec/resolution';
import type { ToolRef } from '@/spec/tool';
import type { WorkspaceManagementSnapshot } from '@/spec/workspace_store';

import type {
	IModelPresetStoreAPI,
	IToolStoreAPI,
	IWorkspaceAggregateAPI,
	IWorkspaceManagementAPI,
	IWorkspaceRuntimeAPI,
	IWorkspaceStoreAPI,
} from '@/apis/interface';

export class WorkspaceManagementAPI implements IWorkspaceManagementAPI {
	constructor(
		public readonly store: IWorkspaceStoreAPI,
		public readonly runtime: IWorkspaceRuntimeAPI,
		public readonly aggregate: IWorkspaceAggregateAPI,
		private readonly toolStore: IToolStoreAPI,
		private readonly modelPresetStore: IModelPresetStoreAPI
	) {}

	async getWorkspaceManagementSnapshot(workspace: ArtifactRef): Promise<WorkspaceManagementSnapshot> {
		const [load, artifacts] = await Promise.all([
			this.store.loadWorkspace(workspace),
			this.store.listWorkspaceArtifacts(workspace),
		]);

		return {
			workspace: load.Workspace,
			load,
			artifacts,
		};
	}

	resolveMappedToolTarget(target: MappedTarget): Promise<ToolRef> {
		return this.toolStore.resolveMappedToolTarget(target);
	}

	resolveMappedModelTarget(target: MappedTarget): Promise<ModelPresetRef> {
		return this.modelPresetStore.resolveMappedModelTarget(target);
	}
}
