import type { ProviderSDKType } from '@/spec/inference';
import type { ToolListItem, ToolStoreChoice } from '@/spec/tool';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { toolChoiceFromListItem } from '@/apis/tool_management';

import { toolIdentityKey } from '@/tools/lib/tool_identity_utils';
import { computeToolUserArgsStatus } from '@/tools/lib/tool_userargs_utils';

export type WebSearchChoiceTemplate = Omit<ToolStoreChoice, 'choiceID'> & {
	toolType: ToolStoreChoiceType.WebSearch;
};

export function normalizeWebSearchChoiceTemplates(
	templates: WebSearchChoiceTemplate[] | null | undefined
): WebSearchChoiceTemplate[] {
	return templates?.length ? [templates[0]] : [];
}

export function webSearchTemplateFromChoice(choice: ToolStoreChoice): WebSearchChoiceTemplate {
	if (choice.toolType !== ToolStoreChoiceType.WebSearch || choice.implementationKind !== ToolImplType.SDK) {
		throw new Error('The selected tool is not a provider web-search tool.');
	}

	const { choiceID: _choiceID, ...template } = choice;
	return {
		...template,
		toolType: ToolStoreChoiceType.WebSearch,
		autoExecute: false,
	};
}

export function webSearchTemplateFromToolListItem(item: ToolListItem): WebSearchChoiceTemplate {
	return webSearchTemplateFromChoice(toolChoiceFromListItem(item, false));
}

function getEligibleWebSearchTools(tools: ToolListItem[], sdkType: ProviderSDKType): ToolListItem[] {
	return tools.filter(item => {
		const implementation = item.toolDefinition.implementation;
		return (
			implementation.kind === ToolImplType.SDK &&
			implementation.sdkToolType === ToolStoreChoiceType.WebSearch &&
			implementation.sdkType === sdkType.toString()
		);
	});
}

export function buildWebSearchChoicesForSubmit(templates: WebSearchChoiceTemplate[]): ToolStoreChoice[] {
	return normalizeWebSearchChoiceTemplates(templates).map(template => ({
		...template,
		choiceID: getUUIDv7(),
	}));
}

export function webSearchIdentityKey(value: Pick<ToolStoreChoice, 'target'>): string {
	return toolIdentityKey(value.target);
}

export function getWebSearchConfiguration(
	templates: WebSearchChoiceTemplate[],
	tools: ToolListItem[],
	catalogReady: boolean,
	sdkType: ProviderSDKType
) {
	const eligible = catalogReady ? getEligibleWebSearchTools(tools, sdkType) : [];
	const active = templates[0];
	const item = active
		? eligible.find(candidate => webSearchIdentityKey(candidate) === webSearchIdentityKey(active))
		: undefined;
	const status = item
		? computeToolUserArgsStatus(item.toolDefinition.userArgSchema, active?.userArgSchemaInstance)
		: undefined;

	return {
		eligible,
		active,
		item,
		status,
		blocked: Boolean(active && (!catalogReady || !item || (status?.hasSchema && !status.isSatisfied))),
	};
}
