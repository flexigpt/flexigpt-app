import type { RefObject } from 'react';
import { useEffect, useState } from 'react';
import { FiFolder, FiLink, FiPaperclip, FiUpload, FiX } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { AttachmentContentBlockMode, UIAttachment } from '@/spec/attachment';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
import { HoverTip } from '@/components/hover_tip';

import type { DirectoryAttachmentGroup } from '@/chats/composer/attachments/attachment_editor_utils';
import { AttachmentChip } from '@/chats/composer/attachments/attachment_chip';
import { uiAttachmentKey } from '@/chats/composer/attachments/attachment_editor_utils';
import { UrlAttachmentModal } from '@/chats/composer/attachments/attachment_url_modal';

interface AttachmentBottomBarChipProps {
	store: MenuStore;
	buttonRef: RefObject<HTMLButtonElement | null>;
	shortcut?: string;
	onAttachFiles: () => Promise<void> | void;
	onAttachDirectory: () => Promise<void> | void;
	onAttachURL: (url: string) => Promise<void> | void;
	onOpenAttachmentUrlModal?: () => void;
	onUrlAttachmentModalClose?: () => void;
	attachments: UIAttachment[];
	directoryGroups: DirectoryAttachmentGroup[];
	onRemoveAttachment: (attachment: UIAttachment) => void;
	onChangeAttachmentContentBlockMode: (attachment: UIAttachment, mode: AttachmentContentBlockMode) => void;
	onRemoveDirectoryGroup: (groupID: string) => void;
	isInputLocked?: boolean;
}

function AttachmentBottomBarChipInner({
	store,
	buttonRef,
	shortcut,
	onAttachFiles,
	onAttachDirectory,
	onAttachURL,
	onOpenAttachmentUrlModal,
	onUrlAttachmentModalClose,
	attachments,
	directoryGroups,
	onRemoveAttachment,
	onChangeAttachmentContentBlockMode,
	onRemoveDirectoryGroup,
	isInputLocked = false,
}: AttachmentBottomBarChipProps) {
	const [isUrlModalOpen, setIsUrlModalOpen] = useState(false);
	const open = useStoreState(store, 'open');
	const tooltip = shortcut ? `Attach files, folders, or URLs (${shortcut})` : 'Attach files, folders, or URLs';

	const ownedAttachmentKeys = new Set(directoryGroups.flatMap(group => group.ownedAttachmentKeys));
	const standaloneAttachments = attachments.filter(attachment => !ownedAttachmentKeys.has(uiAttachmentKey(attachment)));
	const selectedCount = attachments.length + directoryGroups.length;

	const closeMenu = () => {
		store.hide();
	};

	const handlePickFiles = async () => {
		await onAttachFiles();
		closeMenu();
	};

	const handlePickDirectory = async () => {
		await onAttachDirectory();
		closeMenu();
	};

	const handlePickURL = () => {
		onOpenAttachmentUrlModal?.();
		closeMenu();
		setIsUrlModalOpen(true);
	};

	return (
		<div className="relative shrink-0" data-bottom-bar-attachments>
			<HoverTip content={tooltip} placement="top">
				<MenuButton
					ref={buttonRef}
					store={store}
					disabled={isInputLocked}
					className={`${actionTriggerChipButtonClasses} hover:text-base-content ${isInputLocked ? 'opacity-60' : ''}`}
					aria-label={tooltip}
				>
					<ActionTriggerChipContent icon={<FiPaperclip size={16} />} label="Attachments" open={open} />
				</MenuButton>
			</HoverTip>

			<Menu
				store={store}
				gutter={8}
				overflowPadding={8}
				portal
				className={actionTriggerMenuWideClasses}
				data-menu-kind="attachments"
				autoFocusOnShow
			>
				<MenuItem
					onClick={() => {
						void handlePickFiles();
					}}
					className={actionTriggerMenuItemClasses}
				>
					<FiUpload size={14} />
					<span>Multiple Files...</span>
				</MenuItem>
				<MenuItem
					onClick={() => {
						void handlePickDirectory();
					}}
					className={actionTriggerMenuItemClasses}
				>
					<FiFolder size={14} />
					<span>Folder...</span>
				</MenuItem>
				<MenuItem onClick={handlePickURL} className={actionTriggerMenuItemClasses}>
					<FiLink size={14} />
					<span>Link or URL...</span>
				</MenuItem>

				<ComposerDropdownSelectedPanel
					label="Added to this message"
					count={selectedCount}
					emptyLabel="Attachments selected for this message will appear here."
				>
					{directoryGroups.map(group => (
						<div key={group.id} className="bg-base-200 flex min-w-0 items-center gap-2 rounded-xl px-2 py-1 text-xs">
							<FiFolder size={13} className="shrink-0" />
							<span className="min-w-0 flex-1 truncate">{group.label}</span>
							<span className="text-base-content/60 shrink-0">{group.attachmentKeys.length} attached</span>
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error shrink-0 p-0 shadow-none"
								onClick={() => {
									onRemoveDirectoryGroup(group.id);
								}}
								aria-label={`Remove folder ${group.label}`}
								title="Remove folder attachment group"
							>
								<FiX size={13} />
							</button>
						</div>
					))}

					{standaloneAttachments.map(attachment => (
						<AttachmentChip
							key={uiAttachmentKey(attachment)}
							attachment={attachment}
							onRemoveAttachment={onRemoveAttachment}
							onChangeAttachmentContentBlockMode={onChangeAttachmentContentBlockMode}
							fullWidth
						/>
					))}
				</ComposerDropdownSelectedPanel>
			</Menu>

			<UrlAttachmentModal
				isOpen={isUrlModalOpen && !isInputLocked}
				onClose={() => {
					setIsUrlModalOpen(false);
					onUrlAttachmentModalClose?.();
				}}
				onAttachURL={onAttachURL}
			/>
		</div>
	);
}

export function AttachmentBottomBarChip(props: AttachmentBottomBarChipProps) {
	const { isInputLocked = false, store } = props;

	useEffect(() => {
		if (isInputLocked) {
			store.hide();
		}
	}, [isInputLocked, store]);

	// A lock transition remounts the stateful picker, which resets the URL
	// modal without scheduling a state update from an effect.
	return (
		<AttachmentBottomBarChipInner
			key={isInputLocked ? 'locked' : 'unlocked'}
			store={store}
			buttonRef={props.buttonRef}
			shortcut={props.shortcut}
			onAttachFiles={props.onAttachFiles}
			onAttachDirectory={props.onAttachDirectory}
			onAttachURL={props.onAttachURL}
			onOpenAttachmentUrlModal={props.onOpenAttachmentUrlModal}
			onUrlAttachmentModalClose={props.onUrlAttachmentModalClose}
			attachments={props.attachments}
			directoryGroups={props.directoryGroups}
			onRemoveAttachment={props.onRemoveAttachment}
			onChangeAttachmentContentBlockMode={props.onChangeAttachmentContentBlockMode}
			onRemoveDirectoryGroup={props.onRemoveDirectoryGroup}
			isInputLocked={isInputLocked}
		/>
	);
}
