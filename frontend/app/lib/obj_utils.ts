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
