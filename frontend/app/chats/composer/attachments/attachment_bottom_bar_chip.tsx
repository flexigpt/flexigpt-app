import type { RefObject } from 'react';
import { useEffect, useState } from 'react';
import { FiFolder, FiLink, FiPaperclip, FiUpload } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { UIAttachment } from '@/spec/attachment';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	ActionTriggerMenu,
	ActionTriggerMenuBody,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuMetaClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
import { HoverTip } from '@/components/hover_tip';

import type { DirectoryAttachmentGroup } from '@/chats/composer/attachments/attachment_editor_utils';
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
	removedStandaloneAttachments: UIAttachment[];
	directoryGroups: DirectoryAttachmentGroup[];
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
	removedStandaloneAttachments,
	directoryGroups,
	isInputLocked = false,
}: AttachmentBottomBarChipProps) {
	const [isUrlModalOpen, setIsUrlModalOpen] = useState(false);
	const open = useStoreState(store, 'open');
	const tooltip = shortcut ? `Attach files, folders, or URLs (${shortcut})` : 'Attach files, folders, or URLs';

	const attachmentCount = attachments.length;
	const directoryCount = directoryGroups.length;
	const activeAttachmentKeys = new Set(attachments.map(attachment => uiAttachmentKey(attachment)));
	const notAddedAttachmentKeys = new Set<string>();

	for (const attachment of removedStandaloneAttachments) {
		const key = uiAttachmentKey(attachment);
		if (!activeAttachmentKeys.has(key)) {
			notAddedAttachmentKeys.add(key);
		}
	}
	for (const group of directoryGroups) {
		for (const attachment of group.removedAttachments ?? []) {
			const key = uiAttachmentKey(attachment);
			if (!activeAttachmentKeys.has(key)) {
				notAddedAttachmentKeys.add(key);
			}
		}
	}

	const notAddedAttachmentCount = notAddedAttachmentKeys.size;
	const attachmentStatusSummary =
		attachmentCount > 0 || notAddedAttachmentCount > 0
			? `${attachmentCount} added${notAddedAttachmentCount > 0 ? ` · ${notAddedAttachmentCount} not added` : ''}`
			: null;
	const hasAttachmentChoices = attachmentStatusSummary !== null || directoryCount > 0;
	const addedAttachmentSummary = [
		attachmentStatusSummary,
		directoryCount > 0 ? `${directoryCount} folder${directoryCount === 1 ? '' : 's'}` : null,
	]
		.filter((value): value is string => value !== null)
		.join(' · ');
	const chipTone = open
		? 'border-base-300 bg-base-300/60'
		: hasAttachmentChoices
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: '';

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
					className={`${actionTriggerChipButtonClasses} ${chipTone} hover:text-base-content ${
						isInputLocked ? 'opacity-60' : ''
					}`}
					aria-label={tooltip}
				>
					<ActionTriggerChipContent icon={<FiPaperclip size={16} />} label="Attachments" open={open} />
				</MenuButton>
			</HoverTip>

			<ActionTriggerMenu store={store} data-menu-kind="attachments">
				<ActionTriggerMenuHeader>
					<div className={actionTriggerMenuHeaderRowClasses}>
						<div className="min-w-0 flex-1 truncate text-xs font-semibold">Attachments</div>
						<span className={actionTriggerMenuMetaClasses}>{addedAttachmentSummary || '0 added'}</span>
					</div>
				</ActionTriggerMenuHeader>

				<ActionTriggerMenuBody>
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
				</ActionTriggerMenuBody>

				<ComposerDropdownSelectedPanel
					label="Added attachments"
					ariaLabel="Added attachments"
					isEmpty={!hasAttachmentChoices}
					emptyLabel="Choose Files, Folder, or Link / URL above to add it to this message."
					help="Added files and folders appear in the composer attachment bar. Not-added candidates remain available there until restored, cleared, or sent."
				>
					{null}
				</ComposerDropdownSelectedPanel>
			</ActionTriggerMenu>

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
			removedStandaloneAttachments={props.removedStandaloneAttachments}
			directoryGroups={props.directoryGroups}
			isInputLocked={isInputLocked}
		/>
	);
}
