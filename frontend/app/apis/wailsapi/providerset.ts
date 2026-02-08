import type { StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, ModelParam, ProviderName } from '@/spec/inference';
import type { ToolStoreChoice } from '@/spec/tool';

import { ensureMakeID } from '@/lib/uuid_utils';

import type { IProviderSetAPI } from '@/apis/interface';
import { CancelCompletion, FetchCompletion } from '@/apis/wailsjs/go/main/ProviderSetWrapper';
import type { spec as wailsSpec } from '@/apis/wailsjs/go/models';
import { EventsOff, EventsOn } from '@/apis/wailsjs/runtime/runtime';

/**
 * @public
 */
export class WailsProviderSetAPI implements IProviderSetAPI {
	// Need an eventflow for getting completion.
	// Implemented that in main App Wrapper than aiprovider go package.
	// Wrapper redirects to providerSet after doing event handling
	async fetchCompletion(
		provider: ProviderName,
		modelParams: ModelParam,
		current: StoreConversationMessage,
		history?: StoreConversationMessage[],
		toolStoreChoices?: ToolStoreChoice[],
		requestId?: string,
		signal?: AbortSignal,
		onStreamTextData?: (text: string) => void,
		onStreamThinkingData?: (text: string) => void
	): Promise<CompletionResponseBody | undefined> {
		const rid = ensureMakeID(requestId);

		let textCallbackId = '';
		let thinkingCallbackId = '';
		let abortHandler: (() => void) | undefined;

		if (onStreamTextData) {
			textCallbackId = `text-${rid}`;

			let lastTrimmedText = '';
			const textCb = (t: string) => {
				const d = t.trim();
				if (d !== lastTrimmedText) {
					lastTrimmedText = d;
					onStreamTextData(t);
				}
			};
			EventsOn(textCallbackId, textCb);
		}

		if (onStreamThinkingData) {
			thinkingCallbackId = `thinking-${rid}`;
			let lastTrimmedThinking = '';
			const thinkingCb = (t: string) => {
				const d = t.trim();
				if (d !== lastTrimmedThinking) {
					lastTrimmedThinking = d;
					onStreamThinkingData(t);
				}
			};
			EventsOn(thinkingCallbackId, thinkingCb);
		}

		const body = {
			modelParam: modelParams as wailsSpec.ModelParam,
			current: current as wailsSpec.ConversationMessage,
			history: history ? ([...history] as wailsSpec.ConversationMessage[]) : [],
			toolStoreChoices: toolStoreChoices ? ([...toolStoreChoices] as wailsSpec.ToolStoreChoice[]) : [],
		} as wailsSpec.CompletionRequestBody;

		const abortPromise: Promise<never> = new Promise((_, reject) => {
			if (!signal) return;

			// Already aborted before we even start => do NOT start backend call.

			if (signal.aborted) {
				reject(new DOMException('Aborted', 'AbortError'));
				return;
			}

			abortHandler = () => {
				// Detach server-side
				this.cancelCompletion(rid).catch(() => {});
				reject(new DOMException('Aborted', 'AbortError'));
			};

			signal.addEventListener('abort', abortHandler, { once: true });
		});

		// Start backend call only after abort handling is attached / checked.
		const responsePromise = FetchCompletion(provider, body, textCallbackId, thinkingCallbackId, rid);

		try {
			const resp = await Promise.race([responsePromise, abortPromise]);
			const respBody = resp.Body as wailsSpec.CompletionResponseBody;
			return respBody as CompletionResponseBody;
		} finally {
			// Always clean up

			// Detach the abort handler if it was attached
			if (signal && abortHandler) {
				signal.removeEventListener('abort', abortHandler);
			}

			// Local event cleanup
			if (textCallbackId) EventsOff(textCallbackId);
			if (thinkingCallbackId) EventsOff(thinkingCallbackId);
		}
	}

	async cancelCompletion(requestId: string): Promise<void> {
		if (!requestId) return;
		try {
			await CancelCompletion(requestId);
		} catch {
			/* Swallow any Go-side error; we only care that the signal aborts */
		}
	}
}
