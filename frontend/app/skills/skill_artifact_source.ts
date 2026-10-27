import type { ArtifactRef } from '@/spec/artifact';
import type { ArtifactSkillDetails, SkillDocumentInput } from '@/spec/skill';

import { skillManagementAPI } from '@/apis/baseapi';

import type { ManagementArtifactSourceValue } from '@/components/managementui/management_artifact_source';

import { encodeSkillDocument } from '@/skills/skill_management_utils';

export interface LoadedSkillSource {
	details: ArtifactSkillDetails;
	revision: number;
}

export function canonicalSkillSource(document: SkillDocumentInput): string {
	return new TextDecoder().decode(new Uint8Array(encodeSkillDocument(document)));
}

/**
 * A pinned template must not silently switch revisions. A read-only details
 * view can explicitly opt into resolving a newer revision after a stale read.
 */
export async function loadSkillArtifactSource(
	ref: ArtifactRef,
	expectedRevision?: number,
	allowCurrentOnStale = false
): Promise<ManagementArtifactSourceValue<LoadedSkillSource>> {
	const revision = expectedRevision ?? (await skillManagementAPI.getSkill(ref)).revision;
	let details: ArtifactSkillDetails;

	try {
		details = await skillManagementAPI.getArtifactSkillDetails(ref, revision);
	} catch (cause) {
		if (!allowCurrentOnStale || expectedRevision === undefined) {
			throw cause;
		}
		const latest = await skillManagementAPI.getSkill(ref);
		if (latest.revision === revision) {
			throw cause;
		}
		details = await skillManagementAPI.getArtifactSkillDetails(ref, latest.revision);
	}

	if (expectedRevision !== undefined && !allowCurrentOnStale && details.artifact.revision !== expectedRevision) {
		throw new Error('The Skill changed. Refresh it before using this template.');
	}

	const loaded = { details, revision: details.artifact.revision };
	return {
		data: loaded,
		text: canonicalSkillSource(details.document),
		fileName: 'SKILL.md',
		extension: 'md',
	};
}
