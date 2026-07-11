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
import type { MCPToolSelection } from '@/spec/mcp';
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

export async function toStoreConversationAsync(conversation: Conversation): Promise<StoreConversation> {
	const messages: StoreConversationMessage[] = [];
	let sliceStartedAt = getHydrationClock();

	for (let index = 0; index < conversation.messages.length; index += 1) {
		messages.push(toStoreConversationMessage(conversation.messages[index]));

		if (
			index < conversation.messages.length - 1 &&
			getHydrationClock() - sliceStartedAt >= HYDRATION_MAIN_THREAD_BUDGET_MS
		) {
			await yieldHydrationToMainThread();
			sliceStartedAt = getHydrationClock();
		}
	}

	return {
		...conversation,
		messages,
	};
}

interface ConversationHydrationContext {
	choiceMap: Map<string, ToolStoreChoice>;
	toolCallMap: Map<string, ToolCall>;
	mcpToolSelectionMap: ReturnType<typeof buildMCPToolSelectionMap>;
}

async function buildConversationHydrationContext(
	store: StoreConversation,
	shouldContinue: () => boolean
): Promise<ConversationHydrationContext | undefined> {
	const choiceMap = new Map<string, ToolStoreChoice>();
	let toolCallMap = new Map<string, ToolCall>();
	const contextMCPSelections = new Map<string, MCPToolSelection>();
	const debugMCPSelections = new Map<string, MCPToolSelection>();
	let sliceStartedAt = getHydrationClock();

	for (let index = 0; index < store.messages.length; index += 1) {
		if (!shouldContinue()) {
			return undefined;
		}

		const message = store.messages[index];

		for (const choice of message.toolStoreChoices ?? []) {
			choiceMap.set(choice.choiceID, choice);
		}

		toolCallMap = collectToolCallsFromInputs(message.inputs, toolCallMap);
		toolCallMap = collectToolCallsFromOutputs(message.outputs, toolCallMap);

		for (const [key, selection] of buildMCPToolSelectionMap(message.mcpContext) ?? []) {
			contextMCPSelections.set(key, selection);
		}
		for (const [key, selection] of buildMCPToolSelectionMap(undefined, message.debugDetails) ?? []) {
			debugMCPSelections.set(key, selection);
		}

		if (index < store.messages.length - 1 && getHydrationClock() - sliceStartedAt >= HYDRATION_MAIN_THREAD_BUDGET_MS) {
			await yieldHydrationToMainThread();
			sliceStartedAt = getHydrationClock();
		}
	}

	for (const [key, selection] of debugMCPSelections) {
		contextMCPSelections.set(key, selection);
	}

	return {
		choiceMap,
		toolCallMap,
		mcpToolSelectionMap: contextMCPSelections.size > 0 ? contextMCPSelections : undefined,
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

	const context = await buildConversationHydrationContext(store, shouldContinue);
	if (!context || !shouldContinue()) {
		return undefined;
	}

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
