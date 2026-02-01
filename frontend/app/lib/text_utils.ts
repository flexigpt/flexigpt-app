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

/**
 * @public
 */
export function getBlockQuotedLines(content: string): string {
	// Split the content into lines.
	const lines = content.split('\n');
	// Prepend each line with "> ".
	for (let i = 0; i < lines.length; i++) {
		lines[i] = '> ' + lines[i];
	}
	// Join the lines back together as blockquote.
	return lines.join('\n');
}

export enum CustomMDLanguage {
	ThinkingSummary = 'thinkingsummary',
	Thinking = 'thinking',
}

export function stripCustomMDFences(markdown: string): string {
	// Remove all ~~~thinking blocks
	let r = markdown.replace(/(^|\n)~~~thinking\s*[\s\S]*?\n~~~\s*/g, '$1');
	r = r.replace(/(^|\n)~~~thinkingsummary\s*[\s\S]*?\n~~~\s*/g, '$1');
	return r;
}

// keep letters, digits, space and hyphen; trim & limit to 64 chars
export const sanitizeConversationTitle = (raw: string): string =>
	raw
		.replace(/[^a-zA-Z0-9 -]/g, '') //  ← note the blank space and the hyphen at the end
		.trim()
		.slice(0, 64);

export function expandTabsToSpaces(line: string, tabSize = 2) {
	let out = '';
	let col = 0;
	for (const ch of line) {
		if (ch === '\t') {
			const n = tabSize - (col % tabSize);
			out += ' '.repeat(n);
			col += n;
		} else {
			out += ch;
			col = ch === '\n' ? 0 : col + 1;
		}
	}
	return out;
}

export function cssEscape(s: string) {
	try {
		return CSS.escape(s);
	} catch {
		return s.replace(/[^a-zA-Z0-9_-]/g, '\\$&');
	}
}

// simple {{var}} replacement using provided values (leave unknown tokens intact)
export function replaceDoubleBraces(text: string, vars: Record<string, unknown>): string {
	if (!text) return '';
	return text.replace(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g, (_m, name: string) => {
		const v = vars[name];
		// eslint-disable-next-line @typescript-eslint/no-base-to-string
		return v !== undefined && v !== null ? String(v) : _m;
	});
}
