import type { ConversationMessage } from '@/spec/conversation';
import { InputKind, RoleEnum } from '@/spec/inference';
import type { IncludePreviousMessages } from '@/spec/modelpreset';

function isInstructionMessage(message: ConversationMessage): boolean {
	return message.role === RoleEnum.System || message.role === RoleEnum.Developer;
}

function messageHasUserToolOutputs(message: ConversationMessage): boolean {
	if (message.role !== RoleEnum.User) return false;

	// UI-derived field is the cheapest signal and is present for hydrated history
	// as well as freshly-built user messages from the editor.
	if ((message.uiToolOutputs?.length ?? 0) > 0) return true;

	// Fall back to the persisted/raw input unions for robustness.
	return (
		message.inputs?.some(
			input =>
				input.kind === InputKind.FunctionToolOutput ||
				input.kind === InputKind.CustomToolOutput ||
				input.kind === InputKind.WebSearchToolOutput
		) ?? false
	);
}

function isPureUserTurn(message: ConversationMessage): boolean {
	return message.role === RoleEnum.User && !messageHasUserToolOutputs(message);
}

/**
 * Numeric includePreviousMessages is interpreted as:
 *
 *   "include the current turn plus N previous pure user turns"
 *
 * A pure user turn is a user message that does NOT already contain tool outputs.
 * Tool-output followup user messages are not counted as standalone start turns,
 * because they depend on earlier assistant tool calls and the earlier user turn
 * that requested those tools.
 */
function findRequestedPureUserStartIndex(
	messages: ConversationMessage[],
	includePreviousMessages: IncludePreviousMessages
): number {
	if (messages.length === 0) return 0;
	if (includePreviousMessages === 'all') return 0;

	const safeCount = Math.max(0, includePreviousMessages);
	const targetPureUserTurns = safeCount + 1; // current turn + N previous pure user turns

	let seenPureUserTurns = 0;

	for (let i = messages.length - 1; i >= 0; i -= 1) {
		if (!isPureUserTurn(messages[i])) continue;

		seenPureUserTurns += 1;
		if (seenPureUserTurns >= targetPureUserTurns) {
			return i;
		}
	}

	// Not enough pure user turns exist; include everything.
	return 0;
}

/**
 * Preserve all instruction-bearing top-level messages before the selected start.
 *
 * This intentionally keeps all earlier system/developer messages, even if they
 * were interleaved among older omitted turns. That is the safer bias:
 * instructions should be preserved, while older user/assistant/tool turns may
 * be dropped according to the selected pure-user-turn boundary.
 *
 * Any provider-specific collapsing/merging of multiple system/developer blocks
 * should happen later in the provider adapter layer, not here.
 */
function collectPinnedInstructionMessagesBeforeIndex(
	messages: ConversationMessage[],
	beforeIdx: number
): ConversationMessage[] {
	if (beforeIdx <= 0) return [];
	return messages.slice(0, beforeIdx).filter(isInstructionMessage);
}

/**
 * Build the message list to send to inference.
 *
 * Semantics:
 * - "all" => send everything
 * - numeric N => include the current turn plus N previous pure user turns
 * - preserve all top-level system/developer messages that occur before the chosen start turn
 * - include the full suffix from the chosen pure user turn onward
 */
export function sliceMessagesForSend(
	messages: ConversationMessage[],
	includePreviousMessages: IncludePreviousMessages
): ConversationMessage[] {
	if (messages.length === 0) return [];
	if (includePreviousMessages === 'all') return messages;

	const startIdx = findRequestedPureUserStartIndex(messages, includePreviousMessages);
	if (startIdx <= 0) return messages;

	const pinnedInstructionPrefix = collectPinnedInstructionMessagesBeforeIndex(messages, startIdx);
	if (pinnedInstructionPrefix.length === 0) {
		return messages.slice(startIdx);
	}

	return [...pinnedInstructionPrefix, ...messages.slice(startIdx)];
}
