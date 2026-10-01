import { useCallback, useMemo, useRef, useState } from 'react';
import { FiAlertCircle, FiMonitor, FiMoon, FiSun } from 'react-icons/fi';

import type { AppTheme } from '@/spec/setting';
import {
	CustomThemeDark,
	CustomThemeLight,
	CustomThemeSystem,
	DAISYUI_BUILTIN_THEMES,
	ThemeType,
} from '@/spec/setting';

import { getErrorMessage } from '@/lib/error_utils';

import { updateStartupTheme, useStartupTheme } from '@/hooks/use_startup_theme';
import { useTheme } from '@/hooks/use_theme_provider';

import { settingstoreAPI } from '@/apis/baseapi';

import type { DropdownItem } from '@/components/dropdown';
import { Dropdown } from '@/components/dropdown';

function toThemeType(name: string): ThemeType {
	switch (name) {
		case CustomThemeSystem:
			return ThemeType.System;
		case CustomThemeLight:
			return ThemeType.Light;
		case CustomThemeDark:
			return ThemeType.Dark;
		default:
			return ThemeType.Other;
	}
}

const isOtherThemeName = (name: string): boolean => DAISYUI_BUILTIN_THEMES.includes(name);

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
	const current = useMemo(() => toThemeType(providerTheme), [providerTheme]);

	const dropdownItems = useMemo(
		() =>
			Object.fromEntries(DAISYUI_BUILTIN_THEMES.map(name => [name, { isEnabled: true }])) as Record<
				string,
				DropdownItem
			>,
		[]
	);

	const [otherName, setOtherName] = useState(() => getInitialOtherName(startupTheme, providerTheme));
	const [saving, setSaving] = useState(false);
	const [saveError, setSaveError] = useState('');
	const savingRef = useRef(false);

	const selectedOtherName = current === ThemeType.Other && isOtherThemeName(providerTheme) ? providerTheme : otherName;

	const applyTheme = useCallback(
		async (type: ThemeType, name: string) => {
			if (savingRef.current || !name || name === providerTheme) {
				return;
			}

			const previousTheme = providerTheme;
			const next: AppTheme = { type, name };

			savingRef.current = true;
			setSaving(true);
			setSaveError('');

			try {
				setTheme(name);
				await settingstoreAPI.setAppTheme(next);
				updateStartupTheme(next);
			} catch (error) {
				console.error('Failed to save theme', error);
				setTheme(previousTheme);
				setSaveError(getErrorMessage(error, 'Could not save theme.'));
			} finally {
				savingRef.current = false;
				setSaving(false);
			}
		},
		[providerTheme, setTheme]
	);

	return (
		<section className="min-w-0" aria-busy={saving}>
			<h2 className="text-base-content/60 mb-3 text-xs font-medium">Theme</h2>

			<div className="space-y-3">
				<div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 p-2">
					<label className="flex cursor-pointer items-center gap-1.5 text-sm">
						<input
							type="radio"
							name="app-theme"
							className="radio radio-accent radio-sm"
							checked={current === ThemeType.System}
							disabled={saving}
							onChange={() => {
								void applyTheme(ThemeType.System, CustomThemeSystem);
							}}
						/>
						<FiMonitor className="shrink-0" size={14} aria-hidden="true" />
						<span>System</span>
					</label>

					<label className="flex cursor-pointer items-center gap-1.5 text-sm">
						<input
							type="radio"
							name="app-theme"
							className="radio radio-accent radio-sm"
							checked={current === ThemeType.Light}
							disabled={saving}
							onChange={() => {
								void applyTheme(ThemeType.Light, CustomThemeLight);
							}}
						/>
						<FiSun className="shrink-0" size={14} aria-hidden="true" />
						<span>Light</span>
					</label>

					<label className="flex cursor-pointer items-center gap-1.5 text-sm">
						<input
							type="radio"
							name="app-theme"
							className="radio radio-accent radio-sm"
							checked={current === ThemeType.Dark}
							disabled={saving}
							onChange={() => {
								void applyTheme(ThemeType.Dark, CustomThemeDark);
							}}
						/>
						<FiMoon className="shrink-0" size={14} aria-hidden="true" />
						<span>Dark</span>
					</label>
				</div>

				<div className="flex min-w-0 items-start gap-3">
					<label className="flex shrink-0 cursor-pointer items-center gap-1.5 p-2 text-sm">
						<input
							type="radio"
							name="app-theme"
							className="radio radio-accent radio-sm"
							checked={current === ThemeType.Other}
							disabled={saving}
							onChange={() => {
								void applyTheme(ThemeType.Other, selectedOtherName);
							}}
						/>
						<span>More themes</span>
					</label>

					<div className="min-w-0 flex-1">
						<Dropdown<string>
							dropdownItems={dropdownItems}
							selectedKey={selectedOtherName}
							onChange={name => {
								setOtherName(name);
								void applyTheme(ThemeType.Other, name);
							}}
							filterDisabled={false}
							title="Select theme"
							getDisplayName={name => name.charAt(0).toUpperCase() + name.slice(1)}
							disabled={saving}
							inlineMenu
							maxMenuHeight={180}
						/>
					</div>
				</div>

				{saveError ? (
					<div className="alert alert-error rounded-lg px-3 py-2 text-xs" role="alert">
						<FiAlertCircle className="shrink-0" size={14} aria-hidden="true" />
						<span className="wrap-break-word">{saveError}</span>
					</div>
				) : null}
			</div>
		</section>
	);
}

export function ThemeSelector() {
	const [startupTheme, startupReady] = useStartupTheme();
	const { theme: providerTheme, setTheme } = useTheme();

	if (!startupReady) {
		return (
			<section className="min-w-0" aria-busy="true">
				<h2 className="text-base-content/60 mb-3 text-xs font-medium">Theme</h2>

				<output className="text-base-content/60 flex items-center gap-2 text-xs">
					<span className="loading loading-dots loading-sm" aria-hidden="true" />
					<span>Loading theme...</span>
				</output>
			</section>
		);
	}

	return <ThemeSelectorContent startupTheme={startupTheme} providerTheme={providerTheme} setTheme={setTheme} />;
}
