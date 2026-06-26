import type { ConversationMessage } from '@/spec/conversation';
import type {
	CompletionResponseBody,
	InferenceError,
	InferenceUsage,
	ModelParam,
	OutputUnion,
	ProviderName,
	ReasoningContent,
	ToolCall,
	ToolOutput,
	UIToolCall,
	UIToolOutput,
	URLCitation,
	WebSearchToolOutputItemUnion,
} from '@/spec/inference';
import { CitationKind, ContentItemKind, OutputKind, RoleEnum, Status } from '@/spec/inference';
import type { MCPConversationContext, MCPProviderToolMapping, MCPToolSelection } from '@/spec/mcp';
import type { ModelPresetID } from '@/spec/modelpreset';
import type { ToolStoreChoice } from '@/spec/tool';
import { ToolStoreChoiceType } from '@/spec/tool';

import { buildJSONOrTextCodeBlock } from '@/lib/jsonschema_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { aggregateAPI } from '@/apis/baseapi';

import { collectToolCallsFromOutputs } from '@/tools/lib/tool_call_utils';
import {
	extractPrimaryTextFromToolOutputs,
	formatToolOutputSummary,
	mapToolOutputItemsToToolOutputs,
} from '@/tools/lib/tool_output_utils';

export async function HandleCompletion(
	provider: ProviderName,
	modelPresetID: ModelPresetID,
	modelParams: ModelParam,
	currentUserMsg: ConversationMessage,
	history: ConversationMessage[],
	toolStoreChoices: ToolStoreChoice[] | undefined,
	mcpContext: MCPConversationContext | undefined,
	assistantPlaceholder: ConversationMessage,
	skillSessionID?: string,
	requestId?: string,
	signal?: AbortSignal,
	onStreamTextData?: (textData: string) => void,
	onStreamThinkingData?: (thinkingData: string) => void
): Promise<{
	responseMessage: ConversationMessage | undefined;
	rawResponse?: CompletionResponseBody;
}> {
	// console.log('history to completion', JSON.stringify(history, null, 2));
	const choiceMap = new Map<string, ToolStoreChoice>((toolStoreChoices ?? []).map(choice => [choice.choiceID, choice]));

	const resp = await aggregateAPI.fetchCompletion(
		provider,
		modelPresetID,
		modelParams,
		currentUserMsg,
		history,
		toolStoreChoices,
		mcpContext,
		skillSessionID,
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
		const assistantMsg = buildAssistantMessageFromResponse(
			assistantPlaceholder.id,
			modelParams,
			resp,
			choiceMap,
			mcpContext
		);
		return { responseMessage: assistantMsg, rawResponse: resp };
	}

	// Error with no outputs at all -> fall back to existing "error stub".
	return getErrorStub(modelParams, assistantPlaceholder, resp, undefined);
	// Important:
	// Transport/runtime failures must be finalized by the caller because only
	// the caller has access to the live stream buffers. If we swallow the
	// error here, already-streamed text/thinking gets lost. So throw error always.
}

function getErrorStub(
	modelParams: ModelParam,
	assistantPlaceholder: ConversationMessage,
	rawResponse: CompletionResponseBody | undefined,
	errorObj: any
) {
	let error: InferenceError;
	if (rawResponse?.inferenceResponse && rawResponse.inferenceResponse.error) {
		error = rawResponse.inferenceResponse.error;
	} else {
		let msg: string;
		try {
			msg = JSON.stringify(errorObj, null, 2);
		} catch {
			msg = String(errorObj);
		}
		error = {
			code: 'unknown',
			message: msg,
		} as InferenceError;
	}

	const errMsg = typeof error?.message === 'string' ? error.message.trim() : '';
	const outText = errMsg ? `> Error: ${errMsg}` : '> Error: Got error in API processing.';
	const outputs: OutputUnion[] = [
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
	let detailsMarkdown = '';
	let debugDetails: any;
	if (rawResponse?.inferenceResponse) {
		detailsMarkdown =
			getDebugDetailsMarkdown(rawResponse.inferenceResponse.debugDetails, rawResponse.inferenceResponse.error) ?? '';
		debugDetails = rawResponse.inferenceResponse.debugDetails;
	}

	if (errorObj !== undefined && errorObj !== null) {
		const errorBlock = buildJSONOrTextCodeBlock(errorObj) ?? undefined;
		if (errorBlock) {
			detailsMarkdown = detailsMarkdown
				? `${detailsMarkdown}\n\n### Error\n\n${errorBlock}`
				: `### Error\n\n${errorBlock}`;
		}
	}

	const errorMessage: ConversationMessage = {
		...assistantPlaceholder,
		modelParam: modelParams,
		status: Status.Failed,
		error,
		uiContent: outText,
		outputs,
		debugDetails,
		uiDebugDetails: detailsMarkdown,
	};

	return { responseMessage: errorMessage, rawResponse };
}

export function getDebugDetailsMarkdown(debugObj?: any, errorObj?: any): string | undefined {
	const parts: string[] = [];

	const pushDetailsBlock = (title: string, value: unknown) => {
		const block = buildJSONOrTextCodeBlock(value) ?? undefined;
		if (!block) {
			return;
		}
		parts.push(title, block);
	};

	// 1. Error object should always be first, if present
	if (errorObj !== undefined && errorObj !== null) {
		pushDetailsBlock('### Error', errorObj);
	}

	// 2. Handle debug details
	if (debugObj !== undefined && debugObj !== null) {
		const isMap = debugObj instanceof Map;
		const isObjectLike = typeof debugObj === 'object' && !isMap;

		const record: Record<string, unknown> | undefined = isObjectLike
			? (debugObj as Record<string, unknown>)
			: undefined;

		const hasKey = (key: string): boolean => {
			if (isMap) {
				return (debugObj as Map<string, unknown>).has(key);
			}
			if (record) {
				return key in record;
			}
			return false;
		};

		const getValue = (key: string): unknown => {
			if (isMap) {
				return (debugObj as Map<string, unknown>).get(key);
			}
			if (record) {
				return record[key];
			}
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
				pushDetailsBlock('### Error details', getValue('errorDetails'));
			}

			if (hasRequest) {
				pushDetailsBlock('### Request debug details', getValue('requestDetails'));
			}

			if (hasResponse) {
				pushDetailsBlock('### Response debug details', getValue('responseDetails'));
			}

			if (hasProvider) {
				pushDetailsBlock('### Provider response debug details', getValue('providerResponse'));
			}
		} else {
			// No special keys: fallback to original behavior (one block)
			pushDetailsBlock('### Debug details', debugObj);
		}
	}

	if (parts.length === 0) {
		return undefined;
	}

	return parts.join('\n\n');
}

export function buildMCPToolSelectionMap(
	mcpContext?: MCPConversationContext,
	debugDetails?: any
): Map<string, MCPToolSelection> | undefined {
	const map = new Map<string, MCPToolSelection>();

	for (const server of mcpContext?.servers ?? []) {
		for (const tool of server.selectedTools ?? []) {
			const normalized: MCPToolSelection = {
				...tool,
				bundleID: tool.bundleID || server.bundleID,
				serverID: tool.serverID || server.serverID,
			};

			addMCPToolSelectionToMap(map, normalized);
		}
	}
	for (const mapping of extractMCPToolMappings(debugDetails)) {
		const selection = mappingToMCPToolSelection(mapping);
		if (selection) {
			addMCPToolSelectionToMap(map, selection);
		}
	}

	return map.size > 0 ? map : undefined;
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
	if (!value || typeof value !== 'object' || Array.isArray(value)) {
		return undefined;
	}
	return value as Record<string, unknown>;
}

function mcpSelectionKeys(selection: MCPToolSelection): string[] {
	const keys: string[] = [];

	if (selection.choiceID) {
		keys.push(`choice:${selection.choiceID}`);
	}
	if (selection.providerToolName) {
		keys.push(`name:${selection.providerToolName}`);
	}
	if (selection.toolName) {
		keys.push(`name:${selection.toolName}`);
	}

	return keys;
}

function addMCPToolSelectionToMap(map: Map<string, MCPToolSelection>, selection: MCPToolSelection) {
	for (const key of mcpSelectionKeys(selection)) {
		map.set(key, selection);
	}
}
function mappingToMCPToolSelection(mapping: unknown): MCPToolSelection | undefined {
	const obj = asRecord(mapping);
	if (!obj) {
		return undefined;
	}
	const bundleID = obj.bundleID;
	const serverID = obj.serverID;
	const toolName = obj.toolName;
	const providerToolName = obj.providerToolName;
	const choiceID = obj.choiceID;
	const toolDigest = obj.toolDigest;
	const appResourceUri = obj.appResourceUri;
	const visibility = obj.visibility;
	if (typeof bundleID !== 'string' || !bundleID) {
		return undefined;
	}
	if (typeof serverID !== 'string' || !serverID) {
		return undefined;
	}
	if (typeof toolName !== 'string' || !toolName) {
		return undefined;
	}
	return {
		bundleID,
		serverID,
		toolName,
		providerToolName: typeof providerToolName === 'string' ? providerToolName : undefined,
		choiceID: typeof choiceID === 'string' ? choiceID : undefined,
		digest: typeof toolDigest === 'string' ? toolDigest : undefined,
		appResourceUri: typeof appResourceUri === 'string' ? appResourceUri : undefined,
		visibility: Array.isArray(visibility) ? visibility.filter((v): v is string => typeof v === 'string') : undefined,
	};
}

function extractMCPToolMappings(debugDetails?: any): MCPProviderToolMapping[] {
	const root = asRecord(debugDetails);
	if (!root) {
		return [];
	}
	const mcpDebug = asRecord(root.mcp) ?? root;
	const rawMappings = mcpDebug.toolMappings;
	if (!Array.isArray(rawMappings)) {
		return [];
	}
	return rawMappings.filter(mapping => mappingToMCPToolSelection(mapping)) as MCPProviderToolMapping[];
}

function findMCPToolSelectionForToolLike(
	value: { choiceID?: string; name?: string },
	mcpToolSelectionMap?: Map<string, MCPToolSelection>
): MCPToolSelection | undefined {
	if (!mcpToolSelectionMap) {
		return undefined;
	}

	if (value.choiceID) {
		const byChoice = mcpToolSelectionMap.get(`choice:${value.choiceID}`);
		if (byChoice) {
			return byChoice;
		}
	}

	if (value.name) {
		const byName = mcpToolSelectionMap.get(`name:${value.name}`);
		if (byName) {
			return byName;
		}
	}

	return undefined;
}

function buildAssistantMessageFromResponse(
	baseId: string,
	modelParams: ModelParam,
	resp: CompletionResponseBody,
	choiceMap: Map<string, ToolStoreChoice>,
	mcpContext?: MCPConversationContext
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
		choiceMap,
		buildMCPToolSelectionMap(mcpContext, inf.debugDetails)
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
	choiceMap: Map<string, ToolStoreChoice>,
	mcpToolSelectionMap?: Map<string, MCPToolSelection>
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
				if (!msg || !msg.contents) {
					break;
				}
				for (const c of msg.contents) {
					if (c.kind === ContentItemKind.Text && c.textItem) {
						const raw = c.textItem?.text;
						// Preserve provider text exactly as produced so the final
						// message matches the streamed text and does not "jump" on completion.
						if (typeof raw === 'string' && raw.length > 0) {
							textParts.push(raw);
						}

						const itemCitations = c.textItem.citations;

						if (itemCitations && itemCitations.length > 0) {
							for (const cit of itemCitations) {
								if (cit.kind !== CitationKind.URL || !cit.urlCitation?.url) {
									continue;
								}
								const u = cit.urlCitation;
								const key = `${u.url}|${u.startIndex ?? ''}|${u.endIndex ?? ''}|${u.title ?? ''}`;
								if (seenCitationKeys.has(key)) {
									continue;
								}
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
				const uiFunctionToolCall = deriveUIToolCallFromToolCall(o.functionToolCall, choiceMap, mcpToolSelectionMap);
				if (uiFunctionToolCall) {
					toolCalls.push(uiFunctionToolCall);
				}
				break;
			}
			case OutputKind.CustomToolCall: {
				const uiCustomToolCall = deriveUIToolCallFromToolCall(o.customToolCall, choiceMap, mcpToolSelectionMap);
				if (uiCustomToolCall) {
					toolCalls.push(uiCustomToolCall);
				}
				break;
			}
			case OutputKind.WebSearchToolCall: {
				const uiWebsearchToolCall = deriveUIToolCallFromToolCall(o.webSearchToolCall, choiceMap, mcpToolSelectionMap);
				if (uiWebsearchToolCall) {
					toolCalls.push(uiWebsearchToolCall);
				}
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
					toolOutputs.push(buildUIToolOutputFromToolOutput(out, choiceMap, toolCallMap, mcpToolSelectionMap));
				}
				break;
			}
		}
	}

	const content = textParts.join('');

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
	choiceMap: Map<string, ToolStoreChoice>,
	mcpToolSelectionMap?: Map<string, MCPToolSelection>
): UIToolCall | undefined {
	if (!toolCall) {
		return undefined;
	}

	const choiceID = toolCall.choiceID;
	if (!choiceID) {
		return undefined;
	}

	// NOTE: runtime-injected tools (e.g. skills-*) are not in toolStoreChoices,
	// so they will not be present in this map. Do NOT drop the tool call.
	const toolStoreChoice = choiceMap.get(choiceID); // ToolStoreChoice | undefined
	const mcpToolSelection = findMCPToolSelectionForToolLike(toolCall, mcpToolSelectionMap);

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
		mcpToolSelection,
	};
}

export function buildUIToolOutputFromToolOutput(
	out: ToolOutput,
	choiceMap: Map<string, ToolStoreChoice>,
	toolCallMap?: Map<string, ToolCall>,
	mcpToolSelectionMap?: Map<string, MCPToolSelection>
): UIToolOutput {
	const isError = out.isError;
	const toolStoreChoice = choiceMap.get(out.choiceID);
	const call = toolCallMap?.get(out.callID);
	const mcpToolSelection = call
		? findMCPToolSelectionForToolLike(call, mcpToolSelectionMap)
		: findMCPToolSelectionForToolLike(out, mcpToolSelectionMap);
	const mcpApp = mcpToolSelection?.appResourceUri ? { resourceUri: mcpToolSelection.appResourceUri } : undefined;
	const summaryBase = formatToolOutputSummary(out.name);

	const toolOutputs = mapToolOutputItemsToToolOutputs(out.contents);

	const primaryText = extractPrimaryTextFromToolOutputs(toolOutputs);
	const summary = isError && primaryText ? `Tool Error: ${primaryText.split('\n')[0].slice(0, 80)}` : summaryBase;

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
		toolOutputs: toolOutputs,
		webSearchToolOutputItems: webSearchOutputs,

		toolStoreChoice,
		mcpToolSelection,
		mcpApp,
		isError: isError,
		errorMessage: isError ? primaryText : undefined,

		// Hydrate from the original call, if present.
		arguments: call?.arguments,
		webSearchToolCallItems: call?.webSearchToolCallItems,
	};
}
