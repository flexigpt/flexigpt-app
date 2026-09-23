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

import type { AgentRuntimeSnapshot } from '@/chats/composer/agents/agent_runtime';
import type { EditorAreaHandle } from '@/chats/composer/editor/editor_area';
import type {
	AssistantTurnFinishedPayload,
	EditorExternalMessage,
	EditorSubmitPayload,
} from '@/chats/composer/editor/editor_types';
import type { ChatWorkflowStarter } from '@/chats/conversation/starter_intent';
import { areAgentRuntimeSnapshotsEqual, EMPTY_AGENT_RUNTIME_SNAPSHOT } from '@/chats/composer/agents/agent_runtime';
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
		const [_runtimeSnapshot, setRuntimeSnapshot] = useState<AgentRuntimeSnapshot>(EMPTY_AGENT_RUNTIME_SNAPSHOT);

		const editorAreaRef = useRef<EditorAreaHandle>(null);
		const composerContext = useComposerContextState();
		const systemPrompt = useAgentSystemPrompt({
			modelDefaultPrompt: composerContext.selectedModel.systemPrompt,
			modelOptionsLoaded: composerContext.modelOptionsLoaded,
		});

		const replaceRuntimeSnapshot = useCallback((next: AgentRuntimeSnapshot) => {
			setRuntimeSnapshot(previous => (areAgentRuntimeSnapshotsEqual(previous, next) ? previous : next));
		}, []);

		const updateRuntimeSnapshot = useCallback((updater: (previous: AgentRuntimeSnapshot) => AgentRuntimeSnapshot) => {
			setRuntimeSnapshot(previous => {
				const next = updater(previous);
				return areAgentRuntimeSnapshotsEqual(previous, next) ? previous : next;
			});
		}, []);

		const applyAgentRuntime = useCallback(
			(recipe: PreparedAgentStarter) => {
				if (recipe.startingText.trim()) {
					editorAreaRef.current?.setDraftTextIfEmpty(recipe.startingText);
				}

				const toolChoices = recipe.toolSelections.map(selection => selection.choice);
				const conversationTools = toolChoices.filter(choice => choice.toolType !== ToolStoreChoiceType.WebSearch);
				const webSearchTools = toolChoices.filter(choice => choice.toolType === ToolStoreChoiceType.WebSearch);

				if (toolChoices.length > 0) {
					editorAreaRef.current?.setConversationToolsFromChoices(conversationTools);
					editorAreaRef.current?.setWebSearchFromChoices(webSearchTools);
				}

				if (recipe.enabledSkillRefs.length > 0 || recipe.activeSkillRefs.length > 0) {
					editorAreaRef.current?.setInstalledSkillStateFromAgent(recipe.enabledSkillRefs, recipe.activeSkillRefs);
				}

				if (recipe.mcpContext) {
					editorAreaRef.current?.setMCPContextFromMessage(recipe.mcpContext);
				}

				updateRuntimeSnapshot(previous => ({
					conversationToolChoices: toolChoices.length > 0 ? conversationTools : previous.conversationToolChoices,
					webSearchChoices: toolChoices.length > 0 ? webSearchTools : previous.webSearchChoices,
					enabledSkillRefs:
						recipe.enabledSkillRefs.length > 0 ? [...recipe.enabledSkillRefs] : previous.enabledSkillRefs,
					activeSkillRefs: recipe.activeSkillRefs.length > 0 ? [...recipe.activeSkillRefs] : previous.activeSkillRefs,
					mcpContext: recipe.mcpContext ?? previous.mcpContext,
				}));
			},
			[updateRuntimeSnapshot]
		);

		const agent = useAgentManager(composerContext, { applyAgentRuntime }, systemPrompt);

		const resetComposerState = useCallback(() => {
			setAbortConfirmationRequested(false);
			agent.clearAgentTracking();

			editorAreaRef.current?.resetEditor();
			editorAreaRef.current?.setConversationToolsFromChoices([]);
			editorAreaRef.current?.setWebSearchFromChoices([]);
			editorAreaRef.current?.clearMCPContext();
			editorAreaRef.current?.clearWorkspace();
			editorAreaRef.current?.setSkillStateFromMessage([], [], {
				syncSession: SkillSessionSyncMode.None,
				forceResetSession: true,
			});

			replaceRuntimeSnapshot(EMPTY_AGENT_RUNTIME_SNAPSHOT);

			const model = composerContext.resetForNewConversation();
			systemPrompt.resetForNewConversation(model.systemPrompt);
		}, [agent, composerContext, replaceRuntimeSnapshot, systemPrompt]);

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
				focus: () => {
					editorAreaRef.current?.focus();
				},
				resetEditor: () => {
					editorAreaRef.current?.resetEditor();
				},
				resetForNewConversation,
				openTemplateMenu: () => {
					editorAreaRef.current?.openTemplateMenu();
				},
				openToolMenu: () => {
					editorAreaRef.current?.openToolMenu();
				},
				openAttachmentMenu: () => {
					editorAreaRef.current?.openAttachmentMenu();
				},
				openSystemPromptMenu: () => {
					editorAreaRef.current?.openSystemPromptMenu();
				},
				openSkillsMenu: () => {
					editorAreaRef.current?.openSkillsMenu();
				},
				openMCPMenu: () => {
					editorAreaRef.current?.openMCPMenu();
				},
				openWorkspaceMenu: () => {
					editorAreaRef.current?.openWorkspaceMenu();
				},
				requestStopResponse: () => {
					editorAreaRef.current?.requestStopResponse();
				},
				loadWorkflowStarter: async starter => {
					resetComposerState();
					editorAreaRef.current?.setDraftText(starter.draft ?? '');

					if (starter.agent) {
						return agent.selectAgent(starter.agent);
					}

					return agent.ensureDefaultAgent();
				},
				loadExternalMessage: message => {
					editorAreaRef.current?.loadExternalMessage(message);
				},
				loadToolCalls: toolCalls => {
					editorAreaRef.current?.loadToolCalls(toolCalls);
				},
				finishAssistantTurn: payload => {
					editorAreaRef.current?.finishAssistantTurn(payload);
				},
				setConversationToolsFromChoices: tools => {
					editorAreaRef.current?.setConversationToolsFromChoices(tools);
					updateRuntimeSnapshot(previous => ({
						...previous,
						conversationToolChoices: [...tools],
					}));
				},
				setWebSearchFromChoices: tools => {
					editorAreaRef.current?.setWebSearchFromChoices(tools);
					updateRuntimeSnapshot(previous => ({
						...previous,
						webSearchChoices: [...tools],
					}));
				},
				appendMCPAppContextUpdate: update => {
					editorAreaRef.current?.appendMCPAppContextUpdate(update);
				},
				applyAttachmentsDrop: payload => {
					editorAreaRef.current?.applyAttachmentsDrop(payload);
				},
				setSkillStateFromMessage: (enabledRefs, activeRefs, options) => {
					editorAreaRef.current?.setSkillStateFromMessage(enabledRefs, activeRefs, options);
					updateRuntimeSnapshot(previous => ({
						...previous,
						enabledSkillRefs: [...enabledRefs],
						activeSkillRefs: [...activeRefs],
					}));
				},
				restoreConversationContext: context => {
					setAbortConfirmationRequested(false);
					agent.clearAgentTracking();

					const model = composerContext.restoreConversationContext(context);
					systemPrompt.restoreConversationContext(model?.systemPrompt ?? '', context.modelParam);

					editorAreaRef.current?.setConversationToolsFromChoices(context.toolChoices);
					editorAreaRef.current?.setWebSearchFromChoices(context.webSearchChoices);
					editorAreaRef.current?.setMCPContextFromMessage(context.mcpContext);
					editorAreaRef.current?.setMCPAppContextUpdatesFromMessage(context.mcpAppContextUpdates);
					editorAreaRef.current?.setSkillStateFromMessage(context.enabledSkillRefs, context.activeSkillRefs, {
						syncSession: SkillSessionSyncMode.None,
						forceResetSession: true,
					});
					editorAreaRef.current?.setWorkspaceSelectionFromMessage(context.workspaceSelection, false);

					replaceRuntimeSnapshot({
						conversationToolChoices: [...context.toolChoices],
						webSearchChoices: [...context.webSearchChoices],
						enabledSkillRefs: [...context.enabledSkillRefs],
						activeSkillRefs: [...context.activeSkillRefs],
						mcpContext: context.mcpContext,
					});
				},
			}),
			[
				agent,
				composerContext,
				replaceRuntimeSnapshot,
				resetForNewConversation,
				resetComposerState,
				systemPrompt,
				updateRuntimeSnapshot,
			]
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
					onAgentRuntimeStateChange={replaceRuntimeSnapshot}
					editingMessageId={editingMessageId}
					cancelEditing={onCancelEditing}
					systemPrompt={systemPrompt}
				/>
			</div>
		);
	})
);
