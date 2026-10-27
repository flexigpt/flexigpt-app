import type { RefObject, SubmitEventHandler } from 'react';
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiAlertCircle, FiCheck, FiEye, FiFile, FiFilePlus, FiPaperclip } from 'react-icons/fi';

import type { MenuStore } from '@ariakit/react';
import { MenuButton, MenuItem, useStoreState } from '@ariakit/react';

import type { PathAttachmentsResult } from '@/spec/attachment';
import type { SkillArgument } from '@/spec/skill';
import { ArtifactState } from '@/spec/artifact';
import { SkillInsert } from '@/spec/skill';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { ComposerSkillCatalogItem } from '@/apis/skill_management';
import { skillManagementAPI } from '@/apis/baseapi';

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
import { GroupedMenuSection } from '@/components/grouped_menu_sections';
import { HoverTip } from '@/components/hover_tip';
import { ManagementArtifactSource } from '@/components/managementui/management_artifact_source';
import {
	ManagementModalBody,
	ManagementModalDialog,
	ManagementModalPanel,
} from '@/components/managementui/management_modal_layout';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalHeader } from '@/components/modal/modal_header';
import { searchableMenuEmptyStateClasses, SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

import type { LoadedSkillSource } from '@/skills/skill_artifact_source';
import { loadSkillArtifactSource } from '@/skills/skill_artifact_source';
import { SkillDetailsModal, skillDetailsSubjectFromComposerSkill } from '@/skills/skill_details_modal';
import { getSkillDisplayName, getSkillSecondaryLabel, getSkillSourceLabel } from '@/skills/skill_presentation';
import { selectVerifiedTemplateResourcePaths, uniqueTemplateResourcePaths } from '@/skills/template_resource_selection';

interface SkillTemplateDropdownProps {
	store: MenuStore;
	open: boolean;
	loading: boolean;
	items: ComposerSkillCatalogItem[];
	loadError?: string | null;
	actionError?: string | null;
	onRetry?: () => void;
	onView: (item: ComposerSkillCatalogItem) => void;
	onPick: (item: ComposerSkillCatalogItem) => void;
}

interface SkillTemplateMenuItemProps {
	store: MenuStore;
	item: ComposerSkillCatalogItem;
	onPick: (item: ComposerSkillCatalogItem) => void;
	onView: (item: ComposerSkillCatalogItem) => void;
}

interface SkillTemplateInsertOutcome {
	resourceAttachmentWarnings?: string[];
}

interface SkillTemplateInsertRequest {
	arguments: Record<string, string>;
	resources: {
		allowedPaths: string[];
		selectedPaths: string[];
	};
}

interface SkillTemplateRenderModalProps {
	isOpen: boolean;
	item: ComposerSkillCatalogItem | null;
	onClose: () => void;
	onInsert: (request: SkillTemplateInsertRequest) => Promise<SkillTemplateInsertOutcome>;
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
		request: SkillTemplateInsertRequest
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
	return getSkillDisplayName(item);
}

function skillTemplateKey(item: ComposerSkillCatalogItem): string {
	return `${item.ref.rootID}:${item.ref.artifactID}`;
}

function compareSkillTemplateListItems(left: ComposerSkillCatalogItem, right: ComposerSkillCatalogItem): number {
	const labelCompare = skillTemplateCollator.compare(getSkillTemplateLabel(left), getSkillTemplateLabel(right));
	if (labelCompare !== 0) {
		return labelCompare;
	}

	return skillTemplateCollator.compare(skillTemplateKey(left), skillTemplateKey(right));
}

function templateIsSelectable(item: ComposerSkillCatalogItem): boolean {
	return (
		item.insert === SkillInsert.UserMessage &&
		item.runtimeResolved &&
		item.enabled &&
		item.state === ArtifactState.Available
	);
}

function approvedTemplateResourcePaths(item: ComposerSkillCatalogItem | null | undefined): string[] {
	return uniqueTemplateResourcePaths(item?.resources?.locations);
}

function templateResourceFileName(path: string): string {
	const parts = path.replaceAll('\\', '/').split('/').filter(Boolean);

	return parts.at(-1) || 'Resource file';
}

function templateResourceLabels(paths: readonly string[]): Map<string, string> {
	const occurrences = new Map<string, number>();
	const totals = new Map<string, number>();

	for (const path of paths) {
		const name = templateResourceFileName(path);
		totals.set(name, (totals.get(name) ?? 0) + 1);
	}

	const labels = new Map<string, string>();

	for (const path of paths) {
		const name = templateResourceFileName(path);
		const occurrence = (occurrences.get(name) ?? 0) + 1;
		occurrences.set(name, occurrence);

		labels.set(path, (totals.get(name) ?? 0) > 1 ? `${name} (${occurrence})` : name);
	}

	return labels;
}

function resourceWarningsFromAttachmentResult(result: PathAttachmentsResult): string[] {
	const errorCount = new Set((result.errors ?? []).map(error => error.trim()).filter(Boolean)).size;

	if (errorCount === 0) {
		return [];
	}

	return [`${errorCount} selected template resource${errorCount === 1 ? '' : 's'} could not be attached.`];
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

function templateGroupLabel(item: ComposerSkillCatalogItem): string {
	if (item.builtIn) {
		return 'Built-in templates';
	}

	const source = getSkillSourceLabel(item);

	if (source === 'Managed') {
		return 'Managed templates';
	}

	if (source === 'External') {
		return 'Other templates';
	}

	return source;
}

function SkillTemplateMenuItem({ store, item, onPick, onView }: SkillTemplateMenuItemProps) {
	const unavailable = !templateIsSelectable(item);
	const secondaryName = getSkillSecondaryLabel(item);
	const sourceLabel = getSkillSourceLabel(item);

	return (
		<div className="border-base-300 hover:bg-base-200 flex items-start gap-2 rounded-xl border p-2 transition-colors">
			<MenuItem
				store={store}
				data-searchable-menu-item="true"
				hideOnClick={false}
				disabled={unavailable}
				className={`data-active-item:bg-base-200 flex min-w-0 flex-1 items-start gap-2 rounded-lg p-1 text-left outline-none ${
					unavailable ? 'cursor-not-allowed opacity-60' : ''
				}`}
				title={item.description}
				onClick={() => {
					if (!unavailable) {
						onPick(item);
					}
				}}
			>
				<FiFilePlus size={14} className="text-warning mt-0.5 shrink-0" />

				<div className="min-w-0 flex-1">
					<div className="truncate text-xs font-medium">{getSkillTemplateLabel(item)}</div>
					{secondaryName ? <div className="mt-1 truncate text-[10px] opacity-70">{secondaryName}</div> : null}
					{item.description ? <div className="mt-1 line-clamp-2 text-xs opacity-75">{item.description}</div> : null}
				</div>

				<div className="ml-auto flex shrink-0 flex-wrap justify-end gap-1">
					{item.hasArguments ? <span className="badge badge-warning badge-xs">Args</span> : null}
					{item.hasResources ? <span className="badge badge-info badge-xs">Resources</span> : null}
					<span className="badge badge-ghost badge-xs max-w-24 truncate" title={sourceLabel}>
						{sourceLabel}
					</span>
					{unavailable ? <span className="badge badge-warning badge-xs">Unavailable</span> : null}
				</div>
			</MenuItem>

			<HoverTip content="View Skill details" placement="top" wrapperElement="div" wrapperClassName="inline-flex">
				<button
					type="button"
					className="btn btn-ghost btn-xs btn-square shrink-0 rounded-lg"
					aria-label={`View details for ${getSkillTemplateLabel(item)}`}
					onClick={event => {
						event.preventDefault();
						event.stopPropagation();
						onView(item);
					}}
				>
					<FiEye size={14} />
				</button>
			</HoverTip>
		</div>
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
	onView,
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
				{ value: item.name, weight: 5 },
				{ value: item.description, weight: 4 },
				...item.plugins.flatMap(plugin => [
					{ value: plugin.displayName, weight: 3 },
					{ value: plugin.name, weight: 2 },
				]),
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
		displayedItems.find(item => {
			return templateIsSelectable(item);
		}) ?? null;

	const templateHeaderMeta = isSearchQueryActive(searchQuery)
		? `${displayedItems.length} match${displayedItems.length === 1 ? '' : 'es'}`
		: `${items.length} available`;

	const groupedItems = useMemo(
		() =>
			[...Map.groupBy(displayedItems, templateGroupLabel)].toSorted(([left], [right]) =>
				skillTemplateCollator.compare(left, right)
			),
		[displayedItems]
	);

	return (
		<ActionTriggerMenu store={store} data-menu-kind="templates">
			{open ? (
				<>
					<ActionTriggerMenuHeader>
						<SearchableMenuInput
							open={open}
							query={searchQuery}
							onQueryChange={setSearchQuery}
							placeholder="Search templates…"
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

						<div className={actionTriggerMenuHeaderRowClasses}>
							<div className="min-w-0 flex-1 truncate text-xs font-semibold">Templates</div>
							<span className={actionTriggerMenuMetaClasses}>{templateHeaderMeta}</span>
						</div>

						{loadError ? (
							<div className="alert alert-warning rounded-xl text-xs">
								<FiAlertCircle size={14} className="shrink-0" />
								<div className="min-w-0 flex-1">
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
							<div className="alert alert-error rounded-xl text-xs">
								<FiAlertCircle size={14} className="shrink-0" />
								<span>{actionError}</span>
							</div>
						) : null}
					</ActionTriggerMenuHeader>

					<ActionTriggerMenuBody>
						{loading ? (
							<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
								Loading templates…
							</div>
						) : items.length === 0 ? (
							<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
								No user-message templates are available.
							</div>
						) : displayedItems.length === 0 ? (
							<div className={searchableMenuEmptyStateClasses}>No templates match your search.</div>
						) : (
							<div className="space-y-2">
								{groupedItems.map(([group, templates], index) => (
									<GroupedMenuSection
										key={group}
										title={group}
										ariaLabel={`${group} templates`}
										separatorBefore={index > 0}
									>
										{templates.map(item => (
											<SkillTemplateMenuItem
												key={skillTemplateKey(item)}
												store={store}
												item={item}
												onPick={onPick}
												onView={onView}
											/>
										))}
									</GroupedMenuSection>
								))}
							</div>
						)}
					</ActionTriggerMenuBody>

					<ComposerDropdownSelectedPanel
						label="Template insertion"
						ariaLabel="Template insertion"
						isEmpty
						emptyLabel="Choose a template above to render it into the composer."
						help="Templates render into ordinary draft text. They do not remain selected or join a Skill session after insertion."
					/>
				</>
			) : null}
		</ActionTriggerMenu>
	);
}

function SkillTemplateRenderModalContent({
	item,
	onInsert,
}: Omit<SkillTemplateRenderModalProps, 'isOpen' | 'onClose'>) {
	const [argumentsJSON, setArgumentsJSON] = useState('{}');
	const [verifiedSource, setVerifiedSource] = useState<LoadedSkillSource | null>(null);
	const [argumentValues, setArgumentValues] = useState<Record<string, string>>(() =>
		Object.fromEntries((item?.arguments ?? []).map(argument => [argument.name, argument.default ?? '']))
	);
	const resourceSelectionTouchedRef = useRef(false);
	const declaredArguments: SkillArgument[] | undefined = verifiedSource?.details.document.arguments ?? item?.arguments;
	const sourceHasDeclaredArguments = verifiedSource !== null || item?.arguments !== undefined;
	const templateHasArguments =
		verifiedSource !== null
			? (verifiedSource.details.document.arguments?.length ?? 0) > 0
			: item?.hasArguments === true;
	const approvedResourcePaths = useMemo(
		() =>
			verifiedSource
				? uniqueTemplateResourcePaths(verifiedSource.details.resources.locations)
				: approvedTemplateResourcePaths(item),
		[item, verifiedSource]
	);
	const resourceLabels = useMemo(() => templateResourceLabels(approvedResourcePaths), [approvedResourcePaths]);
	const [selectedResourcePaths, setSelectedResourcePaths] = useState<string[]>(() =>
		approvedTemplateResourcePaths(item)
	);
	const [submitError, setSubmitError] = useState('');
	const [resourceAttachmentWarnings, setResourceAttachmentWarnings] = useState<string[]>([]);
	const [inserted, setInserted] = useState(false);
	const [submitting, setSubmitting] = useState(false);
	const { requestClose, unmountingRef } = useModalDialogController();

	const selectedApprovedResources = selectVerifiedTemplateResourcePaths(approvedResourcePaths, selectedResourcePaths);
	const allApprovedResourcesSelected =
		approvedResourcePaths.length > 0 && selectedApprovedResources.length === approvedResourcePaths.length;
	const hasResourceSelection =
		verifiedSource !== null
			? verifiedSource.details.resources.hasResources || approvedResourcePaths.length > 0
			: Boolean(item?.hasResources) || approvedResourcePaths.length > 0;
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
				const outcome = await onInsert({
					arguments: declaredArguments !== undefined ? argumentValues : parseTemplateArguments(argumentsJSON),
					resources: {
						allowedPaths: approvedResourcePaths,
						selectedPaths: selectedResourcePaths,
					},
				});

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
		<ManagementModalPanel>
			<ModalHeader
				title="Use Template"
				description={`${getSkillTemplateLabel(item)} will be rendered before insertion.`}
				onClose={() => {
					requestClose();
				}}
				closeDisabled={submitting}
			/>

			<form className="flex min-h-0 flex-1 flex-col" onSubmit={handleSubmit}>
				<ManagementModalBody>
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
									{resourceAttachmentWarnings.map(warning => (
										<li key={warning}>{warning}</li>
									))}
								</ul>
							</div>
						</div>
					) : null}

					<ManagementArtifactSource
						label="template SKILL.md source"
						defaultOpen
						load={() => loadSkillArtifactSource(item.ref, item.revision)}
						onLoaded={loaded => {
							setVerifiedSource(loaded);
							setArgumentValues(
								Object.fromEntries(
									(loaded.details.document.arguments ?? []).map(argument => [argument.name, argument.default ?? ''])
								)
							);
							setSelectedResourcePaths([...new Set(loaded.details.resources.locations)]);
						}}
						note="The complete verified template includes frontmatter and unrendered placeholders. Copy and Export use this same source."
					/>

					{declaredArguments !== undefined && declaredArguments.length > 0 ? (
						<div className="space-y-3">
							{declaredArguments.map(argument => (
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
											setArgumentValues(previous => ({
												...previous,
												[argument.name]: value,
											}));
										}}
									/>
									{argument.default !== undefined ? (
										<span className="text-base-content/60 mt-1 block text-xs">
											Default: {argument.default || '(empty value)'}
										</span>
									) : null}
								</label>
							))}
						</div>
					) : templateHasArguments && !sourceHasDeclaredArguments ? (
						<div>
							<p className="text-warning mb-2 text-xs">
								This template accepts arguments, but its argument definitions are unavailable. Known values can still be
								supplied as JSON.
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
								spellCheck={false}
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
										Select resource files to attach alongside the rendered template text.
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
												resourceSelectionTouchedRef.current = true;
												setSelectedResourcePaths(approvedResourcePaths);
											}}
										>
											Select all
										</button>
										<button
											type="button"
											className="btn btn-ghost btn-xs rounded-lg"
											disabled={formLocked || selectedApprovedResources.length === 0}
											onClick={() => {
												resourceSelectionTouchedRef.current = true;
												setSelectedResourcePaths([]);
											}}
										>
											Clear selection
										</button>
									</div>

									<div className="border-base-300 mt-3 max-h-52 space-y-1 overflow-y-auto rounded-xl border p-2">
										{approvedResourcePaths.map(resourcePath => {
											const resourceLabel = resourceLabels.get(resourcePath) ?? 'Resource file';
											const selected = selectedResourcePaths.includes(resourcePath);

											return (
												<label
													key={resourcePath}
													className="hover:bg-base-100 flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1"
												>
													<input
														type="checkbox"
														className="checkbox checkbox-sm"
														checked={selected}
														disabled={formLocked}
														onChange={() => {
															resourceSelectionTouchedRef.current = true;
															setSelectedResourcePaths(previous =>
																previous.includes(resourcePath)
																	? previous.filter(path => path !== resourcePath)
																	: [...previous, resourcePath]
															);
														}}
														aria-label={`Attach template resource ${resourceLabel}`}
													/>
													<span className="min-w-0 flex-1 truncate text-xs">{resourceLabel}</span>
												</label>
											);
										})}
									</div>
								</>
							) : (
								<div className="alert alert-warning mt-3 rounded-xl text-xs">
									No verified resource files are currently available. The template text can still be inserted.
								</div>
							)}

							{(verifiedSource?.details.resources.moreLocations ?? item.resources?.moreLocations) ? (
								<div className="alert alert-info mt-3 rounded-xl text-xs">
									Some additional resource files are available but cannot be attached from this menu.
								</div>
							) : null}
						</section>
					) : null}
				</ManagementModalBody>
				<ModalActions>
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
		</ManagementModalPanel>
	);
}

function SkillTemplateRenderModal(props: SkillTemplateRenderModalProps) {
	const { isOpen, item } = props;

	if (!isOpen || !item || typeof document === 'undefined' || !document.body) {
		return null;
	}

	return (
		<ManagementModalDialog isOpen={isOpen} onClose={props.onClose} blockCancel>
			<SkillTemplateRenderModalContent
				key={`${skillTemplateKey(item)}:${item.revision}`}
				item={item}
				onInsert={props.onInsert}
			/>
		</ManagementModalDialog>
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
	const [templateItem, setTemplateItem] = useState<ComposerSkillCatalogItem | null>(null);
	const [viewedItem, setViewedItem] = useState<ComposerSkillCatalogItem | null>(null);
	const [templateActionError, setTemplateActionError] = useState<string | null>(null);

	const handlePick = useCallback(
		async (item: ComposerSkillCatalogItem) => {
			setTemplateActionError(null);

			if (isInputLocked || !templateIsSelectable(item)) {
				setTemplateActionError(item.runtimeError || 'This template is unavailable or disabled.');
				return;
			}

			store.hide();
			setTemplateItem(item);
		},
		[isInputLocked, store]
	);

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
				onView={item => {
					store.hide();
					setViewedItem(item);
				}}
				onPick={item => {
					void handlePick(item);
				}}
			/>

			<SkillTemplateRenderModal
				isOpen={templateItem !== null && !isInputLocked}
				item={templateItem}
				onClose={() => {
					setTemplateItem(null);
				}}
				onInsert={request => {
					if (!templateItem) {
						return Promise.resolve({});
					}

					return onInsertRendered(templateItem, request);
				}}
			/>

			<SkillDetailsModal
				isOpen={viewedItem !== null}
				skill={viewedItem ? skillDetailsSubjectFromComposerSkill(viewedItem) : null}
				onClose={() => {
					setViewedItem(null);
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
	const hoverTipContent = shortcut ? `Insert template (${shortcut})` : 'Insert a template';
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
			request: SkillTemplateInsertRequest
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
				arguments: request.arguments,
			});

			if (inputLockedRef.current) {
				throw new Error(
					'Template prepared, but the Composer became locked before insertion. Try again when it is available.'
				);
			}

			await onInsertTemplateText(rendered.text);
			onTemplateRendered(item.ref);

			const requestedResourcePaths = selectVerifiedTemplateResourcePaths(
				request.resources.allowedPaths,
				request.resources.selectedPaths
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
				} catch {
					resourceAttachmentWarnings = ['Selected template resources could not be attached.'];
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
		[onAttachApprovedTemplateResourcePaths, onInsertTemplateText, onTemplateRendered, store]
	);

	return (
		<div className="relative shrink-0" data-bottom-bar-skill-templates>
			<HoverTip
				content={hoverTipContent}
				placement="top"
				wrapperElement="div"
				wrapperClassName="inline-flex max-w-full"
			>
				<MenuButton
					ref={buttonRef}
					store={store}
					disabled={isInputLocked}
					className={`${actionTriggerChipButtonClasses} ${
						open ? 'border-base-300 bg-base-300/60' : 'border-transparent'
					} ${isInputLocked ? 'opacity-60' : ''}`}
					aria-label={shortcut ? `Insert template (${shortcut})` : 'Insert template'}
				>
					<ActionTriggerChipContent icon={<FiFile size={14} />} label="Templates" open={open} />
				</MenuButton>
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
