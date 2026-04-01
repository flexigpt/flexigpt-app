import { useCallback, useEffect, useMemo, useRef } from 'react';

import { FiChevronUp, FiX, FiZap } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore } from '@ariakit/react';

import type { SkillListItem, SkillRef } from '@/spec/skill';

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
	setEnabledSkillRefs,
	onEnableAll,
	onDisableAll,
	isInputLocked = false,
}: {
	allSkills: SkillListItem[];
	loading: boolean;
	enabledSkillRefs: SkillRef[];
	setEnabledSkillRefs: React.Dispatch<React.SetStateAction<SkillRef[]>>;
	onEnableAll: () => void;
	onDisableAll: () => void;
	isInputLocked?: boolean;
}) {
	const menu = useMenuStore({ placement: 'top-end', focusLoop: true });
	useEffect(() => {
		if (isInputLocked) menu.hide();
	}, [isInputLocked, menu]);

	const enabledKeySet = useMemo(() => new Set(enabledSkillRefs.map(skillRefKey)), [enabledSkillRefs]);
	const enabledCount = enabledSkillRefs.length;
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

	const containerClassName =
		(isEnabled
			? 'bg-warning/10 text-neutral-custom border-warning/50 hover:bg-warning/15'
			: 'bg-base-200 text-neutral-custom border-0 hover:bg-base-300/80') +
		(isInputLocked ? ' opacity-60' : '') +
		' flex items-center gap-1 rounded-2xl border px-2 py-0';

	const title = useMemo(() => {
		const lines: string[] = [];
		lines.push('Skills');
		lines.push(isEnabled ? `Status: Enabled (${enabledCount})` : 'Status: Disabled');
		if (totalCount > 0) lines.push(`Available: ${totalCount}`);
		return lines.join('\n');
	}, [enabledCount, isEnabled, totalCount]);

	return (
		<div className={containerClassName} title={title} data-bottom-bar-skills>
			<div className="flex items-center gap-2">
				<FiZap size={14} />
				<span className="max-w-24 truncate">Skills</span>
				{loading && totalCount === 0 ? (
					<span className="text-xs opacity-70">Loading…</span>
				) : (
					<span className="text-xs opacity-70">{enabledCount}</span>
				)}
			</div>

			{isEnabled ? (
				<button
					type="button"
					className="btn btn-ghost btn-xs p-0 shadow-none"
					onClick={e => {
						stop(e);
						onDisableAll();
					}}
					title="Remove all selected skills"
					aria-label="Remove all selected skills"
					disabled={isInputLocked}
				>
					<FiX size={12} />
				</button>
			) : null}

			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs p-0 shadow-none"
				aria-label="Choose skills"
				title="Choose skills"
				disabled={isInputLocked}
			>
				<FiChevronUp size={14} />
			</MenuButton>

			<Menu
				store={menu}
				gutter={6}
				className="rounded-box bg-base-100 text-base-content border-base-300 z-50 max-h-72 min-w-96 overflow-y-auto border p-2 shadow-xl"
				autoFocusOnShow
				portal
			>
				<div className="mb-2 flex items-center justify-between gap-2">
					<div className="text-base-content/70 text-xs font-semibold">Skills</div>
					<div className="text-base-content/60 text-xs">
						{enabledCount}/{totalCount} enabled
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
