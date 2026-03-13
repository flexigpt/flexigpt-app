import { forwardRef, type RefObject, useImperativeHandle, useRef, useState } from 'react';

import type { AttachmentsDroppedPayload } from '@/spec/attachment';
import type { UIToolCall } from '@/spec/inference';
import { DefaultUIChatOptions, type UIChatOption } from '@/spec/modelpreset';
import type { SkillRef } from '@/spec/skill';
import type { ToolStoreChoice } from '@/spec/tool';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';

import { AssistantContextBar } from '@/chats/assitantcontexts/context_bar';
import { EditorArea, type EditorAreaHandle } from '@/chats/inputarea/input_editor';
import type { EditorExternalMessage, EditorSubmitPayload } from '@/chats/inputarea/input_editor_utils';

export interface InputBoxHandle {
	getUIChatOptions: () => UIChatOption;
	focus: () => void;
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
}

interface InputBoxProps {
	onSend: (message: EditorSubmitPayload, options: UIChatOption) => Promise<void>;
	isBusy: boolean;
	abortRef: RefObject<AbortController | null>;
	shortcutConfig: ShortcutConfig;
	editingMessageId: string | null;
	onCancelEditing: () => void;
}

export const InputBox = forwardRef<InputBoxHandle, InputBoxProps>(function InputBox(
	{ onSend, isBusy, abortRef, shortcutConfig, editingMessageId, onCancelEditing },
	ref
) {
	const [chatOptions, setUIChatOptions] = useState<UIChatOption>(DefaultUIChatOptions);
	const [abortConfirmationRequested, setAbortConfirmationRequested] = useState(false);

	const inputAreaRef = useRef<EditorAreaHandle>(null);

	const showAbortModal = isBusy && abortConfirmationRequested;

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
		}),
		[chatOptions]
	);

	return (
		<div className="bg-base-200 w-full min-w-0">
			<AssistantContextBar onOptionsChange={setUIChatOptions} /* hand the aggregated options up */ />

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
					isBusy={isBusy}
					currentProviderSDKType={chatOptions.providerSDKType}
					shortcutConfig={shortcutConfig}
					onSubmit={handleSubmitMessage}
					onRequestStop={() => {
						if (isBusy) {
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
