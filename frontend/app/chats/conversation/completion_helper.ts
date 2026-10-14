import type {
	OutputUnion,
	ReasoningContent,
	ToolCall,
	ToolOutput,
	UIToolCall,
	UIToolOutput,
	URLCitation,
	WebSearchToolOutputItemUnion,
} from '@/spec/inference';
import type { MCPToolSelection } from '@/spec/session_mcp';
import type { ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import { CitationKind, ContentItemKind, OutputKind, UIToolCallStatus } from '@/spec/inference';
import { ToolImplType, ToolStoreChoiceType } from '@/spec/tool';

import { buildJSONOrTextCodeBlock } from '@/lib/jsonschema_utils';

import { isSkillsToolName } from '@/skills/lib/skill_identity_utils';
import { collectToolCallsFromOutputs } from '@/tools/lib/tool_call_utils';
import { toolSelectionFromChoice } from '@/tools/lib/tool_choice_utils';
import {
	extractPrimaryTextFromToolOutputs,
	formatToolOutputSummary,
	mapToolOutputItemsToToolOutputs,
} from '@/tools/lib/tool_output_utils';

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

export function deriveUIFieldsFromOutputUnion(
	outputs: OutputUnion[] | undefined,
	choiceMap: Map<string, ToolStoreChoice>,
	mcpToolSelectionMap?: Map<string, MCPToolSelection>,
	selectionIssues?: Map<string, ToolSelectionIssue>
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
				const uiFunctionToolCall = deriveUIToolCallFromToolCall(
					o.functionToolCall,
					choiceMap,
					mcpToolSelectionMap,
					selectionIssues
				);
				if (uiFunctionToolCall) {
					toolCalls.push(uiFunctionToolCall);
				}
				break;
			}
			case OutputKind.CustomToolCall: {
				const uiCustomToolCall = deriveUIToolCallFromToolCall(
					o.customToolCall,
					choiceMap,
					mcpToolSelectionMap,
					selectionIssues
				);
				if (uiCustomToolCall) {
					toolCalls.push(uiCustomToolCall);
				}
				break;
			}
			case OutputKind.WebSearchToolCall: {
				const uiWebsearchToolCall = deriveUIToolCallFromToolCall(
					o.webSearchToolCall,
					choiceMap,
					mcpToolSelectionMap,
					selectionIssues
				);
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
					toolOutputs.push(
						buildUIToolOutputFromToolOutput(out, choiceMap, toolCallMap, mcpToolSelectionMap, selectionIssues)
					);
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
	mcpToolSelectionMap?: Map<string, MCPToolSelection>,
	selectionIssues?: Map<string, ToolSelectionIssue>
): UIToolCall | undefined {
	if (!toolCall) {
		return undefined;
	}

	// NOTE: runtime-injected tools (e.g. skills-*) are not in toolStoreChoices,
	// so they will not be present in this map. Do NOT drop the tool call.
	const mcpToolSelection = findMCPToolSelectionForToolLike(toolCall, mcpToolSelectionMap);
	const choiceID = toolCall.choiceID || mcpToolSelection?.choiceID || toolCall.callID || toolCall.name;
	const toolStoreChoice = toolCall.choiceID ? choiceMap.get(toolCall.choiceID) : undefined;

	const type = toolCall.type as unknown as ToolStoreChoiceType;
	const issue = selectionIssues?.get(toolCall.choiceID);
	const providerManaged =
		type === ToolStoreChoiceType.WebSearch || toolStoreChoice?.implementationKind === ToolImplType.SDK;
	const unresolvedClientTool =
		!providerManaged && !toolStoreChoice && !mcpToolSelection && !isSkillsToolName(toolCall.name);
	const status: UIToolCall['status'] = providerManaged
		? UIToolCallStatus.Succeeded
		: unresolvedClientTool
			? UIToolCallStatus.Failed
			: UIToolCallStatus.Pending;

	return {
		id: toolCall.id || toolCall.callID || `${toolCall.name}:${choiceID}`,
		callID: toolCall.callID,
		name: toolCall.name,
		arguments: toolCall.arguments ?? '',
		webSearchToolCallItems: toolCall.webSearchToolCallItems,
		type: type,
		choiceID,
		// The LLM would consider the status of tool call as done as soon as it is in output,
		// for us the call is pending here and then it will run and move to final status.
		status: status,
		toolStoreChoice,
		toolSelection: toolStoreChoice ? toolSelectionFromChoice(toolStoreChoice) : issue?.selection,
		errorMessage: unresolvedClientTool
			? issue?.message || 'This call cannot be matched to an available selected Tool.'
			: undefined,
		mcpToolSelection,
	};
}

function buildUIToolOutputFromToolOutput(
	out: ToolOutput,
	choiceMap: Map<string, ToolStoreChoice>,
	toolCallMap?: Map<string, ToolCall>,
	mcpToolSelectionMap?: Map<string, MCPToolSelection>,
	selectionIssues?: Map<string, ToolSelectionIssue>
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
		toolSelection: toolStoreChoice
			? toolSelectionFromChoice(toolStoreChoice)
			: selectionIssues?.get(out.choiceID)?.selection,
		mcpToolSelection,
		mcpApp,
		isError: isError,
		errorMessage: isError ? primaryText : undefined,

		// Hydrate from the original call, if present.
		arguments: call?.arguments,
		webSearchToolCallItems: call?.webSearchToolCallItems,
	};
}
