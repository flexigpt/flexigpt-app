import { useEffect } from 'react';

import type { PromptRoleEnum, PromptTemplateRef } from '@/spec/prompt';

export type SetSystemPromptForChatDetail = {
	prompt: string;
	role?: PromptRoleEnum.System | PromptRoleEnum.Developer;
	displayName?: string;
	sourceTemplate?: PromptTemplateRef;
};

const SET_SYSTEM_PROMPT_FOR_CHAT_EVENT = 'chat:set-system-prompt-for-chat';

export function dispatchSetSystemPromptForChat(detail: SetSystemPromptForChatDetail) {
	window.dispatchEvent(
		new CustomEvent<SetSystemPromptForChatDetail>(SET_SYSTEM_PROMPT_FOR_CHAT_EVENT, {
			detail,
		})
	);
}

export function useSetSystemPromptForChat(handler: (detail: SetSystemPromptForChatDetail) => void) {
	useEffect(() => {
		const onEvt = (e: Event) => {
			const ce = e as CustomEvent<SetSystemPromptForChatDetail>;
			handler(ce.detail);
		};
		window.addEventListener(SET_SYSTEM_PROMPT_FOR_CHAT_EVENT, onEvt as EventListener);
		return () => {
			window.removeEventListener(SET_SYSTEM_PROMPT_FOR_CHAT_EVENT, onEvt as EventListener);
		};
	}, [handler]);
}
