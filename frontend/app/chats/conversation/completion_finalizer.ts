import type { Conversation, ConversationMessage } from '@/spec/conversation';
import type { CompletionResponseBody, InferenceError, OutputUnion, ReasoningContent, Status } from '@/spec/inference';
import { ContentItemKind, OutputKind, RoleEnum } from '@/spec/inference';

export function streamHasData(value: string): boolean {
	return value.length > 0;
}

function closeUnterminatedCodeFence(text: string): string {
	let openFence: string | undefined;

	for (const line of text.split('\n')) {
		const match = /^ {0,3}(`{3,}|~{3,})/.exec(line);
		if (!match?.[1]) {
			continue;
		}

		const marker = match[1];
		if (!openFence) {
			openFence = marker;
			continue;
		}

		if (marker.startsWith(openFence)) {
			openFence = undefined;
		}
	}

	if (!openFence) {
		return text;
	}

	return `${text}${text.endsWith('\n') ? '' : '\n'}${openFence}`;
}

function appendTerminalLine(text: string, terminalLine: string): string {
	if (!text) {
		return terminalLine;
	}

	const boundedText = closeUnterminatedCodeFence(text);
	return `${boundedText}${boundedText.endsWith('\n') ? '\n' : '\n\n'}${terminalLine}`;
}

function chooseAssistantText(finalText: string, streamedText: string, preferStreamedText = false): string {
	if (!streamedText) {
		return finalText;
	}
	if (!finalText) {
		return streamedText;
	}

	// Prefer the complete value when one source is a prefix of the other.
	if (finalText.startsWith(streamedText)) {
		return finalText;
	}
	if (streamedText.startsWith(finalText)) {
		return streamedText;
	}

	// On an API failure, emitted text is the most trustworthy representation
	// of what the user actually saw before the request ended.
	return preferStreamedText ? streamedText : finalText;
}

function upsertAssistantTextInOutputs(
	outputs: OutputUnion[] | undefined,
	assistantMessageID: string,
	status: Status,
	text: string
): OutputUnion[] {
	const nextOutputs: OutputUnion[] = [...(outputs ?? [])];
	let foundAssistantOutput = false;
	let wroteText = false;

	for (let outputIndex = 0; outputIndex < nextOutputs.length; outputIndex += 1) {
		const output = nextOutputs[outputIndex];
		if (
			output.kind !== OutputKind.OutputMessage ||
			!output.outputMessage ||
			output.outputMessage.role !== RoleEnum.Assistant
		) {
			continue;
		}

		foundAssistantOutput = true;
		const contents = [...(output.outputMessage.contents ?? [])];
		let foundTextItem = false;

		for (let contentIndex = 0; contentIndex < contents.length; contentIndex += 1) {
			const content = contents[contentIndex];
			if (content.kind !== ContentItemKind.Text || !content.textItem) {
				continue;
			}

			foundTextItem = true;
			contents[contentIndex] = {
				...content,
				textItem: {
					...content.textItem,
					text: wroteText ? '' : text,
				},
			};
			wroteText = true;
		}

		if (!foundTextItem && !wroteText) {
			contents.push({
				kind: ContentItemKind.Text,
				textItem: { text },
			});
			wroteText = true;
		}

		nextOutputs[outputIndex] = {
			...output,
			outputMessage: {
				...output.outputMessage,
				status,
				contents,
			},
		};
	}

	if (!foundAssistantOutput || !wroteText) {
		nextOutputs.push({
			kind: OutputKind.OutputMessage,
			outputMessage: {
				id: assistantMessageID,
				role: RoleEnum.Assistant,
				status,
				contents: [
					{
						kind: ContentItemKind.Text,
						textItem: { text },
					},
				],
			},
		});
	}

	return nextOutputs;
}

function mergeStreamedThinking(
	existing: ReasoningContent[] | undefined,
	streamedThinking: string
): ReasoningContent[] | undefined {
	if (!streamHasData(streamedThinking)) {
		return existing;
	}

	const existingThinking = (existing ?? []).flatMap(reasoning => reasoning.thinking ?? []).join('\n\n');
	if (existingThinking.includes(streamedThinking)) {
		return existing;
	}

	// This is intentionally UI-only. Conversation persistence strips
	// uiReasoningContents while the partial answer remains in outputs.
	return [...(existing ?? []), { thinking: [streamedThinking] } as ReasoningContent];
}

function withAssistantText(
	message: ConversationMessage,
	status: Status,
	text: string,
	error?: InferenceError
): ConversationMessage {
	return {
		...message,
		status,
		error: error ?? message.error,
		uiContent: text,
		outputs: upsertAssistantTextInOutputs(message.outputs, message.id, status, text),
	};
}

export function reconcileAssistantTextWithStream(
	message: ConversationMessage,
	streamedText: string
): ConversationMessage {
	const reconciledText = chooseAssistantText(message.uiContent ?? '', streamedText);
	if (reconciledText === (message.uiContent ?? '')) {
		return message;
	}

	return withAssistantText(message, message.status, reconciledText);
}

export function buildTerminalAssistantMessage(args: {
	baseMessage: ConversationMessage;
	status: Status;
	partialText: string;
	partialThinking?: string;
	terminalLine: string;
	error?: InferenceError;
	baseText?: string;
	preferStreamedText?: boolean;
	includeStreamedThinking?: boolean;
}): ConversationMessage {
	const selectedText = chooseAssistantText(
		args.baseText ?? args.baseMessage.uiContent ?? '',
		args.partialText,
		args.preferStreamedText
	);
	const next = withAssistantText(
		args.baseMessage,
		args.status,
		appendTerminalLine(selectedText, args.terminalLine),
		args.error
	);

	if (args.includeStreamedThinking) {
		next.uiReasoningContents = mergeStreamedThinking(args.baseMessage.uiReasoningContents, args.partialThinking ?? '');
	}

	return next;
}

export function getInferenceFailureMessage(error: InferenceError | undefined, fallback: string): string {
	const message = typeof error?.message === 'string' ? error.message.replaceAll(/\s+/g, ' ').trim() : '';
	return message || fallback;
}

export function getFailureTerminalLine(message: string, endedAbruptly: boolean): string {
	return endedAbruptly ? `> Error: API ended abruptly. ${message}` : `> Error: ${message}`;
}

export function hasNonTextAssistantOutcome(message: ConversationMessage): boolean {
	if ((message.uiToolCalls?.length ?? 0) > 0 || (message.uiToolOutputs?.length ?? 0) > 0) {
		return true;
	}

	return (message.outputs ?? []).some(
		output =>
			output.kind === OutputKind.FunctionToolCall ||
			output.kind === OutputKind.CustomToolCall ||
			output.kind === OutputKind.WebSearchToolCall ||
			output.kind === OutputKind.WebSearchToolOutput
	);
}

export function applyCompletionMetadata(
	conversation: Conversation,
	currentUserMessageID: string | undefined,
	rawResponse: CompletionResponseBody | undefined
): Conversation {
	if (!currentUserMessageID || !rawResponse || (!rawResponse.hydratedCurrentInputs && !rawResponse.workspaceUsage)) {
		return conversation;
	}

	return {
		...conversation,
		messages: conversation.messages.map(message =>
			message.id === currentUserMessageID
				? {
						...message,
						inputs: rawResponse.hydratedCurrentInputs ?? message.inputs,
						workspaceUsage: rawResponse.workspaceUsage,
					}
				: message
		),
	};
}
