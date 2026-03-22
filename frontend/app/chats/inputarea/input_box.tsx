import { forwardRef, type RefObject, useImperativeHandle, useRef, useState } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { RestorableConversationContext } from '@/spec/conversation';
import type { UIToolCall } from '@/spec/inference';
import { type UIChatOption } from '@/spec/modelpreset';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { AssistantContextBar } from '@/chats/inputarea/assitantcontexts/context_bar';
import { useAssistantContextState } from '@/chats/inputarea/assitantcontexts/use_assistant_context_state';
import { EditorArea, type EditorAreaHandle } from '@/chats/inputarea/input_editor';
import type { EditorExternalMessage, EditorSubmitPayload } from '@/chats/inputarea/input_editor_utils';

export interface InputBoxHandle {
	getUIChatOptions: () => UIChatOption;
	focus: () => void;
	resetEditor: () => void;
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
			setConversationToolsFromChoices: tools => {
				inputAreaRef.current?.setConversationToolsFromChoices(tools);
			},
			setWebSearchFromChoices: choices => {
				inputAreaRef.current?.setWebSearchFromChoices(choices);
			},
			applyAttachmentsDrop: (payload: AttachmentsDroppedPayload) => {
				inputAreaRef.current?.applyAttachmentsDrop(payload);
			},
			setEnabledSkillRefsFromMessage: (refs: SkillRef[]) => {
				inputAreaRef.current?.setEnabledSkillRefsFromMessage(refs);
			},
			setActiveSkillRefsFromMessage: (refs: SkillRef[]) => {
				inputAreaRef.current?.setActiveSkillRefsFromMessage(refs);
			},
			restoreConversationContext: context => {
				assistantContext.restoreConversationContext(context);
			},
		}),
		[assistantContext, chatOptions]
	);

	return (
		<div className="bg-base-200 w-full min-w-0">
			<AssistantContextBar context={assistantContext} />

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
					editingMessageId={editingMessageId}
					cancelEditing={onCancelEditing}
				/>
			</div>
		</div>
	);
});
