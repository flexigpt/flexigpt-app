import { useEffect, useSyncExternalStore } from 'react';

import { type AppTheme, ThemeType } from '@/spec/setting';

import { settingstoreAPI } from '@/apis/baseapi';

const SYSTEM_THEME: AppTheme = { type: ThemeType.System, name: 'system' };

let _startupTheme: Readonly<AppTheme> | null = null;
let _themeInitPromise: Promise<Readonly<AppTheme>> | null = null;
const _listeners = new Set<() => void>();

function emitStartupThemeChange(): void {
	for (const listener of _listeners) {
		listener();
	}
}

function subscribeStartupTheme(listener: () => void): () => void {
	_listeners.add(listener);
	return () => {
		_listeners.delete(listener);
	};
}

function getStartupThemeSnapshot(): Readonly<AppTheme> | null {
	return _startupTheme;
}

// Call during bootstrap - **exactly once** per tab.
export async function initStartupTheme(): Promise<Readonly<AppTheme>> {
	if (_startupTheme) return _startupTheme;
	if (_themeInitPromise) return _themeInitPromise; // already running

	_themeInitPromise = (async () => {
		try {
			const settings = await settingstoreAPI.getSettings();
			_startupTheme = Object.freeze(settings.appTheme);
			console.log('loaded startup theme from backend: ', _startupTheme.name);
		} catch (err) {
			console.error('Failed to fetch theme, falling back to system.', err);
			_startupTheme = Object.freeze(SYSTEM_THEME);
		}

		emitStartupThemeChange();
		return _startupTheme;
	})();

	return _themeInitPromise;
}

// Synchronous access **after** `initStartupTheme()` finished.
export function getStartupThemeSync(): Readonly<AppTheme> {
	if (!_startupTheme) throw new Error('Theme has not been initialised - call initStartupTheme() first.');
	return _startupTheme;
}

// Call after the user changes the theme successfully.
export function updateStartupTheme(theme: AppTheme): void {
	_startupTheme = Object.freeze(theme);
	emitStartupThemeChange();
}

export function useStartupTheme(): [AppTheme | null, boolean] {
	const theme = useSyncExternalStore(subscribeStartupTheme, getStartupThemeSnapshot, getStartupThemeSnapshot);

	useEffect(() => {
		if (!_startupTheme) {
			void initStartupTheme();
		}
	}, []);

	return [theme as AppTheme | null, theme !== null];
}
