import type { RefObject, SubmitEventHandler } from 'react';
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { createPortal } from 'react-dom';

import { FiAlertCircle, FiFilePlus, FiPaperclip, FiX } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { SkillArgument, SkillListItem, SkillRef } from '@/spec/skill';

import { skillStoreAPI } from '@/apis/baseapi';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';
import { GroupedMenuSection } from '@/components/grouped_menu_sections';
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
	attachedResourcePaths?: string[];
	templateRef: SkillRef;
	templateName: string;
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
	onPick: (item: SkillListItem) => void;
}

interface SkillTemplateRenderModalProps {
	isOpen: boolean;
	item: SkillListItem | null;
	onClose: () => void;
	onInsert: (args: SkillTemplateInsertArgs) => Promise<void> | void;
}

interface RenderFormState {
	arguments: Record<string, string>;
	attachResources: boolean;
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
	return Boolean(resources?.hasResources && resources.totalCount > 0);
}

function normalizeRelativeResourceLocation(location: string): string | null {
	const raw = location.trim();
	if (!raw) {
		return null;
	}

	const slash = raw.replaceAll('\\', '/');
	if (slash.startsWith('/') || /^[A-Za-z]:\//.test(slash) || slash.startsWith('//')) {
		return null;
	}

	const parts = slash.split('/').filter(Boolean);
	if (parts.length === 0 || parts.some(part => part === '.' || part === '..')) {
		return null;
	}

	return parts.join('/');
}

function joinSkillResourcePath(skillLocation: string, resourceLocation: string): string | null {
	const base = skillLocation.trim();
	const relative = normalizeRelativeResourceLocation(resourceLocation);
	if (!base || !relative) {
		return null;
	}

	const separator = base.includes('\\') && !base.includes('/') ? '\\' : '/';
	const normalizedRelative = separator === '\\' ? relative.replaceAll('/', '\\') : relative;
	return `${base.replace(/[\\/]+$/, '')}${separator}${normalizedRelative}`;
}

function getSafeResourcePaths(item: SkillListItem | null): string[] {
	if (!item) {
		return [];
	}

	const base = item.skillDefinition.location ?? '';
	const out: string[] = [];
	const seen = new Set<string>();

	for (const resourceLocation of item.skillDefinition.resources?.locations ?? []) {
		const joined = joinSkillResourcePath(base, resourceLocation);
		if (!joined || seen.has(joined)) {
			continue;
		}
		seen.add(joined);
		out.push(joined);
	}

	return out;
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

	const refresh = useCallback(async () => {
		setLoading(true);
		try {
			const next = await collectUserMessageSkillTemplates();
			setItems(next);
		} catch (error) {
			console.error('Failed to load user-message skill templates:', error);
			setItems([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void refresh();
	}, [refresh]);

	useEffect(() => {
		if (!open) {
			return;
		}
		// oxlint-disable-next-line jsreact-hooks/set-state-in-effect
		void refresh();
	}, [open, refresh]);

	return { items, loading, refresh };
}

function renderSkillTemplateMenuItem(item: SkillListItem, onPick: (item: SkillListItem) => void) {
	const args = item.skillDefinition.arguments ?? [];
	const resources = item.skillDefinition.resources;
	const resourceCount = resources?.hasResources ? resources.totalCount : 0;

	return (
		<MenuItem
			key={skillTemplateKey(item)}
			data-searchable-menu-item="true"
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

function SkillTemplateDropdown({ store, open, loading, items, onPick }: SkillTemplateDropdownProps) {
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
					<div className="text-base-content/70 mb-2 px-1 text-xs">
						Templates are skills with <span className="font-mono">insert: user-message</span>. They render into plain
						composer text and are not loaded as active skill-session instructions.
					</div>
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

function SkillTemplateRenderModalContent({ item, onClose, onInsert }: Omit<SkillTemplateRenderModalProps, 'isOpen'>) {
	const args = useMemo(() => getTemplateArguments(item), [item]);
	const safeResourcePaths = useMemo(() => getSafeResourcePaths(item), [item]);
	const [formState, setFormState] = useState<RenderFormState>(() => ({
		arguments: getDefaultArgumentValues(args),
		attachResources: safeResourcePaths.length > 0,
	}));
	const [submitError, setSubmitError] = useState('');
	const [submitting, setSubmitting] = useState(false);
	const dialogRef = useRef<HTMLDialogElement | null>(null);
	const isUnmountingRef = useRef(false);

	useEffect(() => {
		const dialog = dialogRef.current;
		if (!dialog) {
			return;
		}
		if (!dialog.open) {
			try {
				dialog.showModal();
			} catch {
				// Keep rendering safely if showModal fails.
			}
		}
		return () => {
			isUnmountingRef.current = true;
			if (dialog.open) {
				dialog.close();
			}
		};
	}, []);

	const requestClose = useCallback(() => {
		const dialog = dialogRef.current;
		if (dialog?.open) {
			dialog.close();
			return;
		}
		onClose();
	}, [onClose]);

	const handleDialogClose = useCallback(() => {
		if (isUnmountingRef.current) {
			return;
		}
		onClose();
	}, [onClose]);

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (!item || submitting) {
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
				await onInsert({
					text: rendered.text,
					attachedResourcePaths: formState.attachResources ? safeResourcePaths : [],
					templateRef: skillRefFromListItem(item),
					templateName: getSkillTemplateLabel(item),
				});
				requestClose();
			})
			.catch((error: unknown) => {
				setSubmitError(error instanceof Error ? error.message : 'Failed to render template.');
			})
			.finally(() => {
				setSubmitting(false);
			});
	};

	if (!item || typeof document === 'undefined' || !document.body) {
		return null;
	}

	const resourceCount = item.skillDefinition.resources?.totalCount ?? 0;

	return createPortal(
		<dialog
			ref={dialogRef}
			className="modal"
			onClose={handleDialogClose}
			onCancel={event => {
				event.preventDefault();
			}}
		>
			<div className="modal-box bg-base-200 max-h-[80vh] max-w-2xl overflow-hidden rounded-2xl p-0">
				<div className="max-h-[80vh] overflow-y-auto p-6">
					<div className="mb-4 flex items-center justify-between gap-3">
						<div>
							<h3 className="text-lg font-bold">Use Template</h3>
							<p className="text-base-content/70 mt-1 text-xs">
								{getSkillTemplateLabel(item)} renders into plain composer text.
							</p>
						</div>
						<button
							type="button"
							className="btn btn-sm btn-circle bg-base-300"
							onClick={requestClose}
							aria-label="Close"
						>
							<FiX size={12} />
						</button>
					</div>

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
										</div>
									))}
								</div>
							</div>
						) : (
							<div className="text-base-content/70 text-sm">This template has no arguments.</div>
						)}

						{hasResources(item) ? (
							<div className="border-base-content/10 bg-base-100 rounded-2xl border p-3">
								<div className="flex items-start gap-3">
									<input
										type="checkbox"
										className="checkbox checkbox-sm mt-1 rounded-sm"
										checked={formState.attachResources}
										onChange={event => {
											setFormState(prev => ({
												...prev,
												attachResources: event.target.checked,
											}));
										}}
										disabled={safeResourcePaths.length === 0}
									/>
									<div className="min-w-0 flex-1">
										<div className="font-medium">
											Attach {resourceCount} resource{resourceCount === 1 ? '' : 's'} as files
										</div>
										<div className="text-base-content/70 mt-1 text-xs">
											Resources are files under the skill folder. They will be attached to this message only, not loaded
											as active skill-session context.
										</div>
										{safeResourcePaths.length > 0 ? (
											<ul className="mt-2 max-h-32 space-y-1 overflow-auto">
												{safeResourcePaths.map(path => (
													<li key={path} className="bg-base-200 rounded-xl px-2 py-1 font-mono text-xs break-all">
														{path}
													</li>
												))}
											</ul>
										) : (
											<div className="text-warning mt-2 text-xs">
												No safe relative resource paths were available to attach.
											</div>
										)}
									</div>
								</div>
							</div>
						) : null}

						<div className="modal-action">
							<button type="button" className="btn bg-base-300 rounded-xl" onClick={requestClose}>
								Cancel
							</button>
							<button type="submit" className="btn btn-primary rounded-xl" disabled={submitting}>
								{submitting ? 'Rendering…' : 'Insert'}
							</button>
						</div>
					</form>
				</div>
			</div>
		</dialog>,
		document.body
	);
}

function SkillTemplateRenderModal(props: SkillTemplateRenderModalProps) {
	const { isOpen, item } = props;

	if (!isOpen || !item || typeof document === 'undefined' || !document.body) {
		return null;
	}

	return createPortal(
		<SkillTemplateRenderModalContent
			key={skillTemplateKey(item)}
			item={item}
			onClose={props.onClose}
			onInsert={props.onInsert}
		/>,
		document.body
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
	const tooltip = shortcut
		? `Insert template (${shortcut})\nTemplates are user-message skills rendered into plain composer text.`
		: 'Insert template\nTemplates are user-message skills rendered into plain composer text.';
	const { items: templates, loading } = useUserMessageSkillTemplates(open);
	const [modalItem, setModalItem] = useState<SkillListItem | null>(null);

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
			if (args.attachedResourcePaths?.length) {
				await onAttachResourcePaths?.(args.attachedResourcePaths);
			}
			await onInsertTemplateText(args.text);
			store.hide();
		},
		[onAttachResourcePaths, onInsertTemplateText, store]
	);

	const handlePick = useCallback(
		async (item: SkillListItem) => {
			if ((item.skillDefinition.arguments?.length ?? 0) > 0 || hasResources(item)) {
				setModalItem(item);
				return;
			}

			try {
				const rendered = await skillStoreAPI.renderSkill(skillRefFromListItem(item), {});
				if (rendered.insert !== 'user-message') {
					throw new Error(`Expected a user-message template, but renderer returned insert=${rendered.insert}.`);
				}
				await onInsertTemplateText(rendered.text);
			} catch (error) {
				console.error('Failed to render skill template:', error);
			} finally {
				store.hide();
			}
		},
		[onInsertTemplateText, store]
	);

	return (
		<div className="relative shrink-0" data-bottom-bar-skill-templates>
			<HoverTip content={tooltip} placement="top">
				<MenuButton
					ref={buttonRef}
					store={store}
					disabled={isInputLocked}
					className={`${actionTriggerChipButtonClasses} hover:text-base-content ${isInputLocked ? 'opacity-60' : ''}`}
					aria-label={shortcut ? `Insert template (${shortcut})` : 'Insert template'}
				>
					<ActionTriggerChipContent icon={<FiFilePlus size={16} />} label="Templates" open={open} />
				</MenuButton>
			</HoverTip>

			<SkillTemplateDropdown
				store={store}
				open={open}
				loading={loading}
				items={templates}
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
				onInsert={handleInsertRendered}
			/>
		</div>
	);
}

export const SkillTemplateBottomBarChip = memo(SkillTemplateBottomBarChipInner);
