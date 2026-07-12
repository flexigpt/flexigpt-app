export function createAbortError(message = 'The asynchronous operation was cancelled.'): Error {
	if (typeof DOMException !== 'undefined') {
		return new DOMException(message, 'AbortError');
	}

	const error = new Error(message);
	error.name = 'AbortError';
	return error;
}

export function isAbortError(error: unknown): boolean {
	return (
		typeof error === 'object' &&
		error !== null &&
		'name' in error &&
		(error as { name?: unknown }).name === 'AbortError'
	);
}

export function throwIfAborted(signal?: AbortSignal): void {
	if (signal?.aborted) {
		throw createAbortError();
	}
}

/**
 * Rejects immediately when the supplied signal is aborted, even when the
 * underlying API does not consume AbortSignal itself.
 *
 * The underlying operation is not forcefully terminated, but its eventual
 * result is detached from the caller and cannot update resource state.
 */
export function raceWithAbortSignal<T>(operation: Promise<T>, signal?: AbortSignal): Promise<T> {
	if (!signal) {
		return operation;
	}
	if (signal.aborted) {
		return Promise.reject(createAbortError());
	}

	return new Promise<T>((resolve, reject) => {
		const handleAbort = () => {
			reject(createAbortError());
		};

		signal.addEventListener('abort', handleAbort, { once: true });

		// oxlint-disable-next-line promise/prefer-catch
		operation.then(
			value => {
				signal.removeEventListener('abort', handleAbort);
				resolve(value);
			},
			(error: unknown) => {
				signal.removeEventListener('abort', handleAbort);
				reject(error);
			}
		);
	});
}

export function withTimeout<T>(
	operation: Promise<T>,
	timeoutMS: number,
	timeoutMessage: string,
	signal?: AbortSignal
): Promise<T> {
	if (!Number.isFinite(timeoutMS) || timeoutMS <= 0) {
		return raceWithAbortSignal(operation, signal);
	}
	if (signal?.aborted) {
		return Promise.reject(createAbortError());
	}

	return new Promise<T>((resolve, reject) => {
		let settled = false;

		const cleanup = () => {
			globalThis.clearTimeout(timer);
			signal?.removeEventListener('abort', handleAbort);
		};

		const settle = (callback: () => void) => {
			if (settled) {
				return;
			}
			settled = true;
			cleanup();
			callback();
		};

		const handleAbort = () => {
			settle(() => {
				reject(createAbortError());
			});
		};

		const timer = globalThis.setTimeout(() => {
			settle(() => {
				const error = new Error(timeoutMessage);
				error.name = 'TimeoutError';
				reject(error);
			});
		}, timeoutMS);

		signal?.addEventListener('abort', handleAbort, { once: true });

		// oxlint-disable-next-line promise/prefer-catch
		operation.then(
			value => {
				settle(() => {
					resolve(value);
				});
			},
			(error: unknown) => {
				settle(() => {
					reject(error);
				});
			}
		);
	});
}

/**
 * Maps values while preserving input order and limiting concurrent work.
 */
export async function mapWithConcurrency<T, R>(
	items: readonly T[],
	concurrency: number,
	mapper: (item: T, index: number) => Promise<R>,
	signal?: AbortSignal
): Promise<R[]> {
	if (!Number.isInteger(concurrency) || concurrency < 1) {
		throw new RangeError('Concurrency must be a positive integer.');
	}

	throwIfAborted(signal);

	const results = Array.from<R>({ length: items.length });
	let nextIndex = 0;

	const worker = async () => {
		while (true) {
			throwIfAborted(signal);

			const index = nextIndex;
			nextIndex += 1;

			if (index >= items.length) {
				return;
			}

			results[index] = await mapper(items[index], index);
		}
	};

	const workerCount = Math.min(concurrency, items.length);
	await Promise.all(Array.from({ length: workerCount }, () => worker()));
	return results;
}
