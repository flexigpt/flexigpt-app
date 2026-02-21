import type {
	RuntimeSkillFilter,
	Skill,
	SkillBundle,
	SkillDef,
	SkillListItem,
	SkillRecord,
	SkillSession,
	SkillType,
} from '@/spec/skill';

import type { ISkillStoreAPI } from '@/apis/interface';
import {
	CloseSkillSession,
	CreateSkillSession,
	DeleteSkill,
	DeleteSkillBundle,
	GetSkill,
	GetSkillsPromptXML,
	ListRuntimeSkills,
	ListSkillBundles,
	ListSkills,
	PatchSkill,
	PatchSkillBundle,
	PutSkill,
	PutSkillBundle,
} from '@/apis/wailsjs/go/main/SkillStoreWrapper';
import type { spec } from '@/apis/wailsjs/go/models';

/**
 * @public
 */
export class WailsSkillStoreAPI implements ISkillStoreAPI {
	async listSkillBundles(
		bundleIDs?: string[],
		includeDisabled?: boolean,
		pageSize?: number,
		pageToken?: string
	): Promise<{ skillBundles: SkillBundle[]; nextPageToken?: string }> {
		const req = {
			BundleIDs: bundleIDs,
			IncludeDisabled: includeDisabled,
			PageSize: pageSize,
			PageToken: pageToken,
		};
		const resp = await ListSkillBundles(req as spec.ListSkillBundlesRequest);
		return {
			skillBundles: (resp.Body?.skillBundles ?? []) as SkillBundle[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async putSkillBundle(
		bundleID: string,
		slug: string,
		displayName: string,
		isEnabled: boolean,
		description?: string
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				slug,
				displayName,
				isEnabled,
				description,
			} as spec.PutSkillBundleRequestBody,
		};
		await PutSkillBundle(req as spec.PutSkillBundleRequest);
	}

	async patchSkillBundle(bundleID: string, isEnabled: boolean): Promise<void> {
		const req = {
			BundleID: bundleID,
			Body: {
				isEnabled,
			},
		};
		await PatchSkillBundle(req as spec.PatchSkillBundleRequest);
	}

	async deleteSkillBundle(bundleID: string): Promise<void> {
		const req: spec.DeleteSkillBundleRequest = { BundleID: bundleID };
		await DeleteSkillBundle(req);
	}

	async listSkills(
		bundleIDs?: string[],
		types?: SkillType[],
		includeDisabled?: boolean,
		includeMissing?: boolean,
		recommendedPageSize?: number,
		pageToken?: string
	): Promise<{ skillListItems: SkillListItem[]; nextPageToken?: string }> {
		const req = {
			BundleIDs: bundleIDs,
			Types: types,
			IncludeDisabled: includeDisabled,
			IncludeMissing: includeMissing,
			RecommendedPageSize: recommendedPageSize,
			PageToken: pageToken,
		};
		const resp = await ListSkills(req as spec.ListSkillsRequest);
		return {
			skillListItems: (resp.Body?.skillListItems ?? []) as SkillListItem[],
			nextPageToken: resp.Body?.nextPageToken ?? undefined,
		};
	}

	async putSkill(
		bundleID: string,
		skillSlug: string,
		skillType: SkillType,
		location: string,
		name: string,
		isEnabled: boolean,
		displayName?: string,
		description?: string,
		tags?: string[]
	): Promise<void> {
		const req = {
			BundleID: bundleID,
			SkillSlug: skillSlug,
			Body: {
				skillType,
				location,
				name,
				isEnabled,
				displayName,
				description,
				tags,
			} as spec.PutSkillRequestBody,
		};
		await PutSkill(req as spec.PutSkillRequest);
	}

	async patchSkill(bundleID: string, skillSlug: string, isEnabled?: boolean, location?: string): Promise<void> {
		const req = {
			BundleID: bundleID,
			SkillSlug: skillSlug,
			Body: {
				isEnabled,
				location,
			} as spec.PatchSkillRequestBody,
		};
		await PatchSkill(req as spec.PatchSkillRequest);
	}

	async deleteSkill(bundleID: string, skillSlug: string): Promise<void> {
		const req: spec.DeleteSkillRequest = { BundleID: bundleID, SkillSlug: skillSlug };
		await DeleteSkill(req);
	}

	async getSkill(bundleID: string, skillSlug: string): Promise<Skill | undefined> {
		const req: spec.GetSkillRequest = { BundleID: bundleID, SkillSlug: skillSlug };
		const resp = await GetSkill(req);
		return resp?.Body as Skill;
	}

	async getSkillsPromptXML(filter?: RuntimeSkillFilter): Promise<string> {
		const req = {
			Body: { filter: filter as spec.RuntimeSkillFilter } as spec.GetSkillsPromptXMLRequestBody,
		} as spec.GetSkillsPromptXMLRequest;
		const resp = await GetSkillsPromptXML(req);
		return resp?.Body?.xml || '';
	}

	async createSkillSession(maxActivePerSession?: number, activeSkills?: SkillDef[]): Promise<SkillSession> {
		const req = {
			Body: {
				maxActivePerSession,
				activeSkills,
			} as spec.CreateSkillSessionRequestBody,
		} as spec.CreateSkillSessionRequest;

		const resp = await CreateSkillSession(req);
		return {
			sessionID: resp?.Body?.sessionID ?? '',
			activeSkills: (resp?.Body?.activeSkills ?? []) as SkillDef[],
		};
	}

	async closeSkillSession(sessionID: string): Promise<void> {
		const req: spec.CloseSkillSessionRequest = { SessionID: sessionID };
		await CloseSkillSession(req);
	}

	async listRuntimeSkills(filter?: RuntimeSkillFilter): Promise<SkillRecord[]> {
		const req = {
			Body: { filter: filter as spec.RuntimeSkillFilter } as spec.ListRuntimeSkillsRequestBody,
		} as spec.ListRuntimeSkillsRequest;

		const resp = await ListRuntimeSkills(req);
		return (resp?.Body?.skills ?? []) as SkillRecord[];
	}
}
