import type { WorkspaceDirectoryView, WorkspaceDirectoryWorkspace } from '@/spec/workspace';

function normalized(value: string | undefined): string {
	return value?.trim() ?? '';
}

function workspaceRepositoryName(directory: WorkspaceDirectoryView): string {
	return normalized(directory.root.displayName) || 'Repository';
}

function workspaceName(workspace: WorkspaceDirectoryWorkspace): string {
	return (
		normalized(workspace.workspace.artifact.displayName) ||
		normalized(workspace.workspace.artifact.logicalName) ||
		'Workspace'
	);
}

export function workspaceDisplayName(
	directory: WorkspaceDirectoryView,
	workspace: WorkspaceDirectoryWorkspace
): string {
	const repository = workspaceRepositoryName(directory);
	const name = workspaceName(workspace);

	return repository.localeCompare(name, undefined, { sensitivity: 'base' }) === 0
		? repository
		: `${repository} / ${name}`;
}

export function workspaceRef(workspace: WorkspaceDirectoryWorkspace) {
	return {
		rootID: workspace.workspace.artifact.rootID,
		artifactID: workspace.workspace.artifact.id,
	};
}
