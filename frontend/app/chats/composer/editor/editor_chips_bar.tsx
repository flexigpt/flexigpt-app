import { memo } from 'react';

import type { AttachmentContentBlockMode, UIAttachment } from '@/spec/attachment';
import type { UIToolCall, UIToolOutput } from '@/spec/inference';

import { DirectoryChip } from '@/chats/composer/attachments/attachment_directory_chip';
import { type DirectoryAttachmentGroup, uiAttachmentKey } from '@/chats/composer/attachments/attachment_editor_utils';
import { StandaloneAttachmentsChip } from '@/chats/composer/attachments/attachment_standalone_chips';
import { ToolChipsComposerRow } from '@/chats/composer/tools/tool_chips_composer';

interface EditorChipsBarProps {
	attachments: UIAttachment[];
	directoryGroups: DirectoryAttachmentGroup[];

	toolCalls?: UIToolCall[];
	toolOutputs?: UIToolOutput[];
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
	onOpenToolCallDetails?: (call: UIToolCall) => void;
}

/**
 * Unified chips bar for:
 *   - Standalone attachments
 *   - Directory groups
 *   - Tool call & tool output chips
 *
 * Order (left → right):
 *   attachments → directories → tool runners / outputs
 */
export const EditorChipsBar = memo(function EditorChipsBar({
	attachments,
	directoryGroups,
	toolCalls = [],
	toolOutputs = [],
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
	onOpenToolCallDetails,
}: EditorChipsBarProps) {
	const hasVisibleToolCalls = toolCalls.some(
		toolCall => toolCall.status !== 'discarded' && toolCall.status !== 'succeeded'
	);

	const hasAnyChips =
		attachments.length > 0 || directoryGroups.length > 0 || hasVisibleToolCalls || toolOutputs.length > 0;

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

	if (!hasAnyChips) {
		return null;
	}

	return (
		<div
			className="app-scrollbar-thin flex w-max shrink-0 flex-nowrap items-center gap-1 p-1"
			style={{ scrollbarGutter: 'stable' }}
		>
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
