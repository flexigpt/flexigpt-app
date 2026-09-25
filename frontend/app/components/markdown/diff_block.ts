function looksLikeOpenAIPatch(value: string): boolean {
	return /^\*\*\*\s+Begin\s+Patch\s*$/im.test(value) || /^\*\*\*\s+(?:Update|Add|Delete)\s+File:\s+/im.test(value);
}

function isUnifiedDiffLanguage(language: string): boolean {
	const normalized = language.trim().toLowerCase();
	return normalized === 'diff' || normalized === 'patch' || normalized === 'udiff';
}

export function looksLikeUnifiedDiff(value: string, language = ''): boolean {
	const text = value.replaceAll('\r\n', '\n').replaceAll('\r', '\n');

	if (looksLikeOpenAIPatch(text)) {
		return true;
	}
	if (isUnifiedDiffLanguage(language)) {
		return true;
	}
	if (/^diff --git\s+/m.test(text)) {
		return true;
	}
	if (/^Index:\s+/m.test(text)) {
		return true;
	}
	const hasPlainFileHeader = /^---\s+/m.test(text) && /^\+\+\+\s+/m.test(text);
	if (hasPlainFileHeader) {
		return true;
	}

	const hasHunkHeader = /^@@\s*-?\d+(?:,\d+)?\s*\+?\d+(?:,\d+)?\s*@@/m.test(text);
	if (!hasHunkHeader) {
		return false;
	}

	// Let hunk-only LLM diffs through, but avoid showing the apply UI for
	// arbitrary code snippets that merely contain an @@ marker.
	return /^(?:\+[^+\n]|-[^-\n]| [^\n])/m.test(text);
}
