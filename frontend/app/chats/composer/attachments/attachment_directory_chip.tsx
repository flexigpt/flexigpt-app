import { useMemo, useState } from 'react';

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

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { UIAttachment } from '@/spec/attachment';
import { AttachmentContentBlockMode } from '@/spec/attachment';

import { AttachmentChip } from '@/chats/composer/attachments/attachment_chip';
import type { DirectoryAttachmentGroup } from '@/chats/composer/attachments/attachment_editor_utils';
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
	const directoryMenu = useMenuStore({ placement: 'bottom-start', focusLoop: true });
	const menuOpen = useStoreState(directoryMenu, 'open');

	const [query, setQuery] = useState('');
	const [removedOpen, setRemovedOpen] = useState(false);

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

	const knownScannedCount = Math.max(group.scannedFileCount, attachedCount + removedFiles.length);

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

	const title = tooltipLines.join('\n');
	const MenuChevron = menuOpen ? FiChevronDown : FiChevronUp;

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
		<div
			className="bg-base-200 hover:bg-base-300/80 text-base-content flex shrink-0 items-center gap-2 rounded-2xl px-2 py-0"
			title={title}
			data-attachment-chip="directory"
		>
			<FiFolder className="shrink-0" size={14} />
			<span className="min-w-0 flex-1 truncate">{group.label}</span>
			<span className="text-base-content/60 whitespace-nowrap">
				{attachedCount} attached
				{removedFiles.length > 0 ? ` · ${removedFiles.length} out` : ''}
				{overflowFileCount > 0 ? ` +${overflowFileCount} more` : ''}
			</span>

			<MenuButton
				store={directoryMenu}
				className="btn btn-ghost btn-xs p-0 shadow-none"
				aria-label={`Show files in folder ${group.dirPath || group.label}`}
				title={group.dirPath || group.label}
			>
				<MenuChevron size={14} />
			</MenuButton>

			<button
				type="button"
				className="btn btn-ghost btn-xs text-error shrink-0 p-0 shadow-none"
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
				gutter={8}
				overflowPadding={8}
				className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-128 w-xl max-w-[calc(100vw-1rem)] min-w-72 overflow-y-auto border p-2 shadow-xl focus-visible:outline-none"
				autoFocusOnShow
			>
				<div className="bg-base-100 sticky top-0 z-10 px-2 py-4">
					<div className="mb-2 flex items-start justify-between gap-3">
						<div className="min-w-0">
							<div className="truncate text-xs font-semibold" title={group.dirPath}>
								Files in “{group.label}”
							</div>
							<div className="text-base-content/60 text-xs">
								{attachedCount} attached · {removedFiles.length} not attached · {knownScannedCount} scanned
								{group.hasMore ? ' · more not scanned' : ''}
							</div>
						</div>
						<span className="text-base-content/60 text-xs whitespace-nowrap">Limit {MAX_FILES_PER_DIRECTORY}</span>
					</div>

					<label
						className={`input input-sm flex w-full items-center gap-2 rounded-xl ${filter.error ? 'input-error' : ''}`}
					>
						<FiSearch className="shrink-0" size={14} aria-hidden="true" />
						<input
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
							className="grow"
							placeholder="Filter: test.go, *_test.go, !*.go, /regex/i"
							aria-label={`Search files in ${group.label}`}
							spellCheck={false}
						/>
						{query && (
							<button
								type="button"
								className="btn btn-ghost btn-xs p-0"
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

					<div className={`mt-1 text-xs ${filter.error ? 'text-error' : 'text-base-content/60'}`}>
						{filter.error ??
							(hasQuery
								? `${matchingAttached.length} attached match · ${matchingRemoved.length} not attached match`
								: 'Text, * and ? globs, ! to invert, or /expression/flags.')}
					</div>

					{hasQuery && !filter.error && (
						<div className="mt-2 flex flex-wrap gap-2">
							<button
								type="button"
								className="btn btn-error btn-xs rounded-lg"
								disabled={matchingAttached.length === 0}
								onClick={() => {
									removeFromDirectory(matchingAttached);
								}}
							>
								Remove matches ({matchingAttached.length})
							</button>

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
					)}
				</div>

				<div className="text-base-content/70 mb-1 text-xs font-semibold">
					Attached ({hasQuery ? matchingAttached.length : attachedCount})
				</div>

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

				{matchingAttached.length === 0 && (
					<div className="text-base-content/60 rounded-xl px-2 py-3 text-center text-xs">
						{hasQuery ? 'No attached files match this filter.' : 'No files are currently attached.'}
					</div>
				)}

				{removedFiles.length > 0 && (
					<details
						className="border-base-300 mt-2 border-t pt-2"
						open={removedOpen}
						onToggle={event => {
							setRemovedOpen(event.currentTarget.open);
						}}
					>
						<summary className="hover:bg-base-200 flex cursor-pointer list-none items-center gap-2 rounded-lg px-2 py-1 text-xs font-semibold">
							{removedOpen ? <FiChevronDown size={13} /> : <FiChevronUp size={13} />}
							<span>Not attached ({removedFiles.length})</span>
							{hasQuery && <span className="text-base-content/60 ml-auto">{matchingRemoved.length} match</span>}
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
											title={canRestore ? 'Attach this file again' : 'The 128-file limit is full'}
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
					<div className="border-base-300 mt-2 space-y-1 border-t pt-2">
						<div className="text-base-content/70 text-xs font-semibold">Not scanned or unreadable</div>

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
					<div className="text-base-content/70 text-xs">No readable files could be attached from this folder.</div>
				)}
			</Menu>
		</div>
	);
}
