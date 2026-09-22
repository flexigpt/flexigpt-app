// oxlint-disable typescript/parameter-properties
import type { ArtifactRef, MappedTarget } from '@/spec/artifact';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { ToolRef } from '@/spec/tool';
import type {
	WorkspaceArtifactView,
	WorkspaceDefaultPolicyView,
	WorkspaceDirectoryRef,
	WorkspaceDirectoryView,
	WorkspaceMCPServerLoadPlan,
	WorkspacePage,
	WorkspacePageRequest,
	WorkspacePromptPlan,
	WorkspaceRuntimePlan,
	WorkspaceRuntimeSelection,
	WorkspaceSkillLoadPlan,
} from '@/spec/workspace';

import type {
	IModelPresetStoreAPI,
	IToolStoreAPI,
	IWorkspaceManagementAPI,
	IWorkspaceRuntimeAPI,
	IWorkspaceStoreAPI,
} from '@/apis/interface';

export class WorkspaceManagementAPI implements IWorkspaceManagementAPI {
	constructor(
		public readonly store: IWorkspaceStoreAPI,
		public readonly runtime: IWorkspaceRuntimeAPI,
		private readonly toolStore: IToolStoreAPI,
		private readonly modelPresetStore: IModelPresetStoreAPI
	) {}

	getWorkspaceDefaultPolicy(): Promise<WorkspaceDefaultPolicyView> {
		return this.store.getWorkspaceDefaultPolicy();
	}

	getWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView> {
		return this.store.getWorkspaceDirectory(directory);
	}

	listWorkspaceDirectories(request: WorkspacePageRequest): Promise<WorkspacePage> {
		return this.store.listWorkspaceDirectories(request);
	}

	listWorkspaceDirectoryArtifacts(directory: WorkspaceDirectoryRef): Promise<WorkspaceArtifactView[]> {
		return this.store.listWorkspaceDirectoryArtifacts(directory);
	}

	refreshWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView> {
		return this.store.refreshWorkspaceDirectory(directory);
	}

	registerWorkspaceDirectory(path: string): Promise<WorkspaceDirectoryView> {
		return this.store.registerWorkspaceDirectory(path);
	}

	removeWorkspaceDirectory(directory: WorkspaceDirectoryRef, expectedRevision: number): Promise<void> {
		return this.store.removeWorkspaceDirectory(directory, expectedRevision);
	}

	setWorkspaceDirectoryArtifactEnabled(
		directory: WorkspaceDirectoryRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceArtifactView> {
		return this.store.setWorkspaceDirectoryArtifactEnabled(directory, artifact, expectedRevision, enabled);
	}

	setWorkspaceDirectoryEnabled(
		directory: WorkspaceDirectoryRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceDirectoryView> {
		return this.store.setWorkspaceDirectoryEnabled(directory, expectedRevision, enabled);
	}

	composeWorkspacePrompt(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspacePromptPlan> {
		return this.runtime.composeWorkspacePrompt(workspace, artifacts);
	}

	loadWorkspaceMCPServers(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceMCPServerLoadPlan> {
		return this.runtime.loadWorkspaceMCPServers(workspace, artifacts);
	}

	loadWorkspaceSkills(workspace: ArtifactRef, artifacts: ArtifactRef[]): Promise<WorkspaceSkillLoadPlan> {
		return this.runtime.loadWorkspaceSkills(workspace, artifacts);
	}

	resolveWorkspaceRuntimePlan(
		workspace: ArtifactRef,
		selection: WorkspaceRuntimeSelection
	): Promise<WorkspaceRuntimePlan> {
		return this.runtime.resolveWorkspaceRuntimePlan(workspace, selection);
	}

	resolveMappedToolTarget(target: MappedTarget): Promise<ToolRef> {
		return this.toolStore.resolveMappedToolTarget(target);
	}

	resolveMappedModelTarget(target: MappedTarget): Promise<ModelPresetRef> {
		return this.modelPresetStore.resolveMappedModelTarget(target);
	}
}
