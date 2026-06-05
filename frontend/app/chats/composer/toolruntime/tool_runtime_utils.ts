import type { UIToolCall } from '@/spec/inference';
import { MCPExecutionMode } from '@/spec/mcp';
import { SKILLS_AUTOEXEC_TOOL_CHOICES } from '@/spec/skill';

import { isRunnableComposerToolCall } from '@/tools/lib/tool_call_utils';

type AutoSubmitTracker = {
	observedCallKeys: Set<string>;
	allObservedCallsAreAutoExecute: boolean;
	attemptedBatchSignature: string | null;
};

export function createAutoSubmitTracker(): AutoSubmitTracker {
	return {
		observedCallKeys: new Set<string>(),
		allObservedCallsAreAutoExecute: true,
		attemptedBatchSignature: null,
	};
}

export function getToolAutoSubmitKey(value: { id: string; callID: string }): string {
	return value.callID || value.id;
}

export function isAutoSubmitEligibleToolCall(toolCall: UIToolCall): boolean {
	return (
		!toolCall.suppressAutoExecute &&
		isRunnableComposerToolCall(toolCall) &&
		(Boolean(toolCall.toolStoreChoice?.autoExecute) ||
			toolCall.mcpToolSelection?.executionMode === MCPExecutionMode.MCPExecutionModeAuto ||
			SKILLS_AUTOEXEC_TOOL_CHOICES.has(toolCall.choiceID))
	);
}

export function getPendingRunnableToolCalls(toolCalls: UIToolCall[]): UIToolCall[] {
	return toolCalls.filter(toolCall => toolCall.status === 'pending' && isRunnableComposerToolCall(toolCall));
}

export function getNextPendingAutoExecutableToolCall(toolCalls: UIToolCall[]): UIToolCall | null {
	const tc = toolCalls.filter(toolCall => {
		return (
			toolCall.status === 'pending' && isRunnableComposerToolCall(toolCall) && isAutoSubmitEligibleToolCall(toolCall)
		);
	});
	return tc[0] ?? null;
}
