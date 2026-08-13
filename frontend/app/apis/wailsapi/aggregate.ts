import type { StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, ModelParam, ProviderName } from '@/spec/inference';
import type { MCPConversationContext } from '@/spec/mcp_artifact';
import type { ModelPresetID, PostProviderPresetPayload } from '@/spec/modelpreset';
import type { AuthKeyName, AuthKeyType } from '@/spec/setting';
import type { ToolStoreChoice } from '@/spec/tool';
import type { ApplyUnifiedDiffArgs, ApplyUnifiedDiffOut } from '@/spec/unified_diff';

import { ensureMakeID } from '@/lib/uuid_utils';

import type { IAggregateAPI } from '@/apis/interface';
import { optionalWailsBody, requireNonBlankString, throwIfAborted } from '@/apis/wailsapi/transport';
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

export class WailsAggregateAPI implements IAggregateAPI {
	async applyUnifiedDiff(args: ApplyUnifiedDiffArgs): Promise<ApplyUnifiedDiffOut> {
		const resp = await ApplyUnifiedDiff(args as texttoolSpec.ApplyUnifiedDiffArgs);
		if (resp === null || typeof resp !== 'object') {
			throw new Error('ApplyUnifiedDiff returned an invalid response.');
		}
		return resp as ApplyUnifiedDiffOut;
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
		const rid = ensureMakeID(requestId);

		// Do not subscribe to events or invoke Go when the caller cancelled before
		// the request reached this boundary.
		throwIfAborted(signal);

		let textCallbackId = '';
		let thinkingCallbackId = '';
		let abortHandler: (() => void) | undefined;

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

		const body = {
			modelParam: modelParams as wailsSpec.ModelParam,
			current: current as wailsSpec.ConversationMessage,
			history: history ? ([...history] as wailsSpec.ConversationMessage[]) : [],
			toolStoreChoices: toolStoreChoices ? ([...toolStoreChoices] as wailsSpec.ToolStoreChoice[]) : [],
			mcpContext: mcpContext,
			skillSessionID: skillSessionID ?? '',
		} as wailsSpec.CompletionRequestBody;

		// oxlint-disable-next-line promise/param-names
		const abortPromise = new Promise<never>((_, reject) => {
			if (!signal) {
				return;
			}

			abortHandler = () => {
				void CancelCompletion(rid).catch(() => {});
				reject(new DOMException('Aborted', 'AbortError'));
			};

			signal.addEventListener('abort', abortHandler, { once: true });
		});

		// Start backend call only after abort handling is attached / checked.
		const responsePromise = FetchCompletion(provider, modelPresetID, body, textCallbackId, thinkingCallbackId, rid);

		try {
			const resp = await Promise.race([responsePromise, abortPromise]);
			const respBody = optionalWailsBody(resp.Body);
			return respBody as CompletionResponseBody | undefined;
		} finally {
			// Always clean up

			// Detach the abort handler if it was attached
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
		}
	}

	async cancelCompletion(requestId: string): Promise<void> {
		await CancelCompletion(requireNonBlankString(requestId, 'requestId'));
	}
}
