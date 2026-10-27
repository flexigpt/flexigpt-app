import type { WorkspaceDirectoryRef, WorkspaceRef, WorkspaceSourceExport } from '@/spec/workspace';

import { workspaceManagementAPI } from '@/apis/baseapi';

import type { ManagementArtifactSourceValue } from '@/components/managementui/management_artifact_source';
import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';

import { workspaceSourceExtension } from '@/workspaces/workspace_source_utils';

async function loadWorkspaceSource(
	directory: WorkspaceDirectoryRef,
	workspace: WorkspaceRef
): Promise<ManagementArtifactSourceValue<WorkspaceSourceExport>> {
	const source = await workspaceManagementAPI.exportWorkspaceSource(directory, workspace);
	const extension = workspaceSourceExtension(source.format);

	return {
		data: source,
		text: source.content,
		fileName: source.suggestedFileName || `workspace.${extension}`,
		extension,
	};
}

interface WorkspaceSourceProps {
	directory: WorkspaceDirectoryRef;
	workspace: WorkspaceRef;
}

export function WorkspaceSource({ directory, workspace }: WorkspaceSourceProps) {
	return (
		<ManagementArtifactSource
			label="workspace source"
			load={() => loadWorkspaceSource(directory, workspace)}
			note="This is the shareable Workspace source returned by the Workspace resource API."
		/>
	);
}
