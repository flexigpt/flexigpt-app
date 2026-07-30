import type { InstalledSkillRef, SkillListItem, SkillRef, SkillSelection } from '@/spec/skill';

const INSTALLED_SKILL_IDENTITY_PREFIX = 'installed/';

function normalizedFields(ref: SkillRef): InstalledSkillRef | undefined {
	const bundleID = ref.bundleID?.trim();
	const skillSlug = ref.skillSlug?.trim();
	const skillID = ref.skillID?.trim();

	if (!bundleID || !skillSlug || !skillID) {
		return undefined;
	}

	return { bundleID, skillSlug, skillID };
}

function getSkillRefStableIdentity(ref: SkillRef | null | undefined): string | undefined {
	if (!ref) {
		return undefined;
	}

	const normalized = normalizedFields(ref);
	return normalized
		? `${INSTALLED_SKILL_IDENTITY_PREFIX}${normalized.bundleID}/${normalized.skillSlug}/${normalized.skillID}`
		: undefined;
}

export function skillRefKey(ref: SkillRef): string {
	const stableIdentity = getSkillRefStableIdentity(ref);
	if (stableIdentity) {
		return stableIdentity;
	}

	return `invalid:${ref.bundleID?.trim() ?? ''}:${ref.skillSlug?.trim() ?? ''}:${ref.skillID?.trim() ?? ''}`;
}

export function isInstalledSkillRef(ref: SkillRef): boolean {
	return normalizedFields(ref) !== undefined;
}

export function requireInstalledSkillRef(ref: SkillRef, label = 'Skill reference'): InstalledSkillRef {
	const normalized = normalizedFields(ref);
	if (!normalized) {
		throw new Error(`${label} must contain an installed bundleID, skillSlug, and skillID.`);
	}

	return normalized;
}

export function formatSkillRef(ref: SkillRef): string {
	const normalized = normalizedFields(ref);
	if (!normalized) {
		return 'Invalid Skill reference';
	}

	return `${normalized.bundleID}/${normalized.skillSlug}#${normalized.skillID}`;
}

export function skillRefFromListItem(item: SkillListItem): SkillRef {
	return {
		bundleID: item.bundleID,
		skillSlug: item.skillSlug,
		skillID: item.skillDefinition.id,
	};
}

export function dedupeSkillRefs(refs: SkillRef[] | null | undefined): SkillRef[] {
	const out: SkillRef[] = [];
	const seen = new Set<string>();

	for (const r of refs ?? []) {
		const normalized = normalizedFields(r);
		if (!normalized) {
			continue;
		}
		const k = skillRefKey(normalized);
		if (seen.has(k)) {
			continue;
		}
		seen.add(k);
		out.push(normalized);
	}
	return out;
}

export const normalizeSkillSelectionsToRefs = (sels: SkillSelection[] | null | undefined): SkillRef[] => {
	const r = sels?.map(item => {
		return item.skillRef;
	});
	return normalizeSkillRefs(r ?? []);
};

export const normalizeSkillRefs = (refs: SkillRef[] | null | undefined): SkillRef[] => {
	return dedupeSkillRefs(refs ?? []);
};

export const buildSkillRefsFingerprint = (refs: SkillRef[] | null | undefined): string => {
	const keys = normalizeSkillRefs(refs)
		.map(r => skillRefKey(r))
		.toSorted();
	return keys.join('|');
};

export const clampActiveSkillRefsToEnabled = (
	enabledRefs: SkillRef[] | null | undefined,
	activeRefs: SkillRef[] | null | undefined
): SkillRef[] => {
	const enabled = normalizeSkillRefs(enabledRefs);
	if (enabled.length === 0) {
		return [];
	}

	const allow = new Set(enabled.map(r => skillRefKey(r)));
	return dedupeSkillRefs(
		(activeRefs ?? []).filter(ref => {
			const normalized = normalizedFields(ref);
			return normalized !== undefined && allow.has(skillRefKey(normalized));
		})
	);
};

export const areSkillRefListsEqual = (a: SkillRef[] | null | undefined, b: SkillRef[] | null | undefined): boolean => {
	const left = normalizeSkillRefs(a);
	const right = normalizeSkillRefs(b);

	if (left.length !== right.length) {
		return false;
	}

	for (let i = 0; i < left.length; i += 1) {
		if (skillRefKey(left[i]) !== skillRefKey(right[i])) {
			return false;
		}
	}
	return true;
};

export function isSkillsToolName(name: string | undefined): boolean {
	const n = (name ?? '').trim();
	return n.startsWith('skills-');
}
