import {
	type Dispatch,
	type SetStateAction,
	type SyntheticEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
} from 'react';

import { FiCheck, FiX, FiZap } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, type MenuStore, useMenuStore, useStoreState } from '@ariakit/react';

import type { SkillListItem, SkillRef } from '@/spec/skill';

import {
	ActionTriggerChipContent,
	actionTriggerChipSurfaceClasses,
	actionTriggerMenuItemClasses,
	actionTriggerMenuWideClasses,
} from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';
import { GroupedMenuSection, GroupedMenuSubheading } from '@/components/grouped_menu_sections';

import { dedupeSkillRefs, skillRefFromListItem, skillRefKey } from '@/skills/lib/skill_identity_utils';

type BundleGroup = {
	bundleID: string;
	bundleSlug: string;
	skills: SkillListItem[];
};

const skillDropdownCollator = new Intl.Collator(undefined, {
	numeric: true,
	sensitivity: 'base',
});

function stop(e: SyntheticEvent) {
	e.preventDefault();
	e.stopPropagation();
}

function skillListItemKey(item: SkillListItem): string {
	return skillRefKey(skillRefFromListItem(item));
}

function getSkillDisplayLabel(item: SkillListItem): string {
	return item.skillDefinition.displayName?.trim() || item.skillDefinition.name?.trim() || item.skillSlug;
}

function compareSkillListItems(a: SkillListItem, b: SkillListItem): number {
	const bundleSlugCompare = skillDropdownCollator.compare(a.bundleSlug, b.bundleSlug);
	if (bundleSlugCompare !== 0) return bundleSlugCompare;

	const bundleIDCompare = skillDropdownCollator.compare(a.bundleID, b.bundleID);
	if (bundleIDCompare !== 0) return bundleIDCompare;

	const skillSlugCompare = skillDropdownCollator.compare(a.skillSlug, b.skillSlug);
	if (skillSlugCompare !== 0) return skillSlugCompare;

	const skillNameCompare = skillDropdownCollator.compare(a.skillDefinition.name, b.skillDefinition.name);
	if (skillNameCompare !== 0) return skillNameCompare;

	return skillDropdownCollator.compare(skillListItemKey(a), skillListItemKey(b));
}

function compareBundleGroups(a: BundleGroup, b: BundleGroup): number {
	return (
		skillDropdownCollator.compare(a.bundleSlug, b.bundleSlug) || skillDropdownCollator.compare(a.bundleID, b.bundleID)
	);
}

function getBundleKindLabel(group: BundleGroup): string {
	const builtInCount = group.skills.filter(item => item.isBuiltIn).length;
	if (builtInCount === 0) return 'custom';
	if (builtInCount === group.skills.length) return 'built-in';
	return 'mixed';
}

function BundleCheckbox({
	checked,
	indeterminate,
	onChange,
	isInputLocked,
}: {
	checked: boolean;
	indeterminate: boolean;
	onChange: (next: boolean) => void;
	isInputLocked: boolean;
}) {
	const ref = useRef<HTMLInputElement | null>(null);

	useEffect(() => {
		if (!ref.current) return;
		ref.current.indeterminate = indeterminate;
	}, [indeterminate]);

	return (
		<input
			ref={ref}
			type="checkbox"
			className="checkbox checkbox-xs rounded-sm"
			disabled={isInputLocked}
			checked={checked}
			onChange={e => {
				if (isInputLocked) return;
				onChange(e.currentTarget.checked);
			}}
			onPointerDown={stop}
			onClick={stop}
			aria-label="Toggle bundle skills"
		/>
	);
}

export function SkillsBottomBarChip({
	store,
	shortcut,
	allSkills,
	loading,
	enabledSkillRefs,
	activeSkillRefs,
	setEnabledSkillRefs,
	onEnableAll,
	onDisableAll,
	isInputLocked = false,
}: {
	store: MenuStore;
	shortcut: string;
	allSkills: SkillListItem[];
	loading: boolean;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	setEnabledSkillRefs: Dispatch<SetStateAction<SkillRef[]>>;
	onEnableAll: () => void;
	onDisableAll: () => void;
	isInputLocked?: boolean;
}) {
	const internalMenu = useMenuStore({ placement: 'top', focusLoop: true });
	const menu = store ?? internalMenu;
	const open = useStoreState(menu, 'open');

	useEffect(() => {
		if (isInputLocked) menu.hide();
	}, [isInputLocked, menu]);

	const enabledKeySet = useMemo(() => new Set(enabledSkillRefs.map(skillRefKey)), [enabledSkillRefs]);
	const activeKeySet = useMemo(() => new Set(activeSkillRefs.map(skillRefKey)), [activeSkillRefs]);

	const availableSkillKeySet = useMemo(
		() => new Set((allSkills ?? []).map(item => skillRefKey(skillRefFromListItem(item)))),
		[allSkills]
	);

	const enabledCount = useMemo(() => {
		if (loading) return enabledKeySet.size;

		let count = 0;
		for (const key of enabledKeySet) {
			if (availableSkillKeySet.has(key)) count += 1;
		}
		return count;
	}, [availableSkillKeySet, enabledKeySet, loading]);

	const activeCount = useMemo(() => {
		if (loading) return activeKeySet.size;

		let count = 0;
		for (const key of activeKeySet) {
			if (availableSkillKeySet.has(key)) count += 1;
		}
		return count;
	}, [activeKeySet, availableSkillKeySet, loading]);

	const totalCount = allSkills.length;
	const isEnabled = enabledCount > 0;

	const groups: BundleGroup[] = useMemo(() => {
		const map = new Map<string, BundleGroup>();

		for (const item of [...(allSkills ?? [])].sort(compareSkillListItems)) {
			const id = item.bundleID || 'unknown-bundle';
			const slug = item.bundleSlug || id;
			const existing = map.get(id);

			if (existing) {
				existing.skills.push(item);
			} else {
				map.set(id, {
					bundleID: id,
					bundleSlug: slug,
					skills: [item],
				});
			}
		}

		return Array.from(map.values())
			.map(group => ({
				...group,
				skills: [...group.skills].sort(compareSkillListItems),
			}))
			.sort(compareBundleGroups);
	}, [allSkills]);

	const setSkillEnabled = useCallback(
		(ref: SkillRef, enabled: boolean) => {
			const k = skillRefKey(ref);

			setEnabledSkillRefs(prev => {
				const byKey = new Map<string, SkillRef>();

				for (const r of prev ?? []) {
					byKey.set(skillRefKey(r), r);
				}

				if (enabled) {
					byKey.set(k, ref);
				} else {
					byKey.delete(k);
				}

				return Array.from(byKey.values());
			});
		},
		[setEnabledSkillRefs]
	);

	const toggleSkillItem = useCallback(
		(item: SkillListItem) => {
			const ref = skillRefFromListItem(item);
			const k = skillRefKey(ref);
			setSkillEnabled(ref, !enabledKeySet.has(k));
		},
		[enabledKeySet, setSkillEnabled]
	);

	const setBundleEnabled = useCallback(
		(group: BundleGroup, enabled: boolean) => {
			const refs = group.skills.map(skillRefFromListItem);
			const refKeys = refs.map(skillRefKey);

			setEnabledSkillRefs(prev => {
				const byKey = new Map<string, SkillRef>();

				for (const r of prev ?? []) {
					byKey.set(skillRefKey(r), r);
				}

				if (enabled) {
					for (const r of refs) {
						byKey.set(skillRefKey(r), r);
					}
				} else {
					for (const k of refKeys) {
						byKey.delete(k);
					}
				}

				return dedupeSkillRefs(Array.from(byKey.values()));
			});
		},
		[setEnabledSkillRefs]
	);

	const title = useMemo(() => {
		const lines: string[] = [];
		lines.push(shortcut ? `Attach skills (${shortcut})` : 'Attach skills');
		lines.push(isEnabled ? `Status: Enabled (${enabledCount})` : 'Status: Disabled');
		lines.push(`Active now: ${activeCount}`);
		if (totalCount > 0) lines.push(`Available: ${totalCount}`);
		if (loading && totalCount === 0) lines.push('Loading available skills…');
		return lines.join('\n');
	}, [activeCount, enabledCount, isEnabled, loading, shortcut, totalCount]);

	const chipToneClasses =
		enabledCount > 0
			? 'border-secondary/50 bg-secondary/10 hover:bg-secondary/15'
			: open
				? 'border-base-300 bg-base-300/60'
				: 'border-transparent';

	const renderSkillItem = (item: SkillListItem) => {
		const ref = skillRefFromListItem(item);
		const k = skillRefKey(ref);
		const checked = enabledKeySet.has(k);
		const isActive = activeKeySet.has(k);
		const label = getSkillDisplayLabel(item);

		return (
			<MenuItem
				key={k}
				hideOnClick={false}
				className="data-active-item:bg-base-200 flex items-center gap-2 rounded-xl px-2 py-1 pl-6 outline-none"
				title={`${item.bundleSlug}/${item.skillSlug} • ${item.skillDefinition.type} • ${item.skillDefinition.location}`}
				onClick={() => {
					if (isInputLocked) return;
					toggleSkillItem(item);
				}}
			>
				<input
					type="checkbox"
					className="checkbox checkbox-xs rounded-sm"
					checked={checked}
					disabled={isInputLocked}
					onChange={e => {
						stop(e);
						if (isInputLocked) return;
						setSkillEnabled(ref, e.currentTarget.checked);
					}}
					onPointerDown={stop}
					onClick={stop}
					aria-label={`Toggle skill ${label}`}
				/>

				<div className="min-w-0 flex-1">
					<div className="truncate text-xs font-medium">{label}</div>
					<div className="text-base-content/60 truncate text-xs">
						{item.skillDefinition.type} • {item.skillDefinition.location} • {item.skillDefinition.name}
					</div>

					{isActive || !item.skillDefinition.isEnabled ? (
						<div className="mt-1 flex items-center gap-1">
							{isActive ? <span className="badge badge-success badge-xs">Active</span> : null}
							{!item.skillDefinition.isEnabled ? <span className="badge badge-warning badge-xs">Disabled</span> : null}
						</div>
					) : null}
				</div>
			</MenuItem>
		);
	};

	return (
		<div className="relative shrink-0" data-bottom-bar-skills>
			<HoverTip content={title} placement="top" wrapperElement="div" wrapperClassName="inline-flex max-w-full">
				<div
					className={`${actionTriggerChipSurfaceClasses} border ${chipToneClasses} ${isInputLocked ? 'opacity-60' : ''}`}
				>
					<MenuButton
						store={menu}
						className="btn btn-xs app-text-neutral h-auto min-h-0 flex-1 gap-0 border-none bg-transparent p-0 text-left font-normal shadow-none hover:bg-transparent"
						aria-label={shortcut ? `Attach skills (${shortcut})` : 'Attach skills'}
						disabled={isInputLocked}
					>
						<ActionTriggerChipContent
							icon={<FiZap size={14} />}
							label="Skills"
							count={
								isEnabled ? (
									<span className="badge badge-success badge-xs bg-success/30">{enabledCount}</span>
								) : undefined
							}
							suffix={
								activeCount > 0 ? (
									<span className="badge badge-info badge-xs bg-info/30">Active {activeCount}</span>
								) : isEnabled ? (
									<FiCheck size={14} className="shrink-0" />
								) : undefined
							}
							open={open}
						/>
					</MenuButton>

					{enabledCount > 0 ? (
						<button
							type="button"
							className="btn btn-ghost btn-xs app-text-neutral hover:bg-base-300/80 ml-1 h-auto min-h-0 shrink-0 px-1 py-0 shadow-none"
							onClick={event => {
								stop(event);
								onDisableAll();
								menu.hide();
							}}
							aria-label="Clear all skills"
							title="Clear all skills"
							disabled={isInputLocked}
						>
							<FiX size={12} />
						</button>
					) : null}
				</div>
			</HoverTip>

			<Menu store={menu} gutter={8} overflowPadding={8} className={actionTriggerMenuWideClasses} autoFocusOnShow portal>
				<div className="mb-2 flex items-center justify-between gap-2 px-1">
					<div className="text-base-content/70 text-xs font-semibold">Skills</div>
					<div className="text-base-content/60 flex items-center gap-2 text-xs">
						<span>Enabled: {enabledCount}</span>
						<span>•</span>
						<span>Active: {activeCount}</span>
						<span>•</span>
						<span>{totalCount} available</span>
					</div>
				</div>

				{loading ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>Loading skills…</div>
				) : totalCount === 0 ? (
					<div className={`${actionTriggerMenuItemClasses} text-base-content/60 cursor-default`}>
						No skills available
					</div>
				) : (
					<>
						<div className="space-y-2">
							{groups.map((group, groupIndex) => {
								const bundleRefs = group.skills.map(skillRefFromListItem);
								const bundleTotal = bundleRefs.length;
								const bundleEnabled = bundleRefs.filter(r => enabledKeySet.has(skillRefKey(r))).length;

								const bundleChecked = bundleEnabled > 0 && bundleEnabled === bundleTotal;
								const bundleIndeterminate = bundleEnabled > 0 && bundleEnabled < bundleTotal;

								const enabledSkills = group.skills.filter(item => enabledKeySet.has(skillListItemKey(item)));
								const availableSkills = group.skills.filter(item => !enabledKeySet.has(skillListItemKey(item)));
								const showSubheadings = enabledSkills.length > 0 && availableSkills.length > 0;

								return (
									<GroupedMenuSection
										key={group.bundleID}
										title={group.bundleSlug}
										ariaLabel={`${group.bundleSlug} skills`}
										separatorBefore={groupIndex > 0}
										meta={
											<>
												<span className="badge badge-ghost badge-xs">
													{bundleEnabled}/{bundleTotal}
												</span>
												<span className="badge badge-ghost badge-xs">{getBundleKindLabel(group)}</span>
											</>
										}
									>
										<MenuItem
											hideOnClick={false}
											className="data-active-item:bg-base-200 flex items-center gap-2 rounded-xl px-2 py-1 outline-none"
											onClick={() => {
												if (isInputLocked) return;
												setBundleEnabled(group, bundleEnabled !== bundleTotal);
											}}
										>
											<BundleCheckbox
												isInputLocked={isInputLocked}
												checked={bundleChecked}
												indeterminate={bundleIndeterminate}
												onChange={next => {
													setBundleEnabled(group, next);
												}}
											/>
											<div className="min-w-0 flex-1">
												<div className="truncate text-xs font-semibold">All skills in bundle</div>
												<div className="text-base-content/60 text-xs">
													{bundleEnabled}/{bundleTotal} enabled
												</div>
											</div>
										</MenuItem>

										{enabledSkills.length > 0 ? (
											<>
												{showSubheadings ? <GroupedMenuSubheading>Enabled</GroupedMenuSubheading> : null}
												<div className="space-y-1">{enabledSkills.map(renderSkillItem)}</div>
											</>
										) : null}

										{availableSkills.length > 0 ? (
											<>
												{showSubheadings ? (
													<GroupedMenuSubheading separated={enabledSkills.length > 0}>Available</GroupedMenuSubheading>
												) : null}
												<div className="space-y-1">{availableSkills.map(renderSkillItem)}</div>
											</>
										) : null}
									</GroupedMenuSection>
								);
							})}
						</div>

						<div className="border-base-300 mt-2 flex items-center justify-end gap-2 border-t pt-2">
							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={isInputLocked || totalCount === 0 || enabledCount === totalCount}
								onClick={e => {
									stop(e);
									onEnableAll();
								}}
								title="Select all skills"
							>
								Select all
							</button>

							<button
								type="button"
								className="btn btn-ghost btn-xs rounded-lg"
								disabled={isInputLocked || enabledCount === 0}
								onClick={e => {
									stop(e);
									onDisableAll();
									menu.hide();
								}}
								title="Clear all selected skills"
							>
								Clear all
							</button>
						</div>
					</>
				)}
			</Menu>
		</div>
	);
}
