import type { AgentCapabilityOccurrence } from '@/spec/agent';
import type { ArtifactRef } from '@/spec/artifact';
import { CapabilityTargetForm } from '@/spec/artifact';

import { mcpManagementAPI, modelManagementAPI, textStoreAPI } from '@/apis/baseapi';

import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';
import { ManagementDetailsModal } from '@/components/managementui/management_details_modal';
import { ManagementJSONSource } from '@/components/managementui/management_json_source';

import { loadAgentArtifactSource } from '@/agents/agent_artifact_source';
import { SkillDetailsModal, skillDetailsSubjectFromReference } from '@/skills/skill_details_modal';
import { ToolDetailsModal } from '@/tools/tool_details_modal';

interface AgentContributionDetailsModalProps {
	isOpen: boolean;
	occurrence: AgentCapabilityOccurrence | null;
	onClose: () => void;
}

function artifactRefForOccurrence(occurrence: AgentCapabilityOccurrence): ArtifactRef | undefined {
	if (occurrence.target?.form !== CapabilityTargetForm.Artifact || !occurrence.target.artifact) {
		return undefined;
	}

	return occurrence.target.artifact;
}

function occurrenceTitle(occurrence: AgentCapabilityOccurrence): string {
	return occurrence.name?.trim() || `Agent ${occurrence.type}`;
}

function SourceDetails({
	occurrence,
	artifact,
	onClose,
}: {
	occurrence: AgentCapabilityOccurrence;
	artifact: ArtifactRef;
	onClose: () => void;
}) {
	const title = occurrenceTitle(occurrence);
	const modalKey = `agent-contribution:${occurrence.type}:${artifact.rootID}:${artifact.artifactID}:${occurrence.path}`;

	if (occurrence.type === 'skill') {
		return (
			<SkillDetailsModal
				isOpen
				skill={skillDetailsSubjectFromReference(artifact, {
					displayName: title,
					sourceLabel: 'Agent',
				})}
				showPreview
				onClose={onClose}
			/>
		);
	}

	if (occurrence.type === 'tool') {
		return (
			<ToolDetailsModal
				isOpen
				subject={{
					ref: artifact,
					displayName: title,
				}}
				onClose={onClose}
			/>
		);
	}

	return (
		<ManagementDetailsModal isOpen onClose={onClose} title={title} modalKey={modalKey}>
			{occurrence.type === 'text' ? (
				<ManagementArtifactSource
					label="text source"
					load={async () => {
						const text = await textStoreAPI.materializeText(artifact);

						return {
							data: text,
							text: text.content,
							fileName: 'text.txt',
							extension: 'txt',
						};
					}}
				/>
			) : null}

			{occurrence.type === 'model' ? (
				<ManagementJSONSource
					label="model JSON"
					fileName="model.json"
					load={async () => (await modelManagementAPI.getModel(artifact)).document}
					note="This is the shareable Model source."
				/>
			) : null}

			{occurrence.type === 'mcp' ? (
				<ManagementArtifactSource
					label="MCP server YAML"
					load={() => mcpManagementAPI.exportMCPServerSource(artifact)}
					note="This portable server declaration excludes installation secrets and OAuth tokens."
				/>
			) : null}

			{occurrence.type === 'agent' ? (
				<ManagementArtifactSource
					label="agent source"
					load={() => loadAgentArtifactSource(artifact)}
					note="The extension follows the Agent source format supplied by the exporter."
				/>
			) : null}

			{!['text', 'model', 'mcp', 'agent'].includes(occurrence.type) ? (
				<p className="text-base-content/60 text-sm">A portable source is not available for this contribution type.</p>
			) : null}
		</ManagementDetailsModal>
	);
}

export function AgentContributionDetailsModal({ isOpen, occurrence, onClose }: AgentContributionDetailsModalProps) {
	if (!isOpen || !occurrence) {
		return null;
	}

	const artifact = artifactRefForOccurrence(occurrence);

	if (!artifact) {
		return null;
	}

	return <SourceDetails occurrence={occurrence} artifact={artifact} onClose={onClose} />;
}
