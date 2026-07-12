import { useCallback, useEffect, useRef, useState } from 'react';

export interface UsePendingActionsResult {
	pendingKeys: ReadonlySet<string>;
	isPending: (key: string) => boolean;
	runAction: <T>(key: string, action: () => Promise<T>) => Promise<T | undefined>;
}

/**
 * Tracks independent asynchronous mutations.
 *
 * The ref is the atomic source used by event handlers. The state Set exists
 * only to update rendering. This prevents a second click from entering the
 * same operation before React commits the first state update.
 */
export function usePendingActions(): UsePendingActionsResult {
	const [pendingKeys, setPendingKeys] = useState<ReadonlySet<string>>(() => new Set());
	const pendingKeysRef = useRef<Set<string>>(new Set());
	const mountedRef = useRef(false);

	useEffect(() => {
		mountedRef.current = true;
		const c = pendingKeysRef.current;
		return () => {
			mountedRef.current = false;
			c.clear();
		};
	}, []);

	const publishPendingKeys = useCallback(() => {
		if (mountedRef.current) {
			setPendingKeys(new Set(pendingKeysRef.current));
		}
	}, []);

	const runAction = useCallback(
		async <T>(key: string, action: () => Promise<T>): Promise<T | undefined> => {
			const normalizedKey = key.trim();
			if (!normalizedKey) {
				throw new Error('Pending action key must not be empty.');
			}

			if (pendingKeysRef.current.has(normalizedKey)) {
				return undefined;
			}

			pendingKeysRef.current.add(normalizedKey);
			publishPendingKeys();

			try {
				return await action();
			} finally {
				pendingKeysRef.current.delete(normalizedKey);
				publishPendingKeys();
			}
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
