import type { SetStateAction } from 'react';

export const resolveStateUpdate = <T>(update: SetStateAction<T>, prev: T): T => {
	return typeof update === 'function' ? (update as (prevState: T) => T)(prev) : update;
};
