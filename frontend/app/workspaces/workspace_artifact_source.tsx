import type { WorkspaceArtifactView, WorkspaceDirectoryRef } from '@/spec/workspace';
import { artifactRefKey } from '@/spec/artifact';

import { workspaceManagementAPI } from '@/apis/baseapi';

import { ManagementJSONSource } from '@/components/managementui/management_json_source';

function safeFileStem(value: string, fallback: string): string {
	return (
		value
			.trim()
			.toLowerCase()
			.replaceAll(/[^a-z0-9._-]+/g, '-')
			.replaceAll(/^-+|-+$/g, '') || fallback
	);
}

/**
 * Workspace APIs expose management projections rather than raw repository
 * file bytes. This component exports the current authoritative projection.
 */
export function WorkspaceDirectoryRecordSource({ directory }: { directory: WorkspaceDirectoryRef }) {
	return (
		<ManagementJSONSource
			label="Workspace directory record JSON"
			fileName={`${safeFileStem(directory.rootID, 'workspace-directory')}.json`}
			load={() => workspaceManagementAPI.getWorkspaceDirectory(directory)}
			note="Exports the current Workspace directory record, including effective Workspace declarations and diagnostics."
		/>
	);
}

export function WorkspaceArtifactRecordSource({
	directory,
	artifact,
}: {
	directory: WorkspaceDirectoryRef;
	artifact: WorkspaceArtifactView;
}) {
	const displayName = artifact.displayName || artifact.logicalName || artifact.artifact.artifactID;

	return (
		<ManagementJSONSource
			label="Workspace artifact record JSON"
			fileName={`${safeFileStem(displayName, 'workspace-artifact')}.json`}
			load={async () => {
				const artifacts = await workspaceManagementAPI.listWorkspaceDirectoryArtifacts(directory);
				const current = artifacts.find(
					candidate => artifactRefKey(candidate.artifact) === artifactRefKey(artifact.artifact)
				);

				if (!current) {
					throw new Error('This Workspace artifact is no longer available.');
				}

				return current;
			}}
			note="Exports the current Workspace artifact record. Repository source-file bytes are not included in the Workspace API projection."
		/>
	);
}
