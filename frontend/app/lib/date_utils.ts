const norm = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate());

export function groupByDateBuckets<T>(
	items: T[],
	getDate: (item: T) => Date
): Record<'Today' | 'Yesterday' | 'Last 7 Days' | 'Older', T[]> {
	const today = norm(new Date());
	const yesterday = new Date(today);
	yesterday.setDate(today.getDate() - 1);
	const last7 = new Date(today);
	last7.setDate(today.getDate() - 7);

	const buckets = {
		Today: [] as T[],
		Yesterday: [] as T[],
		'Last 7 Days': [] as T[],
		Older: [] as T[],
	};

	items.forEach(item => {
		const day = norm(getDate(item));
		if (day >= today) {
			buckets.Today.push(item);
		} else if (day >= yesterday) {
			buckets.Yesterday.push(item);
		} else if (day >= last7) {
			buckets['Last 7 Days'].push(item);
		} else {
			buckets.Older.push(item);
		}
	});

	return buckets;
}

export function formatDateAsString(d: Date | string): string {
	return new Date(d).toLocaleDateString('en-US', {
		day: '2-digit',
		month: 'short',
		year: 'numeric',
	});
}

export function parseAnyToTime(val: any): Date | undefined {
	if (val === null) {
		return undefined;
	}

	// If it's a string, try to parse as ISO date
	if (typeof val === 'string') {
		const d = new Date(val);
		if (!Number.isNaN(d.getTime())) {
			return d;
		}
	}

	// If it's a number, treat as Unix timestamp (seconds or ms)
	if (typeof val === 'number') {
		// Heuristic: if it's less than 10^12, treat as seconds, else ms
		if (val < 1e12) {
			return new Date(val * 1000); // seconds to ms
		}
		return new Date(val); // already ms
	}

	return undefined;
}
