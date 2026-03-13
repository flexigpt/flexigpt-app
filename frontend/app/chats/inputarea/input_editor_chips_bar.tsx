import { memo } from 'react';

import type { Path } from 'platejs';

import type { AttachmentContentBlockMode, UIAttachment } from '@/spec/attachment';
import type { UIToolCall, UIToolOutput } from '@/spec/inference';

import { DirectoryChip } from '@/chats/attachments/attachment_directory_chip';
import { type DirectoryAttachmentGroup, uiAttachmentKey } from '@/chats/attachments/attachment_editor_utils';
import { StandaloneAttachmentsChip } from '@/chats/attachments/attachment_standalone_chips';
import type { ConversationToolStateEntry } from '@/chats/tools/conversation_tool_utils';
import { ConversationToolsChip } from '@/chats/tools/conversation_tools_chip';
import { ToolChipsComposerRow } from '@/chats/tools/tool_chips_composer';
import { ToolChoicesChip } from '@/chats/tools/tool_choices_chip';
import type { ToolSelectionElementNode } from '@/chats/tools/tool_editor_utils';

interface EditorChipsBarProps {
	attachments: UIAttachment[];
	directoryGroups: DirectoryAttachmentGroup[];
	conversationTools?: ConversationToolStateEntry[];

	// Tool calls & outputs (tool runners / results)
	toolCalls?: UIToolCall[];
	toolOutputs?: UIToolOutput[];
	toolEntries: Array<[ToolSelectionElementNode, Path]>;
	isBusy?: boolean;
	onRunToolCall?: (id: string) => void | Promise<void>;
	onDiscardToolCall?: (id: string) => void;
	onOpenOutput?: (output: UIToolOutput) => void;
	onRemoveOutput?: (id: string) => void;
	onRetryErroredOutput?: (output: UIToolOutput) => void;

	onRemoveAttachment: (att: UIAttachment) => void;
	onChangeAttachmentContentBlockMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
	onRemoveDirectoryGroup: (groupId: string) => void;
	onRemoveOverflowDir?: (groupId: string, dirPath: string) => void;
	onConversationToolsChange?: (next: ConversationToolStateEntry[]) => void;
	onToggleAttachedToolAutoExecute: (node: ToolSelectionElementNode, next: boolean) => void;
	onRemoveAttachedTool: (node: ToolSelectionElementNode) => void;
	onRemoveAllAttachedTools: (nodes: ToolSelectionElementNode[]) => void;
	onEditAttachedToolOptions: (node: ToolSelectionElementNode) => void;
	onOpenToolCallDetails?: (call: UIToolCall) => void;
	onOpenConversationToolDetails?: (entry: ConversationToolStateEntry) => void;
	onOpenAttachedToolDetails?: (node: ToolSelectionElementNode) => void;
}

/**
 * Unified chips bar for:
 *   - Standalone attachments
 *   - Directory groups
 *   - Tool choices (attached tools)
 *   - Tool call & tool output chips
 *
 * Order (left → right):
 *   attachments → directories → tools → tool runners / outputs
 */
export const EditorChipsBar = memo(function EditorChipsBar({
	attachments,
	directoryGroups,
	conversationTools = [],
	toolCalls = [],
	toolOutputs = [],
	toolEntries,
	isBusy = false,
	onRunToolCall,
	onDiscardToolCall,
	onOpenOutput,
	onRemoveOutput,
	onRetryErroredOutput,
	onRemoveAttachment,
	onChangeAttachmentContentBlockMode,
	onRemoveDirectoryGroup,
	onRemoveOverflowDir,
	onConversationToolsChange,
	onToggleAttachedToolAutoExecute,
	onRemoveAttachedTool,
	onRemoveAllAttachedTools,
	onEditAttachedToolOptions,
	onOpenToolCallDetails,
	onOpenConversationToolDetails,
	onOpenAttachedToolDetails,
}: EditorChipsBarProps) {
	const hasVisibleToolCalls = toolCalls.some(
		toolCall => toolCall.status !== 'discarded' && toolCall.status !== 'succeeded'
	);

	const hasAnyChips =
		(conversationTools?.length ?? 0) > 0 ||
		attachments.length > 0 ||
		directoryGroups.length > 0 ||
		toolEntries.length > 0 ||
		hasVisibleToolCalls ||
		toolOutputs.length > 0;

	// Attachments that are "owned" by a directory group should not show as top-level attachments.
	const ownedKeys = new Set<string>();
	for (const group of directoryGroups) {
		for (const k of group.ownedAttachmentKeys) {
			ownedKeys.add(k);
		}
	}

	const standaloneAttachments = attachments.filter(att => !ownedKeys.has(uiAttachmentKey(att)));

	const runToolCall = onRunToolCall ?? (() => {});
	const discardToolCall = onDiscardToolCall ?? (() => {});
	const openOutput = onOpenOutput ?? (() => {});
	const removeOutput = onRemoveOutput ?? (() => {});
	const retryErroredOutput = onRetryErroredOutput ?? (() => {});
	const openToolCallDetails = onOpenToolCallDetails ?? (() => {});
	const openConversationToolDetails = onOpenConversationToolDetails ?? (() => {});
	const openAttachedToolDetails = onOpenAttachedToolDetails ?? (() => {});

	if (!hasAnyChips) return null;

	return (
		<div className="flex w-max shrink-0 flex-nowrap items-center gap-1">
			{/* Conversation tools (first, tinted) */}
			<ConversationToolsChip
				tools={conversationTools}
				onChange={onConversationToolsChange}
				onShowToolDetails={openConversationToolDetails}
			/>

			{/* Aggregated chip for standalone attachments */}
			<StandaloneAttachmentsChip
				attachments={standaloneAttachments}
				onRemoveAttachment={onRemoveAttachment}
				onChangeAttachmentContentBlockMode={onChangeAttachmentContentBlockMode}
			/>

			{/* Folder groups */}
			{directoryGroups.map(group => (
				<DirectoryChip
					key={group.id}
					group={group}
					attachments={attachments}
					onRemoveAttachment={onRemoveAttachment}
					onChangeAttachmentContentBlockMode={onChangeAttachmentContentBlockMode}
					onRemoveDirectoryGroup={onRemoveDirectoryGroup}
					onRemoveOverflowDir={onRemoveOverflowDir}
				/>
			))}

			{/* Per-message tool choices (inline-attached tools for this draft) */}
			<ToolChoicesChip
				toolEntries={toolEntries}
				onToggleAutoExecute={onToggleAttachedToolAutoExecute}
				onRemoveTool={onRemoveAttachedTool}
				onRemoveAllTools={onRemoveAllAttachedTools}
				onEditToolOptions={onEditAttachedToolOptions}
				onShowToolDetails={openAttachedToolDetails}
			/>

			{/* Tool-call chips (pending/running/failed) and tool output chips */}
			<ToolChipsComposerRow
				toolCalls={toolCalls}
				toolOutputs={toolOutputs}
				isBusy={isBusy}
				onRunToolCall={runToolCall}
				onDiscardToolCall={discardToolCall}
				onOpenOutput={openOutput}
				onRemoveOutput={removeOutput}
				onRetryErroredOutput={retryErroredOutput}
				onOpenCallDetails={openToolCallDetails}
			/>
		</div>
	);
});
