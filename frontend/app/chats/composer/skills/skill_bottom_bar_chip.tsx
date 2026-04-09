import { useCallback, useEffect, useMemo, useRef } from 'react';

import { FiZap } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore, useStoreState } from '@ariakit/react';

import type { SkillListItem, SkillRef } from '@/spec/skill';

import { actionTriggerChipButtonClasses, ActionTriggerChipContent } from '@/components/action_trigger_chip';
import { HoverTip } from '@/components/ariakit_hover_tip';

import { dedupeSkillRefs, skillRefFromListItem, skillRefKey } from '@/skills/lib/skill_identity_utils';

type BundleGroup = {
	bundleID: string;
	bundleSlug?: string;
	skills: SkillListItem[];
};

function stop(e: React.SyntheticEvent) {
	e.preventDefault();
	e.stopPropagation();
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
	allSkills,
	loading,
	enabledSkillRefs,
	activeSkillRefs,
	setEnabledSkillRefs,
	onEnableAll,
	onDisableAll,
	isInputLocked = false,
}: {
	allSkills: SkillListItem[];
	loading: boolean;
	enabledSkillRefs: SkillRef[];
	activeSkillRefs: SkillRef[];
	setEnabledSkillRefs: React.Dispatch<React.SetStateAction<SkillRef[]>>;
	onEnableAll: () => void;
	onDisableAll: () => void;
	isInputLocked?: boolean;
}) {
	const menu = useMenuStore({ placement: 'top-end', focusLoop: true });
	const open = useStoreState(menu, 'open');

	useEffect(() => {
		if (isInputLocked) menu.hide();
	}, [isInputLocked, menu]);

	const enabledKeySet = useMemo(() => new Set(enabledSkillRefs.map(skillRefKey)), [enabledSkillRefs]);
	const activeKeySet = useMemo(() => new Set(activeSkillRefs.map(skillRefKey)), [activeSkillRefs]);

	const enabledCount = enabledSkillRefs.length;
	const activeCount = activeSkillRefs.length;
	const totalCount = allSkills.length;
	const isEnabled = enabledCount > 0;

	const groups: BundleGroup[] = useMemo(() => {
		const map = new Map<string, BundleGroup>();
		for (const item of allSkills ?? []) {
			const id = item.bundleID ?? 'unknown-bundle';
			const existing = map.get(id);
			if (existing) {
				existing.skills.push(item);
			} else {
				map.set(id, {
					bundleID: id,
					bundleSlug: item.bundleSlug,
					skills: [item],
				});
			}
		}
		return Array.from(map.values()).sort((a, b) =>
			(a.bundleSlug ?? a.bundleID).localeCompare(b.bundleSlug ?? b.bundleID)
		);
	}, [allSkills]);

	const setSkillEnabled = useCallback(
		(ref: SkillRef, enabled: boolean) => {
			const k = skillRefKey(ref);
			setEnabledSkillRefs(prev => {
				const byKey = new Map<string, SkillRef>();
				for (const r of prev ?? []) byKey.set(skillRefKey(r), r);
				if (enabled) byKey.set(k, ref);
				else byKey.delete(k);
				return Array.from(byKey.values());
			});
		},
		[setEnabledSkillRefs]
	);

	const toggleSkillItem = useCallback(
		(item: SkillListItem) => {
			const ref = skillRefFromListItem(item);
			const k = skillRefKey(ref);
			const next = !enabledKeySet.has(k);
			setSkillEnabled(ref, next);
		},
		[enabledKeySet, setSkillEnabled]
	);

	const setBundleEnabled = useCallback(
		(group: BundleGroup, enabled: boolean) => {
			const refs = group.skills.map(skillRefFromListItem);
			const refKeys = refs.map(skillRefKey);

			setEnabledSkillRefs(prev => {
				const byKey = new Map<string, SkillRef>();
				for (const r of prev ?? []) byKey.set(skillRefKey(r), r);

				if (enabled) {
					for (const r of refs) byKey.set(skillRefKey(r), r);
				} else {
					for (const k of refKeys) byKey.delete(k);
				}

				return dedupeSkillRefs(Array.from(byKey.values()));
			});
		},
		[setEnabledSkillRefs]
	);

	const title = useMemo(() => {
		const lines: string[] = [];
		lines.push('Skills');
		lines.push(isEnabled ? `Status: Enabled (${enabledCount})` : 'Status: Disabled');
		lines.push(`Active: ${activeCount}`);
		if (totalCount > 0) lines.push(`Available: ${totalCount}`);
		if (loading && totalCount === 0) lines.push('Loading available skills…');
		return lines.join('\n');
	}, [activeCount, enabledCount, isEnabled, loading, totalCount]);

	return (
		<div className="relative" data-bottom-bar-skills>
			<HoverTip content={title} placement="top">
				<MenuButton
					store={menu}
					className={`${actionTriggerChipButtonClasses} ${open ? 'bg-base-300/80' : ''} ${isInputLocked ? 'opacity-60' : ''}`}
					aria-label="Choose skills"
					disabled={isInputLocked}
				>
					<ActionTriggerChipContent
						icon={<FiZap size={14} />}
						label="Skills"
						count={
							isEnabled ? <span className="badge badge-success badge-xs bg-success/30">{enabledCount}</span> : undefined
						}
						open={open}
					/>
				</MenuButton>
			</HoverTip>

			<Menu
				store={menu}
				gutter={6}
				className="border-base-300 bg-base-100 text-base-content z-50 max-h-80 min-w-96 overflow-y-auto rounded-xl border p-2 text-xs shadow-lg outline-none"
				autoFocusOnShow
				portal
			>
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
					<div className="text-base-content/60 cursor-default rounded-xl px-2 py-1 text-sm">Loading skills…</div>
				) : totalCount === 0 ? (
					<div className="text-base-content/60 cursor-default rounded-xl px-2 py-1 text-sm">No skills available</div>
				) : (
					<>
						{groups.map(group => {
							const bundleLabel = group.bundleSlug ?? group.bundleID;
							const bundleRefs = group.skills.map(skillRefFromListItem);
							const bundleTotal = bundleRefs.length;
							const bundleEnabled = bundleRefs.filter(r => enabledKeySet.has(skillRefKey(r))).length;

							const bundleChecked = bundleEnabled > 0 && bundleEnabled === bundleTotal;
							const bundleIndeterminate = bundleEnabled > 0 && bundleEnabled < bundleTotal;

							return (
								<div key={group.bundleID} className="mb-2 last:mb-0">
									<MenuItem
										hideOnClick={false}
										className="data-active-item:bg-base-200 flex items-center gap-2 rounded-xl px-2 py-1 outline-none"
										onClick={() => {
											if (isInputLocked) return;
											const next = !(bundleEnabled === bundleTotal);
											setBundleEnabled(group, next);
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
											<div className="truncate text-xs font-semibold">{bundleLabel}</div>
											<div className="text-base-content/60 text-xs">
												{bundleEnabled}/{bundleTotal} enabled
											</div>
										</div>
									</MenuItem>

									<div className="mt-1">
										{group.skills.map(item => {
											const ref = skillRefFromListItem(item);
											const k = skillRefKey(ref);
											const checked = enabledKeySet.has(k);
											const isActive = activeKeySet.has(k);
											const label =
												item.skillDefinition.displayName && item.skillDefinition.displayName.length > 0
													? item.skillDefinition.displayName
													: item.skillSlug;

											return (
												<MenuItem
													key={`${item.bundleID}:${item.skillSlug}`}
													hideOnClick={false}
													className="data-active-item:bg-base-200 flex items-center gap-2 rounded-xl px-2 py-1 pl-6 outline-none"
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
														<div className="truncate text-xs">{label}</div>
														<div className="text-base-content/60 truncate text-xs">
															{item.skillDefinition.type} • {item.skillDefinition.location} •{' '}
															{item.skillDefinition.name}
														</div>
														{isActive ? <div className="badge badge-success badge-xs mt-1">Active</div> : null}
													</div>
												</MenuItem>
											);
										})}
									</div>
								</div>
							);
						})}

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
								}}
								title="Remove all selected skills"
							>
								Remove all
							</button>
						</div>
					</>
				)}
			</Menu>
		</div>
	);
}
