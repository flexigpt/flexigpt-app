import type { Skill, SkillArgument, SkillInsert, SkillResourceInfo } from '@/spec/skill';

export type SkillInsertFilter = 'all' | SkillInsert;

export interface NormalizedSkillInsert {
	value: SkillInsert;
	isDefaulted: boolean;
}

export interface SkillMarkdownScaffoldInput {
	name: string;
	description?: string;
	displayName?: string;
	insert: SkillInsert;
	arguments?: SkillArgument[];
	body?: string;
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
	const label = normalized.value === 'user-message' ? 'User message' : 'Instructions';
	return normalized.isDefaulted ? `${label} (default)` : label;
}

export function getSkillInsertShortLabel(insert?: string | null): string {
	return normalizeSkillInsert(insert).value === 'user-message' ? 'User-message' : 'Instructions';
}

export function getSkillInsertDescription(insert?: string | null): string {
	return normalizeSkillInsert(insert).value === 'user-message'
		? 'Rendered into the user message or composer body. It is not loaded as active session context.'
		: 'Loaded as skill instructions/context. It can be preloaded into sessions and shown to the model.';
}

export function getSkillInsertLongGuidance(insert?: string | null): string {
	return normalizeSkillInsert(insert).value === 'user-message'
		? 'Use this when the skill is a prompt-like template. The rendered text is inserted into a user message or composer draft. Indexed filesystem resources can be selected as ordinary message attachments. The skill is not loaded as persistent session context.'
		: 'Use this when the skill is standing instruction or context. It can be enabled for a conversation and loaded as an active session skill.';
}

export function getSkillResourceCountLabel(resources?: SkillResourceInfo | null): string {
	if (!resources?.hasResources || resources.totalCount <= 0) {
		return 'No resources';
	}

	return `${resources.totalCount} resource${resources.totalCount === 1 ? '' : 's'}`;
}

export function getSkillResourceTooltip(resources?: SkillResourceInfo | null): string {
	if (!resources?.hasResources || resources.totalCount <= 0) {
		return 'No indexed resource files were reported for this skill.';
	}

	const lines = [
		`${resources.totalCount} resource${resources.totalCount === 1 ? '' : 's'} reported by the runtime.`,
		'Resources are files inside the skill directory, such as references, assets, or scripts.',
		'User-message template insertion can attach selected indexed resource paths; instruction skills access resources through the skill lifecycle.',
	];

	for (const location of resources.locations ?? []) {
		lines.push(location);
	}

	if (resources.moreLocations) {
		lines.push('More resource locations exist but were omitted from this listing.');
	}

	return lines.join('\n');
}

export function skillHasResources(skill: Pick<Skill, 'resources'>): boolean {
	return skill.resources?.hasResources || (skill.resources?.totalCount !== undefined && skill.resources.totalCount > 0);
}

function skillArgumentCount(skill: Pick<Skill, 'arguments'>): number {
	return skill.arguments?.length ?? 0;
}

export function isInstructionInsertSkill(skill: Pick<Skill, 'insert'>): boolean {
	return normalizeSkillInsert(skill.insert).value === 'instructions';
}

export function skillCanBeRenderedAsInstructionPrompt(
	skill: Pick<Skill, 'insert' | 'arguments' | 'resources'>
): boolean {
	return isInstructionInsertSkill(skill) && skillArgumentCount(skill) === 0 && !skillHasResources(skill);
}

export function skillCanBePreloadedAsActive(skill: Pick<Skill, 'insert' | 'arguments'>): boolean {
	return isInstructionInsertSkill(skill) && skillArgumentCount(skill) === 0;
}

export function getSkillInstructionPromptEligibilityReason(
	skill: Pick<Skill, 'insert' | 'arguments' | 'resources'>
): string | undefined {
	if (!isInstructionInsertSkill(skill)) {
		return 'Only instruction skills can be used as system instructions.';
	}

	if (skillHasResources(skill)) {
		return 'Resource-backed skills cannot be converted into system prompt text. Enable or activate the skill session instead; the model can read resources through skill tools.';
	}

	if (skillArgumentCount(skill) > 0) {
		return 'Argument-backed instruction skills cannot be inserted as plain system instructions. Enable the skill instead and let the skill lifecycle handle argumented use.';
	}

	return undefined;
}

export function getSkillPreloadEligibilityReason(skill: Pick<Skill, 'insert' | 'arguments'>): string | undefined {
	if (!isInstructionInsertSkill(skill)) {
		return 'Only instruction skills can be enabled in a skill session.';
	}

	if (skillArgumentCount(skill) > 0) {
		return 'Argument-backed skills cannot be preloaded as active session skills from presets.';
	}

	return undefined;
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
			if (arg.default !== undefined) {
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

export function getAllSkillTags(skills: Skill[]): string[] {
	const tags = new Set<string>();

	for (const skill of skills) {
		for (const tag of skill.tags ?? []) {
			const value = tag.trim();
			if (value) {
				tags.add(value);
			}
		}
	}

	return [...tags].toSorted((a, b) => a.localeCompare(b));
}

export function skillMatchesTags(skill: Skill, tagFilters: string[]): boolean {
	const filters = tagFilters.map(tag => tag.trim().toLowerCase()).filter(Boolean);
	if (filters.length === 0) {
		return true;
	}

	const tags = new Set((skill.tags ?? []).map(tag => tag.trim().toLowerCase()).filter(Boolean));
	return filters.some(tag => tags.has(tag));
}

export function skillMatchesSearch(skill: Skill, rawQuery: string): boolean {
	const query = rawQuery.trim().toLowerCase();
	if (!query) {
		return true;
	}

	const haystack = [
		skill.displayName,
		skill.name,
		skill.slug,
		skill.description,
		skill.type,
		skill.location,
		skill.insert,
		skill.digest,
		skill.presence?.status,
		...(skill.runtimeWarnings ?? []),
		...(skill.resources?.locations ?? []),
		...(skill.tags ?? []),
		...(skill.arguments ?? []).flatMap(arg => [arg.name, arg.description, arg.default]),
	]
		.filter(Boolean)
		.join('\n')
		.toLowerCase();

	return haystack.includes(query);
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
		if (arg.default !== undefined) {
			segments.push(`default: ${arg.default}`);
		}
		return segments.join(' · ');
	});
}

function yamlQuote(value: string): string {
	const normalized = value.replaceAll('\r\n', '\n').replaceAll('\r', '\n');
	if (!normalized) {
		return '""';
	}
	return JSON.stringify(normalized);
}

function humanizeName(name: string): string {
	return name
		.trim()
		.replaceAll(/[-_]+/g, ' ')
		.replaceAll(/\s+/g, ' ')
		.replaceAll(/\b\w/g, c => c.toUpperCase());
}

export function buildSkillMarkdownScaffold(input: SkillMarkdownScaffoldInput): string {
	const name = input.name.trim() || 'my-skill';
	const description = input.description?.trim() || 'Describe what this skill does and when to use it.';
	const insert = normalizeSkillInsert(input.insert).value;
	const displayName = input.displayName?.trim() || humanizeName(name);
	const body =
		input.body?.trim() ||
		(insert === 'user-message'
			? 'Write the user-message template body here. Use $argument or {{ argument }} placeholders.'
			: 'Write instruction/context material here. The model can load this as active session context.');
	const args = input.arguments?.filter(arg => arg.name.trim()) ?? [];

	const lines: string[] = ['---', `name: ${yamlQuote(name)}`, `description: ${yamlQuote(description)}`];

	if (insert !== 'instructions') {
		lines.push(`insert: ${yamlQuote(insert)}`);
	} else {
		lines.push('insert: instructions');
	}

	if (args.length > 0) {
		lines.push('arguments:');
		for (const arg of args) {
			lines.push(`  - name: ${yamlQuote(arg.name.trim())}`);
			if (arg.description?.trim()) {
				lines.push(`    description: ${yamlQuote(arg.description.trim())}`);
			}
			if (arg.default !== undefined) {
				lines.push(`    default: ${yamlQuote(arg.default)}`);
			}
		}
	}

	lines.push('---', '', `# ${displayName}`, '', body, '');
	return lines.join('\n');
}

export function buildSkillArgumentText(args?: SkillArgument[] | null): string {
	return (args ?? [])
		.map(arg => [arg.name, arg.description ?? '', arg.default ?? ''].join(' | ').replace(/\s+\|\s+\|\s*$/, ''))
		.join('\n');
}

export function buildSkillForkBodyPlaceholder(source: Skill): string {
	const insert = normalizeSkillInsert(source.insert).value;
	const sourceLabel = source.displayName || source.name || source.slug;

	return insert === 'user-message'
		? `Forked from "${sourceLabel}". Replace this placeholder with the new user-message template body.\n\nUse $argument or {{ argument }} placeholders declared above.`
		: `Forked from "${sourceLabel}". Replace this placeholder with instruction/context material.\n\nOnly instruction skills can be activated in a skill session.`;
}
