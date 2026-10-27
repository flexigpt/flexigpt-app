import type { ArtifactRef } from '@/spec/artifact';

import { mcpManagementAPI, modelManagementAPI, textStoreAPI } from '@/apis/baseapi';

import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementJSONSource } from '@/components/managementui/management_json_source';

import { loadAgentArtifactSource } from '@/agents/agent_artifact_source';
import { SkillDetailsModal, skillDetailsSubjectFromReference } from '@/skills/skill_details_modal';
import { ToolDetailsModal } from '@/tools/tool_details_modal';

type WorkspaceContributionSourceKind = 'text' | 'skill' | 'mcp' | 'tool' | 'model' | 'agent' | 'unsupported';

export interface WorkspaceContributionSourceSubject {
	kind: WorkspaceContributionSourceKind;
	artifact: ArtifactRef;
	title: string;
	description?: string;
}

interface WorkspaceContributionDetailsModalProps {
	isOpen: boolean;
	subject: WorkspaceContributionSourceSubject | null;
	onClose: () => void;
}

function WorkspaceSourceDetails({
	subject,
	onClose,
}: {
	subject: WorkspaceContributionSourceSubject;
	onClose: () => void;
}) {
	const modalKey = `workspace-contribution:${subject.kind}:${subject.artifact.rootID}:${subject.artifact.artifactID}`;

	if (subject.kind === 'skill') {
		return (
			<SkillDetailsModal
				isOpen
				skill={skillDetailsSubjectFromReference(subject.artifact, {
					displayName: subject.title,
					description: subject.description,
					sourceLabel: 'Workspace',
				})}
				showPreview
				onClose={onClose}
			/>
		);
	}

	if (subject.kind === 'tool') {
		return (
			<ToolDetailsModal
				isOpen
				subject={{
					ref: subject.artifact,
					displayName: subject.title,
					description: subject.description,
				}}
				onClose={onClose}
			/>
		);
	}

	return (
		<ManagementDetailsModal
			isOpen
			onClose={onClose}
			title={subject.title}
			description={subject.description}
			modalKey={modalKey}
		>
			{subject.kind === 'text' ? (
				<ManagementArtifactSource
					label="repository text"
					load={async () => {
						const text = await textStoreAPI.materializeText(subject.artifact);

						return {
							data: text,
							text: text.content,
							fileName: 'workspace-text.txt',
							extension: 'txt',
						};
					}}
				/>
			) : null}

			{subject.kind === 'mcp' ? (
				<ManagementArtifactSource
					label="MCP server YAML"
					load={() => mcpManagementAPI.exportMCPServerSource(subject.artifact)}
					note="This portable server declaration excludes installation secrets and OAuth tokens."
				/>
			) : null}

			{subject.kind === 'model' ? (
				<ManagementJSONSource
					label="model JSON"
					fileName="model.json"
					load={async () => (await modelManagementAPI.getModel(subject.artifact)).document}
					note="This is the shareable Model source."
				/>
			) : null}

			{subject.kind === 'agent' ? (
				<ManagementArtifactSource
					label="agent source"
					load={() => loadAgentArtifactSource(subject.artifact)}
					note="The extension follows the Agent source format supplied by the exporter."
				/>
			) : null}

			{subject.kind === 'unsupported' ? (
				<p className="text-base-content/60 text-sm">
					A portable source is not exposed for this Workspace item by the current Workspace API.
				</p>
			) : null}
		</ManagementDetailsModal>
	);
}

export function WorkspaceContributionDetailsModal({
	isOpen,
	subject,
	onClose,
}: WorkspaceContributionDetailsModalProps) {
	if (!isOpen || !subject) {
		return null;
	}

	return <WorkspaceSourceDetails subject={subject} onClose={onClose} />;
}
