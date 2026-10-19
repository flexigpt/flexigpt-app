import { memo, useCallback, useMemo } from 'react';

import type { UIModelOption } from '@/spec/model';
import type { SessionConfiguration } from '@/spec/session';

import type { ShortcutConfig } from '@/lib/keyboard_shortcuts';

import type { ComposerBoxHandle } from '@/chats/composer/composer_box';
import type { EditorSubmitPayload } from '@/chats/composer/editor/editor_types';
import { ComposerBox } from '@/chats/composer/composer_box';

export const TabInputPane = memo(function TabInputPane(props: {
	tabId: string;
	active: boolean;
	isBusy: boolean;
	isGenerating: boolean;
	isHydrating: boolean;
	allowContinuationInput: boolean;
	editingMessageId: string | null;
	setInputRef: (tabId: string) => (inst: ComposerBoxHandle | null) => void;
	getAbortRef: (tabId: string) => { current: AbortController | null };
	shortcutConfig: ShortcutConfig;
	sendMessage: (
		tabId: string,
		payload: EditorSubmitPayload,
		options: UIModelOption
	) => Promise<SessionConfiguration | undefined>;
	sendContinuation: (
		tabId: string,
		payload: EditorSubmitPayload,
		options: UIModelOption
	) => Promise<SessionConfiguration | undefined>;
	cancelEditing: (tabId: string) => void;
	cancelRun: (tabId: string) => Promise<void>;
	showAutomationControl: boolean;
	automationHoldEnabled: boolean;
	onToggleAutomationHold: () => void;
}) {
	const {
		tabId,
		active,
		isBusy,
		isGenerating,
		isHydrating,
		allowContinuationInput,
		editingMessageId,
		setInputRef,
		getAbortRef,
		shortcutConfig,
		sendMessage,
		sendContinuation,
		cancelEditing,
		cancelRun,
		showAutomationControl,
		automationHoldEnabled,
		onToggleAutomationHold,
	} = props;

	const composerRef = useMemo(() => setInputRef(tabId), [setInputRef, tabId]);

	const onSend = useCallback(
		(payload: EditorSubmitPayload, options: UIModelOption) =>
			allowContinuationInput ? sendContinuation(tabId, payload, options) : sendMessage(tabId, payload, options),
		[allowContinuationInput, sendContinuation, sendMessage, tabId]
	);
	const onCancelEditing = useCallback(() => {
		cancelEditing(tabId);
	}, [cancelEditing, tabId]);

	return (
		<div className={active ? 'block' : 'hidden'}>
			<ComposerBox
				ref={composerRef}
				onSend={onSend}
				isBusy={isBusy}
				isGenerating={isGenerating}
				isHydrating={isHydrating}
				allowContinuationInput={allowContinuationInput}
				abortRef={getAbortRef(tabId)}
				shortcutConfig={shortcutConfig}
				editingMessageId={editingMessageId}
				onCancelEditing={onCancelEditing}
				onCancelRun={() => {
					return cancelRun(tabId);
				}}
				showAutomationControl={showAutomationControl}
				automationHoldEnabled={automationHoldEnabled}
				onToggleAutomationHold={onToggleAutomationHold}
			/>
		</div>
	);
});
