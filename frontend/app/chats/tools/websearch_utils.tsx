import type { ProviderSDKType } from '@/spec/inference';
import { ToolImplType, type ToolListItem, type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { getUUIDv7 } from '@/lib/uuid_utils';

// Persisted config we keep in UI state (NO choiceID; we generate a fresh one per send).
export type WebSearchChoiceTemplate = Omit<ToolStoreChoice, 'choiceID'> & {
	toolType: ToolStoreChoiceType.WebSearch;
};

export function webSearchTemplateFromChoice(choice: ToolStoreChoice): WebSearchChoiceTemplate {
	const { choiceID: _drop, ...rest } = choice;
	return rest as WebSearchChoiceTemplate;
}

export function webSearchTemplateFromToolListItem(item: ToolListItem): WebSearchChoiceTemplate {
	return {
		bundleID: item.bundleID,
		bundleSlug: item.bundleSlug,
		toolID: item.toolDefinition.id,
		toolSlug: item.toolSlug,
		toolVersion: item.toolVersion,
		toolType: ToolStoreChoiceType.WebSearch,
		displayName: item.toolDefinition.displayName,
		description: item.toolDefinition.description,
		autoExecute: false,
		userArgSchemaInstance: undefined,
	};
}

function buildWebSearchChoiceForSubmit(t: WebSearchChoiceTemplate): ToolStoreChoice {
	return { ...t, choiceID: getUUIDv7() };
}

function isEligibleWebSearchToolForSDKType(item: ToolListItem, sdkType: ProviderSDKType): boolean {
	if (item.toolDefinition.llmToolType !== ToolStoreChoiceType.WebSearch) return false;
	// Provider-managed web-search tools are SDK-bound
	if (item.toolDefinition.type !== ToolImplType.SDK) return false;
	return item.toolDefinition.sdkImpl?.sdkType === sdkType.toString();
}

export function getEligibleWebSearchTools(tools: ToolListItem[], sdkType: ProviderSDKType): ToolListItem[] {
	return tools.filter(t => isEligibleWebSearchToolForSDKType(t, sdkType));
}

export function buildWebSearchChoicesForSubmit(ts: WebSearchChoiceTemplate[]): ToolStoreChoice[] {
	return ts.map(buildWebSearchChoiceForSubmit);
}

export function webSearchIdentityKey(t: { bundleID: string; toolSlug: string; toolVersion: string }): string {
	return `${t.bundleID}/${t.toolSlug}@${t.toolVersion}`;
}
