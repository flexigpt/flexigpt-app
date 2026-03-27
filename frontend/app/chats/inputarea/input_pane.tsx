import { memo, useCallback } from 'react';

import type { UIChatOption } from '@/spec/modelpreset';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import type { EditorSubmitPayload } from '@/chats/inputarea/editor/input_editor_utils';
import { InputBox, type InputBoxHandle } from '@/chats/inputarea/input_box';

export const InputPane = memo(function InputPane(props: {
	tabId: string;
	active: boolean;
	isBusy: boolean;
	isHydrating: boolean;
	editingMessageId: string | null;
	setInputRef: (tabId: string) => (inst: InputBoxHandle | null) => void;
	getAbortRef: (tabId: string) => { current: AbortController | null };
	shortcutConfig: ShortcutConfig;
	sendMessage: (tabId: string, payload: EditorSubmitPayload, options: UIChatOption) => Promise<void>;
	cancelEditing: (tabId: string) => void;
}) {
	const {
		tabId,
		active,
		isBusy,
		isHydrating,
		editingMessageId,
		setInputRef,
		getAbortRef,
		shortcutConfig,
		sendMessage,
		cancelEditing,
	} = props;

	const onSend = useCallback(
		(payload: EditorSubmitPayload, options: UIChatOption) => sendMessage(tabId, payload, options),
		[sendMessage, tabId]
	);
	const onCancelEditing = useCallback(() => {
		cancelEditing(tabId);
	}, [cancelEditing, tabId]);

	return (
		<div className={active ? 'block' : 'hidden'}>
			<InputBox
				ref={setInputRef(tabId)}
				onSend={onSend}
				isBusy={isBusy}
				isHydrating={isHydrating}
				abortRef={getAbortRef(tabId)}
				shortcutConfig={shortcutConfig}
				editingMessageId={editingMessageId}
				onCancelEditing={onCancelEditing}
			/>
		</div>
	);
});
