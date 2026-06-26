import { FiChevronDown, FiChevronUp } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { UIAttachment } from '@/spec/attachment';
import { AttachmentContentBlockMode } from '@/spec/attachment';

import {
	getAttachmentContentBlockModeLabel,
	getAttachmentContentBlockModePillClasses,
	getAttachmentContentBlockModeTooltip,
} from '@/chats/composer/attachments/attachment_mode_menu_utils';

/**
 * Shared styles for the small "mode" menu on each attachment chip.
 * This keeps the same visual language as other menus, but shrink-wraps
 * to the max width of its items.
 */
const modeMenuClasses =
	'rounded-box bg-base-100 text-base-content z-50 ' +
	'max-h-72 w-max min-w-0 overflow-y-auto ' +
	'border border-base-300 p-1 shadow-xl';

const modeMenuItemClasses =
	'flex items-center gap-2 rounded-xl px-2 py-1 text-xs outline-none transition-colors ' +
	'hover:bg-base-200 data-[active-item]:bg-base-300 whitespace-nowrap';

interface AttachmentContentBlockModeMenuProps {
	attachment: UIAttachment;
	onChangeAttachmentContentBlockMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
}

export function AttachmentContentBlockModeMenu({
	attachment,
	onChangeAttachmentContentBlockMode,
}: AttachmentContentBlockModeMenuProps) {
	const menu = useMenuStore({ placement: 'top-start', focusLoop: true });
	const open = useStoreState(menu, 'open');
	const currentLabel = getAttachmentContentBlockModeLabel(attachment.mode);
	const tooltip = getAttachmentContentBlockModeTooltip(attachment.mode);

	const ChevronIcon = open ? FiChevronDown : FiChevronUp;

	return (
		<>
			<MenuButton
				store={menu}
				className={getAttachmentContentBlockModePillClasses(attachment.mode, true)}
				aria-label="Change attachment mode"
				title={tooltip}
				data-attachment-mode-button
			>
				<span>{currentLabel}</span>
				<ChevronIcon className="shrink-0" size={10} aria-hidden="true" />
			</MenuButton>

			<Menu store={menu} gutter={4} className={modeMenuClasses} data-attachment-mode-menu autoFocusOnShow portal>
				{attachment.availableContentBlockModes.map(mode => {
					const label = getAttachmentContentBlockModeLabel(mode);
					const modeTooltip = getAttachmentContentBlockModeTooltip(mode);
					const isActive = mode === attachment.mode;
					const isError = mode === AttachmentContentBlockMode.notReadable;

					return (
						<MenuItem
							key={mode}
							className={modeMenuItemClasses}
							onClick={() => {
								onChangeAttachmentContentBlockMode(attachment, mode);
								menu.hide();
							}}
							aria-pressed={isActive}
							title={modeTooltip}
						>
							<span className={[isActive ? 'font-medium' : '', isError ? 'text-error' : ''].filter(Boolean).join(' ')}>
								{label}
							</span>
						</MenuItem>
					);
				})}
			</Menu>
		</>
	);
}
