import type { ArtifactRef } from '@/spec/artifact';
import type { ArtifactSkillFilter, ArtifactSkillSummary, ResolvedArtifactSkill } from '@/spec/skill';

import type { ISkillAggregateAPI } from '@/apis/interface';
import { requiredObject, requireWailsString, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';
import {
	DescribeArtifactSkill,
	GetArtifactSkillsPrompt,
	ListArtifactSkillRefs,
	ResolveArtifactSkill,
	ResolveArtifactSkills,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/cmd/agentgo/skillaggregatewrapper';

export class WailsSkillAggregateAPI implements ISkillAggregateAPI {
	async getArtifactSkillsPrompt(filter: ArtifactSkillFilter): Promise<string> {
		return requireWailsString(
			await GetArtifactSkillsPrompt(filter as Parameters<typeof GetArtifactSkillsPrompt>[0]),
			'GetArtifactSkillsPrompt'
		);
	}

	async resolveArtifactSkill(skill: ArtifactRef): Promise<ResolvedArtifactSkill> {
		return requiredObject<ResolvedArtifactSkill>(
			await ResolveArtifactSkill(skill as Parameters<typeof ResolveArtifactSkill>[0]),
			'ResolveArtifactSkill'
		);
	}

	async resolveArtifactSkills(skills: ArtifactRef[]): Promise<ResolvedArtifactSkill[]> {
		return wailsObjectArrayOrEmpty<ResolvedArtifactSkill>(
			await ResolveArtifactSkills(skills as Parameters<typeof ResolveArtifactSkills>[0]),
			'ResolveArtifactSkills'
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
