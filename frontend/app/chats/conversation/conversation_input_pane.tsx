import { memo, useCallback, useMemo } from 'react';

import type { UIToolCall, UIToolOutput } from '@/spec/inference';
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
	showPendingToolControls: boolean;
	canContinueWithoutDraft: boolean;
	onRunPendingTools: (continueRun: boolean) => void;
	sessionToolCalls: UIToolCall[];
	sessionToolOutputs: UIToolOutput[];
	onRunSessionToolCall: (call: UIToolCall) => void;
	onRetrySessionToolCall: (call: UIToolCall) => void;
	onSkipSessionToolCall: (call: UIToolCall) => void;
	stagedAppContextUpdateCount: number;
	onDiscardStagedAppContext: () => void;
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
		shortcutConfig,
		sendMessage,
		sendContinuation,
		cancelEditing,
		cancelRun,
		showAutomationControl,
		automationHoldEnabled,
		onToggleAutomationHold,
		showPendingToolControls,
		canContinueWithoutDraft,
		onRunPendingTools,
		sessionToolCalls,
		sessionToolOutputs,
		onRunSessionToolCall,
		onRetrySessionToolCall,
		onSkipSessionToolCall,
		stagedAppContextUpdateCount,
		onDiscardStagedAppContext,
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
				shortcutConfig={shortcutConfig}
				editingMessageId={editingMessageId}
				onCancelEditing={onCancelEditing}
				onCancelRun={() => {
					return cancelRun(tabId);
				}}
				showAutomationControl={showAutomationControl}
				automationHoldEnabled={automationHoldEnabled}
				onToggleAutomationHold={onToggleAutomationHold}
				showPendingToolControls={showPendingToolControls}
				canContinueWithoutDraft={canContinueWithoutDraft}
				onRunPendingTools={onRunPendingTools}
				sessionToolCalls={sessionToolCalls}
				sessionToolOutputs={sessionToolOutputs}
				onRunSessionToolCall={onRunSessionToolCall}
				onRetrySessionToolCall={onRetrySessionToolCall}
				onSkipSessionToolCall={onSkipSessionToolCall}
				stagedAppContextUpdateCount={stagedAppContextUpdateCount}
				onDiscardStagedAppContext={onDiscardStagedAppContext}
			/>
		</div>
	);
});
