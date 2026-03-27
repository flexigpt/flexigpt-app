import { memo, useCallback } from 'react';

import type { UIChatOption } from '@/spec/modelpreset';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import { ComposerBox, type ComposerBoxHandle } from '@/chats/composer/composer_box';
import type { EditorSubmitPayload } from '@/chats/composer/editor/editor_types';

export const TabInputPane = memo(function TabInputPane(props: {
	tabId: string;
	active: boolean;
	isBusy: boolean;
	isHydrating: boolean;
	editingMessageId: string | null;
	setInputRef: (tabId: string) => (inst: ComposerBoxHandle | null) => void;
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
			<ComposerBox
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
