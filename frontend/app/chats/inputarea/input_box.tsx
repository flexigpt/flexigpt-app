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
} from '@/chats/inputarea/assitantcontexts/assistant_preset_runtime';
import { AssistantContextBar } from '@/chats/inputarea/assitantcontexts/context_bar';
import { useAssistantContextState } from '@/chats/inputarea/assitantcontexts/use_assistant_context_state';
import { useAssistantPresetManager } from '@/chats/inputarea/assitantcontexts/use_assistant_preset_manager';
import { EditorArea, type EditorAreaHandle } from '@/chats/inputarea/editor/input_editor';
import type { EditorExternalMessage, EditorSubmitPayload } from '@/chats/inputarea/editor/input_editor_utils';

export interface InputBoxHandle {
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
	setEnabledSkillRefsFromMessage: (refs: SkillRef[]) => void;
	setActiveSkillRefsFromMessage: (refs: SkillRef[]) => void;
	restoreConversationContext: (context: RestorableConversationContext) => void;
}

interface InputBoxProps {
	onSend: (message: EditorSubmitPayload, options: UIChatOption) => Promise<void>;
	isBusy: boolean;
	isHydrating: boolean;
	abortRef: RefObject<AbortController | null>;
	shortcutConfig: ShortcutConfig;
	editingMessageId: string | null;
	onCancelEditing: () => void;
}

export const InputBox = forwardRef<InputBoxHandle, InputBoxProps>(function InputBox(
	{ onSend, isBusy, isHydrating, abortRef, shortcutConfig, editingMessageId, onCancelEditing },
	ref
) {
	const [abortConfirmationRequested, setAbortConfirmationRequested] = useState(false);
	const isGenerating = isBusy;
	const isInputLocked = isGenerating || isHydrating;

	const inputAreaRef = useRef<EditorAreaHandle>(null);
	const assistantContext = useAssistantContextState();
	const chatOptions = assistantContext.chatOptions;
	const showAbortModal = isGenerating && abortConfirmationRequested;

	const [assistantRuntimeSnapshot, setAssistantRuntimeSnapshot] = useState(EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT);
	const pendingPresetResolutionModeRef = useRef<'none' | 'ensure-active' | 'track-default'>('ensure-active');

	const applyAssistantPresetRuntimeSelections = useCallback((prepared: AssistantPresetPreparedApplication) => {
		if (prepared.runtimeSelections.hasToolsSelection) {
			inputAreaRef.current?.setConversationToolsFromChoices(prepared.runtimeSelections.conversationToolChoices);
			inputAreaRef.current?.setWebSearchFromChoices(prepared.runtimeSelections.webSearchChoices);
		}

		if (prepared.runtimeSelections.hasSkillsSelection) {
			inputAreaRef.current?.setEnabledSkillRefsFromMessage(prepared.runtimeSelections.enabledSkillRefs);
			inputAreaRef.current?.setActiveSkillRefsFromMessage([]);
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
				inputAreaRef.current?.focus();
			},
			resetEditor: () => {
				inputAreaRef.current?.resetEditor();
			},
			resetForNewConversation: async () => {
				setAbortConfirmationRequested(false);
				pendingPresetResolutionModeRef.current = 'none';
				inputAreaRef.current?.resetEditor();
				inputAreaRef.current?.setConversationToolsFromChoices([]);
				inputAreaRef.current?.setWebSearchFromChoices([]);
				inputAreaRef.current?.setEnabledSkillRefsFromMessage([]);
				inputAreaRef.current?.setActiveSkillRefsFromMessage([]);
				setAssistantRuntimeSnapshot(EMPTY_ASSISTANT_PRESET_RUNTIME_SNAPSHOT);

				assistantContext.resetForNewConversation();
				const ok = await assistantPreset.resetToBasePreset();
				if (!ok) {
					pendingPresetResolutionModeRef.current = 'ensure-active';
					void flushPendingPresetResolution();
				}
			},

			openTemplateMenu: () => {
				inputAreaRef.current?.openTemplateMenu();
			},
			openToolMenu: () => {
				inputAreaRef.current?.openToolMenu();
			},
			openAttachmentMenu: () => {
				inputAreaRef.current?.openAttachmentMenu();
			},
			loadExternalMessage: msg => {
				inputAreaRef.current?.loadExternalMessage(msg);
			},
			loadToolCalls: toolCalls => {
				inputAreaRef.current?.loadToolCalls(toolCalls);
			},
			applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => {
				inputAreaRef.current?.applyAttachmentsDrop(payload);
			},
			setConversationToolsFromChoices: tools => {
				inputAreaRef.current?.setConversationToolsFromChoices(tools);
				setAssistantRuntimeSnapshot(prev => ({
					...prev,
					conversationToolChoices: [...tools],
				}));
			},
			setWebSearchFromChoices: choices => {
				inputAreaRef.current?.setWebSearchFromChoices(choices);
				setAssistantRuntimeSnapshot(prev => ({
					...prev,
					webSearchChoices: [...choices],
				}));
			},
			setEnabledSkillRefsFromMessage: (refs: SkillRef[]) => {
				inputAreaRef.current?.setEnabledSkillRefsFromMessage(refs);
				setAssistantRuntimeSnapshot(prev => ({
					...prev,
					enabledSkillRefs: [...refs],
				}));
			},
			setActiveSkillRefsFromMessage: (refs: SkillRef[]) => {
				inputAreaRef.current?.setActiveSkillRefsFromMessage(refs);
			},
			restoreConversationContext: context => {
				setAbortConfirmationRequested(false);
				assistantContext.restoreConversationContext(context);
				inputAreaRef.current?.setConversationToolsFromChoices(context.toolChoices);
				inputAreaRef.current?.setWebSearchFromChoices(context.webSearchChoices);
				inputAreaRef.current?.setEnabledSkillRefsFromMessage(context.enabledSkillRefs);
				inputAreaRef.current?.setActiveSkillRefsFromMessage(context.activeSkillRefs);
				setAssistantRuntimeSnapshot({
					conversationToolChoices: [...context.toolChoices],
					webSearchChoices: [...context.webSearchChoices],
					enabledSkillRefs: [...context.enabledSkillRefs],
				});
				pendingPresetResolutionModeRef.current = 'track-default';
				void flushPendingPresetResolution();
			},
		}),
		[assistantContext, assistantPreset, chatOptions, flushPendingPresetResolution]
	);

	return (
		<div className="bg-base-200 w-full min-w-0">
			<AssistantContextBar context={assistantContext} assistantPreset={assistantPreset} />

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
					ref={inputAreaRef}
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
				/>
			</div>
		</div>
	);
});
