import type { ArtifactRef } from '@/spec/artifact';
import type { StoreConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody } from '@/spec/inference';
import type { MCPConversationContext } from '@/spec/mcp';
import type { ModelRequestPatch } from '@/spec/model';
import type { ToolSelection } from '@/spec/tool';

import type { ICompletionAPI } from '@/apis/interface';
import type { main as wailsMain } from '@/apis/wailsjs/go/models';
import { createAbortError, optionalWailsBody, requireNonBlankString, throwIfAborted } from '@/apis/wailsapi/transport';
import { CancelCompletion, FetchCompletion } from '@/apis/wailsjs/go/main/CompletionWrapper';
import { EventsOff, EventsOn } from '@/apis/wailsjs/runtime/runtime';

const activeCompletionRequestIDs = new Set<string>();

export class WailsCompletionAPI implements ICompletionAPI {
	async fetchCompletion(
		model: ArtifactRef,
		requestPatch: ModelRequestPatch | undefined,
		current: StoreConversationMessage,
		history: StoreConversationMessage[] = [],
		toolSelections: ToolSelection[] = [],
		mcpContext?: MCPConversationContext,
		skillSessionID?: string,
		requestID?: string,
		signal?: AbortSignal,
		onStreamTextData?: (text: string) => void,
		onStreamThinkingData?: (thinking: string) => void
	): Promise<CompletionResponseBody | undefined> {
		const id = requireNonBlankString(requestID, 'requestID');
		throwIfAborted(signal);

		if (activeCompletionRequestIDs.has(id)) {
			throw new Error(`A completion with request ID ${id} is already active.`);
		}

		activeCompletionRequestIDs.add(id);

		const requestBody = {
			current,
			history,
			toolSelections,
			requestPatch,
			...(mcpContext ? { mcpContext } : {}),
			skillSessionID: skillSessionID ?? '',
		} as wailsMain.CompletionRequestBody;

		let textEventName = '';
		let thinkingEventName = '';
		let abortHandler: (() => void) | undefined;
		let completionStarted = false;
		let abortHandled = false;

		try {
			if (onStreamTextData) {
				textEventName = `text-${id}`;
				EventsOn(textEventName, onStreamTextData);
			}

			if (onStreamThinkingData) {
				thinkingEventName = `thinking-${id}`;
				EventsOn(thinkingEventName, onStreamThinkingData);
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
						void CancelCompletion(id).catch(() => undefined);
					}
					reject(createAbortError());
				};

				signal.addEventListener('abort', abortHandler, {
					once: true,
				});
			});

			if (signal?.aborted) {
				abortHandler?.();
			}

			if (abortHandled) {
				await abortPromise;
			}

			completionStarted = true;

			const response = await Promise.race([
				FetchCompletion(model, requestBody, textEventName, thinkingEventName, id),
				abortPromise,
			]);

			return optionalWailsBody((response as { Body?: CompletionResponseBody }).Body, 'FetchCompletion');
		} finally {
			if (signal && abortHandler) {
				signal.removeEventListener('abort', abortHandler);
			}

			if (textEventName) {
				EventsOff(textEventName);
			}

			if (thinkingEventName) {
				EventsOff(thinkingEventName);
			}

			activeCompletionRequestIDs.delete(id);
		}
	}

	cancelCompletion(requestID: string): Promise<void> {
		return CancelCompletion(requireNonBlankString(requestID, 'requestID'));
	}
}
