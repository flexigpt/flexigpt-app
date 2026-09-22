import type { ArtifactRef } from '@/spec/artifact';
import type {
	WorkspaceArtifactView,
	WorkspaceDefaultPolicyView,
	WorkspaceDirectoryRef,
	WorkspaceDirectoryView,
	WorkspaceDirectoryWorkspace,
	WorkspacePage,
	WorkspacePageRequest,
} from '@/spec/workspace';
import { WorkspaceDirectoryOrigin } from '@/spec/workspace';

import type { IWorkspaceStoreAPI } from '@/apis/interface';
import { enumFromWails, requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	GetWorkspaceDefaultPolicy,
	GetWorkspaceDirectory,
	ListWorkspaceDirectories,
	ListWorkspaceDirectoryArtifacts,
	RefreshWorkspaceDirectory,
	RegisterWorkspaceDirectory,
	RemoveWorkspaceDirectory,
	SetWorkspaceDirectoryArtifactEnabled,
	SetWorkspaceDirectoryEnabled,
} from '@/apis/wailsjs/go/main/WorkspaceStoreWrapper';

function projectWorkspaceDirectory(value: unknown, operation: string): WorkspaceDirectoryView {
	const directory = requiredObject<WorkspaceDirectoryView>(value, operation);
	const workspaces = directory.workspaces as WorkspaceDirectoryWorkspace[];
	for (const workspace of workspaces) {
		workspace.origin = enumFromWails(workspace.origin, WorkspaceDirectoryOrigin, `${operation}.workspaces.origin`);
	}

	return directory;
}

export class WailsWorkspaceStoreAPI implements IWorkspaceStoreAPI {
	async getWorkspaceDefaultPolicy(): Promise<WorkspaceDefaultPolicyView> {
		return requiredObject<WorkspaceDefaultPolicyView>(await GetWorkspaceDefaultPolicy(), 'GetWorkspaceDefaultPolicy');
	}

	async getWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView> {
		return projectWorkspaceDirectory(
			await GetWorkspaceDirectory(directory as Parameters<typeof GetWorkspaceDirectory>[0]),
			'GetWorkspaceDirectory'
		);
	}

	async listWorkspaceDirectories(request: WorkspacePageRequest): Promise<WorkspacePage> {
		const page = requiredObject<WorkspacePage>(
			await ListWorkspaceDirectories(request as Parameters<typeof ListWorkspaceDirectories>[0]),
			'ListWorkspaceDirectories'
		);

		page.items = page.items.map(directory => projectWorkspaceDirectory(directory, 'ListWorkspaceDirectories.items'));

		return page;
	}

	async listWorkspaceDirectoryArtifacts(directory: WorkspaceDirectoryRef): Promise<WorkspaceArtifactView[]> {
		return wailsObjectArrayOrEmpty<WorkspaceArtifactView>(
			await ListWorkspaceDirectoryArtifacts(directory as Parameters<typeof ListWorkspaceDirectoryArtifacts>[0]),
			'ListWorkspaceDirectoryArtifacts'
		);
	}

	async refreshWorkspaceDirectory(directory: WorkspaceDirectoryRef): Promise<WorkspaceDirectoryView> {
		return projectWorkspaceDirectory(
			await RefreshWorkspaceDirectory(directory as Parameters<typeof RefreshWorkspaceDirectory>[0]),
			'RefreshWorkspaceDirectory'
		);
	}

	async registerWorkspaceDirectory(path: string): Promise<WorkspaceDirectoryView> {
		return projectWorkspaceDirectory(await RegisterWorkspaceDirectory(path), 'RegisterWorkspaceDirectory');
	}

	async removeWorkspaceDirectory(directory: WorkspaceDirectoryRef, expectedRevision: number): Promise<void> {
		await RemoveWorkspaceDirectory(directory as Parameters<typeof RemoveWorkspaceDirectory>[0], expectedRevision);
	}

	async setWorkspaceDirectoryArtifactEnabled(
		directory: WorkspaceDirectoryRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceArtifactView> {
		return requiredObject<WorkspaceArtifactView>(
			await SetWorkspaceDirectoryArtifactEnabled(
				directory as Parameters<typeof SetWorkspaceDirectoryArtifactEnabled>[0],
				artifact as Parameters<typeof SetWorkspaceDirectoryArtifactEnabled>[1],
				expectedRevision,
				enabled
			),
			'SetWorkspaceDirectoryArtifactEnabled'
		);
	}

	async setWorkspaceDirectoryEnabled(
		directory: WorkspaceDirectoryRef,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceDirectoryView> {
		return projectWorkspaceDirectory(
			await SetWorkspaceDirectoryEnabled(
				directory as Parameters<typeof SetWorkspaceDirectoryEnabled>[0],
				expectedRevision,
				enabled
			),
			'SetWorkspaceDirectoryEnabled'
		);
	}
}
