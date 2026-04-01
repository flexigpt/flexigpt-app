import type { UIToolCall } from '@/spec/inference';
import { ToolStoreChoiceType } from '@/spec/tool';

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
		(toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom) &&
		Boolean(toolCall.toolStoreChoice?.autoExecute)
	);
}

export function isRunnableComposerToolCall(toolCall: UIToolCall): boolean {
	return toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom;
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
