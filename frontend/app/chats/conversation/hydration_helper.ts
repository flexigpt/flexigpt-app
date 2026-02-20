// Adjust this import path to wherever your Attachment type lives
import type { Attachment } from '@/spec/attachment';
import {
	type Conversation,
	CONVERSATION_SCHEMA_VERSION,
	type ConversationMessage,
	type StoreConversation,
	type StoreConversationMessage,
} from '@/spec/conversation';
import {
	ContentItemKind,
	InputKind,
	type InputOutputContent,
	type InputOutputContentItemUnion,
	type InputUnion,
	type OutputUnion,
	type ReasoningContent,
	RoleEnum,
	Status,
	type ToolCall,
	type ToolOutput,
	ToolType,
	type UIToolCall,
	type UIToolOutput,
	type URLCitation,
} from '@/spec/inference';
import { type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { generateTitle } from '@/lib/title_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { uiAttachmentToConversation } from '@/chats/attachments/attachment_editor_utils';
import {
	buildUIToolOutputFromToolOutput,
	deriveUIFieldsFromOutputUnion,
	getDebugDetailsMarkdown,
} from '@/chats/conversation/completion_helper';
import type { EditorSubmitPayload } from '@/chats/inputarea/input_editor_utils';
import {
	collectToolCallsFromInputs,
	collectToolCallsFromOutputs,
	mapToolOutputsToToolOutputItems,
} from '@/chats/tools/tool_editor_utils';

export function initConversation(title = 'New Conversation'): Conversation {
	return {
		schemaVersion: CONVERSATION_SCHEMA_VERSION,
		id: getUUIDv7(),
		title: generateTitle(title).title,
		createdAt: new Date(),
		modifiedAt: new Date(),
		messages: [],
	};
}

export function initConversationMessage(role: RoleEnum): ConversationMessage {
	const d = new Date();
	return {
		id: getUUIDv7(),
		createdAt: d,
		role: role,
		status: Status.None,
		uiContent: '',
	};
}

export function deriveConversationToolsFromMessages(messages: ConversationMessage[]): ToolStoreChoice[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const m = messages[i];
		if (m.role === RoleEnum.User && m.toolStoreChoices && m.toolStoreChoices.length > 0) {
			return m.toolStoreChoices.filter(c => c.toolType !== ToolStoreChoiceType.WebSearch);
		}
	}
	return [];
}

export function deriveWebSearchChoiceFromMessages(messages: ConversationMessage[]): ToolStoreChoice[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const m = messages[i];
		if (m.role === RoleEnum.User && m.toolStoreChoices && m.toolStoreChoices.length > 0) {
			return m.toolStoreChoices.filter(c => c.toolType === ToolStoreChoiceType.WebSearch);
		}
	}
	return [];
}

export function buildUserConversationMessageFromEditor(
	payload: EditorSubmitPayload,
	existingId?: string
): ConversationMessage {
	const now = new Date();
	const id = existingId ?? getUUIDv7();

	const text = payload.text.trim();
	const hasText = text.length > 0;

	const contents: InputOutputContentItemUnion[] = [];
	if (hasText) {
		contents.push({
			kind: ContentItemKind.Text,
			textItem: { text },
		});
	}

	const inputMessage: InputOutputContent = {
		id,
		role: RoleEnum.User,
		status: Status.None,
		contents,
	};

	const inputs: InputUnion[] = [
		{
			kind: InputKind.InputMessage,
			inputMessage,
		},
	];
	// console.log('inputs from editor', JSON.stringify(inputs, null, 2));

	// Attach tool outputs as FunctionToolOutput / CustomToolOutput events.
	for (const ui of payload.toolOutputs) {
		const infOut = buildToolOutputFromEditor(ui);
		if (!infOut) continue;

		if (infOut.type === ToolType.Function) {
			inputs.push({
				kind: InputKind.FunctionToolOutput,
				functionToolOutput: infOut,
			});
		} else if (infOut.type === ToolType.Custom) {
			inputs.push({
				kind: InputKind.CustomToolOutput,
				customToolOutput: infOut,
			});
		} else if (infOut.type === ToolType.WebSearch) {
			inputs.push({
				kind: InputKind.WebSearchToolOutput,
				webSearchToolOutput: infOut,
			});
		}
	}

	const attachments =
		payload.attachments.length > 0
			? dedupeAttachmentsByRef(payload.attachments.map(uiAttachmentToConversation))
			: undefined;

	const toolStoreChoices = payload.finalToolChoices.length > 0 ? payload.finalToolChoices : undefined;

	const toolOutputs = payload.toolOutputs.length > 0 ? payload.toolOutputs : undefined;

	const msg: ConversationMessage = {
		id,
		createdAt: now,
		role: RoleEnum.User,
		status: Status.None,
		inputs,
		attachments,
		toolStoreChoices,
		uiContent: text,
		uiToolOutputs: toolOutputs,
	};
	// console.log('msg from editor', JSON.stringify(msg, null, 2));

	return msg;
}

function buildToolOutputFromEditor(ui: UIToolOutput): ToolOutput | undefined {
	// Support function, custom and web-search tool outputs.
	// For web-search outputs we only round‑trip the textual/toolOutputs
	// representation and do not attempt to reconstruct provider‑specific metadata.
	if (
		ui.type !== ToolStoreChoiceType.Function &&
		ui.type !== ToolStoreChoiceType.Custom &&
		ui.type !== ToolStoreChoiceType.WebSearch
	) {
		return undefined;
	}

	const contents = mapToolOutputsToToolOutputItems(ui.toolOutputs);

	return {
		type: ui.type as unknown as ToolType,
		choiceID: ui.choiceID,
		id: ui.id,
		callID: ui.callID,
		role: RoleEnum.Tool,
		status: Status.Completed,
		cacheControl: undefined,
		name: ui.name,
		isError: !!ui.isError,
		signature: undefined,
		contents,
		webSearchToolOutputItems: ui.webSearchToolOutputItems,
	};
}

/**
 * Hydrate a stored conversation (from Go) into the full UI Conversation shape.
 * - Derives message.content, reasoningContents, toolCalls, toolOutputs.
 */
export function hydrateConversation(store: StoreConversation): Conversation {
	// 1. Build global maps for this conversation
	const choiceMap = buildToolStoreChoiceMap(store.messages);
	const toolCallMap = buildToolCallMap(store.messages);

	const hydratedMessages: ConversationMessage[] = store.messages.map(
		(m: StoreConversationMessage): ConversationMessage => {
			const role = m.role;
			let uiContent = '';
			let uiReasoningContents: ReasoningContent[] | undefined = undefined;
			let uiToolCalls: UIToolCall[] | undefined = undefined;
			let uiToolOutputs: UIToolOutput[] | undefined = undefined;
			let uiCitations: URLCitation[] | undefined = undefined;

			const outputs: OutputUnion[] | undefined = m.outputs;
			const inputs: InputUnion[] | undefined = m.inputs;

			if (role === RoleEnum.Assistant) {
				// Text + tool calls + tool outputs from model outputs.
				const derived = deriveUIFieldsFromOutputUnion(outputs, choiceMap);

				uiContent = derived.uiContent;
				uiReasoningContents = derived.uiReasoningContents;
				uiToolCalls = derived.uiToolCalls;
				uiCitations = derived.uiCitations;

				// Prefer outputs from model response; fall back to any tool outputs in inputs.
				uiToolOutputs =
					derived.uiToolOutputs && derived.uiToolOutputs.length > 0
						? derived.uiToolOutputs
						: deriveUIToolOutputsFromInputUnion(inputs, choiceMap, toolCallMap);
			} else if (role === RoleEnum.User) {
				// User content from inputs, tool outputs from inputs.
				uiContent = deriveUIContentFromInputUnion(inputs);
				uiToolOutputs = deriveUIToolOutputsFromInputUnion(inputs, choiceMap, toolCallMap);
			} else {
				// System / developer / other roles: We do not show them in UI text bubble.
			}

			const uiDebugDetails = getDebugDetailsMarkdown(m.debugDetails, m.error);

			return {
				...(m as any),
				uiContent,
				uiReasoningContents,
				uiToolCalls,
				uiToolOutputs,
				uiCitations,
				uiDebugDetails,
			} as ConversationMessage;
		}
	);

	const allMessages = {
		...(store as any),
		messages: hydratedMessages,
	} as Conversation;

	return allMessages;
}

function buildToolStoreChoiceMap(messages: StoreConversationMessage[]): Map<string, ToolStoreChoice> {
	const map = new Map<string, ToolStoreChoice>();

	for (const m of messages) {
		if (!m.toolStoreChoices) continue;

		for (const choice of m.toolStoreChoices) {
			// choiceID is the key used by ToolCall / ToolOutput
			map.set(choice.choiceID, choice);
		}
	}

	return map;
}

function buildToolCallMap(messages: StoreConversationMessage[]): Map<string, ToolCall> {
	let map = new Map<string, ToolCall>();
	for (const m of messages) {
		map = collectToolCallsFromInputs(m.inputs, map);
		map = collectToolCallsFromOutputs(m.outputs, map);
	}
	return map;
}
/**
 * Rebuild user-visible content from InputUnion(s) for a message.
 * Used primarily for user turns, since `content` is not persisted.
 */
function deriveUIContentFromInputUnion(inputs?: InputUnion[]): string {
	if (!inputs || inputs.length === 0) return '';

	for (const iu of inputs) {
		// We want only the first user input content.
		// Attachments that are processed as text etc are not shown in UI.
		if (iu.kind !== InputKind.InputMessage || iu.inputMessage?.role != RoleEnum.User || !iu.inputMessage.contents)
			continue;

		for (const c of iu.inputMessage.contents) {
			if (c.kind === ContentItemKind.Text && c.textItem?.text) {
				const t = c.textItem.text.trim();
				if (t) return t;
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

	for (const iu of inputs) {
		let out: ToolOutput | undefined;

		if (iu.kind === InputKind.FunctionToolOutput && iu.functionToolOutput) {
			out = iu.functionToolOutput;
		} else if (iu.kind === InputKind.CustomToolOutput && iu.customToolOutput) {
			out = iu.customToolOutput;
		} else if (iu.kind === InputKind.WebSearchToolOutput && iu.webSearchToolOutput) {
			out = iu.webSearchToolOutput;
		} else {
			continue;
		}

		uiOutputs.push(buildUIToolOutputFromToolOutput(out, choiceMap, toolCallMap));
	}

	return uiOutputs;
}

function attachmentRefKey(a: Attachment): string {
	const kind = a.kind ?? '';

	if (a.fileRef) {
		const p = a.fileRef.origPath || a.fileRef.path || a.fileRef.name || '';
		return `${kind}|file|${p}`;
	}
	if (a.imageRef) {
		const p = a.imageRef.origPath || a.imageRef.path || a.imageRef.name || '';
		return `${kind}|image|${p}`;
	}
	if (a.urlRef) {
		const u = a.urlRef.origNormalized || a.urlRef.normalized || a.urlRef.url || '';
		return `${kind}|url|${u}`;
	}
	if (a.genericRef) {
		const h = a.genericRef.origHandle || a.genericRef.handle || '';
		return `${kind}|handle|${h}`;
	}

	// fallback
	return `${kind}|label|${a.label ?? ''}`;
}

export function dedupeAttachmentsByRef<T extends Attachment>(attachments?: T[]): T[] | undefined {
	if (!attachments || attachments.length === 0) return undefined;

	const seen = new Set<string>();
	const out: T[] = [];

	for (const a of attachments) {
		const key = attachmentRefKey(a);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(a);
	}
	return out.length > 0 ? out : undefined;
}
