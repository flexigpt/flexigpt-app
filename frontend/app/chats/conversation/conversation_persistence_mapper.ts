import type {
	Conversation,
	ConversationMessage,
	StoreConversation,
	StoreConversationMessage,
} from '@/spec/conversation';
import type { InputUnion, ToolCall, ToolOutput, UIToolOutput } from '@/spec/inference';
import type { MCPToolSelection } from '@/spec/mcp';
import type { ToolSelection, ToolSelectionIssue, ToolStoreChoice } from '@/spec/tool';
import { ContentItemKind, InputKind, RoleEnum } from '@/spec/inference';

import type { ToolSelectionHydration } from '@/tools/lib/tool_selection_hydration';
import {
	buildMCPToolSelectionMap,
	buildUIToolOutputFromToolOutput,
	deriveUIFieldsFromOutputUnion,
} from '@/chats/conversation/completion_helper';
import { collectToolCallsFromInputs, collectToolCallsFromOutputs } from '@/tools/lib/tool_call_utils';
import { toolSelectionFromChoice } from '@/tools/lib/tool_choice_utils';
import { hydrateToolSelectionsForUI } from '@/tools/lib/tool_selection_hydration';

function toStoreConversationMessage(message: ConversationMessage): StoreConversationMessage {
	const {
		uiContent: _uiContent,
		uiDebugDetails: _uiDebugDetails,
		uiReasoningContents: _uiReasoningContents,
		uiToolCalls: _uiToolCalls,
		uiToolOutputs: _uiToolOutputs,
		uiToolChoices: _uiToolChoices,
		uiToolSelectionIssues: _uiToolSelectionIssues,
		uiCitations: _uiCitations,
		...storeMessage
	} = message;

	return {
		...storeMessage,
		toolSelections: storeMessage.toolSelections?.map(selection => toolSelectionFromChoice(selection)),
	};
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
	issueMap: Map<string, ToolSelectionIssue>;
	toolCallMap: Map<string, ToolCall>;
	mcpToolSelectionMap: Map<string, MCPToolSelection>;
}

function installMCPMappings(context: ConversationHydrationContext, message: StoreConversationMessage): void {
	for (const [key, selection] of buildMCPToolSelectionMap(
		message.mcpContext,
		message.debugDetails,
		message.mcpToolMappings
	) ?? []) {
		context.mcpToolSelectionMap.set(key, selection);
	}
}

function installToolSelections(
	context: ConversationHydrationContext,
	message: StoreConversationMessage,
	hydratedBySelection: WeakMap<ToolSelection, ToolSelectionHydration>
): {
	choices: ToolStoreChoice[];
	issues: ToolSelectionIssue[];
} {
	const choices: ToolStoreChoice[] = [];
	const issues: ToolSelectionIssue[] = [];

	for (const selection of message.toolSelections ?? []) {
		const hydrated = hydratedBySelection.get(selection);
		if (!hydrated) {
			throw new Error('A stored Tool selection was not included in conversation hydration.');
		}

		if (hydrated.choice) {
			choices.push(hydrated.choice);
			context.choiceMap.set(selection.choiceID, hydrated.choice);
			context.issueMap.delete(selection.choiceID);
			continue;
		}

		const issue = hydrated.issue ?? {
			selection,
			message: `Tool "${selection.target.name}" is unavailable.`,
		};
		issues.push(issue);
		context.choiceMap.delete(selection.choiceID);
		context.issueMap.set(selection.choiceID, issue);
	}

	return { choices, issues };
}

function hydrateConversationMessage(
	message: StoreConversationMessage,
	context: ConversationHydrationContext,
	hydratedBySelection: WeakMap<ToolSelection, ToolSelectionHydration>
): ConversationMessage {
	/*
	 * Tool outputs on a user message answer calls from an earlier assistant
	 * turn. Resolve those before this message installs a potentially newer
	 * selection with a reused choiceID.
	 */
	context.toolCallMap = collectToolCallsFromInputs(message.inputs, context.toolCallMap);
	const inputToolOutputs = deriveUIToolOutputsFromInputUnion(
		message.inputs,
		context.choiceMap,
		context.toolCallMap,
		context.mcpToolSelectionMap,
		context.issueMap
	);

	if (message.role !== RoleEnum.User) {
		installMCPMappings(context, message);
	}

	context.toolCallMap = collectToolCallsFromOutputs(message.outputs, context.toolCallMap);

	const { choices, issues } = installToolSelections(context, message, hydratedBySelection);

	if (message.role === RoleEnum.User) {
		installMCPMappings(context, message);
		return {
			...message,
			uiContent: deriveUIContentFromInputUnion(message.inputs, message.id),
			uiToolOutputs: inputToolOutputs.length > 0 ? inputToolOutputs : undefined,
			uiToolChoices: choices.length > 0 ? choices : undefined,
			uiToolSelectionIssues: issues.length > 0 ? issues : undefined,
			uiDebugDetails: undefined,
		};
	}

	if (message.role === RoleEnum.Assistant) {
		const derived = deriveUIFieldsFromOutputUnion(
			message.outputs,
			context.choiceMap,
			context.mcpToolSelectionMap,
			context.issueMap
		);

		return {
			...message,
			...derived,
			uiToolOutputs:
				derived.uiToolOutputs && derived.uiToolOutputs.length > 0
					? derived.uiToolOutputs
					: inputToolOutputs.length > 0
						? inputToolOutputs
						: undefined,
			uiToolChoices: choices.length > 0 ? choices : undefined,
			uiToolSelectionIssues: issues.length > 0 ? issues : undefined,
			uiDebugDetails: undefined,
		};
	}

	return {
		...message,
		uiContent: '',
		uiToolOutputs: inputToolOutputs.length > 0 ? inputToolOutputs : undefined,
		uiToolChoices: choices.length > 0 ? choices : undefined,
		uiToolSelectionIssues: issues.length > 0 ? issues : undefined,
		uiDebugDetails: undefined,
	};
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

	const hydratedSelections = await hydrateToolSelectionsForUI(
		store.messages.flatMap(message => message.toolSelections ?? []),
		shouldContinue
	);
	if (!hydratedSelections || !shouldContinue()) {
		return undefined;
	}

	const hydratedBySelection = new WeakMap<ToolSelection, ToolSelectionHydration>();
	for (const hydrated of hydratedSelections) {
		hydratedBySelection.set(hydrated.selection, hydrated);
	}

	const context: ConversationHydrationContext = {
		choiceMap: new Map(),
		issueMap: new Map(),
		toolCallMap: new Map(),
		mcpToolSelectionMap: new Map(),
	};

	const messages: ConversationMessage[] = [];
	let sliceStartedAt = getHydrationClock();

	for (let index = 0; index < store.messages.length; index += 1) {
		if (!shouldContinue()) {
			return undefined;
		}

		messages.push(hydrateConversationMessage(store.messages[index], context, hydratedBySelection));

		if (index < store.messages.length - 1 && getHydrationClock() - sliceStartedAt >= HYDRATION_MAIN_THREAD_BUDGET_MS) {
			await yieldHydrationToMainThread();
			sliceStartedAt = getHydrationClock();
		}
	}

	return {
		...store,
		messages,
	};
}

function getUserInputMessageText(input: InputUnion): string {
	for (const content of input.inputMessage?.contents ?? []) {
		if (content.kind !== ContentItemKind.Text || !content.textItem?.text) {
			continue;
		}

		const text = content.textItem.text.trim();
		if (text) {
			return text;
		}
	}

	return '';
}

function isGeneratedCurrentContextInput(input: InputUnion): boolean {
	const id = input.inputMessage?.id;
	return id === 'mcp-context' || id?.startsWith('workspace-context:') === true;
}

function deriveUIContentFromInputUnion(inputs?: InputUnion[], expectedInputMessageID?: string): string {
	const userInputs = (inputs ?? []).filter(
		input => input.kind === InputKind.InputMessage && input.inputMessage?.role === RoleEnum.User
	);

	if (expectedInputMessageID) {
		const original = userInputs.find(input => input.inputMessage?.id === expectedInputMessageID);
		if (original) {
			return getUserInputMessageText(original);
		}
	}

	for (const input of userInputs.toReversed()) {
		if (isGeneratedCurrentContextInput(input)) {
			continue;
		}

		const text = getUserInputMessageText(input);
		if (text) {
			return text;
		}
	}

	return '';
}

function deriveUIToolOutputsFromInputUnion(
	inputs: InputUnion[] | undefined,
	choiceMap: Map<string, ToolStoreChoice>,
	toolCallMap: Map<string, ToolCall>,
	mcpToolSelectionMap: Map<string, MCPToolSelection>,
	issueMap: Map<string, ToolSelectionIssue>
): UIToolOutput[] {
	const uiOutputs: UIToolOutput[] = [];

	for (const input of inputs ?? []) {
		let output: ToolOutput | undefined;

		switch (input.kind) {
			case InputKind.FunctionToolOutput:
				output = input.functionToolOutput;
				break;
			case InputKind.CustomToolOutput:
				output = input.customToolOutput;
				break;
			case InputKind.WebSearchToolOutput:
				output = input.webSearchToolOutput;
				break;
			default:
				break;
		}

		if (output) {
			uiOutputs.push(buildUIToolOutputFromToolOutput(output, choiceMap, toolCallMap, mcpToolSelectionMap, issueMap));
		}
	}

	return uiOutputs;
}
