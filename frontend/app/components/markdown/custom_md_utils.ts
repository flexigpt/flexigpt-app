export enum CustomMDLanguage {
	ThinkingSummary = 'thinkingsummary',
	Thinking = 'thinking',
}

export function stripCustomMDFences(markdown: string): string {
	// Remove all ~~~thinking blocks
	let r = markdown.replaceAll(/(^|\n)~~~thinking\s*[\s\S]*?\n~~~\s*/g, '$1');
	r = r.replaceAll(/(^|\n)~~~thinkingsummary\s*[\s\S]*?\n~~~\s*/g, '$1');
	return r;
}
