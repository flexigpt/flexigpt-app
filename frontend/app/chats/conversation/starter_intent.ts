export interface ChatWorkflowStarterAssistantPresetRef {
	bundleID: string;
	assistantPresetSlug: string;
	assistantPresetVersion: string;
}

export interface ChatWorkflowStarter {
	workflowID?: string;
	draft: string;
	assistantPreset?: ChatWorkflowStarterAssistantPresetRef;
}

const WORKFLOW_STARTER_QUERY_KEYS = [
	'workflow',
	'draft',
	'assistantPresetBundleID',
	'assistantPresetSlug',
	'assistantPresetVersion',
] as const;

function readTrimmedSearchParam(searchParams: URLSearchParams, key: string): string {
	return searchParams.get(key)?.trim() ?? '';
}

export function parseChatWorkflowStarterSearchParams(searchParams: URLSearchParams): ChatWorkflowStarter | null {
	const workflowID = readTrimmedSearchParam(searchParams, 'workflow');
	const draft = searchParams.get('draft') ?? '';

	const assistantPresetBundleID = readTrimmedSearchParam(searchParams, 'assistantPresetBundleID');
	const assistantPresetSlug = readTrimmedSearchParam(searchParams, 'assistantPresetSlug');
	const assistantPresetVersion = readTrimmedSearchParam(searchParams, 'assistantPresetVersion');

	const hasStarterSignal =
		workflowID || draft.trim() || assistantPresetBundleID || assistantPresetSlug || assistantPresetVersion;

	if (!hasStarterSignal) {
		return null;
	}

	const hasCompleteAssistantPresetRef = assistantPresetBundleID && assistantPresetSlug && assistantPresetVersion;

	return {
		workflowID: workflowID || undefined,
		draft,
		assistantPreset: hasCompleteAssistantPresetRef
			? {
					bundleID: assistantPresetBundleID,
					assistantPresetSlug,
					assistantPresetVersion,
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
