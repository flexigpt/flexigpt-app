import type { MCPConversationContext } from '@/spec/mcp';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';
import { ToolStoreChoiceType } from '@/spec/tool';

import { areComparableValuesEqual } from '@/lib/obj_utils';

import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import { areSkillRefListsEqual } from '@/skills/lib/skill_identity_utils';
import { areToolChoiceListsEqual } from '@/tools/lib/tool_choice_utils';

export interface AgentRuntimeSnapshot {
	conversationToolChoices: ToolStoreChoice[];
	webSearchChoices: ToolStoreChoice[];
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	mcpContext?: MCPConversationContext;
}

export const EMPTY_AGENT_RUNTIME_SNAPSHOT: AgentRuntimeSnapshot = {
	conversationToolChoices: [],
	webSearchChoices: [],
	enabledSkillRefs: [],
	activeSkillRefs: [],
	mcpContext: undefined,
};

export function areAgentRuntimeSnapshotsEqual(left: AgentRuntimeSnapshot, right: AgentRuntimeSnapshot): boolean {
	return (
		areToolChoiceListsEqual(left.conversationToolChoices, right.conversationToolChoices) &&
		areToolChoiceListsEqual(left.webSearchChoices, right.webSearchChoices) &&
		areSkillRefListsEqual(left.enabledSkillRefs, right.enabledSkillRefs) &&
		areSkillRefListsEqual(left.activeSkillRefs, right.activeSkillRefs) &&
		areComparableValuesEqual(left.mcpContext, right.mcpContext)
	);
}

export function mapWebSearchTemplatesToChoices(templates: WebSearchChoiceTemplate[]): ToolStoreChoice[] {
	return templates.map((template, index) => ({
		choiceID: `agent-web-search:${index}:${template.bundleID}:${template.toolSlug}:${template.toolVersion}`,
		bundleID: template.bundleID,
		bundleSlug: template.bundleSlug,
		toolID: template.toolID,
		toolSlug: template.toolSlug,
		toolVersion: template.toolVersion,
		toolType: template.toolType ?? ToolStoreChoiceType.WebSearch,
		displayName: template.displayName,
		description: template.description,
		autoExecute: template.autoExecute,
		userArgSchemaInstance: template.userArgSchemaInstance,
	}));
}
