import type {
	Conversation,
	ConversationMessage,
	StoreConversation,
	StoreConversationMessage,
} from '@/spec/conversation';
import {
	ContentItemKind,
	InputKind,
	type InputUnion,
	type OutputUnion,
	type ReasoningContent,
	RoleEnum,
	type ToolCall,
	type ToolOutput,
	type UIToolCall,
	type UIToolOutput,
	type URLCitation,
} from '@/spec/inference';
import { type ToolStoreChoice } from '@/spec/tool';

import {
	buildUIToolOutputFromToolOutput,
	deriveUIFieldsFromOutputUnion,
	getDebugDetailsMarkdown,
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

export function hydrateConversation(store: StoreConversation): Conversation {
	const choiceMap = buildToolStoreChoiceMap(store.messages);
	const toolCallMap = buildToolCallMap(store.messages);

	const hydratedMessages: ConversationMessage[] = store.messages.map(message => {
		const role = message.role;

		let uiContent = '';
		let uiReasoningContents: ReasoningContent[] | undefined;
		let uiToolCalls: UIToolCall[] | undefined;
		let uiToolOutputs: UIToolOutput[] | undefined;
		let uiCitations: URLCitation[] | undefined;

		const outputs: OutputUnion[] | undefined = message.outputs;
		const inputs: InputUnion[] | undefined = message.inputs;

		if (role === RoleEnum.Assistant) {
			const derived = deriveUIFieldsFromOutputUnion(outputs, choiceMap);

			uiContent = derived.uiContent;
			uiReasoningContents = derived.uiReasoningContents;
			uiToolCalls = derived.uiToolCalls;
			uiCitations = derived.uiCitations;
			uiToolOutputs =
				derived.uiToolOutputs && derived.uiToolOutputs.length > 0
					? derived.uiToolOutputs
					: deriveUIToolOutputsFromInputUnion(inputs, choiceMap, toolCallMap);
		} else if (role === RoleEnum.User) {
			uiContent = deriveUIContentFromInputUnion(inputs);
			uiToolOutputs = deriveUIToolOutputsFromInputUnion(inputs, choiceMap, toolCallMap);
		}

		const uiDebugDetails = getDebugDetailsMarkdown(message.debugDetails, message.error);

		return {
			...(message as any),
			uiContent,
			uiReasoningContents,
			uiToolCalls,
			uiToolOutputs,
			uiCitations,
			uiDebugDetails,
		} as ConversationMessage;
	});

	return {
		...(store as any),
		messages: hydratedMessages,
	} as Conversation;
}

function buildToolStoreChoiceMap(messages: StoreConversationMessage[]): Map<string, ToolStoreChoice> {
	const map = new Map<string, ToolStoreChoice>();

	for (const message of messages) {
		if (!message.toolStoreChoices) continue;

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
	if (!inputs || inputs.length === 0) return '';

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
				if (text) return text;
			}
		}
	}

	return '';
}

function deriveUIToolOutputsFromInputUnion(
	inputs: InputUnion[] | undefined,
	choiceMap: Map<string, ToolStoreChoice>,
	toolCallMap: Map<string, ToolCall>
): UIToolOutput[] {
	if (!inputs || inputs.length === 0) return [];

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

		uiOutputs.push(buildUIToolOutputFromToolOutput(out, choiceMap, toolCallMap));
	}

	return uiOutputs;
}
