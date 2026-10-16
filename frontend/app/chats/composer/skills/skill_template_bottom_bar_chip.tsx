import type { RefObject, SubmitEventHandler } from 'react';
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiAlertCircle, FiCheck, FiFile, FiFilePlus, FiPaperclip } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { Menu, MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { PathAttachmentsResult } from '@/spec/attachment';
import { ArtifactState } from '@/spec/artifact';
import { SkillInsert } from '@/spec/skill';

import { getDistinctUserFacingLabel, getPrimaryUserFacingLabel } from '@/lib/user_facing_presentation';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { ComposerSkillCatalogItem } from '@/apis/skill_management';
import { skillManagementAPI } from '@/apis/baseapi';

import {
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { ComposerDropdownSelectedPanel } from '@/components/composer_dropdown_selected_panel';
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

interface SkillTemplateDropdownProps {
	store: MenuStore;
	open: boolean;
	loading: boolean;
	items: ComposerSkillCatalogItem[];
	loadError?: string | null;
	actionError?: string | null;
	onRetry?: () => void;
	onPick: (item: ComposerSkillCatalogItem) => void;
}

interface SkillTemplateInsertOutcome {
	resourceAttachmentWarnings?: string[];
}

interface SkillTemplateRenderModalProps {
	isOpen: boolean;
	item: ComposerSkillCatalogItem | null;
	onClose: () => void;
	onInsert: (args: Record<string, string>, resourcePaths: string[]) => Promise<SkillTemplateInsertOutcome>;
}

interface SkillTemplatePickerProps {
	store: MenuStore;
	open: boolean;
	loading: boolean;
	items: ComposerSkillCatalogItem[];
	loadError?: string | null;
	onRefresh?: () => Promise<void> | void;
	isInputLocked: boolean;
	onInsertRendered: (
		item: ComposerSkillCatalogItem,
		args: Record<string, string>,
		resourcePaths: string[]
	) => Promise<SkillTemplateInsertOutcome>;
}

export interface SkillTemplateBottomBarChipProps {
	store: MenuStore;
	buttonRef: RefObject<HTMLButtonElement | null>;
	shortcut?: string;
	items: ComposerSkillCatalogItem[];
	loading: boolean;
	loadError?: string | null;
	onRefresh?: () => Promise<void> | void;
	onTemplateRendered: (skillRef: ComposerSkillCatalogItem['ref']) => void;
	onInsertTemplateText: (text: string) => Promise<void> | void;
	onAttachApprovedTemplateResourcePaths: (
		paths: string[],
		canApply: () => boolean
	) => Promise<PathAttachmentsResult | undefined>;
	isInputLocked?: boolean;
}

const skillTemplateCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function getSkillTemplateLabel(item: ComposerSkillCatalogItem): string {
	return getPrimaryUserFacingLabel(item.displayName, item.name);
}

function skillTemplateKey(item: ComposerSkillCatalogItem): string {
	return `${item.ref.rootID}:${item.ref.artifactID}`;
}

function compareSkillTemplateListItems(a: ComposerSkillCatalogItem, b: ComposerSkillCatalogItem): number {
	const labelCompare = skillTemplateCollator.compare(getSkillTemplateLabel(a), getSkillTemplateLabel(b));
	if (labelCompare !== 0) {
		return labelCompare;
	}

	return skillTemplateCollator.compare(skillTemplateKey(a), skillTemplateKey(b));
}

function templateIsSelectable(item: ComposerSkillCatalogItem): boolean {
	return item.runtimeResolved && item.enabled && item.state === ArtifactState.Available;
}

function approvedTemplateResourcePaths(item: ComposerSkillCatalogItem | null | undefined): string[] {
	const paths: string[] = [];
	const seen = new Set<string>();

	for (const location of item?.resources?.locations ?? []) {
		// Preserve the exact backend-approved path. The non-blank check only
		// rejects malformed transport values and does not rewrite filenames.
		if (location.trim().length === 0 || seen.has(location)) {
			continue;
		}

		seen.add(location);
		paths.push(location);
	}

	return paths;
}

function selectedApprovedTemplateResourcePaths(
	approvedPaths: readonly string[],
	selectedPaths: readonly string[]
): string[] {
	const selected = new Set(selectedPaths);
	return approvedPaths.filter(path => selected.has(path));
}

function resourceWarningsFromAttachmentResult(result: PathAttachmentsResult): string[] {
	const unique = [...new Set((result.errors ?? []).map(error => error.trim()).filter(Boolean))];

	const maximumVisibleWarnings = 8;
	if (unique.length <= maximumVisibleWarnings) {
		return unique;
	}

	return [
		...unique.slice(0, maximumVisibleWarnings),
		`${unique.length - maximumVisibleWarnings} additional resource attachment errors were omitted.`,
	];
}

function parseTemplateArguments(raw: string): Record<string, string> {
	const trimmed = raw.trim();
	if (!trimmed) {
		return {};
	}

	let value: unknown;
	try {
		value = JSON.parse(trimmed);
	} catch {
		throw new Error('Template arguments must be a JSON object.');
	}

	if (!value || typeof value !== 'object' || Array.isArray(value)) {
		throw new Error('Template arguments must be a JSON object.');
	}

	const result: Record<string, string> = {};
	for (const [key, argument] of Object.entries(value)) {
		if (!key.trim()) {
			throw new Error('Template argument names must not be empty.');
		}
		if (typeof argument === 'string') {
			result[key] = argument;
			continue;
		}
		if (typeof argument === 'number' || typeof argument === 'boolean') {
			result[key] = String(argument);
			continue;
		}
		if (argument === null) {
			result[key] = '';
			continue;
		}
		throw new Error(`Template argument "${key}" must be a string, number, boolean, or null.`);
	}

	return result;
}

function renderSkillTemplateMenuItem(item: ComposerSkillCatalogItem, onPick: (item: ComposerSkillCatalogItem) => void) {
	const disabled = !templateIsSelectable(item);
	const secondaryName = getDistinctUserFacingLabel(getSkillTemplateLabel(item), item.name);

	return (
		<MenuItem
			key={skillTemplateKey(item)}
			data-searchable-menu-item="true"
			hideOnClick={false}
			disabled={disabled}
			onClick={() => {
				if (disabled) {
					return;
				}
				onPick(item);
			}}
			className={`${actionTriggerMenuItemClasses} items-start`}
			title={item.description}
		>
			<FiFilePlus size={14} className="text-warning mt-0.5 shrink-0" />

			<div className="min-w-0 flex-1">
				<div className="truncate text-xs font-medium">{getSkillTemplateLabel(item)}</div>
				{secondaryName ? <div className="mt-1 truncate text-[10px] opacity-70">{secondaryName}</div> : null}
			</div>

			<div className="ml-auto flex shrink-0 items-center gap-1">
				{item.hasArguments ? <span className="badge badge-warning badge-xs">Args</span> : null}
				{item.hasResources ? <span className="badge badge-info badge-xs">Resources</span> : null}
				{item.builtIn ? <span className="badge badge-ghost badge-xs">Built-in</span> : null}
				{!templateIsSelectable(item) ? <span className="badge badge-warning badge-xs">Unavailable</span> : null}
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
				{ value: item.name, weight: 6 },
				{ value: item.description, weight: 3 },
				{ value: item.ref.rootID, weight: 2 },
				{ value: item.ref.artifactID, weight: 2 },
				{ value: item.sourceID, weight: 2 },
				{ value: item.location, weight: 2 },
				...item.plugins.flatMap(plugin => [
					{ value: plugin.name, weight: 3 },
					{ value: plugin.displayName, weight: 3 },
				]),
				...(item.runtimeWarnings ?? []).map(warning => ({ value: warning, weight: 1 })),
				...(item.tags ?? []).map(tag => ({ value: tag, weight: 2 })),
				...(item.arguments ?? []).flatMap(argument => [
					{ value: argument.name, weight: 2 },
					{ value: argument.description, weight: 1 },
					{ value: argument.default, weight: 1 },
				]),
			],
			fallbackCompare: compareSkillTemplateListItems,
		});
	}, [items, searchQuery]);

	const firstTemplate =
		displayedItems.find(t => {
			return templateIsSelectable(t);
		}) ?? null;

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
						Templates are skills with <span className="font-mono">insert: user-message</span>. They render into composer
						text. Selected indexed resources are added as ordinary Composer attachments; the template itself does not
						remain selected after insertion.
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
				<>
					{[
						...Map.groupBy(
							displayedItems,
							item =>
								item.plugins[0]?.displayName ??
								(item.builtIn ? 'Built-in Skills' : item.sourceID || 'Unattached Skills')
						),
					].map(([group, templates]) => (
						<GroupedMenuSection key={group} title={group} ariaLabel={`${group} templates`}>
							{templates.map(item => renderSkillTemplateMenuItem(item, onPick))}
						</GroupedMenuSection>
					))}

					<ComposerDropdownSelectedPanel
						label="Template state"
						emptyLabel="Inserted templates become ordinary draft text. No template remains selected or joins a Skill session."
					/>
				</>
			)}
		</Menu>
	);
}

function SkillTemplateRenderModalContent({
	item,
	onInsert,
}: Omit<SkillTemplateRenderModalProps, 'isOpen' | 'onClose'>) {
	const [argumentsJSON, setArgumentsJSON] = useState('{}');
	const [argumentValues, setArgumentValues] = useState<Record<string, string>>(() =>
		Object.fromEntries((item?.arguments ?? []).map(argument => [argument.name, argument.default ?? '']))
	);
	const approvedResourcePaths = useMemo(() => approvedTemplateResourcePaths(item), [item]);
	const [selectedResourcePaths, setSelectedResourcePaths] = useState<string[]>(() => {
		return approvedResourcePaths;
	});
	const [submitError, setSubmitError] = useState('');
	const [resourceAttachmentWarnings, setResourceAttachmentWarnings] = useState<string[]>([]);
	const [inserted, setInserted] = useState(false);
	const [submitting, setSubmitting] = useState(false);
	const { requestClose, unmountingRef } = useModalDialogController();

	const selectedApprovedResources = selectedApprovedTemplateResourcePaths(approvedResourcePaths, selectedResourcePaths);
	const allApprovedResourcesSelected =
		approvedResourcePaths.length > 0 && selectedApprovedResources.length === approvedResourcePaths.length;
	const hasResourceSelection = Boolean(item?.hasResources) || approvedResourcePaths.length > 0;
	const formLocked = submitting || inserted;

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (!item || formLocked) {
			return;
		}

		setSubmitting(true);
		setSubmitError('');
		setResourceAttachmentWarnings([]);

		void Promise.resolve()
			.then(async () => {
				const outcome = await onInsert(
					item.arguments !== undefined ? argumentValues : parseTemplateArguments(argumentsJSON),
					selectedApprovedResources
				);
				if (unmountingRef.current) {
					return;
				}

				const warnings = outcome?.resourceAttachmentWarnings ?? [];
				if (warnings.length > 0) {
					setResourceAttachmentWarnings(warnings);
					setInserted(true);
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

	return (
		<div className="modal-box bg-base-200 max-h-[80vh] max-w-2xl overflow-hidden rounded-2xl p-0">
			<div className="max-h-[80vh] overflow-y-auto p-6">
				<ModalHeader
					title="Use Template"
					description={`${getSkillTemplateLabel(item)} is rendered by the Session Runtime before insertion.`}
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

					{resourceAttachmentWarnings.length > 0 ? (
						<div className="alert alert-warning rounded-2xl text-sm">
							<FiAlertCircle size={14} />
							<div className="min-w-0">
								<div className="font-semibold">
									Template text was inserted, but some selected resources were not attached.
								</div>
								<ul className="mt-1 list-disc space-y-1 pl-5 text-xs">
									{resourceAttachmentWarnings.map((warning, index) => (
										<li key={`${index}:${warning}`} className="wrap-break-word">
											{warning}
										</li>
									))}
								</ul>
							</div>
						</div>
					) : null}

					{item.hasArguments && item.arguments !== undefined ? (
						<div className="space-y-3">
							{item.arguments.map(argument => (
								<label key={argument.name} className="block">
									<span className="mb-1 block text-sm font-medium">{argument.name}</span>
									{argument.description ? (
										<span className="text-base-content/60 mb-2 block text-xs">{argument.description}</span>
									) : null}
									<input
										className="input w-full rounded-xl"
										value={argumentValues[argument.name] ?? ''}
										placeholder={argument.default ?? ''}
										disabled={formLocked}
										onChange={event => {
											const value = event.currentTarget.value;
											setArgumentValues(previous => ({ ...previous, [argument.name]: value }));
										}}
									/>
									{argument.default !== undefined ? (
										<span className="text-base-content/60 mt-1 block text-xs">
											Default: {argument.default || '(empty string)'}
										</span>
									) : null}
								</label>
							))}
						</div>
					) : item.hasArguments ? (
						<div>
							<p className="text-warning mb-2 text-xs">
								Argument definitions are not exposed for this source by the current management read API. Known argument
								names can still be supplied as JSON.
							</p>
							<label htmlFor="user-message-skill-arguments" className="label py-1">
								<span className="text-sm">Arguments JSON</span>
							</label>
							<textarea
								id="user-message-skill-arguments"
								className="textarea w-full rounded-xl font-mono text-xs"
								rows={8}
								value={argumentsJSON}
								onChange={event => {
									setArgumentsJSON(event.currentTarget.value);
								}}
								disabled={formLocked}
								placeholder={'{\n  "argument": "value"\n}'}
								spellCheck="false"
							/>
						</div>
					) : (
						<div className="text-base-content/70 text-sm">This template has no declared arguments.</div>
					)}
					{hasResourceSelection ? (
						<section className="border-base-300 rounded-xl border p-3" aria-label="Template resource attachments">
							<div className="flex flex-wrap items-start justify-between gap-2">
								<div className="min-w-0">
									<div className="flex items-center gap-2 text-sm font-semibold">
										<FiPaperclip size={14} />
										Template resources
									</div>
									<p className="text-base-content/70 mt-1 text-xs">
										Select verified resource files to attach alongside the rendered template text. Their attachment
										modes can be changed after insertion.
									</p>
								</div>
								{approvedResourcePaths.length > 0 ? (
									<span className="badge badge-ghost badge-sm">
										{selectedApprovedResources.length}/{approvedResourcePaths.length} selected
									</span>
								) : null}
							</div>

							{approvedResourcePaths.length > 0 ? (
								<>
									<div className="mt-3 flex flex-wrap gap-2">
										<button
											type="button"
											className="btn btn-ghost btn-xs rounded-lg"
											disabled={formLocked || allApprovedResourcesSelected}
											onClick={() => {
												setSelectedResourcePaths(approvedResourcePaths);
											}}
										>
											Select all listed
										</button>
										<button
											type="button"
											className="btn btn-ghost btn-xs rounded-lg"
											disabled={formLocked || selectedApprovedResources.length === 0}
											onClick={() => {
												setSelectedResourcePaths([]);
											}}
										>
											Clear selection
										</button>
									</div>

									<div className="border-base-300 mt-3 max-h-52 space-y-1 overflow-y-auto rounded-xl border p-2">
										{approvedResourcePaths.map(resourcePath => {
											const selected = selectedResourcePaths.includes(resourcePath);

											return (
												<label
													key={resourcePath}
													className="hover:bg-base-100 flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1"
													title={resourcePath}
												>
													<input
														type="checkbox"
														className="checkbox checkbox-sm"
														checked={selected}
														disabled={formLocked}
														onChange={() => {
															setSelectedResourcePaths(previous =>
																previous.includes(resourcePath)
																	? previous.filter(path => path !== resourcePath)
																	: [...previous, resourcePath]
															);
														}}
														aria-label={`Attach resource ${resourcePath}`}
													/>
													<span className="min-w-0 flex-1 truncate font-mono text-xs">{resourcePath}</span>
												</label>
											);
										})}
									</div>
								</>
							) : (
								<div className="alert alert-warning mt-3 rounded-xl text-xs">
									The runtime reports resource files for this template, but no verified local paths are available for
									this revision. The template text can still be inserted.
								</div>
							)}

							{item.resources?.moreLocations ? (
								<div className="alert alert-info mt-3 rounded-xl text-xs">
									Only the verified resource paths listed above can be attached here. Additional indexed resource paths
									were omitted from the bounded Skill details response.
								</div>
							) : null}
						</section>
					) : null}

					<ModalActions className="-mx-6 mt-6 -mb-6">
						{inserted ? (
							<button
								type="button"
								className="btn btn-primary rounded-xl"
								onClick={() => {
									requestClose(true);
								}}
							>
								<FiCheck size={14} />
								Done
							</button>
						) : (
							<>
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
								<button type="submit" className="btn btn-primary rounded-xl" disabled={formLocked}>
									{submitting
										? 'Preparing…'
										: selectedApprovedResources.length > 0
											? `Insert + attach (${selectedApprovedResources.length})`
											: 'Insert'}
								</button>
							</>
						)}
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
			<SkillTemplateRenderModalContent
				key={`${skillTemplateKey(item)}:${item.revision}`}
				item={item}
				onInsert={props.onInsert}
			/>
		</ModalDialog>
	);
}

function SkillTemplatePicker({
	store,
	open,
	loading,
	items,
	loadError,
	onRefresh,
	isInputLocked,
	onInsertRendered,
}: SkillTemplatePickerProps) {
	const [modalItem, setModalItem] = useState<ComposerSkillCatalogItem | null>(null);
	const [templateActionError, setTemplateActionError] = useState<string | null>(null);

	const handlePick = useCallback(
		async (item: ComposerSkillCatalogItem) => {
			setTemplateActionError(null);

			if (isInputLocked || !templateIsSelectable(item)) {
				setTemplateActionError(item.runtimeError || 'This template is unavailable or disabled.');
				return;
			}

			if (item.hasArguments || item.hasResources || approvedTemplateResourcePaths(item).length > 0) {
				store.hide();
				setModalItem(item);
				return;
			}

			try {
				await onInsertRendered(item, {}, []);
			} catch (error) {
				setTemplateActionError(error instanceof Error ? error.message : 'Template rendering was rejected.');
			}
		},
		[isInputLocked, onInsertRendered, store]
	);

	const modalIsOpen = modalItem !== null && !isInputLocked;

	return (
		<>
			<SkillTemplateDropdown
				store={store}
				open={open}
				loading={loading}
				items={items}
				loadError={loadError}
				actionError={templateActionError}
				onRetry={
					onRefresh
						? () => {
								void onRefresh();
							}
						: undefined
				}
				onPick={item => {
					void handlePick(item);
				}}
			/>

			<SkillTemplateRenderModal
				isOpen={modalIsOpen}
				item={modalItem}
				onClose={() => {
					setModalItem(null);
				}}
				onInsert={(args, resourcePaths) => {
					if (modalItem) {
						return onInsertRendered(modalItem, args, resourcePaths);
					}
					return Promise.resolve({});
				}}
			/>
		</>
	);
}

function SkillTemplateBottomBarChipInner({
	store,
	buttonRef,
	shortcut,
	items,
	loading,
	loadError,
	onRefresh,
	onTemplateRendered,
	onInsertTemplateText,
	onAttachApprovedTemplateResourcePaths,
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
						'Templates with arguments or indexed resources open a configuration dialog first. Selected verified resource files are added as ordinary Composer attachments.',
						'There is no active-template count or clear action after insertion.',
					],
				},
			]}
		/>
	);
	const templates = useMemo(
		() => items.filter(item => item.insert === SkillInsert.UserMessage).toSorted(compareSkillTemplateListItems),
		[items]
	);
	const templateCatalogRefreshAttemptedRef = useRef(false);
	const inputLockedRef = useRef(isInputLocked);

	useEffect(() => {
		const refresh = onRefresh;
		if (!open || loading || templateCatalogRefreshAttemptedRef.current || !refresh) {
			return;
		}

		templateCatalogRefreshAttemptedRef.current = true;
		void (async () => {
			try {
				await refresh();
			} catch (error) {
				console.error('Failed to refresh Composer template catalog:', error);
			}
		})();
	}, [loading, onRefresh, open]);

	useEffect(() => {
		inputLockedRef.current = isInputLocked;

		if (isInputLocked) {
			store.hide();
		}
	}, [isInputLocked, store]);

	const handleInsertRendered = useCallback(
		async (
			item: ComposerSkillCatalogItem,
			args: Record<string, string>,
			selectedResourcePaths: string[]
		): Promise<SkillTemplateInsertOutcome> => {
			if (inputLockedRef.current) {
				throw new Error('Composer input is locked.');
			}

			const rendered = await skillManagementAPI.prepareUserMessageSkillTemplate({
				skill: {
					rootID: item.ref.rootID,
					artifactID: item.ref.artifactID,
					expectedRevision: item.revision,
					...(item.definitionDigest ? { expectedDigest: item.definitionDigest } : {}),
				},
				arguments: args,
			});

			if (inputLockedRef.current) {
				throw new Error(
					'Template prepared, but the Composer became locked before insertion. Try again when it is available.'
				);
			}
			await onInsertTemplateText(rendered.text);
			onTemplateRendered(item.ref);

			const requestedResourcePaths = selectedApprovedTemplateResourcePaths(
				approvedTemplateResourcePaths(item),
				selectedResourcePaths
			);
			let resourceAttachmentWarnings: string[] | undefined;

			if (requestedResourcePaths.length > 0) {
				try {
					const attachmentResult = await onAttachApprovedTemplateResourcePaths(
						requestedResourcePaths,
						() => !inputLockedRef.current
					);
					if (!attachmentResult) {
						resourceAttachmentWarnings = ['Selected template resources could not be prepared for attachment.'];
					} else {
						const warnings = resourceWarningsFromAttachmentResult(attachmentResult);
						if (warnings.length > 0) {
							resourceAttachmentWarnings = warnings;
						}
					}
				} catch (error) {
					resourceAttachmentWarnings = [
						error instanceof Error ? error.message : 'Selected template resources could not be attached.',
					];
				}
			}

			if (inputLockedRef.current) {
				resourceAttachmentWarnings = [
					...(resourceAttachmentWarnings ?? []),
					'Template text was inserted, but the Composer became locked before all selected resources could be attached.',
				];
			}

			store.hide();

			return { resourceAttachmentWarnings };
		},
		[onAttachApprovedTemplateResourcePaths, onInsertTemplateText, store, onTemplateRendered]
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

			<SkillTemplatePicker
				key={isInputLocked ? 'input-locked-picker' : 'input-unlocked-picker'}
				store={store}
				open={open}
				loading={loading}
				items={templates}
				loadError={loadError}
				onRefresh={onRefresh}
				isInputLocked={isInputLocked}
				onInsertRendered={handleInsertRendered}
			/>
		</div>
	);
}

export const SkillTemplateBottomBarChip = memo(SkillTemplateBottomBarChipInner);
