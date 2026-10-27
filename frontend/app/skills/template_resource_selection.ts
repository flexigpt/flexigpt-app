/**
 * Keeps resource attachment selection inside the exact allowlist produced by
 * the verified SKILL.md details response. The output always preserves the
 * source-provided path values and source-provided order.
 */
export function uniqueTemplateResourcePaths(paths: readonly string[] | undefined): string[] {
	const seen = new Set<string>();
	const output: string[] = [];

	for (const path of paths ?? []) {
		if (!path.trim() || seen.has(path)) {
			continue;
		}

		seen.add(path);
		output.push(path);
	}

	return output;
}

export function selectVerifiedTemplateResourcePaths(
	allowedPaths: readonly string[],
	selectedPaths: readonly string[]
): string[] {
	const selected = new Set(selectedPaths);

	return uniqueTemplateResourcePaths(allowedPaths).filter(path => selected.has(path));
}
