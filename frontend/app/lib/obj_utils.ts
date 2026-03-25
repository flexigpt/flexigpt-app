export function omitManyKeys<T extends object, K extends keyof T>(obj: T, keysToRemove: readonly K[]): Omit<T, K> {
	return Object.fromEntries(Object.entries(obj).filter(([key]) => !keysToRemove.includes(key as K))) as Omit<T, K>;
}

export function stripUndefinedDeep<T>(value: T): T {
	if (Array.isArray(value)) {
		return value.map(item => stripUndefinedDeep(item) as unknown) as T;
	}

	if (value && typeof value === 'object') {
		const out: Record<string, unknown> = {};
		for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
			if (v === undefined) continue;
			out[k] = stripUndefinedDeep(v);
		}
		return out as T;
	}

	return value;
}

export function dedupeStringArray(values: string[]): string[] {
	const seen = new Set<string>();
	const out: string[] = [];

	for (const value of values) {
		if (seen.has(value)) continue;
		seen.add(value);
		out.push(value);
	}

	return out;
}

export function arraysEqual(a: string[] = [], b: string[] = []): boolean {
	if (a.length !== b.length) return false;
	return a.every((value, index) => value === b[index]);
}

export function parsePositiveInteger(value: string): number | undefined {
	const trimmed = value.trim();
	if (!trimmed) return undefined;

	const parsed = Number(trimmed);
	if (!Number.isInteger(parsed) || parsed <= 0) {
		return undefined;
	}

	return parsed;
}

export function parseOptionalNumber(val: string, defaultVal?: number): number | undefined {
	const trimmed = val.trim();
	if (trimmed === '' || !trimmed) {
		if (defaultVal) {
			return defaultVal;
		}
		return undefined;
	}

	const parsed = Number(trimmed);
	if (Number.isNaN(parsed) || !Number.isFinite(parsed)) {
		if (defaultVal) {
			return defaultVal;
		}
		return undefined;
	}

	return parsed;
}
