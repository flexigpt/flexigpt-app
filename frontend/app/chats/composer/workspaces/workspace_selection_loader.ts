import type { WorkspaceContextView, WorkspaceRef, WorkspaceSkillView, WorkspaceView } from '@/spec/workspace';

import { workspaceAPI } from '@/apis/baseapi';

function getErrorMessage(error: unknown, fallback: string): string {
	return error instanceof Error && error.message.trim() ? error.message : fallback;
}

export interface LoadedWorkspaceSelectionCatalog {
	workspace?: WorkspaceView;
	catalogRevision?: number;
	contexts: WorkspaceContextView[];
	skills: WorkspaceSkillView[];
	catalogKnown: boolean;
	errors: string[];
}

export async function loadWorkspaceSelectionCatalog(
	workspaceRef: WorkspaceRef
): Promise<LoadedWorkspaceSelectionCatalog> {
	const [workspaceResult, catalogResult, contextsResult, skillsResult] = await Promise.allSettled([
		workspaceAPI.getWorkspace(workspaceRef),
		workspaceAPI.getWorkspaceCatalog(workspaceRef),
		workspaceAPI.listWorkspaceContexts(workspaceRef),
		workspaceAPI.listWorkspaceSkills(workspaceRef),
	]);

	const workspace =
		workspaceResult.status === 'fulfilled'
			? workspaceResult.value
			: catalogResult.status === 'fulfilled'
				? catalogResult.value.workspace
				: undefined;

	const errors: string[] = [];
	if (!workspace) {
		errors.push(
			workspaceResult.status === 'rejected'
				? getErrorMessage(workspaceResult.reason, 'The selected Workspace is unavailable.')
				: 'The selected Workspace is unavailable.'
		);
	}
	if (catalogResult.status === 'rejected') {
		errors.push(getErrorMessage(catalogResult.reason, 'Workspace catalog status could not be loaded.'));
	}
	if (contextsResult.status === 'rejected') {
		errors.push(getErrorMessage(contextsResult.reason, 'Workspace Context records could not be loaded.'));
	}
	if (skillsResult.status === 'rejected') {
		errors.push(getErrorMessage(skillsResult.reason, 'Workspace Skill records could not be loaded.'));
	}

	return {
		workspace,
		catalogRevision: catalogResult.status === 'fulfilled' ? catalogResult.value.catalogRevision : undefined,
		contexts: contextsResult.status === 'fulfilled' ? contextsResult.value : [],
		skills: skillsResult.status === 'fulfilled' ? skillsResult.value : [],
		// Context and Skill Artifact projections determine whether missing
		// selections can be evaluated. Runtime-provider failure is reported
		// separately and makes Workspace Skills unavailable without making the
		// Artifact catalog itself unknown.
		catalogKnown: contextsResult.status === 'fulfilled' && skillsResult.status === 'fulfilled',
		errors,
	};
}
