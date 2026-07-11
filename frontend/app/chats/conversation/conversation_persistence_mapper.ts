import type {
	Conversation,
	ConversationMessage,
	StoreConversation,
	StoreConversationMessage,
} from '@/spec/conversation';
import type {
	InputUnion,
	OutputUnion,
	ReasoningContent,
	ToolCall,
	ToolOutput,
	UIToolCall,
	UIToolOutput,
	URLCitation,
} from '@/spec/inference';
import { ContentItemKind, InputKind, RoleEnum } from '@/spec/inference';
import type { ToolStoreChoice } from '@/spec/tool';

import {
	buildMCPToolSelectionMap,
	buildUIToolOutputFromToolOutput,
	deriveUIFieldsFromOutputUnion,
} from '@/chats/conversation/completion_helper';
import { collectToolCallsFromInputs, collectToolCallsFromOutputs } from '@/tools/lib/tool_call_utils';

function toStoreConversationMessage(message: ConversationMessage): StoreConversationMessage {
	const {
		uiContent: _uiContent,
		uiDebugDetails: _uiDebugDetails,
		uiReasoningContents: _uiReasoningContents,
		uiToolCalls: _uiToolCalls,
		uiToolOutputs: _uiToolOutputs,
		uiCitations: _uiCitations,
		...storeMessage
	} = message;

	return storeMessage;
}

export function toStoreConversation(conversation: Conversation): StoreConversation {
	return {
		...conversation,
		messages: conversation.messages.map(toStoreConversationMessage),
	};
}

interface ConversationHydrationContext {
	choiceMap: Map<string, ToolStoreChoice>;
	toolCallMap: Map<string, ToolCall>;
	mcpToolSelectionMap: ReturnType<typeof buildMCPToolSelectionMapFromMessages>;
}

function buildConversationHydrationContext(store: StoreConversation): ConversationHydrationContext {
	return {
		choiceMap: buildToolStoreChoiceMap(store.messages),
		toolCallMap: buildToolCallMap(store.messages),
		mcpToolSelectionMap: buildMCPToolSelectionMapFromMessages(store.messages),
	};
}

function hydrateConversationMessage(
	message: StoreConversationMessage,
	context: ConversationHydrationContext
): ConversationMessage {
	const { choiceMap, toolCallMap, mcpToolSelectionMap } = context;
	const role = message.role;

	let uiContent = '';
	let uiReasoningContents: ReasoningContent[] | undefined;
	let uiToolCalls: UIToolCall[] | undefined;
	let uiToolOutputs: UIToolOutput[] | undefined;
	let uiCitations: URLCitation[] | undefined;

	const outputs: OutputUnion[] | undefined = message.outputs;
	const inputs: InputUnion[] | undefined = message.inputs;

	if (role === RoleEnum.Assistant) {
		const derived = deriveUIFieldsFromOutputUnion(outputs, choiceMap, mcpToolSelectionMap);

		uiContent = derived.uiContent;
		uiReasoningContents = derived.uiReasoningContents;
		uiToolCalls = derived.uiToolCalls;
		uiCitations = derived.uiCitations;
		uiToolOutputs =
			derived.uiToolOutputs && derived.uiToolOutputs.length > 0
				? derived.uiToolOutputs
				: deriveUIToolOutputsFromInputUnion(inputs, choiceMap, toolCallMap, mcpToolSelectionMap);
	} else if (role === RoleEnum.User) {
		uiContent = deriveUIContentFromInputUnion(inputs);
		uiToolOutputs = deriveUIToolOutputsFromInputUnion(inputs, choiceMap, toolCallMap, mcpToolSelectionMap);
	}

	return {
		...(message as any),
		uiContent,
		uiReasoningContents,
		uiToolCalls,
		uiToolOutputs,
		uiCitations,
		uiDebugDetails: undefined,
	} as ConversationMessage;
}

const HYDRATION_MAIN_THREAD_BUDGET_MS = 8;

function getHydrationClock(): number {
	return typeof performance !== 'undefined' ? performance.now() : Date.now();
}

async function yieldHydrationToMainThread(): Promise<void> {
	const scheduler = (
		globalThis as typeof globalThis & {
			scheduler?: {
				yield?: () => Promise<void>;
			};
		}
	).scheduler;

	if (scheduler?.yield) {
		await scheduler.yield();
		return;
	}

	await new Promise<void>(resolve => {
		setTimeout(resolve, 0);
	});
}

export async function hydrateConversationAsync(
	store: StoreConversation,
	shouldContinue: () => boolean = () => true
): Promise<Conversation | undefined> {
	if (!shouldContinue()) {
		return undefined;
	}

	const context = buildConversationHydrationContext(store);
	const hydratedMessages: ConversationMessage[] = [];
	let sliceStartedAt = getHydrationClock();

	for (let index = 0; index < store.messages.length; index += 1) {
		if (!shouldContinue()) {
			return undefined;
		}

		hydratedMessages.push(hydrateConversationMessage(store.messages[index], context));

		if (index < store.messages.length - 1 && getHydrationClock() - sliceStartedAt >= HYDRATION_MAIN_THREAD_BUDGET_MS) {
			await yieldHydrationToMainThread();
			sliceStartedAt = getHydrationClock();
		}
	}

	return {
		...(store as any),
		messages: hydratedMessages,
	} as Conversation;
}

function buildMCPToolSelectionMapFromMessages(messages: StoreConversationMessage[]) {
	const syntheticContext = {
		servers: messages.flatMap(message => message.mcpContext?.servers ?? []),
	};

	const combined = buildMCPToolSelectionMap(syntheticContext) ?? new Map();

	for (const message of messages) {
		const fromDebug = buildMCPToolSelectionMap(undefined, message.debugDetails);
		for (const [key, selection] of fromDebug ?? []) {
			combined.set(key, selection);
		}
	}

	return combined.size > 0 ? combined : undefined;
}

function buildToolStoreChoiceMap(messages: StoreConversationMessage[]): Map<string, ToolStoreChoice> {
	const map = new Map<string, ToolStoreChoice>();

	for (const message of messages) {
		if (!message.toolStoreChoices) {
			continue;
		}

		for (const choice of message.toolStoreChoices) {
			map.set(choice.choiceID, choice);
		}
	}

	return map;
}

function buildToolCallMap(messages: StoreConversationMessage[]): Map<string, ToolCall> {
	let map = new Map<string, ToolCall>();

	for (const message of messages) {
		map = collectToolCallsFromInputs(message.inputs, map);
		map = collectToolCallsFromOutputs(message.outputs, map);
	}

	return map;
}

function deriveUIContentFromInputUnion(inputs?: InputUnion[]): string {
	if (!inputs || inputs.length === 0) {
		return '';
	}

	for (const input of inputs) {
		if (
			input.kind !== InputKind.InputMessage ||
			input.inputMessage?.role !== RoleEnum.User ||
			!input.inputMessage.contents
		) {
			continue;
		}

		for (const content of input.inputMessage.contents) {
			if (content.kind === ContentItemKind.Text && content.textItem?.text) {
				const text = content.textItem.text.trim();
				if (text) {
					return text;
				}
			}
		}
	}

	return '';
}

function deriveUIToolOutputsFromInputUnion(
	inputs: InputUnion[] | undefined,
	choiceMap: Map<string, ToolStoreChoice>,
	toolCallMap: Map<string, ToolCall>,
	mcpToolSelectionMap?: ReturnType<typeof buildMCPToolSelectionMap>
): UIToolOutput[] {
	if (!inputs || inputs.length === 0) {
		return [];
	}

	const uiOutputs: UIToolOutput[] = [];

	for (const input of inputs) {
		let out: ToolOutput | undefined;

		if (input.kind === InputKind.FunctionToolOutput && input.functionToolOutput) {
			out = input.functionToolOutput;
		} else if (input.kind === InputKind.CustomToolOutput && input.customToolOutput) {
			out = input.customToolOutput;
		} else if (input.kind === InputKind.WebSearchToolOutput && input.webSearchToolOutput) {
			out = input.webSearchToolOutput;
		} else {
			continue;
		}

		uiOutputs.push(buildUIToolOutputFromToolOutput(out, choiceMap, toolCallMap, mcpToolSelectionMap));
	}

	return uiOutputs;
}
