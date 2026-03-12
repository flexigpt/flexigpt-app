import { useSyncExternalStore } from 'react';

import mermaid, { type MermaidConfig } from 'mermaid';

import { ALL_DARK_THEMES } from '@/spec/theme_consts';

import { useTheme } from '@/hooks/use_theme_provider';

const PREFERS_DARK_QUERY = '(prefers-color-scheme: dark)';

function subscribePrefersDark(onStoreChange: () => void): () => void {
	if (typeof window === 'undefined' || !window.matchMedia) {
		return () => {};
	}

	const mediaQuery = window.matchMedia(PREFERS_DARK_QUERY);
	const handler = () => {
		onStoreChange();
	};

	mediaQuery.addEventListener('change', handler);
	return () => {
		mediaQuery.removeEventListener('change', handler);
	};
}

function getPrefersDarkSnapshot(): boolean | undefined {
	if (typeof window === 'undefined' || !window.matchMedia) {
		return undefined;
	}
	return window.matchMedia(PREFERS_DARK_QUERY).matches;
}

function getPrefersDarkServerSnapshot(): boolean | undefined {
	return undefined;
}

function usePrefersDark(): boolean | undefined {
	return useSyncExternalStore(subscribePrefersDark, getPrefersDarkSnapshot, getPrefersDarkServerSnapshot);
}

export function useIsDarkMermaid(): boolean {
	const { theme: providerTheme } = useTheme();
	const prefersDark = usePrefersDark();

	/* “system” → fall back to prefers-color-scheme */
	if (providerTheme === 'system') {
		return prefersDark ?? false;
	}

	return ALL_DARK_THEMES.includes(providerTheme as any);
}

type RenderResult = Awaited<ReturnType<typeof mermaid.render>>;

let queue: Promise<unknown> = Promise.resolve();
let lastInitKey: string | null = null;

function makeInitKey(config: MermaidConfig): string {
	// Only include parts that impact SVG output
	const keyObj = {
		theme: config.theme,
		securityLevel: config.securityLevel,
		suppressErrorRendering: config.suppressErrorRendering,
		// themeVariables is common for controlling background etc.
		themeVariables: config.themeVariables ?? null,
	};
	return JSON.stringify(keyObj);
}

export function renderMermaidQueued(id: string, code: string, config: MermaidConfig): Promise<RenderResult> {
	const task = async () => {
		const initKey = makeInitKey(config);
		if (initKey !== lastInitKey) {
			mermaid.initialize(config);
			lastInitKey = initKey;
		}
		return mermaid.render(id, code);
	};

	const result = queue.then(task);
	// Keep queue alive even if one render fails
	queue = result.catch(() => undefined);
	return result;
}
