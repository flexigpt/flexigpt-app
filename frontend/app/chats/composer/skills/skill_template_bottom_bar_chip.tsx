import type { RefObject, SubmitEventHandler } from 'react';
import { memo, useCallback, useEffect, useMemo, useState } from 'react';

import { FiAlertCircle, FiFile, FiFilePlus, FiPaperclip } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { SkillArgument, SkillListItem } from '@/spec/skill';
import { SkillType } from '@/spec/skill';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import { skillStoreAPI } from '@/apis/baseapi';

import {
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { GroupedMenuSection } from '@/components/grouped_menu_sections';
import { HoverTip, HoverTipContent } from '@/components/hover_tip';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';
import { searchableMenuEmptyStateClasses, SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

import { skillRefFromListItem, skillRefKey } from '@/skills/lib/skill_identity_utils';

interface SkillTemplateInsertArgs {
	text: string;
	attachedSkillPaths?: string[];
}

interface SkillTemplateGroup {
	bundleID: string;
	bundleSlug: string;
	options: SkillListItem[];
}

interface SkillTemplateDropdownProps {
	store: MenuStore;
	open: boolean;
	loading: boolean;
	items: SkillListItem[];
	loadError?: string | null;
	actionError?: string | null;
	onRetry?: () => void;
	onPick: (item: SkillListItem) => void;
}

interface SkillTemplateInsertResult {
	attachmentError?: string;
}

interface SkillTemplateRenderModalProps {
	isOpen: boolean;
	item: SkillListItem | null;
	onClose: () => void;
	onInsert: (
		args: SkillTemplateInsertArgs
	) => Promise<SkillTemplateInsertResult | undefined> | SkillTemplateInsertResult | undefined;
}

interface RenderFormState {
	arguments: Record<string, string>;
}

export interface SkillTemplateBottomBarChipProps {
	store: MenuStore;
	buttonRef: RefObject<HTMLButtonElement | null>;
	shortcut?: string;
	onInsertTemplateText: (text: string) => Promise<void> | void;
	onAttachResourcePaths?: (paths: string[]) => Promise<void> | void;
	isInputLocked?: boolean;
}

const skillTemplateCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function getSkillTemplateLabel(item: SkillListItem): string {
	return (
		item.skillDefinition.displayName?.trim() ||
		item.skillDefinition.name?.trim() ||
		item.skillDefinition.slug?.trim() ||
		item.skillSlug
	);
}

function skillTemplateKey(item: SkillListItem): string {
	return skillRefKey(skillRefFromListItem(item));
}

function compareSkillTemplateListItems(a: SkillListItem, b: SkillListItem): number {
	const bundleSlugCompare = skillTemplateCollator.compare(a.bundleSlug, b.bundleSlug);
	if (bundleSlugCompare !== 0) {
		return bundleSlugCompare;
	}

	const labelCompare = skillTemplateCollator.compare(getSkillTemplateLabel(a), getSkillTemplateLabel(b));
	if (labelCompare !== 0) {
		return labelCompare;
	}

	const slugCompare = skillTemplateCollator.compare(a.skillSlug, b.skillSlug);
	if (slugCompare !== 0) {
		return slugCompare;
	}

	return skillTemplateCollator.compare(skillTemplateKey(a), skillTemplateKey(b));
}

function groupSkillTemplates(items: SkillListItem[]): SkillTemplateGroup[] {
	const groupsByBundle = new Map<string, SkillTemplateGroup>();

	for (const item of [...items].toSorted(compareSkillTemplateListItems)) {
		const groupKey = item.bundleID || item.bundleSlug;
		let group = groupsByBundle.get(groupKey);

		if (!group) {
			group = {
				bundleID: item.bundleID,
				bundleSlug: item.bundleSlug || item.bundleID,
				options: [],
			};
			groupsByBundle.set(groupKey, group);
		}

		group.options.push(item);
	}

	return [...groupsByBundle.values()];
}

function getTemplateArguments(item: SkillListItem | null): SkillArgument[] {
	return item?.skillDefinition.arguments ?? [];
}

function getDefaultArgumentValues(args: SkillArgument[]): Record<string, string> {
	return Object.fromEntries(args.map(arg => [arg.name, arg.default ?? ''] as const));
}

function hasResources(item: SkillListItem | null): boolean {
	const resources = item?.skillDefinition.resources;
	return resources?.hasResources || (resources?.totalCount ?? 0) > 0;
}

function isAbsoluteLocalPath(path: string): boolean {
	const value = path.trim();
	return value.startsWith('/') || value.startsWith('\\\\') || /^[A-Za-z]:[\\/]/.test(value);
}

function isSkillMarkdownPath(path: string): boolean {
	const normalized = path.trim().replaceAll('\\', '/').replaceAll(/\/+$/g, '');
	const basename = normalized.slice(normalized.lastIndexOf('/') + 1);
	return basename.toLowerCase() === 'skill.md';
}

function normalizeLocalPath(path: string): string {
	return path.trim().replaceAll('\\', '/').replaceAll(/\/+/g, '/').replaceAll(/\/+$/g, '');
}

function hasParentTraversal(path: string): boolean {
	return normalizeLocalPath(path)
		.split('/')
		.some(segment => segment === '..');
}

function isPathInsideSkillDirectory(skillDirectory: string, candidate: string): boolean {
	const base = normalizeLocalPath(skillDirectory);
	const value = normalizeLocalPath(candidate);

	if (!base || !value) {
		return false;
	}

	const caseInsensitive = /^[A-Za-z]:\//.test(base);
	const normalizedBase = caseInsensitive ? base.toLowerCase() : base;
	const normalizedValue = caseInsensitive ? value.toLowerCase() : value;
	return normalizedValue === normalizedBase || normalizedValue.startsWith(`${normalizedBase}/`);
}

function joinLocalResourcePath(skillDirectory: string, resourceLocation: string): string {
	const base = skillDirectory.trim().replaceAll(/[\\/]+$/g, '');
	const relative = resourceLocation
		.trim()
		.replace(/^\.[\\/]/, '')
		.replaceAll(/^[\\/]+/g, '');

	if (!base) {
		return relative;
	}
	if (!relative) {
		return base;
	}

	const separator = base.includes('\\') && !base.includes('/') ? '\\' : '/';
	return `${base}${separator}${relative}`;
}

function getSkillAttachmentPaths(item: SkillListItem | null): string[] {
	if (!item || !hasResources(item)) {
		return [];
	}

	if (item.skillDefinition.type !== SkillType.FS) {
		return [];
	}

	const skillDirectory = item.skillDefinition.location?.trim();
	if (!skillDirectory || !isAbsoluteLocalPath(skillDirectory) || hasParentTraversal(skillDirectory)) {
		return [];
	}

	const resourceLocations = item.skillDefinition.resources?.locations ?? [];
	const paths = resourceLocations
		.map(location => location.trim())
		.filter(Boolean)
		.map(location => {
			if (hasParentTraversal(location)) {
				return '';
			}

			if (isAbsoluteLocalPath(location)) {
				return isPathInsideSkillDirectory(skillDirectory, location) ? location : '';
			}

			const joined = joinLocalResourcePath(skillDirectory, location);
			return isPathInsideSkillDirectory(skillDirectory, joined) ? joined : '';
		})
		.filter(path => path.length > 0 && !isSkillMarkdownPath(path));

	return [...new Set(paths)];
}

async function collectUserMessageSkillTemplates(): Promise<SkillListItem[]> {
	const out: SkillListItem[] = [];
	let pageToken: string | undefined = undefined;

	for (let guard = 0; guard < 50; guard += 1) {
		const resp = await skillStoreAPI.listSkills({
			bundleIDs: [],
			types: [],
			inserts: ['user-message'],
			includeDisabled: false,
			includeMissing: false,
			recommendedPageSize: 200,
			pageToken,
		});

		out.push(...(resp.skillListItems ?? []));
		pageToken = resp.nextPageToken;
		if (!pageToken) {
			break;
		}
	}

	return out.toSorted(compareSkillTemplateListItems);
}

function useUserMessageSkillTemplates(open: boolean) {
	const [items, setItems] = useState<SkillListItem[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const refresh = useCallback(async () => {
		setLoading(true);
		setError(null);
		try {
			const next = await collectUserMessageSkillTemplates();
			setItems(next);
		} catch (loadError) {
			console.error('Failed to load user-message skill templates:', loadError);
			setItems([]);
			setError(loadError instanceof Error ? loadError.message : 'Failed to load user-message skill templates.');
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		if (!open) {
			return;
		}
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void refresh();
	}, [open, refresh]);

	return { items, loading, error, refresh };
}

function renderSkillTemplateMenuItem(item: SkillListItem, onPick: (item: SkillListItem) => void) {
	const args = item.skillDefinition.arguments ?? [];
	const resources = item.skillDefinition.resources;
	const resourceCount = resources?.hasResources ? resources.totalCount : 0;

	return (
		<MenuItem
			key={skillTemplateKey(item)}
			data-searchable-menu-item="true"
			hideOnClick={false}
			onClick={() => {
				onPick(item);
			}}
			className={`${actionTriggerMenuItemClasses} items-start`}
			title={`${item.bundleSlug}/${item.skillSlug}\n${item.skillDefinition.description ?? ''}`}
		>
			<FiFilePlus size={14} className="text-warning mt-0.5 shrink-0" />

			<div className="min-w-0 flex-1">
				<div className="truncate text-xs font-medium">{getSkillTemplateLabel(item)}</div>
				<div className="mt-1 flex min-w-0 items-center gap-2 text-[10px] opacity-70">
					<span className="truncate">{item.skillSlug}</span>
					<span>•</span>
					<span>{item.skillDefinition.name}</span>
				</div>
			</div>

			<div className="ml-auto flex shrink-0 items-center gap-1">
				{args.length > 0 ? (
					<span className="badge badge-warning badge-xs">
						{args.length} arg{args.length === 1 ? '' : 's'}
					</span>
				) : null}
				{resourceCount > 0 ? (
					<span
						className="badge badge-info badge-xs"
						title="This template references resources. You can attach those files when inserting the template."
					>
						<FiPaperclip size={10} />
						<span className="ml-1">{resourceCount}</span>
					</span>
				) : null}
				<span className="badge badge-ghost badge-xs">{item.isBuiltIn ? 'built-in' : 'custom'}</span>
			</div>
		</MenuItem>
	);
}

function SkillTemplateDropdown({
	store,
	open,
	loading,
	items,
	loadError,
	actionError,
	onRetry,
	onPick,
}: SkillTemplateDropdownProps) {
	const menuContentElement = useStoreState(store, 'contentElement');
	const [searchQuery, setSearchQuery] = useSearchableMenuState(open);

	const displayedItems = useMemo(() => {
		if (!isSearchQueryActive(searchQuery)) {
			return items;
		}

		return rankSearchableItems(items, {
			query: searchQuery,
			getKey: skillTemplateKey,
			getFields: item => [
				{ value: getSkillTemplateLabel(item), weight: 7 },
				{ value: item.skillSlug, weight: 6 },
				{ value: item.skillDefinition.name, weight: 5 },
				{ value: item.bundleSlug, weight: 4 },
				{ value: item.skillDefinition.description, weight: 3 },
				{ value: item.skillDefinition.location, weight: 2 },
				...(item.skillDefinition.tags ?? []).map(tag => ({ value: tag, weight: 2 })),
				...(item.skillDefinition.arguments ?? []).flatMap(arg => [
					{ value: arg.name, weight: 2 },
					{ value: arg.description, weight: 1 },
					{ value: arg.default, weight: 1 },
				]),
			],
			fallbackCompare: compareSkillTemplateListItems,
		});
	}, [items, searchQuery]);

	const groupedTemplates = useMemo(() => groupSkillTemplates(displayedItems), [displayedItems]);
	const firstTemplate = displayedItems[0] ?? null;

	return (
		<Menu
			store={store}
			gutter={8}
			overflowPadding={8}
			portal
			className={actionTriggerMenuWideClasses}
			data-menu-kind="templates"
			autoFocusOnShow={false}
		>
			{!open ? null : (
				<>
					<div className="mb-2 flex items-center justify-between gap-2 px-1">
						<div className="text-base-content/70 text-xs font-semibold">Templates</div>
						<span className="badge badge-ghost badge-xs">Available {items.length}</span>
					</div>
					<div className="text-base-content/70 mb-2 px-1 text-xs">
						Templates are skills with <span className="font-mono">insert: user-message</span>. They render into plain
						composer text and do not remain selected after insertion.
					</div>
					{loadError ? (
						<div className="alert alert-warning mb-2 rounded-xl text-xs">
							<FiAlertCircle size={14} className="shrink-0" />
							<div className="grow">
								<div className="font-semibold">Templates could not be refreshed</div>
								<div>{loadError}</div>
							</div>
							{onRetry ? (
								<button type="button" className="btn btn-xs rounded-lg" onClick={onRetry}>
									Retry
								</button>
							) : null}
						</div>
					) : null}
					{actionError ? (
						<div className="alert alert-error mb-2 rounded-xl text-xs">
							<FiAlertCircle size={14} className="shrink-0" />
							<div className="grow">{actionError}</div>
						</div>
					) : null}
					<SearchableMenuInput
						open={open}
						query={searchQuery}
						onQueryChange={setSearchQuery}
						placeholder="Search templates…"
						resultCount={displayedItems.length}
						totalCount={items.length}
						disabled={loading || items.length === 0}
						onFocusFirstItem={() => {
							focusFirstSearchableMenuItem(menuContentElement);
						}}
						onEnterFirstResult={() => {
							if (firstTemplate) {
								onPick(firstTemplate);
							}
						}}
						onEscape={() => {
							store.hide();
						}}
					/>
				</>
			)}

			{!open ? null : loading ? (
				<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>Loading templates…</div>
			) : items.length === 0 ? (
				<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
					No user-message skill templates available
				</div>
			) : displayedItems.length === 0 ? (
				<div className={searchableMenuEmptyStateClasses}>No templates match your search.</div>
			) : (
				<div className="space-y-2">
					{groupedTemplates.map((group, groupIndex) => (
						<GroupedMenuSection
							key={group.bundleID || group.bundleSlug}
							title={group.bundleSlug}
							ariaLabel={`${group.bundleSlug} user-message skill templates`}
							separatorBefore={groupIndex > 0}
							meta={<span className="badge badge-ghost badge-xs">{group.options.length}</span>}
						>
							{group.options.filter(item => !hasResources(item)).map(item => renderSkillTemplateMenuItem(item, onPick))}

							{group.options.some(item => hasResources(item)) ? (
								<div className="border-base-300 text-base-content/60 my-2 border-t pt-2 text-xs">
									Templates with resource references
								</div>
							) : null}

							{group.options.filter(item => hasResources(item)).map(item => renderSkillTemplateMenuItem(item, onPick))}
						</GroupedMenuSection>
					))}
				</div>
			)}
		</Menu>
	);
}

function SkillTemplateRenderModalContent({
	item,
	onInsert,
}: Omit<SkillTemplateRenderModalProps, 'isOpen' | 'onClose'>) {
	const args = useMemo(() => getTemplateArguments(item), [item]);
	const skillAttachmentPaths = useMemo(() => getSkillAttachmentPaths(item), [item]);
	const [formState, setFormState] = useState<RenderFormState>(() => ({
		arguments: getDefaultArgumentValues(args),
	}));
	const [submitError, setSubmitError] = useState('');
	const [submitting, setSubmitting] = useState(false);
	const [insertedWithAttachmentWarning, setInsertedWithAttachmentWarning] = useState(false);
	const { requestClose, unmountingRef } = useModalDialogController();
	const [selectedSkillAttachmentPaths, setSelectedSkillAttachmentPaths] = useState<Set<string>>(
		() => new Set(skillAttachmentPaths)
	);

	const selectedAttachmentPaths = skillAttachmentPaths.filter(path => selectedSkillAttachmentPaths.has(path));
	const canInsert = !submitting;

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (!item || !canInsert || insertedWithAttachmentWarning) {
			return;
		}

		setSubmitting(true);
		setSubmitError('');

		void skillStoreAPI
			.renderSkill(skillRefFromListItem(item), formState.arguments)
			.then(async rendered => {
				if (rendered.insert !== 'user-message') {
					throw new Error(`Expected a user-message template, but renderer returned insert=${rendered.insert}.`);
				}
				const result = await onInsert({
					text: rendered.text,
					attachedSkillPaths: selectedAttachmentPaths.length > 0 ? selectedAttachmentPaths : undefined,
				});

				if (unmountingRef.current) {
					return;
				}

				if (result?.attachmentError) {
					setInsertedWithAttachmentWarning(true);
					setSubmitError(result.attachmentError);
					return;
				}

				requestClose(true);
			})
			.catch((error: unknown) => {
				if (!unmountingRef.current) {
					setSubmitError(error instanceof Error ? error.message : 'Failed to render template.');
				}
			})
			.finally(() => {
				if (!unmountingRef.current) {
					setSubmitting(false);
				}
			});
	};

	if (!item) {
		return null;
	}

	const resourceCount = item.skillDefinition.resources?.totalCount ?? 0;

	return (
		<div className="modal-box bg-base-200 max-h-[80vh] max-w-2xl overflow-hidden rounded-2xl p-0">
			<div className="max-h-[80vh] overflow-y-auto p-6">
				<ModalHeader
					title="Use Template"
					description={`${getSkillTemplateLabel(item)} renders into plain composer text.`}
					onClose={() => {
						requestClose();
					}}
					closeDisabled={submitting}
				/>

				<form className="space-y-4" onSubmit={handleSubmit}>
					{submitError ? (
						<div className="alert alert-error rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<span>{submitError}</span>
						</div>
					) : null}

					{args.length > 0 ? (
						<div className="space-y-3">
							<div className="text-sm font-semibold">Arguments</div>
							<div className="grid grid-cols-1 gap-3 md:grid-cols-2">
								{args.map(arg => (
									<div key={arg.name} className="border-base-content/10 rounded-2xl border p-3">
										<label className="text-sm font-medium">{arg.name}</label>
										{arg.description ? (
											<div className="text-base-content/70 mt-1 text-xs">{arg.description}</div>
										) : null}
										<input
											className="input bg-base-100 mt-3 w-full rounded-xl"
											value={formState.arguments[arg.name] ?? ''}
											onChange={event => {
												const value = event.target.value;
												setFormState(prev => ({
													...prev,
													arguments: {
														...prev.arguments,
														[arg.name]: value,
													},
												}));
											}}
											placeholder={arg.default ?? ''}
											spellCheck="false"
										/>
										<div className="text-base-content/60 mt-1 text-xs">
											Blank values are allowed and are passed to the skill renderer as empty strings.
										</div>
									</div>
								))}
							</div>
						</div>
					) : (
						<div className="text-base-content/70 text-sm">This template has no arguments.</div>
					)}

					{hasResources(item) ? (
						<div className="border-base-content/10 bg-base-100 rounded-2xl border p-3">
							<div className="font-medium">Optional resource attachments</div>
							<div className="text-base-content/70 mt-1 text-xs">
								The rendered template body is inserted into the composer. Select only the indexed resource files that
								should accompany the user message. SKILL.md is not attached again.
							</div>

							{skillAttachmentPaths.length > 0 ? (
								<div className="mt-2 max-h-40 space-y-1 overflow-auto">
									{skillAttachmentPaths.map(path => (
										<label key={path} className="bg-base-200 flex cursor-pointer items-start gap-2 rounded-xl p-2">
											<input
												type="checkbox"
												className="checkbox checkbox-xs mt-0.5"
												checked={selectedSkillAttachmentPaths.has(path)}
												onChange={event => {
													const checked = event.target.checked;
													setSelectedSkillAttachmentPaths(previous => {
														const next = new Set(previous);
														if (checked) {
															next.add(path);
														} else {
															next.delete(path);
														}
														return next;
													});
												}}
											/>
											<span className="min-w-0 font-mono text-xs break-all">{path}</span>
										</label>
									))}
								</div>
							) : (
								<div className="text-warning mt-2 text-xs">
									{resourceCount} resource{resourceCount === 1 ? ' is' : 's are'} indexed, but no local filesystem paths
									were exposed. You can still insert the rendered template without attachments.
								</div>
							)}

							{item.skillDefinition.resources?.moreLocations ? (
								<div className="text-base-content/60 mt-2 text-xs">
									Additional indexed resources were omitted by the backend and are not selected automatically.
								</div>
							) : null}
						</div>
					) : null}

					<ModalActions className="-mx-6 mt-6 -mb-6">
						<button
							type="button"
							className="btn bg-base-300 rounded-xl"
							onClick={() => {
								requestClose();
							}}
							disabled={submitting}
						>
							Cancel
						</button>
						<button
							type={insertedWithAttachmentWarning ? 'button' : 'submit'}
							className="btn btn-primary rounded-xl"
							disabled={!canInsert}
							onClick={() => {
								if (insertedWithAttachmentWarning) {
									requestClose();
								}
							}}
						>
							{insertedWithAttachmentWarning
								? 'Close'
								: submitting
									? 'Rendering…'
									: selectedAttachmentPaths.length > 0
										? `Insert with ${selectedAttachmentPaths.length} resource attachment${selectedAttachmentPaths.length === 1 ? '' : 's'}`
										: 'Insert'}
						</button>
					</ModalActions>
				</form>
			</div>
		</div>
	);
}

function SkillTemplateRenderModal(props: SkillTemplateRenderModalProps) {
	const { isOpen, item } = props;

	if (!isOpen || !item || typeof document === 'undefined' || !document.body) {
		return null;
	}

	return (
		<ModalDialog isOpen={isOpen} onClose={props.onClose} blockCancel>
			<SkillTemplateRenderModalContent key={skillTemplateKey(item)} item={item} onInsert={props.onInsert} />
		</ModalDialog>
	);
}

function SkillTemplateBottomBarChipInner({
	store,
	buttonRef,
	shortcut,
	onInsertTemplateText,
	onAttachResourcePaths,
	isInputLocked = false,
}: SkillTemplateBottomBarChipProps) {
	const open = useStoreState(store, 'open');
	const hoverTipContent = (
		<HoverTipContent
			title={shortcut ? `Insert template (${shortcut})` : 'Insert template'}
			description="Templates are user-message skills that render into plain composer text."
			sections={[
				{
					id: 'current-state',
					title: 'Current state',
					items: ['No template is active. Inserted templates do not remain selected after insertion.'],
				},
				{
					id: 'insertion-behavior',
					title: 'When you insert',
					items: [
						'The rendered template text is added directly to the composer.',
						'Templates with arguments or resources open a configuration dialog first.',
						'There is no active-template count or clear action after insertion.',
					],
				},
			]}
		/>
	);
	const { items: templates, loading, error: templateLoadError, refresh } = useUserMessageSkillTemplates(open);
	const [modalItem, setModalItem] = useState<SkillListItem | null>(null);
	const [templateActionError, setTemplateActionError] = useState<string | null>(null);

	useEffect(() => {
		if (!isInputLocked) {
			return;
		}
		store.hide();
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setModalItem(null);
	}, [isInputLocked, store]);

	const handleInsertRendered = useCallback(
		async (args: SkillTemplateInsertArgs) => {
			await onInsertTemplateText(args.text);

			if (args.attachedSkillPaths?.length) {
				try {
					await onAttachResourcePaths?.(args.attachedSkillPaths);
				} catch (error) {
					console.error('Template text inserted but resource attachment failed:', error);
					return {
						attachmentError:
							error instanceof Error && error.message.trim()
								? `Template text was inserted, but resource attachment failed: ${error.message}`
								: 'Template text was inserted, but resource attachment failed. You can attach the files manually.',
					};
				}
			}

			store.hide();
		},
		[onAttachResourcePaths, onInsertTemplateText, store]
	);

	const handlePick = useCallback(
		async (item: SkillListItem) => {
			setTemplateActionError(null);

			if ((item.skillDefinition.arguments?.length ?? 0) > 0 || hasResources(item)) {
				store.hide();
				setModalItem(item);
				return;
			}

			let inserted = false;
			try {
				const rendered = await skillStoreAPI.renderSkill(skillRefFromListItem(item), {});
				if (rendered.insert !== 'user-message') {
					throw new Error(`Expected a user-message template, but renderer returned insert=${rendered.insert}.`);
				}
				await onInsertTemplateText(rendered.text);
				inserted = true;
			} catch (error) {
				console.error('Failed to render skill template:', error);
				setTemplateActionError(error instanceof Error ? error.message : 'Failed to render template.');
			} finally {
				if (inserted) {
					store.hide();
				}
			}
		},
		[onInsertTemplateText, store]
	);

	return (
		<div className="relative shrink-0" data-bottom-bar-skill-templates>
			<HoverTip
				content={hoverTipContent}
				placement="top"
				wrapperElement="div"
				wrapperClassName="inline-flex max-w-full"
				tooltipClassName="max-w-sm"
			>
				<div
					className={`${actionTriggerChipSurfaceClasses} border ${
						open ? 'border-base-300 bg-base-300/60' : 'border-transparent'
					} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						ref={buttonRef}
						store={store}
						disabled={isInputLocked}
						className="btn btn-xs app-text-neutral h-auto min-h-0 flex-1 gap-0 border-none bg-transparent p-0 text-left font-normal shadow-none hover:bg-transparent"
						aria-label={shortcut ? `Insert template (${shortcut})` : 'Insert template'}
					>
						<ActionTriggerChipContent icon={<FiFile size={14} />} label="Templates" open={open} />
					</MenuButton>
				</div>
			</HoverTip>

			<SkillTemplateDropdown
				store={store}
				open={open}
				loading={loading}
				items={templates}
				loadError={templateLoadError}
				actionError={templateActionError}
				onRetry={() => {
					void refresh();
				}}
				onPick={item => {
					void handlePick(item);
				}}
			/>

			<SkillTemplateRenderModal
				isOpen={modalItem !== null}
				item={modalItem}
				onClose={() => {
					setModalItem(null);
				}}
				onInsert={t => {
					return handleInsertRendered(t);
				}}
			/>
		</div>
	);
}

export const SkillTemplateBottomBarChip = memo(SkillTemplateBottomBarChipInner);
