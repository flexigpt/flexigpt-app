import type { AgentExportResult } from '@/spec/agent';
import type { ArtifactRef } from '@/spec/artifact';

import { agentManagementAPI } from '@/apis/baseapi';

import type { ManagementArtifactSourceValue } from '@/components/managementui/management_artifact_source';

function agentSourceExtension(result: AgentExportResult): string {
	const matched = /\.([A-Za-z0-9]+)$/.exec(result.suggestedFileName);

	if (matched?.[1]?.toLowerCase() === 'yml') {
		return 'yml';
	}

	return 'yaml';
}

/**
 * Canonical Agent export used by Management and Composer detail surfaces.
 * The backend owns YAML normalization and filename selection.
 */
export async function loadAgentArtifactSource(
	agent: ArtifactRef
): Promise<ManagementArtifactSourceValue<AgentExportResult>> {
	const result = await agentManagementAPI.exportAgent(agent);

	return {
		data: result,
		text: result.content,
		fileName: result.suggestedFileName,
		extension: agentSourceExtension(result),
	};
}
