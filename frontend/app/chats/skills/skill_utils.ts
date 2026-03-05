import type { SkillListItem, SkillRef } from '@/spec/skill';

export function skillRefKey(ref: SkillRef): string {
	return `${ref.bundleID}:${ref.skillSlug}:${ref.skillID}`;
}

export function skillRefFromListItem(item: SkillListItem): SkillRef {
	return {
		bundleID: item.bundleID,
		skillSlug: item.skillSlug,
		skillID: item.skillDefinition.id,
	};
}

export function dedupeSkillRefs(refs: SkillRef[]): SkillRef[] {
	const out: SkillRef[] = [];
	const seen = new Set<string>();
	for (const r of refs ?? []) {
		const k = skillRefKey(r);
		if (seen.has(k)) continue;
		seen.add(k);
		out.push(r);
	}
	return out;
}
