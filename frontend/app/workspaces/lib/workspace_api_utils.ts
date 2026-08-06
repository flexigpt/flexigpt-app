import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import type {
	CreateEmptyWorkspaceBody,
	CreateFilesystemWorkspaceBody,
	WorkspaceRef,
	WorkspaceView,
} from '@/spec/workspace';
import { DEFAULT_WORKSPACE_ROOT_ID } from '@/spec/workspace';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { workspaceAPI } from '@/apis/baseapi';

import { sortWorkspaces } from '@/workspaces/lib/workspace_utils';

export type CreateFilesystemWorkspaceInput = Omit<CreateFilesystemWorkspaceBody, 'workspaceID' | 'sourceID'>;
export type CreateEmptyWorkspaceInput = Omit<CreateEmptyWorkspaceBody, 'workspaceID'>;

export function workspaceRefKey(workspace: WorkspaceRef): string {
	return `${workspace.rootID}:${workspace.collectionID}`;
}

export function artifactRefKey(artifact: ArtifactRef): string {
	return `${artifact.rootID}:${artifact.artifactID}`;
}

export function workspaceRefsEqual(
	left: WorkspaceRef | null | undefined,
	right: WorkspaceRef | null | undefined
): boolean {
	return left?.rootID === right?.rootID && left?.collectionID === right?.collectionID;
}

export async function listAllWorkspaces(): Promise<WorkspaceView[]> {
	return sortWorkspaces(await workspaceAPI.listWorkspaces(DEFAULT_WORKSPACE_ROOT_ID));
}

async function createWorkspaceInResolvedRoot<T>(
	preferredRootID: ArtifactRootID | undefined,
	create: (rootID: ArtifactRootID) => Promise<T>
): Promise<T> {
	const rootID = preferredRootID === DEFAULT_WORKSPACE_ROOT_ID ? preferredRootID : DEFAULT_WORKSPACE_ROOT_ID;
	return create(rootID);
}

export async function createFilesystemWorkspaceCollection(
	body: CreateFilesystemWorkspaceInput,
	preferredRootID?: ArtifactRootID
): Promise<WorkspaceView> {
	return createWorkspaceInResolvedRoot(preferredRootID, rootID =>
		workspaceAPI.createFilesystemWorkspace(rootID, {
			...body,
			workspaceID: getUUIDv7(),
			sourceID: getUUIDv7(),
		})
	);
}

export async function createEmptyWorkspaceCollection(
	body: CreateEmptyWorkspaceInput,
	preferredRootID?: ArtifactRootID
): Promise<WorkspaceView> {
	return createWorkspaceInResolvedRoot(preferredRootID, rootID =>
		workspaceAPI.createEmptyWorkspace(rootID, {
			...body,
			workspaceID: getUUIDv7(),
		})
	);
}
