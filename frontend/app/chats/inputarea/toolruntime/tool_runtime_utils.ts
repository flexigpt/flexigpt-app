import type { UIToolCall } from '@/spec/inference';
import { ToolStoreChoiceType } from '@/spec/tool';

export function isRunnableComposerToolCall(toolCall: UIToolCall): boolean {
	return toolCall.type === ToolStoreChoiceType.Function || toolCall.type === ToolStoreChoiceType.Custom;
}

function isAutoExecutableComposerToolCall(toolCall: UIToolCall): boolean {
	return (
		toolCall.status === 'pending' &&
		isRunnableComposerToolCall(toolCall) &&
		Boolean(toolCall.toolStoreChoice?.autoExecute)
	);
}

export function getPendingRunnableToolCalls(toolCalls: UIToolCall[]): UIToolCall[] {
	return toolCalls.filter(toolCall => toolCall.status === 'pending' && isRunnableComposerToolCall(toolCall));
}

export function getNextPendingAutoExecutableToolCall(toolCalls: UIToolCall[]): UIToolCall | null {
	const tc = toolCalls.filter(isAutoExecutableComposerToolCall);
	return tc[0] ?? null;
}

export function isSkillsToolName(name: string | undefined): boolean {
	const n = (name ?? '').trim();
	return n.startsWith('skills.');
}
