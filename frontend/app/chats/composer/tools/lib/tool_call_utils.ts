import type { UIToolCall } from '@/spec/inference';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import { isSkillsToolName } from '@/apis/skill_management';

export function isRunnableComposerToolCall(toolCall: UIToolCall): boolean {
	if (toolCall.mcpToolSelection) {
		return true;
	}
	const callableType = toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom;

	return (
		callableType &&
		(isSkillsToolName(toolCall.name) || toolCall.toolStoreChoice?.implementationKind === ToolImplType.Go)
	);
}

/**
 * Includes local calls that cannot execute because their historical Tool
 * selection is no longer resolvable. They still need a submitted error result.
 */
export function requiresComposerToolResponse(toolCall: UIToolCall): boolean {
	const clientType = toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom;

	return (
		clientType &&
		(Boolean(toolCall.mcpToolSelection) || toolCall.toolStoreChoice?.implementationKind !== ToolImplType.SDK)
	);
}
