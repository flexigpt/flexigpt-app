import type { SkillDef, SkillListItem } from '@/spec/skill';

export function skillDefKey(def: SkillDef): string {
	return `${def.type}:${def.location}:${def.name}`;
}

export function skillDefFromListItem(item: SkillListItem): SkillDef {
	const s = item.skillDefinition;
	return {
		type: s.type,
		name: s.name,
		location: s.location,
	};
}

export function dedupeSkillDefs(defs: SkillDef[]): SkillDef[] {
	const out: SkillDef[] = [];
	const seen = new Set<string>();
	for (const d of defs ?? []) {
		const k = skillDefKey(d);
		if (seen.has(k)) continue;
		seen.add(k);
		out.push(d);
	}
	return out;
}
