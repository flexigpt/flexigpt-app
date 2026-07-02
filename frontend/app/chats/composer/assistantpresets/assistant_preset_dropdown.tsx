import { useMemo } from 'react';

import { FiCheck, FiEye, FiRefreshCcw, FiTrash2 } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import {
	actionTriggerChipButtonClasses,
	ActionTriggerChipContent,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';
import { GroupedMenuSection, GroupedMenuSubheading } from '@/components/grouped_menu_sections';
import { searchableMenuEmptyStateClasses, SearchableMenuInput } from '@/components/searchmenu/searchable_menu';
import {
	focusFirstSearchableMenuItem,
	isSearchQueryActive,
	rankSearchableItems,
	useSearchableMenuState,
} from '@/components/searchmenu/searchable_menu_utils';

import type { AssistantPresetOptionItem } from '@/chats/composer/assistantpresets/assistant_preset_runtime';

type AssistantPresetOptionItemWithBundleSortFields = AssistantPresetOptionItem & {
	bundleSlug?: string;
	bundleID?: string;
	bundleId?: string;
	bundle?: {
		slug?: string;
		id?: string;
	};
};

interface AssistantPresetOptionGroup {
	bundleSlug: string;
	bundleDisplayName: string;
	selectableOptions: AssistantPresetOptionItem[];
	unavailableOptions: AssistantPresetOptionItem[];
}

const assistantPresetSlugCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function getAssistantPresetBundleSlug(option: AssistantPresetOptionItem): string {
	const optionWithBundle = option as AssistantPresetOptionItemWithBundleSortFields;

	return (
		optionWithBundle.bundleSlug ||
		optionWithBundle.bundle?.slug ||
		optionWithBundle.bundleID ||
		optionWithBundle.bundleId ||
		optionWithBundle.bundle?.id ||
		option.bundleDisplayName ||
		option.key
	);
}

function compareAssistantPresetOptions(a: AssistantPresetOptionItem, b: AssistantPresetOptionItem): number {
	const bundleSlugCompare = assistantPresetSlugCollator.compare(
		getAssistantPresetBundleSlug(a),
		getAssistantPresetBundleSlug(b)
	);
	if (bundleSlugCompare !== 0) {
		return bundleSlugCompare;
	}

	const presetSlugCompare = assistantPresetSlugCollator.compare(a.preset.slug, b.preset.slug);
	if (presetSlugCompare !== 0) {
		return presetSlugCompare;
	}

	const presetVersionCompare = assistantPresetSlugCollator.compare(a.preset.version, b.preset.version);
	if (presetVersionCompare !== 0) {
		return presetVersionCompare;
	}

	return assistantPresetSlugCollator.compare(a.key, b.key);
}

function groupAssistantPresetOptions(presetOptions: AssistantPresetOptionItem[]): AssistantPresetOptionGroup[] {
	const groupsByBundleSlug = new Map<
		string,
		{ bundleSlug: string; bundleDisplayName: string; options: AssistantPresetOptionItem[] }
	>();

	for (const option of [...presetOptions].toSorted(compareAssistantPresetOptions)) {
		const bundleSlug = getAssistantPresetBundleSlug(option);
		let group = groupsByBundleSlug.get(bundleSlug);

		if (!group) {
			group = {
				bundleSlug,
				bundleDisplayName: option.bundleDisplayName,
				options: [],
			};
			groupsByBundleSlug.set(bundleSlug, group);
		}

		group.options.push(option);
	}

	return [...groupsByBundleSlug.values()].map(group => ({
		bundleSlug: group.bundleSlug,
		bundleDisplayName: group.bundleDisplayName,
		selectableOptions: group.options.filter(option => option.isSelectable),
		unavailableOptions: group.options.filter(option => !option.isSelectable),
	}));
}

interface AssistantPresetDropdownProps {
	presetOptions: AssistantPresetOptionItem[];
	selectedPresetKey: string | null;
	selectedPreset: AssistantPresetOptionItem | null;
	loading: boolean;
	error: string | null;
	actionError: string | null;
	isApplying: boolean;
	basePresetKey: string | null;
	selectedPresetModifiedLabels: string[];
	canResetToBasePreset: boolean;
	onViewPreset: (preset: AssistantPresetOptionItem) => void;
	onReapplySelectedPreset: () => Promise<boolean>;
	onResetToBasePreset: () => Promise<boolean>;
	onSelectPreset: (presetKey: string) => Promise<boolean>;
}

export function AssistantPresetDropdown({
	presetOptions,
	selectedPresetKey,
	selectedPreset,
	loading,
	error,
	actionError,
	isApplying,
	basePresetKey,
	selectedPresetModifiedLabels,
	canResetToBasePreset,
	onViewPreset,
	onReapplySelectedPreset,
	onResetToBasePreset,
	onSelectPreset,
}: AssistantPresetDropdownProps) {
	const menu = useMenuStore({ placement: 'top', focusLoop: true });
	const open = useStoreState(menu, 'open');
	const menuContentElement = useStoreState(menu, 'contentElement');
	const [searchQuery, setSearchQuery] = useSearchableMenuState(open);

	const triggerLabel = selectedPreset ? selectedPreset.displayName : 'Assistant';
	const triggerTitle = selectedPreset
		? `${selectedPreset.displayName} — ${selectedPreset.bundleDisplayName}`
		: 'Apply assistant preset';

	const groupedPresetOptions = useMemo(() => groupAssistantPresetOptions(presetOptions), [presetOptions]);

	const displayedGroupedPresetOptions = useMemo(() => {
		if (!isSearchQueryActive(searchQuery)) {
			return groupedPresetOptions;
		}

		const rankedOptions = rankSearchableItems(presetOptions, {
			query: searchQuery,
			getKey: option => option.key,
			getFields: option => [
				{ value: option.displayName, weight: 6 },
				{ value: option.preset.slug, weight: 5 },
				{ value: option.preset.version, weight: 4 },
				{ value: option.bundleDisplayName, weight: 3 },
				{ value: option.bundleSlug, weight: 3 },
				{ value: option.description, weight: 2 },
				{ value: option.availabilityReason, weight: 1 },
			],
			fallbackCompare: compareAssistantPresetOptions,
		});
		const rankByKey = new Map(rankedOptions.map((option, index) => [option.key, index] as const));
		const rankOptions = (options: AssistantPresetOptionItem[]) =>
			options
				.filter(option => rankByKey.has(option.key))
				.toSorted((a, b) => (rankByKey.get(a.key) ?? 0) - (rankByKey.get(b.key) ?? 0));

		return groupedPresetOptions
			.map(group => ({
				...group,
				selectableOptions: rankOptions(group.selectableOptions),
				unavailableOptions: rankOptions(group.unavailableOptions),
			}))
			.filter(group => group.selectableOptions.length > 0 || group.unavailableOptions.length > 0);
	}, [groupedPresetOptions, presetOptions, searchQuery]);

	const displayedPresetCount = displayedGroupedPresetOptions.reduce(
		(sum, group) => sum + group.selectableOptions.length + group.unavailableOptions.length,
		0
	);
	const firstSelectablePreset =
		displayedGroupedPresetOptions.flatMap(group => group.selectableOptions).find(option => option.isSelectable) ?? null;

	const renderPresetOption = (option: AssistantPresetOptionItem) => {
		const isBasePreset = option.key === basePresetKey;
		const isSelected = option.key === selectedPresetKey;
		const isDisabled = isApplying || !option.isSelectable;
		const modifiedTip =
			selectedPresetModifiedLabels.length > 0
				? `Modified sections: ${selectedPresetModifiedLabels.join(', ')}`
				: 'Preset-managed sections are currently in sync';
		const resetTip =
			selectedPresetModifiedLabels.length > 0
				? `Reset preset-managed sections: ${selectedPresetModifiedLabels.join(', ')}`
				: 'Reapply current assistant preset';
		const clearTip = isBasePreset ? 'Base preset is already active' : 'Switch back to the base assistant preset';

		return (
			<div
				key={option.key}
				className={`border-base-300 flex w-full flex-col rounded-lg border p-2 text-left transition-colors ${
					isSelected ? 'bg-base-200' : 'hover:bg-base-200'
				}`}
			>
				<div className="flex w-full items-start gap-2">
					<MenuItem
						store={menu}
						data-searchable-menu-item="true"
						disabled={isDisabled}
						className={`data-active-item:bg-base-200 flex min-w-0 flex-1 items-start gap-2 rounded-lg p-1 text-left outline-none ${
							isDisabled ? (option.isSelectable ? 'cursor-wait opacity-70' : 'cursor-not-allowed opacity-60') : ''
						}`}
						onClick={() => {
							if (isDisabled) {
								return;
							}
							void (async () => {
								const ok = await onSelectPreset(option.key);
								if (ok) {
									menu.hide();
								}
							})();
						}}
					>
						<div className="pt-0.5">{isSelected ? <FiCheck size={14} /> : <span className="w-3" />}</div>

						<div className="min-w-0 flex-1">
							<div className="truncate text-xs font-medium">{option.displayName}</div>
							<div className="mt-1 flex items-center gap-2 text-[10px] opacity-70">
								<span>
									{option.preset.slug}@{option.preset.version}
								</span>
							</div>
							{option.description ? (
								<div className="mt-1 line-clamp-2 text-xs opacity-75">{option.description}</div>
							) : null}
							{!option.isSelectable ? (
								<div className="text-warning mt-1 text-xs">
									{option.availabilityReason ?? 'This preset is not currently available.'}
								</div>
							) : null}
						</div>
					</MenuItem>

					<div className="flex shrink-0 items-start gap-1">
						<HoverTip
							content={isSelected ? 'View active assistant preset details' : 'View assistant preset details'}
							placement="top"
							wrapperElement="div"
							wrapperClassName="inline-flex"
						>
							<button
								type="button"
								className="btn btn-ghost btn-xs btn-square rounded-lg"
								onClick={() => {
									onViewPreset(option);
								}}
							>
								<FiEye size={14} />
							</button>
						</HoverTip>

						{!option.isSelectable ? <span className="badge badge-warning badge-xs shrink-0">Unavailable</span> : null}
					</div>
				</div>

				{isSelected ? (
					<div className="border-base-300 mt-2 ml-5 flex flex-wrap items-center justify-between gap-1 border-t p-0 pt-1">
						<div className="flex items-center gap-1">
							<HoverTip content={modifiedTip} placement="top" wrapperElement="div" wrapperClassName="inline-flex">
								<span
									className={`badge badge-xs ${selectedPresetModifiedLabels.length > 0 ? 'badge-warning' : 'badge-success'}`}
								>
									{selectedPresetModifiedLabels.length > 0 ? 'Modified' : 'In sync'}
								</span>
							</HoverTip>

							{isBasePreset ? <span className="badge badge-ghost badge-xs">Base</span> : null}
						</div>
						<div className="flex items-center gap-1">
							<HoverTip content={resetTip} placement="top" wrapperElement="div" wrapperClassName="inline-flex">
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									disabled={isApplying}
									onClick={() => {
										void (async () => {
											const ok = await onReapplySelectedPreset();
											if (ok) {
												menu.hide();
											}
										})();
									}}
								>
									<FiRefreshCcw size={14} className="mr-1" />
									{selectedPresetModifiedLabels.length > 0 ? 'Reset' : 'Reapply'}
								</button>
							</HoverTip>

							<HoverTip content={clearTip} placement="top" wrapperElement="div" wrapperClassName="inline-flex">
								<button
									type="button"
									className="btn btn-ghost btn-xs rounded-lg"
									disabled={isApplying || isBasePreset || !canResetToBasePreset}
									onClick={() => {
										void (async () => {
											const ok = await onResetToBasePreset();
											if (ok) {
												menu.hide();
											}
										})();
									}}
								>
									<FiTrash2 size={14} className="mr-1" />
									Clear to base
								</button>
							</HoverTip>
						</div>
					</div>
				) : null}
			</div>
		);
	};

	return (
		<div className="flex w-full justify-center">
			<div className="relative w-full">
				<HoverTip content={triggerTitle} placement="top" wrapperElement="div" wrapperClassName="w-full">
					<MenuButton store={menu} className={`${actionTriggerChipButtonClasses} w-full flex-1 justify-center`}>
						<ActionTriggerChipContent
							label={triggerLabel}
							open={open}
							suffix={selectedPreset ? <FiCheck size={14} className="shrink-0" /> : undefined}
							labelClassName="min-w-0 truncate text-center text-xs font-normal"
							className="w-full justify-center"
						/>
					</MenuButton>
				</HoverTip>

				{open ? (
					<Menu
						store={menu}
						portal
						gutter={8}
						overflowPadding={8}
						autoFocusOnShow={false}
						className={actionTriggerMenuWideClasses}
					>
						<div className="mb-2 px-1 text-xs opacity-70">
							Assistant presets seed starting text, model, instructions, tools, skills, and MCPs.
						</div>

						{error ? (
							<div className="alert alert-error mb-2 rounded-2xl text-xs">
								<span>{error}</span>
							</div>
						) : null}

						{actionError ? (
							<div className="alert alert-error mb-2 rounded-2xl text-xs">
								<span>{actionError}</span>
							</div>
						) : null}

						{presetOptions.length > 0 ? (
							<SearchableMenuInput
								open={open}
								query={searchQuery}
								onQueryChange={setSearchQuery}
								placeholder="Search assistant presets…"
								resultCount={displayedPresetCount}
								totalCount={presetOptions.length}
								disabled={loading}
								onFocusFirstItem={() => {
									focusFirstSearchableMenuItem(menuContentElement);
								}}
								onEnterFirstResult={() => {
									if (!firstSelectablePreset || isApplying) {
										return;
									}
									void (async () => {
										const ok = await onSelectPreset(firstSelectablePreset.key);
										if (ok) {
											menu.hide();
										}
									})();
								}}
								onEscape={() => {
									menu.hide();
								}}
							/>
						) : null}

						{loading ? (
							<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default p-2`}>
								Loading assistant presets…
							</div>
						) : presetOptions.length === 0 ? (
							<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default p-2`}>
								No enabled assistant presets available.
							</div>
						) : displayedPresetCount === 0 ? (
							<div className={searchableMenuEmptyStateClasses}>No assistant presets match your search.</div>
						) : (
							<div className="space-y-2">
								{displayedGroupedPresetOptions.map((group, groupIndex) => (
									<GroupedMenuSection
										key={group.bundleSlug}
										title={group.bundleDisplayName}
										ariaLabel={`${group.bundleDisplayName} assistant presets`}
										separatorBefore={groupIndex > 0}
									>
										{group.selectableOptions.length > 0 && group.unavailableOptions.length > 0 ? (
											<GroupedMenuSubheading>Available</GroupedMenuSubheading>
										) : null}

										{group.selectableOptions.map(renderPresetOption)}

										{group.unavailableOptions.length > 0 ? (
											<GroupedMenuSubheading tone="warning" separated={group.selectableOptions.length > 0}>
												Unavailable
											</GroupedMenuSubheading>
										) : null}

										{group.unavailableOptions.map(renderPresetOption)}
									</GroupedMenuSection>
								))}
							</div>
						)}
					</Menu>
				) : null}
			</div>
		</div>
	);
}
