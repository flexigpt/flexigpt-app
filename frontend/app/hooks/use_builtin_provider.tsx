import { useEffect, useState } from 'react';

import type { ProviderName } from '@/spec/inference';
import type { ProviderPreset } from '@/spec/modelpreset';
import type { AuthKeyName, AuthKeyType } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { getAllProviderPresetsMap } from '@/apis/list_helper';

/* ────────────────────────────────────────────────────────────────────────── */
/*  Internal cache (module scope = one copy per page-load)                   */
/* ────────────────────────────────────────────────────────────────────────── */
let builtInAuthKeys: ReadonlySet<AuthKeyName> | null = null;
let initPromise: Promise<void> | null = null;

export function initBuiltIns(): Promise<void> {
	if (initPromise) {
		return initPromise;
	} // already started

	initPromise = (async () => {
		/* One single call to the remote/helper func */
		const allPresets = await getAllProviderPresetsMap(true);
		const builtIns = filterBuiltInPresets(allPresets);

		/* No need for a second fetch – just derive the key set now. */
		builtInAuthKeys = Object.freeze(new Set(Object.keys(builtIns)));
	})();

	return initPromise;
}

function filterBuiltInPresets(presets: Record<ProviderName, ProviderPreset>): Record<ProviderName, ProviderPreset> {
	return Object.fromEntries(Object.entries(presets).filter(([, p]) => p.isBuiltIn)) as Record<
		ProviderName,
		ProviderPreset
	>;
}

function getBuiltInProviderAuthKeyNamesSync(): ReadonlySet<AuthKeyName> {
	if (!builtInAuthKeys) {
		throw new Error('initBuiltIns() has not finished');
	}
	return builtInAuthKeys;
}

export function isBuiltInProviderAuthKeyName(authKeyType: AuthKeyType, name: AuthKeyName): boolean {
	if (authKeyType !== AuthKeyTypeProvider) {
		return false;
	}
	return getBuiltInProviderAuthKeyNamesSync().has(name);
}

export function useBuiltInsReady(): boolean {
	const [ready, setReady] = useState(false);

	useEffect(() => {
		let cancelled = false;

		// the global promise is created only once
		void initBuiltIns()
			.then(() => {
				if (!cancelled) {
					setReady(true);
				}
			})
			.catch((err: unknown) => {
				if (!cancelled) {
					console.error('Failed to initialize built-ins', err);
				}
			});

		return () => {
			cancelled = true;
		};
	}, []);

	return ready;
}
