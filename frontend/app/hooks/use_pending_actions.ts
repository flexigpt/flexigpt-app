import { useCallback, useEffect, useRef, useState } from 'react';

export interface UsePendingActionsResult {
	pendingKeys: ReadonlySet<string>;
	isPending: (key: string) => boolean;
	runAction: <T>(key: string, action: () => Promise<T>) => Promise<T>;
}

/**
 * Tracks independent asynchronous mutations.
 *
 * The ref is the atomic source used by event handlers. The state Set exists
 * only to update rendering. Duplicate callers share the in-flight promise,
 * so a second click cannot trigger a competing mutation or prematurely
 * advance UI state before React commits the first state update.
 */
export function usePendingActions(): UsePendingActionsResult {
	const [pendingKeys, setPendingKeys] = useState<ReadonlySet<string>>(() => new Set());
	const pendingKeysRef = useRef<Set<string>>(new Set());
	const pendingPromisesRef = useRef<Map<string, Promise<unknown>>>(new Map());
	const mountedRef = useRef(false);

	useEffect(() => {
		mountedRef.current = true;
		const c = pendingKeysRef.current;
		const pendingPromises = pendingPromisesRef.current;
		return () => {
			mountedRef.current = false;
			c.clear();
			pendingPromises.clear();
		};
	}, []);

	const publishPendingKeys = useCallback(() => {
		if (mountedRef.current) {
			setPendingKeys(new Set(pendingKeysRef.current));
		}
	}, []);

	const runAction = useCallback(
		<T>(key: string, action: () => Promise<T>): Promise<T> => {
			const normalizedKey = key.trim();
			if (!normalizedKey) {
				throw new Error('Pending action key must not be empty.');
			}

			const pendingAction = pendingPromisesRef.current.get(normalizedKey);
			if (pendingAction) {
				return pendingAction as Promise<T>;
			}

			pendingKeysRef.current.add(normalizedKey);
			const actionPromise: Promise<T> = Promise.resolve().then(action);
			pendingPromisesRef.current.set(normalizedKey, actionPromise);
			publishPendingKeys();

			void actionPromise
				.finally(() => {
					if (pendingPromisesRef.current.get(normalizedKey) === actionPromise) {
						pendingPromisesRef.current.delete(normalizedKey);
						pendingKeysRef.current.delete(normalizedKey);
						publishPendingKeys();
					}
				})
				.catch(() => undefined);

			return actionPromise;
		},
		[publishPendingKeys]
	);

	const isPending = useCallback((key: string) => pendingKeys.has(key), [pendingKeys]);

	return {
		pendingKeys,
		isPending,
		runAction,
	};
}
