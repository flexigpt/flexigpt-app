import { type Dispatch, type SetStateAction, useCallback } from 'react';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';

import type { AttachedToolEntry } from '@/chats/composer/platedoc/tool_document_ops';
import { useComposerToolConfig } from '@/chats/composer/toolruntime/use_composer_tool_config';
import { type AutoExecState, useToolAutoExecDrainer } from '@/chats/composer/toolruntime/use_tool_auto_exec_drainer';
import type { ComposerToolRuntimeState } from '@/chats/composer/toolruntime/use_tool_runtime';
import { useComposerToolRuntime } from '@/chats/composer/toolruntime/use_tool_runtime';
import { type WebSearchChoiceTemplate } from '@/chats/composer/tools/websearch_utils';
import type { ConversationToolStateEntry } from '@/tools/lib/conversation_tool_utils';

interface UseComposerToolsArgs {
	isBusy: boolean;
	isSubmitting: boolean;
	ensureSkillSession: () => Promise<string | null>;
	listActiveSkillRefs: (sid: string) => Promise<SkillRef[]>;
	setActiveSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	getCurrentSkillSessionID: () => string | null;
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
	ensureSkillSession,
	listActiveSkillRefs,
	setActiveSkillRefs,
	getCurrentSkillSessionID,
	getAttachedToolEntries,
	externalExecutionBlocked = false,
}: UseComposerToolsArgs): UseComposerToolsResult {
	const runtime = useComposerToolRuntime({
		ensureSkillSession,
		listActiveSkillRefs,
		setActiveSkillRefs,
		getCurrentSkillSessionID,
	});

	const config = useComposerToolConfig({
		getAttachedToolEntries,
	});

	const autoExecBlocked = isBusy || isSubmitting || externalExecutionBlocked;

	const { state: autoExecState } = useToolAutoExecDrainer({
		toolCalls: runtime.toolCalls,
		isBlocked: autoExecBlocked,
		runToolCall: async id => {
			await runtime.runToolCall(id);
		},
	});

	const handleRunSingleToolCall = useCallback(
		async (id: string) => {
			await runtime.runToolCall(id);
		},
		[runtime]
	);

	const handleDiscardToolCall = useCallback(
		(id: string) => {
			runtime.discardToolCall(id);
		},
		[runtime]
	);

	const handleRemoveToolOutput = useCallback(
		(id: string) => {
			runtime.removeToolOutput(id);
		},
		[runtime]
	);

	const handleRetryErroredOutput = useCallback(
		(output: UIToolOutput) => {
			runtime.retryErroredOutput(output);
		},
		[runtime]
	);

	const handleAttachedToolsChanged = useCallback(() => {
		config.recomputeAttachedToolArgsBlocked();
	}, [config]);

	const clearComposerToolsState = useCallback(() => {
		runtime.clearToolRuntime();
		config.clearAttachedToolValidation();
	}, [config, runtime]);

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
