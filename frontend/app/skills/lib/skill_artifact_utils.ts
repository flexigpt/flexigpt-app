import type { Skill, SkillArgument, SkillInsert } from '@/spec/skill';

export type SkillInsertFilter = 'all' | SkillInsert;

export interface NormalizedSkillInsert {
	value: SkillInsert;
	isDefaulted: boolean;
}

export function normalizeSkillInsert(insert?: string | null): NormalizedSkillInsert {
	if (insert === 'user-message' || insert === 'instructions') {
		return { value: insert, isDefaulted: false };
	}

	return { value: 'instructions', isDefaulted: true };
}

export function skillMatchesInsertFilter(insert?: string | null, filter: SkillInsertFilter = 'all'): boolean {
	if (filter === 'all') {
		return true;
	}

	return normalizeSkillInsert(insert).value === filter;
}

export function getSkillInsertLabel(insert?: string | null): string {
	const normalized = normalizeSkillInsert(insert);
	return normalized.isDefaulted ? `${normalized.value} (default)` : normalized.value;
}

export function getSkillInsertShortLabel(insert?: string | null): string {
	return normalizeSkillInsert(insert).value === 'user-message' ? 'User-message' : 'Instructions';
}

export function getSkillInsertDescription(insert?: string | null): string {
	return normalizeSkillInsert(insert).value === 'user-message'
		? 'Rendered into the user message or composer body. It is not loaded as active session context.'
		: 'Loaded as skill instructions/context. It can be preloaded into sessions and shown to the model.';
}

export function getSkillArgumentCountLabel(args?: SkillArgument[] | null): string {
	const count = args?.length ?? 0;
	return count === 0 ? 'No args' : `${count} arg${count === 1 ? '' : 's'}`;
}

export function getSkillArgumentTooltip(args?: SkillArgument[] | null): string {
	if (!args?.length) {
		return 'No arguments declared.';
	}

	return args
		.map(arg => {
			const pieces = [arg.name];
			if (arg.default) {
				pieces.push(`default: ${arg.default}`);
			}
			if (arg.description) {
				pieces.push(arg.description);
			}
			return pieces.join(' · ');
		})
		.join('\n');
}

export function getSkillInsertCounts(skills: Skill[]): Record<SkillInsert, number> {
	return skills.reduce(
		(acc, skill) => {
			const normalized = normalizeSkillInsert(skill.insert);
			acc[normalized.value] += 1;
			return acc;
		},
		{ instructions: 0, 'user-message': 0 }
	);
}

export function stringifySkillFrontmatter(rawFrontmatter?: Record<string, any> | null): string {
	if (!rawFrontmatter || Object.keys(rawFrontmatter).length === 0) {
		return '';
	}

	try {
		return JSON.stringify(rawFrontmatter, null, 2);
	} catch {
		return '';
	}
}

export function formatSkillArgumentList(args?: SkillArgument[] | null): string[] {
	if (!args?.length) {
		return [];
	}

	return args.map(arg => {
		const segments = [arg.name];
		if (arg.description) {
			segments.push(arg.description);
		}
		if (arg.default) {
			segments.push(`default: ${arg.default}`);
		}
		return segments.join(' · ');
	});
}
