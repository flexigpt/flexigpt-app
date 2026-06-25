// Deeper (longer) paths first; for equal depth, lexicographic descending.
export function comparePathDeepestFirst(a: ReadonlyArray<number>, b: ReadonlyArray<number>): number {
	if (a.length !== b.length) {
		return b.length - a.length;
	}
	for (let i = 0; i < a.length; i++) {
		const da = a[i] ?? 0;
		const db = b[i] ?? 0;
		if (da !== db) {
			return db - da;
		}
	}
	return 0;
}

export function compareEntryByPathDeepestFirst<T extends [unknown, ReadonlyArray<number>]>(a: T, b: T): number {
	return comparePathDeepestFirst(a[1], b[1]);
}
