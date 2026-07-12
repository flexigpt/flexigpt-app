import type { Dispatch, SetStateAction } from 'react';
import { useCallback, useEffect, useRef, useState } from 'react';

type AsyncResourcePhase = 'idle' | 'loading' | 'refreshing' | 'error';

interface AsyncResourceState<T> {
	data: T;
	error: unknown;
	phase: AsyncResourcePhase;
	hasResolved: boolean;
}

interface UseAsyncResourceOptions<T> {
	initialData: T;
	enabled?: boolean;
	clearErrorOnReload?: boolean;
}

export interface UseAsyncResourceResult<T> {
	data: T;
	error: unknown;
	phase: AsyncResourcePhase;
	isLoading: boolean;
	isRefreshing: boolean;
	hasResolved: boolean;
	reload: () => Promise<T | undefined>;
	reloadOrThrow: () => Promise<T>;
	setData: Dispatch<SetStateAction<T>>;
}

function createAbortError(): Error {
	if (typeof DOMException !== 'undefined') {
		return new DOMException('The async resource request was cancelled.', 'AbortError');
	}

	const error = new Error('The async resource request was cancelled.');
	error.name = 'AbortError';
	return error;
}

function isAbortError(error: unknown): boolean {
	return error instanceof Error && error.name === 'AbortError';
}

/**
 * Loads a replaceable asynchronous resource.
 *
 * A new reload aborts the previous request and stale responses are ignored,
 * even when the underlying API does not consume the AbortSignal.
 *
 * The loader should be wrapped in useCallback so that dependency changes
 * intentionally trigger a new load.
 */
export function useAsyncResource<T>(
	loader: (signal: AbortSignal) => Promise<T>,
	{ initialData, enabled = true, clearErrorOnReload = true }: UseAsyncResourceOptions<T>
): UseAsyncResourceResult<T> {
	const [state, setState] = useState<AsyncResourceState<T>>(() => ({
		data: initialData,
		error: undefined,
		phase: enabled ? 'loading' : 'idle',
		hasResolved: false,
	}));

	const mountedRef = useRef(false);
	const requestIDRef = useRef(0);
	const controllerRef = useRef<AbortController | null>(null);

	const reloadOrThrow = useCallback(async (): Promise<T> => {
		const requestID = (requestIDRef.current += 1);

		controllerRef.current?.abort();
		const controller = new AbortController();
		controllerRef.current = controller;

		if (mountedRef.current) {
			setState(previous => ({
				...previous,
				error: clearErrorOnReload ? undefined : previous.error,
				phase: previous.hasResolved ? 'refreshing' : 'loading',
			}));
		}

		try {
			const value = await loader(controller.signal);

			if (controller.signal.aborted || !mountedRef.current || requestIDRef.current !== requestID) {
				throw createAbortError();
			}

			setState({
				data: value,
				error: undefined,
				phase: 'idle',
				hasResolved: true,
			});

			return value;
		} catch (error) {
			const shouldIgnore =
				controller.signal.aborted || !mountedRef.current || requestIDRef.current !== requestID || isAbortError(error);

			if (!shouldIgnore) {
				setState(previous => ({
					...previous,
					error,
					phase: 'error',
				}));
			}

			throw error;
		} finally {
			if (controllerRef.current === controller) {
				controllerRef.current = null;
			}
		}
	}, [clearErrorOnReload, loader]);

	const reload = useCallback(async (): Promise<T | undefined> => {
		try {
			return await reloadOrThrow();
		} catch {
			return undefined;
		}
	}, [reloadOrThrow]);

	const setData = useCallback<Dispatch<SetStateAction<T>>>(value => {
		setState(previous => {
			const nextData = typeof value === 'function' ? (value as (current: T) => T)(previous.data) : value;

			return {
				...previous,
				data: nextData,
				error: undefined,
				hasResolved: true,
				phase: previous.phase === 'error' ? 'idle' : previous.phase,
			};
		});
	}, []);

	useEffect(() => {
		mountedRef.current = true;

		const start = Promise.resolve().then(() => {
			if (!mountedRef.current) {
				return;
			}

			if (enabled) {
				void reload();
				return;
			}

			setState(previous => ({
				...previous,
				phase: 'idle',
			}));
		});

		void start;

		return () => {
			mountedRef.current = false;
			controllerRef.current?.abort();
			controllerRef.current = null;
		};
	}, [enabled, reload]);

	return {
		data: state.data,
		error: state.error,
		phase: state.phase,
		isLoading: state.phase === 'loading',
		isRefreshing: state.phase === 'refreshing',
		hasResolved: state.hasResolved,
		reload,
		reloadOrThrow,
		setData,
	};
}
