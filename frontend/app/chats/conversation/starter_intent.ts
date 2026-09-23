import type { ArtifactRef } from '@/spec/artifact';

export interface ChatWorkflowStarter {
	workflowID?: string;
	draft: string;
	agent?: ArtifactRef;
}

const WORKFLOW_STARTER_QUERY_KEYS = ['workflow', 'draft', 'agentRootID', 'agentArtifactID'] as const;

function readTrimmedSearchParam(searchParams: URLSearchParams, key: string): string {
	return searchParams.get(key)?.trim() ?? '';
}

export function parseChatWorkflowStarterSearchParams(searchParams: URLSearchParams): ChatWorkflowStarter | null {
	const workflowID = readTrimmedSearchParam(searchParams, 'workflow');
	const draft = searchParams.get('draft') ?? '';

	const agentRootID = readTrimmedSearchParam(searchParams, 'agentRootID');
	const agentArtifactID = readTrimmedSearchParam(searchParams, 'agentArtifactID');

	const hasStarterSignal = workflowID || draft.trim() || agentRootID || agentArtifactID;

	if (!hasStarterSignal) {
		return null;
	}

	const hasCompleteAgentRef = agentRootID && agentArtifactID;

	return {
		workflowID: workflowID || undefined,
		draft,
		agent: hasCompleteAgentRef
			? {
					rootID: agentRootID,
					artifactID: agentArtifactID,
				}
			: undefined,
	};
}

export function removeChatWorkflowStarterSearchParams(source: URLSearchParams): URLSearchParams {
	const next = new URLSearchParams(source);
	for (const key of WORKFLOW_STARTER_QUERY_KEYS) {
		next.delete(key);
	}
	return next;
}
