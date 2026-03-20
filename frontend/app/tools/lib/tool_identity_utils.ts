// Build a stable identity key for a tool selection (bundle + slug + version).
// Prefer bundleID when present, otherwise fall back to bundleSlug.
export function toolIdentityKey(
	bundleID: string | undefined,
	bundleSlug: string | undefined,
	toolSlug: string,
	toolVersion: string
): string {
	const bundlePart = bundleID ? `id:${bundleID}` : `slug:${bundleSlug ?? ''}`;
	return `${bundlePart}/${toolSlug}@${toolVersion}`;
}

/**
 * Human-friendly tool name for display.
 * Accepts forms like:
 *   "bundleSlug/toolSlug@version"
 *   "bundleID/toolSlug@version"
 *   "toolSlug"
 */
export function getPrettyToolName(name: string): string {
	if (!name) return 'Tool';
	let base = name;
	if (base.includes('/')) {
		const parts = base.split('/');
		base = parts[parts.length - 1] || base;
	}
	if (base.includes('@')) {
		base = base.split('@')[0] || base;
	}
	return base.replace(/[-_]/g, ' ');
}
