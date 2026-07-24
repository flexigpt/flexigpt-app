import type {
	InstalledSkillRef,
	SkillListItem,
	SkillRef,
	SkillRefKind,
	SkillSelection,
	WorkspaceSkillRef,
} from '@/spec/skill';
import { SkillRefKind as SkillRefKindValue } from '@/spec/skill';

const INSTALLED_SKILL_IDENTITY_PREFIX = 'installed/';
const WORKSPACE_SKILL_IDENTITY_PREFIX = 'workspace/';

export interface WorkspaceSkillRefParts {
	rootID: string;
	recordID: string;
}

function hasAnyInstalledField(ref: SkillRef): boolean {
	return Boolean(ref.bundleID?.trim() || ref.skillSlug?.trim() || ref.skillID?.trim());
}

function installedFields(ref: SkillRef): InstalledSkillRef | undefined {
	const bundleID = ref.bundleID?.trim();
	const skillSlug = ref.skillSlug?.trim();
	const skillID = ref.skillID?.trim();

	if (!bundleID || !skillSlug || !skillID) {
		return undefined;
	}

	return { bundleID, skillSlug, skillID };
}

function parseInstalledIdentity(identity: string): InstalledSkillRef | undefined {
	const value = identity.trim();
	const relative = value.startsWith(INSTALLED_SKILL_IDENTITY_PREFIX)
		? value.slice(INSTALLED_SKILL_IDENTITY_PREFIX.length)
		: '';
	const parts = relative.split('/');

	if (parts.length !== 3 || parts.some(part => !part || part.trim() !== part)) {
		return undefined;
	}

	return {
		bundleID: parts[0],
		skillSlug: parts[1],
		skillID: parts[2],
	};
}

function parseWorkspaceIdentity(identity: string): WorkspaceSkillRefParts | undefined {
	const value = identity.trim();
	const relative = value.startsWith(WORKSPACE_SKILL_IDENTITY_PREFIX)
		? value.slice(WORKSPACE_SKILL_IDENTITY_PREFIX.length)
		: '';
	const parts = relative.split('/');

	if (parts.length !== 2 || parts.some(part => !part || part.trim() !== part)) {
		return undefined;
	}

	return {
		rootID: parts[0],
		recordID: parts[1],
	};
}

function getSkillRefKind(ref: SkillRef | null | undefined): SkillRefKind {
	if (!ref) {
		return SkillRefKindValue.Invalid;
	}

	const identity = ref.identity?.trim();
	if (identity) {
		if (hasAnyInstalledField(ref)) {
			return SkillRefKindValue.Invalid;
		}

		if (parseWorkspaceIdentity(identity)) {
			return SkillRefKindValue.Workspace;
		}
		if (parseInstalledIdentity(identity)) {
			return SkillRefKindValue.Installed;
		}
		return SkillRefKindValue.Invalid;
	}

	return installedFields(ref) ? SkillRefKindValue.Installed : SkillRefKindValue.Invalid;
}

export function normalizeSkillRef(ref: SkillRef): SkillRef | undefined {
	switch (getSkillRefKind(ref)) {
		case SkillRefKindValue.Installed: {
			const fromIdentity = ref.identity ? parseInstalledIdentity(ref.identity) : undefined;
			return fromIdentity ?? installedFields(ref);
		}

		case SkillRefKindValue.Workspace: {
			const parts = ref.identity ? parseWorkspaceIdentity(ref.identity) : undefined;
			return parts ? { identity: `${WORKSPACE_SKILL_IDENTITY_PREFIX}${parts.rootID}/${parts.recordID}` } : undefined;
		}

		default:
			return undefined;
	}
}

/**
 * Stable identity used for equality, deduplication, fingerprints, session
 * partitioning, and display fallback.
 *
 * Installed refs normalize to the same key whether an old caller supplied
 * fields or `installed/<bundle>/<slug>/<id>`.
 */
function getSkillRefStableIdentity(ref: SkillRef | null | undefined): string | undefined {
	if (!ref) {
		return undefined;
	}

	switch (getSkillRefKind(ref)) {
		case SkillRefKindValue.Installed: {
			const installed = ref.identity ? parseInstalledIdentity(ref.identity) : installedFields(ref);
			return installed
				? `${INSTALLED_SKILL_IDENTITY_PREFIX}${installed.bundleID}/${installed.skillSlug}/${installed.skillID}`
				: undefined;
		}

		case SkillRefKindValue.Workspace: {
			const workspace = ref.identity ? parseWorkspaceIdentity(ref.identity) : undefined;
			return workspace ? `${WORKSPACE_SKILL_IDENTITY_PREFIX}${workspace.rootID}/${workspace.recordID}` : undefined;
		}

		default:
			return undefined;
	}
}

export function skillRefKey(ref: SkillRef): string {
	const stableIdentity = getSkillRefStableIdentity(ref);
	if (stableIdentity) {
		return stableIdentity;
	}

	return `invalid:${ref.identity?.trim() ?? ''}:${ref.bundleID?.trim() ?? ''}:${ref.skillSlug?.trim() ?? ''}:${ref.skillID?.trim() ?? ''}`;
}

export function isWorkspaceSkillRef(ref: SkillRef): boolean {
	return getSkillRefKind(ref) === SkillRefKindValue.Workspace;
}

export function isInstalledSkillRef(ref: SkillRef): boolean {
	return getSkillRefKind(ref) === SkillRefKindValue.Installed;
}

export function createWorkspaceSkillRef(rootID: string, recordID: string): WorkspaceSkillRef | undefined {
	const normalizedRootID = rootID.trim();
	const normalizedRecordID = recordID.trim();

	if (!normalizedRootID || !normalizedRecordID || normalizedRootID.includes('/') || normalizedRecordID.includes('/')) {
		return undefined;
	}

	return {
		identity: `${WORKSPACE_SKILL_IDENTITY_PREFIX}${normalizedRootID}/${normalizedRecordID}`,
	};
}

export function getWorkspaceSkillRefParts(
	ref: SkillRef | string | null | undefined
): WorkspaceSkillRefParts | undefined {
	const identity = typeof ref === 'string' ? ref : ref?.identity;
	return identity ? parseWorkspaceIdentity(identity) : undefined;
}

export function requireInstalledSkillRef(ref: SkillRef, label = 'Skill reference'): InstalledSkillRef {
	const normalized = normalizeSkillRef(ref);
	if (!normalized || !isInstalledSkillRef(normalized)) {
		throw new Error(`${label} must contain an installed bundleID, skillSlug, and skillID.`);
	}

	return {
		bundleID: normalized.bundleID as string,
		skillSlug: normalized.skillSlug as string,
		skillID: normalized.skillID as string,
	};
}

export function formatSkillRef(ref: SkillRef): string {
	const normalized = normalizeSkillRef(ref);
	if (!normalized) {
		return 'Invalid Skill reference';
	}

	if (isWorkspaceSkillRef(normalized)) {
		return normalized.identity as string;
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
		const normalized = normalizeSkillRef(r);
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
			const normalized = normalizeSkillRef(ref);
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
