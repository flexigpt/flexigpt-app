import { type ConversationMessage } from '@/spec/conversation';
import type { ProviderName, URLCitation, WebSearchToolOutputItemUnion } from '@/spec/inference';
import {
	CitationKind,
	type CompletionResponseBody,
	ContentItemKind,
	type InferenceError,
	type InferenceUsage,
	type ModelParam,
	OutputKind,
	type OutputUnion,
	type ReasoningContent,
	RoleEnum,
	Status,
	type ToolCall,
	type ToolOutput,
	type UIToolCall,
	type UIToolOutput,
} from '@/spec/inference';
import { type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { getUUIDv7 } from '@/lib/uuid_utils';

import { providerSetAPI } from '@/apis/baseapi';

import {
	collectToolCallsFromOutputs,
	extractPrimaryTextFromToolStoreOutputs,
	formatToolOutputSummary,
	mapToolOutputItemsToToolStoreOutputs,
} from '@/chats/tools/tool_editor_utils';

export async function HandleCompletion(
	provider: ProviderName,
	modelParams: ModelParam,
	currentUserMsg: ConversationMessage,
	history: ConversationMessage[],
	toolStoreChoices: ToolStoreChoice[] | undefined,
	assistantPlaceholder: ConversationMessage,
	requestId?: string,
	signal?: AbortSignal,
	onStreamTextData?: (textData: string) => void,
	onStreamThinkingData?: (thinkingData: string) => void
): Promise<{
	responseMessage: ConversationMessage | undefined;
	rawResponse?: CompletionResponseBody;
}> {
	try {
		// console.log('history to completion', JSON.stringify(history, null, 2));
		const choiceMap = new Map<string, ToolStoreChoice>(
			(toolStoreChoices ?? []).map(choice => [choice.choiceID, choice])
		);

		const resp = await providerSetAPI.fetchCompletion(
			provider,
			modelParams,
			currentUserMsg,
			history,
			toolStoreChoices,
			requestId,
			signal,
			onStreamTextData,
			onStreamThinkingData
		);

		if (!resp) {
			return { responseMessage: undefined, rawResponse: undefined };
		}

		const inf = resp.inferenceResponse;
		const hasModelError = !!inf?.error;
		const hasOutputs = !!inf?.outputs && inf.outputs.length > 0;

		if (!hasModelError || hasOutputs) {
			const assistantMsg = buildAssistantMessageFromResponse(assistantPlaceholder.id, modelParams, resp, choiceMap);
			return { responseMessage: assistantMsg, rawResponse: resp };
		}

		// Error with no outputs at all -> fall back to existing "error stub".
		return getErrorStub(modelParams, assistantPlaceholder, resp, undefined);
	} catch (error) {
		if ((error as DOMException).name === 'AbortError') {
			throw error;
		}
		return getErrorStub(modelParams, assistantPlaceholder, undefined, error);
	}
}

function getErrorStub(
	modelParams: ModelParam,
	assistantPlaceholder: ConversationMessage,
	rawResponse: CompletionResponseBody | undefined,
	errorObj: any
) {
	assistantPlaceholder.modelParam = modelParams;
	assistantPlaceholder.status = Status.Failed;
	// Optionally keep the raw error on the message
	if (rawResponse?.inferenceResponse && rawResponse.inferenceResponse.error) {
		assistantPlaceholder.error = rawResponse.inferenceResponse.error;
	} else {
		assistantPlaceholder.error = {
			code: 'unknown',
			message: JSON.stringify(errorObj, null, 2),
		} as InferenceError;
	}

	const outText = (assistantPlaceholder.uiContent || '') + '\n\n>Got error in API processing.';
	assistantPlaceholder.uiContent = outText;
	assistantPlaceholder.outputs = [
		{
			kind: OutputKind.OutputMessage,
			outputMessage: {
				id: getUUIDv7(),
				role: RoleEnum.Assistant,
				status: Status.Failed,
				contents: [
					{
						kind: ContentItemKind.Text,
						textItem: {
							text: outText,
						},
					},
				],
			},
		},
	];

	// Prefer backend debugDetails if present
	let detailsMarkdown: string = '';
	if (rawResponse?.inferenceResponse) {
		detailsMarkdown =
			getDebugDetailsMarkdown(rawResponse.inferenceResponse.debugDetails, rawResponse.inferenceResponse.error) ?? '';
		assistantPlaceholder.debugDetails = rawResponse.inferenceResponse.debugDetails;
	}

	if (errorObj !== undefined && errorObj !== null) {
		detailsMarkdown = detailsMarkdown + '\n\n### Error\n\n' + getQuotedJSON(errorObj);
	}

	assistantPlaceholder.uiDebugDetails = detailsMarkdown;

	return { responseMessage: assistantPlaceholder, rawResponse };
}

export function getDebugDetailsMarkdown(debugObj?: any, errorObj?: any): string | undefined {
	const parts: string[] = [];

	const pushJSONBlock = (title: string, value: unknown) => {
		if (value === undefined) return;

		try {
			parts.push(title, getQuotedJSON(value));
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			parts.push(title, `\`[Failed to serialize ${title.replace(/^#+\s*/, '').toLowerCase()}: ${msg}]\``);
		}
	};

	// 1. Error object should always be first, if present
	if (errorObj !== undefined) {
		pushJSONBlock('### Error', errorObj);
	}

	// 2. Handle debug details
	if (debugObj !== undefined && debugObj !== null) {
		const isMap = debugObj instanceof Map;
		const isObjectLike = typeof debugObj === 'object' && !isMap;

		const asRecord: Record<string, unknown> | undefined = isObjectLike
			? (debugObj as Record<string, unknown>)
			: undefined;

		const hasKey = (key: string): boolean => {
			if (isMap) return (debugObj as Map<string, unknown>).has(key);
			if (asRecord) return key in asRecord;
			return false;
		};

		const getValue = (key: string): unknown => {
			if (isMap) return (debugObj as Map<string, unknown>).get(key);
			if (asRecord) return asRecord[key];
			return undefined;
		};

		const hasRequest = hasKey('requestDetails');
		const hasResponse = hasKey('responseDetails');
		const hasProvider = hasKey('providerResponse');
		const hasErrorDetails = hasKey('errorDetails');

		const hasStructuredKeys = hasRequest || hasResponse || hasProvider || hasErrorDetails;

		if (hasStructuredKeys) {
			// Order: error details (from debug) → request → response → provider
			if (hasErrorDetails) {
				pushJSONBlock('### Error details', getValue('errorDetails'));
			}

			if (hasRequest) {
				pushJSONBlock('### Request debug details', getValue('requestDetails'));
			}

			if (hasResponse) {
				pushJSONBlock('### Response debug details', getValue('responseDetails'));
			}

			if (hasProvider) {
				pushJSONBlock('### Provider response debug details', getValue('providerResponse'));
			}
		} else {
			// No special keys: fallback to original behavior (one block)
			pushJSONBlock('### Debug details', debugObj);
		}
	}

	if (parts.length === 0) {
		return undefined;
	}

	return parts.join('\n\n');
}

function buildAssistantMessageFromResponse(
	baseId: string,
	modelParams: ModelParam,
	resp: CompletionResponseBody,
	choiceMap: Map<string, ToolStoreChoice>
): ConversationMessage | undefined {
	const now = new Date();
	const id = baseId || getUUIDv7();

	if (!resp.inferenceResponse) {
		return undefined;
	}

	const inf = resp.inferenceResponse;
	const outputs = inf.outputs ?? [];
	const usage: InferenceUsage | undefined = inf.usage;
	const error = inf.error;

	const { uiContent, uiReasoningContents, uiToolCalls, uiToolOutputs, uiCitations } = deriveUIFieldsFromOutputUnion(
		outputs,
		choiceMap
	);
	const debugDetails = inf.debugDetails;
	const uiDebugDetails = getDebugDetailsMarkdown(inf.debugDetails, inf.error);

	const s = error ? Status.Failed : Status.Completed;
	const msg: ConversationMessage = {
		id,
		createdAt: now,
		role: RoleEnum.Assistant,
		status: s,
		modelParam: modelParams,
		outputs,
		usage,
		error,
		debugDetails,
		uiContent,
		uiReasoningContents,
		uiToolCalls,
		uiToolOutputs,
		uiCitations,
		uiDebugDetails,
	};
	// console.log('assistant from message out', JSON.stringify(msg, null, 2));
	return msg;
}

export function deriveUIFieldsFromOutputUnion(
	outputs: OutputUnion[] | undefined,
	choiceMap: Map<string, ToolStoreChoice>
): {
	uiContent: string;
	uiReasoningContents?: ReasoningContent[];
	uiToolCalls?: UIToolCall[];
	uiToolOutputs?: UIToolOutput[];
	uiCitations?: URLCitation[];
} {
	if (!outputs || outputs.length === 0) {
		return { uiContent: '' };
	}

	const textParts: string[] = [];
	const reasoning: ReasoningContent[] = [];
	const toolCalls: UIToolCall[] = [];
	const toolOutputs: UIToolOutput[] = [];
	const citations: URLCitation[] = [];
	const seenCitationKeys = new Set<string>();

	let toolCallMap: Map<string, ToolCall> | undefined;

	for (const o of outputs) {
		switch (o.kind) {
			case OutputKind.OutputMessage: {
				const msg = o.outputMessage;
				if (!msg || !msg.contents) break;
				for (const c of msg.contents) {
					if (c.kind === ContentItemKind.Text && c.textItem) {
						const t = c.textItem.text.trim();
						if (t) textParts.push(t);

						const itemCitations = c.textItem.citations;
						if (itemCitations && itemCitations.length > 0) {
							for (const cit of itemCitations) {
								if (cit.kind !== CitationKind.URL || !cit.urlCitation?.url) continue;
								const u = cit.urlCitation;
								const key = `${u.url}|${u.startIndex ?? ''}|${u.endIndex ?? ''}|${u.title ?? ''}`;
								if (seenCitationKeys.has(key)) continue;
								seenCitationKeys.add(key);
								citations.push(u);
							}
						}
					}
				}
				break;
			}

			case OutputKind.ReasoningMessage:
				if (o.reasoningMessage) {
					reasoning.push(o.reasoningMessage);
				}
				break;

			case OutputKind.FunctionToolCall: {
				const uiFunctionToolCall = deriveUIToolCallFromToolCall(o.functionToolCall, choiceMap);
				if (uiFunctionToolCall) toolCalls.push(uiFunctionToolCall);
				break;
			}
			case OutputKind.CustomToolCall: {
				const uiCustomToolCall = deriveUIToolCallFromToolCall(o.customToolCall, choiceMap);
				if (uiCustomToolCall) toolCalls.push(uiCustomToolCall);
				break;
			}
			case OutputKind.WebSearchToolCall: {
				const uiWebsearchToolCall = deriveUIToolCallFromToolCall(o.webSearchToolCall, choiceMap);
				if (uiWebsearchToolCall) toolCalls.push(uiWebsearchToolCall);
				break;
			}

			case OutputKind.WebSearchToolOutput: {
				const out = o.webSearchToolOutput;
				if (out) {
					if (!toolCallMap) {
						// outputs is definitely defined here because of the early return
						toolCallMap = collectToolCallsFromOutputs(outputs);
					}
					// Only called when a real ToolOutput exists.
					toolOutputs.push(buildUIToolOutputFromToolOutput(out, choiceMap, toolCallMap));
				}
				break;
			}
		}
	}

	const content = textParts.join('\n\n');

	return {
		uiContent: content,
		uiReasoningContents: reasoning.length > 0 ? reasoning : undefined,
		uiToolCalls: toolCalls.length > 0 ? toolCalls : undefined,
		uiToolOutputs: toolOutputs.length > 0 ? toolOutputs : undefined,
		uiCitations: citations.length > 0 ? citations : undefined,
	};
}

function deriveUIToolCallFromToolCall(
	toolCall: ToolCall | undefined,
	choiceMap: Map<string, ToolStoreChoice>
): UIToolCall | undefined {
	if (!toolCall) return undefined;

	const choiceID = toolCall.choiceID;
	if (!choiceID) return undefined;

	const toolStoreChoice = choiceMap.get(choiceID); // ToolStoreChoice | undefined
	if (!toolStoreChoice) return undefined;
	const type = toolCall.type as unknown as ToolStoreChoiceType;

	// For provider-managed web-search, the "call" appearing in outputs
	// means "search was (or will be) handled by the provider", not a
	// pending client-side action. Mark it as 'succeeded' so it is never
	// treated as a pending/runnable chip.
	const status: UIToolCall['status'] = type === ToolStoreChoiceType.WebSearch ? 'succeeded' : 'pending';

	return {
		id: toolCall.id || toolCall.callID,
		callID: toolCall.callID,
		name: toolCall.name,
		arguments: toolCall.arguments ?? '',
		webSearchToolCallItems: toolCall.webSearchToolCallItems,
		type: type,
		choiceID: toolCall.choiceID,
		// The LLM would consider the status of tool call as done as soon as it is in output,
		// for us the call is pending here and then it will run and move to final status.
		status: status,
		toolStoreChoice,
	};
}

export function buildUIToolOutputFromToolOutput(
	out: ToolOutput,
	choiceMap: Map<string, ToolStoreChoice>,
	toolCallMap?: Map<string, ToolCall>
): UIToolOutput {
	const isError = out.isError;
	const toolStoreChoice = choiceMap.get(out.choiceID);
	const call = toolCallMap?.get(out.callID);
	const summaryBase = formatToolOutputSummary(out.name);

	const toolStoreOutputs = mapToolOutputItemsToToolStoreOutputs(out.contents);

	const primaryText = extractPrimaryTextFromToolStoreOutputs(toolStoreOutputs);
	const summary = isError && primaryText ? `Error: ${primaryText.split('\n')[0].slice(0, 80)}` : summaryBase;

	let webSearchOutputs: WebSearchToolOutputItemUnion[] | undefined;
	if (out.webSearchToolOutputItems && out.webSearchToolOutputItems.length > 0) {
		webSearchOutputs = out.webSearchToolOutputItems;
	}
	return {
		id: out.id,
		callID: out.callID,
		name: out.name,
		choiceID: out.choiceID,

		// ToolType and ToolStoreChoiceType share the same string enum values.
		type: out.type as unknown as ToolStoreChoiceType,

		summary: summary,
		toolStoreOutputs: toolStoreOutputs,
		webSearchToolOutputItems: webSearchOutputs,

		toolStoreChoice:
			toolStoreChoice ??
			({
				// Very defensive fallback; ideally you never hit this.
				choiceID: out.choiceID,
				bundleID: '',
				toolSlug: out.name,
				toolVersion: '',
				toolType: out.type as unknown as ToolStoreChoiceType,
				autoExecute: false,
			} as ToolStoreChoice),

		isError: isError,
		errorMessage: isError ? primaryText : undefined,

		// Hydrate from the original call, if present.
		arguments: call?.arguments,
		webSearchToolCallItems: call?.webSearchToolCallItems,
	};
}

function getQuotedJSON(obj: any): string {
	return '```json\n' + JSON.stringify(obj, null, 2) + '\n```';
}
