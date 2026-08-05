import type { ArtifactRef } from '@/spec/artifact';
import type { SkillListItem, SkillRef, SkillSelection } from '@/spec/skill';

export function isSkillArtifactRef(ref: unknown): ref is ArtifactRef {
	if (ref === null || typeof ref !== 'object') {
		return false;
	}
	const value = ref as Partial<ArtifactRef>;
	return (
		typeof value.rootID === 'string' &&
		value.rootID.trim().length > 0 &&
		typeof value.artifactID === 'string' &&
		value.artifactID.trim().length > 0
	);
}

export function skillRefKey(ref: SkillRef): string {
	return `${ref.rootID}:${ref.artifactID}`;
}

export function formatSkillRef(ref: SkillRef): string {
	return `${ref.rootID}/${ref.artifactID}`;
}

export function skillRefFromListItem(item: SkillListItem): SkillRef {
	return item.skillDefinition.ref;
}

function toArtifactRef(ref: SkillRef): ArtifactRef | null {
	if (!isSkillArtifactRef(ref)) {
		return null;
	}
	return { rootID: ref.rootID, artifactID: ref.artifactID };
}

export function toArtifactRefs(refs: SkillRef[] | null | undefined): ArtifactRef[] {
	if (!refs || refs.length === 0) {
		return [];
	}
	const out: ArtifactRef[] = [];
	const seen = new Set<string>();

	for (const r of refs) {
		const aRef = toArtifactRef(r);
		if (!aRef) {
			continue;
		}
		const key = `${aRef.rootID}:${aRef.artifactID}`;
		if (!seen.has(key)) {
			seen.add(key);
			out.push(aRef);
		}
	}
	return out;
}

export function dedupeSkillRefs(refs: SkillRef[] | null | undefined): SkillRef[] {
	const seen = new Set<string>();
	const out: SkillRef[] = [];
	for (const r of refs ?? []) {
		if (!isSkillArtifactRef(r)) {
			continue;
		}
		const key = skillRefKey(r);
		if (!seen.has(key)) {
			seen.add(key);
			out.push({ rootID: r.rootID, artifactID: r.artifactID });
		}
		continue;
	}
	return out;
}

export const normalizeSkillSelectionsToRefs = (sels: SkillSelection[] | null | undefined): SkillRef[] => {
	const r = sels?.map(item => {
		return item.artifact;
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
			return allow.has(skillRefKey(ref));
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
