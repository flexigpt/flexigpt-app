export interface SharedAsyncCatalog<T> {
	load(force?: boolean): Promise<T>;
	invalidate(): void;
}

export function createSharedAsyncCatalog<T>(loader: () => Promise<T>): SharedAsyncCatalog<T> {
	let cached: T | undefined;
	let generation = 0;
	let active:
		| {
				generation: number;
				promise: Promise<T>;
		  }
		| undefined;

	const invalidate = () => {
		generation += 1;
		cached = undefined;
	};

	const load = async (force = false): Promise<T> => {
		if (force) {
			invalidate();
		}

		if (cached !== undefined) {
			return cached;
		}

		if (active) {
			if (active.generation === generation) {
				return active.promise;
			}

			await active.promise.catch(() => undefined);
			return load(false);
		}

		const requestGeneration = generation;
		const promise = loader();
		active = {
			generation: requestGeneration,
			promise,
		};

		try {
			const value = await promise;
			if (generation === requestGeneration) {
				cached = value;
			}
			return value;
		} finally {
			if (active?.promise === promise) {
				active = undefined;
			}
		}
	};

	return {
		load,
		invalidate,
	};
}
