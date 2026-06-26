import type { ReactNode } from 'react';
import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useMemo, useState } from 'react';

const noTransition = () => {
	const style = document.createElement('style');
	style.textContent = '*{transition:none!important;}';
	document.head.append(style);
	void document.body.offsetHeight;
	requestAnimationFrame(() => {
		style.remove();
	});
};

interface ThemeCtx {
	theme: string;
	setTheme: (t: string) => void;
}

const ThemeContext = createContext<ThemeCtx>({
	theme: 'system',
	setTheme: () => {},
});

interface GenericThemeProviderProps {
	children: ReactNode;
	storageKey: string;
	defaultTheme: string;
	lightTheme: string;
	darkTheme: string;
}

export function GenericThemeProvider({
	children,
	storageKey,
	defaultTheme,
	lightTheme,
	darkTheme,
}: GenericThemeProviderProps) {
	const [theme, _setTheme] = useState<string>(() => {
		const saved = localStorage.getItem(storageKey);
		if (saved) {
			return saved;
		} // already persisted – just use it

		localStorage.setItem(storageKey, defaultTheme);
		return defaultTheme; // first run → persist + use
	});

	/* apply theme before paint */
	useLayoutEffect(() => {
		const effective =
			theme === 'system' ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? darkTheme : lightTheme) : theme;

		noTransition();
		document.documentElement.dataset.theme = effective;
	}, [theme, darkTheme, lightTheme]);

	/* follow OS preference while in “system” mode */
	useEffect(() => {
		if (theme !== 'system') {
			return;
		}

		const mql = window.matchMedia('(prefers-color-scheme: dark)');

		const applySystemTheme = () => {
			noTransition();
			const effective = mql.matches ? darkTheme : lightTheme;
			document.documentElement.dataset.theme = effective;
		};

		applySystemTheme(); // make sure it is correct right now
		mql.addEventListener('change', applySystemTheme);

		return () => {
			mql.removeEventListener('change', applySystemTheme);
		};
	}, [theme, darkTheme, lightTheme]);

	/* wrapped setter keeps localStorage in-sync */
	const setTheme = useCallback(
		(t: string) => {
			_setTheme(t);
			localStorage.setItem(storageKey, t);
		},
		[storageKey]
	);

	const value = useMemo(() => ({ theme, setTheme }), [theme, setTheme]);

	return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

// oxlint-disable-next-line react/only-export-components
export const useTheme = (): ThemeCtx => useContext(ThemeContext);
