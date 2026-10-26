import { useMemo, useState } from 'react';
import { FiChevronDown, FiChevronUp, FiPaperclip, FiPlus, FiX } from 'react-icons/fi';

import { MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { AttachmentContentBlockMode, UIAttachment } from '@/spec/attachment';

import {
	ActionTriggerChipContent,
	actionTriggerChipMenuButtonClasses,
	actionTriggerChipRemoveButtonClasses,
	actionTriggerChipSurfaceClasses,
	ActionTriggerMenu,
	ActionTriggerMenuBody,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
	actionTriggerMenuMetaClasses,
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
	removedAttachments: UIAttachment[];
	onRemoveAttachment: (att: UIAttachment) => void;
	onRestoreAttachments: (attachments: UIAttachment[]) => void;
	onClearAttachments: () => void;
	onChangeAttachmentContentBlockMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
}

/**
 * Aggregated "Attachments" chip for standalone files/links.
 * Opens a dropdown listing each attachment as a full-width chip.
 * Removed attachments remain available until the message is sent or the
 * collection-level clear action is used.
 */
export function StandaloneAttachmentsChip({
	attachments,
	removedAttachments,
	onRemoveAttachment,
	onRestoreAttachments,
	onClearAttachments,
	onChangeAttachmentContentBlockMode,
}: StandaloneAttachmentsChipProps) {
	const attachedCount = attachments.length;
	const removedCount = removedAttachments.length;
	const totalCount = attachedCount + removedCount;
	const menu = useMenuStore({ placement: 'top-start', focusLoop: true });
	const open = useStoreState(menu, 'open');
	const [query, setQuery] = useSearchableMenuState(open);
	const [removedOpen, setRemovedOpen] = useState(false);
	const { filteredAttachments, filteredRemovedAttachments } = useMemo(() => {
		const normalized = query.trim().toLocaleLowerCase();
		if (!normalized) {
			return {
				filteredAttachments: attachments,
				filteredRemovedAttachments: removedAttachments,
			};
		}

		const matchesQuery = (attachment: UIAttachment) =>
			[getAttachmentDisplayLabel(attachment), getUIAttachmentPath(attachment)]
				.join('\n')
				.toLocaleLowerCase()
				.includes(normalized);

		return {
			filteredAttachments: attachments.filter(a => {
				return matchesQuery(a);
			}),
			filteredRemovedAttachments: removedAttachments.filter(a => {
				return matchesQuery(a);
			}),
		};
	}, [attachments, query, removedAttachments]);

	const hasQuery = query.trim().length > 0;
	const attachmentSummary = `${attachedCount} added${removedCount > 0 ? ` · ${removedCount} not added` : ''}`;
	const headerMeta = hasQuery
		? `${filteredAttachments.length} added · ${filteredRemovedAttachments.length} not added`
		: attachmentSummary;

	if (totalCount === 0) {
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
					<ActionTriggerChipContent
						className="text-base-content"
						icon={<FiPaperclip size={14} />}
						label="Attachments"
						secondaryLabel={attachmentSummary}
						secondaryLabelClassName="text-base-content/60 shrink-0 whitespace-nowrap text-xs"
						open={open}
					/>
				</MenuButton>

				<button
					type="button"
					className={`${actionTriggerChipRemoveButtonClasses} ml-1`}
					onClick={() => {
						menu.hide();
						onClearAttachments();
					}}
					title="Clear all attachment choices"
					aria-label="Clear all attachment choices"
				>
					<FiX size={14} />
				</button>
			</div>

			<ActionTriggerMenu store={menu}>
				<ActionTriggerMenuHeader>
					<SearchableMenuInput
						open={open}
						query={query}
						onQueryChange={setQuery}
						placeholder="Search added and not-added files and links…"
						onEscape={() => {
							menu.hide();
						}}
					/>
					<div className={actionTriggerMenuHeaderRowClasses}>
						<div className="min-w-0 flex-1 truncate text-xs font-semibold">Attachments</div>
						<span className={actionTriggerMenuMetaClasses}>{headerMeta}</span>
					</div>
				</ActionTriggerMenuHeader>

				{removedCount > 0 ? (
					<ActionTriggerMenuBody className="space-y-2">
						{hasQuery ? (
							<div className="flex flex-wrap gap-2 px-1">
								<button
									type="button"
									className="btn btn-xs rounded-lg"
									disabled={filteredRemovedAttachments.length === 0}
									onClick={() => {
										onRestoreAttachments(filteredRemovedAttachments);
									}}
									title="Attach matching files and links"
								>
									<FiPlus size={12} />
									Add matches ({filteredRemovedAttachments.length})
								</button>
							</div>
						) : null}

						<details
							className="border-base-300 rounded-xl border p-1"
							open={removedOpen}
							onToggle={event => {
								setRemovedOpen(event.currentTarget.open);
							}}
						>
							<summary className="hover:bg-base-200 flex cursor-pointer list-none items-center gap-2 rounded-lg px-2 py-1 text-xs font-semibold">
								{removedOpen ? <FiChevronDown size={13} /> : <FiChevronUp size={13} />}
								<span>Not added</span>
								<span className="text-base-content/60 ml-auto text-[11px]">
									{hasQuery
										? `${filteredRemovedAttachments.length}/${removedCount} shown`
										: `${removedCount} attachment${removedCount === 1 ? '' : 's'}`}
								</span>
							</summary>

							<div className="mt-1 space-y-1">
								{filteredRemovedAttachments.map(attachment => {
									const label = getAttachmentDisplayLabel(attachment);
									const path = getUIAttachmentPath(attachment);
									const title = path && path !== label ? `${label}\n${path}` : path || label;

									return (
										<div
											key={uiAttachmentKey(attachment)}
											className="bg-base-200/60 flex items-center gap-2 rounded-xl px-2 py-1"
											title={title}
										>
											<FiPaperclip className="shrink-0" size={14} />
											<span className="min-w-0 flex-1 truncate text-xs">{label}</span>
											<button
												type="button"
												className="btn btn-ghost btn-xs shrink-0 rounded-lg"
												onClick={() => {
													onRestoreAttachments([attachment]);
												}}
												title="Attach this item again"
												aria-label={`Attach ${label} again`}
											>
												<FiPlus size={13} />
												Add
											</button>
										</div>
									);
								})}

								{filteredRemovedAttachments.length === 0 ? (
									<div className="text-base-content/60 p-2 text-xs">No not-added attachments match this search.</div>
								) : null}
							</div>
						</details>
					</ActionTriggerMenuBody>
				) : null}

				<ComposerDropdownSelectedPanel
					label="Added files and links"
					ariaLabel="Added files and links"
					separatorBefore={removedCount > 0}
					isEmpty={filteredAttachments.length === 0}
					emptyLabel={
						hasQuery
							? 'No added files or links match this search. Clear the search to review all added items.'
							: 'No files or links are currently added. Choose Add on a not-added item above to include it.'
					}
					help="Added attachments are sent with this message. Removed attachments remain available until you send or clear the full set."
					summary={
						attachedCount > 0 ? (
							<span>
								{hasQuery ? `${filteredAttachments.length}/${attachedCount} shown` : `${attachedCount} added`}
							</span>
						) : undefined
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
			</ActionTriggerMenu>
		</div>
	);
}
