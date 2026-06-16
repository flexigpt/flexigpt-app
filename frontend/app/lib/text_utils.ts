export function validateSlug(slug: string): string | undefined {
	const trimmed = slug.trim();
	if (!trimmed) return 'Slug is required.';
	if (trimmed.length > 64) return 'Slug must be at most 64 characters.';
	if (!/^[a-zA-Z][a-zA-Z0-9-]*$/.test(trimmed)) {
		return 'Slug must start with a letter, and contain only letters, numbers, and "-".';
	}
	return undefined;
}

export function validateTags(tags: string): string | undefined {
	const tagArr = tags
		.split(',')
		.map(t => t.trim())
		.filter(Boolean);

	const seen = new Set<string>();
	for (let i = 0; i < tagArr.length; i++) {
		const tag = tagArr[i];
		if (tag.length > 64) {
			return `Tag "${tag}" is too long (max 64 characters).`;
		}
		if (!/^[a-zA-Z_][a-zA-Z0-9_-]*$/.test(tag)) {
			return `Tag "${tag}" is invalid. Tags must start with a letter or underscore, then letters, numbers, "-", or "_".`;
		}
		if (seen.has(tag)) {
			return `Duplicate tag "${tag}".`;
		}
		seen.add(tag);
	}
	return undefined;
}

export function cssEscape(s: string) {
	try {
		return CSS.escape(s);
	} catch {
		return s.replace(/[^a-zA-Z0-9_-]/g, '\\$&');
	}
}
