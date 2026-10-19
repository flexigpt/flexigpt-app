import { useMemo } from 'react';
import { FiPaperclip, FiX } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { AttachmentContentBlockMode, UIAttachment } from '@/spec/attachment';

import {
	ActionTriggerChipContent,
	actionTriggerChipMenuButtonClasses,
	actionTriggerChipSurfaceClasses,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuMetaClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
import { SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import { useSearchableMenuState } from '@/components/searchmenu/searchable_menu_utils';

import { AttachmentChip } from '@/chats/composer/attachments/attachment_chip';
import {
	getAttachmentDisplayLabel,
	getUIAttachmentPath,
	uiAttachmentKey,
} from '@/chats/composer/attachments/attachment_editor_utils';

interface StandaloneAttachmentsChipProps {
	attachments: UIAttachment[];
	onRemoveAttachment: (att: UIAttachment) => void;
	onChangeAttachmentContentBlockMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
}

/**
 * Aggregated "Attachments" chip for standalone files/links.
 * Opens a dropdown listing each attachment as a full-width chip.
 * Includes a "remove all" cross that clears all standalone attachments.
 */
export function StandaloneAttachmentsChip({
	attachments,
	onRemoveAttachment,
	onChangeAttachmentContentBlockMode,
}: StandaloneAttachmentsChipProps) {
	const count = attachments.length;
	const menu = useMenuStore({ placement: 'bottom-start', focusLoop: true });
	const open = useStoreState(menu, 'open');
	const [query, setQuery] = useSearchableMenuState(open);
	const filteredAttachments = useMemo(() => {
		const normalized = query.trim().toLocaleLowerCase();
		if (!normalized) {
			return attachments;
		}

		return attachments.filter(attachment =>
			[getAttachmentDisplayLabel(attachment), getUIAttachmentPath(attachment)]
				.join('\n')
				.toLocaleLowerCase()
				.includes(normalized)
		);
	}, [attachments, query]);

	if (count === 0) {
		return null;
	}

	return (
		<div className="relative shrink-0" data-attachment-chip="attachments-group">
			<div
				className={`${actionTriggerChipSurfaceClasses} ${
					open ? 'border-base-300 bg-base-300/60' : 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
				}`}
			>
				<MenuButton
					store={menu}
					className={actionTriggerChipMenuButtonClasses}
					aria-label="Show attached files and links"
				>
					<ActionTriggerChipContent icon={<FiPaperclip size={14} />} label="Attachments" open={open} />
				</MenuButton>
			</div>

			<Menu
				store={menu}
				gutter={8}
				overflowPadding={8}
				portal
				className={actionTriggerMenuWideClasses}
				autoFocusOnShow={false}
			>
				<ActionTriggerMenuHeader>
					<SearchableMenuInput
						open={open}
						query={query}
						onQueryChange={setQuery}
						placeholder="Search added files and links…"
						onEscape={() => {
							menu.hide();
						}}
					/>
					<div className={actionTriggerMenuHeaderRowClasses}>
						<div className="min-w-0 flex-1 truncate text-xs font-semibold">Attachments</div>
						<span className={actionTriggerMenuMetaClasses}>
							{query.trim() ? `${filteredAttachments.length} matches` : `${count} added`}
						</span>
					</div>
				</ActionTriggerMenuHeader>

				<ComposerDropdownSelectedPanel
					label="Added files and links"
					ariaLabel="Added files and links"
					isEmpty={filteredAttachments.length === 0}
					emptyLabel="No added files or links match this search. Clear the search to review all added items."
					help="These attachments are included with this message. Remove individual items below or clear the full set."
					summary={<span>{query.trim() ? `${filteredAttachments.length}/${count} shown` : `${count} added`}</span>}
					actions={
						<button
							type="button"
							className="btn btn-ghost btn-xs text-error rounded-lg"
							onClick={() => {
								for (const att of attachments) {
									onRemoveAttachment(att);
								}
							}}
						>
							<FiX size={12} />
							Clear all
						</button>
					}
				>
					{filteredAttachments.map(att => (
						<MenuItem
							key={uiAttachmentKey(att)}
							store={menu}
							hideOnClick={false}
							className="data-active-item:bg-base-200 rounded-xl"
						>
							<AttachmentChip
								attachment={att}
								onRemoveAttachment={onRemoveAttachment}
								onChangeAttachmentContentBlockMode={onChangeAttachmentContentBlockMode}
								fullWidth
							/>
						</MenuItem>
					))}
				</ComposerDropdownSelectedPanel>
			</Menu>
		</div>
	);
}
