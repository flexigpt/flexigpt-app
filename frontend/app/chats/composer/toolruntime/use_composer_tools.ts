import type { Dispatch, SetStateAction } from 'react';
import { useCallback } from 'react';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { ToolStoreChoice } from '@/spec/tool';

import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import type { AutoExecState } from '@/chats/composer/toolruntime/use_tool_auto_exec_drainer';
import type { ComposerToolRuntimeState } from '@/chats/composer/toolruntime/use_tool_runtime';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';
import { useComposerToolConfig } from '@/chats/composer/toolruntime/use_composer_tool_config';
import { useComposerToolRuntime } from '@/chats/composer/toolruntime/use_tool_runtime';

interface UseComposerToolsArgs {
	isBusy: boolean;
	isSubmitting: boolean;
	getAttachedToolEntries: (uniqueByIdentity?: boolean) => AttachedToolEntry[];
	externalExecutionBlocked?: boolean;
}

interface UseComposerToolsResult {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
	setToolOutputs: Dispatch<SetStateAction<UIToolOutput[]>>;
	conversationToolsState: ConversationToolStateEntry[];
	setConversationToolsState: Dispatch<SetStateAction<ConversationToolStateEntry[]>>;
	webSearchTemplates: WebSearchChoiceTemplate[];
	setWebSearchTemplates: Dispatch<SetStateAction<WebSearchChoiceTemplate[]>>;
	toolArgsBlocked: boolean;
	hasPendingToolCalls: boolean;
	hasRunningToolCalls: boolean;
	recomputeAttachedToolArgsBlocked: () => void;
	runAllPendingToolCalls: () => Promise<UIToolOutput[]>;
	handleRunSingleToolCall: (id: string) => Promise<void>;
	handleDiscardToolCall: (id: string) => void;
	handleRemoveToolOutput: (id: string) => void;
	handleRetryErroredOutput: (output: UIToolOutput) => void;
	handleAttachedToolsChanged: () => void;
	applyConversationToolsFromChoices: (tools: ToolStoreChoice[]) => void;
	applyWebSearchFromChoices: (tools: ToolStoreChoice[]) => void;
	loadToolCalls: (toolCalls: UIToolCall[]) => void;
	clearComposerToolsState: () => void;
	getToolRuntimeSnapshot: () => ComposerToolRuntimeState;
	autoExecState: AutoExecState;
}

export function useComposerTools({
	isBusy,
	isSubmitting,
	getAttachedToolEntries,
	externalExecutionBlocked = false,
}: UseComposerToolsArgs): UseComposerToolsResult {
	const runtime = useComposerToolRuntime();

	const config = useComposerToolConfig({
		getAttachedToolEntries,
	});

	const autoExecBlocked = isBusy || isSubmitting || externalExecutionBlocked;

	const runToolCall = runtime.runToolCall;
	const discardToolCall = runtime.discardToolCall;
	const removeToolOutput = runtime.removeToolOutput;
	const retryErroredOutput = runtime.retryErroredOutput;
	const clearToolRuntime = runtime.clearToolRuntime;
	const recomputeAttachedToolArgsBlocked = config.recomputeAttachedToolArgsBlocked;
	const clearAttachedToolValidation = config.clearAttachedToolValidation;

	const autoExecState = { phase: autoExecBlocked ? ('blocked' as const) : ('idle' as const), runningCallId: null };

	const handleRunSingleToolCall = useCallback(
		async (id: string) => {
			await runToolCall(id);
		},
		[runToolCall]
	);

	const handleDiscardToolCall = useCallback(
		(id: string) => {
			discardToolCall(id);
		},
		[discardToolCall]
	);

	const handleRemoveToolOutput = useCallback(
		(id: string) => {
			removeToolOutput(id);
		},
		[removeToolOutput]
	);

	const handleRetryErroredOutput = useCallback(
		(output: UIToolOutput) => {
			retryErroredOutput(output);
		},
		[retryErroredOutput]
	);

	const handleAttachedToolsChanged = useCallback(() => {
		recomputeAttachedToolArgsBlocked();
	}, [recomputeAttachedToolArgsBlocked]);

	const clearComposerToolsState = useCallback(() => {
		clearToolRuntime();
		clearAttachedToolValidation();
	}, [clearAttachedToolValidation, clearToolRuntime]);

	return {
		toolCalls: runtime.toolCalls,
		toolOutputs: runtime.toolOutputs,
		setToolOutputs: runtime.setToolOutputs,
		conversationToolsState: config.conversationToolsState,
		setConversationToolsState: config.setConversationToolsState,
		webSearchTemplates: config.webSearchTemplates,
		setWebSearchTemplates: config.setWebSearchTemplates,
		toolArgsBlocked: config.toolArgsBlocked,
		hasPendingToolCalls: runtime.hasPendingToolCalls,
		hasRunningToolCalls: runtime.hasRunningToolCalls,
		recomputeAttachedToolArgsBlocked: config.recomputeAttachedToolArgsBlocked,
		runAllPendingToolCalls: runtime.runAllPendingToolCalls,
		handleRunSingleToolCall,
		handleDiscardToolCall,
		handleRemoveToolOutput,
		handleRetryErroredOutput,
		handleAttachedToolsChanged,
		applyConversationToolsFromChoices: config.applyConversationToolsFromChoices,
		applyWebSearchFromChoices: config.applyWebSearchFromChoices,
		loadToolCalls: runtime.loadToolCalls,
		clearComposerToolsState,
		getToolRuntimeSnapshot: runtime.getToolRuntimeSnapshot,
		autoExecState,
	};
}
