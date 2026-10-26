import { useEffect, useMemo, useRef, useState } from 'react';
import {
	FiAlertTriangle,
	FiChevronDown,
	FiChevronUp,
	FiFileText,
	FiFolder,
	FiPlus,
	FiSearch,
	FiX,
} from 'react-icons/fi';

import { MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { UIAttachment } from '@/spec/attachment';
import { AttachmentContentBlockMode } from '@/spec/attachment';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerChipRemoveButtonClasses,
	ActionTriggerMenu,
	ActionTriggerMenuBody,
	ActionTriggerMenuHeader,
	actionTriggerMenuHeaderRowClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
import { useSearchableMenuState } from '@/components/searchmenu/searchable_menu_utils';

import type { DirectoryAttachmentGroup } from '@/chats/composer/attachments/attachment_editor_utils';
import { AttachmentChip } from '@/chats/composer/attachments/attachment_chip';
import {
	getAttachmentDisplayLabel,
	getUIAttachmentPath,
	MAX_FILES_PER_DIRECTORY,
	uiAttachmentKey,
} from '@/chats/composer/attachments/attachment_editor_utils';
import { getAttachmentContentBlockModePillClasses } from '@/chats/composer/attachments/attachment_mode_menu_utils';

interface DirectoryFilter {
	error?: string;
	matches: (value: string) => boolean;
}

interface DirectoryChipProps {
	group: DirectoryAttachmentGroup;
	attachments: UIAttachment[];
	onRemoveDirectoryAttachments: (groupId: string, attachments: UIAttachment[]) => void;
	onRestoreDirectoryAttachments: (groupId: string, attachments: UIAttachment[]) => void;
	onChangeAttachmentContentBlockMode: (att: UIAttachment, mode: AttachmentContentBlockMode) => void;
	onRemoveDirectoryGroup: (groupId: string) => void;
	onRemoveOverflowDir?: (groupId: string, dirPath: string) => void;
}

function escapeRegExp(value: string): string {
	return value.replaceAll(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Supported query syntax:
 *
 * - test.go            Case-insensitive substring match
 * - *_test.go          Case-insensitive glob match
 * - !*.go              Invert a plain-text, glob, or regex match
 * - /_test\.go$/i      JavaScript regular expression
 */
function buildDirectoryFilter(rawQuery: string): DirectoryFilter {
	let query = rawQuery.trim();
	if (!query) {
		return { matches: () => true };
	}

	let inverted = false;
	if (query.startsWith('!')) {
		inverted = true;
		query = query.slice(1).trim();
	}

	if (!query) {
		return {
			error: 'Enter a pattern after !.',
			matches: () => false,
		};
	}

	let matcher: (value: string) => boolean;

	if (query.startsWith('/')) {
		const finalSlash = query.lastIndexOf('/');

		if (finalSlash <= 0) {
			return {
				error: 'Regex patterns must end with /. For example: /_test\\.go$/',
				matches: () => false,
			};
		}

		const source = query.slice(1, finalSlash);
		const flags = query.slice(finalSlash + 1);

		if (!/^[imsu]*$/.test(flags)) {
			return {
				error: 'Supported regex flags are i, m, s, and u.',
				matches: () => false,
			};
		}

		try {
			const expression = new RegExp(source, flags);
			matcher = value => expression.test(value);
		} catch (error) {
			return {
				error: error instanceof Error ? error.message : 'Invalid regular expression.',
				matches: () => false,
			};
		}
	} else if (query.includes('*') || query.includes('?')) {
		const source = escapeRegExp(query).replaceAll('\\*', '.*').replaceAll('\\?', '.');
		const expression = new RegExp(`^${source}$`, 'i');
		matcher = value => expression.test(value);
	} else {
		const normalizedQuery = query.toLocaleLowerCase();
		matcher = value => value.toLocaleLowerCase().includes(normalizedQuery);
	}

	return {
		matches: value => (inverted ? !matcher(value) : matcher(value)),
	};
}

function getRelativeAttachmentPath(group: DirectoryAttachmentGroup, attachment: UIAttachment): string {
	const fullPath = getUIAttachmentPath(attachment).replaceAll('\\', '/');
	const directoryPath = group.dirPath.replaceAll('\\', '/').replace(/\/+$/, '');

	if (fullPath.toLocaleLowerCase().startsWith(`${directoryPath.toLocaleLowerCase()}/`)) {
		return fullPath.slice(directoryPath.length + 1);
	}

	return fullPath || getAttachmentDisplayLabel(attachment);
}

/**
 * Directory pill with a searchable menu of attached and not-attached files.
 *
 * The backend scans up to MAX_DIRECTORY_FILES_TO_SCAN candidates, but this UI
 * keeps only MAX_FILES_PER_DIRECTORY files actively attached. The remainder
 * are retained in group.removedAttachments so users can filter and restore
 * them without re-opening the directory picker.
 */
export function DirectoryChip({
	group,
	attachments,
	onRemoveDirectoryAttachments,
	onRestoreDirectoryAttachments,
	onChangeAttachmentContentBlockMode,
	onRemoveDirectoryGroup,
	onRemoveOverflowDir,
}: DirectoryChipProps) {
	const directoryMenu = useMenuStore({ placement: 'top-start', focusLoop: true });
	const menuOpen = useStoreState(directoryMenu, 'open');

	const [query, setQuery] = useSearchableMenuState(menuOpen);
	const [removedOpen, setRemovedOpen] = useState(false);
	const filterInputRef = useRef<HTMLInputElement | null>(null);

	useEffect(() => {
		if (!menuOpen || typeof window === 'undefined') {
			return;
		}

		const frame = window.requestAnimationFrame(() => {
			filterInputRef.current?.focus({ preventScroll: true });
		});

		return () => {
			window.cancelAnimationFrame(frame);
		};
	}, [menuOpen]);

	const attachmentByKey = new Map<string, UIAttachment>();
	for (const attachment of attachments) {
		attachmentByKey.set(uiAttachmentKey(attachment), attachment);
	}

	const attachedFiles = group.attachmentKeys.flatMap(key => {
		const attachment = attachmentByKey.get(key);
		return attachment ? [attachment] : [];
	});

	const removedFiles = group.removedAttachments ?? [];
	const attachedCount = attachedFiles.length;
	const filter = useMemo(() => buildDirectoryFilter(query), [query]);
	const hasQuery = query.trim().length > 0;

	const matchingAttached = filter.error
		? []
		: attachedFiles.filter(attachment => filter.matches(getRelativeAttachmentPath(group, attachment)));

	const matchingRemoved = filter.error
		? []
		: removedFiles.filter(attachment => filter.matches(getRelativeAttachmentPath(group, attachment)));

	const availableSlots = Math.max(0, MAX_FILES_PER_DIRECTORY - attachedCount);
	const restorableMatches = matchingRemoved.slice(0, availableSlots);
	const overflowFileCount = group.overflowDirs.reduce((sum, overflow) => sum + (overflow.fileCount ?? 0), 0);

	const hasDirectoryMenuBodyContent =
		(hasQuery && !filter.error) ||
		removedFiles.length > 0 ||
		group.overflowDirs.length > 0 ||
		(attachedCount === 0 && removedFiles.length === 0 && group.overflowDirs.length === 0);

	const directoryHeaderMeta = filter.error
		? 'Invalid filter'
		: hasQuery
			? `${matchingAttached.length} added match${matchingAttached.length === 1 ? '' : 'es'} · ${
					matchingRemoved.length
				} not-added match${matchingRemoved.length === 1 ? '' : 'es'}`
			: `${attachedCount} added · ${removedFiles.length} not added`;

	const tooltipLines: string[] = [];
	if (group.dirPath && group.dirPath !== group.label) {
		tooltipLines.push(group.dirPath);
	}
	tooltipLines.push(`${attachedCount} file${attachedCount === 1 ? '' : 's'} attached`);
	if (removedFiles.length > 0) {
		tooltipLines.push(`${removedFiles.length} file${removedFiles.length === 1 ? '' : 's'} not attached`);
	}
	if (overflowFileCount > 0) {
		tooltipLines.push(
			`${overflowFileCount} additional file${overflowFileCount === 1 ? '' : 's'} not scanned from subfolders`
		);
	}

	const directoryChipSummary = `${attachedCount} added${
		removedFiles.length > 0 ? ` · ${removedFiles.length} not added` : ''
	}${overflowFileCount > 0 ? ` +${overflowFileCount} more` : ''}`;

	const title = tooltipLines.join('\n');

	const removeFromDirectory = (files: UIAttachment[]) => {
		if (files.length === 0) {
			return;
		}
		onRemoveDirectoryAttachments(group.id, files);
	};

	const restoreToDirectory = (files: UIAttachment[]) => {
		if (files.length === 0) {
			return;
		}
		onRestoreDirectoryAttachments(group.id, files);
	};

	return (
		<div className="relative shrink-0" data-attachment-chip="directory">
			<div className="flex items-center gap-1">
				<MenuButton
					store={directoryMenu}
					className={`${actionTriggerChipButtonClasses} ${
						menuOpen ? 'border-base-300 bg-base-300/60' : 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
					}`}
					aria-label={`Show files in folder ${group.dirPath || group.label}`}
					title={title}
				>
					<ActionTriggerChipContent
						className="text-base-content"
						icon={<FiFolder size={14} />}
						label={group.label}
						secondaryLabel={directoryChipSummary}
						open={menuOpen}
					/>
				</MenuButton>

				<button
					type="button"
					className={actionTriggerChipRemoveButtonClasses}
					onClick={() => {
						onRemoveDirectoryGroup(group.id);
					}}
					title="Remove folder and all attached files"
					aria-label={`Remove folder ${group.label} and all attached files`}
				>
					<FiX size={14} />
				</button>
			</div>

			<ActionTriggerMenu store={directoryMenu}>
				<ActionTriggerMenuHeader>
					<label className="relative block">
						<FiSearch
							size={12}
							className="text-base-content/50 pointer-events-none absolute top-1/2 left-2 -translate-y-1/2"
						/>
						<input
							ref={filterInputRef}
							type="search"
							value={query}
							onChange={event => {
								setQuery(event.target.value);
							}}
							onKeyDown={event => {
								if (event.key !== 'Escape') {
									event.stopPropagation();
								}
							}}
							className={`input input-xs bg-base-200 border-base-300 h-7 w-full rounded-xl pr-8 pl-7 text-xs outline-none ${
								filter.error ? 'input-error' : ''
							}`}
							placeholder="Filter: test.go, *_test.go, !*.go, /regex/i"
							aria-label={`Search files in ${group.label}`}
							autoComplete="off"
							autoCorrect="off"
							spellCheck={false}
						/>
						{query && (
							<button
								type="button"
								className="btn btn-ghost btn-xs absolute top-1/2 right-1 size-5 min-h-0 -translate-y-1/2 rounded-full p-0"
								onClick={() => {
									setQuery('');
								}}
								aria-label="Clear file filter"
								title="Clear filter"
							>
								<FiX size={13} />
							</button>
						)}
					</label>

					<div className={actionTriggerMenuHeaderRowClasses}>
						<div className="min-w-0 flex-1 truncate text-xs font-semibold" title={group.dirPath}>
							Files in “{group.label}”
						</div>
						<span
							className={`shrink-0 text-[11px] whitespace-nowrap ${
								filter.error ? 'text-error' : 'text-base-content/60'
							}`}
						>
							{directoryHeaderMeta}
						</span>
					</div>

					{filter.error ? <div className="alert alert-error rounded-xl p-2 text-xs">{filter.error}</div> : null}
				</ActionTriggerMenuHeader>

				<ActionTriggerMenuBody className="space-y-2">
					{hasQuery && !filter.error ? (
						<div className="flex flex-wrap gap-2 px-1">
							<button
								type="button"
								className="btn btn-xs rounded-lg"
								disabled={restorableMatches.length === 0}
								onClick={() => {
									restoreToDirectory(restorableMatches);
								}}
								title={
									matchingRemoved.length > availableSlots
										? `Only ${availableSlots} attachment slots are available`
										: 'Attach matching files'
								}
							>
								<FiPlus size={12} />
								Add matches ({restorableMatches.length})
							</button>
						</div>
					) : null}

					{removedFiles.length > 0 && (
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
									{hasQuery ? `${matchingRemoved.length}/${removedFiles.length} shown` : `${removedFiles.length} files`}
								</span>
							</summary>

							<div className="mt-1 space-y-1">
								{matchingRemoved.map(attachment => {
									const key = uiAttachmentKey(attachment);
									const relativePath = getRelativeAttachmentPath(group, attachment);
									const canRestore = attachedCount < MAX_FILES_PER_DIRECTORY;

									return (
										<div
											key={key}
											className="bg-base-200/60 flex items-center gap-2 rounded-xl px-2 py-1"
											title={getUIAttachmentPath(attachment)}
										>
											<FiFileText className="shrink-0" size={14} />
											<span className="min-w-0 flex-1 truncate text-xs">{relativePath}</span>
											<button
												type="button"
												className="btn btn-ghost btn-xs shrink-0 rounded-lg"
												disabled={!canRestore}
												onClick={() => {
													restoreToDirectory([attachment]);
												}}
												title={canRestore ? 'Attach this file again' : 'The file limit is full'}
												aria-label={`Attach ${relativePath} again`}
											>
												<FiPlus size={13} />
												Add
											</button>
										</div>
									);
								})}

								{matchingRemoved.length === 0 && (
									<div className="text-base-content/60 p-2 text-xs">No not-attached files match this filter.</div>
								)}
							</div>
						</details>
					)}

					{group.overflowDirs.length > 0 && (
						<div className="border-base-300 rounded-xl border p-2">
							<div className="text-base-content/70 mb-1 text-xs font-semibold">Not scanned or unreadable</div>

							{group.overflowDirs.map(overflow => {
								const relativePath = overflow.relativePath || overflow.dirPath;

								return (
									<MenuItem
										key={overflow.dirPath}
										store={directoryMenu}
										hideOnClick={false}
										className="bg-base-200/60 data-active-item:bg-base-200 flex items-start gap-2 rounded-xl px-2 py-1"
									>
										<FiAlertTriangle size={14} className="text-warning mt-0.5 shrink-0" />
										<div className="min-w-0 flex-1">
											<div
												className="truncate text-xs font-medium"
												title={
													overflow.dirPath !== relativePath ? `${relativePath}\n${overflow.dirPath}` : overflow.dirPath
												}
											>
												{relativePath}
											</div>
											<div className="text-base-content/70 truncate text-xs">
												{overflow.fileCount} item{overflow.fileCount === 1 ? '' : 's'} not attached in this folder
												{overflow.partial ? ' (folder only partially scanned)' : ''}
											</div>
										</div>

										<span
											className={getAttachmentContentBlockModePillClasses(
												AttachmentContentBlockMode.notReadable,
												false
											)}
											title="Too many files in this subfolder; they were not attached."
										>
											Not readable
										</span>

										{onRemoveOverflowDir && (
											<button
												type="button"
												className="btn btn-ghost btn-xs text-error shrink-0 px-1 py-0 shadow-none"
												onClick={() => {
													onRemoveOverflowDir(group.id, overflow.dirPath);
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

					{attachedCount === 0 && removedFiles.length === 0 && group.overflowDirs.length === 0 && (
						<div className="border-base-300/70 bg-base-200/40 text-base-content/60 rounded-xl border border-dashed p-2 text-xs">
							No readable files could be attached from this folder.
						</div>
					)}
				</ActionTriggerMenuBody>

				<ComposerDropdownSelectedPanel
					label="Added files"
					ariaLabel={`Added files from ${group.label}`}
					separatorBefore={hasDirectoryMenuBodyContent}
					isEmpty={attachedCount === 0}
					emptyLabel="Choose Add on a not-added file above to include it from this folder."
					help={`Filter supports text, * and ? globs, ! to invert, and /expression/flags. Up to ${MAX_FILES_PER_DIRECTORY} files can be added from this folder.`}
					summary={
						attachedCount > 0 ? (
							<span>
								{hasQuery && !filter.error
									? `${matchingAttached.length}/${attachedCount} shown`
									: `${attachedCount} added`}
							</span>
						) : undefined
					}
					actions={
						hasQuery && !filter.error ? (
							<button
								type="button"
								className="btn btn-ghost btn-xs text-error rounded-lg"
								disabled={matchingAttached.length === 0}
								onClick={() => {
									removeFromDirectory(matchingAttached);
								}}
							>
								Remove matches ({matchingAttached.length})
							</button>
						) : null
					}
				>
					{matchingAttached.map(attachment => (
						<MenuItem
							key={uiAttachmentKey(attachment)}
							store={directoryMenu}
							hideOnClick={false}
							className="data-active-item:bg-base-200 mb-1 rounded-xl last:mb-0"
						>
							<AttachmentChip
								attachment={attachment}
								onRemoveAttachment={() => {
									removeFromDirectory([attachment]);
								}}
								onChangeAttachmentContentBlockMode={onChangeAttachmentContentBlockMode}
								fullWidth
							/>
						</MenuItem>
					))}

					{matchingAttached.length === 0 ? (
						<div className="text-base-content/60 rounded-xl px-2 py-3 text-center text-xs">
							{hasQuery ? 'No attached files match this filter.' : 'No files are currently attached.'}
						</div>
					) : null}
				</ComposerDropdownSelectedPanel>
			</ActionTriggerMenu>
		</div>
	);
}
