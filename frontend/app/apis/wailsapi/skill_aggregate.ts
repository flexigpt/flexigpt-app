import type { ArtifactRef } from '@/spec/artifact';
import type { ArtifactSkillFilter, ArtifactSkillSummary, ResolvedArtifactSkill } from '@/spec/skill';

import type { ISkillAggregateAPI } from '@/apis/interface';
import { requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	DescribeArtifactSkill,
	ListArtifactSkillRefs,
	ResolveArtifactSkill,
} from '@/apis/wailsjs/go/main/SkillAggregateWrapper';

export class WailsSkillAggregateAPI implements ISkillAggregateAPI {
	async resolveArtifactSkill(skill: ArtifactRef): Promise<ResolvedArtifactSkill> {
		return requiredObject<ResolvedArtifactSkill>(
			await ResolveArtifactSkill(skill as Parameters<typeof ResolveArtifactSkill>[0]),
			'ResolveArtifactSkill'
		);
	}

	async listArtifactSkillRefs(filter: ArtifactSkillFilter): Promise<ArtifactRef[]> {
		return wailsObjectArrayOrEmpty<ArtifactRef>(
			await ListArtifactSkillRefs(filter as Parameters<typeof ListArtifactSkillRefs>[0]),
			'ListArtifactSkillRefs'
		);
	}

	async describeArtifactSkill(skill: ArtifactRef): Promise<ArtifactSkillSummary> {
		return requiredObject<ArtifactSkillSummary>(
			await DescribeArtifactSkill(skill as Parameters<typeof DescribeArtifactSkill>[0]),
			'DescribeArtifactSkill'
		);
	}
}
