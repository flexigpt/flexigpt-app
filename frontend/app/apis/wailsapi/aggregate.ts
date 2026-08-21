import type { StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, ModelParam, ProviderName } from '@/spec/inference';
import type { MCPConversationContext } from '@/spec/mcp_artifact';
import type { ModelPresetID, PostProviderPresetPayload } from '@/spec/modelpreset';
import type { AuthKeyName, AuthKeyType } from '@/spec/setting';
import type { ToolStoreChoice } from '@/spec/tool';
import type { ApplyUnifiedDiffArgs, ApplyUnifiedDiffOut } from '@/spec/unified_diff';

import { ensureMakeID } from '@/lib/uuid_utils';

import type { IAggregateAPI } from '@/apis/interface';
import {
	createAbortError,
	optionalWailsBody,
	requireNonBlankString,
	requireWailsBody,
	throwIfAborted,
} from '@/apis/wailsapi/transport';
import {
	ApplyUnifiedDiff,
	CancelCompletion,
	DeleteAuthKey,
	DeleteProviderPreset,
	FetchCompletion,
	PostProviderPreset,
	SetAuthKey,
} from '@/apis/wailsjs/go/main/AggregrateWrapper';
import type { texttool as texttoolSpec, spec as wailsSpec } from '@/apis/wailsjs/go/models';
import { EventsOff, EventsOn } from '@/apis/wailsjs/runtime/runtime';

const activeCompletionRequestIDs = new Set<string>();

export class WailsAggregateAPI implements IAggregateAPI {
	async applyUnifiedDiff(args: ApplyUnifiedDiffArgs): Promise<ApplyUnifiedDiffOut> {
		const resp = await ApplyUnifiedDiff(args as texttoolSpec.ApplyUnifiedDiffArgs);
		return requireWailsBody(resp as ApplyUnifiedDiffOut | null | undefined, 'ApplyUnifiedDiff');
	}

	async postProviderPreset(providerName: ProviderName, payload: PostProviderPresetPayload): Promise<void> {
		const r = {
			ProviderName: requireNonBlankString(providerName, 'providerName'),
			Body: payload as wailsSpec.PostProviderPresetRequestBody,
		};
		await PostProviderPreset(r as wailsSpec.PostProviderPresetRequest);
	}

	async deleteProviderPreset(providerName: ProviderName): Promise<void> {
		const r = {
			ProviderName: requireNonBlankString(providerName, 'providerName'),
		};
		await DeleteProviderPreset(r as wailsSpec.DeleteProviderPresetRequest);
	}

	async deleteAuthKey(type: AuthKeyType, keyName: AuthKeyName): Promise<void> {
		const r = {
			Type: type,
			KeyName: keyName,
		};
		await DeleteAuthKey(r as wailsSpec.DeleteAuthKeyRequest);
	}

	async setAuthKey(type: AuthKeyType, keyName: AuthKeyName, secret: string): Promise<void> {
		const r = {
			Type: type,
			KeyName: keyName,
			Body: {
				secret: secret,
			},
		};
		await SetAuthKey(r as wailsSpec.SetAuthKeyRequest);
	}

	// Need an eventflow for getting completion.
	// Implemented that in main App Wrapper than aiprovider go package.
	// Wrapper redirects to providerSet after doing event handling
	async fetchCompletion(
		provider: ProviderName,
		modelPresetID: ModelPresetID,
		modelParams: ModelParam,
		current: StoreConversationMessage,
		history?: StoreConversationMessage[],
		toolStoreChoices?: ToolStoreChoice[],
		mcpContext?: MCPConversationContext,
		skillSessionID?: string,
		requestId?: string,
		signal?: AbortSignal,
		onStreamTextData?: (text: string) => void,
		onStreamThinkingData?: (text: string) => void
	): Promise<CompletionResponseBody | undefined> {
		const rid = requireNonBlankString(ensureMakeID(requestId), 'requestId');

		// Do not subscribe to events or invoke Go when the caller cancelled before
		// the request reached this boundary.
		throwIfAborted(signal);

		let textCallbackId = '';
		let thinkingCallbackId = '';
		let abortHandler: (() => void) | undefined;

		const body = {
			modelParam: modelParams as wailsSpec.ModelParam,
			current: current as wailsSpec.ConversationMessage,
			history: history ? ([...history] as wailsSpec.ConversationMessage[]) : [],
			toolStoreChoices: toolStoreChoices ? ([...toolStoreChoices] as wailsSpec.ToolStoreChoice[]) : [],
			mcpContext: mcpContext,
			skillSessionID: skillSessionID ?? '',
		} as wailsSpec.CompletionRequestBody;

		let completionStarted = false;
		let abortHandled = false;

		if (activeCompletionRequestIDs.has(rid)) {
			throw new Error(`A completion with request ID ${rid} is already active.`);
		}

		activeCompletionRequestIDs.add(rid);

		try {
			/*
			 * Event registration belongs inside the try block. If the second
			 * registration or the bridge invocation throws synchronously, all
			 * successfully registered events are still removed in finally.
			 */
			if (onStreamTextData) {
				textCallbackId = `text-${rid}`;

				let lastText = '';
				const textCb = (t: string) => {
					if (t !== lastText) {
						lastText = t;
						onStreamTextData(t);
					}
				};
				EventsOn(textCallbackId, textCb);
			}

			if (onStreamThinkingData) {
				thinkingCallbackId = `thinking-${rid}`;
				let lastThinking = '';
				const thinkingCb = (t: string) => {
					if (t !== lastThinking) {
						lastThinking = t;
						onStreamThinkingData(t);
					}
				};
				EventsOn(thinkingCallbackId, thinkingCb);
			}

			// oxlint-disable-next-line promise/param-names
			const abortPromise = new Promise<never>((_, reject) => {
				if (!signal) {
					return;
				}

				abortHandler = () => {
					if (abortHandled) {
						return;
					}

					abortHandled = true;
					if (completionStarted) {
						void CancelCompletion(rid).catch(() => {});
					}
					reject(createAbortError());
				};

				signal.addEventListener('abort', abortHandler, { once: true });
			});

			// Catch an abort that happened after the initial check but before
			// the listener above was installed.
			if (signal?.aborted) {
				abortHandler?.();
			}

			if (abortHandled) {
				await abortPromise;
			}

			completionStarted = true;
			const responsePromise = FetchCompletion(provider, modelPresetID, body, textCallbackId, thinkingCallbackId, rid);
			const resp = await Promise.race([responsePromise, abortPromise]);

			if (resp === null || typeof resp !== 'object' || Array.isArray(resp)) {
				throw new TypeError('FetchCompletion returned an invalid response.');
			}

			const respBody = optionalWailsBody(resp.Body, 'FetchCompletion');
			return respBody as CompletionResponseBody | undefined;
		} finally {
			if (signal && abortHandler) {
				signal.removeEventListener('abort', abortHandler);
			}

			// Local event cleanup
			if (textCallbackId) {
				EventsOff(textCallbackId);
			}
			if (thinkingCallbackId) {
				EventsOff(thinkingCallbackId);
			}

			activeCompletionRequestIDs.delete(rid);
		}
	}

	async cancelCompletion(requestId: string): Promise<void> {
		await CancelCompletion(requireNonBlankString(requestId, 'requestId'));
	}
}
