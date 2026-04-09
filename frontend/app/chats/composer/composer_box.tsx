import { forwardRef, type RefObject, useCallback, useEffect, useImperativeHandle, useRef, useState } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { RestorableConversationContext } from '@/spec/conversation';
import type { UIToolCall } from '@/spec/inference';
import { type UIChatOption } from '@/spec/modelpreset';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import {
	type AssistantPresetPreparedApplication,
	EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT,
} from '@/chats/composer/assistantpresets/assistant_preset_runtime';
import { useAssistantPresetManager } from '@/chats/composer/assistantpresets/use_assistant_preset_manager';
import { EditorContextBar } from '@/chats/composer/contextarea/context_bar';
import { useAssistantContextState } from '@/chats/composer/contextarea/use_context_state';
import { EditorArea, type EditorAreaHandle } from '@/chats/composer/editor/editor_area';
import type { EditorExternalMessage, EditorSubmitPayload } from '@/chats/composer/editor/editor_types';
import { useComposerSystemPrompt } from '@/chats/composer/systemprompts/use_composer_system_prompt';

export interface ComposerBoxHandle {
	getUIChatOptions: () => UIChatOption;
	focus: () => void;
	resetEditor: () => void;
	resetForNewConversation: () => Promise<void>;
	openTemplateMenu: () => void;
	openToolMenu: () => void;
	openAttachmentMenu: () => void;
	loadExternalMessage: (msg: EditorExternalMessage) => void;
	loadToolCalls: (toolCalls: UIToolCall[]) => void;
	setConversationToolsFromChoices: (tools: ToolStoreChoice[]) => void;
	setWebSearchFromChoices: (tools: ToolStoreChoice[]) => void;
	applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => void;
	setSkillStateFromMessage: (
		enabledRefs: SkillRef[],
		activeRefs: SkillRef[],
		options?: {
			syncSession?: 'none' | 'if-session-exists' | 'ensure-if-enabled';
			forceResetSession?: boolean;
		}
	) => void;
	restoreConversationContext: (context: RestorableConversationContext) => void;
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

export const ComposerBox = forwardRef<ComposerBoxHandle, ComposerBoxProps>(function ComposerBox(
	{ onSend, isBusy, isHydrating, abortRef, shortcutConfig, editingMessageId, onCancelEditing },
	ref
) {
	const [abortConfirmationRequested, setAbortConfirmationRequested] = useState(false);
	const isGenerating = isBusy;
	const isInputLocked = isGenerating || isHydrating;

	const editorAreaRef = useRef<EditorAreaHandle>(null);
	const assistantContext = useAssistantContextState();
	const systemPrompt = useComposerSystemPrompt({
		modelDefaultPrompt: assistantContext.selectedModel.systemPrompt,
		modelOptionsLoaded: assistantContext.modelOptionsLoaded,
	});
	const chatOptions = assistantContext.chatOptions;
	const showAbortModal = isGenerating && abortConfirmationRequested;

	const [assistantRuntimeSnapshot, setAssistantRuntimeSnapshot] = useState(EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT);
	const pendingPresetResolutionModeRef = useRef<'none' | 'ensure-active' | 'track-default'>('ensure-active');

	const applyAssistantPresetRuntimeSelections = useCallback((prepared: AssistantPresetPreparedApplication) => {
		if (prepared.runtimeSelections.hasToolsSelection) {
			editorAreaRef.current?.setConversationToolsFromChoices(prepared.runtimeSelections.conversationToolChoices);
			editorAreaRef.current?.setWebSearchFromChoices(prepared.runtimeSelections.webSearchChoices);
		}

		if (prepared.runtimeSelections.hasSkillsSelection) {
			editorAreaRef.current?.setSkillStateFromMessage(
				prepared.runtimeSelections.enabledSkillRefs,
				prepared.runtimeSelections.activeSkillRefs,
				{ syncSession: 'ensure-if-enabled' }
			);
		}

		setAssistantRuntimeSnapshot(prev => ({
			conversationToolChoices: prepared.runtimeSelections.hasToolsSelection
				? prepared.runtimeSelections.conversationToolChoices
				: prev.conversationToolChoices,
			webSearchChoices: prepared.runtimeSelections.hasToolsSelection
				? prepared.runtimeSelections.webSearchChoices
				: prev.webSearchChoices,
			enabledSkillRefs: prepared.runtimeSelections.hasSkillsSelection
				? prepared.runtimeSelections.enabledSkillRefs
				: prev.enabledSkillRefs,
		}));
	}, []);

	const assistantPreset = useAssistantPresetManager({
		context: assistantContext,
		systemPrompt,
		runtimeSnapshot: assistantRuntimeSnapshot,
		applyRuntimeSelections: applyAssistantPresetRuntimeSelections,
	});

	const flushPendingPresetResolution = useCallback(async () => {
		if (!assistantContext.modelOptionsLoaded || assistantPreset.loading || assistantPreset.isApplying) {
			return false;
		}

		if (pendingPresetResolutionModeRef.current === 'track-default') {
			const ok = await assistantPreset.trackDefaultPresetWithoutApplying();
			if (ok) {
				pendingPresetResolutionModeRef.current = 'none';
			}
			return ok;
		}

		if (pendingPresetResolutionModeRef.current === 'ensure-active') {
			const ok = await assistantPreset.ensureActivePreset();
			if (ok) {
				pendingPresetResolutionModeRef.current = 'none';
			}
			return ok;
		}

		return true;
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [
		assistantContext.modelOptionsLoaded,
		assistantPreset.ensureActivePreset,
		assistantPreset.isApplying,
		assistantPreset.loading,
		assistantPreset.trackDefaultPresetWithoutApplying,
	]);

	useEffect(() => {
		void flushPendingPresetResolution();
	}, [flushPendingPresetResolution]);

	const handleSubmitMessage = (payload: EditorSubmitPayload) => {
		// Clear any stale abort confirmation request before starting a new send.
		setAbortConfirmationRequested(false);

		// Return the promise so <EditorArea /> can await it and surface
		// any synchronous errors from sendMessage (e.g. validation).
		return onSend(payload, chatOptions);
	};

	useImperativeHandle(
		ref,
		() => ({
			getUIChatOptions: () => chatOptions,
			focus: () => {
				editorAreaRef.current?.focus();
			},
			resetEditor: () => {
				editorAreaRef.current?.resetEditor();
			},
			resetForNewConversation: async () => {
				setAbortConfirmationRequested(false);
				pendingPresetResolutionModeRef.current = 'none';
				editorAreaRef.current?.resetEditor();
				editorAreaRef.current?.setConversationToolsFromChoices([]);
				editorAreaRef.current?.setWebSearchFromChoices([]);
				editorAreaRef.current?.setSkillStateFromMessage([], [], { syncSession: 'none', forceResetSession: true });
				setAssistantRuntimeSnapshot(EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT);

				const nextSelectedModel = assistantContext.resetForNewConversation();
				systemPrompt.resetForNewConversation(nextSelectedModel.systemPrompt);
				const ok = await assistantPreset.resetToBasePreset();
				if (!ok) {
					pendingPresetResolutionModeRef.current = 'ensure-active';
					void flushPendingPresetResolution();
				}
			},

			openTemplateMenu: () => {
				editorAreaRef.current?.openTemplateMenu();
			},
			openToolMenu: () => {
				editorAreaRef.current?.openToolMenu();
			},
			openAttachmentMenu: () => {
				editorAreaRef.current?.openAttachmentMenu();
			},
			loadExternalMessage: msg => {
				editorAreaRef.current?.loadExternalMessage(msg);
			},
			loadToolCalls: toolCalls => {
				editorAreaRef.current?.loadToolCalls(toolCalls);
			},
			applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => {
				editorAreaRef.current?.applyAttachmentsDrop(payload);
			},
			setConversationToolsFromChoices: tools => {
				editorAreaRef.current?.setConversationToolsFromChoices(tools);
				setAssistantRuntimeSnapshot(prev => ({
					...prev,
					conversationToolChoices: [...tools],
				}));
			},
			setWebSearchFromChoices: choices => {
				editorAreaRef.current?.setWebSearchFromChoices(choices);
				setAssistantRuntimeSnapshot(prev => ({
					...prev,
					webSearchChoices: [...choices],
				}));
			},
			setSkillStateFromMessage: (enabledRefs, activeRefs, options) => {
				editorAreaRef.current?.setSkillStateFromMessage(enabledRefs, activeRefs, options);
				setAssistantRuntimeSnapshot(prev => ({
					...prev,
					enabledSkillRefs: [...enabledRefs],
				}));
			},
			restoreConversationContext: context => {
				setAbortConfirmationRequested(false);
				const nextSelectedModel = assistantContext.restoreConversationContext(context);
				systemPrompt.restoreConversationContext(nextSelectedModel?.systemPrompt ?? '', context.modelParam);
				editorAreaRef.current?.setConversationToolsFromChoices(context.toolChoices);
				editorAreaRef.current?.setWebSearchFromChoices(context.webSearchChoices);
				editorAreaRef.current?.setSkillStateFromMessage(context.enabledSkillRefs, context.activeSkillRefs, {
					syncSession: 'none',
					forceResetSession: true,
				});
				setAssistantRuntimeSnapshot({
					conversationToolChoices: [...context.toolChoices],
					webSearchChoices: [...context.webSearchChoices],
					enabledSkillRefs: [...context.enabledSkillRefs],
				});
				pendingPresetResolutionModeRef.current = 'track-default';
				void flushPendingPresetResolution();
			},
		}),
		[assistantContext, assistantPreset, chatOptions, flushPendingPresetResolution, systemPrompt]
	);

	return (
		<div className="bg-base-200 w-full min-w-0">
			<EditorContextBar context={assistantContext} assistantPreset={assistantPreset} systemPrompt={systemPrompt} />

			<DeleteConfirmationModal
				isOpen={showAbortModal}
				onClose={() => {
					setAbortConfirmationRequested(false);
				}}
				onConfirm={() => {
					setAbortConfirmationRequested(false);
					abortRef.current?.abort();
				}}
				title="Abort generation?"
				message="Partial answer that has already been received will stay in the chat. Do you want to stop the request?"
				confirmButtonText="Abort"
			/>

			<div className="overflow-x-hidden overflow-y-auto">
				<EditorArea
					ref={editorAreaRef}
					isGenerating={isGenerating}
					isInputLocked={isInputLocked}
					currentProviderSDKType={chatOptions.providerSDKType}
					shortcutConfig={shortcutConfig}
					onSubmit={handleSubmitMessage}
					onRequestStop={() => {
						if (isGenerating) {
							setAbortConfirmationRequested(true);
						}
					}}
					onAssistantPresetRuntimeStateChange={setAssistantRuntimeSnapshot}
					editingMessageId={editingMessageId}
					cancelEditing={onCancelEditing}
					systemPrompt={systemPrompt}
				/>
			</div>
		</div>
	);
});
