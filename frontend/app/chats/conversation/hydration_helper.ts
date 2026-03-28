import type { Attachment } from '@/spec/attachment';
import {
	type Conversation,
	CONVERSATION_SCHEMA_VERSION,
	type ConversationMessage,
	type RestorableConversationContext,
} from '@/spec/conversation';
import {
	ContentItemKind,
	InputKind,
	type InputOutputContent,
	type InputOutputContentItemUnion,
	type InputUnion,
	type ModelParam,
	RoleEnum,
	Status,
	type ToolOutput,
	type ToolOutputItemUnion,
	ToolType,
	type UIToolOutput,
} from '@/spec/inference';
import type { ModelPresetRef } from '@/spec/modelpreset';
import type { SkillRef } from '@/spec/skill';
import { type ToolStoreChoice, ToolStoreChoiceType } from '@/spec/tool';

import { stripUndefinedDeep } from '@/lib/obj_utils';
import { generateTitle } from '@/lib/title_utils';
import { getUUIDv7 } from '@/lib/uuid_utils';

import { uiAttachmentToConversation } from '@/chats/composer/attachments/attachment_editor_utils';
import type { EditorSubmitPayload } from '@/chats/composer/editor/editor_types';
import { mapToolOutputsToToolOutputItems } from '@/tools/lib/tool_output_utils';

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
	const now = new Date();
	return {
		id: getUUIDv7(),
		createdAt: now,
		role,
		status: Status.None,
		uiContent: '',
	};
}

function deriveConversationToolsFromMessages(messages: ConversationMessage[]): ToolStoreChoice[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) continue;
		return (message.toolStoreChoices ?? []).filter(choice => choice.toolType !== ToolStoreChoiceType.WebSearch);
	}
	return [];
}

function deriveWebSearchChoiceFromMessages(messages: ConversationMessage[]): ToolStoreChoice[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) continue;
		return (message.toolStoreChoices ?? []).filter(choice => choice.toolType === ToolStoreChoiceType.WebSearch);
	}
	return [];
}

function deriveEnabledSkillRefsFromMessages(messages: ConversationMessage[]): SkillRef[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) continue;
		return message.enabledSkillRefs ?? [];
	}
	return [];
}

function deriveActiveSkillRefsFromMessages(messages: ConversationMessage[]): SkillRef[] {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const message = messages[i];
		if (message.role !== RoleEnum.User) continue;
		return message.activeSkillRefs ?? [];
	}
	return [];
}

function findLastModelPresetRef(messages: ConversationMessage[]): ModelPresetRef | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const ref = messages[i].modelPresetRef;
		if (!ref) continue;
		if (!ref.providerName || !ref.modelPresetID) continue;
		return ref;
	}
	return undefined;
}

function findLastModelParam(messages: ConversationMessage[]): ModelParam | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const modelParam = messages[i].modelParam;
		if (modelParam) return modelParam;
	}
	return undefined;
}

function modelParamsEqual(a?: ModelParam, b?: ModelParam): boolean {
	return JSON.stringify(stripUndefinedDeep(a)) === JSON.stringify(stripUndefinedDeep(b));
}

function findLastPersistedModelParam(messages: ConversationMessage[]): ModelParam | undefined {
	for (let i = messages.length - 1; i >= 0; i -= 1) {
		const modelParam = messages[i].modelParam;
		if (modelParam) return modelParam;
	}
	return undefined;
}

export function shouldPersistAssistantModelParam(messages: ConversationMessage[], nextModelParam: ModelParam): boolean {
	const previous = findLastPersistedModelParam(messages);
	return !previous || !modelParamsEqual(previous, nextModelParam);
}

export function applyAssistantPersistenceContext(
	message: ConversationMessage,
	modelPresetRef: ModelPresetRef,
	modelParam?: ModelParam
): ConversationMessage {
	const next = {
		...message,
		modelPresetRef,
	} as ConversationMessage & { modelParam?: ModelParam };

	if (modelParam) next.modelParam = modelParam;
	else delete next.modelParam;

	return next;
}

export function deriveRestorableConversationContextFromMessages(
	messages: ConversationMessage[]
): RestorableConversationContext {
	return {
		modelPresetRef: findLastModelPresetRef(messages),
		modelParam: findLastModelParam(messages),
		toolChoices: deriveConversationToolsFromMessages(messages),
		webSearchChoices: deriveWebSearchChoiceFromMessages(messages),
		enabledSkillRefs: deriveEnabledSkillRefsFromMessages(messages),
		activeSkillRefs: deriveActiveSkillRefsFromMessages(messages),
	};
}

export function buildUserConversationMessageFromEditor(
	payload: EditorSubmitPayload,
	existingId?: string,
	modelPresetRef?: ModelPresetRef
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

	for (const uiToolOutput of payload.toolOutputs) {
		const inferenceToolOutput = buildToolOutputFromEditor(uiToolOutput);
		if (!inferenceToolOutput) continue;

		if (inferenceToolOutput.type === ToolType.Function) {
			inputs.push({
				kind: InputKind.FunctionToolOutput,
				functionToolOutput: inferenceToolOutput,
			});
		} else if (inferenceToolOutput.type === ToolType.Custom) {
			inputs.push({
				kind: InputKind.CustomToolOutput,
				customToolOutput: inferenceToolOutput,
			});
		} else if (inferenceToolOutput.type === ToolType.WebSearch) {
			inputs.push({
				kind: InputKind.WebSearchToolOutput,
				webSearchToolOutput: inferenceToolOutput,
			});
		}
	}

	const attachments =
		payload.attachments.length > 0
			? dedupeAttachmentsByRef(payload.attachments.map(uiAttachmentToConversation))
			: undefined;

	const toolStoreChoices = payload.finalToolChoices.length > 0 ? payload.finalToolChoices : undefined;
	const toolOutputs = payload.toolOutputs.length > 0 ? payload.toolOutputs : undefined;

	const enabledSkillRefs = payload.enabledSkillRefs ?? [];
	let activeSkillRefs = payload.activeSkillRefs ?? [];

	if (enabledSkillRefs.length === 0) {
		activeSkillRefs = [];
	}

	return {
		id,
		createdAt: now,
		role: RoleEnum.User,
		status: Status.None,
		modelPresetRef,
		inputs,
		attachments,
		toolStoreChoices,
		enabledSkillRefs: enabledSkillRefs.length > 0 ? enabledSkillRefs : undefined,
		activeSkillRefs: activeSkillRefs.length > 0 ? activeSkillRefs : undefined,
		uiContent: text,
		uiToolOutputs: toolOutputs,
	};
}

function buildToolOutputFromEditor(ui: UIToolOutput): ToolOutput | undefined {
	if (
		ui.type !== ToolStoreChoiceType.Function &&
		ui.type !== ToolStoreChoiceType.Custom &&
		ui.type !== ToolStoreChoiceType.WebSearch
	) {
		return undefined;
	}

	let contents = mapToolOutputsToToolOutputItems(ui.toolOutputs);
	const hasContents = (contents?.length ?? 0) > 0;
	const hasWebSearchToolOutputItems = (ui.webSearchToolOutputItems?.length ?? 0) > 0;

	// Preserve tool errors as real tool output payloads.
	// Without this, an errored output that only has `errorMessage` becomes
	// an empty tool-output union on the wire.
	if (ui.isError && !hasContents && !hasWebSearchToolOutputItems) {
		const msg = 'Tool call returned a error, rectify the call.';
		contents = [
			{
				kind: ContentItemKind.Text,
				textItem: { text: ui.errorMessage ? 'ErrorMessage: ' + ui.errorMessage : msg },
			},
		] as ToolOutputItemUnion[];
	}

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

function attachmentRefKey(attachment: Attachment): string {
	const kind = attachment.kind ?? '';

	if (attachment.fileRef) {
		const path = attachment.fileRef.origPath || attachment.fileRef.path || attachment.fileRef.name || '';
		return `${kind}|file|${path}`;
	}
	if (attachment.imageRef) {
		const path = attachment.imageRef.origPath || attachment.imageRef.path || attachment.imageRef.name || '';
		return `${kind}|image|${path}`;
	}
	if (attachment.urlRef) {
		const url = attachment.urlRef.origNormalized || attachment.urlRef.normalized || attachment.urlRef.url || '';
		return `${kind}|url|${url}`;
	}
	if (attachment.genericRef) {
		const handle = attachment.genericRef.origHandle || attachment.genericRef.handle || '';
		return `${kind}|handle|${handle}`;
	}

	return `${kind}|label|${attachment.label ?? ''}`;
}

export function dedupeAttachmentsByRef<T extends Attachment>(attachments?: T[]): T[] | undefined {
	if (!attachments || attachments.length === 0) return undefined;

	const seen = new Set<string>();
	const out: T[] = [];

	for (const attachment of attachments) {
		const key = attachmentRefKey(attachment);
		if (seen.has(key)) continue;
		seen.add(key);
		out.push(attachment);
	}

	return out.length > 0 ? out : undefined;
}
