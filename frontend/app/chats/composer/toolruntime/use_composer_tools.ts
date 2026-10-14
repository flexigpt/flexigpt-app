import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useRef, useState } from 'react';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { ToolStoreChoice } from '@/spec/tool';

import { resolveStateUpdate } from '@/lib/hook_utils';

import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import type { WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';
import { useComposerToolConfig } from '@/chats/composer/toolruntime/use_composer_tool_config';

interface UseComposerToolsArgs {
	isBusy: boolean;
	isSubmitting: boolean;
	getAttachedToolEntries: (uniqueByIdentity?: boolean) => AttachedToolEntry[];
	externalExecutionBlocked?: boolean;
}

interface ComposerToolRuntimeState {
	toolCalls: UIToolCall[];
	toolOutputs: UIToolOutput[];
}

interface AutoExecState {
	phase: 'idle' | 'blocked';
	runningCallId: null;
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

const EMPTY_TOOL_CALLS: UIToolCall[] = [];

export function useComposerTools({ getAttachedToolEntries }: UseComposerToolsArgs): UseComposerToolsResult {
	const config = useComposerToolConfig({
		getAttachedToolEntries,
	});
	const [toolOutputsState, setToolOutputsState] = useState<UIToolOutput[]>([]);
	const toolOutputsRef = useRef<UIToolOutput[]>([]);

	const setToolOutputs = useCallback<Dispatch<SetStateAction<UIToolOutput[]>>>(update => {
		const next = resolveStateUpdate(update, toolOutputsRef.current);
		toolOutputsRef.current = next;
		setToolOutputsState(next);
	}, []);

	/*
	 * Session Runtime owns accepted Tool calls, execution, retries, skips,
	 * approvals, and automatic continuation. Composer retains only historical
	 * draft output display state while editing a message. It intentionally
	 * never receives a runnable local Tool-call queue.
	 */
	const noLocalToolExecution = useCallback(async (_id: string) => {
		return undefined;
	}, []);

	const clearComposerToolsState = useCallback(() => {
		toolOutputsRef.current = [];
		setToolOutputsState([]);
		config.clearAttachedToolValidation();
	}, [config]);

	const handleAttachedToolsChanged = useCallback(() => {
		config.recomputeAttachedToolArgsBlocked();
	}, [config]);

	return {
		toolCalls: EMPTY_TOOL_CALLS,
		toolOutputs: toolOutputsState,
		setToolOutputs,
		conversationToolsState: config.conversationToolsState,
		setConversationToolsState: config.setConversationToolsState,
		webSearchTemplates: config.webSearchTemplates,
		setWebSearchTemplates: config.setWebSearchTemplates,
		toolArgsBlocked: config.toolArgsBlocked,
		hasPendingToolCalls: false,
		hasRunningToolCalls: false,
		recomputeAttachedToolArgsBlocked: config.recomputeAttachedToolArgsBlocked,
		runAllPendingToolCalls: async () => [],
		handleRunSingleToolCall: noLocalToolExecution,
		handleDiscardToolCall: () => undefined,
		handleRemoveToolOutput: id => {
			setToolOutputs(previous => previous.filter(output => output.id !== id));
		},
		handleRetryErroredOutput: () => undefined,
		handleAttachedToolsChanged,
		applyConversationToolsFromChoices: config.applyConversationToolsFromChoices,
		applyWebSearchFromChoices: config.applyWebSearchFromChoices,
		loadToolCalls: () => undefined,
		clearComposerToolsState,
		getToolRuntimeSnapshot: () => ({
			toolCalls: EMPTY_TOOL_CALLS,
			toolOutputs: toolOutputsRef.current,
		}),
		autoExecState: {
			phase: 'idle',
			runningCallId: null,
		},
	};
}
