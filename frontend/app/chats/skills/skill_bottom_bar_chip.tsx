import { useCallback, useEffect, useMemo, useRef } from 'react';

import { FiChevronUp, FiX, FiZap } from 'react-icons/fi';

import { Menu, MenuButton, MenuItem, useMenuStore } from '@ariakit/react';

import type { SkillDef, SkillListItem } from '@/spec/skill';

import { dedupeSkillDefs, skillDefFromListItem, skillDefKey } from '@/chats/skills/skill_utils';

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
}: {
	checked: boolean;
	indeterminate: boolean;
	onChange: (next: boolean) => void;
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
			className="checkbox checkbox-xs"
			checked={checked}
			onChange={e => {
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
	enabledSkills,
	setEnabledSkills,
	onEnableAll,
	onDisableAll,
}: {
	allSkills: SkillListItem[];
	loading: boolean;
	enabledSkills: SkillDef[];
	setEnabledSkills: React.Dispatch<React.SetStateAction<SkillDef[]>>;
	onEnableAll: () => void;
	onDisableAll: () => void;
}) {
	const menu = useMenuStore({ placement: 'top-end', focusLoop: true });

	const enabledKeySet = useMemo(() => new Set(enabledSkills.map(skillDefKey)), [enabledSkills]);
	const enabledCount = enabledSkills.length;
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
				map.set(id, { bundleID: id, bundleSlug: item.bundleSlug, skills: [item] });
			}
		}
		return Array.from(map.values()).sort((a, b) =>
			(a.bundleSlug ?? a.bundleID).localeCompare(b.bundleSlug ?? b.bundleID)
		);
	}, [allSkills]);

	const setSkillEnabled = useCallback(
		(def: SkillDef, enabled: boolean) => {
			const k = skillDefKey(def);
			setEnabledSkills(prev => {
				const byKey = new Map<string, SkillDef>();
				for (const d of prev ?? []) byKey.set(skillDefKey(d), d);
				if (enabled) byKey.set(k, def);
				else byKey.delete(k);
				return Array.from(byKey.values());
			});
		},
		[setEnabledSkills]
	);

	const toggleSkillItem = useCallback(
		(item: SkillListItem) => {
			const def = skillDefFromListItem(item);
			const k = skillDefKey(def);
			const next = !enabledKeySet.has(k);
			setSkillEnabled(def, next);
		},
		[enabledKeySet, setSkillEnabled]
	);

	const setBundleEnabled = useCallback(
		(group: BundleGroup, enabled: boolean) => {
			const defs = group.skills.map(skillDefFromListItem);
			const defKeys = defs.map(skillDefKey);

			setEnabledSkills(prev => {
				const byKey = new Map<string, SkillDef>();
				for (const d of prev ?? []) byKey.set(skillDefKey(d), d);

				if (enabled) {
					for (const d of defs) byKey.set(skillDefKey(d), d);
				} else {
					for (const k of defKeys) byKey.delete(k);
				}

				return dedupeSkillDefs(Array.from(byKey.values()));
			});
		},
		[setEnabledSkills]
	);

	const enableAndOpen = useCallback(() => {
		onEnableAll();
		menu.show();
	}, [menu, onEnableAll]);

	const containerClassName =
		(isEnabled
			? 'bg-warning/10 text-neutral-custom border-warning/50 hover:bg-warning/15'
			: 'bg-base-200 text-neutral-custom border-0 hover:bg-base-300/80') +
		' flex items-center gap-1 rounded-2xl border px-2 py-0';

	const title = useMemo(() => {
		const lines: string[] = [];
		lines.push('Skills');
		lines.push(isEnabled ? `Status: Enabled (${enabledCount})` : 'Status: Disabled');
		if (totalCount > 0) lines.push(`Available: ${totalCount}`);
		return lines.join('\n');
	}, [enabledCount, isEnabled, totalCount]);

	if (loading && totalCount === 0) {
		// Still show the chip as "Enable skills" so UX is stable.
	}

	return (
		<div className={containerClassName} title={title} data-bottom-bar-skills>
			{/* When disabled: clicking enables all skills and opens menu */}
			{isEnabled ? (
				<div className="flex items-center gap-2">
					<FiZap size={14} />
					<span className="max-w-24 truncate">Skills</span>
					<span className="text-xs opacity-70">{enabledCount}</span>
				</div>
			) : (
				<button type="button" className="flex items-center gap-2" onClick={enableAndOpen} aria-label="Enable skills">
					<FiZap size={14} />
					<span className="max-w-32 truncate">Enable skills</span>
					{loading ? <span className="text-xs opacity-70">Loading…</span> : null}
				</button>
			)}

			{/* Disable all (only when enabled) */}
			{isEnabled ? (
				<button
					type="button"
					className="btn btn-ghost btn-xs p-0 shadow-none"
					onClick={e => {
						stop(e);
						onDisableAll();
						menu.hide();
					}}
					title="Disable all skills"
					aria-label="Disable all skills"
				>
					<FiX size={12} />
				</button>
			) : null}

			{/* Dropdown / editor */}
			<MenuButton
				store={menu}
				className="btn btn-ghost btn-xs p-0 shadow-none"
				aria-label="Choose skills"
				title="Choose skills"
				onClick={() => {
					// If user opens the menu while disabled, match requested behavior:
					// enable all skills, then open the menu for deselection.
					if (!isEnabled) {
						enableAndOpen();
					}
				}}
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
					<div className="text-base-content/70 text-[11px] font-semibold">Skills</div>
					<div className="text-base-content/60 text-[11px]">
						{enabledCount}/{totalCount} enabled
					</div>
				</div>

				{loading ? (
					<div className="text-base-content/60 cursor-default rounded-xl px-2 py-1 text-sm">Loading skills…</div>
				) : totalCount === 0 ? (
					<div className="text-base-content/60 cursor-default rounded-xl px-2 py-1 text-sm">No skills available</div>
				) : (
					groups.map(group => {
						const bundleLabel = group.bundleSlug ?? group.bundleID;
						const bundleDefs = group.skills.map(skillDefFromListItem);
						const bundleTotal = bundleDefs.length;
						const bundleEnabled = bundleDefs.filter(d => enabledKeySet.has(skillDefKey(d))).length;

						const bundleChecked = bundleEnabled > 0 && bundleEnabled === bundleTotal;
						const bundleIndeterminate = bundleEnabled > 0 && bundleEnabled < bundleTotal;

						return (
							<div key={group.bundleID} className="mb-2 last:mb-0">
								<MenuItem
									hideOnClick={false}
									className="data-active-item:bg-base-200 flex items-center gap-2 rounded-xl px-2 py-1 outline-none"
									onClick={() => {
										// clicking bundle row toggles between "all on" and "all off"
										const next = !(bundleEnabled === bundleTotal);
										setBundleEnabled(group, next);
									}}
								>
									<BundleCheckbox
										checked={bundleChecked}
										indeterminate={bundleIndeterminate}
										onChange={next => {
											setBundleEnabled(group, next);
										}}
									/>
									<div className="min-w-0 flex-1">
										<div className="truncate text-xs font-semibold">{bundleLabel}</div>
										<div className="text-base-content/60 text-[11px]">
											{bundleEnabled}/{bundleTotal} enabled
										</div>
									</div>
								</MenuItem>

								<div className="mt-1">
									{group.skills.map(item => {
										const def = skillDefFromListItem(item);
										const k = skillDefKey(def);
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
													toggleSkillItem(item);
												}}
											>
												<input
													type="checkbox"
													className="checkbox checkbox-xs"
													checked={checked}
													onChange={e => {
														stop(e);
														setSkillEnabled(def, e.currentTarget.checked);
													}}
													onPointerDown={stop}
													onClick={stop}
													aria-label={`Toggle skill ${label}`}
												/>
												<div className="min-w-0 flex-1">
													<div className="truncate text-xs">{label}</div>
													<div className="text-base-content/60 truncate text-[11px]">
														{def.type} • {def.location} • {def.name}
													</div>
												</div>
											</MenuItem>
										);
									})}
								</div>
							</div>
						);
					})
				)}
			</Menu>
		</div>
	);
}
