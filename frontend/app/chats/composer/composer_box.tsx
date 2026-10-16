import type { RefObject } from 'react';
import { forwardRef, memo, useCallback, useImperativeHandle, useRef, useState } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { UIToolCall } from '@/spec/inference';
import type { UIModelOption } from '@/spec/model';
import type { RestorableConversationContext, SessionConfiguration } from '@/spec/session';
import type { MCPAppModelContextUpdate } from '@/spec/session_mcp';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';
import { SkillSessionSyncMode } from '@/spec/skill';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

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
	getUIModelOption(): UIModelOption;
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
	restoreSessionConfiguration(configuration: SessionConfiguration): Promise<void>;
}

interface ComposerBoxProps {
	onSend: (message: EditorSubmitPayload, options: UIModelOption) => Promise<void>;
	isBusy: boolean;
	isGenerating: boolean;
	allowContinuationInput?: boolean;
	isHydrating: boolean;
	abortRef: RefObject<AbortController | null>;
	shortcutConfig: ShortcutConfig;
	editingMessageId: string | null;
	onCancelEditing: () => void;
	onCancelRun?: () => Promise<void>;
}

export const ComposerBox = memo(
	forwardRef<ComposerBoxHandle, ComposerBoxProps>(function ComposerBox(
		{
			onSend,
			isBusy,
			isGenerating,
			allowContinuationInput = false,
			isHydrating,
			abortRef,
			shortcutConfig,
			editingMessageId,
			onCancelEditing,
			onCancelRun,
		},
		ref
	) {
		const [abortConfirmationRequested, setAbortConfirmationRequested] = useState(false);
		const editorAreaRef = useRef<EditorAreaHandle>(null);
		const composerContext = useComposerContextState();
		const systemPrompt = useAgentSystemPrompt({
			modelDefaultPrompt: composerContext.selectedModel.systemPrompt,
			modelOptionsLoaded: composerContext.modelOptionsLoaded,
		});

		const applyAgentUserMessageStarter = useCallback((text: string): boolean => {
			return editorAreaRef.current?.setDraftTextIfEmpty(text) ?? false;
		}, []);

		const agent = useAgentManager(composerContext, {
			applyUserMessageStarter: applyAgentUserMessageStarter,
		});

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
			await agent.resetToDefaultAgent();
		}, [agent, resetComposerState]);

		const handleSubmit = useCallback(
			(payload: EditorSubmitPayload) => {
				setAbortConfirmationRequested(false);
				if (!composerContext.hasRunnableModel) {
					return Promise.resolve();
				}

				return onSend(
					{
						...payload,
						agentRef: agent.selectedAgent?.ref,
						agentSelection: agent.selectedAgent
							? {
									rootID: agent.selectedAgent.ref.rootID,
									artifactID: agent.selectedAgent.ref.artifactID,
									expectedRevision: agent.selectedAgent.agent.revision,
									...(agent.selectedAgent.agent.definitionDigest
										? { expectedDigest: agent.selectedAgent.agent.definitionDigest }
										: {}),
								}
							: undefined,
						modelSelectionSource:
							agent.selectedAgent && composerContext.modelSelectionSource === 'agent' ? 'agent' : 'session',
					},
					composerContext.chatOptions
				);
			},
			[
				agent.selectedAgent,
				composerContext.chatOptions,
				composerContext.hasRunnableModel,
				composerContext.modelSelectionSource,
				onSend,
			]
		);

		useImperativeHandle(
			ref,
			() => ({
				getUIModelOption: () => composerContext.chatOptions,
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

					if (!starter.agent) {
						return agent.resetToDefaultAgent();
					}

					const applied = await agent.selectAgent(starter.agent);
					if (!applied) {
						await agent.resetToDefaultAgent();
					}

					return applied;
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

					// Preserve restored conversation state while retaining the
					// Base/default Agent as the visible tracked selection.
					void agent.trackDefaultAgentWithoutApplying();
				},
				restoreSessionConfiguration: async configuration => {
					setAbortConfirmationRequested(false);
					editorAreaRef.current?.resetEditor();

					composerContext.restoreSessionConfiguration(configuration);
					systemPrompt.restoreSessionConfiguration(configuration);

					await agent.restoreSelectedAgent(configuration.agent, configuration.model ? 'session' : 'agent');
					await editorAreaRef.current?.restoreSessionConfiguration(configuration);
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
						if (onCancelRun) {
							void onCancelRun().catch(console.error);
							return;
						}
						abortRef.current?.abort();
					}}
					title="Abort generation?"
					message="Partial output already received remains in the conversation. Stop the request?"
					confirmButtonText="Abort"
				/>

				<EditorArea
					ref={editorAreaRef}
					isGenerating={isGenerating}
					isRunActive={isBusy}
					submissionMode={allowContinuationInput ? 'continuation' : 'message'}
					isInputLocked={(isBusy && !allowContinuationInput) || isHydrating || !composerContext.hasRunnableModel}
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
