import { useEffect, useState } from 'react';

import type { ModelProviderListItem } from '@/spec/model';
import type { AuthKeyName, AuthKeyType } from '@/spec/setting';
import { AuthKeyTypeProvider } from '@/spec/setting';

import { modelManagementAPI } from '@/apis/baseapi';

let builtInAuthKeys: ReadonlySet<AuthKeyName> | null = null;
let initPromise: Promise<void> | null = null;

export function initBuiltIns(): Promise<void> {
	if (initPromise) {
		return initPromise;
	} // already started

	initPromise = (async () => {
		const items = await modelManagementAPI.listProviders();
		const builtIns = filterBuiltInProviders(items);

		builtInAuthKeys = Object.freeze(new Set(builtIns.map(item => item.name)));
	})();

	return initPromise;
}

function filterBuiltInProviders(items: ModelProviderListItem[]): ModelProviderListItem[] {
	return items.filter(item => item.builtIn);
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
