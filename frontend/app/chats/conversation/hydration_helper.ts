import type { ArtifactRef } from '@/spec/artifact';
import type { ModelParam } from '@/spec/inference';
import type { Conversation, ConversationMessage, RestorableConversationContext } from '@/spec/session';
import type { MCPAppModelContextUpdate, MCPConversationContext } from '@/spec/session_mcp';
import type { SkillRef } from '@/spec/skill';
import type { ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import type { WorkspaceConversationSelection } from '@/spec/workspace';
import { RoleEnum } from '@/spec/inference';
import { ToolStoreChoiceType } from '@/spec/tool';

import { generateTitle } from '@/lib/title_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { clampActiveSkillRefsToEnabled, normalizeSkillRefs } from '@/skills/lib/skill_identity_utils';

export function initConversation(title = 'New Conversation'): Conversation {
	return {
		schemaVersion: 'v1',
		id: getUUIDv7(),
		title: generateTitle(title).title,
		createdAt: new Date().toISOString(),
		modifiedAt: new Date().toISOString(),
		messages: [],
	};
}

function deriveMCPContextFromMessages(messages: ConversationMessage[]): MCPConversationContext | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) {
			continue;
		}
		return message.mcpContext;
	}
	return undefined;
}

function deriveMCPAppContextUpdatesFromMessages(
	messages: ConversationMessage[]
): MCPAppModelContextUpdate[] | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) {
			continue;
		}
		return message.mcpAppContextUpdates;
	}
	return undefined;
}

function deriveConversationToolsFromMessages(messages: ConversationMessage[]): ToolStoreChoice[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) {
			continue;
		}
		return (message.uiToolChoices ?? []).filter(choice => choice.toolType !== ToolStoreChoiceType.WebSearch);
	}
	return [];
}

function deriveWebSearchChoiceFromMessages(messages: ConversationMessage[]): ToolStoreChoice[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) {
			continue;
		}
		return (message.uiToolChoices ?? []).filter(choice => choice.toolType === ToolStoreChoiceType.WebSearch);
	}
	return [];
}

function deriveToolSelectionIssues(messages: ConversationMessage[]): ToolSelectionIssue[] {
	for (const message of messages.toReversed()) {
		if (message.role === RoleEnum.User) {
			return message.uiToolSelectionIssues ?? [];
		}
	}
	return [];
}

function deriveEnabledSkillRefsFromMessages(messages: ConversationMessage[]): SkillRef[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) {
			continue;
		}
		return message.enabledSkillRefs ?? [];
	}
	return [];
}

function deriveActiveSkillRefsFromMessages(messages: ConversationMessage[]): SkillRef[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) {
			continue;
		}
		return message.activeSkillRefs ?? [];
	}
	return [];
}

function deriveWorkspaceSelectionFromMessages(
	messages: ConversationMessage[]
): WorkspaceConversationSelection | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) {
			continue;
		}
		return message.workspaceSelection;
	}
	return undefined;
}

function findLastModelRef(messages: ConversationMessage[]): ArtifactRef | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const ref = messages[i].modelRef;
		if (!ref) {
			continue;
		}
		if (!ref.rootID || !ref.artifactID) {
			continue;
		}
		return ref;
	}
	return undefined;
}

function findLastModelParam(messages: ConversationMessage[]): ModelParam | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const modelParam = messages[i].modelParam;
		if (modelParam) {
			return modelParam;
		}
	}
	return undefined;
}

export function deriveRestorableConversationContextFromMessages(
	messages: ConversationMessage[]
): RestorableConversationContext {
	const enabledSkillRefs = normalizeSkillRefs(deriveEnabledSkillRefsFromMessages(messages));
	const activeSkillRefs = clampActiveSkillRefsToEnabled(enabledSkillRefs, deriveActiveSkillRefsFromMessages(messages));
	return {
		modelRef: findLastModelRef(messages),
		modelParam: findLastModelParam(messages),
		toolChoices: deriveConversationToolsFromMessages(messages),
		webSearchChoices: deriveWebSearchChoiceFromMessages(messages),
		toolSelectionIssues: deriveToolSelectionIssues(messages),
		mcpContext: deriveMCPContextFromMessages(messages),
		mcpAppContextUpdates: deriveMCPAppContextUpdatesFromMessages(messages),
		enabledSkillRefs,
		activeSkillRefs,
		workspaceSelection: deriveWorkspaceSelectionFromMessages(messages),
	};
}
