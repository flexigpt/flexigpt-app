import type { RefObject } from 'react';
import { forwardRef, memo, useCallback, useImperativeHandle, useRef, useState } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { RestorableConversationContext } from '@/spec/conversation';
import type { UIToolCall } from '@/spec/inference';
import type { MCPAppModelContextUpdate } from '@/spec/mcp';
import type { UIChatOption } from '@/spec/modelpreset';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';
import { SkillSessionSyncMode } from '@/spec/skill';
import { ToolStoreChoiceType } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import type { PreparedAgentStarter } from '@/apis/agent_management';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import type { EditorAreaHandle } from '@/chats/composer/editor/editor_area';
import type {
	AssistantTurnFinishedPayload,
	EditorExternalMessage,
	EditorSubmitPayload,
} from '@/chats/composer/editor/editor_types';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';
import { useAgentManager } from '@/chats/composer/agents/use_agent_manager';
import { ContextBar } from '@/chats/composer/contextarea/context_bar';
import { useComposerContextState } from '@/chats/composer/contextarea/use_context_state';
import { EditorArea } from '@/chats/composer/editor/editor_area';
import { useAgentSystemPrompt } from '@/chats/composer/skills/use_agent_system_prompt';

export interface ComposerBoxHandle {
	getUIChatOptions(): UIChatOption;
	focus(): void;
	resetEditor(): void;
	resetForNewConversation(): Promise<void>;
	openTemplateMenu(): void;
	openToolMenu(): void;
	openAttachmentMenu(): void;
	openSystemPromptMenu(): void;
	openSkillsMenu(): void;
	openMCPMenu(): void;
	openWorkspaceMenu(): void;
	requestStopResponse(): void;
	loadWorkflowStarter(starter: ChatWorkflowStarter): Promise<boolean>;
	loadExternalMessage(message: EditorExternalMessage): void;
	loadToolCalls(toolCalls: UIToolCall[]): void;
	finishAssistantTurn(payload: AssistantTurnFinishedPayload): void;
	setConversationToolsFromChoices(tools: ToolStoreChoice[]): void;
	setWebSearchFromChoices(tools: ToolStoreChoice[]): void;
	appendMCPAppContextUpdate(update: MCPAppModelContextUpdate): void;
	applyAttachmentsDrop(payload: AttachmentsDroppedPayload): void;
	setSkillStateFromMessage(
		enabledRefs: SkillRef[],
		activeRefs: SkillRef[],
		options?: {
			syncSession?: SkillSessionSyncMode;
			forceResetSession?: boolean;
		}
	): void;
	restoreConversationContext(context: RestorableConversationContext): void;
}

interface ComposerBoxProps {
	onSend: (message: EditorSubmitPayload, options: UIChatOption) => Promise<void>;
	isBusy: boolean;
	isHydrating: boolean;
	abortRef: RefObject<AbortController | null>;
	shortcutConfig: ShortcutConfig;
	editingMessageId: string | null;
	onCancelEditing: () => void;
}

export const ComposerBox = memo(
	forwardRef<ComposerBoxHandle, ComposerBoxProps>(function ComposerBox(
		{ onSend, isBusy, isHydrating, abortRef, shortcutConfig, editingMessageId, onCancelEditing },
		ref
	) {
		const [abortConfirmationRequested, setAbortConfirmationRequested] = useState(false);
		const editorAreaRef = useRef<EditorAreaHandle>(null);
		const composerContext = useComposerContextState();
		const systemPrompt = useAgentSystemPrompt({
			modelDefaultPrompt: composerContext.selectedModel.systemPrompt,
			modelOptionsLoaded: composerContext.modelOptionsLoaded,
		});

		const applyAgentRuntime = useCallback((recipe: PreparedAgentStarter) => {
			const editor = editorAreaRef.current;
			if (!editor) {
				return;
			}
			if (recipe.startingText.trim()) {
				editor.setDraftTextIfEmpty(recipe.startingText);
			}

			const choices = recipe.toolSelections.map(selection => selection.choice);
			if (choices.length > 0) {
				editor.setConversationToolsFromChoices(
					choices.filter(choice => choice.toolType !== ToolStoreChoiceType.WebSearch)
				);
				editor.setWebSearchFromChoices(choices.filter(choice => choice.toolType === ToolStoreChoiceType.WebSearch));
				editor.setToolSelectionIssues([]);
			}

			if (recipe.enabledSkillRefs.length > 0 || recipe.activeSkillRefs.length > 0) {
				editor.setInstalledSkillStateFromAgent(recipe.enabledSkillRefs, recipe.activeSkillRefs);
			}
			if (recipe.mcpContext) {
				editor.setMCPContextFromMessage(recipe.mcpContext);
			}
		}, []);

		const agent = useAgentManager(composerContext, { applyAgentRuntime }, systemPrompt);

		const resetComposerState = useCallback(() => {
			setAbortConfirmationRequested(false);
			agent.clearAgentTracking();

			const editor = editorAreaRef.current;
			editor?.resetEditor();
			editor?.setConversationToolsFromChoices([]);
			editor?.setWebSearchFromChoices([]);
			editor?.setToolSelectionIssues([]);
			editor?.clearMCPContext();
			editor?.clearWorkspace();
			editor?.setSkillStateFromMessage([], [], {
				syncSession: SkillSessionSyncMode.None,
				forceResetSession: true,
			});

			const model = composerContext.resetForNewConversation();
			systemPrompt.resetForNewConversation(model.systemPrompt);
		}, [agent, composerContext, systemPrompt]);

		const resetForNewConversation = useCallback(async () => {
			resetComposerState();
			await agent.ensureDefaultAgent();
		}, [agent, resetComposerState]);

		const handleSubmit = useCallback(
			(payload: EditorSubmitPayload) => {
				setAbortConfirmationRequested(false);
				return onSend(payload, composerContext.chatOptions);
			},
			[composerContext.chatOptions, onSend]
		);

		useImperativeHandle(
			ref,
			() => ({
				getUIChatOptions: () => composerContext.chatOptions,
				focus: () => editorAreaRef.current?.focus(),
				resetEditor: () => editorAreaRef.current?.resetEditor(),
				resetForNewConversation,
				openTemplateMenu: () => editorAreaRef.current?.openTemplateMenu(),
				openToolMenu: () => editorAreaRef.current?.openToolMenu(),
				openAttachmentMenu: () => editorAreaRef.current?.openAttachmentMenu(),
				openSystemPromptMenu: () => editorAreaRef.current?.openSystemPromptMenu(),
				openSkillsMenu: () => editorAreaRef.current?.openSkillsMenu(),
				openMCPMenu: () => editorAreaRef.current?.openMCPMenu(),
				openWorkspaceMenu: () => editorAreaRef.current?.openWorkspaceMenu(),
				requestStopResponse: () => editorAreaRef.current?.requestStopResponse(),
				loadWorkflowStarter: async starter => {
					resetComposerState();
					editorAreaRef.current?.setDraftText(starter.draft ?? '');
					return starter.agent ? agent.selectAgent(starter.agent) : agent.ensureDefaultAgent();
				},
				loadExternalMessage: message => editorAreaRef.current?.loadExternalMessage(message),
				loadToolCalls: calls => editorAreaRef.current?.loadToolCalls(calls),
				finishAssistantTurn: payload => editorAreaRef.current?.finishAssistantTurn(payload),
				setConversationToolsFromChoices: tools => editorAreaRef.current?.setConversationToolsFromChoices(tools),
				setWebSearchFromChoices: tools => editorAreaRef.current?.setWebSearchFromChoices(tools),
				appendMCPAppContextUpdate: update => editorAreaRef.current?.appendMCPAppContextUpdate(update),
				applyAttachmentsDrop: payload => editorAreaRef.current?.applyAttachmentsDrop(payload),
				setSkillStateFromMessage: (enabled, active, options) =>
					editorAreaRef.current?.setSkillStateFromMessage(enabled, active, options),
				restoreConversationContext: context => {
					setAbortConfirmationRequested(false);
					agent.clearAgentTracking();

					const model = composerContext.restoreConversationContext(context);
					systemPrompt.restoreConversationContext(model?.systemPrompt ?? '', context.modelParam);

					const editor = editorAreaRef.current;
					editor?.setConversationToolsFromChoices(context.toolChoices);
					editor?.setWebSearchFromChoices(context.webSearchChoices);
					editor?.setToolSelectionIssues(context.toolSelectionIssues ?? []);
					editor?.setMCPContextFromMessage(context.mcpContext);
					editor?.setMCPAppContextUpdatesFromMessage(context.mcpAppContextUpdates);
					editor?.setSkillStateFromMessage(context.enabledSkillRefs, context.activeSkillRefs, {
						syncSession: SkillSessionSyncMode.None,
						forceResetSession: true,
					});
					editor?.setWorkspaceSelectionFromMessage(context.workspaceSelection, false);
				},
			}),
			[agent, composerContext, resetForNewConversation, resetComposerState, systemPrompt]
		);

		return (
			<div className="bg-base-200 flex w-full min-w-0 flex-col overflow-hidden">
				<div className="shrink-0">
					<ContextBar context={composerContext} agent={agent} />
				</div>

				<DeleteConfirmationModal
					isOpen={isBusy && abortConfirmationRequested}
					onClose={() => {
						setAbortConfirmationRequested(false);
					}}
					onConfirm={() => {
						setAbortConfirmationRequested(false);
						abortRef.current?.abort();
					}}
					title="Abort generation?"
					message="Partial output already received remains in the conversation. Stop the request?"
					confirmButtonText="Abort"
				/>

				<EditorArea
					ref={editorAreaRef}
					isGenerating={isBusy}
					isInputLocked={isBusy || isHydrating}
					currentProviderSDKType={composerContext.chatOptions.providerSDKType}
					shortcutConfig={shortcutConfig}
					onSubmit={handleSubmit}
					onRequestStop={() => {
						if (isBusy) {
							setAbortConfirmationRequested(true);
						}
					}}
					editingMessageId={editingMessageId}
					cancelEditing={onCancelEditing}
					systemPrompt={systemPrompt}
				/>
			</div>
		);
	})
);
