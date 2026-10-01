import { useRef, useState } from 'react';

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

export function ThemeSelector() {
	const [, startupReady] = useStartupTheme();
	const { theme: providerTheme, setTheme } = useTheme();

	const [saving, setSaving] = useState(false);
	const [saveError, setSaveError] = useState('');
	const savingRef = useRef(false);

	const applyTheme = async (name: string) => {
		if (savingRef.current || !name || name === providerTheme) {
			return;
		}

		const previousTheme = providerTheme;
		const next: AppTheme = { type: toThemeType(name), name };

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
	};

	return (
		<div className="space-y-2" aria-busy={!startupReady || saving}>
			<label className="flex items-center justify-between gap-3">
				<span className="text-sm">Theme</span>

				<select
					className="select select-sm w-40"
					value={providerTheme}
					disabled={!startupReady || saving}
					onChange={event => {
						void applyTheme(event.currentTarget.value);
					}}
				>
					<option value={CustomThemeSystem}>System</option>
					<option value={CustomThemeLight}>Light</option>
					<option value={CustomThemeDark}>Dark</option>

					<optgroup label="More themes">
						{DAISYUI_BUILTIN_THEMES.map(name => (
							<option key={name} value={name}>
								{name.charAt(0).toUpperCase() + name.slice(1)}
							</option>
						))}
					</optgroup>
				</select>
			</label>

			{saveError ? (
				<p className="text-error text-xs wrap-break-word" role="alert">
					{saveError}
				</p>
			) : null}
		</div>
	);
}
