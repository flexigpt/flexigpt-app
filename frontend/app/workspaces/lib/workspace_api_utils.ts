import type { ArtifactRef, ArtifactRootID } from '@/spec/artifact';
import type {
	CreateEmptyWorkspaceBody,
	CreateFilesystemWorkspaceBody,
	WorkspaceRef,
	WorkspaceView,
} from '@/spec/workspace';

import { artifactStoreAPI, workspaceAPI } from '@/apis/baseapi';

import { sortWorkspaces } from '@/workspaces/lib/workspace_utils';

const DEFAULT_WORKSPACE_ROOT_DISPLAY_NAME = 'FlexiGPT Workspaces';
const DEFAULT_WORKSPACE_ROOT_DESCRIPTION = 'Technical Artifact Store namespace used for Workspace collections.';

let workspaceRootCreationPromise: Promise<ArtifactRootID> | undefined;

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
	const roots = await artifactStoreAPI.listArtifactRoots();
	if (roots.length === 0) {
		return [];
	}

	const workspaceLists = await Promise.all(roots.map(root => workspaceAPI.listWorkspaces(root.id)));

	return sortWorkspaces(workspaceLists.flat());
}

async function resolveWorkspaceCreationRoot(preferredRootID?: ArtifactRootID): Promise<ArtifactRootID> {
	const roots = await artifactStoreAPI.listArtifactRoots();

	if (preferredRootID) {
		const preferred = roots.find(root => root.id === preferredRootID);
		if (preferred) {
			return preferred.id;
		}
	}

	const defaultRoot = roots.find(
		root => root.displayName.trim().toLowerCase() === DEFAULT_WORKSPACE_ROOT_DISPLAY_NAME.toLowerCase()
	);
	if (defaultRoot) {
		return defaultRoot.id;
	}

	if (roots[0]) {
		return roots[0].id;
	}

	const created = await artifactStoreAPI.createArtifactRoot({
		displayName: DEFAULT_WORKSPACE_ROOT_DISPLAY_NAME,
		description: DEFAULT_WORKSPACE_ROOT_DESCRIPTION,
	});
	return created.id;
}

async function createWorkspaceInResolvedRoot<T>(
	preferredRootID: ArtifactRootID | undefined,
	create: (rootID: ArtifactRootID) => Promise<T>
): Promise<T> {
	const rootPromise = workspaceRootCreationPromise ?? resolveWorkspaceCreationRoot(preferredRootID);
	workspaceRootCreationPromise = rootPromise;

	try {
		return await create(await rootPromise);
	} finally {
		if (workspaceRootCreationPromise === rootPromise) {
			workspaceRootCreationPromise = undefined;
		}
	}
}

export async function createFilesystemWorkspaceCollection(
	body: CreateFilesystemWorkspaceBody,
	preferredRootID?: ArtifactRootID
): Promise<WorkspaceView> {
	return createWorkspaceInResolvedRoot(preferredRootID, rootID => workspaceAPI.createFilesystemWorkspace(rootID, body));
}

export async function createEmptyWorkspaceCollection(
	body: CreateEmptyWorkspaceBody,
	preferredRootID?: ArtifactRootID
): Promise<WorkspaceView> {
	return createWorkspaceInResolvedRoot(preferredRootID, rootID => workspaceAPI.createEmptyWorkspace(rootID, body));
}
