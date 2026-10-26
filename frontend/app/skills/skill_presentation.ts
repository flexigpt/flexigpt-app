import { getDistinctUserFacingLabel, getPrimaryUserFacingLabel } from '@/lib/user_facing_presentation';

interface SkillPresentationSource {
	name?: string;
	displayName?: string;
}

export interface SkillPresentationItem {
	name?: string;
	displayName?: string;
	builtIn?: boolean;
	managed?: boolean;
	plugins?: readonly SkillPresentationSource[];
}

function normalized(value: string | undefined): string {
	return value?.trim() ?? '';
}

export function getSkillDisplayName(skill: Pick<SkillPresentationItem, 'displayName' | 'name'>): string {
	const label = getPrimaryUserFacingLabel(skill.displayName ?? '', skill.name ?? '');

	return normalized(label) || 'Untitled Skill';
}

export function getSkillSecondaryLabel(skill: Pick<SkillPresentationItem, 'displayName' | 'name'>): string | undefined {
	const displayName = normalized(skill.displayName);
	const logicalName = normalized(skill.name);

	if (!displayName || !logicalName) {
		return undefined;
	}

	return normalized(getDistinctUserFacingLabel(displayName, logicalName)) || undefined;
}

export function getSkillSourceLabel(skill: SkillPresentationItem): string {
	if (skill.builtIn) {
		return 'Built-in';
	}

	const sourceNames = [
		...new Set((skill.plugins ?? []).map(plugin => normalized(plugin.displayName || plugin.name)).filter(Boolean)),
	];

	if (sourceNames.length === 1) {
		return sourceNames[0];
	}

	if (sourceNames.length > 1) {
		return `${sourceNames.length} sources`;
	}

	return skill.managed ? 'Managed' : 'External';
}
