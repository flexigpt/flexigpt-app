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

export const normalizeSkillRefs = (refs: SkillRef[] | null | undefined): SkillRef[] => {
	return dedupeSkillRefs(refs ?? []);
};

export const buildSkillRefsFingerprint = (refs: SkillRef[] | null | undefined): string => {
	const keys = normalizeSkillRefs(refs).map(skillRefKey);
	keys.sort();
	return keys.join('|');
};

export const clampActiveSkillRefsToEnabled = (
	enabledRefs: SkillRef[] | null | undefined,
	activeRefs: SkillRef[] | null | undefined
): SkillRef[] => {
	const enabled = normalizeSkillRefs(enabledRefs);
	if (enabled.length === 0) return [];

	const allow = new Set(enabled.map(skillRefKey));
	return dedupeSkillRefs((activeRefs ?? []).filter(ref => allow.has(skillRefKey(ref))));
};

export const areSkillRefListsEqual = (a: SkillRef[] | null | undefined, b: SkillRef[] | null | undefined): boolean => {
	const left = a ?? [];
	const right = b ?? [];

	if (left.length !== right.length) return false;
	for (let i = 0; i < left.length; i += 1) {
		if (skillRefKey(left[i]) !== skillRefKey(right[i])) return false;
	}
	return true;
};

export function isSkillsToolName(name: string | undefined): boolean {
	const n = (name ?? '').trim();
	return n.startsWith('skills-');
}
