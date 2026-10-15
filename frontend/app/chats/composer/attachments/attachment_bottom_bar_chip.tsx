import type { RefObject } from 'react';
import { useEffect, useState } from 'react';
import { FiFolder, FiLink, FiPaperclip, FiUpload } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/hover_tip';

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
	isInputLocked = false,
}: AttachmentBottomBarChipProps) {
	const [isUrlModalOpen, setIsUrlModalOpen] = useState(false);
	const open = useStoreState(store, 'open');
	const tooltip = shortcut ? `Attach files, folders, or URLs (${shortcut})` : 'Attach files, folders, or URLs';

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
			isInputLocked={isInputLocked}
		/>
	);
}
