import type { ReactNode } from 'react';
import { useLayoutEffect, useMemo, useSyncExternalStore } from 'react';

/**
 * @public
 */
export type TitleBarSlots = {
	/**
	 * Extra content on the left, AFTER the fixed app title/version (optional).
	 * Good for page title chips, breadcrumbs, etc.
	 */
	left?: ReactNode;

	/** Center content: search bar, page title, etc. */
	center?: ReactNode;

	/** Right content: page actions, etc. (window buttons are always after this) */
	right?: ReactNode;
};

type StoreState = {
	slots: TitleBarSlots;
	owner: symbol | null;
};

let state: StoreState = { slots: {}, owner: null };
const listeners = new Set<() => void>();

function emit() {
	for (const l of listeners) l();
}

function subscribe(listener: () => void) {
	listeners.add(listener);
	return () => listeners.delete(listener);
}

function getSnapshot() {
	return state;
}

export function useTitleBarSlots(): TitleBarSlots {
	return useSyncExternalStore(subscribe, getSnapshot, getSnapshot).slots;
}

function setTitleBarSlots(next: TitleBarSlots, owner: symbol) {
	state = { slots: next, owner };
	emit();
}

function clearTitleBarSlots(owner: symbol) {
	if (state.owner !== owner) return;
	state = { slots: {}, owner: null };
	emit();
}

/**
 * Call from any page to set titlebar content while that page is mounted.
 *
 * Example:
 *   useTitleBarContent({ center: <MySearch /> }, [someDep])
 */
export function useTitleBarContent(slots: TitleBarSlots, deps: unknown[] = []) {
	const owner = useMemo(() => Symbol('titlebar-owner'), []);

	// useLayoutEffect reduces visual flicker when switching routes
	useLayoutEffect(() => {
		setTitleBarSlots(slots, owner);
		return () => {
			clearTitleBarSlots(owner);
		};
	}, [owner, ...deps]);
}
