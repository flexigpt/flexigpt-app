import { useCallback, useMemo, useRef, useState } from 'react';

import { FiAlertCircle, FiMonitor, FiMoon, FiSun } from 'react-icons/fi';

import type { AppTheme } from '@/spec/setting';
import { ThemeType } from '@/spec/setting';
import { CustomThemeDark, CustomThemeLight, CustomThemeSystem, DAISYUI_BUILTIN_THEMES } from '@/spec/theme_consts';

import { updateStartupTheme, useStartupTheme } from '@/hooks/use_startup_theme';
import { useTheme } from '@/hooks/use_theme_provider';

import { settingstoreAPI } from '@/apis/baseapi';

import type { DropdownItem } from '@/components/dropdown';
import { Dropdown } from '@/components/dropdown';

const isOtherThemeName = (n: string): boolean => DAISYUI_BUILTIN_THEMES.includes(n);
const toThemeType = (name: string): ThemeType => {
	if (name === CustomThemeLight) {
		return ThemeType.Light;
	}
	if (name === CustomThemeDark) {
		return ThemeType.Dark;
	}
	if (name === 'system') {
		return ThemeType.System;
	}
	return ThemeType.Other;
};

function getInitialOtherName(startupTheme?: AppTheme | null, providerTheme?: string): string {
	if (startupTheme?.type === ThemeType.Other && isOtherThemeName(startupTheme.name)) {
		return startupTheme.name;
	}

	if (providerTheme && toThemeType(providerTheme) === ThemeType.Other && isOtherThemeName(providerTheme)) {
		return providerTheme;
	}

	return DAISYUI_BUILTIN_THEMES[0];
}

interface ThemeSelectorContentProps {
	startupTheme?: AppTheme | null;
	providerTheme: string;
	setTheme: (themeName: string) => void;
}

function ThemeSelectorContent({ startupTheme, providerTheme, setTheme }: ThemeSelectorContentProps) {
	/* derived state */
	const current = useMemo(() => toThemeType(providerTheme), [providerTheme]);

	/* dropdown items never change -> memo once */
	const dropdownItems = useMemo(
		() => Object.fromEntries(DAISYUI_BUILTIN_THEMES.map(t => [t, { isEnabled: true }])) as Record<string, DropdownItem>,
		[]
	);

	const [otherName, setOtherName] = useState<string>(() => getInitialOtherName(startupTheme, providerTheme));
	const [saving, setSaving] = useState(false);
	const [saveError, setSaveError] = useState('');
	const savingRef = useRef(false);

	const selectedOtherName = current === ThemeType.Other && isOtherThemeName(providerTheme) ? providerTheme : otherName;

	/* ————————————————————————————— apply theme —————————————————————————————— */
	const applyTheme = useCallback(
		async (type: ThemeType, name: string) => {
			if (savingRef.current) {
				return;
			}
			if (name === '') {
				console.error('[Theme] empty name recieved');
				return;
			}

			const previousTheme = providerTheme;
			savingRef.current = true;
			setSaving(true);
			setSaveError('');

			/* optimistic update */
			setTheme(name);

			const newTheme: AppTheme = {
				type,
				name,
			};

			try {
				await settingstoreAPI.setAppTheme(newTheme);
				updateStartupTheme(newTheme);
				console.log('[Theme] changed to', newTheme.type, newTheme.name);
			} catch (err) {
				console.error('[Theme] failed to persist, reverting', err);
				setTheme(previousTheme);
				setSaveError(err instanceof Error && err.message.trim() ? err.message : 'Failed to save theme.');
			} finally {
				savingRef.current = false;
				setSaving(false);
			}
		},
		[providerTheme, setTheme]
	);

	/* ————————————————————————————— UI —————————————————————————————— */
	return (
		<div className="flex flex-wrap items-center gap-x-6 gap-y-3" aria-busy={saving}>
			<label className="flex cursor-pointer items-center gap-2">
				<input
					type="radio"
					className="radio radio-accent"
					checked={current === ThemeType.System}
					disabled={saving}
					onChange={() => {
						void applyTheme(ThemeType.System, CustomThemeSystem);
					}}
				/>
				<FiMonitor />
				<span className="text-sm">System</span>
			</label>
			<label className="flex cursor-pointer items-center gap-2">
				<input
					type="radio"
					className="radio radio-accent"
					checked={current === ThemeType.Light}
					disabled={saving}
					onChange={() => {
						void applyTheme(ThemeType.Light, CustomThemeLight);
					}}
				/>
				<FiSun />
				<span className="text-sm">Light</span>
			</label>

			<label className="flex cursor-pointer items-center gap-2">
				<input
					type="radio"
					className="radio radio-accent"
					checked={current === ThemeType.Dark}
					disabled={saving}
					onChange={() => {
						void applyTheme(ThemeType.Dark, CustomThemeDark);
					}}
				/>
				<FiMoon />
				<span className="text-sm">Dark</span>
			</label>

			<label className="flex w-full cursor-pointer items-center gap-2 sm:w-auto">
				<input
					type="radio"
					className="radio radio-accent"
					checked={current === ThemeType.Other}
					disabled={saving}
					onChange={() => {
						void applyTheme(ThemeType.Other, selectedOtherName);
					}}
				/>
				<div className="w-full sm:w-52">
					<Dropdown<string>
						dropdownItems={dropdownItems}
						selectedKey={selectedOtherName}
						onChange={async key => {
							setOtherName(key);
							await applyTheme(ThemeType.Other, key);
						}}
						filterDisabled={false}
						title="Select Theme"
						getDisplayName={k => k[0].toUpperCase() + k.slice(1)}
						disabled={saving}
					/>
				</div>
			</label>

			{saveError ? (
				<div className="text-error flex w-full items-start gap-1 text-xs" role="alert">
					<FiAlertCircle className="mt-0.5 shrink-0" size={12} />
					<span className="wrap-break-word">{saveError}</span>
				</div>
			) : null}
		</div>
	);
}

export function ThemeSelector() {
	const [startupTheme, startupReady] = useStartupTheme();
	const { theme: providerTheme, setTheme } = useTheme();

	if (!startupReady) {
		return <span className="loading loading-dots loading-sm" />;
	}

	return <ThemeSelectorContent startupTheme={startupTheme} providerTheme={providerTheme} setTheme={setTheme} />;
}
