import { FiAlertTriangle, FiChevronUp, FiFolder, FiX } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore } from '@ariakit/react';

import { AttachmentContentBlockMode, type UIAttachment } from '@/spec/attachment';

import { AttachmentChip } from '@/chats/attachments/attachment_chip';
import { type DirectoryAttachmentGroup, uiAttachmentKey } from '@/chats/attachments/attachment_editor_utils';
import { getAttachmentContentBlockModePillClasses } from '@/chats/attachments/attachment_mode_menu';

interface DirectoryChipProps {
	group: DirectoryAttachmentGroup;
	attachments: UIAttachment[];
	onRemoveAttachment: (att: UIAttachment) => void;
	onChangeAttachmentContentBlockMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
	onRemoveDirectoryGroup: (groupId: string) => void;
	onRemoveOverflowDir?: (groupId: string, dirPath: string) => void;
}

/**
 * Directory pill with dropdown menu listing its files and any overflow (error) subdirs.
 * Owns its Ariakit menu store, so hooks are only called at component top level.
 */
export function DirectoryChip({
	group,
	attachments,
	onRemoveAttachment,
	onChangeAttachmentContentBlockMode,
	onRemoveDirectoryGroup,
	onRemoveOverflowDir,
}: DirectoryChipProps) {
	const directoryMenu = useMenuStore({ placement: 'bottom-start', focusLoop: true });

	// Build a map for quick lookup of attachments by key
	const attachmentByKey = new Map<string, UIAttachment>();
	for (const att of attachments) {
		attachmentByKey.set(uiAttachmentKey(att), att);
	}

	const attachedCount = group.attachmentKeys.length;

	const overflowFileCount = group.overflowDirs.reduce((sum, o) => sum + (o.fileCount ?? 0), 0);

	// Tooltip: show full folder path + any extra info that isn't obvious from the chip text
	const tooltipLines: string[] = [];
	if (group.dirPath && group.dirPath !== group.label) {
		tooltipLines.push(group.dirPath);
	}
	tooltipLines.push(`${attachedCount} file${attachedCount === 1 ? '' : 's'} attached`);
	if (overflowFileCount > 0) {
		tooltipLines.push(
			`${overflowFileCount} additional file${overflowFileCount === 1 ? '' : 's'} not attached from subfolders`
		);
	}
	const title = tooltipLines.join('\n');

	return (
		<div
			className="bg-base-200 hover:bg-base-300/80 text-base-content flex shrink-0 items-center gap-2 rounded-2xl px-2 py-0"
			title={title}
			data-attachment-chip="directory"
		>
			<FiFolder className="shrink-0" size={14} />
			<span className="min-w-0 flex-1 truncate">{group.label}</span>
			<span className="text-base-content/60 whitespace-nowrap">
				{attachedCount} files
				{overflowFileCount > 0 ? ` +${overflowFileCount} more` : ''}
			</span>

			{/* Menu trigger */}
			<MenuButton
				store={directoryMenu}
				className="btn btn-ghost btn-xs px-0 py-0 shadow-none"
				aria-label={`Show files in folder ${group.dirPath || group.label}`}
				title={group.dirPath || group.label}
			>
				<FiChevronUp size={14} />
			</MenuButton>

			{/* Remove entire folder group */}
			<button
				type="button"
				className="btn btn-ghost btn-xs text-error shrink-0 px-0 py-0 shadow-none"
				onClick={() => {
					onRemoveDirectoryGroup(group.id);
				}}
				title="Remove this folder and its attached files"
				aria-label="Remove folder attachment group"
			>
				<FiX size={14} />
			</button>

			<Menu
				store={directoryMenu}
				gutter={6}
				className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 min-w-65 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="text-base-content/70 mb-1 text-[11px] font-semibold">Files in “{group.label}”</div>

				{group.attachmentKeys.map(key => {
					const att = attachmentByKey.get(key);
					if (!att) return null;

					return (
						<MenuItem
							key={key}
							store={directoryMenu}
							hideOnClick={false} // keep menu open when interacting with chip
							className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						>
							<AttachmentChip
								attachment={att}
								onRemoveAttachment={onRemoveAttachment}
								onChangeAttachmentContentBlockMode={onChangeAttachmentContentBlockMode}
								fullWidth
							/>
						</MenuItem>
					);
				})}

				{group.overflowDirs.length > 0 && (
					<div className="border-base-300 mt-2 space-y-1 border-t pt-2">
						{group.overflowDirs.map(od => {
							const rel = od.relativePath || od.dirPath;
							return (
								<MenuItem
									key={od.dirPath}
									store={directoryMenu}
									hideOnClick={false}
									className="bg-base-200/60 data-active-item:bg-base-200 flex items-start gap-2 rounded-xl px-2 py-1"
								>
									<FiAlertTriangle size={14} className="text-warning mt-0.5" />
									<div className="min-w-0 flex-1">
										<div
											className="truncate text-xs font-medium"
											title={od.dirPath !== rel ? `${rel}\n${od.dirPath}` : od.dirPath}
										>
											{rel}
										</div>
										<div className="text-base-content/70 truncate text-[11px]">
											{od.fileCount} item
											{od.fileCount === 1 ? '' : 's'} not attached in this folder
											{od.partial ? ' (folder only partially scanned)' : ''}
										</div>
									</div>
									<span
										className={getAttachmentContentBlockModePillClasses(AttachmentContentBlockMode.notReadable, false)}
										title="Too many files in this subfolder; they were not attached."
									>
										Not readable
									</span>
									{onRemoveOverflowDir && (
										<button
											type="button"
											className="btn btn-ghost btn-xs text-error shrink-0 px-1 py-0 shadow-none"
											onClick={() => {
												onRemoveOverflowDir(group.id, od.dirPath);
											}}
											title="Hide this skipped subfolder notice"
											aria-label="Hide skipped subfolder notice"
										>
											<FiX size={14} />
										</button>
									)}
								</MenuItem>
							);
						})}
					</div>
				)}

				{group.attachmentKeys.length === 0 && group.overflowDirs.length === 0 && (
					<div className="text-base-content/70 text-[11px]">No readable files could be attached from this folder.</div>
				)}
			</Menu>
		</div>
	);
}
