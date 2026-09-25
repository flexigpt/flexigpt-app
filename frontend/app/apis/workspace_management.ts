// oxlint-disable typescript/parameter-properties
import type { ArtifactRef, MappedTarget } from '@/spec/artifact';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { ResolvedToolView } from '@/spec/tool';
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

import { createSharedAsyncCatalog } from '@/lib/shared_async_catalog';

import type {
	IModelPresetStoreAPI,
	IToolTargetResolver,
	IWorkspaceManagementAPI,
	IWorkspaceRuntimeAPI,
	IWorkspaceStoreAPI,
} from '@/apis/interface';

const COMPOSER_WORKSPACE_PAGE_SIZE = 100;
const MAX_COMPOSER_WORKSPACE_PAGE_HOPS = 10_000;

export class WorkspaceManagementAPI implements IWorkspaceManagementAPI {
	constructor(
		public readonly store: IWorkspaceStoreAPI,
		public readonly runtime: IWorkspaceRuntimeAPI,
		private readonly tools: IToolTargetResolver,
		private readonly modelPresetStore: IModelPresetStoreAPI
	) {}

	private readonly composerWorkspaceDirectoriesCatalog = createSharedAsyncCatalog<WorkspaceDirectoryView[]>(() =>
		this.listComposerWorkspaceDirectoriesUncached()
	);

	/**
	 * Shared directory and Workspace declaration catalog.
	 * It intentionally excludes Workspace runtime plans.
	 */
	listComposerWorkspaceDirectories(force = false): Promise<WorkspaceDirectoryView[]> {
		return this.composerWorkspaceDirectoriesCatalog.load(force);
	}

	invalidateComposerWorkspaceCatalog(): void {
		this.composerWorkspaceDirectoriesCatalog.invalidate();
	}

	private async listComposerWorkspaceDirectoriesUncached(): Promise<WorkspaceDirectoryView[]> {
		const output: WorkspaceDirectoryView[] = [];
		const cursors = new Set<string>();
		let cursor: string | undefined;

		for (let hop = 0; hop < MAX_COMPOSER_WORKSPACE_PAGE_HOPS; hop += 1) {
			const page = await this.store.listWorkspaceDirectories({
				cursor,
				limit: COMPOSER_WORKSPACE_PAGE_SIZE,
			});
			output.push(...(page.items ?? []));

			if (!page.nextCursor) {
				return output.toSorted((left, right) =>
					left.root.displayName.localeCompare(right.root.displayName, undefined, {
						sensitivity: 'base',
					})
				);
			}
			if (cursors.has(page.nextCursor)) {
				throw new Error('Workspace directory pagination returned a repeated cursor.');
			}
			cursors.add(page.nextCursor);
			cursor = page.nextCursor;
		}

		throw new Error(`Workspace directory pagination exceeded ${MAX_COMPOSER_WORKSPACE_PAGE_HOPS} pages.`);
	}

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

	async refreshWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView> {
		const refreshed = await this.store.refreshWorkspaceDirectory(directory);
		this.invalidateComposerWorkspaceCatalog();
		return refreshed;
	}

	async registerWorkspaceDirectory(path: string): Promise<WorkspaceDirectoryView> {
		const directory = await this.store.registerWorkspaceDirectory(path);
		this.invalidateComposerWorkspaceCatalog();
		return directory;
	}

	async removeWorkspaceDirectory(directory: WorkspaceDirectoryRef, expectedRevision: number): Promise<void> {
		await this.store.removeWorkspaceDirectory(directory, expectedRevision);
		this.invalidateComposerWorkspaceCatalog();
	}

	async setWorkspaceDirectoryArtifactEnabled(
		directory: WorkspaceDirectoryRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceArtifactView> {
		const updated = await this.store.setWorkspaceDirectoryArtifactEnabled(
			directory,
			artifact,
			expectedRevision,
			enabled
		);
		this.invalidateComposerWorkspaceCatalog();
		return updated;
	}

	async setWorkspaceDirectoryEnabled(
		directory: WorkspaceDirectoryRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceDirectoryView> {
		const updated = await this.store.setWorkspaceDirectoryEnabled(directory, expectedRevision, enabled);
		this.invalidateComposerWorkspaceCatalog();
		return updated;
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

	resolveMappedTool(target: MappedTarget): Promise<ResolvedToolView> {
		return this.tools.resolveMappedTool(target);
	}

	resolveMappedModelTarget(target: MappedTarget): Promise<ModelPresetRef> {
		return this.modelPresetStore.resolveMappedModelTarget(target);
	}
}
