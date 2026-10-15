import type { ArtifactRef } from '@/spec/artifact';
import type { PluginListItem } from '@/spec/plugin';
import type { SkillArgument, SkillDocumentInput, SkillInsert, StoreSkillListItem } from '@/spec/skill';
import { ArtifactState } from '@/spec/artifact';

import type { SkillManagementPresentation } from '@/apis/skill_management';

export interface SkillManagementFilter {
	query: string;
	insert: SkillInsert | SkillManagementInsertFilter.All;
	tags: string[];
}

export enum SkillManagementInsertFilter {
	All = 'all',
}

export function skillArtifactKey(ref: ArtifactRef): string {
	return JSON.stringify([ref.rootID, ref.artifactID]);
}

export function sameSkillArtifact(left: ArtifactRef, right: ArtifactRef): boolean {
	return left.rootID === right.rootID && left.artifactID === right.artifactID;
}

export function canManageSkillPlugin(plugin: PluginListItem): boolean {
	return !plugin.builtIn && (plugin.baseline || plugin.editable);
}

export function canWriteSkillPlugin(plugin: PluginListItem): boolean {
	return canManageSkillPlugin(plugin) && plugin.enabled && plugin.state === ArtifactState.Available;
}

export function makeUniqueSkillName(seed: string, existingNames: readonly string[]): string {
	const base =
		seed
			.trim()
			.toLowerCase()
			.replaceAll(/[^a-z0-9-]+/g, '-')
			.replaceAll(/-+/g, '-')
			.replaceAll(/^-|-$/g, '')
			.slice(0, 64) || 'skill';

	const existing = new Set(existingNames);
	if (!existing.has(base)) {
		return base;
	}

	for (let index = 2; ; index += 1) {
		const suffix = `-${index}`;
		const candidate = `${base.slice(0, 64 - suffix.length)}${suffix}`;
		if (!existing.has(candidate)) {
			return candidate;
		}
	}
}

export function makeUniqueSkillCopyName(seed: string, existingNames: readonly string[]): string {
	const normalized =
		seed
			.trim()
			.toLowerCase()
			.replaceAll(/[^a-z0-9-]+/g, '-')
			.replaceAll(/-+/g, '-')
			.replaceAll(/^-|-$/g, '')
			.slice(0, 58) || 'skill';

	return makeUniqueSkillName(`${normalized}-copy`, existingNames);
}

function escapeArgumentColumn(value: string): string {
	return value.replaceAll('\\', '\\\\').replaceAll('|', '\\|').replaceAll('\r', '\\r').replaceAll('\n', '\\n');
}

/**
 * Omitted defaults remain omitted. Pipes and line breaks round-trip instead
 * of becoming extra columns or declarations.
 */
export function formatSkillArguments(args: readonly SkillArgument[] | undefined): string {
	return (args ?? [])
		.map(argument => {
			const columns = [argument.name];
			if (argument.description !== undefined || argument.default !== undefined) {
				columns.push(argument.description ?? '');
			}
			if (argument.default !== undefined) {
				columns.push(argument.default);
			}
			return columns
				.map(c => {
					return escapeArgumentColumn(c);
				})
				.join(' | ');
		})
		.join('\n');
}

function splitArgumentColumns(line: string): string[] {
	const columns: string[] = [];
	let current = '';

	for (let index = 0; index < line.length; index += 1) {
		const character = line[index];

		if (character === '\\' && index + 1 < line.length) {
			const escaped = line[index + 1];
			if (escaped === '\\' || escaped === '|') {
				current += escaped;
				index += 1;
				continue;
			}
			if (escaped === 'n' || escaped === 'r') {
				current += escaped === 'n' ? '\n' : '\r';
				index += 1;
				continue;
			}
		}

		if (character === '|') {
			columns.push(current.trim());
			current = '';
		} else {
			current += character;
		}
	}

	columns.push(current.trim());
	return columns;
}

export function parseSkillArguments(raw: string): {
	values: SkillArgument[];
	error?: string;
} {
	const values: SkillArgument[] = [];
	const seen = new Set<string>();

	for (const [index, rawLine] of raw.split('\n').entries()) {
		const line = rawLine.trim();
		if (!line || line.startsWith('#')) {
			continue;
		}

		const columns = splitArgumentColumns(line);
		if (columns.length > 3) {
			return {
				values: [],
				error: `Argument line ${index + 1} has too many columns. Escape a literal pipe as \\|.`,
			};
		}

		const name = columns[0] ?? '';
		if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) {
			return { values: [], error: `Argument name on line ${index + 1} is invalid.` };
		}
		if (seen.has(name)) {
			return { values: [], error: `Argument "${name}" appears more than once.` };
		}

		seen.add(name);
		values.push({
			name,
			description: columns.length >= 2 ? columns[1] || undefined : undefined,
			default: columns.length === 3 ? columns[2] : undefined,
		});
	}

	return { values };
}

export function encodeSkillDocument(document: SkillDocumentInput): number[] {
	const frontmatter: Record<string, unknown> = {
		...document.rawFrontmatter,
		name: document.name,
		description: document.description,
		insert: document.insert,
		displayName: document.displayName,
		tags: document.tags?.length ? document.tags : undefined,
		arguments: document.arguments?.length ? document.arguments : undefined,
	};

	const lines = ['---'];
	for (const [key, value] of Object.entries(frontmatter)) {
		if (value === undefined) {
			continue;
		}
		// JSON scalar/flow values are valid YAML and avoid string interpolation.
		lines.push(`${JSON.stringify(key)}: ${JSON.stringify(value)}`);
	}
	lines.push('---', '', document.markdownBody.trim(), '');
	return [...new TextEncoder().encode(lines.join('\n'))];
}

export function skillMatchesManagementFilter(
	skill: StoreSkillListItem,
	presentation: SkillManagementPresentation | undefined,
	filter: SkillManagementFilter
): boolean {
	const document = presentation?.document;
	const insert = document?.insert ?? presentation?.summary?.Insert;
	if (filter.insert !== SkillManagementInsertFilter.All && insert !== filter.insert) {
		return false;
	}

	const tags = new Set((document?.tags ?? []).map(tag => tag.trim().toLowerCase()));
	if (filter.tags.length > 0 && !filter.tags.some(tag => tags.has(tag.toLowerCase()))) {
		return false;
	}

	const query = filter.query.trim().toLowerCase();
	if (!query) {
		return true;
	}

	const text = [
		skill.displayName,
		skill.name,
		skill.description,
		skill.state,
		insert,
		presentation?.location,
		presentation?.details?.artifact.binding.locator,
		presentation?.details?.artifact.binding.sourceID,
		...(presentation?.resources?.locations ?? []),
		...(presentation?.runtimeWarnings ?? []),
		...(presentation?.diagnostics ?? []).flatMap(diagnostic => [
			diagnostic.code,
			diagnostic.message,
			diagnostic.location?.locator,
		]),
		...(document?.tags ?? []),
		...(document?.arguments ?? []).flatMap(argument => [argument.name, argument.description, argument.default]),
		...(presentation?.errors ?? []),
	]
		.filter(Boolean)
		.join('\n')
		.toLowerCase();

	return text.includes(query);
}
