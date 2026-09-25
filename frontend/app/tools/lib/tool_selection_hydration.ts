import type { MappedTarget } from '@/spec/artifact';
import type { ResolvedToolView, ToolSelection, ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';

import { mapWithConcurrency } from '@/lib/async_utils';
import { getErrorMessage } from '@/lib/error_utils';

import { toolManagementAPI } from '@/apis/baseapi';
import { toolStoreChoiceFromSelection } from '@/apis/tool_management';

import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';

export interface ToolSelectionHydration {
	selection: ToolSelection;
	choice?: ToolStoreChoice;
	issue?: ToolSelectionIssue;
}

interface TargetResolution {
	key: string;
	resolved?: ResolvedToolView;
	error?: string;
}

export async function hydrateToolSelectionsForUI(
	selections: ToolSelection[],
	shouldContinue: () => boolean = () => true
): Promise<ToolSelectionHydration[] | undefined> {
	if (!shouldContinue()) {
		return undefined;
	}

	const targets = new Map<string, MappedTarget>();
	for (const selection of selections) {
		targets.set(toolIdentityKey(selection.target), selection.target);
	}

	const resolutions = await mapWithConcurrency(
		[...targets.entries()],
		4,
		async ([key, target]): Promise<TargetResolution> => {
			try {
				const resolved = await toolManagementAPI.resolveMappedTool(target);
				return { key, resolved };
			} catch (error) {
				return {
					key,
					error: getErrorMessage(error, `Tool "${target.name}" is unavailable.`),
				};
			}
		}
	);

	if (!shouldContinue()) {
		return undefined;
	}

	const byTargetKey = new Map(resolutions.map(value => [value.key, value] as const));

	return selections.map(selection => {
		const resolved = byTargetKey.get(toolIdentityKey(selection.target));

		if (resolved?.resolved) {
			return {
				selection,
				choice: toolStoreChoiceFromSelection(selection, resolved.resolved),
			};
		}

		return {
			selection,
			issue: {
				selection,
				message: resolved?.error ?? `Tool "${selection.target.name}" is unavailable.`,
			},
		};
	});
}
