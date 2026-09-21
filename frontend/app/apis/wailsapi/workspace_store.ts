import type {
	ArtifactRef,
	ArtifactRootID,
	CapabilityPlan,
	StoreArtifact,
	StoreArtifactRoot,
	StoreArtifactRootDraft,
	StoreArtifactSourceSummary,
} from '@/spec/artifact';
import type {
	FilesystemSourceRegistration,
	Workspace,
	WorkspaceArtifactView,
	WorkspaceLoad,
	WorkspacePathRegistration,
	WorkspacePathRegistrationResult,
	WorkspaceRefresh,
} from '@/spec/workspace';

import type { IWorkspaceStoreAPI } from '@/apis/interface';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	AddWorkspacePath,
	CreateWorkspaceRoot,
	GetWorkspace,
	ListWorkspaceArtifacts,
	ListWorkspaceRoots,
	ListWorkspaces,
	LoadWorkspace,
	RefreshWorkspace,
	RegisterFilesystemWorkspaceSource,
	ResolveWorkspaceArtifactCapabilities,
	ResolveWorkspaceCapabilities,
	SetWorkspaceArtifactEnabled,
} from '@/apis/wailsjs/go/main/WorkspaceStoreWrapper';

export class WailsWorkspaceStoreAPI implements IWorkspaceStoreAPI {
	async addWorkspacePath(request: WorkspacePathRegistration): Promise<WorkspacePathRegistrationResult> {
		return requiredObject<WorkspacePathRegistrationResult>(
			await AddWorkspacePath(request as Parameters<typeof AddWorkspacePath>[0]),
			'AddWorkspacePath'
		);
	}

	async createWorkspaceRoot(request: StoreArtifactRootDraft): Promise<StoreArtifactRoot> {
		return requiredObject<StoreArtifactRoot>(
			await CreateWorkspaceRoot(request as Parameters<typeof CreateWorkspaceRoot>[0]),
			'CreateWorkspaceRoot'
		);
	}

	async getWorkspace(workspace: ArtifactRef): Promise<Workspace> {
		return requiredObject<Workspace>(
			await GetWorkspace(workspace as Parameters<typeof GetWorkspace>[0]),
			'GetWorkspace'
		);
	}

	async listWorkspaceArtifacts(workspace: ArtifactRef): Promise<StoreArtifact[]> {
		return wailsObjectArrayOrEmpty<StoreArtifact>(
			await ListWorkspaceArtifacts(workspace as Parameters<typeof ListWorkspaceArtifacts>[0]),
			'ListWorkspaceArtifacts'
		);
	}

	async listWorkspaceRoots(): Promise<StoreArtifactRoot[]> {
		return wailsObjectArrayOrEmpty<StoreArtifactRoot>(await ListWorkspaceRoots(), 'ListWorkspaceRoots');
	}

	async listWorkspaces(rootID: ArtifactRootID): Promise<Workspace[]> {
		return wailsObjectArrayOrEmpty<Workspace>(
			await ListWorkspaces(rootID as Parameters<typeof ListWorkspaces>[0]),
			'ListWorkspaces'
		);
	}

	async loadWorkspace(workspace: ArtifactRef): Promise<WorkspaceLoad> {
		return requiredObject<WorkspaceLoad>(
			await LoadWorkspace(workspace as Parameters<typeof LoadWorkspace>[0]),
			'LoadWorkspace'
		);
	}

	async refreshWorkspace(workspace: ArtifactRef): Promise<WorkspaceRefresh> {
		return requiredObject<WorkspaceRefresh>(
			await RefreshWorkspace(workspace as Parameters<typeof RefreshWorkspace>[0]),
			'RefreshWorkspace'
		);
	}

	async registerFilesystemWorkspaceSource(request: FilesystemSourceRegistration): Promise<StoreArtifactSourceSummary> {
		return requiredObject<StoreArtifactSourceSummary>(
			await RegisterFilesystemWorkspaceSource(request as Parameters<typeof RegisterFilesystemWorkspaceSource>[0]),
			'RegisterFilesystemWorkspaceSource'
		);
	}

	async resolveWorkspaceArtifactCapabilities(artifact: ArtifactRef): Promise<CapabilityPlan> {
		return requiredObject<CapabilityPlan>(
			await ResolveWorkspaceArtifactCapabilities(
				artifact as Parameters<typeof ResolveWorkspaceArtifactCapabilities>[0]
			),
			'ResolveWorkspaceArtifactCapabilities'
		);
	}

	async resolveWorkspaceCapabilities(workspace: ArtifactRef): Promise<CapabilityPlan> {
		return requiredObject<CapabilityPlan>(
			await ResolveWorkspaceCapabilities(workspace as Parameters<typeof ResolveWorkspaceCapabilities>[0]),
			'ResolveWorkspaceCapabilities'
		);
	}

	async setWorkspaceArtifactEnabled(
		workspace: ArtifactRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceArtifactView> {
		return requiredObject<WorkspaceArtifactView>(
			await SetWorkspaceArtifactEnabled(
				workspace as Parameters<typeof SetWorkspaceArtifactEnabled>[0],
				artifact as Parameters<typeof SetWorkspaceArtifactEnabled>[1],
				expectedRevision,
				enabled
			),
			'SetWorkspaceArtifactEnabled'
		);
	}
}
